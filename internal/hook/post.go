// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// PostOutput is the PostToolUse hook output: the children's contexts as one additionalContext.
type PostOutput struct {
	HookSpecificOutput PostSpecificOutput `json:"hookSpecificOutput"`
}

// PostSpecificOutput is the PostToolUse object of the engine's hook output.
type PostSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// PostToolUse is `broodforge hook post-tool-use` as the engine runs it (ADR 0025 rule 2): in
// code reach, with a copy named, the repository's matching PostToolUse children run serially
// and their additionalContext, or an exit-2 child's stderr, is printed as this hook's own
// additionalContext, joined by newlines; nothing printed when there is none. It never blocks:
// every path exits 0, an error written to stderr as `hook: <what>` with nothing on stdout.
func PostToolUse(stdin io.Reader, stdout, stderr io.Writer, lookup func(string) (string, bool), environ []string, now func() time.Time) (code int) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintf(stderr, "hook: internal error: %v\n", p)
		}
		code = 0
	}()
	if err := postToolUse(stdin, stdout, lookup, environ, now); err != nil {
		msg := err.Error()
		if _, ok := err.(*Error); !ok {
			msg = "hook: " + msg
		}
		fmt.Fprintln(stderr, msg)
	}
	return 0
}

func postToolUse(stdin io.Reader, stdout io.Writer, lookup func(string) (string, bool), environ []string, now func() time.Time) error {
	workflow, _ := lookup(EnvWorkflow)
	reach, _ := lookup(EnvReach)
	if workflow == "" {
		return nil // no copy: no children, nothing to say
	}
	r, err := ParseReach(reach)
	if err != nil {
		return err
	}
	if r != policy.ReachCode {
		return nil
	}
	tree, ok := lookup(EnvTree)
	if !ok || tree == "" {
		return &Error{EnvTree + " is not set"}
	}
	b, err := io.ReadAll(io.LimitReader(stdin, MaxInput+1))
	if err != nil {
		return &Error{"cannot read stdin"}
	}
	if len(b) > MaxInput {
		return &Error{"stdin is over 64 MiB"}
	}
	var in struct {
		ToolName  string `json:"tool_name"`
		ToolUseID string `json:"tool_use_id"`
	}
	if t := bytes.TrimSpace(b); len(t) == 0 || t[0] != '{' || json.Unmarshal(t, &in) != nil {
		return &Error{"stdin is not a JSON object"}
	}
	if in.ToolName == "" {
		return &Error{"no tool_name"}
	}
	session, _ := lookup(EnvSession)
	log, _ := lookup(EnvHookLog)
	var logErr error
	c := Chain{Workflow: workflow, Tree: tree, Environ: environ, now: now}
	if log != "" {
		c.Log = func(l ChildLine) {
			if logErr == nil {
				logErr = appendLog(log, childLogLine{Time: now().UTC().Format(time.RFC3339), Session: session,
					Tool: in.ToolName, ToolUseID: in.ToolUseID, ChildLine: l})
			}
		}
	}
	contexts, err := c.Post(in.ToolName, b)
	if err != nil {
		return err
	}
	if len(contexts) > 0 {
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(PostOutput{PostSpecificOutput{EventPostToolUse, strings.Join(contexts, "\n")}}); err != nil {
			return err
		}
		if _, err := stdout.Write(out.Bytes()); err != nil {
			return &Error{"cannot write stdout"}
		}
	}
	return logErr
}
