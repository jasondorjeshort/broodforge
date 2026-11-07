---
date: 2026-10-23
status: open
area: sessions / delegation
---

# Subagents in a Discord session end at a deferred call; skills, resume and the handoff are missing

## Question

Can a Discord session (s0008, project `scratch`, reach `code`, mode `prompt`, workflow injected
per ADR 0025 from `61cccdc`) follow the repository's delegation rule: brief an `architect`, let it
write a proposal, and continue it when it stops?

## What was tried

The driver asked for the engine's `auto` permission mode to be planned for the next wave. The
coordinator made three `architect` launches and one continuation, briefed to write proposal 0027 (then numbered 0026; ADR 0026 took the number on 2026-10-24)
(no code, no commit):

1. Background launch. The `architect` said the `proposal` skill was not registered and ended.
2. A `SendMessage` to continue it was refused by the bot's hook.
3. A fresh background launch told to read `.claude/skills/proposal/SKILL.md` as a file.
4. A foreground launch (`run_in_background: false`) told to write a first draft early.

Then a `reviewer` of this finding's commit, single-call probes, and, on the driver's word, steps 4
and 5 of `docs/operator/2026-10-23-real-turns-c2-c5-c6.md` as this session's work.

## Result

- Each `architect` launch made 25 to 30 tool calls in about 40 seconds and ended without writing a
  file. The report handed back was the agent's first sentence only ("I'll start by reading ...").
  The background run's output file was empty (0 bytes), so the last call cannot be seen.
- Foreground and background ended alike, so the end of the coordinator's turn (a process per turn)
  is not the cause. (A different loss, policy mode: `2026-10-24-background-subagents-die-when-the-turn-ends.md`.)
- A `reviewer` launched in the foreground to review this finding's commit (read-only tools: Read,
  Glob, Grep, Bash; told to report, not edit) ended after 2 tool calls in about 4 seconds, its
  report `(Subagent completed but returned no output.)`. It never wrote, so a deferred `Write`
  cannot be the only trigger.
- Single-call probes, each a foreground subagent told to make one call and quote its result:

  | Call | Agent | Hook's answer (inferred from the reach table) | Outcome |
  |---|---|---|---|
  | none (Write asked for, not in its `tools:`) | `reviewer` | none | full report; no Write tool listed |
  | Read of `CLAUDE.md` | `runner` | allow | full report, the line quoted |
  | Bash `git log --oneline -1` in the tree | `runner` | defer (prompt) | `returned no output` |
  | Write of a file in the tree | `tester` | defer (prompt) | `returned no output`, no file |
  | Write of a file outside the tree | `tester` | deny | full report quoting `Denied by Broodforge: Write on ... is outside this channel's reach (code)` |
  | Write of the same file in the tree, after the driver was asked to switch to policy mode | `tester` | allow | full report quoting `File created successfully`; the file held its line |

- In the probes an allow and a deny returned to the subagent and it carried on; a defer ended it.
  The four earlier runs fit that pattern (each would have reached a Bash or a Write in the tree
  early) but their last calls are not seen, so for them it is unconfirmed.
- The last probe's allow has two possible causes: the switch to policy mode, or a session allow of
  `Write` given by the driver at an earlier prompt. The mode is not visible from inside the
  session; the code order settles it (below).
- After the last probe, a `reviewer` of this commit ran 21 calls, Bash included, over about four
  minutes and returned a full report: the first subagent in this session to use Bash and finish.
- What ended: only the subagent. Its Agent result is `ok` with no output, while the engine turn
  went on: after the in-tree probes the audit shows no `turn-ended` and no `waiting-for-answer`
  for turn 8, the coordinator wrote text and ran the next probe, and the turn ended later (below).
  The engine returned no top-level deferred result for the subagent's call. The coordinator's own
  deferred calls (an `Edit` to `docs/ideas/`, a `Bash` commit) did run after their prompts. The
  audit shows no prompt for the subagent's call; the hook log shows the hook's answer (below).
- The coordinator's own deferred calls end its turn too: in prompt mode, after a prompted `Edit`,
  `Write` or `Bash`, the coordinator's next input was the bare line `Continue from where you left
  off.` instead of the call's context, and it had to re-establish where it was. Work survives
  (the call ran), but a multi-step plan is cut at every prompt.
- ADR 0025 check C2, the `additionalContext` half, seen live: after the coordinator's `Write` of
  this finding, the repository's `post-edit` hook ran as a `PostToolUse` child and its context
  reached the model (`post-edit: docs/findings/... is not listed in findings/README.md. Add one
  line.`). The exit-2 half and the chain fixture's proof (the operator file, step 1) are still open.
- ADR 0025 check C5, first evidence: the `reviewer` probe (the operator file's step 5) reported no
  Write tool, so an agent's `tools:` from `--agents` restricts it in a session. Not the exact
  companion case (an agent with `tools: Read` alone), and `session tail` is not yet read.
- ADR 0025 check C6, first evidence: the `tester`'s quoted `Denied by Broodforge: Write on ...`
  is the bot's hook answering a subagent's call (`03-hook-contract.md` calls this unobserved). The
  full proof (the operator file's step 4: `session tail` indentation, `"source":"hook"` rows,
  `session show`'s `review:` line) is still open.
- The Skill tool lists none of the repository's skills (`proposal`, `brief`, `commit`, ...): the
  expected gap of ADR 0025 option D, now seen to stop an agent whose instructions name a skill.
  Listing them would not be enough: `Skill` falls in the policy's `ClassOther`, a deny in every
  reach (`internal/policy/policy.go`).
- `SendMessage` is outside `code` reach (also `ClassOther`): `Denied by Broodforge: SendMessage is
  outside this channel's reach (code); nobody in this session can allow it`. A stopped agent can
  only be relaunched from scratch, at its full cost (three Fable runs, about 200k tokens, no
  output).
- The header's import of the handoff loads nothing: `@docs/HANDOFF.md in CLAUDE.md:6 was not
  loaded: docs/HANDOFF.md is not in the copy`. The copy holds `CLAUDE.md` and `.claude/` only
  (rule 5), so a session has no coordinator state.
- The driver typed `/effort high` in Discord. It reached the engine as a plain message: `/effort`
  did not run. The bot's `/session effort` exists.
- `auto` is not implemented: `mode` accepts `prompt` and `policy` only and `Args` passes
  `--permission-mode default` (`internal/engine/claudecode.go`). The driver's direction is in
  `docs/ideas/2026-10-23-engine-auto-permission-mode-as-a-third-session-mode.md`.

### Read from the audit (`session show`, `session tail --all`, `audit --kind permission-decision,mode-changed`)

| Probe (turn) | Time | Decision row | Prompt / resume | Mode |
|---|---|---|---|---|
| runner Bash `git log` (8) | 01:56:01 | none: `tool-call` only (says nothing of the hook, below) | no `waiting-for-answer`, no participant row | prompt |
| tester Write in tree (8) | 01:56:08 | none: `tool-call` only (says nothing of the hook, below) | no `waiting-for-answer`, no participant row | prompt |
| tester Write outside tree (8) | 01:56:21 | deny, source hook, decided by policy | none needed | prompt |
| tester Write in tree after the switch (9) | 02:00:46 | allow, source hook, decided by policy | none needed | policy |

- Every probe's `subagent-result` row says `ok`, within 0 to 2 seconds of its call, so the audit
  records the subagent ending, not failing. The allowed Read probe has an `allow` row.
- The coordinator's own deferred calls (turns 3, 4, 6, 8) each show `deferred` (policy),
  `waiting-for-answer`, then `allow` decided by the participant with the call's tool-use id. The
  two in-tree probes have none of the three. The audit cannot say whether the hook ran for them
  or what it returned: the bot writes a deferred row only from the engine's `deferred_tool_use`
  result (`deferLocked`, `internal/session/turn.go`) and an allow or deny row only when a
  hook-decided call's result arrives (`toolResult`), and the hook process writes no audit row.
  The hook log answers it (below).
- `mode-changed` prompt to policy, 02:00:16 by the driver: 3 minutes 53 seconds after the last
  prompt-mode probe (01:56:23) and 30 seconds before the policy-mode Write. It is the only mode
  row in the session.
- Session allow: the audit cannot tell `allow` from `allow-session` (`actions.go` records
  `DecisionAllow` for both), and a participant did allow the coordinator's Write at turn 4. What
  ties the last probe to the mode is the code order: `internal/policy/policy.go` (around lines
  297-300) returns Allow for an `ask` cell in policy mode before the session allow list is
  checked, so in policy mode the Write is allowed whether or not `Write` was session-allowed.

### Read from the hook log (`sessions/s0008/home/hook.log` under the root)

One JSON row per hook decision, times in UTC (the same clock as the times above). A row has no
field naming the agent, so a subagent's call is placed by its time after the `Agent` row.

| Probe | Log time | Row |
|---|---|---|
| runner Bash `git log` | 01:56:01 | `git-guard.js` PreToolUse passed; then `cell: ask`, `decision: defer` |
| tester Write in tree | 01:56:09 | `cell: ask`, `decision: defer` |
| tester Write outside tree | 01:56:21 | `cell: deny`, `decision: deny` |
| tester Write in tree, policy mode | 02:00:46 | `cell: ask`, `decision: allow`; `post-edit.js` PostToolUse passed |

- The hook ran for both in-tree probes in prompt mode and returned `defer`, which matches the
  reach table's inference. The hook log has no later row for either tool-use id: no resolving
  allow, unlike the coordinator's deferred `Edit` calls (01:56:36 defer, 01:57:04 allow).

### Second occurrence: instance `dev-live`, session s000g, 2026-11-02

- Prompt mode, engine 2.1.316, driven from Discord by the owner, building a pass. The coordinator
  launched three `coder` subagents: two in the background ("Quieter, named bot DMs", 20:49:25 and
  20:49:41 UTC) and one in the foreground (20:50:47, its prompt opening "Two earlier agents on this
  brief ended after their first message ... with nothing done").
- Each subagent's first `Bash` call hit the `ask` cell: the hook log shows `"cell":"ask","decision":"defer"`
  (five Bash defers between 20:49:27 and 20:50:53). The subagent transcripts
  (`<root>/.../projects/<slug>/<session>/subagents/agent-*.jsonl`) end on a `hook_deferred_tool`
  attachment with `"permissionMode":"default"`, and the meta files say `"finished":true`. The first
  agent's whole transcript is one sentence ("I'll start by reading CLAUDE.md and the rules.") and
  one Bash call; the foreground one ran six Reads (the `allow` cell), then ended at its first Bash.
- No prompt reached the venue for any of them (no `waiting-for-answer`). The coordinator then did the
  work itself in the main process (the hook log shows 53 Edits there).
- Control: a `reviewer` launched at 21:24:14 went through the reviewer cell (`defer`, then `allow`
  on the resume at 21:24:16) and ran to a full report (297 transcript lines). Its Bash calls were
  allowed by then: the hook log's Bash decisions turn `allow` later in the session (a session allow
  or a mode switch by the driver).
- s000e, the session before it, launched no subagent, so it is not evidence.

### Third look: the engine ends the subagent, not the bot (2026-11-07, engine 2.1.319)

- Three real turns on the PC from one scratch tree (`hello.txt`; `.claude/agents/probe.md`, Haiku, tools Read and Bash). The
  probe ran `echo hi`. One prompt
  for all three: launch `probe` (else `general-purpose`) in the foreground with model `haiku`, have it run `echo hi`, use no
  other tool, and quote the Agent result. The coordinator ran on `--model opus`. The raw streams, hook stdin and transcripts
  stay in `~/AppData/Local/Temp/bf-subdefer/`.
- The bot's path: `engine record --mode prompt` on the dev root, with the dev instance's subscription credential. There was
  no `probe` type (`--setting-sources user` drops the tree's agents; the 2026-10-23 settings-sources finding), so the
  subagent was `general-purpose` on Haiku. The recording imports cleanly (`importfixture`; a golden was written), but it is
  not in `fixtures/engine/`, because `TestEveryFixtureReplays` refuses it (below). The imported copy stays in the
  scratch directory as `fixture-imported/`.
- The control: bare `claude -p --permission-mode default --output-format stream-json --verbose --model opus --settings
  <file>` under the owner's login, with every `CLAUDE*` name unset, as the spawn strips them. Its stub hook logs stdin and
  prints the bot hook's exact JSON: `defer` (reason `Broodforge: waiting for a person in this session to answer`) for the
  `ask`-cell tools, `allow` for the rest. It found `probe`.

| Run | What the engine did with the subagent's deferred `Bash` |
|---|---|
| Bot, hook `defer` (`hook.log` `"cell":"ask","decision":"defer"` for its id) | No `tool_result` for it, ever. `system/task_updated` `patch.status` `completed`, then `system/task_notification` `status` `completed`. The `Agent` result is a success (`tool_result_meta` `accept`, source `hook`): text `(Subagent completed but returned no output.)` plus `agentId: <id> (use SendMessage ... to continue this agent)`; `tool_use_result` `status` `completed`, `content` `[]`, `canContinueAgent` true. The coordinator answered; `result` `end_turn`, no `deferred_tool_use`, `permission_denials` `[]`, exit 0. |
| Control, stub `defer` | The same, line for line. The line-type sequences differ only by the owner's plugin `SessionStart` hook lines and one coordinator text line. |
| Control, stub `ask` | `system/permission_denied` with `agent_id` and `decision_reason_type` `hook`, then an error `tool_result` to the subagent (`reject`, source `config`). The subagent went on and reported the refusal. `permission_denials` lists the call. No `control_request`. |

- On both paths the subagent's transcript ends on a `hook_deferred_tool` attachment (`toolUseID`, `toolName`, `toolInput`,
  `"permissionMode":"default"`), and its meta file says `"requestShape":"foreground"`, `"requestNonInteractive":true`, `"finished":true`.
- Hook stdin (control): the coordinator's `Agent` call carries `session_id`, `transcript_path`, `cwd`, `prompt_id`,
  `permission_mode`, `effort`, `hook_event_name`, `tool_name`, `tool_input` and `tool_use_id`. The subagent's `Bash` call
  carries the same fields plus `agent_id` and `agent_type`; the session, transcript and prompt ids are the parent's. The bot's hook reads neither `agent_id` nor `agent_type`.
- So the bot drops nothing. Under `-p` the engine ends a subagent at a hook `defer` and does not pass the defer up, so
  `deferLocked` has nothing to read. Only a top-level call's defer ends the process with `tool_deferred`. An `ask` is a
  refusal that the subagent survives. The 2026-10-31 spike's `"handback":"withheld"` is absent on both paths here. That
  spike ran 2.1.316; whether its environment differed was not tested.
- A second result: in all three streams the `rate_limit_event`'s `unifiedWindows` holds `seven_day` alone, with no
  `five_hour`, on both accounts. Every rate-limit line of the 2.1.296 fixtures has `five_hour`. `ParseLine` reads the
  missing window as zeros, so the meter of ADR 0021 would record a five-hour window of `0` that resets at `0`. The
  replay test's invariant (`internal/engine/parse_test.go`, `KindRateLimit`: both `resetsAt` non-zero) refused
  line 4, a rate-limit event with `seven_day` alone.

## Next

- The hook's `defer` for a subagent's call is now seen (Result, "Read from the hook log"); the
  audit shows no deferred row and no prompt, and the subagent ends while the engine turn goes on.
  A second occurrence on 2026-11-02 (s000g, `dev-live`, three `coder` launches; Result, "Second
  occurrence") shows the same end at the first deferred Bash.
  Answered 2026-11-07 (Result, "Third look"): the engine ends the subagent itself and raises no top-level
  `deferred_tool_use`; a bare `claude -p` does the same. Still open: a design answer for the
  architect. Prompt mode cannot host subagents until such a call resumes. What the engine offers is in the Third look:
  `agent_id` and `agent_type` on the hook's stdin; `canContinueAgent` and `SendMessage`, which is
  outside `code` reach; the loss is visible in the stream, as a subagent `tool_use` with no result before its
  `Agent` result, matching a `defer` row in `hook.log`; and `ask`, which is a refusal the subagent survives.
  `--permission-prompt-tool` is untried. The question bears on `auto` too
  (its classifier prompts would take the same path; today the engine forces `auto` back to `default`
  under the spawn's `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, so `auto` is no workaround: the 2026-10-31
  auto spike finding, row 3), and until it is answered a session that
  delegates runs in policy mode.
- Answered 2026-11-07: ADR 0021 Amendments 2026-11-07 (an absent window is NULL, printed `-`); the recording is
  `fixtures/engine/subagent-deferred/`.
- Decide whether `SendMessage` and `Skill` belong in `code` reach (the policy table,
  `03-hook-contract.md`); option D for skills needs the `Skill` row as well as its droplet row.
- Decide whether the copy takes the files `CLAUDE.md` imports outside `.claude/`, or the handoff
  reaches a session another way.
- Proposal 0027 for `auto` (0026 went to the held-sessions ADR) is still unwritten; it is the next wave's first architect brief.
