// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// The session commands of 05-debug-interface-session.md (Session lifecycle without Discord)
// that P10 builds: start, send, answer, mode, interrupt, end, invite, show, tail, list and
// reach-report. outbox and transcript are P11's.

var tailColumns = []string{"seq", "time", "type", "summary"}

// actingFlags are --as and --roles, which every command acting for a person takes.
var actingFlags = []Flag{
	{Name: "as", Kind: FlagString, Value: "<user id>", Usage: "act for this registered user"},
	{Name: "roles", Kind: FlagString, Value: "<id,...>", Usage: "the role ids the user holds; given, the channel's list is checked"},
}

// sessionStart is `broodforge session start --channel <id> --as <user id> [--roles <ids>]
// [--credential <id>] [--mode prompt|policy] [--reach read-only|code] [--fixture <dir>]
// [--idle-timeout-seconds <n>] [--prompt-timeout-seconds <n>]`: prints the session id.
var sessionStart = Command{
	Name:   "session start",
	Class:  ClassWrite,
	Expect: true,
	Flags: append([]Flag{
		{Name: "channel", Kind: FlagString, Value: "<id>", Usage: "the configured channel"},
		{Name: "credential", Kind: FlagString, Value: "<id>", Usage: "the credential (default: the driver's)"},
		{Name: "mode", Kind: FlagString, Value: "prompt|policy", Usage: "the permission mode (default: the channel's)"},
		{Name: "reach", Kind: FlagString, Value: "read-only|code", Usage: "override the channel's reach (dry-run only)"},
		{Name: "fixture", Kind: FlagString, Value: "<dir>", Usage: "the fake's fixture: a name under engine.fakeFixtures or a path (dry-run only)"},
		{Name: "idle-timeout-seconds", Kind: FlagInt, Value: "<n>", Usage: "override the idle timeout (dry-run only)"},
		{Name: "prompt-timeout-seconds", Kind: FlagInt, Value: "<n>", Usage: "override the prompt timeout (dry-run only)"},
	}, actingFlags...),
	Columns: []string{"id"},
}

// sessionSend is `broodforge session send <id> --as <user id> [--roles <ids>] [--wait] [--tail]
// "<text>"` (`-` reads stdin): `turn  <n>  started|queued`; --wait returns at the turn's end
// (or a wait for an answer, P10 half b) and prints `turn-ended  <n>  <result>` instead; --tail
// streams the turn's events meanwhile.
var sessionSend = Command{
	Name:   "session send",
	Class:  ClassWrite,
	Expect: true,
	Flags: append([]Flag{
		{Name: "wait", Kind: FlagBool, Usage: "return when the turn ends"},
		{Name: "tail", Kind: FlagBool, Usage: "stream the turn's events until it ends"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
		{Name: "text", Kind: FlagString, Value: "<text>"},
	}, actingFlags...),
	Columns:      []string{"kind", "turn", "detail"},
	EventColumns: tailColumns,
}

// sessionList is `broodforge session list [--state <state>]`.
var sessionList = Command{
	Name:    "session list",
	Class:   ClassRead,
	List:    true,
	Flags:   []Flag{{Name: "state", Kind: FlagString, Value: "<state>", Usage: "only sessions in this state"}},
	Columns: session.ListColumns,
	Local:   runSessionList,
}

// sessionShow is `broodforge session show <id>`: labelled lines (renderSessionShow), --json
// the record.
var sessionShow = Command{
	Name:  "session show",
	Class: ClassRead,
	Flags: []Flag{{Name: "id", Kind: FlagString, Value: "<session id>"}},
	Local: runSessionShow,
}

// sessionTail is `broodforge session tail <id> [--follow] [--since <seq>] [--limit <n>|--all]`:
// `<seq>  <time>  <type>  <summary>` per event. The events are the bot's, in memory, so
// without a bot it is not-running.
var sessionTail = Command{
	Name:  "session tail",
	Class: ClassRead,
	List:  true,
	Flags: []Flag{
		{Name: "follow", Kind: FlagBool, Usage: "stream new events until the session closes"},
		{Name: "since", Kind: FlagInt, Value: "<seq>", Usage: "only events after this seq"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
	},
	Columns:      tailColumns,
	EventColumns: tailColumns,
}

func init() {
	commands = append(commands, &sessionStart, &sessionSend, &sessionList, &sessionShow, &sessionTail)
	positionals[&sessionSend] = []string{"id", "text"}
	positionals[&sessionShow] = []string{"id"}
	positionals[&sessionTail] = []string{"id"}
	extraArgs[&sessionStart] = absoluteFixture
	extraArgs[&sessionSend] = sendText
	renderers[&sessionShow] = renderSessionShow
}

// absoluteFixture sends a --fixture path as an absolute path, since the bot's working directory
// is not the client's; a bare name stays a name under engine.fakeFixtures.
func absoluteFixture(env *Env) (map[string]string, error) {
	f := env.String("fixture")
	if f == "" || !strings.ContainsAny(f, `/\`) {
		return nil, nil
	}
	abs, err := filepath.Abs(f)
	if err != nil {
		return nil, &usageError{"session start: --fixture " + f + ": " + err.Error()}
	}
	return map[string]string{"fixture": filepath.ToSlash(abs)}, nil
}

// sendText reads the message from stdin when it is `-`.
func sendText(env *Env) (map[string]string, error) {
	if env.String("text") != "-" {
		return nil, nil
	}
	b, err := io.ReadAll(io.LimitReader(env.Stdin, 1<<20+1))
	if err != nil {
		return nil, &usageError{"session send: cannot read stdin: " + err.Error()}
	}
	if len(b) > 1<<20 {
		return nil, &usageError{"session send: a message from stdin is at most 1 MiB"}
	}
	return map[string]string{"text": strings.TrimRight(string(b), "\r\n")}, nil
}

// encodeSession is json.Marshal without HTML escaping, so a trace line's `->` prints as written.
func encodeSession[T session.Started | session.Detail | []session.ListRow | session.Registration |
	[]session.Registration | session.Event | []session.Event | sendLine](v T) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}

// sessionError maps the session service's and its dependencies' refusals to the wire error.
func sessionError(err error) error {
	var se *session.Error
	if errors.As(err, &se) {
		return control.NewError(se.Code, se.Message)
	}
	if e, ok := credentials.IsRefusal(err); ok {
		return control.NewError(e.Code, e.Message)
	}
	var ge *git.Error
	if errors.As(err, &ge) {
		return control.NewError(ge.Code, ge.Message)
	}
	return localError(err)
}

// sendLine is session send's result: `turn  <n>  started|queued`, or with --wait the outcome.
type sendLine struct {
	Kind   string `json:"kind"`
	Turn   int    `json:"turn"`
	Detail string `json:"detail"`
}

func runSessionList(_ context.Context, env *Env) error {
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	rows, err := session.List(r, env.String("state"))
	if err != nil {
		return err
	}
	out, err := encodeSession(rows)
	if err != nil {
		return err
	}
	return env.Out.Result(out, session.ListColumns)
}

func runSessionShow(_ context.Context, env *Env) error {
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	d, err := session.Show(r, env.String("id"))
	if err != nil {
		return sessionError(err)
	}
	out, err := encodeSession(d)
	if err != nil {
		return err
	}
	return renderSessionShow(env, out)
}

// renderSessionShow prints `session show`: `session: <id>  <state>  <channel>  <project>
// <driver>  <mode>  <reach>  <harness>`, `credential: <id>`, `engine-session: <id or ->`,
// `turns: <n>`, one `participants: <user id>  driver|contributor  <joined at>` per participant,
// `worktree: <path>  <branch>  <state>` (or -), `pending: <tool use id>  <tool>  <target>  since
// <time>  until <deadline>` (or -).
func renderSessionShow(env *Env, raw json.RawMessage) error {
	var d session.Detail
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	env.Out.SetToken(d.State)
	if env.JSON {
		return env.Out.Record(raw, nil)
	}
	lines := []string{
		"session: " + strings.Join([]string{d.ID, d.State, d.Channel, d.Project, d.Driver, d.Mode, d.Reach, d.Harness}, "  "),
		"credential: " + d.Credential,
		"engine-session: " + orDash(deref(d.EngineSessionID)),
		fmt.Sprintf("turns: %d", d.Turns),
	}
	for _, p := range d.Participants {
		lines = append(lines, "participants: "+p.User+"  "+p.Role+"  "+p.JoinedAt)
	}
	if w := d.Worktree; w != nil {
		lines = append(lines, "worktree: "+w.Path+"  "+w.Branch+"  "+w.State)
	} else {
		lines = append(lines, "worktree: -")
	}
	if p := d.Pending; p != nil {
		lines = append(lines, "pending: "+p.ToolUseID+"  "+p.Tool+"  "+orDash(p.Target)+"  since "+p.Since+"  until "+p.Until)
	} else {
		lines = append(lines, "pending: -")
	}
	for _, l := range lines {
		if _, err := fmt.Fprintln(env.Out.w, l); err != nil {
			return err
		}
	}
	return nil
}

// SessionHandlers registers the session service's commands on a control registry: `serve`
// and the tests call it. register and register.set-default ride along.
func SessionHandlers(svc *session.Service) func(*control.Registry) {
	return func(reg *control.Registry) {
		registerHandlers(svc, reg)
		reg.Handle("session.start", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct {
				Channel, As, Roles, Credential, Mode, Reach, Fixture string
				IdleTimeoutSeconds, PromptTimeoutSeconds             int
			}
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			st, err := svc.Start(ctx, session.StartRequest{Channel: a.Channel, As: a.As, Roles: a.Roles, Credential: a.Credential,
				Mode: a.Mode, Reach: a.Reach, Fixture: a.Fixture, IdleTimeoutSeconds: a.IdleTimeoutSeconds,
				PromptTimeoutSeconds: a.PromptTimeoutSeconds})
			if err != nil {
				return nil, sessionError(err)
			}
			return encodeSession(st)
		})
		reg.Handle("session.send", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct {
				ID, Text, As, Roles string
				Wait, Tail          bool
			}
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			sub, err := svc.Submit(ctx, session.SendRequest{Session: a.ID, As: a.As, Roles: a.Roles, Text: a.Text})
			if err != nil {
				return nil, sessionError(err)
			}
			return followTurn(ctx, call, svc, a.ID, sub, a.Wait, a.Tail)
		})
		reg.Handle("session.answer", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct {
				ID, Decision, ToolUseID, Reason, As, Roles string
				Wait, Tail                                 bool
			}
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			sub, err := svc.Answer(ctx, session.AnswerRequest{ActRequest: session.ActRequest{Session: a.ID, As: a.As, Roles: a.Roles},
				ToolUseID: a.ToolUseID, Decision: a.Decision, Reason: a.Reason})
			if err != nil {
				return nil, sessionError(err)
			}
			return followTurn(ctx, call, svc, a.ID, sub, a.Wait, a.Tail)
		})
		reg.Handle("session.interrupt", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct{ ID, As, Roles string }
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			ev, err := svc.Interrupt(ctx, session.ActRequest{Session: a.ID, As: a.As, Roles: a.Roles})
			if err != nil {
				return nil, sessionError(err)
			}
			return encodeSession(sendLine{Kind: string(ev.Type), Turn: ev.Turn, Detail: ev.Result})
		})
		reg.Handle("session.mode", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct{ ID, Mode, As, Roles string }
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			ev, err := svc.SetMode(session.ModeRequest{ActRequest: session.ActRequest{Session: a.ID, As: a.As, Roles: a.Roles}, Mode: a.Mode})
			if err != nil {
				return nil, sessionError(err)
			}
			return encodeSession(sendLine{Kind: string(ev.Type), Turn: ev.Turn, Detail: ev.Result})
		})
		reg.Handle("session.invite", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct{ ID, User, As, Roles string }
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			r, err := svc.Invite(session.InviteRequest{ActRequest: session.ActRequest{Session: a.ID, As: a.As, Roles: a.Roles}, User: a.User})
			if err != nil {
				return nil, sessionError(err)
			}
			return encodeSession(r)
		})
		reg.Handle("session.end", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct {
				ID, As, Roles string
				NoClosingTurn bool
			}
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			out, err := svc.End(ctx, session.EndRequest{ActRequest: session.ActRequest{Session: a.ID, As: a.As, Roles: a.Roles},
				NoClosingTurn: a.NoClosingTurn})
			if err != nil {
				var ge *git.Error
				if errors.As(err, &ge) && ge.Code == git.CodeRefusedAttribution {
					if eerr := emitCommits(call, ge.Commits); eerr != nil {
						return nil, eerr
					}
				}
				return nil, sessionError(err)
			}
			return encodeWorktree(out)
		})
		reg.Handle("session.show", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct{ ID string }
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			d, err := svc.Show(a.ID)
			if err != nil {
				return nil, sessionError(err)
			}
			return encodeSession(d)
		})
		reg.Handle("session.tail", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct {
				ID     string
				Follow bool
				Since  int64
			}
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			if !a.Follow {
				evs, err := svc.Events(a.ID, a.Since)
				if err != nil {
					return nil, sessionError(err)
				}
				if evs == nil {
					evs = []session.Event{}
				}
				return encodeSession(evs)
			}
			err := svc.Follow(ctx, a.ID, a.Since, func(ev session.Event) (bool, error) {
				b, err := encodeSession(ev)
				if err != nil {
					return true, err
				}
				return false, call.Emit(b)
			})
			if err != nil && !errors.Is(err, context.Canceled) {
				return nil, sessionError(err)
			}
			return nil, nil
		})
	}
}

// followTurn is the result of a send or an answer: `turn  <n>  started|queued|resumed`, or with
// --wait the outcome, `turn-ended  <n>  <result>` or `waiting-for-answer  <n>  <call>`; --tail
// streams the turn's events meanwhile.
func followTurn(ctx context.Context, call *control.Call, svc *session.Service, id string, sub session.Submitted, wait, tail bool) (json.RawMessage, error) {
	line := sendLine{Kind: "turn", Turn: sub.Turn, Detail: sub.State}
	if !wait && !tail {
		return encodeSession(line)
	}
	var outcome *sendLine
	err := svc.Follow(ctx, id, sub.Seq, func(ev session.Event) (bool, error) {
		if tail {
			b, err := encodeSession(ev)
			if err != nil {
				return true, err
			}
			if err := call.Emit(b); err != nil {
				return true, err
			}
		}
		switch {
		case ev.Type == session.EventTurnEnded && ev.Turn == sub.Turn:
			outcome = &sendLine{Kind: string(ev.Type), Turn: ev.Turn, Detail: ev.Result}
		case ev.Type == session.EventWaitingForAnswer && ev.Turn == sub.Turn:
			outcome = &sendLine{Kind: string(ev.Type), Turn: ev.Turn, Detail: ev.Summary}
		case ev.Type == session.EventSessionInterrupted || ev.Type == session.EventSessionEnded:
			outcome = &sendLine{Kind: string(ev.Type), Turn: sub.Turn, Detail: ev.Summary}
		}
		return outcome != nil, nil
	})
	if err != nil {
		return nil, sessionError(err)
	}
	if outcome == nil {
		return nil, control.NewError(session.CodeSessionClosed, id+" closed before turn "+fmt.Sprint(sub.Turn)+" ended")
	}
	if wait {
		return encodeSession(*outcome)
	}
	return encodeSession(line)
}

// sessionAnswer is `broodforge session answer <id> allow|deny|allow-session [--tool-use-id
// <id>] [--reason <text>] [--as <user id>] [--roles <ids>] [--wait] [--tail]`: the pending
// call's answer (the channel's `answer` list); --tool-use-id must name the pending call when
// given. Prints `turn  <n>  resumed`, or with --wait the outcome as `session send` does.
var sessionAnswer = Command{
	Name:   "session answer",
	Class:  ClassWrite,
	Expect: true,
	Flags: append([]Flag{
		{Name: "tool-use-id", Kind: FlagString, Value: "<id>", Usage: "the call answered (default: the pending one)"},
		{Name: "reason", Kind: FlagString, Value: "<text>", Usage: "a deny's reason, handed to the agent"},
		{Name: "wait", Kind: FlagBool, Usage: "return when the turn ends or waits again"},
		{Name: "tail", Kind: FlagBool, Usage: "stream the turn's events until then"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
		{Name: "decision", Kind: FlagString, Value: "allow|deny|allow-session"},
	}, actingFlags...),
	Columns:      []string{"kind", "turn", "detail"},
	EventColumns: tailColumns,
}

// sessionInterrupt is `broodforge session interrupt <id>`: stops the running turn (the `answer`
// list) and prints `turn-ended  <n>  interrupted|completed` once it has ended.
var sessionInterrupt = Command{
	Name:    "session interrupt",
	Class:   ClassWrite,
	Expect:  true,
	Flags:   append([]Flag{{Name: "id", Kind: FlagString, Value: "<session id>"}}, actingFlags...),
	Columns: []string{"kind", "turn", "detail"},
}

// sessionMode is `broodforge session mode <id> prompt|policy`: the driver's switch, from the next
// turn; prints `mode-changed  <next turn>  <mode>`.
var sessionMode = Command{
	Name:  "session mode",
	Class: ClassWrite,
	Flags: append([]Flag{
		{Name: "id", Kind: FlagString, Value: "<session id>"},
		{Name: "mode", Kind: FlagString, Value: "prompt|policy"},
	}, actingFlags...),
	Columns: []string{"kind", "turn", "detail"},
}

// sessionInvite is `broodforge session invite <id> --user <user id>`: the driver's; prints the
// invited registration.
var sessionInvite = Command{
	Name:  "session invite",
	Class: ClassWrite,
	Flags: append([]Flag{
		{Name: "user", Kind: FlagString, Value: "<user id>", Usage: "the registered user invited"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
	}, actingFlags...),
	Columns: session.RegistrationColumns,
}

// sessionEnd is `broodforge session end <id> [--no-closing-turn]`: the end-of-session sequence
// (the `answer` list); prints the handoff's line as `worktree handoff` does, its failing commits
// first on a refused attribution, after which the session stays open.
var sessionEnd = Command{
	Name:   "session end",
	Class:  ClassWrite,
	Expect: true,
	Flags: append([]Flag{
		{Name: "no-closing-turn", Kind: FlagBool, Usage: "skip the closing turn that writes the pull request"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
	}, actingFlags...),
	EventColumns: commitColumns,
	Columns:      []string{"outcome", "remote", "refspec", "commits", "url", "request"},
}

// sessionReachReport is `broodforge session reach-report <id>`: the policy table's column for
// the session's reach, one `<tools>  <cell under the tree or canonical>  <cell elsewhere>` line
// per row, then `binds: -` (none until the linux-sudo-bwrap backend, P14); --json both.
var sessionReachReport = Command{
	Name:   "session reach-report",
	Expect: true,
	Class:  ClassRead,
	Flags:  []Flag{{Name: "id", Kind: FlagString, Value: "<session id>"}},
	Local:  runReachReport,
}

func init() {
	commands = append(commands, &sessionAnswer, &sessionInterrupt, &sessionMode, &sessionInvite, &sessionEnd, &sessionReachReport)
	positionals[&sessionAnswer] = []string{"id", "decision"}
	positionals[&sessionInterrupt] = []string{"id"}
	positionals[&sessionMode] = []string{"id", "mode"}
	positionals[&sessionInvite] = []string{"id"}
	required[&sessionInvite] = []string{"user"}
	positionals[&sessionEnd] = []string{"id"}
	positionals[&sessionReachReport] = []string{"id"}
}

func runReachReport(_ context.Context, env *Env) error {
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	rep, err := session.Reach(r, env.String("id"))
	if err != nil {
		return sessionError(err)
	}
	if env.JSON {
		b, err := json.Marshal(rep)
		if err != nil {
			return err
		}
		return env.Out.Record(b, nil)
	}
	env.Out.SetToken(rep.Reach)
	for _, row := range rep.Policy {
		if _, err := fmt.Fprintln(env.Out.w, strings.Join(row.Tools, ",")+"  "+row.InTree+"  "+row.Elsewhere); err != nil {
			return err
		}
	}
	binds := "binds: -"
	for _, b := range rep.Binds {
		binds = "binds: " + b.Source + "  " + b.Target + "  " + b.Mode
		if _, err := fmt.Fprintln(env.Out.w, binds); err != nil {
			return err
		}
	}
	if len(rep.Binds) == 0 {
		_, err = fmt.Fprintln(env.Out.w, binds)
	}
	return err
}
