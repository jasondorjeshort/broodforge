// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/session"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// serveCmd is `broodforge serve [--discord live|dry-run]` (05-debug-interface.md, Bot
// lifecycle): the bot in the foreground through 03's startup steps 1 to 8, then the control
// socket until `stop`, SIGTERM or Ctrl-C. A failed step is `error: start-<step>: <reason>`,
// exit 1. It prints `serving  <socket>` once the socket answers. Only dry-run is built: the
// gateway (steps 9 and 10) is P12's, so live is refused at the start.
var serveCmd = Command{
	Name:  "serve",
	Class: ClassFile,
	Flags: []Flag{
		{Name: "discord", Kind: FlagString, Value: "live|dry-run", Usage: "override discord.mode for this run"},
	},
	Columns: serveColumns,
	Local:   runServe,
}

var serveColumns = []string{"result", "socket"}

func init() { commands = append(commands, &serveCmd) }

// Start step names: `error: start-<step>: <reason>` (03-architecture.md, Startup).
const (
	stepRoot        = "start-root"
	stepConfig      = "start-config"
	stepDiscord     = "start-discord"
	stepCredentials = "start-credentials"
	stepStore       = "start-store"
	stepEngine      = "start-engine"
	stepPreflight   = "start-preflight"
	stepReconcile   = "start-reconcile"
	stepControl     = control.CodeStartControl
)

// defaultStopTimeout is how long shutdown waits for the running turns (03, Shutdown).
const defaultStopTimeout = 30 * time.Second

// bot is one running instance: what the startup steps opened and the handlers serve.
type bot struct {
	root       string
	mu         sync.Mutex // guards cfg and lastGood, which config reload replaces
	cfg        *config.Config
	lastGood   *time.Time // set when the bot runs on data/config-last-good/
	discord    string
	st         *store.Store
	aud        *audit.Writer
	creds      *credentials.Service
	backend    spawn.Backend
	svc        *session.Service
	adapter    *discord.Adapter
	gateway    *discord.Gateway // nil on a dry run
	wt         *WorktreeService
	srv        *control.Server
	addr       string
	stopReq    chan stopRequest
	engineName string
}

// stopRequest is `stop` asking the main loop to shut down.
type stopRequest struct {
	timeout   time.Duration
	outcome   chan stopOutcome
	responded chan struct{}
}

type stopOutcome struct {
	marked []string
	err    error
}

func stepError(step string, err error) error {
	var ce *control.Error
	if errors.As(err, &ce) && strings.HasPrefix(ce.Code, "start-") {
		return ce
	}
	msg := err.Error()
	var inv *config.InvalidError
	if errors.As(err, &inv) && len(inv.Problems) > 0 {
		msg = inv.Problems[0].String()
		if len(inv.Problems) > 1 {
			msg += fmt.Sprintf(" (and %d more)", len(inv.Problems)-1)
		}
	}
	return control.NewError(step, msg)
}

func runServe(_ context.Context, env *Env) error {
	b, err := startBot(env)
	if err != nil {
		return err
	}
	served := make(chan error, 1)
	go func() { served <- b.srv.Serve() }()
	// The socket prints as the not-running error names it, unescaped.
	if env.JSON {
		line, err := json.Marshal(map[string]string{"result": "serving", "socket": b.addr})
		if err == nil {
			err = env.Out.Record(line, serveColumns)
		}
		if err != nil {
			return err
		}
	} else {
		env.Out.SetToken("serving")
		fmt.Fprintln(env.Out.w, "serving  "+b.addr)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	var req *stopRequest
	timeout := defaultStopTimeout
	select {
	case r := <-b.stopReq:
		req, timeout = &r, r.timeout
	case <-sigs:
	case err := <-served:
		b.close()
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	marked, serr := b.svc.Shutdown(ctx)
	cancel()
	if req != nil {
		req.outcome <- stopOutcome{marked, serr}
		select {
		case <-req.responded:
		case <-time.After(5 * time.Second):
		}
	}
	b.close()
	if err := <-served; err != nil && serr == nil {
		serr = err
	}
	if serr != nil {
		var se *session.Error
		if errors.As(serr, &se) {
			return control.NewError(se.Code, se.Message)
		}
		return serr
	}
	return nil
}

func (b *bot) close() {
	if b.gateway != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		b.gateway.Close(ctx)
		cancel()
	}
	if b.adapter != nil {
		b.adapter.Close()
	}
	_ = b.srv.Close()
	_ = b.st.Close()
}

// startBot runs 03's startup steps 1 to 8; a failure closes what was opened.
func startBot(env *Env) (*bot, error) {
	// (1) The instance root.
	root, err := filepath.Abs(env.Root)
	if err != nil {
		return nil, stepError(stepRoot, err)
	}
	root = filepath.ToSlash(root)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil, control.NewError(stepRoot, root+" is not a directory")
	}
	b := &bot{root: root, stopReq: make(chan stopRequest)}

	// (2) config/, or the last-good copy (ADR 0021); credential ids are checked at step 3.
	c, loadErr := config.Load(root, config.AnyCredential)
	if loadErr != nil {
		lg, err := config.LoadLastGood(root, config.AnyCredential)
		if err != nil {
			return nil, stepError(stepConfig, loadErr)
		}
		t, _ := config.LastGoodTime(root)
		c, b.lastGood = lg, &t
	}
	b.discord = c.Instance.Discord.Mode
	if f := env.String("discord"); f != "" {
		b.discord = f
	}
	switch b.discord {
	case config.DiscordDryRun:
	case config.DiscordLive:
		// config check requires these only when the file says live; --discord live needs them too.
		var missing []string
		for _, k := range []struct{ key, v string }{{"discord.applicationId", c.Instance.Discord.ApplicationID},
			{"discord.guildId", c.Instance.Discord.GuildID}, {"discord.botCredentialId", c.Instance.Discord.BotCredentialID}} {
			if k.v == "" {
				missing = append(missing, k.key)
			}
		}
		if len(missing) > 0 {
			return nil, control.NewError(stepDiscord, "live needs "+strings.Join(missing, ", ")+" in config/instance.json")
		}
	default:
		return nil, control.NewError(stepDiscord, "discord mode "+quoteArg(b.discord)+": want live or dry-run")
	}

	// The control address is probed as soon as instanceName is known, before anything is
	// written: a second bot on this root must neither rewrite the last-good copy and the audit
	// log nor reconcile the first bot's open sessions as interrupted (step 7).
	addr, err := control.Address(root, c.Instance.InstanceName)
	if err != nil {
		return nil, stepError(stepControl, err)
	}
	if err := probeControl(context.Background(), addr, control.Dial); err != nil {
		return nil, err
	}
	b.addr = addr

	// (3) The key file and the credential store.
	keyPath, err := credentials.KeyPath(root, c.Instance.InstanceName, c.Instance.Credentials.KeyFile, os.LookupEnv)
	if err != nil {
		return nil, stepError(stepCredentials, err)
	}
	if _, err := credentials.OpenVault(credentials.StorePath(root), keyPath); err != nil {
		return nil, stepError(stepCredentials, err)
	}

	// (4) data/store.sqlite and its migrations.
	if b.st, err = store.Create(root); err != nil {
		return nil, stepError(stepStore, localError(err))
	}
	fail := func(step string, err error) (*bot, error) {
		_ = b.st.Close()
		return nil, stepError(step, err)
	}
	var sink audit.Sink
	if p := c.Instance.Audit.FileSink; p != nil && *p != "" {
		sink = audit.NewFileSink(*p)
	}
	b.aud = audit.NewWriter(sink)
	// Step 3's second half: the configuration's credential ids against the store's records.
	if c, err = config.LoadDir(c.Dir, credentials.Exists(b.st)); err != nil {
		return fail(stepCredentials, err)
	}
	b.cfg = c
	for _, w := range c.Warnings {
		fmt.Fprintf(env.Stderr, "warning: %s\n", w)
	}
	var invalid error
	if b.lastGood != nil {
		invalid = loadErr
	}
	if _, err := b.recordConfig(c, invalid, "last-good"); err != nil {
		return fail(stepConfig, err)
	}

	// (5) The engine binary and its version, when a channel runs the real engine.
	needEngine := false
	for _, ch := range c.Channels() {
		needEngine = needEngine || ch.Harness == config.HarnessClaudeCode
	}
	b.engineName = config.HarnessFake
	if needEngine {
		b.engineName = config.HarnessClaudeCode
		_, got, err := spawn.EngineVersion(c.Instance.Engine.Path)
		if err != nil {
			return fail(stepEngine, spawnError(err))
		}
		if got != c.Instance.Engine.Version {
			return fail(stepEngine, fmt.Errorf("engine.version is %s, %s --version says %s", c.Instance.Engine.Version, c.Instance.Engine.Path, got))
		}
	}

	// (6) spawn preflight; with only fake channels the engine checks do not apply.
	if b.backend, err = spawn.New(spawnConfig(c)); err != nil {
		return fail(stepPreflight, spawnError(err))
	}
	for _, chk := range b.backend.Preflight() {
		if chk.OK || (!needEngine && (chk.Name == "engine-path" || chk.Name == "engine-version")) {
			continue
		}
		return fail(stepPreflight, fmt.Errorf("%s: %s", chk.Code(), chk.Detail))
	}
	if err := spawn.JoinKillOnCloseJob(); err != nil {
		fmt.Fprintf(env.Stderr, "warning: serve: the bot is in no job of its own: %v\n", err)
	}
	if b.creds, err = credentials.NewService(b.st, b.aud, credentials.Config{Root: root, KeyFile: keyPath,
		EnginePath: c.Instance.Engine.Path, Spawn: b.backend}); err != nil {
		return fail(stepCredentials, credentialError(err))
	}

	// (7) Reconciliation (ADR 0021): instance-started, then the report's events after it.
	if err := b.started(); err != nil {
		return fail(stepReconcile, err)
	}
	if _, err := session.Reconcile(b.st, b.aud, root, time.Now()); err != nil {
		return fail(stepReconcile, err)
	}

	b.wt = &WorktreeService{Root: root, Config: c, Current: b.config, Store: b.st, Audit: b.aud, Backend: b.backend,
		Secret: b.creds.Secret}
	// The Discord adapter without the gateway (P11): the service tells it of each new session.
	panels, err := storedPanels(b.st)
	if err != nil {
		return fail(stepDiscord, err)
	}
	b.adapter = discord.New(discord.Config{Config: b.config, Channel: b.channel, DryRun: b.discord == config.DiscordDryRun,
		Groups:      func() map[string][]string { return b.config().Instance.RoleGroups },
		Credentials: discordCredentials{b.creds}, Panels: panels, SavePanel: savePanel(b.st)})
	if b.discord == config.DiscordLive {
		// Step 9's first half: the token and the ids, before anything is served; Open connects.
		if b.gateway, err = b.newGateway(env, keyPath); err != nil {
			return fail(stepDiscord, err)
		}
	}
	b.svc = session.New(session.Config{
		Root: root, Instance: c.Instance.InstanceName, DryRun: b.discord == config.DiscordDryRun, Started: b.adapter.SessionStarted,
		Groups:  func() map[string][]string { return b.config().Instance.RoleGroups },
		Channel: b.channel, Fixtures: fixturesDir(c), Handoff: b.wt.Handoff,
		Engine: b.engine, CreateTree: func(ctx context.Context, s git.Session) (string, error) {
			return git.Create(ctx, b.backend, s)
		},
		RemoveTree:     func(ctx context.Context, s git.Session) error { return git.Discard(ctx, b.backend, s) },
		FetchCanonical: b.fetchCanonical,
		Refresh:        func(ctx context.Context, s git.Session) (string, error) { return git.Refresh(ctx, b.backend, s) },
		Workflow:       sessionWorkflow,
		Commits:        b.wt.Commits,
		ProjectRepo:    b.projectRepo,
		Store:          b.st, Audit: b.aud,
	})
	b.adapter.Attach(b.svc)

	// (7) continued: every interrupted session held in memory, so end reaches it (ADR 0026). Its
	// View's notice waits in the coalescer, which starts below.
	held, err := b.svc.LoadInterrupted(context.Background())
	if err != nil {
		b.adapter.Close()
		return fail(stepReconcile, err)
	}
	heldLine := "-"
	if len(held) > 0 {
		heldLine = strings.Join(held, " ")
	}
	fmt.Fprintln(env.Stderr, "held  "+heldLine)

	// (8) The control socket.
	reg := control.NewRegistry()
	b.register(reg)
	if b.srv, err = control.Listen(b.addr, reg); err != nil {
		b.adapter.Close()
		return fail(stepControl, err)
	}
	var open func(context.Context) error
	if b.gateway != nil {
		open = b.gateway.Open
	}
	if err := openDiscord(open, b.adapter.Run); err != nil {
		b.close()
		return nil, stepError(stepDiscord, err)
	}
	return b, nil
}

// openDiscord is startup steps 9 and 10 and the coalescer's start, in ADR 0026's order: a dry
// run (open nil) starts the coalescer at once; a live instance opens the gateway first (the
// commands, the sweep binding the venues of earlier runs, the connection, the start-up report)
// and starts the coalescer only after, so a held session's notice is never sent to a venue the
// gateway does not know yet. A failed open starts nothing.
func openDiscord(open func(context.Context) error, run func()) error {
	if open == nil {
		run()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	err := open(ctx)
	cancel()
	if err != nil {
		return err
	}
	run()
	return nil
}

// newGateway builds the live gateway: the bot token from the credential store, the ids from
// config/instance.json, the start-up report (04's operator flow) for the notice channel.
func (b *bot) newGateway(env *Env, keyPath string) (*discord.Gateway, error) {
	d := b.cfg.Instance.Discord
	token, err := credentials.BotToken(b.st, b.root, keyPath, d.BotCredentialID)
	if err != nil {
		return nil, credentialError(err)
	}
	report, err := startupReport(b.st, b.cfg)
	if err != nil {
		return nil, err
	}
	return discord.NewGateway(b.adapter, discord.GatewayConfig{Token: string(token), ApplicationID: d.ApplicationID,
		GuildID: d.GuildID, NoticeChannel: d.NoticeChannelID, Instance: b.cfg.Instance.InstanceName, Report: report, Log: env.Stderr})
}

// startupReport is 04's operator flow for the notice channel: `Broodforge <instance> started:
// engine <version>, <n> projects. Interrupted sessions: s0003 (#broodforge).`; not 04: the
// engine is `fake` when none is configured, and the second sentence is left out when none was.
func startupReport(st *store.Store, c *config.Config) (string, error) {
	version := c.Instance.Engine.Version
	if version == "" {
		version = config.HarnessFake
	}
	text := fmt.Sprintf("Broodforge %s started: engine %s, %d projects.", c.Instance.InstanceName, version, len(c.Projects))
	rows, err := session.List(st, session.StateInterrupted)
	if err != nil {
		return "", err
	}
	var parts []string
	for _, r := range rows {
		parts = append(parts, r.ID+" (<#"+r.Channel+">)")
	}
	if len(parts) > 0 {
		text += " Interrupted sessions: " + strings.Join(parts, ", ") + "."
	}
	return text, nil
}

func quoteArg(s string) string { return `"` + s + `"` }

func fixturesDir(c *config.Config) string {
	if f := c.Instance.Engine.FakeFixtures; f != nil {
		return *f
	}
	return ""
}

// channel translates one configured channel for the session service, defaults merged in.
func (b *bot) channel(id string) (session.Channel, bool) {
	c := b.config()
	p := c.ChannelProject(id)
	if p == nil {
		return session.Channel{}, false
	}
	e := c.Effective(p, id)
	w := p.WorkflowOrDefault()
	deref := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	return session.Channel{ID: id, Project: p.ID, ProjectName: p.Name, Reach: e.Reach, Mode: e.Mode, Harness: e.Harness,
		Roles:       session.Roles{Start: e.Roles.Start, Contribute: e.Roles.Contribute, Answer: e.Roles.Answer},
		IdleTimeout: time.Duration(e.IdleTimeoutMinutes) * time.Minute, PromptTimeout: time.Duration(e.PromptTimeoutMinutes) * time.Minute,
		Model: deref(e.Model), Effort: deref(e.Effort),
		Repo: session.Repo{Canonical: p.Repo.Canonical, Remote: p.Repo.Remote, DefaultBranch: p.Repo.DefaultBranch,
			Handoff: p.HandoffOrDefault(), GitCredentialID: p.Repo.GitCredentialID},
		Workflow: session.WorkflowKey{Inject: w.Inject, Hooks: w.Hooks}}, true
}

// projectRepo is a configured project's repository and handoff mode, for a held session whose
// channel may have left the configuration (session.Config.ProjectRepo).
func (b *bot) projectRepo(id string) (session.Repo, bool) {
	p := b.config().Project(id)
	if p == nil {
		return session.Repo{}, false
	}
	return session.Repo{Canonical: p.Repo.Canonical, Remote: p.Repo.Remote, DefaultBranch: p.Repo.DefaultBranch,
		Handoff: p.HandoffOrDefault(), GitCredentialID: p.Repo.GitCredentialID}, true
}

// fetchCanonical is the fetch at session start (session.Config.FetchCanonical): git.Fetch in
// the canonical with the backend's askpass helper and the project's git credential's value, as
// the push carries it. A value that cannot be read skips the fetch as stale with the reason.
func (b *bot) fetchCanonical(ctx context.Context, s git.Session, credentialID string) (git.FetchResult, error) {
	var token []byte
	if credentialID != "" {
		v, err := b.creds.Secret(credentialID)
		if err != nil {
			return git.FetchResult{Stale: true, Error: "git credential " + credentialID + ": " + err.Error()}, nil
		}
		token = v
	}
	return git.Fetch(ctx, s, b.backend.Askpass(), token)
}

// engine is the session service's engine factory: the fake per session (its turn counter is
// the session's), the real engine through the configured backend.
func (b *bot) engine(harness, fixture, received string) (engine.Engine, error) {
	switch harness {
	case config.HarnessFake:
		return engine.NewFake(fixture, received)
	case config.HarnessClaudeCode:
		return &engine.ClaudeCode{Backend: b.backend, Secret: b.creds.Secret}, nil
	}
	return nil, fmt.Errorf("harness %s: want %s or %s", quoteArg(harness), config.HarnessClaudeCode, config.HarnessFake)
}

// config is the configuration as loaded now; lastGoodAt is when the last-good copy it runs on
// was written, nil when it runs on config/. config reload replaces both.
func (b *bot) config() *config.Config {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cfg
}

func (b *bot) lastGoodAt() *time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastGood
}

// recordConfig is ADR 0021's load record: for a valid config/ (invalid nil), the last-good
// copy rewritten and configuration-changed with the file hash and the changed top-level keys,
// which it returns (keys, when given, are those changed keys: a reload's, against the running
// configuration); for an invalid one, configuration-invalid with the errors and what the bot
// runs on instead (the last-good copy at start, the previous configuration at a reload).
func (b *bot) recordConfig(c *config.Config, invalid error, using string, keys ...string) (string, error) {
	if invalid != nil {
		return "", b.st.Write(func(tx *store.Tx) error {
			_, err := b.aud.Append(tx, audit.Event{Kind: audit.KindConfigurationInvalid, Actor: session.ActorInstance,
				Details: session.Details("errors", control.OneLine(invalid.Error()), "using", using)})
			return err
		})
	}
	var prev *config.Config
	if lg, err := config.LoadLastGood(b.root, config.AnyCredential); err == nil {
		prev = lg
	}
	if err := config.WriteLastGood(b.root, c); err != nil {
		return "", err
	}
	h := sha256.New()
	for _, f := range c.Files {
		h.Write([]byte(f.Name + "\n"))
		h.Write(f.Raw)
	}
	if keys == nil {
		keys = changedKeys(prev, c)
	}
	changed := strings.Join(keys, ",")
	return changed, b.st.Write(func(tx *store.Tx) error {
		_, err := b.aud.Append(tx, audit.Event{Kind: audit.KindConfigurationChanged, Actor: session.ActorInstance,
			Details: session.Details("hash", hex.EncodeToString(h.Sum(nil)), "changed", changed)})
		return err
	})
}

// changedKeys names what differs from the previous load: instance.json's top-level keys, and
// each project file added, removed or changed. With no previous load every key is new.
func changedKeys(prev, cur *config.Config) []string {
	files := func(c *config.Config) map[string]json.RawMessage {
		m := map[string]json.RawMessage{}
		if c != nil {
			for _, f := range c.Files {
				m[f.Name] = f.Stripped
			}
		}
		return m
	}
	before, after := files(prev), files(cur)
	keys := func(raw json.RawMessage) map[string]json.RawMessage {
		m := map[string]json.RawMessage{}
		_ = json.Unmarshal(raw, &m)
		return m
	}
	var out []string
	bi, ai := keys(before["instance.json"]), keys(after["instance.json"])
	for k, v := range ai {
		if !bytes.Equal(bi[k], v) {
			out = append(out, k)
		}
	}
	for k := range bi {
		if _, ok := ai[k]; !ok {
			out = append(out, k)
		}
	}
	for name, v := range after {
		if name != "instance.json" && !bytes.Equal(before[name], v) {
			out = append(out, name)
		}
	}
	for name := range before {
		if _, ok := after[name]; !ok && name != "instance.json" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// started writes instance-started, the head of the start-up report.
func (b *bot) started() error {
	cfgState := "ok"
	if b.lastGood != nil {
		cfgState = "last-good"
	}
	return b.st.Write(func(tx *store.Tx) error {
		_, err := b.aud.Append(tx, audit.Event{Kind: audit.KindInstanceStarted, Actor: session.ActorInstance,
			Details: session.Details("instance", b.cfg.Instance.InstanceName, "discord", b.discord,
				"config", cfgState, "engine", b.engineName, "backend", b.backend.Name())})
		return err
	})
}

// register puts every command the bot serves on the registry: the session service, status
// and stop, and the write handlers of earlier passes (credential, worktree). Reads the bot
// does not register (audit, usage, reconcile, session list, register list|show) read the
// store file (cli.go, execute).
func (b *bot) register(reg *control.Registry) {
	CredentialHandlers(b.creds)(reg)
	WorktreeHandlers(b.wt)(reg)
	SessionHandlers(b.svc)(reg)
	DiscordHandlers(b.adapter, b.svc)(reg)
	reg.Handle("status", b.status)
	reg.Handle("stop", b.stop)
	reg.Handle("config.reload", b.reload)
}

// stop is `stop [--timeout 30]`: 03's shutdown, answered after the sessions are interrupted
// and before the socket closes.
func (b *bot) stop(ctx context.Context, call *control.Call) (json.RawMessage, error) {
	var a struct{ Timeout int }
	if err := json.Unmarshal(call.Args, &a); err != nil {
		return nil, control.NewError(control.CodeBadArgs, err.Error())
	}
	timeout := defaultStopTimeout
	if a.Timeout > 0 {
		timeout = time.Duration(a.Timeout) * time.Second
	}
	req := stopRequest{timeout: timeout, outcome: make(chan stopOutcome, 1), responded: make(chan struct{})}
	select {
	case b.stopReq <- req:
	default:
		return nil, control.NewError(session.CodeShuttingDown, "the bot is stopping already")
	}
	call.AfterResponse(func() { close(req.responded) })
	var out stopOutcome
	select {
	case out = <-req.outcome:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if out.err != nil {
		var se *session.Error
		if errors.As(out.err, &se) {
			return nil, control.NewError(se.Code, se.Message)
		}
		return nil, out.err
	}
	sessions := out.marked
	if sessions == nil {
		sessions = []string{}
	}
	return json.Marshal(struct {
		Result      string   `json:"result"`
		Interrupted int      `json:"interrupted"`
		Sessions    []string `json:"sessions"`
	}{"stopped", len(out.marked), sessions})
}

// dialFunc opens a control connection: control.Dial, or a test's.
type dialFunc func(ctx context.Context, addr string) (io.ReadWriteCloser, error)

// probeControl lets the start go on only when nothing listens at addr (not-running). A bot
// answering is `start-control: another bot owns <addr>`; any other failure (connect-failed:
// access denied, every pipe instance busy) may be a live bot too, so it refuses with the reason.
func probeControl(ctx context.Context, addr string, dial dialFunc) error {
	conn, err := dial(ctx, addr)
	if err == nil {
		_ = conn.Close()
		return control.NewError(stepControl, "another bot owns "+addr)
	}
	var ce *control.Error
	if errors.As(err, &ce) && ce.Code == control.CodeNotRunning {
		return nil
	}
	return control.NewError(stepControl, "cannot tell whether a bot owns "+addr+": "+err.Error())
}
