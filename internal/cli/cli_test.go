// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(t *testing.T, args []string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestRunNoArgsIsUsage(t *testing.T) {
	code, stdout, stderr := run(t, nil)
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.HasPrefix(stderr, "usage: broodforge ") {
		t.Errorf("stderr = %q, want usage line first", stderr)
	}
}

func TestRunWithArgsReportsNoCommandsThenUsage(t *testing.T) {
	code, stdout, stderr := run(t, []string{"--root", "x", "foo"})
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.HasPrefix(stderr, "error: usage: no commands exist yet\nusage: broodforge ") {
		t.Errorf("stderr = %q, want error line then usage line", stderr)
	}
}
