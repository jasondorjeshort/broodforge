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
