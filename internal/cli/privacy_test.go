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
	"github.com/jasondorjeshort/broodforge/internal/workflow"
	"github.com/jasondorjeshort/broodforge/internal/workflow/workflowtest"
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

// The P17 clause of P16's Proof: a commit whose Co-Authored-By names the identity of the
// fixture's .claude/hooks/policy.json is refused with no workflow copy and passes once a copy
// rendered from the fixture is at workflow/<id>. The fixture's placeholder address passes the
// scan anywhere, so the test gives the identity an address the scan reads (pieced, so the
// repository's own scan of this file reads none) before committing the fixture.
func TestProofPushScanWorkflowCopy(t *testing.T) {
	p := newWorktreeProof(t)
	s := p.start(t, "s0001")
	tree := filepath.Join(s.Dir(), "tree")
	addr := "alice" + "@" + "mail.net"
	c := gittest.Commit(t, gittest.Driver, nil, tree, "README", "base\nmore\n",
		"Add a line\n\nCo-Authored-By: Alice Example <"+addr+">")
	root := []string{"--root", p.root}

	r := clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...)
	t.Logf("$ broodforge worktree check s0001  (no copy)\n%s%s", r.Stdout, r.Stderr)
	want(t, r, 1, "ok  pre-push\n"+c+"  ok\nprivacy: "+c+"  email  message:3\n",
		"error: handoff-refused-privacy: 1 of 1 commits carry personal data\n")

	canonical := workflowtest.Canonical(t, func(dir string) {
		policy := filepath.Join(dir, ".claude", "hooks", "policy.json")
		b, err := os.ReadFile(policy)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(policy, []byte(strings.Replace(string(b), "alice@example.com", addr, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := workflow.Render(t.Context(), workflow.Options{Canonical: canonical, Branch: "main"}, workflow.Dir(p.root, "s0001")); err != nil {
		t.Fatal(err)
	}
	r = clitest.Run(t, nil, append(root, "worktree", "check", "s0001")...)
	t.Logf("$ broodforge worktree check s0001  (copy at workflow/s0001)\n%s%s", r.Stdout, r.Stderr)
	want(t, r, 0, "ok  pre-push\n"+c+"  ok\nok  1\n", "")

	// A copy whose policy file does not parse refuses the scan (check, handoff) as a failed read,
	// and never fails worktree bundle, which does not scan.
	policy := filepath.Join(workflow.Dir(p.root, "s0001"), ".claude", "hooks", "policy.json")
	if err := os.WriteFile(policy, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := clitest.Run(t, nil, append(root, "worktree", "bundle", "s0001")...); r.Code != 0 {
		t.Errorf("bundle with a malformed copy policy: exit %d %s%s", r.Code, r.Stdout, r.Stderr)
	}
	for _, cmd := range []string{"check", "handoff"} {
		r = clitest.Run(t, nil, append(root, "worktree", cmd, "s0001")...)
		if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: handoff-refused-fetch: the workflow copy's .claude/hooks/policy.json: ") {
			t.Errorf("%s with a malformed copy policy: exit %d stderr %q", cmd, r.Code, r.Stderr)
		}
	}
}
