// SPDX-License-Identifier: GPL-3.0-or-later

package git_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
)

// The handoff's push scan with identities whose addresses are not placeholders (pieced, so the
// repository's own scan of this file reads none): the driver authors, a participant and the
// model co-author, and only a home path in an added line refuses; after the amend it pushes.
func TestHandoffRefusesPrivacyThenPushesTheAmend(t *testing.T) {
	f := newFixture(t)
	me := gittest.Who{Name: "Alice", Email: "alice" + "@" + "mail.net"}
	meID := git.Identity{Name: me.Name, Email: me.Email}
	bobID := git.Identity{Name: "Bob", Email: "bob" + "@" + "mail.net"}
	clean := gittest.Commit(t, me, nil, f.tree, "a.txt", "a\n", msg("one", bobID, opus))
	home := gittest.Commit(t, me, nil, f.tree, "notes.txt", "x\nsee /home/alice/notes\n", msg("two", bobID, opus))
	attr := git.Attribution{Driver: meID, Participants: []git.Contributor{{Identity: bobID, Since: t0}},
		Models: []git.Contributor{{Identity: opus, Since: t0}}}
	h := &git.Handoff{Backend: f.backend, Session: f.sess, Mode: git.ModePush, Attribution: attr,
		Record: func(e audit.Event) error { f.events = append(f.events, e); return nil }, Actor: "host"}

	_, err := h.Run(context.Background(), true)
	var ge *git.Error
	if !errors.As(err, &ge) || ge.Code != git.CodeRefusedPrivacy {
		t.Fatalf("dry run: %v", err)
	}
	if want := []git.PrivacyHit{{Commit: home, Class: git.ClassHomePath, Location: "notes.txt:2"}}; !reflect.DeepEqual(ge.Hits, want) || ge.Checked != 2 {
		t.Errorf("hits %+v of %d, want %+v of 2", ge.Hits, ge.Checked, want)
	}
	if want := "1 commits carry personal data. " + home[:7] + ": home-path at notes.txt:2"; ge.Message != want {
		t.Errorf("message %q, want %q", ge.Message, want)
	}
	if _, err := os.Stat(f.tree); err != nil || strings.Contains(gittest.Git(t, f.remote, "branch"), "s0001") {
		t.Errorf("a refused handoff removed the tree or pushed: %v", err)
	}
	var details struct {
		Code    string
		Privacy []struct{ Commit, Class, Location string }
	}
	if len(f.events) != 1 || json.Unmarshal(f.events[0].Details, &details) != nil || details.Code != git.CodeRefusedPrivacy ||
		len(details.Privacy) != 1 || details.Privacy[0].Location != "notes.txt:2" || strings.Contains(string(f.events[0].Details), "/home/") {
		t.Errorf("audit %s", f.events[0].Details)
	}

	// Without the participant among the identities, their address in the trailer is a hit.
	base := strings.TrimSpace(gittest.Git(t, f.sess.Canonical, "merge-base", git.SessionRef("s0001"), "main"))
	noBob := git.Privacy{}.WithSession(git.Attribution{Driver: meID, Models: attr.Models})
	hits, err := git.ScanPrivacy(context.Background(), f.sess, base, noBob)
	want := []git.PrivacyHit{{Commit: clean, Class: git.ClassEmail, Location: "message:3"},
		{Commit: home, Class: git.ClassEmail, Location: "message:3"}, {Commit: home, Class: git.ClassHomePath, Location: "notes.txt:2"}}
	if err != nil || !reflect.DeepEqual(hits, want) {
		t.Errorf("scan without the participant: %+v %v\nwant %+v", hits, err, want)
	}

	// A read that fails is not a hit: the fetch's code and git's message, no hits, no venue text.
	gone := f.sess
	gone.ID = "s0009"
	_, err = git.ScanPrivacy(context.Background(), gone, base, noBob)
	if !errors.As(err, &ge) || ge.Code != git.CodeRefusedFetch || len(ge.Hits) != 0 ||
		!strings.HasPrefix(ge.Message, "git log: ") || strings.Contains(ge.Message, "personal data") {
		t.Errorf("scan of a missing session ref: %v", err)
	}

	// The agent amends; the scan passes and the handoff pushes.
	if err := os.WriteFile(filepath.Join(f.tree, "notes.txt"), []byte("x\nsee ~/notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gittest.GitAs(t, me, nil, f.tree, "commit", "--quiet", "--amend", "--no-edit", "--", "notes.txt")
	out, err := h.Run(context.Background(), false)
	if err != nil || out.Outcome != git.OutcomePushed || out.Commits != 2 {
		t.Fatalf("after the amend: %+v %v", out, err)
	}
}

// WithCopy adds the privacy section of a workflow copy's .claude/hooks/policy.json: never and
// allow literals and identities, the base's objects or Name <email> strings; no copy or no file
// leaves the lists as they were; a file that does not parse is the fetch's refusal.
func TestPrivacyWithCopy(t *testing.T) {
	base := git.Privacy{Allow: []string{"user"}, Identities: []git.Identity{{Name: "A", Email: "a@example.com"}}}
	got, err := base.WithCopy(filepath.Join(t.TempDir(), "absent"))
	if err != nil || !reflect.DeepEqual(got, base) {
		t.Errorf("no copy: %+v %v", got, err)
	}
	dir := t.TempDir()
	if got, err := base.WithCopy(dir); err != nil || !reflect.DeepEqual(got, base) {
		t.Errorf("no policy file: %+v %v", got, err)
	}
	policy := filepath.Join(dir, ".claude", "hooks", "policy.json")
	if err := os.MkdirAll(filepath.Dir(policy), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(s string) {
		if err := os.WriteFile(policy, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"edit":{},"privacy":{"never":["secretword"],"allow":["bob"],"identities":[{"name":"B","email":"b@example.com"},"C <c@example.com>"]}}`)
	got, err = base.WithCopy(dir)
	want := git.Privacy{Never: []string{"secretword"}, Allow: []string{"user", "bob"}, Identities: []git.Identity{
		{Name: "A", Email: "a@example.com"}, {Name: "B", Email: "b@example.com"}, {Name: "C", Email: "c@example.com"}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("section: got %+v %v", got, err)
	}
	if len(base.Allow) != 1 || len(base.Identities) != 1 {
		t.Errorf("WithCopy changed its receiver: %+v", base)
	}
	for _, bad := range []string{`{`, `{"privacy":{"identities":["nobody"]}}`, `{"privacy":{"identities":[{"name":"N"}]}}`} {
		write(bad)
		var ge *git.Error
		if _, err := base.WithCopy(dir); !errors.As(err, &ge) || ge.Code != git.CodeRefusedFetch {
			t.Errorf("%s: %v", bad, err)
		}
	}
}
