# Run the three real-engine checks of ADR 0025

**Run the real-engine turns that prove checks C2, C5 and C6 of ADR 0025.** They wait on the owner
because each spends the owner's account on real engine turns: the owner runs them, or delegates
them in so many words. The checks and their expected results are owned by
`docs/design/decisions/0025-refusals-and-checks.md` (rows C2, C5, C6) and the Proof lines of P18 and
P19 in `docs/design/08-implementation-plan-session-workflow.md`.

Runs as: the owner, Git Bash in `<repo>`. First: stop the running bot, rebuild with
`CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`, and start `serve` again from `<root>` (the
running binary predates P16). Time: about 30 minutes.

1. C2 (P18), the chain fixture. The root needs `defaults.mode` `policy` (in prompt mode the Write
   is deferred and no `PostToolUse` runs) and `engine.gitBashPath` set. Make an empty `<tmp tree>`, then:

```
bin/broodforge --root <root> engine run --real --credential <id> --workflow tests/hook/chain --cwd <tmp tree> --prompt "Create notes.txt containing hello."
```

   It took when the stream and the transcript show `chain fixture: context from context.sh` and
   `chain fixture: stderr from post-exit2.sh`, `notes.txt` exists (neither blocked), and
   `<root>/sessions/engine-run/home/hook.log` holds two `"outcome":"context"` lines.

2. C5 and C6 (P19), in this repository's Discord channel with the real engine. In
   `<root>/config/projects/broodforge.json` set `"workflow": {"hooks": ["PreToolUse", "PostToolUse"]}`,
   then `bin/broodforge --root <root> config check` (ok). Start a session and note `<id>`.
3. Read the header: `bin/broodforge --root <root> session show --header <id>`. It holds no trailer
   list. Ask the driver for a one-line commit, then `bin/broodforge --root <root> worktree check <id>`
   (clean).
4. C6: ask the driver to launch the `reviewer` to review HEAD. It took when
   `session tail <id> --all` shows the reviewer's calls indented under `[Agent]`,
   `bin/broodforge --root <root> --json audit --kind permission-decision --session <id>` gives those
   tool_use ids with `"source":"hook"`, and `session show <id>` prints `review: toolu_... after <sha>`.
5. C5: ask the driver to launch `reviewer` and have it create `c5.txt` with the Write tool, not Bash.
   It took when the reviewer reports it has no Write tool and no indented `[Write]` line appears in
   `session tail <id> --all`. This repository has no agent with `tools: Read` alone, so this is
   not the exact companion case; that needs such an agent file on `main`. Say so when marking the row.
6. End the session. It took when nothing lands on `main` (`git log main` is unchanged).

When done: mark C2, C5 and C6 verified in `docs/design/decisions/0025-refusals-and-checks.md` and the
matching Proof lines in `docs/design/08-implementation-plan-session-workflow.md` (noting the C5
limit above), then delete this file and its row in `docs/operator/README.md`.
