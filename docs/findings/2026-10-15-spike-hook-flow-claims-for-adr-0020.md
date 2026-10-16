---
date: 2026-10-15
status: open
area: spike / engine contract, hook flow (ADR 0020)
supersedes: 2026-10-15-spike-claims-for-adr-0020.md
---

# C27, C39 and C26 hold on the PC: the defer round trip, the hook exit codes and `--settings` behave as ADR 0020 assumes; the droplet repeats are open

## Question

Do the hook-flow behaviours ADR 0020 assumes hold against the real engine?
`docs/design/07-spike-plan.md` turns the ADR's claims table into runs. This finding is the hook-flow
slice of the ADR 0020 half of the spike's output, split by topic from one finding, and it holds the
spike setup shared by all three: the login row is in
`2026-10-15-spike-login-claims-for-adr-0020.md`, the stream and process rows in
`2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`. It stays `open` while any row is
open, and when the last row closes it is `promoted_to` ADR 0020 with the rows copied into the ADR's
table. ADR 0022's claims are a separate finding.

| Claim | Host | Verdict | When |
|---|---|---|---|
| C26 `--settings` with `--setting-sources user` | PC | holds; the hook also runs with `--setting-sources ""` | 2026-10-16 |
| C27 `defer` round trip | PC, then droplet | holds on the PC; the deny reason reaches the model wrapped | 2026-10-16 |
| C39 hook exit codes | PC, exit 2 again on droplet | holds on the PC; exit 2 blocks, wrapped with the hook command; exit 1 and the timeout fall through | 2026-10-16 |

C27 and C39 on the droplet are separate runs of the same claims and are still open.

## What was tried

Shared setup (the other two findings point here):

- Host: the PC, Windows 11 Pro 10.0.26300, Git Bash (UCRT64), Node v22.23.2.
- Engine: `@anthropic-ai/claude-code` 2.1.296, the global npm install, launched as
  `AppData/Roaming/npm/node_modules/@anthropic-ai/claude-code/bin/claude.exe`. `$RUN --version-only`
  on 2026-10-16: `version 2.1.296 (Claude Code) (pin: match 2.1.296)`.
- Fixtures: `node spike/setup.js pc`, with `BF_SPIKE` set explicitly to the spike root so the
  agent's `TEMP` could not move it. `$S` is `AppData/Local/Temp/bf-spike`. It created `tree` on
  `main` and `canon` on `base` (both at `7295ef5 spike fixture: hello.txt`), wrote
  `cfg/settings.json` (281 bytes) and `spike-hook.json` (17 bytes), and left `cfg` untouched.
  `$CFG` is `$S/cfg`, where the login of the login finding put its credentials.

This finding's rows:

- C27, C39, 2026-10-16, same host, engine, `$FLAGS` and `--model haiku`, `--permission-prompts none`
  never added because no run waited for a prompt. C27 into
  `$S/fx/defer`, hook `defer`, `--mode prompt`: turn 1 "Create a file named hi.txt containing hi.";
  turn 2 `node spike/hook.js mode defer <id>:allow`, then `--resume last` with stdin closed and
  nothing written; turn 3 a fresh session, "Create a file named bye.txt containing bye." (a new name,
  so the model would not Read `hi.txt` first); turn 4 the same resume answered `deny`. C39 into
  `$S/fx/hook-exit2`, `hook-exit1` and `hook-sleep`, one turn each with the plan's prompt. Then
  `$RUN --replay` on each directory. Meters, shared with the S2 turns in the stream and process
  finding (ten engine processes between them): before 5-hour 19%, weekly 5%; after 22%, 6%, and
  the last turn's `rate_limit_event` said 0.22 and 0.06.
- C26, 2026-10-16, same host, engine and `$FLAGS`, `--model haiku`, two engine processes (four with C28's), no
  `control_request`, `--permission-prompts none` never added. Into `$S/fx/settings-a` and `settings-b`, hook `allow`: `node spike/setup.js
  settings --no-hook` (so `$CFG/settings.json` became `{}`), `node spike/setup.js settings --out
  $S/alt-settings.json` (the template's hook block only), then one Read turn with `--settings
  $S/alt-settings.json` and `--setting-sources user`, and one more with `--setting-sources ""` (the
  empty argument is on the recorded command line in `meta-1.json`); `node spike/setup.js settings`
  restored `$CFG/settings.json` (281 bytes). Meters, shared with the C28 turns in the stream and
  process finding and the C34 turn in the ADR 0022 finding: before 5-hour 25%, weekly 7%; after
  27%, 7%; the last `rate_limit_event` said 0.27 and 0.07.

## Result

C27 holds. Sessions `93ab3f0d-b9f2-40b7-8921-a95858da0251` (turns 1, 2) and
`c8922a32-fb6f-4ab5-ab21-e9c38b9b0bd4` (turns 3, 4); every process exited 0 in 3 to 4 s.

- Turn 1: the hook saw `Write` `toolu_01NmkmkiJgpriuU7yQrtDWYi` and deferred it; no ToolSearch,
  Haiku went straight to Write although the prompt names no tool. `result` `subtype` `success`,
  `stop_reason` `tool_deferred`, `is_error` false, `num_turns` 1, `permission_denials` `[]`,
  `deferred_tool_use` `{id, name: "Write", input}`. `hi.txt` absent. Turn 3 the same for `bye.txt`.
- Turn 2, resume with stdin closed: `PreToolUse` fired again with the same `tool_use_id`
  (`hook.log` `decided` `answer:allow`); the `tool_result` (`is_error` absent, which the parser treats as false, `File created
  successfully at: ...hi.txt ...`) was the third stdout line, before `system/init`, after a
  `command_lifecycle` and an `attachment` line (both `unknown`); `hi.txt` holds `hi`; the model
  answered in text; `result` `end_turn`, `num_turns` 1; `modelUsage` cumulative with turn 1. A
  resume of a deferred turn takes no input: neither the empty line nor stdin JSON was needed.
- Turn 4, resume answered deny: `tool_result` `is_error` true, text verbatim `PreToolUse:Write hook
  error: Denied by Broodforge: spike deny` (wrapped as C39 foresaw, no trailing newline, no command
  in brackets); the line also carries `tool_use_result` `Error: PreToolUse:Write hook error: Denied
  by Broodforge: spike deny` and `tool_result_meta` `[{id, non_execution_kind: "permission-rule",
  permission_decision: {decision: "reject", source: "hook", reason_type: "hook"}}]`;
  `result.permission_denials` lists the Write with its `tool_input`; the model continued to `result`
  `end_turn` (`I couldn't create bye.txt. A PreToolUse hook blocked the write with the message
  "Denied by Broodforge: spike deny." ...`); `bye.txt` absent. So 03's classification strips
  `PreToolUse:<tool> hook error: ` before its prefix test; no fallback is needed.

C39 holds: exit 2 blocks, exit 1 and the timeout do not. One session per turn, exit 0, 11.0 s, 7.1 s
and 38.3 s; the hook's stderr never reaches the engine's stderr (`stderr-1.txt` empty, all three).

- `exit2`: every call blocked (`Read`, then `Bash`; Haiku skipped the Write once the Read failed,
  `num_turns` 3). Each `tool_result` has `is_error` true and the text, verbatim, `PreToolUse:Read
  hook error: [node C:/Users/<pc user>/Documents/src/broodforge/spike/hook.js]: hook: spike exit 2\n`
  (the same with `Bash`): wrapped, and more than the plan foresaw. Between `hook error: ` and the
  stderr the engine inserts the hook's command as `[<command>]: `, and keeps stderr's trailing
  newline. `tool_result_meta` `reject`, `source` `hook`, `reason_type` `hook`; `permission_denials`
  lists both. Not a refutation: 03's strip is `PreToolUse:<tool> hook error: `, then an optional
  `[<command>]: `, then a trim, before the `hook:` test; that is the amendment 03 gets.
- `exit1`: non-blocking, `hook.log` shows all three calls answered `exit1`. The engine's `-p` default
  then: Read allowed (`tool_result_meta` `accept`, `source` `config`, `reason_type` `mode`); Write
  denied with no prompt and no `control_request`, `tool_result` `is_error` true, text `Claude
  requested permissions to write to C:\Users\<pc user>\...\tree\note.txt, but you haven't granted it
  yet.`, preceded by a `system/permission_denied` line (`tool_name`, `tool_use_id`, `message`;
  `unknown` to the parser), meta `reject`, `source` `config`, `non_execution_kind` `user-rejected`,
  listed in `permission_denials`; Bash `echo ok` allowed and run, output `ok`, meta `accept`,
  `source` `config`, `reason_type` `subcommandResults` (the engine's own safe-command list).
- `sleep`: the same three decisions by the engine's default, three `hook.log` lines `decided`
  `sleep`; gaps `tool_use` to `tool_result` 10.2 s (Read), 10.2 s (Write) and 21.2 s (Bash): Write
  and Bash came in one `assistant` message (two lines, one `message.id`) and the engine ran their
  hooks one after the other, each to the `timeout`. The timeout is the hook's hole as the plan put
  it, the default allowed `echo ok`: the plan's consequence (an audit event from the `hook.log` gap,
  a raised `timeout`) goes to ADR 0020 under C39.

C26 holds on both halves. Sessions `11092c22-4838-4626-8bcf-cce61cab09c4` (`--setting-sources
user`) and `3fa204e6-345a-4185-a522-59ea9219d5b2` (`--setting-sources ""`), one Read each, exit 0
in 4.9 s and 5.4 s, `stderr-1.txt` empty, the model answered `hello` both times.

- With `$CFG/settings.json` holding `{}`, each run added exactly one `hook.log` line (22 to 23,
  then 23 to 24): `tool_name` `Read` with the run's `tool_use_id`, `decided` `allow`. The hook could
  only have come from `--settings $S/alt-settings.json`, so `--settings` is honoured beside
  `--setting-sources user`, and an empty `--setting-sources` still runs hooks given by `--settings`.
  Nothing in the Decision changes; the critic's inline alternative (`--settings` with the JSON
  inline) stays open to the architect as a choice, not a blocker.

## Next

- The PC rows are all closed. The open rows: the droplet repeats of C27 and C39's exit 2 wait on
  the droplet operator entries.
- For ADR 0020's amendments: the `[<command>]: ` insert in the exit-2 wrapper (03's strip).
- Fixtures `basic`, `defer`, `hook-exit2`, `hook-exit1`, `hook-sleep`, `interrupted`, `parallel`
  (two turns), `settings-a` and `settings-b` (and `scrub`, the ADR 0022 finding's C34) are recorded
  and replay with no `unknown` for a kind 03 lists (new unknown kinds: `command_lifecycle`,
  `attachment`, `system/permission_denied`, `system/background_tasks_changed`, `system/task_started`,
  `system/task_notification`, `tool_progress`); two independent agents read every `turn-*.jsonl` (PC username in
  paths) and the fixtures were committed to `fixtures/engine/<name>/` 2026-10-16 (`50e0c40`; rule in `9accf2d`). `$S/tree` holds untracked `hi.txt` (C27).
  `$S/alt-settings.json` is left in place; `$CFG/settings.json` is the template again.
- `promoted_to` ADR 0020 when the last row closes.
