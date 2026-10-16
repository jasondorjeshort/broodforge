---
date: 2026-10-15
status: open
area: spike / engine contract (ADR 0020)
---

# C25 and S1 hold on the PC: the piped login takes the code on stdin; the stream has 03's shapes and `modelUsage` is cumulative

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
| C27 `defer` round trip | PC, then droplet | open | |
| C28 parallel calls in prompt mode | PC | open | |
| C29 per-turn overhead | droplet | open | |
| C39 hook exit codes | PC, exit 2 again on droplet | open | |
| S1 stream shape and `modelUsage` delta | PC | holds; AskUserQuestion is absent under `-p` | 2026-10-16 |
| S2 Windows tree kill and resume | PC | open | |

C25 on the droplet is a separate run of the same claim and is still open.

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

## Next

- The open rows above, in the plan's order: C27, C39, S2, C28, C26. C29 and the droplet repeats
  wait on the droplet operator entries.
- `$CFG` holds the project owner's login as of 2026-10-16; every PC turn spends his plan, and the
  stream's `rate_limit_event` shows the meters after each turn.
- The `basic` fixture is recorded and replays clean; a person reads the three `turn-*.jsonl` (PC
  username in paths) before they move to `fixtures/engine/basic/`.
- `promoted_to` ADR 0020 when the last row closes.
