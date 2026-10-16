// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
)

// The P3 Proof (08-implementation-plan.md) on the built binary: `engine run --fake`. Run with
// -v to see each command and its exit code. received-<n>.json goes to a temp directory, so the
// fixtures stay untouched.

const fixtures = "../../fixtures/engine/"

func engineRun(t *testing.T, fixture string, turn string, extra ...string) clitest.Result {
	t.Helper()
	args := append([]string{"engine", "run", "--fake", fixtures + fixture, "--turn", turn, "--prompt", "x", "--received-dir", t.TempDir()}, extra...)
	return clitest.Run(t, nil, args...)
}

type event struct {
	Type      string `json:"type"`
	MessageID string `json:"messageId"`
	Blocks    []struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	} `json:"blocks"`
	Deferred *struct {
		ToolUseID string `json:"toolUseId"`
	} `json:"deferred"`
	Code        *int `json:"code"`
	Interrupted bool `json:"interrupted"`
}

func events(t *testing.T, stdout string) []event {
	t.Helper()
	var evs []event
	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		var ev event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("not a JSON line: %q", line)
		}
		evs = append(evs, ev)
	}
	return evs
}

func TestProofEngineRunUnknown(t *testing.T) {
	r := engineRun(t, "basic", "1")
	if n := strings.Count(r.Stdout, `"type":"unknown"`); r.Code != 0 || n != 3 || r.Stderr != "" {
		t.Errorf("basic turn 1: exit %d, %d unknown lines, stderr %q; want 0, 3, none", r.Code, n, r.Stderr)
	}
	// Text and --json print the same JSONL.
	if j := clitest.Run(t, nil, "--json", "engine", "run", "--fake", fixtures+"basic", "--prompt", "x", "--received-dir", t.TempDir()); j.Stdout != r.Stdout {
		t.Error("--json output differs from text output")
	}
}

func TestProofEngineRunParallel(t *testing.T) {
	r := engineRun(t, "parallel", "1")
	var uses []string
	var msgs []string
	var deferred string
	for _, ev := range events(t, r.Stdout) {
		for _, b := range ev.Blocks {
			if ev.Type == "assistant" && b.Kind == "tool-use" {
				uses, msgs = append(uses, b.ID), append(msgs, ev.MessageID)
			}
		}
		if ev.Type == "result" && ev.Deferred != nil {
			deferred = ev.Deferred.ToolUseID
		}
	}
	if r.Code != 0 || len(uses) != 2 || msgs[0] != msgs[1] || deferred != uses[1] {
		t.Errorf("parallel turn 1: exit %d, tool uses %v under %v, deferred %q; want two under one message, the second deferred", r.Code, uses, msgs, deferred)
	}
}

func TestProofEngineRunExitCode(t *testing.T) {
	for _, tc := range []struct {
		turn   string
		code   int
		stderr string
	}{
		{"1", 1, "error: engine-exit: the engine exited 1\n"},
		{"2", 1, "error: engine-exit: the engine exited 1\n"},
		{"3", 0, ""},
	} {
		r := engineRun(t, "interrupted", tc.turn)
		evs := events(t, r.Stdout)
		last := evs[len(evs)-1]
		if r.Code != tc.code || r.Stderr != tc.stderr || last.Type != "exit" || last.Code == nil || *last.Code != tc.code {
			t.Errorf("interrupted turn %s: exit %d stderr %q last %+v; want %d %q", tc.turn, r.Code, r.Stderr, last, tc.code, tc.stderr)
		}
	}
}

func TestEngineRunExpect(t *testing.T) {
	// Turn 1 has a result but exits 1, a refusal: its token is the error code.
	if r := engineRun(t, "interrupted", "1", "--expect", "engine-exit"); r.Code != 0 || r.Stderr != "" {
		t.Errorf("turn 1 --expect engine-exit: exit %d stderr %q", r.Code, r.Stderr)
	}
	if r := engineRun(t, "basic", "1", "--expect", "result"); r.Code != 0 || r.Stderr != "" {
		t.Errorf("basic turn 1 --expect result: exit %d stderr %q", r.Code, r.Stderr)
	}
	// Reproduce (05): a bare copy of one turn file, no fixture.json, replays and exits 0; a
	// turn whose stream has no result fails --expect result.
	for _, tc := range []struct {
		src, token string
	}{
		{"parallel/turn-2.jsonl", "result"},
		{"interrupted/turn-2.jsonl", "no-result"},
	} {
		dir := t.TempDir()
		data, err := os.ReadFile(fixtures + tc.src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "turn-1.jsonl"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if r := clitest.Run(t, nil, "engine", "run", "--fake", dir, "--prompt", "x", "--expect", tc.token); r.Code != 0 || r.Stderr != "" {
			t.Errorf("bare %s --expect %s: exit %d stderr %q", tc.src, tc.token, r.Code, r.Stderr)
		}
		if tc.token == "no-result" {
			r := clitest.Run(t, nil, "engine", "run", "--fake", dir, "--prompt", "x", "--expect", "result")
			if r.Code != 1 || r.Stderr != "error: expect: wanted result, got no-result\n" {
				t.Errorf("bare %s --expect result: exit %d stderr %q", tc.src, r.Code, r.Stderr)
			}
		}
	}
}

func TestEngineRunErrors(t *testing.T) {
	r := clitest.Run(t, nil, "engine", "run", "--prompt", "x")
	want(t, r, 2, "", "error: usage: engine run: --fake <fixture dir> or --real is required\n")
	r = engineRun(t, "basic", "9")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: no-turn: no such turn: ") {
		t.Errorf("turn 9: exit %d stderr %q", r.Code, r.Stderr)
	}
	r = clitest.Run(t, nil, "engine", "run", "--fake", fixtures+"basic", "--prompt", "x", "--received-dir", filepath.Join(t.TempDir(), "missing"))
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: bad-received-dir: ") {
		t.Errorf("missing received dir: exit %d stderr %q", r.Code, r.Stderr)
	}
	r = clitest.Run(t, nil, "engine", "run", "--fake", fixtures+"no-such-fixture", "--prompt", "x")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: bad-fixture: ") {
		t.Errorf("missing dir: exit %d stderr %q", r.Code, r.Stderr)
	}
}
