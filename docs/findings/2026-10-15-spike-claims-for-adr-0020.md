---
date: 2026-10-15
status: open
area: spike / engine contract (ADR 0020)
---

# C25, S1, C27, C39 and S2 hold on the PC: the piped login, the stream shapes, the defer round trip, the hook exit codes and the tree kill all behave as ADR 0020 assumes

## Question

Do the engine behaviours ADR 0020 assumes hold against the real engine?
`docs/design/07-spike-plan.md` turns the ADR's claims table into runs. This finding is the ADR 0020
half of the spike's output, one row per claim; it stays `open` while any row is open, and when the
last row closes it is `promoted_to` ADR 0020 with the rows copied into the ADR's table. ADR 0022's
claims are a separate finding.

| Claim | Host | Verdict | When |
|---|---|---|---|
| C25 headless login with `CLAUDE_CONFIG_DIR` | PC | holds, piped form | 2026-10-15, again 2026-10-16 |
| C26 `--settings` with `--setting-sources user` | PC | open | |
| C27 `defer` round trip | PC, then droplet | holds on the PC; the deny reason reaches the model wrapped | 2026-10-16 |
| C28 parallel calls in prompt mode | PC | open | |
| C29 per-turn overhead | droplet | open | |
| C39 hook exit codes | PC, exit 2 again on droplet | holds on the PC; exit 2 blocks, wrapped with the hook command; exit 1 and the timeout fall through | 2026-10-16 |
| S1 stream shape and `modelUsage` delta | PC | holds; AskUserQuestion is absent under `-p` | 2026-10-16 |
| S2 Windows tree kill and resume | PC | holds, on the retry prompt | 2026-10-16 |

C25, C27 and C39 on the droplet are separate runs of the same claims and are still open.

## What was tried

- Host: the PC, Windows 11 Pro 10.0.26300, Git Bash (UCRT64), Node v22.23.2.
- Engine: `@anthropic-ai/claude-code` 2.1.296, the global npm install, launched as
  `AppData/Roaming/npm/node_modules/@anthropic-ai/claude-code/bin/claude.exe`. `$RUN --version-only`
  on 2026-10-16: `version 2.1.296 (Claude Code) (pin: match 2.1.296)`.
- Fixtures: `node spike/setup.js pc`, with `BF_SPIKE` set explicitly to the spike root so the
  agent's `TEMP` could not move it. `$S` is `AppData/Local/Temp/bf-spike`. It created `tree` on
  `main` and `canon` on `base` (both at `7295ef5 spike fixture: hello.txt`), wrote
  `cfg/settings.json` (281 bytes) and `spike-hook.json` (17 bytes), and left `cfg` untouched.
  Note the order: the owner ran the login before the fixtures, which is harmless because the `pc`
  step only ensures `cfg` exists and never rewrites it.
- C25, run by the owner because the code is theirs:
  `CLAUDE_CONFIG_DIR="$TEMP/bf-spike/cfg" node spike/login.js`, stdin, stdout and stderr all piped,
  which is the relay's shape. Run twice: 2026-10-15, and 2026-10-16 after the owner deleted `$S/cfg`.
- S1, 2026-10-16, hook mode `allow` (`node spike/hook.js mode allow`), `run.js`'s default
  `--model haiku`, `$FLAGS` as the plan lists them, three engine processes into `$S/fx/basic`:
  turn 1 `$RUN --cfg $S/cfg --cwd $S/tree --out $S/fx/basic --prompt "Read hello.txt and reply
  with its one word."`; turn 2 the same `--out` with `--resume last --prompt "Say the word
  again."`; turn 3 the same with `--resume last --prompt "Call the AskUserQuestion tool to ask me
  which colour I prefer, red or blue."`; then `$RUN --replay $S/fx/basic`.
- Meters: the plan's Cost step reads `~/.claude/usage-state.json`, which the coordinator session's
  status line writes for the account that session runs under. Before the group: 5-hour 15%, weekly
  4%; after: 16%, 4%. Whether that is the spike's account is in Result.
- C27, C39 and S2, 2026-10-16, same host, engine, `$FLAGS` and `--model haiku`, ten engine
  processes, `--permission-prompts none` never added because no run waited for a prompt. C27 into
  `$S/fx/defer`, hook `defer`, `--mode prompt`: turn 1 "Create a file named hi.txt containing hi.";
  turn 2 `node spike/hook.js mode defer <id>:allow`, then `--resume last` with stdin closed and
  nothing written; turn 3 a fresh session, "Create a file named bye.txt containing bye." (a new name,
  so the model would not Read `hi.txt` first); turn 4 the same resume answered `deny`. C39 into
  `$S/fx/hook-exit2`, `hook-exit1` and `hook-sleep`, one turn each with the plan's prompt. S2 into
  `$S/fx/interrupted`, hook `allow`, `--kill-after 5000`: turn 1 the plan's `sleep 30` prompt; turn
  2 the retry, "Run `ping -n 31 127.0.0.1` in Bash, then say done." (why: Result); turn 3 `--resume
  last --prompt "Continue."`, first attempt, no `CLAUDE_CODE_RESUME_INTERRUPTED_TURN`. Then
  `$RUN --replay` on each of the five directories. Meters: before 5-hour 19%, weekly 5%; after 22%,
  6%, and the last turn's `rate_limit_event` said 0.22 and 0.06.

## Result

C25 holds in its first form. No TTY was needed, and neither retry was used.

- The engine printed the authorize URL on its piped stdout, and `login.js` relayed it to the owner.
- `login: paste the code from the browser here and press Enter:` appeared; the pasted code reached
  the engine's stdin and the engine answered `Login successful.`
- `login: engine exited 0`.
- `login: auth status exit 0`, reporting `Login method: Claude Max account`.
- The directory listing carried `.claude.json` (56693 bytes), `.credentials.json` (524 bytes) and a
  `backups` directory.
- 2026-10-16, the owner's second run as himself, same piped form, no `--console`: `Login
  successful.`, `engine exited 0`, `auth status exit 0` reporting his Claude Max account,
  `.claude.json` 58397 bytes, `.credentials.json` 524 bytes, `backups`; then `setup.js settings`
  rewrote `cfg/settings.json`. C25 held a second time.

So the relay sentence in ADR 0020's Credentials section stands on Windows. The `--console` retry and
the direct terminal login, the ADR's other path, were not needed and remain untested on this host.

Whose account `$CFG` holds. The 2026-10-15 login signed in as the PC owner's Claude Max account,
not the project owner's. On 2026-10-16 the owner deleted `$CFG` and ran C25 again as himself, so
from then on the directory holds the project owner's login and every PC engine turn spends his plan.
Two observations about that account, identifiers compared and not printed: its `oauthAccount`
account, organization and email entries equal those in this PC's `~/.claude.json`, the login the
coordinator session runs under, so the `usage-state.json` meters above are the spike's account after
all; and the engine's own stream reports the plan meters of whatever account it runs as, see S1.
Both fixtures and the finding keep the PC owner's username in paths (`C:\Users\<pc user>\...` in
`system/init`'s `cwd` and the hook command), no email and no credential: the person who reads the
fixtures before they are committed should look for that.

S1 holds. Session `f0451bd7-4428-40b5-a52e-ad6b604c210a`, model id `claude-haiku-5-5`, the same
`session_id` on every `system/init` including the two resumes.

- Stream shape, kinds 03 lists, all seen with every field the parser reads (`missing fields 03
  reads: none` on all three turns): `system/init` with `session_id`, `model`, `tools` (a list without
  `AskUserQuestion`) and `capabilities`; `assistant` with `message.id`, `message.model`,
  `message.usage`, a `tool_use` block (`Read`, then `ToolSearch` in turn 3) and text blocks; `user`
  with `tool_result` (`toolu_01Cnjcxy6neu7wKgyKUezhFS`, `is_error` false, text `1\thello\n2\t`);
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

C27 holds. Sessions `93ab3f0d-b9f2-40b7-8921-a95858da0251` (turns 1, 2) and
`c8922a32-fb6f-4ab5-ab21-e9c38b9b0bd4` (turns 3, 4); every process exited 0 in 3 to 4 s.

- Turn 1: the hook saw `Write` `toolu_01NmkmkiJgpriuU7yQrtDWYi` and deferred it; no ToolSearch,
  Haiku went straight to Write although the prompt names no tool. `result` `subtype` `success`,
  `stop_reason` `tool_deferred`, `is_error` false, `num_turns` 1, `permission_denials` `[]`,
  `deferred_tool_use` `{id, name: "Write", input}`. `hi.txt` absent. Turn 3 the same for `bye.txt`.
- Turn 2, resume with stdin closed: `PreToolUse` fired again with the same `tool_use_id`
  (`hook.log` `decided` `answer:allow`); the `tool_result` (`is_error` false, `File created
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

## Next

- The open rows above, in the plan's order: C28, C26 on the PC. C29 and the droplet repeats (C25,
  C27, C39's exit 2) wait on the droplet operator entries.
- For ADR 0020's amendments: the `[<command>]: ` insert in the exit-2 wrapper (03's strip); the
  engine's standalone-`sleep` block, which fires before the hook; the delta rule after a killed turn.
- `$CFG` holds the project owner's login as of 2026-10-16; every PC turn spends his plan, and the
  stream's `rate_limit_event` shows the meters after each turn.
- Fixtures `basic`, `defer`, `hook-exit2`, `hook-exit1`, `hook-sleep` and `interrupted` are recorded
  and replay with no `unknown` for a kind 03 lists (new unknown kinds: `command_lifecycle`,
  `attachment`, `system/permission_denied`, `system/background_tasks_changed`, `system/task_started`,
  `system/task_notification`, `tool_progress`); a person reads every `turn-*.jsonl` (PC username in
  paths) before they move to `fixtures/engine/<name>/`. In `interrupted`, turn 1 is the blocked-sleep
  case and turns 2 and 3 the kill and the resume. `$S/tree` holds an untracked `hi.txt` from C27.
- `promoted_to` ADR 0020 when the last row closes.
