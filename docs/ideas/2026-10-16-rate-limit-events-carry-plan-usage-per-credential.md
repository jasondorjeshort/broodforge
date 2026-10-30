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

Plan: architect, an amendment, no code. Amend `docs/design/03-architecture.md` (`EngineEvent`, and
"Everything else is `unknown`") with a `rate-limit` kind parsed from `rate_limit_event`: `status`,
`rateLimitType`, `unifiedWindows.five_hour` and `seven_day` (`utilization`, `resetsAt`),
`overageStatus`, `isUsingOverage`; shape in `fixtures/engine/recorded-read/turn-1.jsonl` (engine
2.1.296), an absent field stays zero (never-refuse). Decide where the latest event per credential
lives (a table or columns on the credential row, an ADR 0021 amendment) and where it reads (`usage
--credential`, `credential show`, `status --json`); `usage_record` stays token-based (R-44). Record
a dated ADR 0020 amendment and the coder pass in `08-implementation-plan.md` with proof command and
golden regeneration (`-update`). Out of scope: any cap (R-46, the owner's), Discord wording, the
`usage-gate` hook (Hatchery's; carry-back once the meter exists).
