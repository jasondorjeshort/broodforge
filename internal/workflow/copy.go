// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// Codes this package refuses with (05-debug-interface.md, The workflow copy).
const (
	CodeCanonical   = "workflow-canonical"    // the canonical or its default branch cannot be read
	CodeOut         = "workflow-out"          // the copy's directory cannot be made or written
	CodeSettings    = "workflow-settings"     // .claude/settings.json does not parse while hooks are listed
	CodeProbeFailed = "workflow-probe-failed" // a listed hook's runtime is not where the children run
)

// Error is a refusal with a stable code; the CLI prints it as `error: <code>: <message>`.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// The rendered files, at the top of the copy beside the copied CLAUDE.md and .claude.
const (
	InstructionsFile = "instructions.md"
	AgentsFile       = "agents.json"
	HooksFile        = "hooks.json"
)

// InjectNone is workflow.inject's value for a session that takes nothing from the repository.
const InjectNone = "none"

// listed are the paths the copy lists on the default branch (the companion's step 1).
var listed = []string{"CLAUDE.md", ".claude"}

// Dir is a session's copy, <root>/workflow/<session id> (ADR 0025 rule 5).
func Dir(root, id string) string {
	return filepath.ToSlash(filepath.Join(root, "workflow", id))
}

// Options is one copy: the project's canonical and default branch and its workflow key.
type Options struct {
	Canonical string
	Branch    string
	Inject    string   // repository ("" reads the same) or none
	Hooks     []string // the events hooks.json keeps
}

// Skip is a file the render left out, and why.
type Skip struct {
	File string `json:"file"`
	Why  string `json:"why"`
}

// Copy is what Render made.
type Copy struct {
	Dir    string
	Branch string
	Inject string
	Commit string    // the default branch's commit; "" under inject none
	Date   time.Time // its committer date
	Listed []string  // the paths copied; none is the header-alone case
	Files  []string  // the rendered files written, slash paths under Dir, in render order
	Agents []string  // agents.json's names, in file-name order
	Hooks  []Hook    // hooks.json's entries
	Skips  []Skip
}

// None reports whether the session takes nothing from the repository: inject none, or a
// default branch with neither CLAUDE.md nor .claude.
func (c Copy) None() bool { return c.Inject == InjectNone || len(c.Listed) == 0 }

// Line is the value of the `workflow:` line (05; `session show`, P19): `<commit> <date>`, or
// `none (...)` saying why nothing was copied.
func (c Copy) Line() string {
	switch {
	case c.Inject == InjectNone:
		return "none (inject none)"
	case len(c.Listed) == 0:
		return "none (" + c.Branch + " at " + c.Commit + " has no CLAUDE.md or .claude)"
	}
	return c.Commit + " " + c.Date.UTC().Format(time.RFC3339)
}

// Render makes the copy in out, which must be absent or empty: it lists CLAUDE.md and .claude on
// the canonical's default branch, archives only what was listed, extracts the tar stream itself,
// then writes instructions.md (always; without the header, which the session adds), agents.json
// (when an agent is defined) and hooks.json (when an entry of a listed event is). Under inject
// none it writes an empty instructions.md and reads nothing.
func Render(ctx context.Context, o Options, out string) (Copy, error) {
	c := Copy{Dir: filepath.ToSlash(out), Branch: o.Branch, Inject: o.Inject}
	if err := prepareOut(out); err != nil {
		return c, err
	}
	if o.Inject != InjectNone {
		if err := c.fetch(ctx, o); err != nil {
			return c, err
		}
	}
	if err := c.render(o.Hooks); err != nil {
		return c, err
	}
	return c, nil
}

// prepareOut refuses an out that holds anything, so no file of an earlier render survives
// into this one, and makes it when absent.
func prepareOut(out string) error {
	entries, err := os.ReadDir(out)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.MkdirAll(out, 0o755); err != nil {
			return &Error{CodeOut, err.Error()}
		}
		return nil
	case err != nil:
		return &Error{CodeOut, err.Error()}
	case len(entries) > 0:
		return &Error{CodeOut, filepath.ToSlash(out) + " is not empty"}
	}
	return nil
}

// fetch is the companion's steps 1 and 3: the commit and its date, the listing, the archive of
// what was listed, extracted into c.Dir.
func (c *Copy) fetch(ctx context.Context, o Options) error {
	if o.Canonical == "" || o.Branch == "" {
		return &Error{CodeCanonical, "the project names no canonical or default branch"}
	}
	commit, err := gitText(ctx, o.Canonical, "rev-parse", "--verify", "--quiet", "refs/heads/"+o.Branch+"^{commit}")
	if err != nil {
		return &Error{CodeCanonical, o.Canonical + " has no branch " + o.Branch}
	}
	c.Commit = strings.TrimSpace(commit)
	when, err := gitText(ctx, o.Canonical, "show", "-s", "--format=%ct", c.Commit)
	if err != nil {
		return err
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(when), 10, 64)
	if err != nil {
		return &Error{CodeCanonical, "the committer date of " + c.Commit + ": " + err.Error()}
	}
	c.Date = time.Unix(secs, 0).UTC()
	names, err := gitText(ctx, o.Canonical, append([]string{"ls-tree", "-z", "--name-only", c.Commit, "--"}, listed...)...)
	if err != nil {
		return err
	}
	for _, n := range strings.Split(names, "\x00") {
		if n != "" {
			c.Listed = append(c.Listed, n)
		}
	}
	if len(c.Listed) == 0 {
		return nil // the header-alone case: git archive would exit 128 on an unmatched path
	}
	stream, err := gitBytes(ctx, o.Canonical, append([]string{"archive", "--format=tar", c.Commit, "--"}, c.Listed...)...)
	if err != nil {
		return err
	}
	return c.extract(stream)
}

// extract writes the tar stream's directories and regular files under c.Dir. A name outside
// the listed paths, or one filepath.IsLocal refuses on this host (absolute, climbing, a volume,
// a `:` stream, a device name), or holding a backslash (a separator on Windows, so the same tree
// reads alike on both hosts) is refused; git refuses such tree paths too, so this is the second
// line. A symbolic link is left out (the copy must not reach outside itself), as is everything
// else that is not a file or a directory.
func (c *Copy) extract(stream []byte) error {
	tr := tar.NewReader(bytes.NewReader(stream))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return &Error{CodeCanonical, "the archive: " + err.Error()}
		}
		if h.Typeflag == tar.TypeXGlobalHeader || h.Typeflag == tar.TypeXHeader {
			continue
		}
		name := strings.TrimSuffix(h.Name, "/")
		if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.Contains(name, `\`) ||
			!filepath.IsLocal(filepath.FromSlash(name)) || !underListed(name) {
			return &Error{CodeCanonical, "the archive names " + strconv.Quote(h.Name) + ", not a local path under CLAUDE.md or .claude"}
		}
		dst := filepath.Join(c.Dir, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return &Error{CodeOut, err.Error()}
			}
		case tar.TypeReg:
			if err := writeFile(dst, tr, h.Mode); err != nil {
				return err
			}
		case tar.TypeSymlink:
			c.Skips = append(c.Skips, Skip{name, "a symbolic link; not copied"})
		default:
			c.Skips = append(c.Skips, Skip{name, "not a file or a directory; not copied"})
		}
	}
}

func underListed(name string) bool {
	for _, l := range listed {
		if name == l || strings.HasPrefix(name, l+"/") {
			return true
		}
	}
	return false
}

func writeFile(dst string, r io.Reader, mode int64) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return &Error{CodeOut, err.Error()}
	}
	perm := fs.FileMode(0o644)
	if mode&0o111 != 0 {
		perm = 0o755
	}
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return &Error{CodeOut, err.Error()}
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return &Error{CodeOut, err.Error()}
	}
	if err := f.Close(); err != nil {
		return &Error{CodeOut, err.Error()}
	}
	return nil
}

// gitText and gitBytes run the bot's git in the canonical with the bot's environment minus the
// names spawn strips (a surrounding session's GIT_DIR would aim it elsewhere), never a prompt.
func gitText(ctx context.Context, dir string, args ...string) (string, error) {
	b, err := gitBytes(ctx, dir, args...)
	return string(b), err
}

func gitBytes(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = spawn.Environment(os.Environ(), []spawn.Var{{Name: "GIT_TERMINAL_PROMPT", Value: "0"}})
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		if msg == "" {
			msg = err.Error()
		}
		return nil, &Error{CodeCanonical, "git " + args[0] + ": " + strings.TrimSpace(msg)}
	}
	return stdout.Bytes(), nil
}
