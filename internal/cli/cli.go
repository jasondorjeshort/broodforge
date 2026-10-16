// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"fmt"
	"io"
)

// Exit codes, docs/design/05-cli-conventions.md (N-07).
const (
	ExitOK      = 0
	ExitFailure = 1
	ExitUsage   = 2
)

const usage = `usage: broodforge [--root <path>] [--json] <group> <command> [args]
`

// Run dispatches one invocation and returns the process exit code. args excludes the program
// name. No command exists yet, so every invocation is a usage error.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "error: usage: no commands exist yet")
	}
	fmt.Fprint(stderr, usage)
	return ExitUsage
}
