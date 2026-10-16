---
date: 2026-10-15
status: open
area: spike / engine contract, stream and process (ADR 0020)
supersedes: 2026-10-15-spike-claims-for-adr-0020.md
---

# S1, S2 and C28 hold on the PC: the stream shapes and the tree kill behave as ADR 0020 assumes; a parallel pair in prompt mode is deferred twice, not denied; C29 is open

## Question

Do the stream and process behaviours ADR 0020 assumes hold against the real engine?
`docs/design/07-spike-plan.md` turns the ADR's claims table into runs. This finding is the stream
and process slice of the ADR 0020 half of the spike's output, split by topic from one finding: the
login row is in `2026-10-15-spike-login-claims-for-adr-0020.md`, the hook-flow rows in
`2026-10-15-spike-hook-flow-claims-for-adr-0020.md`, which also holds the shared spike setup (host,
engine, fixtures, `$S`, `$CFG`). It stays `open` while any row is open, and when the last row
closes it is `promoted_to` ADR 0020 with the rows copied into the ADR's table. ADR 0022's claims
are a separate finding.

| Claim | Host | Verdict | When |
|---|---|---|---|
| C28 parallel calls in prompt mode | PC | holds (no hang, no crash); the engine's deny never fires: both calls deferred, `result` names the last, the first is dropped and re-issued on resume | 2026-10-16 |
| C29 per-turn overhead | droplet | open | |
| S1 stream shape and `modelUsage` delta | PC | holds; AskUserQuestion is absent under `-p` | 2026-10-16 |
| S2 Windows tree kill and resume | PC | holds, on the retry prompt | 2026-10-16 |

## What was tried

- Host, engine and fixtures: as in the hook-flow finding (the PC, engine 2.1.296, `$FLAGS` as the
  plan lists them, `$S` the spike root, `$CFG` its `cfg` directory).
- S1, 2026-10-16, hook mode `allow` (`node spike/hook.js mode allow`), `run.js`'s default
  `--model haiku`, `$FLAGS` as the plan lists them, three engine processes into `$S/fx/basic`:
  turn 1 `$RUN --cfg $S/cfg --cwd $S/tree --out $S/fx/basic --prompt "Read hello.txt and reply
  with its one word."`; turn 2 the same `--out` with `--resume last --prompt "Say the word
  again."`; turn 3 the same with `--resume last --prompt "Call the AskUserQuestion tool to ask me
  which colour I prefer, red or blue."`; then `$RUN --replay $S/fx/basic`.
- Meters: the plan's Cost step reads `~/.claude/usage-state.json`, which the coordinator session's
  status line writes for the account that session runs under. Before the group: 5-hour 15%, weekly
  4%; after: 16%, 4%. Whether that is the spike's account is in the login finding's Result.
- S2, 2026-10-16, same host, engine, `$FLAGS` and `--model haiku`, `--permission-prompts none`
  never added because no run waited for a prompt. Into `$S/fx/interrupted`, hook `allow`,
  `--kill-after 5000`: turn 1 the plan's `sleep 30` prompt; turn 2 the retry, "Run `ping -n 31
  127.0.0.1` in Bash, then say done." (why: Result); turn 3 `--resume last --prompt "Continue."`,
  first attempt, no `CLAUDE_CODE_RESUME_INTERRUPTED_TURN`. Then `$RUN --replay` on the directory.
  Meters, shared with the C27 and C39 turns in the hook-flow finding: before 5-hour 19%, weekly 5%;
  after 22%, 6%, and the last turn's `rate_limit_event` said 0.22 and 0.06.
- C28, 2026-10-16, same host, engine and `$FLAGS`, `--model haiku` throughout (Haiku emitted the
  pair on its first attempt, so the Sonnet attempt was not needed), no `control_request`,
  `--permission-prompts none` never added. Into `$S/fx/parallel`, hook `defer`, `--mode prompt`:
  turn 1 the plan's prompt; turn 2, outside the plan but inside the budget, `node spike/hook.js
  mode defer <b.txt id>:allow` then `--resume last` with stdin closed, to see what the other call
  of the pair gets. Meters, shared with the C26 turns here and the C34 turn in the ADR 0022
  finding: before 5-hour 25%, weekly 7%; after 27%, 7%; the last `rate_limit_event` said 0.27 and
  0.07.

## Result

S1 holds. Session `f0451bd7-4428-40b5-a52e-ad6b604c210a`, model id `claude-haiku-5-5`, the same
`session_id` on every `system/init` including the two resumes.

- Stream shape, kinds 03 lists, all seen with every field the parser reads (`missing fields 03
  reads: none` on all three turns): `system/init` with `session_id`, `model`, `tools` (a list without
  `AskUserQuestion`) and `capabilities`; `assistant` with `message.id`, `message.model`,
  `message.usage`, a `tool_use` block (`Read`, then `ToolSearch` in turn 3) and text blocks; `user`
  with `tool_result` (`toolu_01Cnjcxy6neu7wKgyKUezhFS`, `is_error` absent, which the parser treats as false, text `1\thello\n2\t`);
  `result` subtype `success` with `stop_reason` `end_turn`, `num_turns` (2, 1, 2), `permission_denials`
  `[]`, `usage` and `modelUsage` keyed by model id.
- Kinds outside 03's subset, parsed as `unknown` as 03 says and counted: `system/commands_changed`
  (two 25 KB lines in turn 1 only, the built-in command list), `rate_limit_event` (one per turn),
  `system/thinking_tokens` (two in turn 3). Replay of the directory: `grep -c '"type":"unknown"'`
  on `events-1..3.jsonl` gives 3, 1, 3; none is a kind 03 lists. One `thinking` block was dropped in
  turn 3 (03 keeps text and `tool_use` only).
- `modelUsage` is cumulative across resumes, `cumulative: consistent` on both resumes (no field of
  any model went down). Turn 1 `claude-haiku-5-5`: `inputTokens` 4, `outputTokens` 86,
  `cacheReadInputTokens` 42676, `cacheCreationInputTokens` 9946, `costUSD` 0.00245936. Turn 2:
  6, 90, 69046, 9983, 0.00273266 (delta 2, 4, 26370, 37, 0.0002733). Turn 3: 10, 351, 121895,
  10303, 0.00345605, `thinkingTokens` 114 (delta 4, 261, 52849, 320, 0.00072339). The `result`
  `usage` block holds the same totals plus per-iteration rows.
- AskUserQuestion under `-p`: no call at all. The tool is not in `init.tools`; the model called
  `ToolSearch` with `select:AskUserQuestion`, got the `tool_result` `No matching deferred tools
  found` (not an error), then answered in text: `I can't call AskUserQuestion here because that
  tool isn't available in this session. ... Red or blue? Which colour do you prefer?`, `stop_reason`
  `end_turn`, `permission_denials` `[]`. The hook saw `Read` and `ToolSearch` only. ADR 0020's
  consequence stands; nothing withdraws it.
- Seen in passing, for later rows. The engine emits one `assistant` line per content block: turn 3's
  `msg_011Cg5LsrqN25meutkqQR2SJ` arrived as a `thinking` line and a `tool_use` line with the same
  `message.id` and identical `message.usage`, so 03's deduplication by `messageId` is needed and a
  C28 message with two `tool_use` blocks may arrive as two lines sharing one id. Every
  `rate_limit_event` carries `unifiedWindows` with `five_hour` and `seven_day` `utilization` and
  `resetsAt` for the account the engine runs as (0.16 and 0.04 here, equal to `usage-state.json`),
  so the owner's plan usage is observable from the stream. Under `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`
  the hook's environment carried every `BROODFORGE_*` name and no `CLAUDE_CONFIG_DIR` (C34's
  question; the engine still ran against `$CFG`).

S2 holds, on the retry prompt.

- Turn 1 tested the kill but not the kill mid-turn: engine 2.1.296 refuses a standalone `sleep` in
  Bash before any hook runs (`tool_result` `is_error` true, `<tool_use_error>Blocked: standalone
  sleep 30. To wait for a condition, use Monitor with an until-loop ... use run_in_background: true
  ...`; no `hook.log` line for that id); Haiku re-ran it with `run_in_background` true (the hook saw
  that one) and reached `result` `end_turn` at about 4 s. The 5 s kill found the tree still alive:
  `38600 claude.exe, 57184 conhost.exe, 52484 bash.exe, 56924 bash.exe, 50184 conhost.exe`;
  `taskkill /T /F` SUCCESS on all five; none present after 2 s; exit 1, `result after interrupt`.
- Turn 2, the retry: no `result`; one `assistant` line, `msg_011Cg5MdoYQSNtGwfRBjjado`, usage input
  2, cache read 19026, cache write 7241, the `Bash` `tool_use` `toolu_01964i6u1W6nm1dgYCEuSSeA`;
  the tree at 5 s: `56840 claude.exe, 5140 conhost.exe, 47336 bash.exe, 55100 bash.exe, 56648
  conhost.exe, 32852 bash.exe, 33900 PING.EXE`; `taskkill` SUCCESS on all seven; `Get-Process -Id`
  on them after 2 s: none, `sameProcess` `[]`; exit 1; stdout ended at the `rate_limit_event`.
- Turn 3, `--resume last --prompt "Continue."`, first attempt: `system/init` with the same
  `session_id` `be129d98-0cf0-4c24-9acc-6d5896b3575e`; the model said `The earlier ping was
  interrupted before it returned, so I'm running it again.`, ran it, `result` `end_turn`,
  `num_turns` 2, 35.6 s. The killed turn's `tool_use_id` appears nowhere in turn 3's stream, and its
  usage is not in the resumed `modelUsage`: turn 3's cumulative cache write 1428 is below turn 2's
  interrupted row 7241, `cumulative: no (a field went down)`. So the engine's cumulative counts only
  turns that reached a `result`, and 03's "sum of the rows written" rule under-counts the turn after
  a kill (clamped `cacheCreationInputTokens` 0 here): that delta should be taken against the last
  `result`'s cumulative, the interrupted row standing as its own record. For the architect.

C28 holds on the plan's terms (the turn ends, exit 0 in 5.3 s, no hang, no crash) but not as ADR
0020 describes it: nothing falls through to the engine's deny. Session
`bc31c3ca-5ef2-4dbe-ae49-8387a22c3274`.

- Turn 1: Haiku's first attempt emitted two `Write` `tool_use` blocks in one message,
  `msg_011Cg5N7YgiSKbzVKCn8DFKz`, arriving as two `assistant` lines with that id and identical
  `usage`: `toolu_01B5b1rkmDCfPP4aQPdM3VCT` (`a.txt`) and `toolu_01FfytaZavRivXiQSPniauVu`
  (`b.txt`). `hook.log` shows `PreToolUse` fired for both, 0.6 s apart, each `decided` `defer`.
  Neither call got a `tool_result` (no `user` line at all); `result` `subtype` `success`,
  `stop_reason` `tool_deferred`, `num_turns` 1, `permission_denials` `[]`, and `deferred_tool_use`
  names only the second call (`b.txt`). Neither file was written. The engine's deny text for a
  parallel call does not exist here: there is nothing to quote.
- Turn 2, the resume with `b.txt` answered `allow`: the hook fired once more for
  `toolu_01FfytaZavRivXiQSPniauVu` only (`decided` `answer:allow`), its `tool_result` `is_error`
  absent (the parser treats a missing value as false), `File created successfully at: ...b.txt ...`; `b.txt` holds `b`. The `a.txt` call never
  fired the hook again and never got a result; the model issued a new `Write` for `a.txt`
  (`toolu_01DSwcaGCZMPUQgCp2cERJcp`), the hook deferred it, and the process ended `tool_deferred`
  naming that new id, `a.txt` still absent. The session transcript under `$CFG/projects` carries a
  `hook_deferred_tool` attachment for both original ids and a `tool_result` for `b.txt` only.
- So: in prompt mode a parallel pair is deferred call by call, the `result` carries the last
  deferral, and the first call is dropped, to be re-issued (new id) by the model after the resume.
  ADR 0020's sentence "parallel tool calls in prompt mode fall through to the engine's deny" and
  03's one-line notice describe a deny that does not happen; a hook-side rule (deny every call after
  the first of one message) would need the message id in the hook input, which the spike hook did
  not log, so whether the input carries it is unobserved. For the architect; the plan's refutation
  condition (a hang or a crash) was not met, so no fallback is taken here.

## Next

- The PC rows are all closed. The open row: C29 waits on the droplet operator entries.
- For ADR 0020's amendments: the engine's standalone-`sleep` block, which fires before the hook;
  the delta rule after a killed turn; the prompt-mode parallel pair (C28): no engine deny, the last
  deferral in `result`, the first call dropped and re-issued after the resume, so 03's "pending
  call" is one of two and the notice has nothing to report; whether the hook input carries the
  message id is the open question there.
- Fixtures `basic`, `interrupted` and `parallel` (two turns) are recorded and replay with no
  `unknown` for a kind 03 lists; the full fixture list and the unknown kinds are in the hook-flow
  finding. In `interrupted`, turn 1 is the blocked-sleep case and turns 2 and 3 the kill and the
  resume. `$S/tree` holds untracked `b.txt` (C28).
- `promoted_to` ADR 0020 when the last row closes.
