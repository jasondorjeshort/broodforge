// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jasondorjeshort/broodforge/internal/hook"
)

// PrePushRefusal is the one line a session's push gets (ADR 0025, rule 4; the 0025 companion's
// pre-push row), `%s` the session id from BROODFORGE_SESSION, PrePushNoSession when it is unset.
// share/hooks/pre-push prints the same line itself when the binary is absent.
// Wording: owner to confirm.
const PrePushRefusal = "Broodforge: session %s does not push. Commit in this tree; Broodforge pushes at the handoff when the session ends."

// PrePushNoSession stands for the session id outside a session.
const PrePushNoSession = "this tree"

// CodePrePushMissing is worktree check's refusal when the tree no longer runs the pre-push hook.
const CodePrePushMissing = "pre-push-missing"

// gitPrePush is the hidden `broodforge git-pre-push` that share/hooks/pre-push runs: the
// refusal line on stderr and exit 1, always; git prints its own failed-to-push line after it.
var gitPrePush = Command{
	Name:   "git-pre-push",
	Class:  ClassStandalone,
	Hidden: true,
	Local:  runGitPrePush,
}

func init() {
	commands = append(commands, &gitPrePush)
}

// PrePushLine is PrePushRefusal for session id, PrePushNoSession when id is "".
func PrePushLine(id string) string {
	if id == "" {
		id = PrePushNoSession
	}
	return fmt.Sprintf(PrePushRefusal, id)
}

func runGitPrePush(_ context.Context, env *Env) error {
	fmt.Fprintln(env.Stderr, PrePushLine(os.Getenv(hook.EnvSession)))
	return &exitError{ExitFailure}
}
