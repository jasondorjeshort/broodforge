// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
)

// The hook half of the P6 Proof (08-implementation-plan.md), on the built binary.

const hookWrite = `{"tool_name":"Write","tool_input":{"file_path":"/tmp/x/a.ts","content":""},` +
	`"tool_use_id":"toolu_01","cwd":"/tmp/x","transcript_path":"/tmp/cfg/projects/x/s.jsonl"}`

var hookEnv = []string{"BROODFORGE_SESSION=s0001", "BROODFORGE_MODE=prompt", "BROODFORGE_REACH=code",
	"BROODFORGE_TREE=/tmp/x", "BROODFORGE_CANONICAL=/tmp/c", "BROODFORGE_HOOK_LOG=/dev/null"}

// runStdin runs the binary with stdin, which clitest.Run does not take.
func runStdin(t *testing.T, env []string, stdin string, args ...string) clitest.Result {
	t.Helper()
	cmd := exec.Command(clitest.Binary(t), args...)
	cmd.Env = clitest.Environ(env)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	r := clitest.Result{Stdout: out.String(), Stderr: errb.String()}
	if ee, ok := err.(*exec.ExitError); ok {
		r.Code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run %q: %v", args, err)
	}
	t.Logf("$ broodforge %s <<< %q\nexit %d\nstdout: %q\nstderr: %q", strings.Join(args, " "), stdin, r.Code, r.Stdout, r.Stderr)
	return r
}

func TestProofHookPreToolUse(t *testing.T) {
	want(t, runStdin(t, hookEnv, hookWrite, "hook", "pre-tool-use"), 0,
		`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"defer","permissionDecisionReason":"Broodforge: waiting for a person in this session to answer"}}`+"\n", "")
	noTranscript := strings.Replace(hookWrite, `,"transcript_path":"/tmp/cfg/projects/x/s.jsonl"`, "", 1)
	want(t, runStdin(t, hookEnv, noTranscript, "hook", "pre-tool-use"), 2, "", "hook: no transcript_path\n")
	want(t, runStdin(t, hookEnv, "not json", "hook", "pre-tool-use"), 2, "", "hook: stdin is not a JSON object\n")
	want(t, runStdin(t, hookEnv[1:], hookWrite, "hook", "pre-tool-use"), 2, "", "hook: BROODFORGE_SESSION is not set\n")
	want(t, runStdin(t, hookEnv, hookWrite, "hook", "pre-tool-use", "extra"), 2, "",
		"hook: usage: hook pre-tool-use: unexpected argument \"extra\"\n")
}

func TestProofHookTry(t *testing.T) {
	want(t, clitest.Run(t, nil, "hook", "try", "--tool", "Read", "--input", `{"file_path":"/tmp/cfg/.credentials.json"}`,
		"--transcript-path", "/tmp/cfg/projects/x/s.jsonl", "--expect", "deny"), 0,
		"deny  Denied by Broodforge: /tmp/cfg/.credentials.json is a credential file\n", "")
	want(t, clitest.Run(t, nil, "--json", "hook", "try", "--tool", "Bash", "--input", `{"command":"ls"}`, "--mode", "policy",
		"--transcript-path", "/tmp/cfg/projects/x/s.jsonl"), 0,
		`{"decision":"allow","reason":"Broodforge: allowed by this channel's reach"}`+"\n", "")
	want(t, clitest.Run(t, nil, "hook", "try", "--tool", "Bash", "--answer", "toolu_01:allow",
		"--transcript-path", "/tmp/cfg/projects/x/s.jsonl", "--expect", "allow"), 0,
		"allow  Broodforge: allowed by a person in this session\n", "")
	want(t, clitest.Run(t, nil, "hook", "try", "--tool", "Read", "--input", `{"file_path":"a"}`), 1, "",
		"error: hook: no transcript_path\n")
	want(t, clitest.Run(t, nil, "hook", "try", "--tool", "Read", "--input", `{"file_path":"a"}`, "--expect", "hook"), 0, "", "")
	want(t, clitest.Run(t, nil, "hook", "try"), 2, "", "error: usage: hook try: --tool <name> is required\n")
}

func TestProofHookCases(t *testing.T) {
	r := clitest.Run(t, nil, "hook", "cases", "../../tests/hook/cases.jsonl")
	if r.Code != 0 || r.Stderr != "" || r.Stdout == "" {
		t.Fatalf("exit %d stderr %q", r.Code, r.Stderr)
	}
	for _, line := range strings.Split(strings.TrimSuffix(r.Stdout, "\n"), "\n") {
		if !strings.HasPrefix(line, "ok  ") && !strings.HasPrefix(line, "skip  ") {
			t.Errorf("case line %q", line)
		}
	}
	bad := t.TempDir() + "/cases.jsonl"
	if err := writeFile(bad, `{"tool":"Bash","input":{},"mode":"policy","reach":"code","transcript":"/c/projects/x/s.jsonl","expect":"defer"}`+"\n"); err != nil {
		t.Fatal(err)
	}
	want(t, clitest.Run(t, nil, "hook", "cases", bad), 1, "FAIL  1  Bash  policy  code  allow  defer  \"\"\n",
		"error: cases-failed: 1 of 1 cases failed\n")
	want(t, clitest.Run(t, nil, "hook", "cases"), 2, "", "error: usage: hook cases: missing <file>\n")
}
