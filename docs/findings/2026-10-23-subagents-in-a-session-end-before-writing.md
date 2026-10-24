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
- The last probe's allow has two explanations: the switch to policy mode, or a session allow of
  `Write` given by the driver at an earlier prompt (`internal/policy/policy.go`, the session-allow
  branch in `Decide`). The mode is not visible from inside the session; `audit`'s `mode-changed`
  row would settle it (not yet read).
- After the last probe, a `reviewer` of this commit ran 21 calls, Bash included, over about four
  minutes and returned a full report: the first subagent in this session to use Bash and finish.
- The likely mechanism: in prompt mode the deferral ends the engine turn, and the subagent is not
  resumed when the driver resumes. The coordinator's own deferred calls (an `Edit` to
  `docs/ideas/`, a `Bash` commit) did run after their prompts. Not yet observed: the deferral
  itself in `session tail` and `audit`, and whether the prompt for the subagent's call ever
  reached Discord.
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

## Next

- Confirm the mechanism from outside the session: `session tail s0008` and `audit` for the probe
  calls (was a `defer` returned, was a prompt posted, what did the resume carry), the
  `mode-changed` row, and whether a session allow of `Write` was in force
  (`BROODFORGE_SESSION_ALLOW`). If the mode explains the last probe, prompt mode cannot host
  subagents until a deferred subagent call resumes; that is a design question for the architect,
  it bears on `auto` too (its classifier prompts would take the same path), and until it is
  answered a session that delegates runs in policy mode.
- Decide whether `SendMessage` and `Skill` belong in `code` reach (the policy table,
  `03-hook-contract.md`); option D for skills needs the `Skill` row as well as its droplet row.
- Decide whether the copy takes the files `CLAUDE.md` imports outside `.claude/`, or the handoff
  reaches a session another way.
- Proposal 0026 for `auto` is still unwritten; it is the next wave's first architect brief.
