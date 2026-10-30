---
date: 2026-10-30
kind: bug
related: [2026-10-30-a-store-failure-inside-a-turn-is-logged-as-a-permission-decision.md]
---

# A transcript decision line names neither its call nor its depth

The session layer emits a call's decision just before its result (`internal/session/turn.go`, the
`EventPermissionDecision` add near line 591), so an Agent call's `allow (policy)` lands after its
subagent's work. `internal/discord/transcript.go` prints a decision as `  - <decision> (<by>)` (line
85) with no call and no depth, and prints subagent text unindented (line 71). A reader of the
transcript cannot tell which call a decision belongs to.

Seen in `tests/discord/golden/subagent-end/transcript-s0001.md` (untracked until P32 commits): a
second `  - allow (policy)` stands alone after "No findings.". Not a golden defect: P32 was barred
from changing the renderer. Found by two reviewers, both pre-existing, neither fixed.

Reproduce: `go test ./internal/discord` has no case for it; read the golden above, or run
`session tail` on a session with an Agent call.

Plan: tester, after P32's commit. In `internal/discord/transcript.go` print a decision as
`  - <decision> (<source>) for <call>` at the call's depth (the event carries `Tool`, `Target`,
`Parent`, `Depth`; the call's line is `session.TraceLine(ev.Tool, ev.Target)`), and indent subagent
text at its depth. Then re-bless with `go test ./internal/discord -run TestGolden -update` (or
`discord render --golden <dir> --update`) and read `git diff tests/discord/golden/`: the golden to re-bless is
`tests/discord/golden/subagent-end/transcript-s0001.md`, and any other golden whose diff shows a
decision line must be read, not accepted blind. Verify: `go test ./internal/discord`; the
`subagent-end` transcript shows the decision beside its call. Wording of the new line is `04-ux.md`
territory: put it on the owner's list in `docs/HANDOFF.md`.
