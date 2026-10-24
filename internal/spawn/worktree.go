// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Codes of the worktree operations.
const (
	CodeWorktree = "worktree"        // worktree-create or worktree-remove failed
	CodeBundle   = "worktree-bundle" // worktree-bundle failed
)

// Worktree is one session's clone, the operand of worktree-create, worktree-bundle and
// worktree-remove (ADR 0022 companion, Clone at session start; Handoff at session end). The
// backend runs these as the engine side: on the same account, as the bot's own account.
type Worktree struct {
	Canonical  string // the project's canonical checkout, bot-owned; the clone's source
	Session    string // <root>/sessions/<id>: the tree is Session/tree, the bundle Session/out.bundle
	BaseBranch string // repo.defaultBranch, the branch cloned
	Base       string // the base commit, refs/sessions/<id> in the canonical at creation
	Branch     string // the session branch, broodforge/<instanceName>/<id>
}

// Tree is <sess>/tree.
func (w Worktree) Tree() string { return filepath.ToSlash(filepath.Join(w.Session, "tree")) }

// BundlePath is <sess>/out.bundle.
func (w Worktree) BundlePath() string {
	return filepath.ToSlash(filepath.Join(w.Session, "out.bundle"))
}

// Bundle is what worktree-bundle wrote: Path is "" when the session has no commits.
type Bundle struct {
	Path    string
	Commits int
}

// sessionDirName is git.ValidSessionID's form; git imports spawn, so spawn keeps this one copy.
var sessionDirName = regexp.MustCompile(`^s[0-9a-z]{4,}$`)

// checkSessionDir refuses anything but an absolute <root>/sessions/s<id> written in its clean
// form (no `..`, nothing Clean would change), so a bad record can never point an operation
// elsewhere, and refuses a session directory or sessions/ that is a symbolic link, a junction
// or another reparse point, through which a removal would reach another tree.
func checkSessionDir(dir string) error {
	bad := func(why string) error {
		return &Error{CodeWorktree, quote(filepath.ToSlash(dir)) + " is not <root>/sessions/<session id>" + why}
	}
	clean := filepath.Clean(dir)
	if filepath.ToSlash(clean) != filepath.ToSlash(dir) || strings.Contains(filepath.ToSlash(dir), "..") {
		return bad(": not a clean path")
	}
	if !filepath.IsAbs(clean) || filepath.Base(filepath.Dir(clean)) != "sessions" || !sessionDirName.MatchString(filepath.Base(clean)) {
		return bad("")
	}
	for _, p := range []string{filepath.Dir(clean), clean} {
		fi, err := os.Lstat(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return &Error{CodeWorktree, err.Error()}
		}
		if fi.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 || !fi.IsDir() {
			return bad(": " + quote(filepath.ToSlash(p)) + " is a link or not a directory")
		}
	}
	return nil
}

func checkWorktree(w Worktree) error {
	if err := checkSessionDir(w.Session); err != nil {
		return err
	}
	if w.Canonical == "" || w.BaseBranch == "" || w.Base == "" || w.Branch == "" {
		return &Error{CodeWorktree, "a worktree needs its canonical, base branch, base commit and branch"}
	}
	return nil
}

// runGit runs git with the bot's environment minus the stripped names (Environment), never a
// terminal prompt, and returns stdout; a failure carries git's stderr.
func runGit(code string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = Environment(os.Environ(), []Var{{"GIT_TERMINAL_PROMPT", "0"}})
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", &Error{code, "git " + subcommand(args) + ": " + firstLine(msg)}
	}
	return stdout.String(), nil
}

// subcommand is the git subcommand after any -C or -c options, for messages.
func subcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-C" || args[i] == "-c" {
			i++
			continue
		}
		return args[i]
	}
	return ""
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(line)
}

// HooksDir is <bf>/share/hooks, which worktree-create names as the clone's core.hooksPath.
func (s *SameAccount) HooksDir() string {
	return filepath.ToSlash(filepath.Join(s.install, "share", "hooks"))
}

// Askpass is the GIT_ASKPASS helper for this host: <bf>/share/askpass.sh, or askpass.cmd on
// Windows (askpass_*.go).
func (s *SameAccount) Askpass() string {
	return filepath.ToSlash(filepath.Join(s.install, "share", askpassScript))
}

// WorktreeCreate clones the canonical into <sess>/tree (`git -c safe.directory=<canon> clone
// --shared --branch <base branch>`), checks out the session branch at the base commit and sets
// core.hooksPath to HooksDir. An existing tree is refused; a failed step removes the tree.
func (s *SameAccount) WorktreeCreate(w Worktree) error {
	if err := checkWorktree(w); err != nil {
		return err
	}
	tree := w.Tree()
	if _, err := os.Lstat(tree); err == nil {
		return &Error{CodeWorktree, tree + " exists"}
	}
	if err := os.MkdirAll(w.Session, 0o750); err != nil {
		return &Error{CodeWorktree, err.Error()}
	}
	steps := [][]string{
		{"-c", "safe.directory=" + filepath.ToSlash(w.Canonical), "clone", "--quiet", "--shared", "--branch", w.BaseBranch, w.Canonical, tree},
		{"-C", tree, "checkout", "--quiet", "-b", w.Branch, w.Base},
		{"-C", tree, "config", "core.hooksPath", s.HooksDir()},
	}
	for _, args := range steps {
		if _, err := runGit(CodeWorktree, args...); err != nil {
			_ = os.RemoveAll(tree)
			return err
		}
	}
	return nil
}

// WorktreeBundle writes <sess>/out.bundle of <base>..<branch> (`git -C <sess>/tree bundle
// create`); a session with no commits since the base writes nothing, removes a stale bundle and
// reports zero.
func (s *SameAccount) WorktreeBundle(w Worktree) (Bundle, error) {
	if err := checkWorktree(w); err != nil {
		return Bundle{}, err
	}
	tree, rng := w.Tree(), w.Base+"..refs/heads/"+w.Branch
	out, err := runGit(CodeBundle, "-C", tree, "rev-list", "--count", rng)
	if err != nil {
		return Bundle{}, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return Bundle{}, &Error{CodeBundle, "rev-list --count: " + err.Error()}
	}
	path := w.BundlePath()
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Bundle{}, &Error{CodeBundle, err.Error()}
	}
	if n == 0 {
		return Bundle{}, nil
	}
	if _, err := runGit(CodeBundle, "-C", tree, "bundle", "create", "--quiet", path, rng); err != nil {
		return Bundle{}, err
	}
	return Bundle{Path: path, Commits: n}, nil
}

// prePushHook is the hook under HooksDir that refuses every push from a session tree (ADR 0025,
// rule 4).
const prePushHook = "pre-push"

// WorktreePrePush reports whether the session's tree still runs the bot's pre-push hook: its
// core.hooksPath, as git resolves it in the tree (`git -C <sess>/tree config --get`), is
// HooksDir exactly, and HooksDir/pre-push is a regular file, executable off Windows (git skips a
// hook without the bit there). An unset or different hooks path is false, not an error.
func (s *SameAccount) WorktreePrePush(session string) (bool, error) {
	if err := checkSessionDir(session); err != nil {
		return false, err
	}
	w := Worktree{Session: session}
	out, err := runGit(CodeWorktree, "-C", w.Tree(), "config", "--default", "", "--get", "core.hooksPath")
	if err != nil {
		return false, err
	}
	// Only git's own line end is stripped: a value with other whitespace around it is a path
	// git does not find the hook under, so it must not compare equal.
	if strings.TrimSuffix(out, "\n") != s.HooksDir() {
		return false, nil
	}
	fi, err := os.Stat(filepath.Join(s.HooksDir(), prePushHook))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, &Error{CodeWorktree, err.Error()}
	}
	return fi.Mode().IsRegular() && (runtime.GOOS == "windows" || fi.Mode().Perm()&0o111 != 0), nil
}

// WorktreeRemove deletes <sess>/tree and <sess>/out.bundle; the session's home stays. A missing
// tree is not an error. os.RemoveAll removes a junction or symbolic link inside the tree as the
// link, not its target, and clears read-only files on Windows.
func (s *SameAccount) WorktreeRemove(session string) error {
	if err := checkSessionDir(session); err != nil {
		return err
	}
	w := Worktree{Session: session}
	if err := os.RemoveAll(w.Tree()); err != nil {
		return &Error{CodeWorktree, err.Error()}
	}
	if err := os.Remove(w.BundlePath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return &Error{CodeWorktree, err.Error()}
	}
	return nil
}
