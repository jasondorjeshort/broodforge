// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// privateTemp points the OS temp directory at a fresh directory for the test, so scratch roots
// made, listed and deleted here never meet anyone else's.
func privateTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TMP", dir)
	t.Setenv("TEMP", dir)
	return dir
}

func wantNotScratch(t *testing.T, err error, part string) {
	t.Helper()
	var se *Error
	if !errors.As(err, &se) || se.Code != CodeNotScratch || !strings.Contains(se.Message, part) {
		t.Fatalf("got %v, want not-scratch containing %q", err, part)
	}
}

func TestNewScratchAndDelete(t *testing.T) {
	tmp := privateTemp(t)
	sc, err := NewScratch()
	if err != nil {
		t.Fatal(err)
	}
	if !IsScratchName(sc.InstanceName) || filepath.Base(sc.Root) != sc.InstanceName || strings.Contains(sc.Root, `\`) {
		t.Fatalf("scratch %+v", sc)
	}
	if err := CheckScratch(sc.Root, FixedName(sc.InstanceName)); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"sessions/s0001/tree", "sessions/s0002", "data"} {
		if err := os.MkdirAll(filepath.Join(sc.Root, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// The key file is read-only, as credential init leaves it; deletion removes it anyway.
	key := filepath.Join(sc.Root, "key.txt")
	if err := os.WriteFile(key, []byte("k"), 0o400); err != nil {
		t.Fatal(err)
	}

	// A refusal by the worktree removal stops the deletion: the root stays.
	refuse := errors.New("refused")
	if err := DeleteScratch(sc.Root, FixedName(sc.InstanceName), func(string) error { return refuse }); !errors.Is(err, refuse) {
		t.Fatalf("got %v, want the removal's refusal", err)
	}
	if _, err := os.Stat(sc.Root); err != nil {
		t.Fatalf("root gone after a refusal: %v", err)
	}

	var removed []string
	err = DeleteScratch(sc.Root, FixedName(sc.InstanceName), func(dir string) error {
		removed = append(removed, strings.TrimPrefix(dir, sc.Root+"/"))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(removed, ",") != "sessions/s0001,sessions/s0002" {
		t.Errorf("worktree removal ran on %q", removed)
	}
	if _, err := os.Stat(sc.Root); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("root still there: %v", err)
	}
	entries, _ := os.ReadDir(tmp)
	if len(entries) != 0 {
		t.Errorf("temp directory not empty: %v", entries)
	}
}

func TestCheckScratchRefuses(t *testing.T) {
	tmp := privateTemp(t)
	sc, err := NewScratch()
	if err != nil {
		t.Fatal(err)
	}
	wantNotScratch(t, CheckScratch(sc.InstanceName, FixedName(sc.InstanceName)), "not an absolute path")
	wantNotScratch(t, CheckScratch(sc.Root, FixedName("home")), `instanceName "home"`)
	wantNotScratch(t, CheckScratch(sc.Root, FixedName("")), `instanceName ""`)
	wantNotScratch(t, CheckScratch(tmp, FixedName("scratch-000000000000")), "not named scratch-")

	other := filepath.Join(t.TempDir(), "scratch-000000000000") // a scratch name outside temp
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	wantNotScratch(t, CheckScratch(other, FixedName("scratch-000000000000")), "not directly under the OS temp directory")
	nested := filepath.Join(sc.Root, "scratch-111111111111")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	wantNotScratch(t, CheckScratch(nested, FixedName("scratch-111111111111")), "not directly under")
	wantNotScratch(t, DeleteScratch(other, FixedName("scratch-000000000000"), func(string) error { return nil }), "not directly under")
	if _, err := os.Stat(other); err != nil {
		t.Errorf("a refused deletion removed %s", other)
	}

	linked := filepath.Join(tmp, "scratch-222222222222")
	link(t, other, linked)
	wantNotScratch(t, CheckScratch(linked, FixedName("scratch-222222222222")), "a link")
}

func TestListScratch(t *testing.T) {
	tmp := privateTemp(t)
	a, err := NewScratch()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewScratch()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(b.Root, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	inst := filepath.Join(b.Root, "config", "instance.json")
	if err := os.WriteFile(inst, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-72 * time.Hour)
	if err := os.Chtimes(inst, old, old); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"scratch-xyz", "other"} { // not scratch names: not listed
		if err := os.Mkdir(filepath.Join(tmp, n), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ListScratch()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("listed %+v, want %s and %s", got, a.InstanceName, b.InstanceName)
	}
	for _, e := range got {
		age := time.Since(e.Created)
		switch e.Name {
		case b.InstanceName:
			if age < 71*time.Hour {
				t.Errorf("%s: age %v, want instance.json's", e.Name, age)
			}
		case a.InstanceName:
			if age > time.Hour {
				t.Errorf("%s: age %v, want the directory's", e.Name, age)
			}
		default:
			t.Errorf("listed %s", e.Name)
		}
	}
}

// link makes at a directory link to target: a symbolic link, or on Windows without the
// privilege a junction.
func link(t *testing.T, target, at string) {
	t.Helper()
	if err := os.Symlink(target, at); err == nil {
		return
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.FromSlash(at), filepath.FromSlash(target)).CombinedOutput(); err != nil {
		t.Skipf("cannot make a symbolic link or a junction: %v %s", err, out)
	}
}

// victim is a directory outside every root holding keep.txt, which no deletion may reach.
func victim(t *testing.T) (dir, keep string) {
	t.Helper()
	dir = t.TempDir()
	keep = filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, keep
}

func TestDeleteScratchLinksGoThroughTheGuard(t *testing.T) {
	privateTemp(t)
	v, keep := victim(t)

	// A link under sessions/ is handed to the worktree removal, whose guard refuses it.
	sc, err := NewScratch()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(sc.Root, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	link(t, v, filepath.Join(sc.Root, "sessions", "s0001"))
	refused := errors.New("guard: a link")
	var got []string
	err = DeleteScratch(sc.Root, FixedName(sc.InstanceName), func(dir string) error {
		got = append(got, dir)
		return refused
	})
	if !errors.Is(err, refused) || len(got) != 1 || got[0] != sc.Root+"/sessions/s0001" {
		t.Errorf("link under sessions/: err %v, removal ran on %v", err, got)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("victim touched: %v", err)
	}
	if _, err := os.Stat(sc.Root); err != nil {
		t.Errorf("root removed after a refusal: %v", err)
	}

	// sessions/ itself a link is refused before anything runs.
	sc2, err := NewScratch()
	if err != nil {
		t.Fatal(err)
	}
	link(t, v, filepath.Join(sc2.Root, "sessions"))
	err = DeleteScratch(sc2.Root, FixedName(sc2.InstanceName), func(string) error {
		t.Error("worktree removal ran through a linked sessions/")
		return nil
	})
	wantNotScratch(t, err, "sessions/ is a link")
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("victim touched: %v", err)
	}
}

func TestListScratchReportsLinks(t *testing.T) {
	tmp := privateTemp(t)
	v, keep := victim(t)
	link(t, v, filepath.Join(tmp, "scratch-dddddddddddd"))
	got, err := ListScratch()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Link || got[0].Name != "scratch-dddddddddddd" {
		t.Fatalf("listed %+v, want the link marked", got)
	}
	wantNotScratch(t, CheckScratchPlace(got[0].Root), "a link")
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("victim touched: %v", err)
	}
}
