// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/engine"
	"github.com/jasondorjeshort/broodforge/internal/hook"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Error codes of engine run and engine record.
const (
	CodeEngineExit = "engine-exit" // the turn's exit code was not 0
	CodeBadFixture = "bad-fixture" // the fixture directory or its fixture.json cannot be read
	CodeNoTurn     = "no-turn"     // the fixture has no turn-<n>.jsonl
	// CodeBadReceivedDir: received-<n>.json cannot be written there (missing, not a directory, read-only).
	CodeBadReceivedDir = "bad-received-dir"
)

// Outcome tokens of engine run, compared by its --expect.
const (
	tokenResult   = "result"
	tokenNoResult = "no-result"
)

// engineRun is the adapter alone, no store, no session (05-debug-interface.md, A turn against
// the engine): `broodforge engine run --fake <fixture dir> [--turn <n>] [--cwd <dir>]
// [--received-dir <dir>] --prompt <text> [--requests allow|deny] [--expect <value>]` replays a
// fixture; `engine run --real --credential <id> --cwd <dir> (--prompt <text> | --answer
// <tool_use_id>:allow|deny [--reason <text>]) [--resume <engine session id>] [--model <model>]
// [--workflow <dir>] [--mode prompt|policy|auto] [--requests allow|deny]` runs the configured
// engine through the spawn backend (runEngineReal, below); --requests answers every request
// (requestsFlag), --answer resumes a deferred call (answerFlag). Either streams the turn's
// EngineEvents as JSONL on stdout, in text mode too, and exits with the turn's exit code (any
// non-zero code is 1, `error: engine-exit: ...`). Its outcome token for --expect is `result`
// when the turn produced a result event and `no-result` otherwise; a non-zero exit is the
// refusal engine-exit.
var engineRun = Command{
	Name:  "engine run",
	Class: ClassStandalone,
	Flags: []Flag{
		{Name: "fake", Kind: FlagString, Value: "<fixture dir>", Usage: "replay this fixture directory"},
		{Name: "real", Kind: FlagBool, Usage: "run the configured engine (needs --root)"},
		{Name: "credential", Kind: FlagString, Value: "<id>", Usage: "--real: the credential the engine runs as"},
		{Name: "resume", Kind: FlagString, Value: "<engine session id>", Usage: "--real: resume this engine session"},
		{Name: "model", Kind: FlagString, Value: "<model>", Usage: "--real: the engine's --model (default: the account's)"},
		{Name: "workflow", Kind: FlagString, Value: "<dir>", Usage: "--real: BROODFORGE_WORKFLOW, a copy whose hooks.json the hook chains"},
		modeFlag,
		{Name: "turn", Kind: FlagInt, Default: "1", Value: "<n>", Usage: "--fake: the turn-<n>.jsonl to play"},
		{Name: "cwd", Kind: FlagString, Value: "<dir>", Usage: "the request's working directory"},
		{Name: "received-dir", Kind: FlagString, Value: "<dir>", Usage: "--fake: where received-<n>.json goes (default: the fixture directory)"},
		{Name: "prompt", Kind: FlagString, Value: "<text>", Usage: "the turn's input"},
		requestsFlag,
		answerFlag,
		reasonFlag,
	},
	Expect: true,
	Local:  runEngine,
}

// answerFlag resumes a deferred call instead of sending input (05-debug-interface.md, engine
// run --real): the turn's TurnRequest.Answer, which the adapter hands the hook as
// BROODFORGE_ANSWER, writing no stdin line (C27), as a session's resume does. It needs a session
// to resume (--resume; engine record's previous turn) and excludes --prompt. reasonFlag is a
// deny answer's text, the hook's BROODFORGE_ANSWER_REASON (a session sends one on a deny only).
var (
	answerFlag = Flag{Name: "answer", Kind: FlagString, Value: "<tool_use_id>:allow|deny",
		Usage: "--real: resume the deferred call with this answer, no input (needs a resume; excludes --prompt)"}
	reasonFlag = Flag{Name: "reason", Kind: FlagString, Value: "<text>",
		Usage: "--answer <id>:deny: the deny text the hook returns (default: its own)"}
)

// answerOf reads --answer and --reason: nil when neither is given; bad-args for a value that is
// not <tool_use_id>:allow|deny, for --answer with --prompt, for --reason without a deny answer.
// Whether there is a session to resume is the caller's check.
func answerOf(env *Env, name string) (*engine.TurnAnswer, string, error) {
	v, reason := env.String("answer"), env.String("reason")
	if v == "" {
		if reason != "" {
			return nil, "", control.NewError(control.CodeBadArgs, name+": --reason needs --answer <tool_use_id>:deny")
		}
		return nil, "", nil
	}
	i := strings.LastIndexByte(v, ':')
	if i <= 0 || (v[i+1:] != requestsAllow && v[i+1:] != requestsDeny) {
		return nil, "", control.NewError(control.CodeBadArgs, name+": --answer "+v+": want <tool_use_id>:allow|deny")
	}
	a := &engine.TurnAnswer{ToolUseID: v[:i], Decision: v[i+1:]}
	switch {
	case env.String("prompt") != "":
		return nil, "", control.NewError(control.CodeBadArgs, name+": --prompt and --answer exclude each other: a resume writes no input")
	case reason != "" && a.Decision != requestsDeny:
		return nil, "", control.NewError(control.CodeBadArgs, name+": --reason needs --answer <tool_use_id>:deny")
	}
	return a, reason, nil
}

// requestsFlag is how a standalone turn answers every request (03-hook-contract-prompt-tool.md,
// Debug surface): each prompt event the engine raises, or the fake waits on, is answered at
// once, one stderr line each, so no standalone run waits on a person. --fake always answers;
// --real and engine record answer under --mode prompt only.
var requestsFlag = Flag{Name: "requests", Kind: FlagString, Value: "<allow|deny>",
	Usage: "answer every request this way (--fake, or --mode prompt; default deny)"}

// Answer policies of --requests.
const (
	requestsAllow = "allow"
	requestsDeny  = "deny"
)

// requestsDenyMessage is the deny --requests deny writes (a placeholder until the owner words it).
const requestsDenyMessage = "Denied by Broodforge: answered by the recording's policy"

// requestsPolicy is --requests checked: "" is deny; with answers false any value is refused,
// since that turn raises no request it could answer.
func requestsPolicy(env *Env, name string, answers bool, why string) (string, error) {
	v := env.String("requests")
	switch {
	case v != "" && !answers:
		return "", control.NewError(control.CodeBadArgs, name+": --requests "+v+": "+why)
	case v == "":
		return requestsDeny, nil
	case v == requestsAllow || v == requestsDeny:
		return v, nil
	}
	return "", control.NewError(control.CodeBadArgs, name+": --requests "+v+": want allow or deny")
}

func init() { commands = append(commands, &engineRun) }

func runEngine(ctx context.Context, env *Env) error {
	dir := env.String("fake")
	switch {
	case env.Bool("real") && dir != "":
		return &usageError{"engine run: --fake and --real exclude each other"}
	case env.Bool("real"):
		return runEngineReal(ctx, env, "engine run", nil)
	case env.String("workflow") != "":
		return &usageError{"engine run: --workflow needs --real"}
	case env.String("mode") != "":
		return &usageError{"engine run: --mode needs --real"}
	case env.String("answer") != "" || env.String("reason") != "":
		return &usageError{"engine run: --answer and --reason need --real"}
	case dir == "":
		return &usageError{"engine run: --fake <fixture dir> or --real is required"}
	}
	if env.String("prompt") == "" {
		return &usageError{"engine run: --prompt <text> is required"}
	}
	policy, err := requestsPolicy(env, "engine run", true, "")
	if err != nil {
		return err
	}
	n := env.Int("turn")
	if n < 1 {
		return &usageError{fmt.Sprintf("engine run: --turn %d: turns count from 1", n)}
	}
	fake, err := engine.NewFake(dir, env.String("received-dir"))
	if errors.Is(err, engine.ErrReceivedDir) {
		return control.NewError(CodeBadReceivedDir, err.Error())
	}
	if err != nil {
		return control.NewError(CodeBadFixture, err.Error())
	}
	fake.SetNextTurn(n)
	h, err := fake.StartTurn(engine.TurnRequest{Cwd: env.String("cwd"), Input: env.String("prompt")})
	if errors.Is(err, engine.ErrNoTurn) {
		return control.NewError(CodeNoTurn, err.Error())
	}
	if errors.Is(err, engine.ErrReceivedDir) {
		return control.NewError(CodeBadReceivedDir, err.Error())
	}
	if err != nil {
		return err
	}
	_, err = streamTurn(env, h, policy)
	return err
}

// streamTurn prints every event of the turn as it comes, drains it to its end, sets the
// outcome token and returns the turn's refusal: engine-exit for an interrupted turn or a
// non-zero exit. requests, when set, answers every prompt event (answerRequest).
func streamTurn(env *Env, h engine.TurnHandle, requests string) (string, error) {
	id, _, err := streamTurnExit(env, h, requests)
	return id, err
}

// answerRequest answers one prompt event by the --requests policy and says so on stderr:
// `request  <request id>  <tool>  <agent id or ->  allow|deny`. A request the turn no longer
// holds open (its stdin closed at the result first) is a warning line.
func answerRequest(env *Env, h engine.TurnHandle, p *engine.PromptEvent, requests string) {
	agent := p.AgentID
	if agent == "" {
		agent = "-"
	}
	if err := h.Answer(p.RequestID, requests == requestsAllow, requestsDenyMessage, false); err != nil {
		fmt.Fprintf(env.Stderr, "warning: request  %s  %s  %s  not answered: %v\n", p.RequestID, p.Tool, agent, err)
		return
	}
	fmt.Fprintf(env.Stderr, "request  %s  %s  %s  %s\n", p.RequestID, p.Tool, agent, requests)
}

// streamTurnExit is streamTurn that also returns the engine session id (its first init's)
// and the exit event.
func streamTurnExit(env *Env, h engine.TurnHandle, requests string) (string, *engine.ExitEvent, error) {
	token := tokenNoResult
	var exit *engine.ExitEvent
	var printErr error
	sessionID := ""
	for ev := range h.Events {
		switch ev.Kind {
		case engine.KindResult:
			token = tokenResult
		case engine.KindExit:
			exit = ev.Exit
		case engine.KindInit:
			if sessionID == "" {
				sessionID = ev.Init.EngineSessionID
			}
		case engine.KindAssistant, engine.KindToolResult, engine.KindRateLimit, engine.KindPrompt, engine.KindUnknown:
		}
		if printErr == nil {
			line, err := ev.MarshalJSON() // not json.Marshal, which would escape HTML again
			if err == nil {
				err = env.Out.Event(line, nil)
			}
			printErr = err
		}
		if ev.Kind == engine.KindPrompt && requests != "" {
			answerRequest(env, h, ev.Prompt, requests) // even while draining: the fake waits on it
		}
	}
	if printErr != nil {
		return sessionID, exit, printErr
	}
	env.Out.SetToken(token)
	return sessionID, exit, exitRefusal(exit)
}

// exitRefusal is nil for a turn that exited 0 by itself, else engine-exit saying how it ended.
func exitRefusal(exit *engine.ExitEvent) error {
	switch {
	case exit == nil:
		return control.NewError(CodeEngineExit, "the turn ended without an exit event")
	case exit.Interrupted:
		return control.NewError(CodeEngineExit, "the turn was interrupted")
	case exit.Code == nil:
		return control.NewError(CodeEngineExit, "the engine exited without a code")
	case *exit.Code != 0:
		return control.NewError(CodeEngineExit, fmt.Sprintf("the engine exited %d", *exit.Code))
	}
	return nil
}

// engineRecord is `broodforge engine record --credential <id> --cwd <dir> --out <fixture
// dir> (--prompt <text> | --answer <tool_use_id>:allow|deny [--reason <text>]) [--model <model>]
// [--mode prompt|policy|auto] [--requests allow|deny]`: engine run --real that also writes the engine's raw stdout as the
// directory's next turn-<n>.jsonl, resuming the previous turn's session, and records the turn
// in fixture.json, under --mode prompt with the requests policy (05-debug-interface.md); --answer
// records the resume of the previous turn's deferred call, so it needs a previous turn. It
// refuses an --out inside the instance root (a session tree, a config directory) or inside --cwd.
var engineRecord = Command{
	Name:  "engine record",
	Class: ClassStandalone,
	Flags: []Flag{
		{Name: "credential", Kind: FlagString, Value: "<id>", Usage: "the credential the engine runs as"},
		{Name: "cwd", Kind: FlagString, Value: "<dir>", Usage: "the working directory"},
		{Name: "out", Kind: FlagString, Value: "<fixture dir>", Usage: "the fixture directory to add a turn to"},
		{Name: "prompt", Kind: FlagString, Value: "<text>", Usage: "the turn's input"},
		{Name: "model", Kind: FlagString, Value: "<model>", Usage: "the engine's --model (default: the account's)"},
		modeFlag,
		requestsFlag,
		answerFlag,
		reasonFlag,
	},
	Expect: true,
	Local:  runEngineRecord,
}

// modeFlag is the session mode a standalone real turn runs in: BROODFORGE_MODE, and
// --permission-mode auto for auto (ADR 0020 as amended 2026-11-01). Default: the instance's
// defaults.mode.
var modeFlag = Flag{Name: "mode", Kind: FlagString, Value: "<prompt|policy|auto>", Usage: "--real: the session mode (default: defaults.mode)"}

func init() { commands = append(commands, &engineRecord) }

// Error codes of the real engine commands.
const (
	CodeBadCwd       = "bad-cwd"        // --cwd is not a directory
	CodeBadOut       = "bad-out"        // the fixture directory cannot take a new turn
	CodeOutInSession = "out-in-session" // engine record's --out is inside the instance root or --cwd
	CodeBadWorkflow  = "bad-workflow"   // engine run's --workflow is not a directory
)

// standaloneSession is the session id the hook sees in a standalone run (engine run --real,
// engine record); its home, and so the hook log, is <root>/sessions/engine-run/home, the
// tree is --cwd and the canonical is the tree.
const standaloneSession = "engine-run"

func runEngineRecord(ctx context.Context, env *Env) error {
	out := env.String("out")
	if out == "" {
		return &usageError{"engine record: --out <fixture dir> is required"}
	}
	return runEngineReal(ctx, env, "engine record", &out)
}

// runEngineReal is one real turn: the credential from the store (an API key's value through
// the key file), the backend from the instance config, the turn streamed as engine run
// --fake streams it. out, when set, is engine record's fixture directory.
func runEngineReal(_ context.Context, env *Env, name string, out *string) error {
	credID, cwd, prompt := env.String("credential"), env.String("cwd"), env.String("prompt")
	answer, reason, err := answerOf(env, name)
	if err != nil {
		return err
	}
	switch {
	case credID == "":
		return &usageError{name + ": --credential <id> is required"}
	case cwd == "":
		return &usageError{name + ": --cwd <dir> is required"}
	case prompt == "" && answer == nil:
		return &usageError{name + ": --prompt <text> is required"}
	case env.Root == "":
		return &usageError{name + ": no instance root: pass --root <path> or set " + RootEnv}
	case answer != nil && out == nil && env.String("resume") == "":
		return control.NewError(control.CodeBadArgs, name+": --answer needs --resume <engine session id>: the answer resumes a deferred call")
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return control.NewError(CodeBadCwd, err.Error())
	}
	if st, err := os.Stat(absCwd); err != nil || !st.IsDir() {
		return control.NewError(CodeBadCwd, filepath.ToSlash(cwd)+" is not a directory")
	}
	if out != nil {
		// Refused before anything is read or spawned.
		if err := checkOut(env.Root, absCwd, *out); err != nil {
			return err
		}
	}
	c, err := config.Load(env.Root, credentialExists)
	if err != nil {
		return configFailure(env, err)
	}
	mode := env.String("mode")
	switch mode {
	case "":
		mode = c.Instance.Defaults.Mode
	case config.ModePrompt, config.ModePolicy, config.ModeAuto:
	default:
		return &usageError{name + ": --mode " + mode + ": want prompt, policy or auto"}
	}
	requests, err := requestsPolicy(env, name, mode == config.ModePrompt, "the mode is "+mode+", and only a prompt-mode process raises requests")
	if err != nil {
		return err
	}
	if mode != config.ModePrompt {
		requests = ""
	}
	req, secret, err := standaloneRequest(env.Root, c, credID, absCwd, mode)
	if err != nil {
		return err
	}
	if out == nil && env.String("workflow") != "" {
		// The chain without a session (ADR 0025 rule 2): the hook reads the copy from
		// BROODFORGE_WORKFLOW; the spawn exports CLAUDE_CODE_GIT_BASH_PATH from
		// engine.gitBashPath, the shell the children run under on the PC.
		wf, err := filepath.Abs(env.String("workflow"))
		if err != nil {
			return control.NewError(CodeBadWorkflow, err.Error())
		}
		if st, err := os.Stat(wf); err != nil || !st.IsDir() {
			return control.NewError(CodeBadWorkflow, filepath.ToSlash(env.String("workflow"))+" is not a directory")
		}
		req.HookEnv = append(req.HookEnv, engine.HookVar{Name: hook.EnvWorkflow, Value: filepath.ToSlash(wf)})
	}
	req.Input, req.EngineSessionID, req.Model = prompt, env.String("resume"), env.String("model")
	if answer != nil {
		// As a session's resume (session.startProcessLocked): the answer in the request, so the
		// adapter writes no stdin line, and a deny's text as the hook's variable.
		req.Answer = answer
		if reason != "" {
			req.HookEnv = append(req.HookEnv, engine.HookVar{Name: hook.EnvAnswerReason, Value: reason})
		}
	}

	var rec *engine.Recording
	if out != nil {
		if rec, err = openRecording(env.Root, absCwd, *out); err != nil {
			return err
		}
		if answer != nil && rec.Resume() == "" {
			rec.Abort()
			return control.NewError(control.CodeBadArgs, name+": --answer needs a previous turn to resume: "+filepath.ToSlash(*out)+" has none")
		}
		req.EngineSessionID = rec.Resume()
		rec.Requests = requests
	}
	sc := spawnConfig(c)
	backend, err := spawn.New(sc)
	if err != nil {
		if rec != nil {
			rec.Abort()
		}
		return spawnError(err)
	}
	// ExitTimeout: the prompt-mode exit bound after the close at result, the same value as
	// the backend's interrupt timeout (ADR 0029, Amendments 2026-11-07 decision 1).
	cc := &engine.ClaudeCode{Backend: backend, Secret: secret, Stderr: env.Stderr, ExitTimeout: sc.InterruptTimeout}
	if rec != nil {
		cc.Raw = rec.Write
	}
	if err := spawn.JoinKillOnCloseJob(); err != nil {
		fmt.Fprintf(env.Stderr, "warning: %s: this process is in no job of its own: %v\n", name, err)
	}
	h, err := cc.StartTurn(req)
	if err != nil {
		if rec != nil {
			rec.Abort()
		}
		return spawnError(err)
	}
	// Ctrl-C interrupts the turn through the backend: the engine is in its own process group.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	ended := make(chan struct{})
	defer close(ended)
	go func() {
		select {
		case <-sig:
			_ = h.Interrupt()
		case <-ended:
		}
	}()
	sessionID, exit, turnErr := streamTurnExit(env, h, requests)
	if rec != nil {
		if err := rec.Finish(exit, sessionID, c.Instance.Engine.Version); err != nil {
			return control.NewError(CodeBadOut, err.Error())
		}
	}
	return turnErr
}

// standaloneRequest builds a standalone run's turn request from the store: the credential's
// delivery, its user's git identity, the hook variables of a code-reach session in mode (--mode,
// else the instance's default mode), and --permission-mode auto when that mode is auto. An API
// key's value is read through the key file only when the engine asks for it.
func standaloneRequest(root string, c *config.Config, credID, cwd, mode string) (engine.TurnRequest, func(string) ([]byte, error), error) {
	var req engine.TurnRequest
	st, err := openStoreAt(root)
	if err != nil {
		return req, nil, err
	}
	defer st.Close()
	rec, err := credentials.Show(st, credID)
	if err != nil {
		return req, nil, credentialError(err)
	}
	if rec.Status == credentials.StatusRemoved {
		return req, nil, control.NewError(credentials.CodeRemoved, credID+" is removed")
	}
	var secret func(string) ([]byte, error)
	switch rec.Kind {
	case credentials.KindSubscription:
		req.Delivery = engine.Delivery{Kind: engine.DeliveryConfigDir, Path: rec.StoreRef}
	case credentials.KindAPIKey:
		req.Delivery = engine.Delivery{Kind: engine.DeliveryAPIKey, CredentialID: rec.ID}
		keyFile, err := credentials.KeyPath(root, c.Instance.InstanceName, c.Instance.Credentials.KeyFile, os.LookupEnv)
		if err != nil {
			return req, nil, credentialError(err)
		}
		ref := rec.StoreRef
		secret = func(string) ([]byte, error) {
			v, err := credentials.OpenVault(credentials.StorePath(root), keyFile)
			if err != nil {
				return nil, credentialError(err)
			}
			b, err := v.Get(ref)
			if err != nil {
				return nil, credentialError(err)
			}
			return b, nil
		}
	default:
		return req, nil, control.NewError(credentials.CodeBadKind, credID+" has kind "+rec.Kind)
	}
	found := false
	if err := st.Query("SELECT git_name, git_email FROM registration WHERE user_id = ?", []store.Value{store.Text(rec.User)},
		func(row *store.Row) error {
			req.GitIdentity, found = engine.GitIdentity{Name: row.Text(0), Email: row.Text(1)}, true
			return nil
		}); err != nil {
		return req, nil, localError(err)
	}
	if !found {
		return req, nil, control.NewError(credentials.CodeUnknownUser, "the user of "+credID+" is not registered")
	}
	home := filepath.ToSlash(filepath.Join(root, "sessions", standaloneSession, "home"))
	tree := filepath.ToSlash(cwd)
	req.SessionID, req.Cwd = standaloneSession, cwd
	req.HookEnv = []engine.HookVar{
		{Name: hook.EnvSession, Value: standaloneSession},
		{Name: hook.EnvMode, Value: mode},
		{Name: hook.EnvReach, Value: config.ReachCode},
		{Name: hook.EnvTree, Value: tree},
		{Name: hook.EnvCanonical, Value: tree},
		{Name: hook.EnvHookLog, Value: home + "/hook.log"},
	}
	if mode == config.ModeAuto {
		req.PermissionMode = engine.PermissionAuto
	}
	req.Mode = mode
	return req, secret, nil
}

// openStoreAt opens the root's store read-only.
func openStoreAt(root string) (*store.Reader, error) {
	r, err := store.OpenReadOnly(root)
	if err != nil {
		return nil, localError(err)
	}
	return r, nil
}

// openRecording opens out for its next turn, refusing a directory inside the instance root
// (sessions/, a harness directory beside its settings.json) or inside the working tree.
func openRecording(root, cwd, out string) (*engine.Recording, error) {
	if err := checkOut(root, cwd, out); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return nil, control.NewError(CodeBadOut, err.Error())
	}
	rec, err := engine.NewRecording(abs)
	if err != nil {
		return nil, control.NewError(CodeBadOut, err.Error())
	}
	return rec, nil
}

// checkOut refuses an --out inside the instance root or the working tree.
func checkOut(root, cwd, out string) error {
	abs, err := filepath.Abs(out)
	if err != nil {
		return control.NewError(CodeBadOut, err.Error())
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return control.NewError(CodeBadOut, err.Error())
	}
	for _, base := range []string{absRoot, cwd} {
		if within(base, abs) {
			return control.NewError(CodeOutInSession, filepath.ToSlash(out)+" is inside "+filepath.ToSlash(base))
		}
	}
	return nil
}

// within reports whether path is base or below it.
func within(base, path string) bool {
	rel, err := filepath.Rel(base, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// spawnConfig is the backend's view of the instance config.
func spawnConfig(c *config.Config) spawn.Config {
	in := c.Instance
	sc := spawn.Config{Backend: in.Spawn.Backend, EnginePath: in.Engine.Path, EngineVersion: in.Engine.Version,
		GitBashPath: in.Engine.GitBashPath, Install: in.Paths.Install}
	if t := in.Spawn.InterruptTimeoutSeconds; t != nil {
		sc.InterruptTimeout = time.Duration(*t) * time.Second
	}
	return sc
}

// spawnError maps a spawn refusal to the CLI's error line; control refusals pass through.
func spawnError(err error) error {
	var se *spawn.Error
	if errors.As(err, &se) {
		return control.NewError(se.Code, se.Message)
	}
	var ce *control.Error
	if errors.As(err, &ce) {
		return ce
	}
	return err
}
