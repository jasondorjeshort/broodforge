// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/hook"
)

// Error codes of hook try and hook cases.
const (
	CodeHook        = "hook"         // the hook's own error, `hook: <what>` without the prefix
	CodeBadCases    = "bad-cases"    // a case file line that is not a case
	CodeCasesFailed = "cases-failed" // one or more cases did not get their expect
)

// hookPreToolUse is `broodforge hook pre-tool-use`, standalone, as the engine runs it
// (03-hook-contract.md): the engine's JSON on stdin and the BROODFORGE_* variables in the
// environment; the decision object on stdout, exit 0; any error `hook: <what>` on stderr and
// exit 2, the one exit the engine reads as a block.
var hookPreToolUse = Command{
	Name:  "hook pre-tool-use",
	Class: ClassStandalone,
	Local: runHookPreToolUse,
}

// hookPostToolUse is `broodforge hook post-tool-use`, standalone, as the engine runs it (ADR
// 0025 rule 2): the repository's PostToolUse children from BROODFORGE_WORKFLOW in code reach,
// their additionalContext on stdout as this hook's own; exit 0 always, an error a `hook:` line
// on stderr.
var hookPostToolUse = Command{
	Name:  "hook post-tool-use",
	Class: ClassStandalone,
	Local: runHookPostToolUse,
}

// hookTry is the same code path with flags: `broodforge hook try --tool <name> --input <json>
// [--tool-use-id <id>] [--mode prompt|policy] [--reach read-only|code] [--tree <path>]
// [--canonical <path>] --transcript-path <path> [--answer <id>:allow|deny] [--answer-reason
// <text>] [--session-allow <names>] [--workflow <dir>] [--expect allow|deny|defer|hook]`,
// printing `<decision>  <reason>`; --workflow runs the copy's PreToolUse children as the hook
// does under BROODFORGE_WORKFLOW. A hook error is `error: hook: <what>`, exit 1. Nothing is logged.
var hookTry = Command{
	Name:   "hook try",
	Class:  ClassStandalone,
	Expect: true,
	Flags: []Flag{
		{Name: "tool", Kind: FlagString, Value: "<name>", Usage: "tool_name"},
		{Name: "input", Kind: FlagString, Default: "{}", Value: "<json>", Usage: "tool_input, a JSON object"},
		{Name: "tool-use-id", Kind: FlagString, Default: hook.DefaultToolUseID, Value: "<id>", Usage: "tool_use_id"},
		{Name: "cwd", Kind: FlagString, Value: "<path>", Usage: "cwd (default: the tree)"},
		{Name: "mode", Kind: FlagString, Default: hook.DefaultMode, Value: "prompt|policy", Usage: "BROODFORGE_MODE"},
		{Name: "reach", Kind: FlagString, Default: hook.DefaultReach, Value: "read-only|code", Usage: "BROODFORGE_REACH"},
		{Name: "tree", Kind: FlagString, Default: hook.DefaultTree, Value: "<path>", Usage: "BROODFORGE_TREE"},
		{Name: "canonical", Kind: FlagString, Value: "<path>", Usage: "BROODFORGE_CANONICAL (default: the tree)"},
		{Name: "transcript-path", Kind: FlagString, Value: "<path>", Usage: "transcript_path, which names the config directory"},
		{Name: "answer", Kind: FlagString, Value: "<id>:allow|deny", Usage: "BROODFORGE_ANSWER"},
		{Name: "answer-reason", Kind: FlagString, Value: "<text>", Usage: "BROODFORGE_ANSWER_REASON"},
		{Name: "session-allow", Kind: FlagString, Value: "<names>", Usage: "BROODFORGE_SESSION_ALLOW"},
		{Name: "workflow", Kind: FlagString, Value: "<dir>", Usage: "BROODFORGE_WORKFLOW: run this copy's PreToolUse children"},
	},
	Columns: tryColumns,
	Local:   runHookTry,
}

// hookCases is `broodforge hook cases <file>`: one `ok|FAIL|skip  <line>  <tool>  <mode>
// <reach>  <got>  <want>  <note>` line per case of a JSONL file (hook.Case), exit 1 with
// cases-failed on any FAIL; a case whose needs path is absent on this host is skip.
var hookCases = Command{
	Name:  "hook cases",
	Class: ClassStandalone,
	Flags: []Flag{
		{Name: "file", Kind: FlagString, Value: "<file>", Usage: "the case file"},
	},
	Columns: hook.CaseColumns,
	Local:   runHookCases,
}

func init() {
	commands = append(commands, &hookPreToolUse, &hookPostToolUse, &hookTry, &hookCases)
	positionals[&hookCases] = []string{"file"}
	hookStyle[&hookPreToolUse] = true
}

func runHookPreToolUse(_ context.Context, env *Env) error {
	if code := hook.PreToolUse(env.Stdin, env.Out.w, env.Stderr, os.LookupEnv, os.Environ(), time.Now); code != 0 {
		return &exitError{code}
	}
	return nil
}

func runHookPostToolUse(_ context.Context, env *Env) error {
	hook.PostToolUse(env.Stdin, env.Out.w, env.Stderr, os.LookupEnv, os.Environ(), time.Now)
	return nil
}

var tryColumns = []string{"decision", "reason"}

type tryResult struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func runHookTry(_ context.Context, env *Env) error {
	if env.String("tool") == "" {
		return &usageError{"hook try: --tool <name> is required"}
	}
	d, reason, err := hook.Try{Tool: env.String("tool"), Input: json.RawMessage(env.String("input")),
		ToolUseID: env.String("tool-use-id"), Cwd: env.String("cwd"), Mode: env.String("mode"),
		Reach: env.String("reach"), Tree: env.String("tree"), Canonical: env.String("canonical"),
		Transcript: env.String("transcript-path"), Answer: env.String("answer"),
		AnswerReason: env.String("answer-reason"), SessionAllow: env.String("session-allow"),
		Workflow: env.String("workflow")}.Run()
	if err != nil {
		return hookError(err)
	}
	out, err := encodeAny(tryResult{d, reason})
	if err != nil {
		return err
	}
	return env.Out.Result(out, tryColumns)
}

func runHookCases(_ context.Context, env *Env) error {
	f, err := os.Open(env.String("file"))
	if err != nil {
		return control.NewError(CodeBadCases, err.Error())
	}
	defer f.Close()
	results, err := hook.Cases(f, filepath.Dir(env.String("file")))
	if err != nil {
		var be *hook.BadCasesError
		if errors.As(err, &be) {
			return control.NewError(CodeBadCases, be.Msg)
		}
		return hookError(err)
	}
	failed := 0
	for _, r := range results {
		if r.Result == hook.ResultFail {
			failed++
		}
		out, err := encodeAny(r)
		if err != nil {
			return err
		}
		if err := env.Out.Record(out, hook.CaseColumns); err != nil { // every case, no list limit
			return err
		}
	}
	if failed > 0 {
		return control.NewError(CodeCasesFailed, fmt.Sprintf("%d of %d cases failed", failed, len(results)))
	}
	return nil
}

// hookError is the hook's own error as the CLI's error line, `error: hook: <what>`.
func hookError(err error) error {
	var he *hook.Error
	if errors.As(err, &he) {
		return control.NewError(CodeHook, he.What)
	}
	return err
}

// encodeAny is json.Marshal without HTML escaping, for the hook and credential commands'
// results.
func encodeAny[T tryResult | hook.CaseResult | credentialRecord | credentialShowRecord | []credentialRecord | loginResult | initResult](v T) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}
