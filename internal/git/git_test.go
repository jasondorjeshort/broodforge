// SPDX-License-Identifier: GPL-3.0-or-later

package git_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

var (
	driver = git.Identity{Name: gittest.Driver.Name, Email: gittest.Driver.Email}
	opus   = git.Identity{Name: "Claude Opus 5.5", Email: git.ModelEmail}
	haiku  = git.Identity{Name: "Claude Haiku 5.5", Email: git.ModelEmail}
	alice  = git.Identity{Name: "Alice", Email: "alice@example.com"}
	t0     = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
)

func TestSessionID(t *testing.T) {
	for row, want := range map[int64]string{1: "s0001", 35: "s000z", 36: "s0010", 1679616: "s10000"} {
		if got := git.SessionID(row); got != want || !git.ValidSessionID(got) {
			t.Errorf("SessionID(%d) = %q, want %q", row, got, want)
		}
	}
	for _, bad := range []string{"", "s01", "S0001", "s000_", "x0001", "s0001/../x"} {
		if git.ValidSessionID(bad) {
			t.Errorf("%q valid", bad)
		}
	}
	if b := git.Branch("pc", "s0001"); b != "broodforge/pc/s0001" {
		t.Errorf("Branch %q", b)
	}
}

func TestRequired(t *testing.T) {
	a := git.Attribution{Driver: driver,
		Participants: []git.Contributor{{Identity: alice, Since: t0.Add(time.Hour)}, {Identity: driver, Since: t0}},
		Models:       []git.Contributor{{Identity: opus, Since: t0}, {Identity: haiku, Since: t0.Add(2 * time.Hour)}, {Identity: opus, Since: t0}}}
	cases := []struct {
		at   time.Time
		want []string
	}{
		{t0.Add(-time.Second), nil},
		{t0, []string{opus.Trailer()}},
		{t0.Add(time.Hour), []string{alice.Trailer(), opus.Trailer()}},
		{t0.Add(3 * time.Hour), []string{alice.Trailer(), opus.Trailer(), haiku.Trailer()}},
	}
	for _, c := range cases {
		if got := a.Required(c.at); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Required(%v) = %q, want %q", c.at, got, c.want)
		}
	}
	if opus.Trailer() != "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" {
		t.Errorf("trailer %q", opus.Trailer())
	}
}

func TestGitHubRepo(t *testing.T) {
	for url, want := range map[string]string{
		"https://github.com/o/r.git":     "o/r",
		"https://github.com/o/r":         "o/r",
		"https://x@github.com/o/r.git\n": "o/r",
		"git@github.com:o/r.git":         "o/r",
		"ssh://git@github.com/o/r.git":   "o/r",
		"https://gitlab.com/o/r.git":     "",
		"C:/tmp/remote/p.git":            "",
		"https://github.com/o/r/x":       "",
	} {
		owner, repo, ok := git.GitHubRepo(url)
		if got := owner + "/" + repo; ok != (want != "") || (ok && got != want) {
			t.Errorf("GitHubRepo(%q) = %q %v, want %q", url, got, ok, want)
		}
	}
}

func TestAskpassAnswer(t *testing.T) {
	env := map[string]string{git.CredentialEnv: "tok"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	if a, err := git.AskpassAnswer("Username for 'https://github.com': ", lookup); a != git.AskpassUser || err != nil {
		t.Errorf("username: %q %v", a, err)
	}
	if a, err := git.AskpassAnswer("Password for 'https://x-access-token@github.com': ", lookup); a != "tok" || err != nil {
		t.Errorf("password: %q %v", a, err)
	}
	delete(env, git.CredentialEnv)
	var ge *git.Error
	if _, err := git.AskpassAnswer("Password: ", lookup); !errors.As(err, &ge) || ge.Code != git.CodeNoCredential {
		t.Errorf("no credential: %v", err)
	}
}

// fixture is a canonical with a bare remote, a same-account backend whose install has no
// hooks, and session s0001 created.
type fixture struct {
	root, remote string
	backend      *spawn.SameAccount
	sess         git.Session
	tree         string
	events       []audit.Event
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	canon, remote := gittest.Project(t, root, "p")
	install := filepath.Join(root, "install")
	b, err := spawn.NewSameAccount(spawn.Config{Backend: spawn.NameSameAccount, Install: install, Binary: filepath.Join(install, "bin", "broodforge")})
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{root: root, remote: remote, backend: b,
		sess: git.Session{Root: root, ID: "s0001", Instance: "test", Canonical: canon, Remote: "origin", DefaultBranch: "main"}}
	base, err := git.Create(context.Background(), b, f.sess)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(gittest.Git(t, canon, "rev-parse", git.SessionRef("s0001"))); got != base {
		t.Errorf("refs/sessions/s0001 %q, base %q", got, base)
	}
	f.tree = filepath.ToSlash(filepath.Join(f.sess.Dir(), "tree"))
	return f
}

func (f *fixture) handoff(mode string) *git.Handoff {
	return &git.Handoff{Backend: f.backend, Session: f.sess, Mode: mode,
		Attribution: git.Attribution{Driver: driver, Models: []git.Contributor{{Identity: opus, Since: t0}}},
		Record:      func(e audit.Event) error { f.events = append(f.events, e); return nil }, Actor: "host"}
}

func (f *fixture) kinds() []string {
	var out []string
	for _, e := range f.events {
		out = append(out, string(e.Kind)+":"+string(e.Phase))
	}
	return out
}

func msg(subject string, trailers ...git.Identity) string {
	m := subject + "\n"
	if len(trailers) > 0 {
		m += "\n"
	}
	for _, i := range trailers {
		m += i.Trailer() + "\n"
	}
	return m
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func TestCreateRefusesAReusedID(t *testing.T) {
	f := newFixture(t)
	var ge *git.Error
	if _, err := git.Create(context.Background(), f.backend, f.sess); !errors.As(err, &ge) || ge.Code != git.CodeCreate {
		t.Errorf("second create: %v", err)
	}
}

func TestHandoffPush(t *testing.T) {
	f := newFixture(t)
	head := gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	out, err := f.handoff(git.ModePush).Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Outcome != git.OutcomePushed || out.Commits != 1 || !out.Removed || out.Refspec != "refs/sessions/s0001:refs/heads/broodforge/test/s0001" {
		t.Errorf("outcome %+v", out)
	}
	if got := strings.TrimSpace(gittest.Git(t, f.remote, "rev-parse", "broodforge/test/s0001")); got != head {
		t.Errorf("remote branch %q, want %q", got, head)
	}
	if exists(f.tree) || gittest.Git(t, f.sess.Canonical, "for-each-ref", "refs/sessions/") != "" {
		t.Error("tree or refs/sessions/s0001 survive a done push")
	}
	if got, want := f.kinds(), []string{"push:attempted", "push:done"}; !reflect.DeepEqual(got, want) {
		t.Errorf("audit %q, want %q", got, want)
	}
}

func TestHandoffRefusesAttributionThenPushesTheAmend(t *testing.T) {
	f := newFixture(t)
	gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	bad := gittest.Commit(t, gittest.Driver, nil, f.tree, "b.txt", "b\n", msg("two"), "--author", "X <x@example.com>")
	h := f.handoff(git.ModePush)
	_, err := h.Run(context.Background(), false)
	var ge *git.Error
	if !errors.As(err, &ge) || ge.Code != git.CodeRefusedAttribution || len(ge.Commits) != 1 {
		t.Fatalf("handoff: %v", err)
	}
	want := "author:X <x@example.com>; missing:" + opus.Trailer()
	if c := ge.Commits[0]; c.Hash != bad || c.Status() != want {
		t.Errorf("failing commit %s %q, want %s %q", c.Hash, c.Status(), bad, want)
	}
	if !exists(f.tree) || strings.Contains(gittest.Git(t, f.remote, "branch"), "s0001") {
		t.Error("a refused handoff removed the tree or pushed")
	}
	if got := f.kinds(); !reflect.DeepEqual(got, []string{"handoff-refused:"}) {
		t.Errorf("audit %q", got)
	}
	var details struct{ Code string }
	if err := json.Unmarshal(f.events[0].Details, &details); err != nil || details.Code != git.CodeRefusedAttribution {
		t.Errorf("details %s", f.events[0].Details)
	}
	// The driver has the agent amend; the refetch is forced and the base is still found.
	gittest.Git(t, f.tree, "commit", "--quiet", "--amend", "--reset-author", "-m", msg("two", opus))
	out, err := h.Run(context.Background(), false)
	if err != nil || out.Outcome != git.OutcomePushed || out.Commits != 2 {
		t.Fatalf("after the amend: %+v %v", out, err)
	}
}

func TestCheckRequiresOnlyWhatCameBefore(t *testing.T) {
	f := newFixture(t)
	at := []string{"GIT_COMMITTER_DATE=2026-01-01T00:30:00Z"}
	gittest.Commit(t, gittest.Driver, at, f.tree, "a.txt", "a\n", msg("one", opus))
	a := git.Attribution{Driver: driver,
		Participants: []git.Contributor{{Identity: alice, Since: t0.Add(time.Hour)}},
		Models:       []git.Contributor{{Identity: opus, Since: t0}, {Identity: haiku, Since: t0.Add(time.Minute)}}}
	c, err := git.Collect(context.Background(), f.backend, f.sess, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Checks) != 1 || c.Checks[0].Status() != "missing:"+haiku.Trailer() {
		t.Errorf("checks %+v", c.Checks)
	}
}

func TestHandoffNoCommits(t *testing.T) {
	f := newFixture(t)
	out, err := f.handoff(git.ModePush).Run(context.Background(), true)
	if err != nil || out.Outcome != git.OutcomeDryRun || out.Commits != 0 || !exists(f.tree) {
		t.Fatalf("dry run: %+v %v", out, err)
	}
	out, err = f.handoff(git.ModePush).Run(context.Background(), false)
	if err != nil || out.Outcome != git.OutcomeNone || !out.Removed || exists(f.tree) {
		t.Fatalf("no commits: %+v %v", out, err)
	}
	if len(f.events) != 0 {
		t.Errorf("audit %q for no commits", f.kinds())
	}
}

func TestHandoffPushRejected(t *testing.T) {
	f := newFixture(t)
	gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	// Someone else's branch of the same name is already on the remote.
	gittest.Git(t, f.sess.Canonical, "push", "--quiet", "origin", "main:refs/heads/broodforge/test/s0001")
	other := gittest.Commit(t, gittest.Driver, nil, f.sess.Canonical, "c.txt", "c\n", "other")
	gittest.Git(t, f.sess.Canonical, "push", "--quiet", "origin", other+":refs/heads/broodforge/test/s0001")
	_, err := f.handoff(git.ModePush).Run(context.Background(), false)
	var ge *git.Error
	if !errors.As(err, &ge) || ge.Code != git.CodeFailedPush {
		t.Fatalf("handoff: %v", err)
	}
	if !exists(f.tree) {
		t.Error("a failed push removed the tree")
	}
	if got := f.kinds(); !reflect.DeepEqual(got, []string{"push:attempted", "push:failed"}) {
		t.Errorf("audit %q", got)
	}
}

// githubFixture points the canonical's origin at a GitHub URL for the request and at the local
// bare repository for the push.
func githubFixture(t *testing.T) *fixture {
	f := newFixture(t)
	gittest.Git(t, f.sess.Canonical, "remote", "set-url", "origin", "https://github.com/o/r.git")
	gittest.Git(t, f.sess.Canonical, "remote", "set-url", "--push", "origin", f.remote)
	gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	return f
}

func TestHandoffPullRequestDryRun(t *testing.T) {
	f := githubFixture(t)
	h := f.handoff(git.ModePullRequest)
	h.CredentialID, h.Title = "cred-git", "Session s0001: hello"
	h.Secret = func(string) ([]byte, error) { t.Error("a dry run read the credential"); return nil, nil }
	out, err := h.Run(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	want := &git.PullRequest{URL: "https://api.github.com/repos/o/r/pulls", Credential: "cred-git",
		Body: git.PullRequestBody{Title: "Session s0001: hello", Head: "broodforge/test/s0001", Base: "main"}}
	if out.Outcome != git.OutcomeDryRun || !reflect.DeepEqual(out.Request, want) {
		t.Errorf("dry run %+v request %+v", out, out.Request)
	}
	if strings.Contains(gittest.Git(t, f.remote, "branch"), "s0001") || len(f.events) != 0 {
		t.Error("a dry run pushed or recorded")
	}
}

func TestHandoffPullRequest(t *testing.T) {
	f := githubFixture(t)
	var got struct {
		path, auth string
		body       git.PullRequestBody
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path, got.auth = r.URL.Path, r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got.body)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"html_url":"https://github.com/o/r/pull/7"}`)
	}))
	defer srv.Close()
	h := f.handoff(git.ModePullRequest)
	h.CredentialID, h.Title, h.API = "cred-git", "T", srv.URL
	h.Forge = git.GitHub{Client: srv.Client()}
	h.Secret = func(id string) ([]byte, error) { return []byte("tok-" + id), nil }
	out, err := h.Run(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Outcome != git.OutcomePullRequest || out.URL == nil || *out.URL != "https://github.com/o/r/pull/7" || !out.Removed {
		t.Errorf("outcome %+v", out)
	}
	if got.path != "/repos/o/r/pulls" || got.auth != "Bearer tok-cred-git" || got.body.Head != "broodforge/test/s0001" || got.body.Base != "main" {
		t.Errorf("request %+v", got)
	}
	want := []string{"push:attempted", "push:done", "pull-request-opened:attempted", "pull-request-opened:done"}
	if k := f.kinds(); !reflect.DeepEqual(k, want) {
		t.Errorf("audit %q, want %q", k, want)
	}
}

func TestHandoffPullRequestHTTPError(t *testing.T) {
	f := githubFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"message":"Validation Failed"}`)
	}))
	defer srv.Close()
	h := f.handoff(git.ModePullRequest)
	h.API, h.Forge = srv.URL, git.GitHub{Client: srv.Client()}
	_, err := h.Run(context.Background(), false)
	var ge *git.Error
	if !errors.As(err, &ge) || ge.Code != git.CodeFailedPullRequest || ge.Message != "HTTP 422: Validation Failed" {
		t.Fatalf("handoff: %v", err)
	}
	if !exists(f.tree) {
		t.Error("a failed request removed the tree")
	}
}

// The default branch moves on after the session starts and the session merges it: the merged
// commits are the canonical's, not the session's, so only the session's own are checked.
func TestHandoffAfterMergingTheDefaultBranch(t *testing.T) {
	f := newFixture(t)
	gittest.Commit(t, gittest.Who{Name: alice.Name, Email: alice.Email}, nil, f.sess.Canonical, "alice.txt", "a\n", "alice's work")
	gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	gittest.Git(t, f.tree, "fetch", "--quiet", "origin")
	gittest.Git(t, f.tree, "merge", "--quiet", "--no-ff", "-m", msg("merge main", opus), "origin/main")
	c, err := git.Collect(context.Background(), f.backend, f.sess, f.handoff(git.ModePush).Attribution)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Checks) != 2 || len(c.Failing()) != 0 || c.Commits != 2 {
		t.Errorf("checks %+v", c.Checks)
	}
	out, err := f.handoff(git.ModePush).Run(context.Background(), false)
	if err != nil || out.Outcome != git.OutcomePushed || out.Commits != 2 {
		t.Fatalf("handoff: %+v %v", out, err)
	}
}

func TestHandoffTwiceIsAlreadyHandedOff(t *testing.T) {
	f := newFixture(t)
	gittest.Commit(t, gittest.Driver, nil, f.tree, "a.txt", "a\n", msg("one", opus))
	h := f.handoff(git.ModePush)
	if _, err := h.Run(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	f.events = nil
	_, err := h.Run(context.Background(), false)
	var ge *git.Error
	if !errors.As(err, &ge) || ge.Code != git.CodeAlreadyHandedOff {
		t.Fatalf("second handoff: %v", err)
	}
	if len(f.events) != 0 {
		t.Errorf("audit %q for an already handed-off session", f.kinds())
	}
}

func TestAskpassAnswersOnlyUsernameAndPassword(t *testing.T) {
	lookup := func(string) (string, bool) { return "tok", true }
	for _, p := range []string{"Are you sure you want to continue connecting (yes/no)? ", "Enter passphrase for key 'k': ", ""} {
		var ge *git.Error
		if a, err := git.AskpassAnswer(p, lookup); !errors.As(err, &ge) || ge.Code != git.CodeNoCredential || a != "" {
			t.Errorf("prompt %q: %q %v", p, a, err)
		}
	}
}
