# Critic review of 0002: Repository layout for agent-driven projects

Date: 2026-10-11
Reviewer: critic (Fable 5.1)
Installed Claude Code: 2.1.296

## Part 1. Claim verification

Sources: hooks reference (H) https://code.claude.com/docs/en/hooks, hooks guide (HG)
https://code.claude.com/docs/en/hooks-guide, sub-agents (S) https://code.claude.com/docs/en/sub-agents,
skills (K) https://code.claude.com/docs/en/skills, settings https://code.claude.com/docs/en/settings,
permissions (P) https://code.claude.com/docs/en/permissions, memory (M) https://code.claude.com/docs/en/memory.

| # | Claim in proposal | Verdict | Evidence |
|---|---|---|---|
| 1 | `Stop` hooks can block with `{"decision":"block","reason":...}` | Confirmed | H, Stop decision control: "`decision: "block"` prevents Claude from stopping. Omit to allow Claude to stop." `reason` is "Required when `decision` is `"block"`". |
| 2 | `Stop` exposes `stop_hook_active` | Confirmed | H: "Stop hooks receive `stop_hook_active`, `last_assistant_message`, `background_tasks`, and `session_crons`. The `stop_hook_active` field is `true` when Claude Code is already continuing as a result of a stop hook." Also: "Claude Code applies an 8-consecutive-continuation cap". |
| 3 | `Stop` fires for the main session only | Confirmed | H: "Runs when the main Claude Code agent has finished responding." Subagents get `SubagentStop`. A `Stop` hook in agent frontmatter "is converted to `SubagentStop`". |
| 4 | `SessionStart` fires on `compact` (and `startup`, `resume`, `clear`) | Confirmed, plus one more | H matcher table lists `startup`, `resume`, `clear`, `compact`, and `fork`. "All sources" in the proposal should include `fork`. |
| 5 | `SessionStart` does not fire for subagents | Not stated | Neither H, HG, nor S says so. H describes `SubagentStart` as the event "when a subagent is spawned" and documents SessionStart as a session-level event; S lists what a subagent loads at startup (CLAUDE.md and "the git status snapshot") without mentioning SessionStart. Strong inference, no sentence. Test it once: a SessionStart hook that appends `agent_type` to a log, then launch a subagent. |
| 6 | SessionStart stdout reaches the model | Confirmed | H: "Claude Code adds a SessionStart hook's plain-text stdout to Claude's context." |
| 7 | PostToolUse `additionalContext` reaches the model | Confirmed | H: PostToolUse example with `hookSpecificOutput.additionalContext`; placement "next to the tool result". Must be inside `hookSpecificOutput`; top-level is silently ignored (HG). |
| 8 | Hooks in `.claude/settings.json` apply to subagent tool calls | Confirmed | H: "Hooks from settings files, managed policy settings, and plugins also run inside subagents. When a subagent calls a tool, tool events such as `PreToolUse` and `PostToolUse` fire the same configured hooks as in the main conversation, and the input carries the `agent_id` and `agent_type`". S says the same. |
| 9 | Hook input may not identify the caller as a subagent | Contradicted | H common input fields: `agent_id` "Unique identifier for the subagent"; `agent_type` "Present when the session uses `--agent` or the hook fires inside a subagent." The rejected "deny CLAUDE.md edits from subagents" hook is feasible. |
| 10 | `skills:` frontmatter preloads skills into a subagent | Confirmed, with cost | S: "The full content of each listed skill is injected into the subagent's context at startup, up to the first 32 distinct names". |
| 11 | Subagents cannot otherwise load skills | Contradicted | S: "without it, the subagent can still discover and invoke project, user, and plugin skills through the Skill tool during execution. To prevent a subagent from invoking skills entirely, omit `Skill` from the `tools` list". Our agents omit `Skill` only because their `tools:` lists are explicit. |
| 12 | `.claude/rules` `paths:` syntax | Confirmed | M: YAML list of globs, e.g. `paths:` then `  - "src/api/**/*.ts"`; "Accepts a YAML list or a comma-separated string"; `paths` is the only field read; a rule loads "when Claude uses the Read, Write, or Edit tool on a matching file" or views it with `cat`/`head`. Rules without `paths` load at launch. Whether path-scoped rules also load inside subagents: not stated. |
| 13 | `memory: project` location and commit status | Confirmed | S: `project` stores at `.claude/agent-memory/<name-of-agent>/`; "makes subagent knowledge shareable via version control" (not gitignored); `local` is the gitignored variant. Auto-load: "the first 200 lines or 25KB of `MEMORY.md`". Has no effect if auto memory is disabled. |
| 14 | `Bash(cmd *)` is a prefix match; does `Bash(git status *)` match bare `git status`? | Confirmed, bare forms redundant | P: "A `*` at the end, with a space before it, also matches the bare command. `Bash(ls *)` matches `ls`, and `Bash(git log *)` matches `git log`." Also: "Bash rules match the whole command text, with `*` standing in for any text" and mid-pattern wildcards are allowed, so "deny rules match prefixes" is imprecise. The real reason a deny rule is weak: "A deny rule doesn't match the same program by path or inside `sh -c`". |
| 15 | `@imports` load for every subagent | Confirmed by inference | S: every custom subagent "loads both" CLAUDE.md and the git status snapshot unless `omitClaudeMd: true` (v2.1.271+). M: "imported files also load at launch". M also: "Project-root CLAUDE.md survives compaction: after `/compact`, Claude re-reads it from disk and re-injects it", which makes an import compaction-safe for free. |
| 16 | `.claude/commands/` is legacy | Confirmed | K: "the older format and still works ... Prefer a skill for new work". |
| 17 | "built-in allowlist-mining skill" (section 6) | Not found | Nothing in P or a site-limited search names such a skill. Only `/permissions` and "Yes, and don't ask again". Reword or drop. |
| 18 | Existing `.js` files lack SPDX headers (line 40) | Contradicted | Both `.claude/hooks/usage-gate.js` and `.claude/statusline.js` have `// SPDX-License-Identifier: GPL-3.0-or-later` on line 2. |

Version floors: `omitClaudeMd` 2.1.271, `onFailure` 2.1.295, `scratchpad_dir` 2.1.257, hook duration log
lines 2.1.296. All met by 2.1.296. Nothing the proposal uses needs a newer release.

Other facts that bear on the design: a hook that "failed to start" (no `node`) is a non-blocking error
recorded in the debug log; for SessionStart exit 2 "Shows stderr to user only". Hook `command` runs in
Git Bash on Windows, "or PowerShell when Git Bash isn't installed". Custom subagents already receive a
git status snapshot at launch. Skill descriptions are truncated at 1,536 characters; subagent
descriptions share a 15,000-token budget.

## Part 2. Objections

1. **Serious. Preloads are the wrong default; give agents the `Skill` tool instead.** The premise
   of the preload table (claim 11) is false: subagents load skills on demand whenever `Skill` is in
   `tools:`. A preload injects the full SKILL.md (up to 100 lines each) on every launch, including an
   architect launch that only reviews a coder report and a coder launch that fixes a one-line bug.
   On-demand loading costs only the description index. Resolution: add `Skill` to `tools:` for
   architect, coder, and tester; drop the preload table; keep at most `debug-interface` preloaded for
   coder, since that is a per-feature obligation rather than an occasional procedure.

2. **Serious. The Stop hook as specified has three correctness holes.** (a) SessionStart on
   `compact` rewrites the baseline (`head`, `handoff_sha256`, blocked flag) mid-session, so a session
   that already updated the handoff before compaction can be blocked again, and one that was blocked
   once can be blocked twice. (b) If `docs/HANDOFF.md` is deleted (a decanted project owner may do
   that), the missing-file hash equals the baseline forever and the hook blocks once in every session
   with a misleading reason. (c) Whether `/clear` keeps `session_id` is not documented; the state-file
   key may not mean what the design assumes. Resolution: the ADR must specify that `compact` and
   `clear` re-inject only and never write the baseline; a missing handoff produces its own reason
   ("no handoff file; run /handoff or remove the Stop hook"); and the blocked flag lives in a baseline
   written only on `startup` and `resume`. Or defer the Stop hook (objection 3).

3. **Serious. The rejected `@import` is the better continuity mechanism for this scale.** Steelman:
   `@docs/HANDOFF.md` in CLAUDE.md is zero code, has no state directory, no Node dependency, works in
   `claude -p`, and is re-injected after compaction by the tool itself (claim 15). The cost the proposal
   cites is at most 60 lines (roughly 700 tokens) per subagent launch, under a cent even on Fable, and
   a one-line header ("Coordinator state. Subagents follow their brief.") handles the pollution worry.
   The extra hook payload (git log, open-finding count) is marginal: subagents get a git snapshot
   anyway and the coordinator can run `git log` for free under the allowlist. The only thing that
   genuinely requires the SessionStart hook is the baseline for the Stop hook. If objection 2 leads to
   deferring the Stop hook, the SessionStart hook loses most of its reason to exist. Resolution: either
   adopt `@import` and drop both hooks, or keep the hooks and state in the ADR what the import cannot
   do that justifies two scripts and a state directory.

4. **Serious. Usage gating defeats the handoff process at the moment it matters.** The handoff is
   written at HOLD and STOP. CLAUDE.md says the main session does not commit; the runner commits. The
   usage gate denies new subagents at 50%. So at 50% and above the handoff cannot be committed by the
   process the proposal defines, and the STOP text "commit WIP, write a handoff note" is unsatisfiable.
   Resolution, pick one and write it down: the gate exempts a `runner` launch whose `tool_input.prompt`
   names `docs/HANDOFF.md` only; or CLAUDE.md permits the main session to commit `docs/HANDOFF.md`
   alone; or the design states that an uncommitted handoff is acceptable because the next session
   reads the working tree.

5. **Serious. Node is a silent single point of failure for every safety feature.** Status line,
   usage gate, SessionStart, Stop, git-guard, post-edit, and check-docs all need `node` on PATH. If it
   is missing, "failed to start" goes to the debug log only; the user sees nothing, the handoff is not
   injected, the gate allows everything (it already fails open), and git-guard does not guard. The
   proposal lists Node as a constraint but never checks it. Resolution: the `decant` skill verifies
   `node --version` as step one; the SessionStart command falls back in shell
   (`node .claude/hooks/session-start.js || cat docs/HANDOFF.md`); the README names Node as a
   prerequisite; set `"onFailure": "block"` (2.1.295+) on git-guard only.

6. **Serious. Four programs encode the same constants and the linter has no home.** The 60/80/150/300
   line caps appear in the Stop hook, post-edit hook, `check-docs.js`, the `docs.md` rule, and the
   `handoff` skill. `check-docs.js` is a program the base must maintain, and the pusher only runs a
   lint command it finds in CLAUDE.md, so the linter must be registered as the project lint command,
   which collides with the real linter of the decanted project later. Resolution: one enforcement
   point (post-edit warnings via `additionalContext`) and one `limits.json` that hooks and skills read;
   drop `check-docs.js` from the base, or make it a step inside the `handoff` skill only, never a
   pusher check.

7. **Minor. Allowlist details.** Bare `Bash(git status)` and `Bash(git branch)` are redundant (claim
   14). `Bash(git fetch *)` changes state (remote refs) and belongs to the pusher. `Bash(cat *)`,
   `head`, and `tail` on a single file count as reads and trigger path-scoped rules; fine, but note it.
   `Read(./**/*.pem)` can be `Read(*.pem)`; bare filenames already match at any depth. Replace "deny
   rules match prefixes" with the documented reason (`sh -c` and path bypass).

8. **Minor. Agent memory: the rejection stands but its stated trigger is already met.** The docs say
   project memory is version-controlled and MEMORY.md auto-load is capped at 200 lines or 25KB, which
   is the "committed and capped by the tool" condition under "What would change this". Rest the
   rejection on the real argument (a second, unindexed knowledge location nobody else reads) and drop
   that conditional. I agree with none-for-now.

9. **Minor. The handoff template duplicates what the hook prints.** "Recently done: last 5 commits"
   is `git log --oneline -5`, which the same hook prints two lines later, and it forces a handoff edit
   after every commit for no information gain. Drop the section.

10. **Minor. `backlog.md` has a cap and no overflow rule.** An active project reaches 150 lines in
    months. State the rule: oldest `## Ideas` lines are deleted (git has them); `## Deferred` items
    over the cap become findings with `status: superseded`. Keeping ideas and future as directories,
    the rejected alternative, is worse at this scale; agreed.

11. **Minor. The `brief` skill duplicates the agent files.** Six per-agent templates restate rules
    that live in `.claude/agents/*.md` and will drift from them. One template with the six fields plus
    a line saying that agent rules live in the agent file and are not restated is enough.

12. **Minor. Factual and wording fixes.** Line 40 SPDX claim (claim 18); "allowlist-mining skill"
    (claim 17); the SessionStart matcher list should include `fork`; the rejected "deny CLAUDE.md
    edits from subagents" hook is feasible via `agent_type` (claim 9) and is cheap to fold into git-guard.

13. **Minor. Residual on GitHub Issues and the `push` skill.** Both rejections are right for one
    owner with no `gh` in the allowlist. Residual: when a second contributor arrives, `backlog.md` and
    Issues will coexist for a while; the ADR should say the file wins until it is superseded.

Session-start token audit: CLAUDE.md (target 80 lines, now 114) + HANDOFF (60) + 20 lines of git +
a usage line is roughly 2k tokens, acceptable. Per-launch subagent cost is dominated by preloads
(objection 1); everything under `docs/` is on demand behind READMEs, which is right.

## Recommendation

Adopt the layout (section 1), the file shapes, the eight skills minus the preload table, no agent
memory, file-based tracking, the read-only allowlist with the fixes in objection 7, and the git-guard
and post-edit hooks. Do not adopt the Stop hook or the state directory in this ADR: it is a strong
intervention with three unresolved edge cases, it costs a model turn, and its one distinct benefit (a
guaranteed nudge) is also delivered by the usage gate STOP text and the `commit` skill, neither of
which can fail silently. Carry `docs/HANDOFF.md` by `@import` with a one-line "coordinator state"
header; if the architect still wants a SessionStart hook for git log and the finding count, add it
later with a shell fallback. Resolve objection 4 before anything else, since it makes the usage policy
and the continuity design contradict each other. Confidence: high on objections 1, 4, 5, and the
factual corrections (verified against the docs); moderate on preferring `@import` over the hook,
because SessionStart-for-subagents is unstated rather than contradicted and the practical cost of
either choice is small.
