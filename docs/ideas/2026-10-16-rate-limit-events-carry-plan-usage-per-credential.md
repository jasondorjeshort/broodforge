---
date: 2026-10-16
kind: feature
related: []
---

# Rate-limit events carry plan usage per credential

The S1 spike run showed that every `rate_limit_event` line in the engine's stream-json carries
`unifiedWindows` with `five_hour` and `seven_day` utilization and `resetsAt`, for the account the
engine runs as. The values matched the status-line meters. A session's plan usage is therefore
observable from the stream itself, per credential, with no `/usage` call and no separate poll.

Anchors: the usage paragraph of `docs/design/decisions/0020-harness-language-library-toolchain.md`
(usage is taken from `result.modelUsage` deltas, which counts tokens but says nothing about plan
meters) and the token tracking in `docs/design/03-architecture.md`.

Possible uses:

- The per-credential meter of R-44 and R-45 reads the latest event instead of estimating.
- A future usage cap (R-46) acts on real utilization rather than a token count.
- The `usage-gate` hook gets a source that does not depend on the status line being fresh.

## Open

- Whether the event is emitted on every turn or only near a limit; every event seen carried the windows, but how often events appear is not yet known.
- Whether `unifiedWindows` is stable across engine versions, or needs a pin and a fake-engine field.

2026-10-17: the owner's first real P7 turn (one Read, engine 2.1.296) carried one `rate_limit_event`
after the first assistant message, `status: allowed_warning`, `seven_day` 0.68, `five_hour` 0.09,
plus `overageStatus` and `overageEnabled`. So it is emitted on an ordinary turn, not only near a
limit; the per-turn frequency question above is half answered.

2026-10-30 (architect): designed. `03-architecture.md` (The engine adapter) has the `rate-limit`
kind; ADR 0020 and ADR 0021 carry dated amendments (the meter beside the token record; the
`credential_meter` table, one row per credential, replaced in place); `05-debug-interface.md` has
the reads (`credential show --json`'s `meter`, `usage --credential`'s `meter` line). The second
Open item is answered by the never-refuse rule: an absent field is its zero value beside a `seen`
time, so a pin that drops the windows is visible, not fatal.

Plan: coder, pass P28 of `../design/08-implementation-plan-engine-stream.md`, which is the brief:
`internal/engine/events.go` and `parse.go` (the kind, the parse, `MarshalJSON`), `parse_test.go`,
the goldens regenerated and their diff read, the migration `0004-credential-meter.sql` and the
store's upsert, the write in `internal/session/turn.go`, `credential show --json`'s `meter` and
`usage --credential`'s `meter` line, `tests/repro/full-session.sh`'s new check, P3's Proof count in
`08-implementation-plan.md`. Proof: the pass's. Out of scope stays: any cap (R-46, the owner's),
Discord wording (the owner's), the `usage-gate` hook (Hatchery's; a carry-back once the meter
exists). Delete this file in the pass's commit.
