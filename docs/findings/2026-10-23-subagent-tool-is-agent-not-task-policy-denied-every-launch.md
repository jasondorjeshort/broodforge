---
date: 2026-10-23
status: resolved
area: hook / policy
---

# The subagent launch tool is `Agent` on the pinned engine; the policy knew only `Task` and denied every launch

## Question

Does the bot's PreToolUse policy allow a session to launch a subagent on the pinned engine?

## What was tried

The critic of proposal 0025 probed the hook input and the stream on the pinned engine (Claude Code
2.1.309) and reported the source of this observation in objection 2 of
`docs/design/proposals/0025-critic-review.md`. `internal/policy` was read against it: `ClassOf`
and the class table keyed `ClassAgent` on `TodoWrite` and `Task` only.

## Result

- The launch tool arrives as `Agent` in the hook input; `init.tools` in the stream still lists
  `Task`. `Agent` fell to `ClassOther`, which every reach denies, so every subagent launch in a
  session was denied.
- No fixture held an `Agent` call, so no test saw it (`03-hook-contract.md` already noted that
  no spike fixture held a subagent call).
- Fixed: `internal/policy/policy.go` classifies `Agent` beside `Task` (kept for an older pin), in
  `ClassOf` and in `ClassRows`; the table test in `policy_test.go`, the reach-report expectation in
  `internal/cli/serve_test.go` and four `Agent` cases in `tests/hook/cases.jsonl` cover it;
  `03-hook-contract.md` names `Agent` in its table row and in the "How the engine runs it" text.
  The class set, the reach table and the hook's output shape are unchanged.
