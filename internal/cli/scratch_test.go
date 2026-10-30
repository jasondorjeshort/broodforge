// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The P9 Proof (08-implementation-plan.md) on the built binary. Every run gets a private OS
// temp directory, so the roots it makes, prunes and deletes are the test's own.

func scratchEnv(t *testing.T) (tmp string, env []string) {
	t.Helper()
	tmp = t.TempDir()
	return tmp, []string{"TMP=" + tmp, "TEMP=" + tmp, "TMPDIR=" + tmp}
}

func scratchRoots(t *testing.T, tmp string) []string {
	t.Helper()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if store.IsScratchName(e.Name()) {
			out = append(out, filepath.ToSlash(filepath.Join(tmp, e.Name())))
		}
	}
	return out
}

func scratchNew(t *testing.T, env []string, extra ...string) (root, name string) {
	t.Helper()
	r := clitest.Run(t, env, append([]string{"scratch", "new", "--json"}, extra...)...)
	var got struct{ Root, InstanceName string }
	if r.Code != 0 || r.Stderr != "" || json.Unmarshal([]byte(r.Stdout), &got) != nil {
		t.Fatalf("scratch new: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	if !store.IsScratchName(got.InstanceName) || filepath.Base(got.Root) != got.InstanceName {
		t.Fatalf("scratch new printed %+v", got)
	}
	return got.Root, got.InstanceName
}

func TestProofScratchNewCheckRehearseDelete(t *testing.T) {
	tmp, env := scratchEnv(t)
	root, _ := scratchNew(t, env)
	want(t, clitest.Run(t, env, "config", "check", "--root", root), 0, "ok  1  2\n", "")
	want(t, clitest.Run(t, env, "--root", root, "credential", "list"), 0, "cred-scratch  api-key  -  scratch  usable  -\n", "")
	if out, err := exec.Command("git", "-C", root+"/projects/scratch", "ls-remote", "origin", "main").Output(); err != nil || !strings.HasSuffix(strings.TrimSpace(string(out)), "refs/heads/main") {
		t.Errorf("origin has no main: %q %v", out, err)
	}

	if r := clitest.Run(t, env, "--root", root, "backup", "snapshot"); r.Code != 0 || !strings.HasSuffix(r.Stdout, "  1\n") {
		t.Fatalf("snapshot: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	r := clitest.Run(t, env, "--root", root, "backup", "rehearse", "--expect", "ok")
	if r.Code != 0 || !strings.HasPrefix(r.Stdout, "ok\nmigration  4\naudit_event  1\ncredential  1\n") || !strings.Contains(r.Stdout, "\nregistration  0\n") ||
		!strings.HasSuffix(r.Stdout, "worktree  0\nreconcile  0 missing\n") || r.Stderr != "" {
		t.Errorf("rehearse: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	if got := scratchRoots(t, tmp); len(got) != 1 || got[0] != root {
		t.Errorf("after the rehearsal: %v, want only %s", got, root)
	}
	r = clitest.Run(t, env, "--root", root, "--json", "backup", "rehearse", "--keep")
	var res struct{ Result, Root string }
	if r.Code != 0 || json.Unmarshal([]byte(r.Stdout), &res) != nil || res.Result != "ok" || res.Root == "" {
		t.Fatalf("rehearse --keep: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	want(t, clitest.Run(t, env, "config", "check", "--root", res.Root), 0, "ok  1  2\n", "")
	want(t, clitest.Run(t, env, "scratch", "delete", res.Root), 0, "removed  "+res.Root+"\n", "")

	// A session directory goes through worktree-remove, whose guard refuses a name that is not
	// a session id: the deletion stops and the root stays.
	if err := os.MkdirAll(root+"/sessions/s0001/tree/sub", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root+"/sessions/junk", 0o700); err != nil {
		t.Fatal(err)
	}
	r = clitest.Run(t, env, "scratch", "delete", root)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: worktree: ") {
		t.Errorf("delete with a bad session directory: exit %d stderr %q", r.Code, r.Stderr)
	}
	if _, err := os.Stat(root + "/config/instance.json"); err != nil {
		t.Errorf("refused deletion removed the config: %v", err)
	}
	if err := os.Remove(root + "/sessions/junk"); err != nil {
		t.Fatal(err)
	}
	want(t, clitest.Run(t, env, "scratch", "delete", root), 0, "removed  "+root+"\n", "")
	if got := scratchRoots(t, tmp); len(got) != 0 {
		t.Errorf("left: %v", got)
	}
}

func TestProofScratchRun(t *testing.T) {
	tmp, env := scratchEnv(t)
	want(t, clitest.Run(t, env, "scratch", "run", "--", "broodforge", "config", "check"), 0, "ok  1  2\n", "")
	if got := scratchRoots(t, tmp); len(got) != 0 {
		t.Errorf("exit 0 kept %v", got)
	}

	r := clitest.Run(t, env, "scratch", "run", "--", "broodforge", "backup", "verify")
	got := scratchRoots(t, tmp)
	if len(got) != 1 {
		t.Fatalf("exit 1 kept %v", got)
	}
	want(t, r, 1, "", "error: no-snapshot: no "+got[0]+"/backup/latest\nscratch root kept: "+got[0]+"\n")

	r = clitest.Run(t, env, "scratch", "run", "--keep", "--", "broodforge", "config", "check")
	if r.Code != 0 || r.Stdout != "ok  1  2\n" || !strings.HasPrefix(r.Stderr, "scratch root kept: ") || len(scratchRoots(t, tmp)) != 2 {
		t.Errorf("--keep: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	want(t, clitest.Run(t, env, "scratch", "run"), 2, "", "error: usage: scratch run: missing <command>\n")
	kept := scratchRoots(t, tmp)
	want(t, clitest.Run(t, env, "scratch", "prune", "--all"), 0, "removed  "+kept[0]+"\nremoved  "+kept[1]+"\n", "")
}

func TestProofScratchDeleteRefuses(t *testing.T) {
	_, env := scratchEnv(t)
	valid, err := filepath.Abs("../../tests/config/valid")
	if err != nil {
		t.Fatal(err)
	}
	valid = filepath.ToSlash(valid)
	want(t, clitest.Run(t, env, "scratch", "delete", valid), 1, "",
		"error: not-scratch: "+valid+": the directory is not named scratch-<12 hex>\n")
	if _, err := os.Stat(valid + "/config/instance.json"); err != nil {
		t.Fatal(err)
	}
	// A scratch name outside the temp directory.
	other := filepath.ToSlash(filepath.Join(t.TempDir(), "scratch-0123456789ab"))
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	r := clitest.Run(t, env, "scratch", "delete", other)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: not-scratch: "+other+": not directly under the OS temp directory") {
		t.Errorf("outside temp: exit %d stderr %q", r.Code, r.Stderr)
	}
}

func TestProofScratchPruneAndProject(t *testing.T) {
	tmp, env := scratchEnv(t)
	old, _ := scratchNew(t, env)
	young, _ := scratchNew(t, env, "--project", old+"/projects/scratch")
	if out, err := exec.Command("git", "-C", young+"/projects/scratch", "remote", "get-url", "origin").Output(); err != nil ||
		strings.TrimSpace(string(out)) != young+"/remote/scratch.git" {
		t.Errorf("--project origin %q %v", out, err)
	}
	if out, err := exec.Command("git", "-C", young+"/projects/scratch", "log", "--format=%s").Output(); err != nil || strings.TrimSpace(string(out)) != "scratch base" {
		t.Errorf("--project history %q %v", out, err)
	}
	then := time.Now().Add(-72 * time.Hour)
	if err := os.Chtimes(old+"/config/instance.json", then, then); err != nil {
		t.Fatal(err)
	}
	// A directory with a scratch name but another instance's config is refused, not removed.
	impostor := filepath.ToSlash(filepath.Join(tmp, "scratch-000000000000"))
	if err := os.MkdirAll(impostor+"/config", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(impostor+"/config/instance.json", []byte(`{"instanceName":"home"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(impostor+"/config/instance.json", then, then); err != nil {
		t.Fatal(err)
	}

	want(t, clitest.Run(t, env, "scratch", "prune", "--older-than", "2"), 0,
		"refused  "+impostor+"  not-scratch: "+impostor+": instanceName \"home\" is not the directory's name\nremoved  "+old+"\n", "")
	want(t, clitest.Run(t, env, "scratch", "prune", "--older-than", "2"), 0,
		"refused  "+impostor+"  not-scratch: "+impostor+": instanceName \"home\" is not the directory's name\n", "")
	want(t, clitest.Run(t, env, "scratch", "prune"), 0,
		"refused  "+impostor+"  not-scratch: "+impostor+": instanceName \"home\" is not the directory's name\n", "")
	want(t, clitest.Run(t, env, "scratch", "prune", "--all"), 0,
		"refused  "+impostor+"  not-scratch: "+impostor+": instanceName \"home\" is not the directory's name\nremoved  "+young+"\n", "")
	if got := scratchRoots(t, tmp); len(got) != 1 || got[0] != impostor {
		t.Errorf("left %v, want only the impostor", got)
	}
}

// tree reads every file and directory under dir, so a refused deletion can be shown to leave a
// directory exactly as it was.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			out[filepath.ToSlash(rel)] = "<dir>"
			return nil
		}
		b, err := os.ReadFile(p)
		out[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sameTree(t *testing.T, what string, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Errorf("%s: %d entries before, %d after: %v", what, len(before), len(after), after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Errorf("%s: %s changed: %q, now %q", what, k, v, after[k])
		}
	}
}

// plant makes dir with no config/instance.json and an interrupted last-good write beside it
// (data/config-last-good.old-1), which config.InstanceName would rename back.
func plant(t *testing.T, dir, name string) {
	t.Helper()
	old := filepath.Join(dir, "data", "config-last-good.old-1")
	if err := os.MkdirAll(old, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "instance.json"), []byte(`{"instanceName":"`+name+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProofScratchRefusalsTouchNothing(t *testing.T) {
	tmp, env := scratchEnv(t)
	not := filepath.ToSlash(filepath.Join(tmp, "notscratch"))
	plant(t, not, "notscratch")
	before := tree(t, not)
	r := clitest.Run(t, env, "scratch", "delete", not)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: not-scratch: ") {
		t.Errorf("delete notscratch: exit %d stderr %q", r.Code, r.Stderr)
	}
	sameTree(t, "notscratch", before, tree(t, not))

	// Named like a root, its only name in a last-good copy: refused by delete and by prune,
	// and nothing renamed.
	alike := filepath.ToSlash(filepath.Join(tmp, "scratch-eeeeeeeeeeee"))
	plant(t, alike, "scratch-eeeeeeeeeeee")
	before = tree(t, alike)
	want(t, clitest.Run(t, env, "scratch", "delete", alike), 1, "",
		"error: not-scratch: "+alike+": instanceName \"\" is not the directory's name\n")
	sameTree(t, "delete scratch-eeee", before, tree(t, alike))

	// A link named like a root is listed and refused, its target untouched.
	victim := t.TempDir()
	if err := os.WriteFile(filepath.Join(victim, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.ToSlash(filepath.Join(tmp, "scratch-dddddddddddd"))
	if err := os.Symlink(victim, linked); err != nil {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.FromSlash(linked), victim).CombinedOutput(); err != nil {
			t.Skipf("cannot make a link: %v %s", err, out)
		}
	}
	want(t, clitest.Run(t, env, "scratch", "prune", "--all"), 0,
		"refused  "+linked+"  link\nrefused  "+alike+"  not-scratch: "+alike+": instanceName \"\" is not the directory's name\n", "")
	sameTree(t, "prune scratch-eeee", before, tree(t, alike))
	if b, err := os.ReadFile(filepath.Join(victim, "keep.txt")); err != nil || string(b) != "keep" {
		t.Errorf("link target touched: %q %v", b, err)
	}
}

func TestProofScratchProjectCloneFails(t *testing.T) {
	tmp, env := scratchEnv(t)
	missing := filepath.Join(t.TempDir(), "no-such-repo")
	r := clitest.Run(t, env, "scratch", "new", "--project", missing)
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: scratch-create: git clone: ") ||
		strings.Contains(r.Stderr, `\`) || strings.Contains(r.Stderr, `"fatal`) {
		t.Errorf("clone of a missing repository: exit %d stderr %q", r.Code, r.Stderr)
	}
	if got := scratchRoots(t, tmp); len(got) != 0 {
		t.Errorf("a failed scratch new left %v", got)
	}
}
