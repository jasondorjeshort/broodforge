---
date: 2026-10-23
status: resolved (P19, ADR 0025)
area: engine / sessions / system prompt
promoted_to: docs/design/decisions/0025-session-workflow-composition.md
---

# The engine keeps the first launch's system prompt, so the per-turn trailer list is stale from turn two

## Question

Broodforge passes `--append-system-prompt` on every turn (a `--resume` launch each time) with
the trailer lines a commit must carry. Does a resumed session see the text of the latest launch, or
the first one?

## What was tried

Not run by the author of this finding. The critic probed it on the pinned engine (Claude Code
2.1.309) for the review of proposal 0025: row V3 and objection 8 of
`docs/design/proposals/0025-critic-review.md`. A session launched with
`--append-system-prompt "The snapshot word is ALPHA."` was resumed with the same flag set to BRAVO;
the model still answered ALPHA. The engine's `--help` adds "until the conversation is compacted".

## Result

- The engine keeps the system prompt rendered at the session's first launch. A later launch's
  `--append-system-prompt` is ignored on resume. After a compaction the prompt is rendered again
  from the flags of the launch that follows it, at a moment nobody chose.
- Broodforge builds a new append every turn: `internal/session/turn.go:230` calls
  `SystemPromptAppend(...)` with the current trailer lines; `internal/session/prompt.go:37-51` writes
  those lines under "End every commit message with exactly these lines:" (or "Commits need no
  Co-Authored-By lines yet."); `internal/engine/claudecode.go:67-68` passes it as the flag.
- Stale from turn two: the trailer list (a participant who joins after turn one is in no prompt's
  list) and the "no Co-Authored-By lines yet" sentence. Session id, project name, the driver (set
  once, `internal/session/service.go`) and the one-call-at-a-time rule do not change, so they are
  unaffected.
- `docs/design/03-architecture.md` (Attribution, around line 219) says the system prompt append
  lists the exact trailer lines; read per turn, that is wrong.
- Not determined: whether the append is re-rendered at a compaction from the latest launch's flags
  or the first (the critic's note says the launch that follows; no probe covered it).

Resolved by P19: the header built in `internal/session/prompt.go` holds no trailer list, so the
engine gets it once and nothing in it goes stale; `03-architecture.md` was corrected under ADR 0025.
