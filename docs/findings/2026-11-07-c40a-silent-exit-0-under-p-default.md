---
date: 2026-11-07
status: resolved
area: spike / hook flow / permission modes
promoted_to: docs/design/decisions/0020-harness-language-library-toolchain.md
---

# A `PreToolUse` hook that exits 0 with nothing on stdout is no decision: the engine's default denied the Write

## Question

C40a of ADR 0020 (Amendments 2026-11-01; `03-hook-contract.md`, Auto mode): is a hook that exits 0
with nothing on stdout "no decision", so the engine's own permission check decides the call, or is
it an `allow`? Under `auto` the hook's `pass` is exactly that silent exit, so an `allow` would make
`auto` policy mode under another name and stop P39 at this finding.

## What was tried

- Host: the PC, Git Bash. Engine `2.1.319 (Claude Code)`, default model `claude-opus-5-5`, the
  owner's existing login, no scratch `CLAUDE_CONFIG_DIR`.
- Cwd `<scratchpad>/cwd` under the OS temp directory, empty; never the repository or a dev root.
- `<scratchpad>/settings.json` registered one `PreToolUse` command hook, empty matcher, timeout 30:
  `node <scratchpad>/hook.js`, which appends its stdin to `<scratchpad>/hook.log`, prints nothing
  and exits 0.
- Command: `claude -p --output-format stream-json --input-format stream-json --verbose
  --permission-mode default --settings <scratchpad>/settings.json`, one user line on stdin asking
  for one `Write` of a new file `c40a.txt` holding `hello`, and no other tool. No
  `--permission-prompts` flag.
- The raw stream, stderr and the hook log stay in the scratch directory.

## Result

A silent exit 0 is no decision. The hook ran (one `hook.log` line: `tool_name` `Write`,
`hook_event_name` `PreToolUse`, `permission_mode` `default`), and the engine's own default then
refused the Write with no prompt, exactly as C39 saw with no hook answering:

- `system/init`: `"permissionMode":"default"`.
- `{"type":"system","subtype":"permission_denied","tool_name":"Write","tool_use_id":"toolu_01Bf...","message":"Claude requested permissions to write to <scratchpad>\\cwd\\c40a.txt, but you haven't granted it yet."}`
- The `tool_result`: `"is_error":true`, the same text; its `tool_result_meta`:
  `[{"id":"toolu_01Bf...","non_execution_kind":"user-rejected","permission_decision":{"decision":"reject","source":"config"}}]`.
- `result`: `"subtype":"success"`, `"is_error":false`, `permission_denials` listing the Write with
  its input; `stop_reason` `end_turn`; exit 0, stderr empty.
- `<scratchpad>/cwd` empty afterwards: `c40a.txt` never landed.

So the decider of a passed call is the engine's (`source` `config` under `default`), and P39 goes
on. What `source` a passed call carries under `auto`, when the classifier decides, is still C40's
open part (row 3 of the 2026-10-31 spike finding).
