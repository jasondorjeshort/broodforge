// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Error is a refusal with a stable code; the control server sends it as its wire error. This
// package does not import control (03's session row), so the bot's handlers map it.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Codes this package refuses with.
const (
	CodeBadArgs          = "bad-args"
	CodeUnknownChannel   = "unknown-channel"
	CodeUnknownSession   = "unknown-session"
	CodeUnregistered     = "unregistered"
	CodeRefused          = "refused"
	CodeLiveInstance     = "live-instance"
	CodeNoCredential     = "no-credential"
	CodeCredentialStatus = "credential-not-usable"
	CodeUnknownFixture   = "unknown-fixture"
	CodeBadFixture       = "bad-fixture"
	CodeSessionClosed    = "session-closed"
	CodeShuttingDown     = "shutting-down"
	CodeBadRegistration  = "bad-registration"
	CodeShutdownTimeout  = "stop-timeout"
)

// Session states (02-domain.md, Session), stored kebab-case.
const (
	StateStarting         = "starting"
	StateWaitingForInput  = "waiting-for-input"
	StateRunningTurn      = "running-turn"
	StateWaitingForAnswer = "waiting-for-answer"
	StateEnding           = "ending"
	StateEnded            = store.SessionEnded
	StateInterrupted      = store.SessionInterrupted
)

// Participant roles (R-13).
const (
	RoleDriver      = "driver"
	RoleContributor = "contributor"
)

// Turn results (02-domain.md, Turn).
const (
	ResultCompleted   = "completed"
	ResultError       = "error"
	ResultInterrupted = "interrupted"
	ResultDeferred    = "deferred"
)

// Audit actors besides a Discord user id (02-domain.md, Audit event).
const (
	ActorHost     = "host"
	ActorPolicy   = "policy"
	ActorHook     = "hook"
	ActorInstance = "instance"
)

// Hook variables of 03-hook-contract.md (Environment). The hook package holds the same names;
// session does not import it (03's session row).
const (
	envSession   = "BROODFORGE_SESSION"
	envMode      = "BROODFORGE_MODE"
	envReach     = "BROODFORGE_REACH"
	envTree      = "BROODFORGE_TREE"
	envCanonical = "BROODFORGE_CANONICAL"
	envHookLog   = "BROODFORGE_HOOK_LOG"
)

// worktreeActive is a worktree row's state while its session uses it (02-domain.md, Worktree).
const worktreeActive = "active"

// defaultFixture is the fake's fixture when a start names none (05, A turn against the fake).
const defaultFixture = "basic"

// Roles are a channel's role lists (03 Configuration).
type Roles struct {
	Start, Contribute, Answer []string
}

// Repo is the project's canonical checkout.
type Repo struct {
	Canonical, Remote, DefaultBranch string
}

// Channel is one configured channel as the bot sees it, defaults merged in: the caller
// translates its configuration into this, so session does not import config.
type Channel struct {
	ID, Project, ProjectName   string
	Reach, Mode, Harness       string
	Roles                      Roles
	IdleTimeout, PromptTimeout time.Duration
	Model, Effort              string
	Repo                       Repo
}

// EngineFactory returns the engine a new session runs on: harness "fake" replays fixture
// (writing received-<n>.json into receivedDir), "claude-code" is the real engine.
type EngineFactory func(harness, fixture, receivedDir string) (engine.Engine, error)

// Config is everything the service needs from the bot.
type Config struct {
	Root     string // absolute
	Instance string // instanceName
	// DryRun is discord.mode dry-run: the debug overrides (--reach, --fixture, timeouts) are
	// refused live-instance otherwise (05-cli-conventions.md, Isolation).
	DryRun     bool
	RoleGroups map[string][]string
	// Channel looks a configured channel up by id.
	Channel func(id string) (Channel, bool)
	// Fixtures is engine.fakeFixtures: a bare --fixture name is a directory under it.
	Fixtures string
	Engine   EngineFactory
	// CreateTree is clone at session start: git.Create through the spawn backend.
	CreateTree func(ctx context.Context, s git.Session) (base string, err error)
	// RemoveTree undoes CreateTree for a start that fails after the clone: git.Discard.
	RemoveTree func(ctx context.Context, s git.Session) error
	Store      *store.Store
	Audit      *audit.Writer
	// Now is the injectable clock; nil is time.Now. Idle and prompt timeouts read it (P10 half b).
	Now func() time.Time
}

// Service is SessionService (03-architecture.md, Components; N-11): start, submit, status and
// subscribe here; answer, setMode, interrupt, end and invite are P10 half (b) and later. Every
// session's turn runs on its own goroutine reading its own engine's events; nothing here waits
// on another session, and store writes are short transactions.
type Service struct {
	cfg     Config
	closing atomic.Bool

	mu   sync.Mutex
	live map[string]*live
}

// New returns the service.
func New(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{cfg: cfg, live: map[string]*live{}}
}

func (s *Service) now() time.Time { return s.cfg.Now().UTC() }
func (s *Service) stamp() string  { return store.FormatTime(s.now()) }
func badArgs(msg string) *Error   { return &Error{CodeBadArgs, msg} }

// Details renders an audit event's details from key and value pairs, an empty value left out.
// Values are strings, so no map holds an `any`; HTML characters are not escaped, so a
// `Name <email>` reads as written.
func Details(kv ...string) json.RawMessage {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" {
			m[kv[i]] = kv[i+1]
		}
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return json.RawMessage("{}")
	}
	return bytes.TrimSpace(b.Bytes())
}

func details(kv ...string) json.RawMessage { return Details(kv...) }

// actorOf is the audit actor of a command acting for user, or the host when none is given.
func actorOf(user string) string {
	if user == "" {
		return ActorHost
	}
	return user
}

// refuse audits an action refused (N-17; 04 Conventions: every refusal is one) and returns err.
func (s *Service) refuse(actor, sessionID, action, reason string, err *Error) error {
	werr := s.cfg.Store.Write(func(tx *store.Tx) error {
		_, e := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindActionRefused, Actor: actor, Session: sessionID,
			Details: details("action", action, "reason", reason, "message", err.Message)})
		return e
	})
	return errors.Join(err, werr)
}

// registration is one registration row.
type registration struct {
	User, Name, Email, DefaultCredential string
}

func (s *Service) registration(user string) (registration, bool, error) {
	r := registration{User: user}
	found := false
	err := s.cfg.Store.Query("SELECT git_name, git_email, coalesce(default_credential_id, '') FROM registration WHERE user_id = ?",
		[]store.Value{store.Text(user)}, func(row *store.Row) error {
			r.Name, r.Email, r.DefaultCredential, found = row.Text(0), row.Text(1), row.Text(2), true
			return nil
		})
	return r, found, err
}

// splitRoles reads --roles: comma-separated role ids; "" is none given.
func splitRoles(s string) []string { return policy.SplitList(s) }

// StartRequest is `session start` (05-debug-interface-session.md).
type StartRequest struct {
	Channel, As, Roles, Credential, Mode, Reach, Fixture string
	IdleTimeoutSeconds, PromptTimeoutSeconds             int
}

// Started is the new session.
type Started struct {
	ID string `json:"id"`
}

// live is a session the bot holds open: its in-memory state beside the store's rows.
type live struct {
	id         string
	ch         Channel
	mode       string
	reach      string
	driver     registration
	credential string
	delivery   engine.Delivery
	eng        engine.Engine
	git        git.Session
	tree, home string
	// idleTimeout and promptTimeout are the session's (channel or debug override); the timers
	// are P10 half (b).
	idleTimeout, promptTimeout time.Duration
	log                        *eventLog

	mu              sync.Mutex
	state           string
	turn            int // the last turn started
	running         *engine.TurnHandle
	turnDone        chan struct{} // closed when the running turn's goroutine ends; nil when idle
	queue           []queued
	engineSessionID string
	participants    map[string]string // user id to role
	// calls are the tool calls seen, by tool_use_id, for the session: a resumed turn's result
	// answers a call the turn before made.
	calls map[string]call
}

// queued is a message waiting for the next turn (ADR 0020: queued, not injected).
type queued struct {
	user, block string
}

func (s *Service) lookup(id string) *live {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.live[id]
}

// resolveFixture is the fake's fixture directory: a bare name under Fixtures, else a path.
func (s *Service) resolveFixture(name string) (string, error) {
	if name == "" {
		name = defaultFixture
	}
	dir := name
	if !strings.ContainsAny(name, `/\`) && s.cfg.Fixtures != "" {
		dir = filepath.Join(s.cfg.Fixtures, name)
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", &Error{CodeUnknownFixture, filepath.ToSlash(dir) + " is not a fixture directory"}
	}
	return filepath.ToSlash(dir), nil
}

// Start is SessionService.start (05 Session lifecycle; 03 Registration and participants): the
// channel, the driver's registration and roles, the credential (R-41), then the session row,
// the clone (ADR 0022 companion) and the worktree row. A refused start is audited and leaves
// no session row and no tree.
func (s *Service) Start(ctx context.Context, req StartRequest) (Started, error) {
	actor := actorOf(req.As)
	if s.closing.Load() {
		return Started{}, &Error{CodeShuttingDown, "the bot is stopping"}
	}
	ch, ok := s.cfg.Channel(req.Channel)
	if !ok {
		return Started{}, s.refuse(actor, "", "start", "unknown-channel", &Error{CodeUnknownChannel, "no channel " + req.Channel})
	}
	if req.As == "" {
		return Started{}, badArgs("session start needs --as <user id>: a session's driver is a registered user")
	}
	if !s.cfg.DryRun && (req.Reach != "" || req.Fixture != "" || req.IdleTimeoutSeconds != 0 || req.PromptTimeoutSeconds != 0) {
		return Started{}, s.refuse(actor, "", "start", "live-instance",
			&Error{CodeLiveInstance, "--reach, --fixture and the timeout overrides are for a dry-run instance"})
	}
	mode, reach := ch.Mode, ch.Reach
	if req.Mode != "" {
		if _, ok := policy.ParseMode(req.Mode); !ok {
			return Started{}, badArgs("--mode " + req.Mode + ": want prompt or policy")
		}
		mode = req.Mode
	}
	if req.Reach != "" {
		if _, ok := policy.ParseReach(req.Reach); !ok {
			return Started{}, badArgs("--reach " + req.Reach + ": want read-only or code")
		}
		reach = req.Reach
	}
	if req.IdleTimeoutSeconds < 0 || req.PromptTimeoutSeconds < 0 {
		return Started{}, badArgs("a timeout override is a number of seconds above 0")
	}
	reg, found, err := s.registration(req.As)
	if err != nil {
		return Started{}, err
	}
	if !found {
		return Started{}, s.refuse(actor, "", "start", "unregistered", &Error{CodeUnregistered, req.As + " is not registered"})
	}
	if roles := splitRoles(req.Roles); req.Roles != "" && !policy.Permitted(roles, ch.Roles.Start, s.cfg.RoleGroups, false) {
		return Started{}, s.refuse(actor, "", "start", "roles",
			&Error{CodeRefused, "start needs a role in " + strings.Join(ch.Roles.Start, ",")})
	}
	credID := req.Credential
	if credID == "" {
		credID = reg.DefaultCredential
	}
	if credID == "" {
		return Started{}, s.refuse(actor, "", "start", "no-credential",
			&Error{CodeNoCredential, req.As + " has no default credential: pass --credential <id> or run register set-default"})
	}
	cred, err := credentials.Show(s.cfg.Store, credID)
	if err != nil {
		if e, ok := credentials.IsRefusal(err); ok {
			return Started{}, s.refuse(actor, "", "start", e.Code, &Error{e.Code, e.Message})
		}
		return Started{}, err
	}
	if cred.Status != credentials.StatusUsable {
		return Started{}, s.refuse(actor, "", "start", "credential",
			&Error{CodeCredentialStatus, credID + " is " + cred.Status})
	}
	var delivery engine.Delivery
	switch cred.Kind {
	case credentials.KindSubscription:
		delivery = engine.Delivery{Kind: engine.DeliveryConfigDir, Path: cred.StoreRef}
	case credentials.KindAPIKey:
		delivery = engine.Delivery{Kind: engine.DeliveryAPIKey, CredentialID: cred.ID}
	default:
		return Started{}, &Error{credentials.CodeBadKind, credID + " has kind " + cred.Kind}
	}
	fixture := ""
	if ch.Harness == "fake" {
		if fixture, err = s.resolveFixture(req.Fixture); err != nil {
			return Started{}, err
		}
	}

	// The session row first: its row id is the session id.
	var id string
	started := s.stamp()
	err = s.cfg.Store.Write(func(tx *store.Tx) error {
		var n int64
		if err := tx.Query("SELECT coalesce(max(rowid), 0) + 1 FROM session", nil, func(r *store.Row) error {
			n = r.Int(0)
			return nil
		}); err != nil {
			return err
		}
		id = git.SessionID(n)
		if _, err := tx.Exec(`INSERT INTO session (rowid, id, channel_id, project_id, driver_id, state, harness, credential_id, mode, reach, started_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, store.Int(n), store.Text(id), store.Text(ch.ID), store.Text(ch.Project),
			store.Text(req.As), store.Text(StateStarting), store.Text(ch.Harness), store.Text(credID), store.Text(mode),
			store.Text(reach), store.Text(started)); err != nil {
			return err
		}
		_, err := tx.Exec("INSERT INTO participant (session_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)",
			store.Text(id), store.Text(req.As), store.Text(RoleDriver), store.Text(started))
		return err
	})
	if err != nil {
		return Started{}, err
	}
	gs := git.Session{Root: s.cfg.Root, ID: id, Instance: s.cfg.Instance, Canonical: ch.Repo.Canonical,
		Remote: orDefault(ch.Repo.Remote, "origin"), DefaultBranch: orDefault(ch.Repo.DefaultBranch, "main")}
	l, stage, err := s.open(ctx, ch, gs, reg, credID, delivery, mode, reach, fixture, req)
	if err != nil {
		code, msg := "session-start", err.Error()
		var se *Error
		var ge *git.Error
		switch {
		case errors.As(err, &se):
			code, msg = se.Code, se.Message
		case errors.As(err, &ge):
			code, msg = ge.Code, ge.Message
		}
		// The row stays, ended, so its id is never handed out again (ids are max(rowid) + 1 and
		// no session row is ever deleted); open removed the tree and the ref.
		_ = s.cfg.Store.Write(func(tx *store.Tx) error {
			_, err := tx.Exec("UPDATE session SET state = ?, ended_at = ?, end_outcome = ? WHERE id = ?",
				store.Text(StateEnded), store.Text(s.stamp()), store.Text("refused: "+code), store.Text(id))
			return err
		})
		return Started{}, s.refuse(actor, id, "start", stage, &Error{code, msg})
	}
	// closing is read under the lock Shutdown takes to list the sessions: a session opened
	// while the bot stops is either in that list or interrupted here.
	s.mu.Lock()
	s.live[id] = l
	stopping := s.closing.Load()
	s.mu.Unlock()
	if stopping {
		_, _ = s.interruptSession(l, "shutdown")
		return Started{}, &Error{CodeShuttingDown, id + " was interrupted: the bot is stopping"}
	}
	return Started{ID: id}, nil
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// Start stages, the reason an action-refused event names when a start fails after its row.
const (
	stageClone  = "clone"
	stageHome   = "home"
	stageEngine = "engine"
	stageStore  = "store"
)

// open clones the tree, makes the session home, records the worktree and the session's
// waiting state, and returns the live session. A failure after the clone removes the tree and
// refs/sessions/<id> (RemoveTree), and the directories open made, so a failed start leaves
// nothing behind; it returns the stage that failed.
func (s *Service) open(ctx context.Context, ch Channel, gs git.Session, reg registration, credID string,
	delivery engine.Delivery, mode, reach, fixture string, req StartRequest) (*live, string, error) {
	if _, err := s.cfg.CreateTree(ctx, gs); err != nil {
		return nil, stageClone, err
	}
	dir := gs.Dir()
	tree, home, received := dir+"/tree", dir+"/home", dir+"/fake"
	fail := func(stage string, err error) (*live, string, error) {
		if s.cfg.RemoveTree != nil {
			if rerr := s.cfg.RemoveTree(ctx, gs); rerr != nil {
				err = errors.Join(err, rerr)
			}
		}
		// Only what open itself made: the home (the trailers file) and the fake's received dir.
		_ = os.RemoveAll(filepath.FromSlash(home))
		_ = os.RemoveAll(filepath.FromSlash(received))
		_ = os.Remove(filepath.FromSlash(dir))
		return nil, stage, err
	}
	if err := os.MkdirAll(filepath.FromSlash(home), 0o750); err != nil {
		return fail(stageHome, err)
	}
	if err := git.WriteTrailers(dir, nil); err != nil {
		return fail(stageHome, err)
	}
	if ch.Harness == "fake" {
		if err := os.MkdirAll(filepath.FromSlash(received), 0o750); err != nil {
			return fail(stageHome, err)
		}
	}
	eng, err := s.cfg.Engine(ch.Harness, fixture, received)
	if err != nil {
		if fixture != "" {
			return fail(stageEngine, &Error{CodeBadFixture, err.Error()})
		}
		return fail(stageEngine, err)
	}
	l := &live{id: gs.ID, ch: ch, mode: mode, reach: reach, driver: reg, credential: credID, delivery: delivery,
		eng: eng, git: gs, tree: tree, home: home, idleTimeout: ch.IdleTimeout, promptTimeout: ch.PromptTimeout,
		log: newEventLog(), state: StateWaitingForInput, participants: map[string]string{reg.User: RoleDriver}, calls: map[string]call{}}
	if req.IdleTimeoutSeconds > 0 {
		l.idleTimeout = time.Duration(req.IdleTimeoutSeconds) * time.Second
	}
	if req.PromptTimeoutSeconds > 0 {
		l.promptTimeout = time.Duration(req.PromptTimeoutSeconds) * time.Second
	}
	now := s.stamp()
	err = s.cfg.Store.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec(`INSERT INTO worktree (path, project_id, session_id, branch, state, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			store.Text(tree), store.Text(ch.Project), store.Text(gs.ID), store.Text(gs.Branch()), store.Text(worktreeActive),
			store.Text(now)); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE session SET state = ? WHERE id = ?", store.Text(StateWaitingForInput), store.Text(gs.ID)); err != nil {
			return err
		}
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindSessionStarted, Actor: actorOf(req.As), Session: gs.ID,
			Registration: reg.User, Details: details("channel", ch.ID, "project", ch.Project, "credential", credID,
				"mode", mode, "reach", reach, "harness", ch.Harness, "branch", gs.Branch(), "tree", tree, "fixture", fixture)})
		return err
	})
	if err != nil {
		return fail(stageStore, err)
	}
	// 04, Starting and the first message: the header's first line.
	l.log.add(Event{Time: now, Type: EventSessionStarted, User: reg.User,
		Summary: "Session " + gs.ID + " on " + ch.ProjectName + ", branch " + gs.Branch() + ", driver " + reg.Name + "."})
	return l, "", nil
}

// Shutdown is 03's shutdown for the sessions: no new messages, every running turn
// interrupted through its engine, then every open session marked interrupted (R-17). It
// returns the sessions it marked; when ctx ends first it returns CodeShutdownTimeout naming the
// sessions still open, which reconciliation covers at the next start.
func (s *Service) Shutdown(ctx context.Context) ([]string, error) {
	s.closing.Store(true)
	s.mu.Lock()
	lives := make([]*live, 0, len(s.live))
	for _, l := range s.live {
		lives = append(lives, l)
	}
	s.mu.Unlock()
	sort.Slice(lives, func(i, j int) bool { return lives[i].id < lives[j].id })
	var open []string
	for _, l := range lives {
		if !waitIdle(ctx, l) {
			open = append(open, l.id)
		}
	}
	if len(open) > 0 {
		return nil, &Error{CodeShutdownTimeout, "open sessions: " + strings.Join(open, " ")}
	}
	var marked []string
	for _, l := range lives {
		ok, err := s.interruptSession(l, "shutdown")
		if err != nil {
			return marked, err
		}
		if ok {
			marked = append(marked, l.id)
		}
	}
	return marked, nil
}

// waitIdle interrupts the session's running turn, again for any turn started meanwhile, and
// waits for it to end; false when ctx ends first.
func waitIdle(ctx context.Context, l *live) bool {
	for {
		l.mu.Lock()
		done, h := l.turnDone, l.running
		l.mu.Unlock()
		if done == nil {
			return true
		}
		if h != nil {
			_ = h.Interrupt()
		}
		select {
		case <-done:
		case <-ctx.Done():
			return false
		}
	}
}

// interruptSession marks an open session interrupted with its audit event and closes its log.
func (s *Service) interruptSession(l *live, reason string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.state == StateEnded || l.state == StateInterrupted {
		return false, nil
	}
	now := s.stamp()
	err := s.cfg.Store.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec("UPDATE session SET state = ?, ended_at = ? WHERE id = ?",
			store.Text(StateInterrupted), store.Text(now), store.Text(l.id)); err != nil {
			return err
		}
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindSessionInterrupted, Actor: ActorInstance,
			Session: l.id, Details: details("reason", reason, "state", l.state)})
		return err
	})
	if err != nil {
		return false, err
	}
	l.state = StateInterrupted
	l.log.add(Event{Time: now, Type: EventSessionInterrupted, Summary: reason})
	l.log.close()
	return true, nil
}

// Counts are status's session line (05 Bot lifecycle).
type Counts struct {
	Open             int `json:"open"`
	WaitingForAnswer int `json:"waitingForAnswer"`
	Interrupted      int `json:"interrupted"`
}

// CountSessions reads the session states from the store.
func CountSessions(q store.Querier) (Counts, error) {
	var c Counts
	err := q.Query("SELECT state, count(*) FROM session GROUP BY state", nil, func(r *store.Row) error {
		n := int(r.Int(1))
		switch r.Text(0) {
		case StateEnded:
		case StateInterrupted:
			c.Interrupted += n
		case StateWaitingForAnswer:
			c.WaitingForAnswer += n
			c.Open += n
		default:
			c.Open += n
		}
		return nil
	})
	return c, err
}
