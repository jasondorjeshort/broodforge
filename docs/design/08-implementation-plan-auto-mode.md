# Implementation plan: the engine's `auto` permission mode as the third session mode

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the pass that builds ADR 0020's
amendment of 2026-11-01 and closes its claim C40. The pass reads `03-hook-contract.md` (Auto mode; What
the bot sees), `04-ux-session-controls.md` (Policy mode, and the third mode; The header's menus),
`05-debug-interface-session.md` and `../findings/2026-10-31-spike-auto-permission-mode-starts-under-p-hook-still-blocks-classifier-never-fired.md`
before its Implements line. Every word a person sees is a placeholder until the owner words it (04, Owner 7).

## P39. The third mode: config value, hook `pass`, spawn flag, switch and menu, the `engine` decider (medium; one real turn on the owner's word)

Implements: ADR 0020 Amendments 2026-11-01 and C40; `03-hook-contract.md` (Environment; Policy table; Auto mode; What
the bot sees; the `onFailure` block); `03-architecture.md` (Configuration `mode`; the auto paragraph under Prompt mode;
`TurnRequest.PermissionMode`); `04-ux-session-controls.md`; `05-debug-interface-session.md`; `05-debug-interface.md` (`hook try
--mode auto`, `--expect pass`); `01-requirements.md` R-26 to R-28. First step, before any other file: C40a, the decisive run of
the hook contract's Auto mode, on the owner's word (a `Write` under `-p --permission-mode default` with a hook that exits 0 silently);
an allow stops the pass at the finding. Files [no-check: the fixtures, script and tests are created by this pass]:
`internal/config/types.go` (`ModeAuto = "auto"`), `validate.go` (the `oneOf` of `defaults.mode` and of each channel's `mode`),
`setuptemplate.go` (the `defaults.mode` description names the third value); `internal/policy/policy.go` (`ModeAuto` in `Mode`,
`ParseMode` and `String`; a `Pass` decision; `Decide`: `CellAsk` under `ModeAuto` is `Allow` for a session-allowed tool, else `Pass`;
`CellWeb` under `ModeAuto` is `Deny`; no other branch changes); `internal/hook/hook.go` (`Pass`: exit 0, nothing on stdout, the
`hook.log` line `pass`; the chain runs as on an allow), `cases.go` (`expect` `pass`), `internal/cli/hook.go` (`--mode` and `--expect`
values; `hook try` prints `pass` with the log line's reason); `internal/spawn/spawn.go` (`"onFailure": "block"` on the `PreToolUse`
entry alone; the settings fixture updated); `internal/engine/events.go` (`TurnRequest.PermissionMode`; `Init.PermissionMode`),
`claudecode.go` (`Args`: `--permission-mode` from it, `default` when empty), `parse.go` (`system/init`'s `permissionMode`; a
`control_request` line as an `EngineEvent` of kind `prompt`: tool, input, `tool_use_id`, reason; one that does not parse stays
`unknown`); `internal/session` (the third value wherever `prompt|policy` is read: `start`, `setMode`, the row, the held-session load;
`PermissionMode` set at each process start from the session's mode, compared with `Init.PermissionMode`, a mismatch the
`engine-mode-mismatch` audit row and `internal-error` event of the hook contract; `classify.go`: `DecidedByEngine = "engine"` only
when the call's cell for the session's reach is `ask`, every other non-`hook` decider `hook bypassed` as today, the `failed; blocking
because onFailure` text a hook failure; shape A: on a `prompt` event interrupt the process through the backend, record the call
`deferred` with decider `engine`, publish `waiting-for-answer`, resume on the answer by id); `internal/session/classify_test.go`
(`TestClassifyAutoReadOnlyTimeoutIsBypassed`: a `Write` result with source `config` in an `auto` session with read-only reach is
`hook bypassed`, never `engine`; its code-reach twin under the tree is `engine`); `internal/discord` (`adapter.go`: the `/session mode`
choice `auto` and its bad-value refusal naming every mode; `gateway.go`: the option's description; the menu's third option, the
placeholder label, post and trace line of 04); `tests/hook/cases.jsonl` (every `ask`-cell row repeated under `auto` with `expect`
`pass`; a session-allowed `Bash` `allow`; `WebFetch` `deny`; every reach and credential deny `deny`; the `reviewer` and `land` rows
gain an `auto` `defer` only once P33 and P35 have built those cells, not before); `fixtures/engine/auto-refused/` (shape B: `init`
with `permissionMode` `auto`, a hook-passed Write refused by a `system/permission_denied` line and an error result) and
`fixtures/engine/auto-prompt/` (shape A: a `control_request` for a Bash call, then the resume's turn 2), both hand-written until the
real turn, with README rows; `tests/repro/auto-mode.sh`.

Debug surface: `hook try --mode auto --tool Write --input '{"file_path":"<tree>/a.go"}' --expect pass`; `hook cases tests/hook/cases.jsonl`;
`session start --channel 100 --as 1001 --roles 10 --mode auto --fixture auto-refused`, then `session show` (`auto` in the mode column) and
`received-1.json` (`permissionMode`, `args` with `--permission-mode auto`); `session mode s0001 prompt` printing `mode-changed`, the next
`received-<n>.json` carrying `default`; `session tail` showing the refused Write with decider `engine`, `audit` holding no `hook-bypassed` row;
`config check` clean on `"mode": "auto"`, an error naming the accepted values on any other word; the spawn's settings file carrying
`onFailure`; on dry-run `discord inject command --channel s0001 --user 1001 --roles 10 "/session mode auto"` and `discord inject click
--custom-id bf:s0001:mode --value auto`, the header edited and the post in the outbox.

Proof [no-check: the script is this pass's]: `CGO_ENABLED=0 go build ./...`, the check line, `CGO_ENABLED=1 go test -race ./...`; `hook cases
tests/hook/cases.jsonl` all `ok`; `scratch run -- bash tests/repro/auto-mode.sh` (a scratch root whose channel `100` has `"mode": "auto"`: the
Debug surface above line by line, both fixtures). Then, on the owner's word, one real turn on the dev instance (`run` skill) in an `auto`
session asking for a call the classifier stops (the finding's Next names three), recorded with `engine record`: it closes C40 by showing the
shape, the `source` value, `init`'s `permissionMode` (C40c), whether `modelUsage` carries a model of the classifier's own (C40b; then P38
owes its trailer) and, under shape A, whether the re-issued call keeps its id; the recording replaces the hand-written fixture of the shape
seen (the import rewrite, P1) and the other is deleted; the finding gets its row 3 and `promoted_to` ADR 0020, whose C40 row takes the
verdicts; an engine that waits with no result under shape A puts `--permission-prompts none` on `auto` processes alone (the amendment's rule).

Does not: change the decision order or the reach table; add `--permission-prompts` before C40 says so; fix the subagent loss (row 4 is
`../findings/2026-10-23-subagents-in-a-session-end-before-writing.md`); word anything; hold a process open through a human wait (shape A
falls back to shape B; the stdin route is the architect's next question); relay a shape B refusal as a prompt (the owner's pending call).
Depends on: P12b (the switch and the menu); P31 (the decision rows as they stand). Not on P33 or P35: their cells join the cases when built.
