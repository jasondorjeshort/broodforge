---
date: 2026-11-08
kind: feature
related: [2026-11-07-a-prompt-mode-process-that-never-prints-result-has-no-bound.md]
---

# Engine conformance as a command, for this engine and the next

Moonshot, not sanity checked. The bot's contract with the engine is now written down as claims
(ADR 0020's C25 to C39, C40, ADR 0029's C41 to C51) with a recording for each
(`fixtures/engine/*`, their `fixture.json` claims, `TestPromptFixtures`, the goldens). Each engine
upgrade so far was checked by hand: a spike brief, the owner's word, a finding. Engine 2.1.319
changed three stream shapes against 2.1.296 (the PostToolUse `attachment` line, no `thinking_tokens`,
`seven_day` alone) and forced `auto` back to `default` under the scrub; nothing in the bot would have
said so before a live session did.

The idea: `engine conform --real --credential <id>` re-records the fixture set's turns under the
installed engine on the owner's word, runs the parser and the fixture checks over the recordings, diffs
each stream's shape (line kinds and the fields the bot reads, not ids or text) against the committed
fixture, and prints one line per claim: holds, changed how, failed. `setup engine` refuses an engine
whose conformance has not run, the way it refuses a mismatched pin today (`mismatch-engine`). The
owner's role: the word and the real-turn cost, the same as a spike's.

The longer reach (the owner, 2026-11-08): Broodforge should work on more than one engine one day. The
claim list and its recordings are then the engine contract, and a second engine (another vendor's
CLI, or a later major version of this one) is adopted by making it pass the same command, with its own
adapter behind `internal/engine`'s interface and its own fixture directory. The hook contract
(`03-hook-contract.md`) is the hardest part to port: a second engine needs a PreToolUse-equivalent and a
prompt-tool-equivalent, or the policy moves into the adapter.

Anchors: `internal/engine/record.go`, `internal/engine/parse.go` and the golden test;
`fixtures/engine/README.md` (the claims); `internal/cli/setup.go` (`mismatch-engine`);
`docs/design/07-spike-plan.md` (the conventions a conformance run inherits); `docs/design/06-platforms.md`.

## Open

- Which fields count as "shape": a list per line kind in the parser, or the golden's event fields.
- The cost per run (today's set is about 25 turns) and whether a subset is enough for a point release.
- The engine interface's seams for a second vendor: a proposal before any code (a major choice).
