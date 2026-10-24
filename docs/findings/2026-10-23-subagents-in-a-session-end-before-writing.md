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
coordinator made three `architect` launches and one continuation, briefed to write proposal 0026
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
  is not the cause.
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

## Next

- The hook's `defer` for a subagent's call is now seen (Result, "Read from the hook log"); the
  audit shows no deferred row and no prompt, and the subagent ends while the engine turn goes on.
  Still open: why the engine returns no top-level `deferred_tool_use` result for a subagent's
  deferred call, so the bot never posts a prompt (the engine, or the bot's handling of a call
  whose tool-use id belongs to a subagent), and a design answer for the
  architect: prompt mode cannot host subagents until such a call resumes; it bears on `auto` too
  (its classifier prompts would take the same path), and until it is answered a session that
  delegates runs in policy mode.
- Decide whether `SendMessage` and `Skill` belong in `code` reach (the policy table,
  `03-hook-contract.md`); option D for skills needs the `Skill` row as well as its droplet row.
- Decide whether the copy takes the files `CLAUDE.md` imports outside `.claude/`, or the handoff
  reaches a session another way.
- Proposal 0026 for `auto` is still unwritten; it is the next wave's first architect brief.
