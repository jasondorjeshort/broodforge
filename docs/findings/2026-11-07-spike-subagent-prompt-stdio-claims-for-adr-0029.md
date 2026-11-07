---
date: 2026-11-07
status: open
area: spike / engine contract, a subagent's prompt on the stdio prompt tool (ADR 0029)
---

# A subagent's hook `ask` reaches the stdio prompt tool on 2.1.319: C42, C44, C45, C50 and C51 hold, C46 fails, option A stays shelved

## Question

ADR 0029 item 10 names the claims its build waits on: must hold C42 (foreground, then background), C44, C45 for a
subagent's `Bash`, C51; must be known C41b, C43a, C47, C49; cheaper if held C46, C50. A must-hold fail revives option A.
The claims are as worded in `docs/design/proposals/0029-subagent-prompt-in-prompt-mode.md` ("Claims to verify"); C51
came from the ADR's reviewer. C41a, C43b and C45 for a top-level `Write` are the critic's run
(`docs/design/proposals/0029-critic-review.md`), cited, not repeated. C48 is option A's only and was skipped.

| Claim | Verdict | Evidence (run directory) |
|---|---|---|
| C42 foreground | holds | a `can_use_tool` request with `agent_id` and the subagent's `tool_use_id`; the allow ran `echo hi` and the `Agent` result carries `hi` (`c42-allow`); a deny with `interrupt` false reached the subagent verbatim, it made its next call and reported (`c42-deny`) |
| C42 background | holds, with a finding | the request carries `agent_id`; the coordinator's first `result` came while it was open; the allow ran it and the engine began a second turn itself, a second `result` in the same process (`c42-bg`) |
| C44 | holds | hook `allow` on a `touch` ran, no request, `source` `hook`; hook `deny`, no request; hook `ask` on a `Read` and on a safe-listed `echo`, one request each (`c44-c51`; a subagent's `echo` in `c42-allow`); `--permission-prompts` off |
| C45, a subagent's `Bash` | holds | every `init` line (thirteen; `c42-bg` has two) `permissionMode` `default`, stderr empty on every run, every subagent `ask` reached the tool |
| C51 | holds | a top-level `defer` under the flag, stdin open: `tool_deferred`, `deferred_tool_use`, no request (`c44-c51`); the resume, stdin open and no line written, ran the call at once under the hook's `updatedInput`, `source` `hook` (`c51-resume`) |
| C41b | both outstanding | two requests 0.2 s apart, the first unanswered; answered second first, each answer applied to its own id (`c41b`) |
| C43a | a deny; the turn goes on | the tool result `Tool permission request failed: AbortError: Tool permission stream closed before response received`, `source` `config`; the subagent reported; `result` `end_turn`; exit 0.6 s later (`c43a`) |
| C47 | the call is dropped | the tree kill: exit 1, no `result`; the resume writes a synthetic `tool_result` for the coordinator's `Agent` call to the transcript only; the subagent's call is neither re-issued nor re-asked (`c47-kill`, `c47-resume`) |
| C49 | the subagent ends, the turn does not | the subagent `killed`; its `Bash` gets the engine's stock rejection, not the bot's `message`; the `Agent` result `[Request interrupted by user for tool use]`; the coordinator takes one more model call; `result` `end_turn` 2.2 s after the deny (`c49`, `c49-resume`) |
| C46 | fails under a hook `ask` (every subagent `Bash` in D) | after an allow carrying `updatedPermissions` (`addRules`, `Bash`, `session`) the next `Bash` under a hook `ask` was a request again; no request carried `permission_suggestions` (`c46-c50`) |
| C50 | holds | an allow with `updatedInput` `echo one-changed` ran that: output `one-changed`, `source` `user_temporary` (`c46-c50`) |

Option A does not revive: every must-hold claim holds.

## What was tried

- The PC, Windows 11, Git Bash, Node v24; engine 2.1.319 (`init` `claude_code_version`), the native install
  `~/.local/bin/claude.exe` started directly, no shell, under the owner's login. Twelve engine starts on 2026-11-07, the
  owner's go-ahead quoted in the brief ("Okay to spend the spike."). Coordinator `--model opus`; the subagent `probe`
  passed by `--agents <run>/agents.json` (Haiku, tools `Bash` and `Read`); `modelUsage` names both models.
- The command line, `internal/engine/claudecode.go` `Args` plus ADR 0029 rule 2 and the stub settings:
  `claude.exe -p --output-format stream-json --verbose --input-format stream-json --permission-mode default
  --setting-sources user --strict-mcp-config --mcp-config <repo>/share/empty/mcp.json [--resume <id>] --model opus
  [--agents <run>/agents.json] --permission-prompt-tool stdio --settings <run>/settings.json`.
- The environment as `internal/spawn` `Environment` builds it: every `CLAUDE*`, `ANTHROPIC_*`, `BROODFORGE_*`, git
  identity, git-local and agent-session name stripped; then the `BROODFORGE_*` hook names (mode `prompt`, reach `code`),
  a spike git identity, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, `DISABLE_AUTOUPDATER=1`. No `CLAUDE_CONFIG_DIR`, so the
  owner's user settings load: no permission rules, a `defaultMode` the flag overrides, and one `SessionStart` plugin
  hook whose two `system/hook_*` lines open every stream. Not reproduced: `--append-system-prompt-file`, `--effort`.
- Settings as `spawn.Settings`: `PreToolUse`, empty matcher, `timeout` 30, `onFailure` `block`, `node <stub> <run>`;
  `PostToolUse` the same stub, logging only. The stub logs its raw stdin (`hook-stdin.jsonl`) and prints
  `internal/hook` `Encode`'s line, the decision per run by rule (D's table: a subagent's `Bash`, `Write`, `Edit` `ask`,
  the coordinator's `defer`, the rest `allow`; C44's by command), the policy's reasons, `ask` ADR 0029's placeholder:
  `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":"Broodforge: asked a person in this session; if this reads as a refusal, no prompt reached them: say so and stop"}}`.
  C51's resume used the land cell's form (`docs/design/03-hook-contract.md`, Decision step 2): `allow`, the reach
  reason, `"updatedInput":{"command":"echo c51-replaced","description":"Print c51-original"}`.
- A Node driver holds stdin open, writes the user line, answers each request by its run's plan, closes stdin 3 s after
  `result` (C43a, C47 and `c42-bg` by plan instead) kills, where the plan says, with `taskkill /T /F` (the spawn's fallback; its job
  object terminate is the same hard kill). Per run, in `<scratchpad>/bf-spike-0029/<run>/`: the raw stream, stderr,
  hook stdin, the lines written (`sent.jsonl`), the driver log, the plan and the engine's transcripts; the driver and
  the stub in `bin/`.
- Meters: `seven_day` 0.36 before, 0.37 after; no rate-limit line carried `five_hour` (as the 2026-10-23 finding).

## Result

The request, the same shape on every run (ids redacted):
`{"type":"control_request","request_id":"<id>","request":{"subtype":"can_use_tool","tool_name":"Bash","display_name":"Bash","input":{"command":"echo hi","description":"Print hi via echo"},"description":"Print hi via echo","decision_reason":"Broodforge: asked a person in this session; if this reads as a refusal, no prompt reached them: say so and stop","decision_reason_type":"hook","decision_reason_hook_event":"PreToolUse","decision_reason_hook_source":"settings","tool_use_id":"<id>","agent_id":"<id>"}}`.
Its `agent_id` is the `task_started` `task_id` and the hook stdin's `agent_id`; hook stdin also has `agent_type`
(`probe`), the request does not. A coordinator's request has no `agent_id` key (`c44-c51`). No `init` field names the
prompt tool, so `permissionMode` and stderr stay the only signs a pin forced it off; neither moved.

The answers written, and what came back:

- Allow: `{"type":"control_response","response":{"subtype":"success","request_id":"<id>","response":{"behavior":"allow","updatedInput":{"command":"echo hi","description":"Print hi via echo"}}}}`.
  The `tool_result` `hi` under the subagent's `parent_tool_use_id`, meta `{"decision":"accept","source":"user_temporary"}`;
  `task_updated` `completed`; the `Agent` result, framed `[Subagent hand-back]`, carries the report, `canContinueAgent` true.
- Deny, `interrupt` false: `{"type":"control_response","response":{"subtype":"success","request_id":"<id>","response":{"behavior":"deny","message":"Denied by Broodforge: a person in this session denied this call","interrupt":false}}}`.
  The `tool_result` `is_error` true, its text the `message` verbatim with no `PreToolUse:` wrapper, `tool_use_result`
  `Error: <message>`, meta `reject`, `source` `user_reject`; no `system/permission_denied` line; `permission_denials`
  lists the call. The subagent's next `Bash` asked, was allowed and ran (C41a's open deny half, for a subagent).
- Deny, `interrupt` true (`c49`): the same envelope, `"message":"Denied by Broodforge: the session was interrupted","interrupt":true`.
  The `tool_result` is the engine's stock text (`The user doesn't want to proceed with this tool use. ... STOP what you
  are doing ...`), not the message; `task_updated` `killed`, `task_notification` `stopped`; the `Agent` result
  `[Request interrupted by user for tool use]`, meta `non_execution_kind` `interrupted`, `source` `hook`. The coordinator
  then answered in text (it could have called tools); `result` `end_turn`, `num_turns` 2, `permission_denials` the
  `Bash`. The transcript ends complete; `c49-resume` appended only the new turn, and the coordinator chose to relaunch
  the subagent, whose `Bash` asked again under a new id.
- Allow with a changed input and a rule (`c46-c50`): `"updatedInput":{"command":"echo one-changed","description":"Print the word one"},"updatedPermissions":[{"type":"addRules","rules":[{"toolName":"Bash"}],"behavior":"allow","destination":"session"}]`.
  Output `one-changed` (C50 holds). `echo two` was a request again (C46 fails; the `updatedPermissions` shape sent, `addRules`, `rules: [{toolName: Bash}]`, `behavior: allow`, `destination: session`, passes the engine's own validator, so the driver is not at fault); whether the rule was dropped or
  outranked by the hook's `ask` this run cannot tell.

C41b: the subagent put `echo alpha` and `echo beta` in one message; the hook ran for both before any answer; both
requests stood open together. `beta` was answered allow first, `alpha` deny 1 s later; `alpha`'s error result came
first, `beta`'s output 1.3 s after, each under its own id. Whether `beta` waited for `alpha`'s answer to start is not
separable here (a `Bash` start costs 1 to 4 s on this host).

C43a: closing stdin 3 s into the wait answered the request at once (meta `reject`, `source` `config`, `reason_type`
`other`); the subagent reported, the coordinator answered, `result` `end_turn`, exit 0 0.6 s after `result`. Every
later request in such a process can only abort the same way.

C47: the tree kill 3 s into the wait: exit 1, no `result`. On disk the coordinator's transcript ends on its `Agent`
`tool_use`, the subagent's on its `Bash` `tool_use`, the subagent's meta file without `finished`. The resume (`Carry on
with the task.`) wrote a `tool_result` for the `Agent` id into the transcript, `[Tool call interrupted: the session
ended before this call's result was recorded, so its outcome is unknown. Check whether it took effect before relying on
it or running it again.]`, then a synthetic assistant `No response requested.`; neither reached the stream, which
opened at `init`. The coordinator did not relaunch the subagent.

C51: the line types match `fixtures/engine/defer` turns 1 and 2 except for the plugin's two `system/hook_*` lines,
the asks run before the defer, and on the resume one more `attachment` (the `PostToolUse` hook, absent from that
fixture's settings). The one difference the open stdin brings: the process does not exit at `result`; it exits 0.5 to
0.6 s after stdin closes (C43b's shape), deferred turn and resume alike. A resume with stdin open needs no line.

Background (`c42-bg`): the `Agent` call with `run_in_background` true returned `Async agent launched` at once; the
subagent's request came 0.2 s before the coordinator's first `result` (`end_turn`), so at that `result` a request was
open and `background_tasks_changed` still listed the task. Answered 3 s after `result`, the call ran;
`task_notification` `completed`, `background_tasks_changed` `[]`, then a second `system/init`, a coordinator turn
quoting the report, and a second `result`. Under ADR 0029 rule 2 (stdin closed on `result`) that request would have
aborted as in C43a.

Fixture-worthy (ADR 0029 rule 9), each with its `hook-stdin.jsonl` for the rule's `hook-stdin-<n>.json`: `c42-allow`
(`prompt-ask-allow`); `c42-deny` (`prompt-ask-deny`, a deny then an allow in one subagent); `c41b` (`prompt-ask-two`,
answered out of order); `c44-c51` with `c51-resume` (the `defer` fixture under the flag, the resume's `updatedInput`
included, which item 10 asks the build pass to re-record); `c42-bg` (two `result` lines in one process, a request open
at the first); `c43a`, `c49`, and `c47-kill` with `c47-resume` (the abort, the interrupting deny, the kill and the
resume's unstreamed synthetic result: paths rule 7 and rule 8's detector meet). `prompt-ask-no-tool` has no recording
here: the 2026-10-23 finding's `control-ask` run (`<scratchpad>/bf-subdefer/`, the same engine, hook `ask`, no flag)
is its shape, with the defer reason where the `ask` placeholder now goes. `prompt-ask-session-allow` has none either:
`c46-c50`'s second request is that answer's shape, but its first carries a changed command. An import drops the
plugin's `system/hook_*` lines and rewrites home paths with the user segment (`cwd`, `transcript_path`, `file_path`, `output_file`, the plugin output; ADR 0018), or the build pass records afresh with `engine record --mode prompt`.

## Next

- The coordinator copies the rows into ADR 0029's record; the build pass imports the recordings above.
- For the architect: (1) rule 2's close on `result` aborts a background subagent's open request and cuts the engine's
  second turn; either close only when no request is open and the last `background_tasks_changed` is empty, or keep
  the header's foreground rule and treat a request open at `result` as rule 7's drop. `c42-bg` is also evidence for
  `2026-10-24-background-subagents-die-when-the-turn-ends.md`: with stdin open the engine did not exit at `result`
  and ran the notification turn itself. (2) C49: `interrupt` true ends
  the subagent, not the turn; the coordinator gets a model call that may issue tools, and the bot's `message` reaches
  nobody, so the subagent reads the engine's text. (3) C46 failed: D's in-process session allow stands as written.
  (4) `source` `user_reject` (deny) and `config` with `reason_type` `other` (abort) join `user_temporary` for 03 and
  the classifier.
- C47's synthetic `tool_result` is written at resume and never streamed: `session` records the interrupted `Agent`
  call itself, as for S2.
