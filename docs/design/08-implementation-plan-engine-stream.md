# Implementation plan: the engine stream's other lines (rate limits, unknown events)

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the passes that widen what the
adapter reads from the engine's stream-json after P3: the `rate_limit_event` line as a `rate-limit`
event with a per-credential meter (P28) and the shrink of every other unknown line (P29), both from
the 2026-10-29 triage. Each pass reads `03-architecture.md` (The engine adapter) and ADR 0020's
Amendments of 2026-10-30 before its Implements line. The recorded fixtures are the test: a pass here
regenerates the parser goldens with `go test ./internal/engine -run TestGolden -update` and reads the
diff before committing it; a golden line that changes for any reason but the pass's own is a bug.

## P28. The `rate-limit` event and the credential meter (small)

Implements: `03-architecture.md` (The engine adapter: the `rate-limit` kind of `EngineEvent`, the
stream subset); ADR 0020 Amendments 2026-10-30 (the meter beside the token record); ADR 0021
Amendments 2026-10-30 (`credential_meter`); `05-debug-interface.md` (Credentials: `credential show`'s
`meter`; Usage and audit queries: the `meter` line of `usage --credential`). Files [no-check: the
migration is created by this pass]: `internal/engine/events.go` (`KindRateLimit` `"rate-limit"`;
`RateLimitEvent{Status, RateLimitType string; FiveHour, SevenDay Window; OverageStatus string;
IsUsingOverage bool}` and `Window{Utilization float64; ResetsAt int64}`; `MarshalJSON`'s case, the
JSON `{"type":"rate-limit","status":..,"rateLimitType":..,"fiveHour":{"utilization":..,"resetsAt":..},
"sevenDay":{..},"overageStatus":..,"isUsingOverage":..}`); `internal/engine/parse.go` (`rawLine` gains
`rate_limit_info` with those fields and `unifiedWindows.five_hour` and `seven_day`; a `type` of
`rate_limit_event` is one `rate-limit` event; an absent field is its zero value, `seven_day_overage_included`,
`overageDisabledReason`, `utilization` and `resetsAt` at the top level are ignored, no new
`DisallowUnknownFields`); `internal/engine/parse_test.go` (the line of `fixtures/engine/recorded-read/turn-1.jsonl`,
`allowed_warning`, seven-day `0.68`, and one with no `unifiedWindows`); every `fixtures/engine/*/events-<n>.jsonl`
regenerated (each changed line was an `unknown` whose `raw` began `{"type":"rate_limit_event"`, nothing else);
`internal/store/migrations/0004-credential-meter.sql` (the ADR 0021 amendment's table: `credential_id`
primary key referencing `credential`, `seen_at`, `session_id`, `status`, `rate_limit_type`,
`five_hour_utilization`, `five_hour_resets_at`, `seven_day_utilization`, `seven_day_resets_at`,
`overage_status`, `is_using_overage`, the parsed values as read); `internal/store/` (the upsert and the
read by credential id); `internal/session/turn.go` (on a `rate-limit` event in a turn's stream, the
upsert for the session's credential in its own short write transaction, the turn's event loop
unchanged otherwise; a store failure is a `store: <error>` event summary as `subagentResult` writes
one, never the turn's end); every switch on `Kind` gains its `KindRateLimit` case for `exhaustive`:
`internal/cli/engine.go` (`runEngine`'s output switch), `internal/session/turn.go` (the turn loop),
`internal/session/classify_test.go` and `internal/engine/parse_test.go`; `internal/cli/engine_test.go`
(its `unknown` count on `basic --turn 1` drops by the parsed line); `internal/credentials/` and `internal/cli/credential.go` (`credential
show --json` carries `meter`: the row with `fiveHour` and `sevenDay` objects, `seenAt` and `session`,
`null` before any event; the text form unchanged); `internal/cli/usage.go` (with `--credential`,
after the `total` line, `meter  <five-hour utilization>  <seven-day utilization>  <status>  <seen at>`
or `meter  -`; `--json`: `meter` as `credential show`'s); `tests/repro/full-session.sh` (after the
turn: `usage --credential cred-scratch` printing `meter  0.16  0.04  allowed  <time>`, the values of
`fixtures/engine/basic/turn-1.jsonl`); `08-implementation-plan.md` (P3's Proof count of `unknown`
lines on `basic --turn 1` reads 2 from this pass, the `rate_limit_event` line now parsed). Debug
surface: `engine run --fake fixtures/engine/recorded-read --turn 1 --prompt x | grep '"type":"rate-limit"'`,
`credential show <id> --json`, `usage --credential <id>`. Proof: that `grep -c` gives 1 and the line
holds `"sevenDay":{"utilization":0.68,"resetsAt":1792728000}` and `"status":"allowed_warning"`; `go test
./internal/engine` with the goldens regenerated and the diff read as above; `scratch run -- bash
tests/repro/full-session.sh` exit 0 with its new line; a store test that two events for one credential
leave one row, the later one's; `credential show cred-scratch --json` on the kept root prints `meter`
with `"sevenDay":{"utilization":0.04,...}`. Does not: cap anything (R-46, the owner's); show the meter
in Discord (04, the owner's); write the meter from `engine run --real` (standalone, no store writer);
touch the `usage-gate` hook (Hatchery's, a carry-back once the meter exists); audit the event.
Depends on: P3, P5, P10.

## P29. The unknown event shrunk to its type, subtype, length and head (small)

Implements: `03-architecture.md` (The engine adapter: `Unknown` as a shape, the stream subset's
last sentence); `05-debug-interface.md` (A turn against the fake engine: `engine run`'s unknown
line). Files: `internal/engine/events.go` (`Unknown *UnknownEvent{Type, Subtype string; Bytes int;
Head string}`; `MarshalJSON`'s case, the JSON `{"type":"unknown","eventType":<type>,"subtype":<subtype>,
"bytes":<n>,"head":<head>}`); `internal/engine/parse.go` (`ParseLine`'s `unknown` value: `Type` and
`Subtype` from the line when it decodes, empty when it is not JSON or not an object; `Bytes` the
trimmed line's length; `Head` its first 200 bytes cut back to a rune boundary, the whole line when
shorter); `internal/engine/parse_test.go` (the unknown-line table: a `commands_changed` line from
`fixtures/engine/basic/turn-1.jsonl`, `bytes` its length and `head` 200 bytes; a line that is not
JSON; a JSON array; a short unknown object whose `head` is the whole line); every
`fixtures/engine/*/events-<n>.jsonl` regenerated (each changed line was an `unknown`, now short);
`internal/cli/engine.go` and `internal/session/turn.go` unchanged (they switch on the kind and
marshal). Debug surface: `engine run --fake fixtures/engine/basic --turn 1 --prompt x | grep
'"type":"unknown"'`. Proof: that `grep` prints two lines, each under 400 bytes with a `head` of at
most 200 bytes, each with `"eventType":"system","subtype":"commands_changed","bytes":25626`; `go test ./internal/engine` with
the goldens regenerated and the diff read; the `engine record` tests of `internal/engine/` pass
unchanged, since the recorder writes the raw line before the parser sees it (`ClaudeCode.Raw`), and
a test asserts that a recorded `turn-<n>.jsonl` holds the full `commands_changed` line. Does not:
add a `commands_changed` kind (the bot reads nothing in it); drop any line from a recording; change
`rate_limit_event`, which P28 parses. Depends on: P3; after P28, so the goldens move once each.
