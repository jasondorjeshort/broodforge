---
date: 2026-11-08
kind: chore
related: [2026-11-08-one-engine-process-per-session-not-per-turn.md, 2026-11-08-a-venue-thread-per-subagent.md]
---

# A concurrency pass over the bot: where it serialises what Go could run at once

The owner's steer (2026-11-08): goroutines are cheap and the point of the language; look for more ways
to spend them. Not sanity checked. Places seen this week where the bot waits in line:

- A turn's event pump holds the session lock (`l.mu`, `internal/session/turn.go`) through the store
  writes of every event; a slow disk stalls the stream read.
- The hook chain runs its children one after another (`internal/hook/chain.go`); each child is a
  process start of its own.
- `session answer`, `send` and the Discord gateway's clicks all serialise on the same session lock,
  so a venue click waits on a turn's event flush.
- The pusher's nine checks run in sequence: the `-race` suite (about three minutes), the hook cases, the
  repro scripts, the scan; most are independent.
- `importfixture`, `engine record` and the goldens are one at a time by design; fine.

The pass: measure first (`go test -race` already runs; add `pprof` to `serve` behind a flag, and a
timing line per turn in the trace), then take the three cheapest wins (the store writes off the pump
goroutine through a bounded channel; the hook chain's independent children in parallel where their
order is not promised; the pusher's independent checks in parallel), each with a race test. A promised
ordering (03-hook-contract: the child chain's order) is not changed without its ADR.

Anchors: `internal/session/turn.go` (`l.mu`), `internal/hook/chain.go`, `internal/discord/gateway.go`,
`.claude/agents/pusher.md` (base-owned: a carry-back for its checks' order).

## Open

- Which orderings are promised and which are accidents; the hook contract names some.
- Whether the SQLite store (single writer) is the real ceiling, which no goroutine count fixes.
