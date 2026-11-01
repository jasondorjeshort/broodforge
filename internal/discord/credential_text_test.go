// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/session"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The texts of the first live run as the owner accepted them on 2026-10-31
// (docs/design/04-ux.md, Credentials and Failure flows).

func TestOwnersTexts20261031(t *testing.T) {
	for got, want := range map[string]string{
		subscriptionReadyText("cred-7k2m", true, "C:/bf/bin/broodforge.exe"):                                                "cred-7k2m is added. Sign it in on the host: C:/bf/bin/broodforge.exe credential login cred-7k2m",
		subscriptionReadyText("cred-7k2m", false, "C:/bf/bin/broodforge.exe"):                                               "cred-7k2m is added. The owner signs it in on the host and sends you a link; reply with the code.",
		withDefaultText("cred-7k2m is added. The owner signs it in on the host and sends you a link; reply with the code."): "cred-7k2m is added. The owner signs it in on the host and sends you a link; reply with the code. It is your default.",
		withDefaultText("cred-7k2m is added. Sign it in on the host: bf credential login cred-7k2m"):                        "cred-7k2m is added. Sign it in on the host: bf credential login cred-7k2m\nIt is your default.",
		noDefaultText(nil, true):                                                    "Could not start a session: you have no default credential; add one with /credential add.",
		noDefaultText(nil, false):                                                   "Could not start a session: you have no default credential; set one with /credential default id:<id>.",
		noDefaultText([]string{"cred-7k2m"}, true):                                  "Could not start a session: you have no default credential; set one with /credential default id:cred-7k2m.",
		noDefaultText([]string{"cred-7k2m", "cred-9q1x"}, true):                     "Could not start a session: you have no default credential; set one with /credential default id:cred-7k2m (or cred-9q1x).",
		noDefaultText([]string{"cred-a", "cred-b", "cred-c"}, true):                 "Could not start a session: you have no default credential; set one with /credential default id:cred-a (or cred-b or cred-c).",
		canonicalDivergedText("main"):                                               "Could not start a session: the remote's main history was rewritten, so the host's copy no longer matches. The owner needs to reset it.",
		DivergedNotice("broodforge", "main", "C:/r/projects/broodforge", "origin"):  "broodforge: host copy of main diverged from the remote. Fix: git -C C:/r/projects/broodforge reset --keep origin/main",
		DivergedNotice("broodforge", "main", "C:/r/my projects/broodforge", "fork"): `broodforge: host copy of main diverged from the remote. Fix: git -C "C:/r/my projects/broodforge" reset --keep fork/main`,
	} {
		if got != want {
			t.Errorf("got  %q\nwant %q", got, want)
		}
	}
}

// TestTildePath: a path under the home directory is written from `~`, in either slash form and
// either case of the drive; any other path is unchanged.
func TestTildePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	slash := filepath.ToSlash(home)
	for in, want := range map[string]string{
		filepath.Join(home, "src", "repo"): "~/src/repo",
		slash + "/x":                       "~/x",
		strings.ToUpper(slash) + "/y":      "~/y",
		slash + "x/z":                      slash + "x/z", // a sibling sharing the prefix
		home:                               home,
		"/elsewhere/projects/p":            "/elsewhere/projects/p",
	} {
		if got := tildePath(in); got != want {
			t.Errorf("tildePath(%q) = %q, want %q", in, got, want)
		}
	}
	if got := DivergedNotice("p", "main", slash+"/r/projects/p", "origin"); !strings.Contains(got, "git -C ~/r/projects/p reset") {
		t.Errorf("the notice names the home directory: %s", got)
	}
}

// failingList is a credential store whose List fails.
type failingList struct{ *storeCredentials }

func (failingList) List(string) ([]CredentialInfo, error) {
	return nil, errors.New("the store is locked")
}

// TestNoDefaultListFails: a member whose credential list cannot be read is never told to add
// one; the refusal names the /credential default form, and the failure is logged.
func TestNoDefaultListFails(t *testing.T) {
	h := newTextHarness(t)
	var log strings.Builder
	h.a.cfg.Credentials, h.a.cfg.Log = failingList{h.creds}, &log
	_, err := h.command("1001", "/session start")
	if got := msgOf(err); strings.Contains(got, "add one with /credential add") ||
		got != "Could not start a session: you have no default credential; set one with /credential default id:<id>." {
		t.Errorf("refusal %q", got)
	}
	if !strings.Contains(log.String(), "warning: discord: the credentials of 1001 for the no-default refusal: the store is locked") {
		t.Errorf("log %q", log.String())
	}
}

// storeCredentials is /credential's store over the real store's credential table, so the
// session service's SetDefault and Start see what /credential add wrote.
type storeCredentials struct {
	mu sync.Mutex
	st *store.Store
	n  int
}

func (c *storeCredentials) Add(user, kind, label string, _ []byte) (CredentialInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	id := "cred-" + strconv.Itoa(c.n)
	status := "usable"
	if kind == kindSubscription {
		status = "needs-login"
	}
	err := c.st.Write(func(tx *store.Tx) error {
		_, err := tx.Exec(`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref) VALUES (?, ?, 'anthropic', ?, ?, ?, '2026-10-31T00:00:00Z', ?)`,
			store.Text(id), store.Text(kind), store.Text(user), store.Text(label), store.Text(status), store.Text(id))
		return err
	})
	return CredentialInfo{ID: id, Kind: kind, Label: label, Status: status}, err
}

func (c *storeCredentials) List(user string) ([]CredentialInfo, error) {
	var out []CredentialInfo
	err := c.st.Query("SELECT id, kind, label, status FROM credential WHERE user_id = ? ORDER BY added_at, id", []store.Value{store.Text(user)},
		func(r *store.Row) error {
			out = append(out, CredentialInfo{ID: r.Text(0), Kind: r.Text(1), Label: r.Text(2), Status: r.Text(3)})
			return nil
		})
	return out, err
}

func (c *storeCredentials) Remove(string, string) (CredentialInfo, error) {
	return CredentialInfo{}, &session.Error{Code: CodeNotBuilt, Message: "not in this test"}
}

// textHarness is a dry-run adapter over a real store and session service, the notice channel
// 300, the fetch at session start refusing canonical-diverged when diverged is set.
type textHarness struct {
	t        *testing.T
	st       *store.Store
	svc      *session.Service
	a        *Adapter
	creds    *storeCredentials
	clock    *session.ManualClock
	diverged bool
	root     string
}

func newTextHarness(t *testing.T) *textHarness {
	t.Helper()
	root := filepath.ToSlash(t.TempDir())
	st, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	h := &textHarness{t: t, st: st, root: root, creds: &storeCredentials{st: st},
		clock: session.NewManualClock(time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC))}
	fixture := delayedBasic(t, 0)
	cfg := &config.Config{
		Instance: config.Instance{InstanceName: "test", RoleGroups: map[string][]string{"trusted": {"10"}},
			Discord:  config.Discord{Mode: config.DiscordDryRun, NoticeChannelID: "300"},
			Defaults: config.Defaults{Mode: config.ModePolicy, Harness: config.HarnessFake, IdleTimeoutMinutes: 60, PromptTimeoutMinutes: 5}},
	}
	roles := session.Roles{Start: []string{"trusted"}, Contribute: []string{"trusted"}, Answer: []string{"driver"}}
	channel := func(id string) (session.Channel, bool) {
		if id != testLobby {
			return session.Channel{}, false
		}
		return session.Channel{ID: id, Project: "p", ProjectName: "Project P", Reach: "code", Mode: "policy", Harness: "fake",
			Roles: roles, Repo: session.Repo{Canonical: root + "/projects/p", Remote: "origin", DefaultBranch: "main"}}, true
	}
	groups := func() map[string][]string { return cfg.Instance.RoleGroups }
	h.a = New(Config{Config: func() *config.Config { return cfg }, Channel: channel, Groups: groups, Clock: h.clock,
		DryRun: true, Credentials: h.creds, HostCommand: testHost})
	h.svc = session.New(session.Config{Root: root, Instance: "test", DryRun: true, Groups: groups, Channel: channel, Started: h.a.SessionStarted,
		Fixtures: filepath.Dir(fixture),
		Engine:   func(_, _, received string) (engine.Engine, error) { return engine.NewFake(fixture, received) },
		FetchCanonical: func(context.Context, git.Session, string) (git.FetchResult, error) {
			if h.diverged {
				return git.FetchResult{}, &git.Error{Code: git.CodeCanonicalDiverged, Message: root + "/projects/p main is at a, origin at b; neither contains the other"}
			}
			return git.FetchResult{}, nil
		},
		CreateTree: func(_ context.Context, s git.Session) (string, error) {
			return "base", os.MkdirAll(filepath.FromSlash(s.Dir()+"/tree"), 0o750)
		},
		Store: st, Audit: audit.NewWriter(nil), Clock: h.clock})
	h.a.Attach(h.svc)
	h.a.Run()
	t.Cleanup(h.a.Close)
	for _, user := range []string{"1001", "1002"} {
		if _, err := h.a.Command(context.Background(), Inbound{Channel: testLobby, User: user,
			Text: "/register name:User " + user + " email:u" + user + "@example.com"}); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

// testHost is the harness's host command; command injects as a dry run with --owner 1001 does.
const testHost = `"C:/Program Files/bf/broodforge.exe" --root C:/r`

func (h *textHarness) command(user, text string) (string, error) {
	h.t.Helper()
	r, err := h.a.Command(context.Background(), Inbound{Channel: testLobby, User: user, Roles: []string{"10"}, Text: text, Owner: "1001"})
	if err != nil {
		return "", err
	}
	return r.Text, nil
}

func (h *textHarness) defaultOf(user string) string {
	h.t.Helper()
	r, ok, err := h.svc.Registered(user)
	if err != nil || !ok {
		h.t.Fatalf("registration %s: %v %v", user, ok, err)
	}
	if r.DefaultCredential == nil {
		return ""
	}
	return *r.DefaultCredential
}

// TestCredentialAddFirstDefault: a member's first credential becomes their default and the
// reply says so; a second leaves the default alone. The owner gets the host command with the
// binary's own path, a member the owner's part; an API key's reply takes the same sentence.
func TestCredentialAddFirstDefault(t *testing.T) {
	h := newTextHarness(t)
	got, err := h.command("1001", "/credential add kind:subscription label:mine")
	if want := "cred-1 is added. Sign it in on the host: " + testHost + " credential login cred-1\nIt is your default."; err != nil || got != want {
		t.Errorf("owner's first: %q %v\nwant %q", got, err, want)
	}
	got, err = h.command("1001", "/credential add kind:subscription label:second")
	if want := "cred-2 is added. Sign it in on the host: " + testHost + " credential login cred-2"; err != nil || got != want {
		t.Errorf("owner's second: %q %v\nwant %q", got, err, want)
	}
	if d := h.defaultOf("1001"); d != "cred-1" {
		t.Errorf("owner's default %q, want cred-1", d)
	}
	got, err = h.command("1002", "/credential add kind:subscription label:mine")
	if want := "cred-3 is added. The owner signs it in on the host and sends you a link; reply with the code. It is your default."; err != nil || got != want {
		t.Errorf("member's first: %q %v\nwant %q", got, err, want)
	}
	if d := h.defaultOf("1002"); d != "cred-3" {
		t.Errorf("member's default %q, want cred-3", d)
	}

	// The API-key form, for a third member: the modal's submission sets the default too.
	if _, err := h.a.Command(context.Background(), Inbound{Channel: testLobby, User: "1003", Text: "/register name:C email:c@example.com"}); err != nil {
		t.Fatal(err)
	}
	r, err := h.a.apiKeySubmitted(Inbound{Channel: testLobby, User: "1003", CustomID: APIKeyModalPrefix + "k"}, "sk-ant-test-0000a7f2")
	if want := "Stored cred-4 (API key, ends in ...a7f2). It is your default."; err != nil || r.Text != want {
		t.Errorf("API key: %q %v\nwant %q", r.Text, err, want)
	}
}

// TestStartRefusalsCredential: a default that needs a login gets 04's needs-a-login line; a
// member with a credential and no default is told the /credential default command with its id.
func TestStartRefusalsCredential(t *testing.T) {
	h := newTextHarness(t)
	if _, err := h.command("1001", "/credential add kind:subscription label:mine"); err != nil {
		t.Fatal(err)
	}
	_, err := h.command("1001", "/session start")
	if want := "Could not start a session: your default credential cred-1 needs a login."; msgOf(err) != want {
		t.Errorf("needs a login: %v\nwant %q", err, want)
	}
	// A credential added outside /credential add (the host's credential add) is no default.
	if _, err := h.creds.Add("1002", kindAPIKey, "k", nil); err != nil {
		t.Fatal(err)
	}
	_, err = h.command("1002", "/session start")
	if want := "Could not start a session: you have no default credential; set one with /credential default id:cred-2."; msgOf(err) != want {
		t.Errorf("no default: %v\nwant %q", err, want)
	}
	// Several: each named.
	if _, err := h.creds.Add("1002", kindAPIKey, "k2", nil); err != nil {
		t.Fatal(err)
	}
	_, err = h.command("1002", "/session start")
	if want := "Could not start a session: you have no default credential; set one with /credential default id:cred-2 (or cred-3)."; msgOf(err) != want {
		t.Errorf("no default, several: %v\nwant %q", err, want)
	}
}

// TestCanonicalDivergedNotice: the start refusal gets the plain line, and the notice channel
// the line with the command that resets the host's copy.
func TestCanonicalDivergedNotice(t *testing.T) {
	h := newTextHarness(t)
	if _, err := h.creds.Add("1001", kindAPIKey, "k", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SetDefault("1001", "cred-1", "1001"); err != nil {
		t.Fatal(err)
	}
	h.diverged = true
	_, err := h.command("1001", "/session start")
	if want := canonicalDivergedText("main"); msgOf(err) != want {
		t.Errorf("refusal: %v\nwant %q", err, want)
	}
	want := "p: host copy of main diverged from the remote. Fix: git -C " + quoteSpace(tildePath(h.root+"/projects/p")) + " reset --keep origin/main"
	deadline := time.Now().Add(10 * time.Second)
	for {
		posts := h.a.Outbox.Channel("300")
		if len(posts) == 1 && posts[0].Body == want {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("notice channel: %+v\nwant one post %q", posts, want)
		}
		h.clock.Advance(time.Second)
		time.Sleep(5 * time.Millisecond)
	}
	if strings.Contains(msgOf(err), "git -C") {
		t.Errorf("the venue's line carries the command: %v", err)
	}
}

// msgOf is a refusal's text as the member reads it, without its code; "" for no refusal.
func msgOf(err error) string {
	var se *session.Error
	if !errors.As(err, &se) {
		return ""
	}
	return se.Message
}

// TestLiveOwner: on a live instance the owner is the bot application's owner, as the gateway
// read it at its start; a dry run's --owner stand-in does not count there.
func TestLiveOwner(t *testing.T) {
	h := newLiveHarness(t)
	if !h.a.isOwner(Inbound{User: "1001"}) || h.a.isOwner(Inbound{User: "1002", Owner: "1002"}) || h.a.isOwner(Inbound{}) {
		t.Errorf("owner %q", h.g.ownerID())
	}
}
