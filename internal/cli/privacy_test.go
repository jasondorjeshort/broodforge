// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The P16 Proof (08-implementation-plan-session-workflow.md), on the built binary against the
// in-process harness: a commit by the seeded driver with the model's trailer passes; a commit
// adding a line with a placeholder home path and a commit whose message carries an address
// refuse worktree check and handoff with one privacy line each; the path's user segment in
// privacy.allow lifts that one; after an amend the check is ok and the handoff pushes. The
// address is pieced, so the repository's own scan of this file reads none.
func TestProofPushScan(t *testing.T) {
	p := newWorktreeProof(t)
	s := p.start(t, "s0001")
	tree := filepath.Join(s.Dir(), "tree")
	good := gittest.Commit(t, gittest.Driver, nil, tree, "README", "base\nmore\n", "Add a line")
	home := gittest.Commit(t, gittest.Driver, nil, tree, "notes.txt", "see /home/alice/notes\n", "Add notes")
	mail := gittest.Commit(t, gittest.Driver, nil, tree, "a.txt", "a\n", "Write to "+"bob"+"@"+"mail.net")
	root := []string{"--root", p.root}
	commits := good + "  ok\n" + home + "  ok\n" + mail + "  ok\n"
	homeLine := "privacy: " + home + "  home-path  notes.txt:1\n"
	mailLine := "privacy: " + mail + "  email  message:1\n"
	refused := func(n int) string {
		return "error: handoff-refused-privacy: " + string(rune('0'+n)) + " of 3 commits carry personal data\n"
	}

	r := clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...)
	t.Logf("$ broodforge worktree check s0001\n%s%s", r.Stdout, r.Stderr)
	want(t, r, 1, "ok  pre-push\n"+commits+homeLine+mailLine, refused(2))
	r = clitest.Run(t, nil, append(root, "worktree", "check", "s0001", "--json")...)
	if !strings.Contains(r.Stdout, `{"privacy":"`+home+`","class":"home-path","location":"notes.txt:1"}`+"\n") {
		t.Errorf("--json rows:\n%s", r.Stdout)
	}

	// handoff refuses the same, the hit lines streamed, the tree kept.
	want(t, clitest.Run(t, nil, append(root, "worktree", "handoff", "s0001")...), 1, homeLine+mailLine, refused(2))
	want(t, clitest.Run(t, nil, append(root, "worktree", "handoff", "s0001", "--expect", "handoff-refused-privacy")...), 0,
		homeLine+mailLine, "")
	if _, err := os.Stat(tree); err != nil {
		t.Errorf("a refused handoff removed the tree: %v", err)
	}

	// The owner lifts the placeholder path's user segment.
	c := *p.cfg.Load()
	c.Instance.Privacy.Allow = []string{"alice"}
	p.cfg.Store(&c)
	want(t, clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...), 1, "ok  pre-push\n"+commits+mailLine, refused(1))

	// The agent amends the message; the commit-msg hook appends the trailer again.
	gittest.Git(t, tree, "commit", "--quiet", "--amend", "-m", "Write to the owner")
	amended := strings.TrimSpace(gittest.Git(t, tree, "rev-parse", "HEAD"))
	want(t, clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...), 0,
		"ok  pre-push\n"+good+"  ok\n"+home+"  ok\n"+amended+"  ok\nok  3\n", "")
	branch := "broodforge/" + p.instance + "/s0001"
	want(t, clitest.Run(t, nil, append(root, "worktree", "handoff", "s0001", "--expect", "pushed")...), 0,
		"pushed  origin  refs/sessions/s0001:refs/heads/"+branch+"  3  -  -\n", "")

	var kinds []string
	if err := p.st.Query("SELECT kind || ':' || coalesce(phase, '') FROM audit_event ORDER BY seq", nil,
		func(r *store.Row) error { kinds = append(kinds, r.Text(0)); return nil }); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(kinds, " "); got != "handoff-refused: handoff-refused: push:attempted push:done" {
		t.Errorf("audit %q", got)
	}
}
