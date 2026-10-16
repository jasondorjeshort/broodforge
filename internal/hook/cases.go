// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// Defaults of hook try's flags and of a case's absent fields: scratch channel 100's mode and
// reach, the current directory as the tree, the tree as the canonical.
const (
	DefaultMode      = "prompt"
	DefaultReach     = "code"
	DefaultTree      = "."
	DefaultToolUseID = "toolu_01"
)

// Outcome tokens beside the three decisions: a hook error, and a case whose needs is absent.
const (
	ExpectError = "error"
	ResultOK    = "ok"
	ResultFail  = "FAIL"
	ResultSkip  = "skip"
)

// Try is the input of one hook try or one case.
type Try struct {
	Tool         string
	Input        json.RawMessage
	ToolUseID    string
	Cwd          string
	Mode         string
	Reach        string
	Tree         string
	Canonical    string
	Transcript   string
	Answer       string
	AnswerReason string
	SessionAllow string
}

// Run decides one try through Decide, with the defaults filled in.
func (t Try) Run() (decision, reason string, err error) {
	s := Settings{Tree: t.Tree, Canonical: t.Canonical}
	if s.Tree == "" {
		s.Tree = DefaultTree
	}
	if s.Canonical == "" {
		s.Canonical = s.Tree
	}
	mode, reach := t.Mode, t.Reach
	if mode == "" {
		mode = DefaultMode
	}
	if reach == "" {
		reach = DefaultReach
	}
	if s.Mode, err = ParseMode(mode); err != nil {
		return "", "", err
	}
	if s.Reach, err = ParseReach(reach); err != nil {
		return "", "", err
	}
	if s.Answer, err = ParseAnswer(t.Answer, t.AnswerReason); err != nil {
		return "", "", err
	}
	s.SessionAllow = policy.SplitList(t.SessionAllow)
	id := t.ToolUseID
	if id == "" {
		id = DefaultToolUseID
	}
	if t.Tool == "" {
		return "", "", &Error{"no tool_name"}
	}
	r, err := Decide(Input{ToolName: t.Tool, ToolInput: t.Input, ToolUseID: id, Cwd: t.Cwd, TranscriptPath: t.Transcript}, s)
	if err != nil {
		return "", "", err
	}
	return r.Decision.String(), r.Reason, nil
}

// Case is one line of a case file (05-debug-interface.md, The PreToolUse hook). needs names a
// path that must exist for the case to run, for a link only some hosts have.
type Case struct {
	Tool         string          `json:"tool"`
	Input        json.RawMessage `json:"input"`
	ToolUseID    string          `json:"toolUseId"`
	Cwd          string          `json:"cwd"`
	Mode         string          `json:"mode"`
	Reach        string          `json:"reach"`
	Tree         string          `json:"tree"`
	Canonical    string          `json:"canonical"`
	Transcript   string          `json:"transcript"`
	Answer       string          `json:"answer"`
	AnswerReason string          `json:"answerReason"`
	SessionAllow string          `json:"sessionAllow"`
	Expect       string          `json:"expect"`
	Needs        string          `json:"needs"`
	Note         string          `json:"note"`
}

// CaseResult is one line of hook cases' output: `ok|FAIL|skip  <line>  <tool>  <mode>  <reach>
// <got>  <want>  <note>`, mode and reach as given ("" for the default).
type CaseResult struct {
	Result string `json:"result"`
	Line   int    `json:"line"`
	Tool   string `json:"tool"`
	Mode   string `json:"mode"`
	Reach  string `json:"reach"`
	Got    string `json:"got"`
	Want   string `json:"want"`
	Note   string `json:"note"`
}

// CaseColumns are CaseResult's keys in text order.
var CaseColumns = []string{"result", "line", "tool", "mode", "reach", "got", "want", "note"}

// BadCasesError is a case file that cannot be read: a line that is not a case.
type BadCasesError struct{ Msg string }

func (e *BadCasesError) Error() string { return e.Msg }

// Cases runs every case of a JSONL file; blank lines are skipped. A hook error is the outcome
// "error", so a case can expect one. A relative tree, canonical, transcript or needs is taken
// from dir, the case file's directory, so the file runs from any working directory; a relative
// tool_input path stays relative to the tree, as the engine's is.
func Cases(r io.Reader, dir string) ([]CaseResult, error) {
	// {dir} is the case file's directory, absolute, with forward slashes; {posixdir} the same
	// with a drive `C:` as `/c`, the form the engine expands back (expand).
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	slashDir := filepath.ToSlash(absDir)
	posixDir := slashDir
	if v := filepath.VolumeName(absDir); len(v) == 2 && v[1] == ':' {
		posixDir = "/" + strings.ToLower(v[:1]) + slashDir[2:]
	}
	subst := strings.NewReplacer("{dir}", slashDir, "{posixdir}", posixDir)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), MaxInput)
	var out []CaseResult
	for n := 1; sc.Scan(); n++ {
		line := []byte(subst.Replace(string(bytes.TrimSpace(sc.Bytes()))))
		if len(line) == 0 {
			continue
		}
		var c Case
		if err := json.Unmarshal(line, &c); err != nil {
			return out, &BadCasesError{fmt.Sprintf("line %d: %v", n, err)}
		}
		switch c.Expect {
		case "allow", "deny", "defer", ExpectError:
		default:
			return out, &BadCasesError{fmt.Sprintf("line %d: expect is %q, want allow, deny, defer or error", n, c.Expect)}
		}
		for _, f := range []*string{&c.Tree, &c.Canonical, &c.Transcript, &c.Needs, &c.Cwd} {
			if *f != "" && !filepath.IsAbs(*f) && !rooted(*f) && filepath.VolumeName(*f) == "" {
				*f = filepath.Join(absDir, *f)
			}
		}
		res := CaseResult{Line: n, Tool: c.Tool, Mode: c.Mode, Reach: c.Reach, Want: c.Expect, Note: c.Note}
		if c.Needs != "" {
			if _, err := os.Lstat(c.Needs); err != nil {
				res.Result, res.Got = ResultSkip, "-"
				out = append(out, res)
				continue
			}
		}
		got, _, err := Try{Tool: c.Tool, Input: c.Input, ToolUseID: c.ToolUseID, Cwd: c.Cwd, Mode: c.Mode, Reach: c.Reach,
			Tree: c.Tree, Canonical: c.Canonical, Transcript: c.Transcript, Answer: c.Answer,
			AnswerReason: c.AnswerReason, SessionAllow: c.SessionAllow}.Run()
		var he *Error
		switch {
		case errors.As(err, &he):
			got = ExpectError
		case err != nil:
			return out, err
		}
		res.Got, res.Result = got, ResultFail
		if got == c.Expect {
			res.Result = ResultOK
		}
		out = append(out, res)
	}
	if err := sc.Err(); err != nil {
		return out, &BadCasesError{err.Error()}
	}
	return out, nil
}
