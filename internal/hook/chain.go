// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// The child chain (ADR 0025 rule 2; 03-hook-contract.md, The child chain): the repository's
// hooks of one event, read from the copy's hooks.json, run serially as children of the bot's
// hook. From a PreToolUse child the chain takes exit 2 (the first stderr line) or a JSON
// permissionDecision deny and nothing else; from a PostToolUse child its additionalContext or
// an exit-2 stderr. Everything else a child says goes to the log and is dropped.

// The chain's variables: the copy, set by spawn (absent: no children), and the shell the spawn
// exports from engine.gitBashPath on a host with drive letters.
const (
	EnvWorkflow    = "BROODFORGE_WORKFLOW"
	EnvGitBashPath = "CLAUDE_CODE_GIT_BASH_PATH"
	envProjectDir  = "CLAUDE_PROJECT_DIR"
)

// The events a repository's hooks may run under (ADR 0025 rule 6).
const (
	EventPreToolUse  = "PreToolUse"
	EventPostToolUse = "PostToolUse"
)

// Budgets: per child the smaller of its own timeout and ChildBudget, the whole chain
// ChainBudget, inside the thirty seconds the engine gives the bot's hook.
const (
	ChildBudget = 10 * time.Second
	ChainBudget = 20 * time.Second
)

// hooksFile is the copy's rendered hook list (internal/workflow, HooksFile).
const hooksFile = "hooks.json"

// posixShell runs the children on a host without drive letters (the droplet).
const posixShell = "/bin/sh"

// maxChildOutput bounds what the chain keeps of a child's stdout and of its stderr.
const maxChildOutput = 1 << 20

// waitAfterKill bounds the wait for a killed child's pipes, which a grandchild may hold open.
const waitAfterKill = time.Second

// child is one hooks.json entry, the shape internal/workflow writes (Hook); the hook imports
// nothing of the bot process, so the five fields are read here.
type child struct {
	Event   string `json:"event"`
	Matcher string `json:"matcher"`
	Command string `json:"command"`
	Name    string `json:"name"`
	Timeout *int   `json:"timeout"`
}

// Chain is one run's setup.
type Chain struct {
	Workflow string   // the copy: hooks.json's directory and the children's CLAUDE_PROJECT_DIR
	Tree     string   // the children's working directory
	Environ  []string // the hook's own environment, handed to every child
	// Log takes one line per child run or skipped matcher; nil logs nothing (hook try, cases).
	Log func(ChildLine)
	now func() time.Time
}

// ChildLine is one hook.log line of the chain: names, exit, time and what was dropped, never
// tool input or a child's output.
type ChildLine struct {
	Event   string   `json:"event"`
	Child   string   `json:"child"`
	Exit    *int     `json:"exit,omitempty"`
	Ms      int64    `json:"ms"`
	Outcome string   `json:"outcome"`
	Detail  string   `json:"detail,omitempty"`
	Dropped []string `json:"dropped,omitempty"`
}

// outcomeKind is what the chain took from one child.
type outcomeKind int

const (
	outcomePassed     outcomeKind = iota // nothing taken: the next child runs
	outcomeDeny                          // exit 2, or a JSON permissionDecision deny
	outcomeOverBudget                    // killed at its own budget
	outcomeChainOver                     // cut, or never run, at the chain's budget
	outcomeNotStarted                    // the shell or the command could not start
	outcomeContext                       // a PostToolUse child's additionalContext or exit-2 stderr
)

func (k outcomeKind) String() string {
	switch k {
	case outcomePassed:
		return "passed"
	case outcomeDeny:
		return "deny"
	case outcomeOverBudget:
		return "over-budget"
	case outcomeChainOver:
		return "over-chain-budget"
	case outcomeNotStarted:
		return "not-started"
	case outcomeContext:
		return "context"
	}
	return "unknown"
}

// childResult is one child's outcome: Text is the deny reason, the not-started reason or the
// context by Kind; Seconds its own budget, which it ran over; Ran whether a child the chain's
// budget stopped had started.
type childResult struct {
	Kind    outcomeKind
	Text    string
	Seconds int
	Ran     bool
	Exit    *int
	Dropped []string
}

// The three refusal texts (decisions/0025-refusals-and-checks.md, Refusals and lifts; wording:
// owner to confirm).
func childDenyText(name, reason string) string {
	return policy.DenyPrefix + " refused by the project's hook " + name + ": " + sentence(reason) +
		". The project's rules on its default branch decide this."
}

func overBudgetText(name string, seconds int, event string) string {
	return fmt.Sprintf("%s the project's hook %s timed out after %s. The project's authors fix it on the default "+
		"branch, or the owner removes %s from workflow.hooks for this project.", policy.DenyPrefix, name, secondsText(seconds), event)
}

// chainOverText is the deny for a child the chain's own budget stopped or never started: not a
// companion row, since the child's own budget did not run out.
func chainOverText(name string, ran bool) string {
	if ran {
		return policy.DenyPrefix + " the project's hook " + name + " was stopped: the project's hooks exceeded twenty seconds together before " +
			name + " finished."
	}
	return policy.DenyPrefix + " the project's hook " + name + " did not run: the project's hooks exceeded twenty seconds together before " +
		name + " ran."
}

func secondsText(n int) string {
	if n == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", n)
}

func notStartedText(name, why, event string) string {
	return policy.DenyPrefix + " the project's hook " + name + " did not start: " + sentence(why) +
		". The project's authors fix it on the default branch, or the owner removes " + event +
		" from workflow.hooks for this project."
}

// sentence trims a reason for the text around it: surrounding space, then one final period, so
// the text's own period does not double it.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "no reason given"
	}
	return strings.TrimSuffix(s, ".")
}

// readChildren is the copy's entries of one event; a copy without hooks.json has none. A copy
// that is not a directory is a hook error: the variable named one, so the hook fails closed.
func readChildren(dir, event string) ([]child, error) {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		why := "not a directory"
		if err != nil {
			why = errText(err)
		}
		return nil, &Error{EnvWorkflow + " names no directory: " + why}
	}
	b, err := os.ReadFile(filepath.Join(dir, hooksFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &Error{"cannot read " + EnvWorkflow + "/" + hooksFile + ": " + errText(err)}
	}
	var all []child
	if err := json.Unmarshal(b, &all); err != nil {
		return nil, &Error{EnvWorkflow + "/" + hooksFile + ": " + err.Error()}
	}
	var out []child
	for _, c := range all {
		if c.Event == event {
			out = append(out, c)
		}
	}
	return out, nil
}

// matches reads a matcher as the engine's hooks documentation defines it: empty or `*` matches
// every tool; anything else is a case-sensitive regular expression the whole tool name must
// match. An invalid pattern matches nothing; its error is returned for the log.
func matches(matcher, tool string) (bool, error) {
	if matcher == "" || matcher == "*" {
		return true, nil
	}
	re, err := regexp.Compile("^(?:" + matcher + ")$")
	if err != nil {
		return false, err
	}
	return re.MatchString(tool), nil
}

// lookupEnv finds name in an environment slice, as the host compares names.
func lookupEnv(environ []string, name string) (string, bool) {
	val, found := "", false
	for _, kv := range environ {
		k, v, ok := strings.Cut(kv, "=")
		if ok && k != "" && (k == name || drives && strings.EqualFold(k, name)) {
			val, found = v, true // the last entry wins, as exec's dedupe keeps it
		}
	}
	return val, found
}

// shell is the program the children run under, or why there is none.
func (c Chain) shell() (string, string) {
	if !drives {
		return posixShell, ""
	}
	if v, _ := lookupEnv(c.Environ, EnvGitBashPath); v != "" {
		return v, ""
	}
	return "", EnvGitBashPath + " is not set, so there is no Git Bash to run it"
}

// childEnv is the hook's environment with CLAUDE_PROJECT_DIR naming the copy, so a command's
// ${CLAUDE_PROJECT_DIR:-.} forms resolve there with no rewrite.
func (c Chain) childEnv() []string {
	out := make([]string, 0, len(c.Environ)+1)
	for _, kv := range c.Environ {
		if k, _, _ := strings.Cut(kv, "="); strings.EqualFold(k, envProjectDir) {
			continue
		}
		out = append(out, kv)
	}
	copyDir, err := filepath.Abs(c.Workflow) // a child's cwd is the tree, so a relative copy would not resolve
	if err != nil {
		copyDir = c.Workflow
	}
	return append(out, envProjectDir+"="+filepath.ToSlash(copyDir))
}

func (c Chain) log(l ChildLine) {
	if c.Log != nil {
		c.Log(l)
	}
}

func (c Chain) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// Pre runs the matching PreToolUse children for a call the bot did not deny. It returns the
// first child deny as the bot's deny with its log trace, or nil when every child passed.
func (c Chain) Pre(tool string, stdin []byte) (deny *policy.Result, trace string, err error) {
	err = c.run(EventPreToolUse, tool, stdin, func(ch child, r childResult) bool {
		name := ch.displayName()
		switch r.Kind {
		case outcomePassed, outcomeContext:
			return true
		case outcomeDeny:
			deny = &policy.Result{Decision: policy.Deny, Reason: childDenyText(name, r.Text)}
			trace = "denied: refused by the project's hook " + name + ": " + sentence(r.Text)
		case outcomeOverBudget:
			deny = &policy.Result{Decision: policy.Deny, Reason: overBudgetText(name, r.Seconds, EventPreToolUse)}
			trace = "denied: the project's hook " + name + " timed out after " + secondsText(r.Seconds)
		case outcomeChainOver:
			deny = &policy.Result{Decision: policy.Deny, Reason: chainOverText(name, r.Ran)}
			trace = "denied: " + strings.TrimSuffix(strings.TrimPrefix(deny.Reason, policy.DenyPrefix+" "), ".")
		case outcomeNotStarted:
			deny = &policy.Result{Decision: policy.Deny, Reason: notStartedText(name, r.Text, EventPreToolUse)}
			trace = "denied: the project's hook " + name + " did not start: " + sentence(r.Text)
		}
		return false
	})
	return deny, trace, err
}

// Post runs the matching PostToolUse children and returns their contexts in order. It never
// stops early: a PostToolUse child cannot block.
func (c Chain) Post(tool string, stdin []byte) ([]string, error) {
	var contexts []string
	err := c.run(EventPostToolUse, tool, stdin, func(_ child, r childResult) bool {
		switch r.Kind {
		case outcomeContext:
			contexts = append(contexts, r.Text)
		case outcomePassed, outcomeDeny, outcomeOverBudget, outcomeChainOver, outcomeNotStarted:
		}
		return true
	})
	return contexts, err
}

func (ch child) displayName() string {
	if ch.Name != "" {
		return ch.Name
	}
	return ch.Command
}

// run runs each matching child of event serially under the chain's budget; next says whether
// to go on after a child.
func (c Chain) run(event, tool string, stdin []byte, next func(child, childResult) bool) error {
	children, err := readChildren(c.Workflow, event)
	if err != nil || len(children) == 0 {
		return err
	}
	deadline := c.clock().Add(ChainBudget)
	for _, ch := range children {
		ok, err := matches(ch.Matcher, tool)
		if err != nil {
			c.log(ChildLine{Event: event, Child: ch.displayName(), Outcome: "skipped", Detail: "invalid matcher " + strconv.Quote(ch.Matcher) + ": " + err.Error()})
			continue
		}
		if !ok {
			continue
		}
		own := ChildBudget
		if ch.Timeout != nil && *ch.Timeout > 0 && time.Duration(*ch.Timeout)*time.Second < own {
			own = time.Duration(*ch.Timeout) * time.Second
		}
		budget, capped := own, false
		if left := deadline.Sub(c.clock()); left < own {
			budget, capped = max(left, 0), true
		}
		start := c.clock()
		var r childResult
		if budget > 0 {
			r = c.runChild(event, ch, stdin, budget)
		} else {
			r = childResult{Kind: outcomeOverBudget}
		}
		if r.Kind == outcomeOverBudget {
			r.Seconds = int(own / time.Second)
			if capped {
				r.Kind, r.Ran = outcomeChainOver, budget > 0
			}
		}
		c.log(ChildLine{Event: event, Child: ch.displayName(), Exit: r.Exit, Ms: c.clock().Sub(start).Milliseconds(),
			Outcome: r.Kind.String(), Detail: logDetail(r), Dropped: r.Dropped})
		if !next(ch, r) {
			return nil
		}
	}
	return nil
}

// logDetail is the log's note for an outcome: the reason of a refusal, never a context.
func logDetail(r childResult) string {
	switch r.Kind {
	case outcomeDeny, outcomeNotStarted:
		return r.Text
	case outcomeOverBudget:
		return "budget " + secondsText(r.Seconds)
	case outcomeChainOver:
		if r.Ran {
			return "stopped at the chain's twenty seconds"
		}
		return "not run: the chain's twenty seconds were spent"
	case outcomePassed, outcomeContext:
	}
	return ""
}

// capped keeps the first maxChildOutput bytes written to it and discards the rest.
type capped struct{ bytes.Buffer }

func (w *capped) Write(p []byte) (int, error) {
	if room := maxChildOutput - w.Len(); room > 0 {
		w.Buffer.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

// runChild runs one child through the shell with the engine's stdin bytes, the tree as its
// working directory and the copy as CLAUDE_PROJECT_DIR, killed at its budget (outcomeOverBudget,
// whose Seconds the caller sets).
func (c Chain) runChild(event string, ch child, stdin []byte, budget time.Duration) childResult {
	sh, why := c.shell()
	if sh == "" {
		return childResult{Kind: outcomeNotStarted, Text: why}
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	cmd := exec.CommandContext(ctx, sh, "-c", ch.Command)
	cmd.Dir = c.Tree
	cmd.Env = c.childEnv()
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errb capped
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.WaitDelay = waitAfterKill
	if err := cmd.Start(); err != nil {
		return childResult{Kind: outcomeNotStarted, Text: errText(err)}
	}
	err := cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return childResult{Kind: outcomeOverBudget}
	}
	code := 0
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee):
		code = ee.ExitCode()
	case errors.Is(err, exec.ErrWaitDelay):
		// The child exited 0, but something it left behind held its pipes until WaitDelay closed
		// them: it is judged by its exit and the output collected, as any exit 0.
		code = cmd.ProcessState.ExitCode()
	case err != nil:
		return childResult{Kind: outcomeNotStarted, Text: errText(err)}
	}
	r := childResult{Exit: &code}
	stderr := strings.TrimSpace(errb.String())
	switch code {
	case 0:
		return judgeStdout(event, out.Bytes(), r)
	case 2:
		if event == EventPostToolUse {
			r.Kind, r.Text = outcomeContext, stderr
			if stderr == "" {
				r.Kind = outcomePassed
			}
			return r
		}
		r.Kind, r.Text = outcomeDeny, firstLine(stderr)
		if r.Text == "" {
			r.Text = "it exited 2 with no message"
		}
		return r
	case 126, 127:
		// The shell's own codes for a command it could not run (not executable, not found).
		r.Kind, r.Text = outcomeNotStarted, firstLine(stderr)
		if r.Text == "" {
			r.Text = fmt.Sprintf("the shell exited %d", code)
		}
		return r
	}
	if out.Len() > 0 {
		r.Dropped = []string{"stdout"}
	}
	return r // any other exit is the engine's non-blocking error: logged, nothing taken
}

// maxReason bounds a reason the chain takes from a child, exit-2 stderr or JSON alike.
const maxReason = 1000

// firstLine is a reason as the chain takes it: the first line, trimmed, at most maxReason bytes
// cut on a rune boundary.
func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	line = strings.TrimSpace(line)
	if len(line) <= maxReason {
		return line
	}
	cut := maxReason
	for cut > 0 && !utf8.RuneStart(line[cut]) {
		cut--
	}
	return line[:cut]
}

// judgeStdout reads an exit-0 child's JSON: for PreToolUse a permissionDecision of deny with
// its reason, for PostToolUse an additionalContext; every other field is dropped and named.
func judgeStdout(event string, stdout []byte, r childResult) childResult {
	if len(bytes.TrimSpace(stdout)) == 0 {
		return r
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(stdout, &top) != nil {
		r.Dropped = []string{"stdout"}
		return r
	}
	var specific map[string]json.RawMessage
	for k, v := range top {
		if k != "hookSpecificOutput" {
			r.Dropped = append(r.Dropped, k)
			continue
		}
		if json.Unmarshal(v, &specific) != nil {
			r.Dropped = append(r.Dropped, k)
		}
	}
	str := func(k string) string {
		var s string
		_ = json.Unmarshal(specific[k], &s)
		return s
	}
	for k := range specific {
		switch {
		case k == "hookEventName":
		case event == EventPreToolUse && k == "permissionDecision" && str(k) == "deny":
			r.Kind, r.Text = outcomeDeny, firstLine(str("permissionDecisionReason"))
		case event == EventPreToolUse && k == "permissionDecisionReason" && str("permissionDecision") == "deny":
		case event == EventPreToolUse && k == "permissionDecision":
			r.Dropped = append(r.Dropped, k+"="+str(k))
		case event == EventPostToolUse && k == "additionalContext" && str(k) != "":
			r.Kind, r.Text = outcomeContext, str(k)
		default:
			r.Dropped = append(r.Dropped, k)
		}
	}
	slices.Sort(r.Dropped)
	return r
}
