// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The scratch commands (05-cli-conventions.md, Scratch; ADR 0021, Scratch). All four are
// standalone: they make or remove roots and need no --root. A root is <temp>/scratch-<12 hex>
// with config.ScratchTemplate's configuration, the store key under the root, a store with no
// registrations and credential cred-scratch (api-key, value "scratch", the fake's own) as an
// instance credential (no user), and project `scratch`: a canonical checkout under projects/ whose
// origin is a bare repository under remote/, so a push works offline.

// Error codes of the scratch commands, beside store's not-scratch.
const (
	CodeScratchCreate = "scratch-create" // making a root failed; nothing is left behind
	CodeScratchRun    = "scratch-run"    // the command did not start; the root is kept
)

// The seeded credential (05-cli-conventions.md, Scratch).
const (
	ScratchCredential = "cred-scratch"
	ScratchValue      = "scratch"
)

// keptLine is the stderr line naming a scratch root left behind.
const keptLine = "scratch root kept: "

// scratchNew is `broodforge scratch new [--project <path or url>]`: makes a root and prints its
// path; --json prints {"root":...,"instanceName":...}.
var scratchNew = Command{
	Name:    "scratch new",
	Class:   ClassStandalone,
	Flags:   []Flag{projectFlag},
	Columns: []string{"root"},
	Local:   runScratchNew,
}

// scratchRun is `broodforge scratch run [--keep] [--project <path or url>] -- <command...>`:
// makes a root, runs the command with BROODFORGE_ROOT naming it (a first word `broodforge`
// is this binary), and exits with the command's code. The root is deleted on exit 0 unless
// --keep; otherwise, or when the deletion fails, it is kept and named on stderr.
var scratchRun = Command{
	Name:  "scratch run",
	Class: ClassStandalone,
	Flags: []Flag{
		{Name: "keep", Kind: FlagBool, Usage: "keep the root on success too"},
		projectFlag,
		{Name: "command", Kind: FlagString},
	},
	Local: runScratchRun,
}

// scratchDelete is `broodforge scratch delete <path>`: `removed  <root>`; anything but a
// scratch root is `error: not-scratch: <root>: <why>`, exit 1, and nothing is removed.
var scratchDelete = Command{
	Name:    "scratch delete",
	Class:   ClassStandalone,
	Flags:   []Flag{{Name: "path", Kind: FlagString}},
	Columns: []string{"result", "root"},
	Local:   runScratchDelete,
}

// scratchPrune is `broodforge scratch prune [--older-than <days>] [--all]`: removes the scratch
// roots in the temp directory created more than --older-than days ago (default 1), or every
// one with --all, printing `removed  <root>` each, or `refused  <root>  <why>` for one the
// scratch check refuses or that is a link (`refused  <root>  link`; still exit 0). Younger
// roots are not printed.
var scratchPrune = Command{
	Name:  "scratch prune",
	Class: ClassStandalone,
	Flags: []Flag{
		{Name: "older-than", Kind: FlagInt, Default: "1", Value: "<days>", Usage: "remove roots created more than this many days ago"},
		{Name: "all", Kind: FlagBool, Usage: "remove every scratch root, whatever its age"},
	},
	Local: runScratchPrune,
}

var projectFlag = Flag{Name: "project", Kind: FlagString, Value: "<path or url>",
	Usage: "clone this repository as project scratch (default: a generated one)"}

func init() {
	commands = append(commands, &scratchNew, &scratchRun, &scratchDelete, &scratchPrune)
	positionals[&scratchRun] = []string{"command"}
	variadic[&scratchRun] = true
	positionals[&scratchDelete] = []string{"path"}
}

// scratchLine is one record of delete and prune.
type scratchLine struct {
	Result string `json:"result"` // removed or refused
	Root   string `json:"root"`
	Reason string `json:"reason,omitempty"`
}

func runScratchNew(_ context.Context, env *Env) error {
	sc, err := createScratch(env.String("project"))
	if err != nil {
		return err
	}
	out, err := encode(sc)
	if err != nil {
		return err
	}
	return env.Out.Result(out, []string{"root"})
}

func runScratchRun(ctx context.Context, env *Env) error {
	argv := env.Strings("command")
	if len(argv) == 0 {
		return &usageError{"scratch run: missing <command...>"}
	}
	sc, err := createScratch(env.String("project"))
	if err != nil {
		return err
	}
	if argv[0] == "broodforge" {
		exe, err := os.Executable()
		if err != nil {
			return keep(env, sc, control.NewError(CodeScratchRun, err.Error()))
		}
		argv[0] = exe
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), RootEnv+"=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, RootEnv+"="+filepath.FromSlash(sc.Root))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = env.Stdin, env.Out.w, env.Stderr
	code := 0
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return keep(env, sc, control.NewError(CodeScratchRun, err.Error()))
		}
		code = ee.ExitCode()
		if code < 0 { // killed by a signal
			code = ExitFailure
		}
	}
	if code != 0 || env.Bool("keep") {
		fmt.Fprintln(env.Stderr, keptLine+sc.Root)
		if code != 0 {
			return &exitError{code}
		}
		return nil
	}
	if err := store.DeleteScratch(sc.Root, store.FixedName(sc.InstanceName), removeTree); err != nil {
		return keep(env, sc, scratchError(err))
	}
	return nil
}

// keep names the kept root on stderr and returns err for the error line.
func keep(env *Env, sc store.Scratch, err error) error {
	fmt.Fprintln(env.Stderr, keptLine+sc.Root)
	return err
}

func runScratchDelete(_ context.Context, env *Env) error {
	root := env.String("path")
	abs, err := filepath.Abs(root)
	if err != nil {
		return control.NewError(store.CodeNotScratch, filepath.ToSlash(root)+": "+err.Error())
	}
	// The place is checked before the name is read, and the name is read without last-good
	// recovery, so a refused path is left exactly as it was.
	if err := store.DeleteScratch(abs, config.PlainInstanceName, removeTree); err != nil {
		return scratchError(err)
	}
	out, err := encode(scratchLine{Result: "removed", Root: filepath.ToSlash(abs)})
	if err != nil {
		return err
	}
	return env.Out.Record(out, []string{"result", "root"})
}

func runScratchPrune(_ context.Context, env *Env) error {
	days := env.Int("older-than")
	if days < 0 {
		return &usageError{"scratch prune: --older-than wants 0 or more days"}
	}
	entries, err := store.ListScratch()
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	for _, e := range entries {
		if !env.Bool("all") && !e.Created.Before(cutoff) {
			continue
		}
		line, cols := scratchLine{Result: "removed", Root: e.Root}, []string{"result", "root"}
		if e.Link {
			line.Result, line.Reason = "refused", "link"
			cols = append(cols, "reason")
		} else if err := store.DeleteScratch(e.Root, config.PlainInstanceName, removeTree); err != nil {
			line.Result, line.Reason = "refused", control.OneLine(scratchError(err).Error())
			cols = append(cols, "reason")
		}
		out, err := encode(line)
		if err != nil {
			return err
		}
		if err := env.Out.Record(out, cols); err != nil {
			return err
		}
	}
	return nil
}

// removeTree is the worktree-removal interface every scratch deletion runs on its session
// directories: the same-account backend's, which scratch roots use on both hosts.
func removeTree(sessionDir string) error { return (&spawn.SameAccount{}).WorktreeRemove(sessionDir) }

// scratchError maps the coded refusals of store and spawn to the CLI's error line.
func scratchError(err error) error {
	var se *spawn.Error
	if errors.As(err, &se) {
		return control.NewError(se.Code, se.Message)
	}
	return localError(err)
}

// createScratch makes a complete scratch root; on any failure it removes what it made.
func createScratch(project string) (store.Scratch, error) {
	sc, err := store.NewScratch()
	if err != nil {
		return sc, control.NewError(CodeScratchCreate, err.Error())
	}
	if err := fillScratch(sc, project); err != nil {
		_ = store.DeleteScratch(sc.Root, store.FixedName(sc.InstanceName), removeTree)
		var ce *control.Error
		if errors.As(err, &ce) {
			return sc, err
		}
		return sc, control.NewError(CodeScratchCreate, control.OneLine(err.Error()))
	}
	return sc, nil
}

func fillScratch(sc store.Scratch, project string) error {
	install, fixtures := scratchInstall()
	branch, err := scratchProject(sc.Root, project)
	if err != nil {
		return err
	}
	o := config.ScratchOptions{Root: sc.Root, InstanceName: sc.InstanceName, Install: install,
		Fixtures: fixtures, DefaultBranch: branch}
	if err := config.WriteScratch(o); err != nil {
		return err
	}
	keyPath, err := credentials.KeyPath(sc.Root, sc.InstanceName, config.ScratchKeyFile, os.LookupEnv)
	if err != nil {
		return err
	}
	if _, err := credentials.Init(keyPath); err != nil {
		return err
	}
	if err := seedScratch(sc.Root, keyPath); err != nil {
		return err
	}
	if _, err := config.Load(sc.Root, credentialExists); err != nil {
		return err
	}
	return nil
}

// seedScratch creates the store and records cred-scratch as an instance credential (user_id
// null, the instance's own; a scratch root has no registrations), its value sealed in the
// credential store, with its credential-added audit event.
func seedScratch(root, keyPath string) error {
	v, err := credentials.OpenVault(credentials.StorePath(root), keyPath)
	if err != nil {
		return err
	}
	if err := v.Put(ScratchCredential, []byte(ScratchValue)); err != nil {
		return err
	}
	st, err := store.Create(root)
	if err != nil {
		return err
	}
	now := store.FormatTime(time.Now())
	aud := audit.NewWriter(nil)
	err = st.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec(`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES (?, ?, ?, NULL, ?, ?, ?, ?)`, store.Text(ScratchCredential), store.Text(credentials.KindAPIKey),
			store.Text(credentials.ProviderAnthropic), store.Text("scratch"),
			store.Text(credentials.StatusUsable), store.Text(now), store.Text(ScratchCredential)); err != nil {
			return err
		}
		cred, _ := json.Marshal(map[string]string{"id": ScratchCredential, "kind": credentials.KindAPIKey, "label": "scratch"})
		_, err := aud.Append(tx, audit.Event{Kind: audit.KindCredentialAdded, Actor: "host", Details: cred})
		return err
	})
	return errors.Join(err, st.Close())
}

// scratchInstall finds paths.install and the fake's fixtures: the first of the binary's
// install directory (the parent of bin/), the working directory and its parents that holds
// fixtures/engine and share/, that is, a checkout of this repository; with none, the binary's
// install directory.
func scratchInstall() (install, fixtures string) {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		candidates = append(candidates, filepath.Dir(filepath.Dir(exe)))
	}
	if wd, err := os.Getwd(); err == nil {
		for d := wd; ; d = filepath.Dir(d) {
			candidates = append(candidates, d)
			if filepath.Dir(d) == d {
				break
			}
		}
	}
	for _, d := range candidates {
		f, fe := os.Stat(filepath.Join(d, "fixtures", "engine"))
		s, se := os.Stat(filepath.Join(d, "share"))
		if fe == nil && se == nil && f.IsDir() && s.IsDir() {
			return filepath.ToSlash(d), filepath.ToSlash(filepath.Join(d, "fixtures", "engine"))
		}
	}
	if len(candidates) == 0 {
		return "", ""
	}
	d := candidates[0]
	return filepath.ToSlash(d), filepath.ToSlash(filepath.Join(d, "fixtures", "engine"))
}

// scratchProject makes projects/scratch, the canonical, and remote/scratch.git, its bare
// origin holding the default branch, and returns that branch: a generated repository on main
// with one commit, or a clone of src on the branch its HEAD names.
func scratchProject(root, src string) (string, error) {
	canonical := filepath.ToSlash(filepath.Join(root, "projects", config.ScratchProject))
	remote := filepath.ToSlash(filepath.Join(root, "remote", config.ScratchProject+".git"))
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		return "", err
	}
	branch := "main"
	if src == "" {
		if err := os.MkdirAll(canonical, 0o700); err != nil {
			return "", err
		}
		if _, err := scratchGit(canonical, "init", "--quiet", "-b", branch); err != nil {
			return "", err
		}
		readme := "Throwaway project of a Broodforge scratch root.\n"
		if err := os.WriteFile(filepath.Join(canonical, "README"), []byte(readme), 0o600); err != nil {
			return "", err
		}
		if _, err := scratchGit(canonical, "add", "README"); err != nil {
			return "", err
		}
		if _, err := scratchGit(canonical, "commit", "--quiet", "-m", "scratch base"); err != nil {
			return "", err
		}
	} else {
		if _, err := scratchGit(root, "clone", "--quiet", src, canonical); err != nil {
			return "", err
		}
		out, err := scratchGit(canonical, "symbolic-ref", "--short", "HEAD")
		if err != nil {
			return "", err
		}
		branch = strings.TrimSpace(out)
		if _, err := scratchGit(canonical, "remote", "remove", "origin"); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(remote, 0o700); err != nil {
		return "", err
	}
	if _, err := scratchGit(remote, "init", "--quiet", "--bare", "-b", branch); err != nil {
		return "", err
	}
	if _, err := scratchGit(canonical, "remote", "add", "origin", remote); err != nil {
		return "", err
	}
	if _, err := scratchGit(canonical, "push", "--quiet", "origin", branch); err != nil {
		return "", err
	}
	return branch, nil
}

// scratchGit runs git in dir with the inherited environment minus the names spawn strips, no
// prompt, no signing and no hooks, committing as the scratch identity.
func scratchGit(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir, "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + os.DevNull}, args...)
	cmd := exec.Command("git", full...)
	id := []spawn.Var{{Name: "GIT_TERMINAL_PROMPT", Value: "0"},
		{Name: "GIT_AUTHOR_NAME", Value: "Broodforge scratch"}, {Name: "GIT_AUTHOR_EMAIL", Value: "scratch@example.com"},
		{Name: "GIT_COMMITTER_NAME", Value: "Broodforge scratch"}, {Name: "GIT_COMMITTER_EMAIL", Value: "scratch@example.com"}}
	cmd.Env = spawn.Environment(os.Environ(), id)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		line, _, _ := strings.Cut(msg, "\n")
		return "", control.NewError(CodeScratchCreate, "git "+args[0]+": "+filepath.ToSlash(strings.TrimSpace(line)))
	}
	return stdout.String(), nil
}
