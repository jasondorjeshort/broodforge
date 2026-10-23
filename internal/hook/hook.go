// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// Error is a failure the hook can see. The engine is told by exit 2 with `hook: <what>` on
// stderr, the one exit it reads as a block (03-hook-contract.md, Failure).
type Error struct{ What string }

func (e *Error) Error() string { return "hook: " + e.What }

// The environment of 03-hook-contract.md (Environment).
const (
	EnvSession      = "BROODFORGE_SESSION"
	EnvMode         = "BROODFORGE_MODE"
	EnvReach        = "BROODFORGE_REACH"
	EnvTree         = "BROODFORGE_TREE"
	EnvCanonical    = "BROODFORGE_CANONICAL"
	EnvAnswer       = "BROODFORGE_ANSWER"
	EnvAnswerReason = "BROODFORGE_ANSWER_REASON"
	EnvSessionAllow = "BROODFORGE_SESSION_ALLOW"
	EnvHookLog      = "BROODFORGE_HOOK_LOG"
)

// MaxInput bounds stdin: a Write's content travels in it.
const MaxInput = 64 << 20

// Settings is the session side of a decision: what the environment carries, or hook try's
// flags.
type Settings struct {
	Mode         policy.Mode
	Reach        policy.Reach
	Tree         string
	Canonical    string
	Answer       *policy.Answer
	SessionAllow []string
}

// Input is the engine's hook input on stdin, the fields the hook reads.
type Input struct {
	ToolName       string
	ToolInput      json.RawMessage // a JSON object; nil or null for none
	ToolUseID      string
	Cwd            string // "" means the tree
	TranscriptPath string
}

// ParseMode, ParseReach and ParseAnswer read the variables' values; try's flags take the same.
func ParseMode(s string) (policy.Mode, error) {
	m, ok := policy.ParseMode(s)
	if !ok {
		return m, &Error{EnvMode + " is " + quoteOrEmpty(s) + ", want policy or prompt"}
	}
	return m, nil
}

func ParseReach(s string) (policy.Reach, error) {
	r, ok := policy.ParseReach(s)
	if !ok {
		return r, &Error{EnvReach + " is " + quoteOrEmpty(s) + ", want read-only or code"}
	}
	return r, nil
}

// ParseAnswer reads `<tool_use_id>:allow|deny`; "" is no answer.
func ParseAnswer(s, reason string) (*policy.Answer, error) {
	if s == "" {
		return nil, nil
	}
	i := strings.LastIndexByte(s, ':')
	if i <= 0 {
		return nil, &Error{EnvAnswer + " is not <tool_use_id>:allow|deny"}
	}
	d, ok := policy.ParseDecision(s[i+1:])
	if !ok {
		return nil, &Error{EnvAnswer + " is not <tool_use_id>:allow|deny"}
	}
	return &policy.Answer{ToolUseID: s[:i], Decision: d, Reason: reason}, nil
}

func quoteOrEmpty(s string) string {
	if s == "" {
		return "empty"
	}
	return fmt.Sprintf("%q", s)
}

// FromEnv reads Settings, the session id and the log path from the environment. Every
// variable but the answer, its reason and the session allow list is required.
func FromEnv(lookup func(string) (string, bool)) (s Settings, session, log string, err error) {
	get := func(name string) (string, error) {
		v, ok := lookup(name)
		if !ok || v == "" {
			return "", &Error{name + " is not set"}
		}
		return v, nil
	}
	if session, err = get(EnvSession); err != nil {
		return
	}
	var mode, reach string
	if mode, err = get(EnvMode); err != nil {
		return
	}
	if s.Mode, err = ParseMode(mode); err != nil {
		return
	}
	if reach, err = get(EnvReach); err != nil {
		return
	}
	if s.Reach, err = ParseReach(reach); err != nil {
		return
	}
	if s.Tree, err = get(EnvTree); err != nil {
		return
	}
	if s.Canonical, err = get(EnvCanonical); err != nil {
		return
	}
	if log, err = get(EnvHookLog); err != nil {
		return
	}
	answer, _ := lookup(EnvAnswer)
	reason, _ := lookup(EnvAnswerReason)
	if s.Answer, err = ParseAnswer(answer, reason); err != nil {
		return
	}
	allow, _ := lookup(EnvSessionAllow)
	s.SessionAllow = policy.SplitList(allow)
	return
}

// ParseInput reads the engine's JSON. tool_name, tool_use_id and transcript_path are
// required; unknown fields are read past, as on the engine stream.
func ParseInput(b []byte) (Input, error) {
	var raw struct {
		ToolName       *string         `json:"tool_name"`
		ToolInput      json.RawMessage `json:"tool_input"`
		ToolUseID      *string         `json:"tool_use_id"`
		Cwd            *string         `json:"cwd"`
		TranscriptPath *string         `json:"transcript_path"`
	}
	if !json.Valid(b) || bytes.TrimSpace(b)[0] != '{' {
		return Input{}, &Error{"stdin is not a JSON object"}
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return Input{}, &Error{"stdin: " + err.Error()}
	}
	in := Input{ToolInput: raw.ToolInput}
	if raw.ToolName == nil || *raw.ToolName == "" {
		return in, &Error{"no tool_name"}
	}
	in.ToolName = *raw.ToolName
	if raw.ToolUseID == nil || *raw.ToolUseID == "" {
		return in, &Error{"no tool_use_id"}
	}
	in.ToolUseID = *raw.ToolUseID
	if raw.Cwd != nil {
		in.Cwd = *raw.Cwd
	}
	if raw.TranscriptPath == nil || *raw.TranscriptPath == "" {
		return in, &Error{"no transcript_path"}
	}
	in.TranscriptPath = *raw.TranscriptPath
	return in, nil
}

// pathKeys are the tool_input fields that hold a path (03-hook-contract.md, Paths).
var pathKeys = []string{"file_path", "path", "notebook_path"}

// Decide is the one code path of pre-tool-use, try and cases: the path checks, then the
// policy decision.
func Decide(in Input, s Settings) (policy.Result, error) {
	cfg, err := configDir(in.TranscriptPath)
	if err != nil {
		return policy.Result{}, err
	}
	tree, err := canonical(s.Tree)
	if err != nil {
		return policy.Result{}, &Error{"cannot resolve " + EnvTree + ": " + errText(err)}
	}
	canon, err := canonical(s.Canonical)
	if err != nil {
		return policy.Result{}, &Error{"cannot resolve " + EnvCanonical + ": " + errText(err)}
	}
	fields := map[string]json.RawMessage{}
	if t := bytes.TrimSpace(in.ToolInput); len(t) > 0 && string(t) != "null" {
		if t[0] != '{' || json.Unmarshal(t, &fields) != nil {
			return policy.Result{}, &Error{"tool_input is not a JSON object"}
		}
	}
	class := policy.ClassOf(in.ToolName)
	type rawPath struct {
		path   string
		search bool
	}
	var raws []rawPath
	search := in.ToolName == "Grep" || in.ToolName == "Glob"
	for _, k := range pathKeys {
		v, ok := fields[k]
		if !ok || string(bytes.TrimSpace(v)) == "null" {
			continue
		}
		var p string
		if json.Unmarshal(v, &p) != nil || p == "" {
			return policy.Result{}, &Error{"tool_input." + k + " is not a path"}
		}
		raws = append(raws, rawPath{p, search})
	}
	ps := newPlaces(tree, canon, cfg)
	// A relative path resolves against the hook input's cwd, as the engine resolves it, else
	// the tree (03-hook-contract.md, Paths).
	base := tree
	if in.Cwd != "" {
		c, err := expand(in.Cwd, ps.home)
		if err == nil {
			err = checkSpelling(c)
		}
		if err == nil {
			c, err = canonical(join(c, tree))
		}
		if err != nil {
			return policy.Result{}, &Error{"cwd is " + errText(err)}
		}
		base = c
	}
	switch in.ToolName {
	case "Glob", "Grep", "LS":
		root := base // a search with no path means cwd, else the tree
		if len(raws) > 0 {
			exp, err := expand(raws[0].path, ps.home)
			if err != nil {
				return policy.Result{}, &Error{"tool_input path is " + err.Error()}
			}
			root = join(exp, base)
		} else {
			raws = append(raws, rawPath{root, search})
		}
		// Glob's pattern is a path up to its first metacharacter (03-hook-contract.md, Paths).
		if v, ok := fields["pattern"]; ok && in.ToolName == "Glob" {
			var pattern string
			if json.Unmarshal(v, &pattern) != nil {
				return policy.Result{}, &Error{"tool_input.pattern is not a string"}
			}
			exp, err := expand(pattern, ps.home)
			if err != nil {
				return policy.Result{}, &Error{"tool_input.pattern is " + err.Error()}
			}
			dirs, err := globDirs(exp)
			if err != nil {
				return policy.Result{}, &Error{"tool_input.pattern is " + err.Error()}
			}
			for _, dir := range dirs {
				if dir != "" {
					raws = append(raws, rawPath{join(dir, root), true})
				}
			}
		}
	default:
		if len(raws) == 0 && (class == policy.ClassRead || class == policy.ClassEdit) {
			return policy.Result{}, &Error{in.ToolName + " has no path in tool_input"}
		}
	}
	paths := make([]policy.Path, 0, len(raws))
	for _, raw := range raws {
		f, err := ps.fact(raw.path, base, raw.search)
		if err != nil {
			return policy.Result{}, err
		}
		paths = append(paths, f)
	}
	return policy.Decide(policy.Request{Tool: in.ToolName, ToolUseID: in.ToolUseID, Mode: s.Mode,
		Reach: s.Reach, Paths: paths, Answer: s.Answer, SessionAllow: s.SessionAllow}), nil
}

// runsChain is whether the repository's PreToolUse children run after the bot's own result:
// code reach only, a decision that is not deny, and a copy named (ADR 0025 rule 2).
func runsChain(r policy.Result, s Settings, workflow string) bool {
	return r.Decision != policy.Deny && s.Reach == policy.ReachCode && workflow != ""
}

// Output is the engine's hook output.
type Output struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// HookSpecificOutput is the PreToolUse decision object.
type HookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason"`
}

// Encode is the output line for a result, newline included.
func Encode(r policy.Result) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(Output{HookSpecificOutput{"PreToolUse", r.Decision.String(), r.Reason}})
	return b.Bytes(), err
}

// logLine is one hook.log line: names and decisions, never tool input.
type logLine struct {
	Time      string `json:"time"`
	Session   string `json:"session"`
	Tool      string `json:"tool"`
	ToolUseID string `json:"toolUseId"`
	Cell      string `json:"cell"`
	Decision  string `json:"decision"`
	// Trace is a child deny's `denied: refused by the project's hook <name>: <reason>`.
	Trace string `json:"trace,omitempty"`
}

// childLogLine is one hook.log line of the chain: the call's names, then the child's line.
type childLogLine struct {
	Time      string `json:"time"`
	Session   string `json:"session"`
	Tool      string `json:"tool"`
	ToolUseID string `json:"toolUseId"`
	ChildLine
}

func appendLog[T logLine | childLogLine](path string, l T) error {
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}
	if path == "/dev/null" {
		path = os.DevNull // the same name on every host, so 05's example line runs on Windows too
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return &Error{"cannot open " + EnvHookLog + ": " + errText(err)}
	}
	_, werr := f.Write(append(b, '\n'))
	cerr := f.Close()
	if werr != nil || cerr != nil {
		return &Error{"cannot append to " + EnvHookLog}
	}
	return nil
}

// PreToolUse is `broodforge hook pre-tool-use` as the engine runs it: the engine's JSON on
// stdin, the session in the environment, the decision on stdout, exit 0. Every error it can
// see, a panic included, is `hook: <what>` on stderr and exit 2.
//
// environ is the hook's own environment, handed to the chain's children; lookup reads it.
func PreToolUse(stdin io.Reader, stdout, stderr io.Writer, lookup func(string) (string, bool), environ []string, now func() time.Time) (code int) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintf(stderr, "hook: internal error: %v\n", p)
			code = 2
		}
	}()
	if err := preToolUse(stdin, stdout, lookup, environ, now); err != nil {
		msg := err.Error()
		if _, ok := err.(*Error); !ok {
			msg = "hook: " + msg
		}
		fmt.Fprintln(stderr, msg)
		return 2
	}
	return 0
}

func preToolUse(stdin io.Reader, stdout io.Writer, lookup func(string) (string, bool), environ []string, now func() time.Time) error {
	s, session, log, err := FromEnv(lookup)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(io.LimitReader(stdin, MaxInput+1))
	if err != nil {
		return &Error{"cannot read stdin"}
	}
	if len(b) > MaxInput {
		return &Error{"stdin is over 64 MiB"}
	}
	in, err := ParseInput(b)
	if err != nil {
		return err
	}
	r, err := Decide(in, s)
	if err != nil {
		return err
	}
	stamp := func() string { return now().UTC().Format(time.RFC3339) }
	var logErr error
	workflow, _ := lookup(EnvWorkflow)
	trace := ""
	if runsChain(r, s, workflow) {
		c := Chain{Workflow: workflow, Tree: s.Tree, Environ: environ, now: now, Log: func(l ChildLine) {
			if logErr == nil {
				logErr = appendLog(log, childLogLine{Time: stamp(), Session: session, Tool: in.ToolName, ToolUseID: in.ToolUseID, ChildLine: l})
			}
		}}
		deny, why, err := c.Pre(in.ToolName, b)
		if err != nil {
			return err
		}
		if logErr != nil {
			return logErr
		}
		if deny != nil {
			deny.Cell = r.Cell
			r, trace = *deny, why
		}
	}
	out, err := Encode(r)
	if err != nil {
		return err
	}
	if err := appendLog(log, logLine{Time: stamp(), Session: session, Tool: in.ToolName,
		ToolUseID: in.ToolUseID, Cell: r.Cell.String(), Decision: r.Decision.String(), Trace: trace}); err != nil {
		return err
	}
	if _, err := stdout.Write(out); err != nil {
		return &Error{"cannot write stdout"}
	}
	return nil
}
