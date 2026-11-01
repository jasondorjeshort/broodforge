// SPDX-License-Identifier: GPL-3.0-or-later

package git_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// TestDivergedNoticeFixRuns: the command the canonical-diverged notice prints (discord's
// DivergedNotice) works on a canonical as setup leaves it, with its default branch checked out,
// where `branch -f` is refused: run as printed, main is the remote's afterwards.
func TestDivergedNoticeFixRuns(t *testing.T) {
	f := newFetchFixture(t)
	theirs := f.pushed(t, "theirs.txt")
	gittest.Commit(t, gittest.Driver, nil, f.canon, "ours.txt", "ours\n", "ours")
	_, err := git.Fetch(context.Background(), f.sess, "", nil)
	var ge *git.Error
	if !errors.As(err, &ge) || ge.Code != git.CodeCanonicalDiverged {
		t.Fatalf("Fetch: %v, want %s", err, git.CodeCanonicalDiverged)
	}

	notice := discord.DivergedNotice("p", "main", f.canon, "origin")
	_, fix, ok := strings.Cut(notice, "Fix: ")
	if !ok {
		t.Fatalf("no command in %q", notice)
	}
	args := shellWords(t, fix)
	if len(args) == 0 || args[0] != "git" {
		t.Fatalf("not a git command: %q", fix)
	}
	cmd := exec.Command("git", args[1:]...)
	cmd.Env = gittest.Env(gittest.Driver)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", fix, err, out)
	}
	if got := f.rev(t, "main"); got != theirs {
		t.Errorf("main is %s after the fix, want the remote's %s", got, theirs)
	}
	if !exists(filepath.Join(f.canon, "theirs.txt")) {
		t.Error("the checkout did not follow the reset")
	}
}

// shellWords splits a pasted command as bash would here: words by spaces, a double-quoted word
// kept whole, and a leading `~/` expanded to the home directory.
func shellWords(t *testing.T, s string) []string {
	t.Helper()
	var out []string
	for s = strings.TrimSpace(s); s != ""; s = strings.TrimSpace(s) {
		var w string
		if strings.HasPrefix(s, `"`) {
			end := strings.Index(s[1:], `"`)
			if end < 0 {
				t.Fatalf("unclosed quote in %q", s)
			}
			w, s = s[1:1+end], s[2+end:]
		} else if i := strings.IndexByte(s, ' '); i >= 0 {
			w, s = s[:i], s[i:]
		} else {
			w, s = s, ""
		}
		if rest, ok := strings.CutPrefix(w, "~/"); ok {
			home, err := os.UserHomeDir()
			if err != nil {
				t.Fatal(err)
			}
			w = filepath.ToSlash(home) + "/" + rest
		}
		out = append(out, w)
	}
	return out
}
