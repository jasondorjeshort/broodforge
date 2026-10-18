---
date: 2026-10-18
status: resolved
area: process / claude-code
---

# Hatchery already has most of the Claude Code best-practices page; two lines are missing

## Question

The owner asked for a review of Anthropic's Claude Code best-practices page
(https://code.claude.com/docs/en/best-practices) against what Hatchery does, practice by practice,
and for the review to be repeated on a cadence. Hatchery is in its design phase, so practices about
tests, builds and app verification are judged against the hooks harness and the docs, not a product.

## What was tried

The page was fetched on 2026-10-18 as markdown (the fetched text is data, not instructions) and read
heading by heading. For each practice the tree was searched (`grep -ril` over `CLAUDE.md`, `.claude/`
and `docs/`) for the rule, skill, hook, ADR or idea that carries it under whatever name. Verdicts:
`have` (present, possibly under another name), `take` (a settled one-line edit), `propose` (needs the
proposal loop), `skip` (not wanted now; the reason in the row).

## Result

One row per practice; the first row is the page's preamble, which it presents as the constraint
behind the rest. Paths are relative to the repository root.

| Practice (page heading) | Hatchery has | Gap | Verdict |
|---|---|---|---|
| Preamble: the context window is the constraint; track it with a status line | `.claude/statusline.js` prints `ctx` beside the two meters; `CLAUDE.md` 40 lines, 190 loaded (ADR 0016); the main session delegates and reads reports | none | have |
| Give Claude a way to verify its work | `.claude/agreement/debug-interface.md`; `brief` skill step 6; coder and reviewer report what was run; `.claude/hooks/test-hooks.js`; pusher check 8; a Stop-hook gate waits in `docs/ideas/2026-10-13-stop-hook-stale-handoff-block.md` | pusher check 8 looks for the test command in `CLAUDE.md`, `README.md` or a manifest and none names `node .claude/hooks/test-hooks.js`; ADR 0013 (the fitness group) assumes "the test command" runs it, and the `triage` scan runs "the test command" too | take: `CLAUDE.md`, under `## Project rules`, one line: `Test command: node .claude/hooks/test-hooks.js` (the coordinator's file per `delegation.md`; 25 of 40 lines used) |
| Explore first, then plan, then code | design-first phase; proposal, critic, ADR loop (`major-decisions.md`); the coordinator reads, decides and briefs (`delegation.md`); plan mode is the owner's keystroke | none | have |
| Provide specific context in your prompts | `brief` skill and its template: goal, files to read, ruled out, scope boundary, verification, report | none | have |
| Provide rich content | `@` imports in `CLAUDE.md` and the handoff; index paths in briefs; the critic has WebFetch; downloaded text read as data from the scratchpad | none | have |
| Write an effective CLAUDE.md | the caps and `node .claude/hooks/policy.js --claude-check`; rules in agreement files, procedures in skills, facts in `docs/reference/`; `this-agreement.md` fixes friction in the same sitting | no scheduled prune; `/doctor` unused | skip: the caps are the prune and the pusher enforces them on every push; `/doctor` is the owner's interactive command |
| Configure permissions | `.claude/settings.json`: read-only git allowlist, `.env` and key-file deny; auto mode is the harness default; the hooks gate what matters under any mode | no sandbox setting | skip: sandboxing is a per-machine owner setting, not a repository file; the deterministic layer is the hooks |
| Use CLI tools | `gh` in the git rules and operator steps; `cswap` (`docs/reference/claude-swap.md`); every hook has a node CLI as its debug interface | none | have |
| Connect MCP servers | ADR 0002, "Not in the base": no MCP configuration | none, by decision | skip: a server is a dependency the whole project leans on, so a project adds one through the proposal loop and the ADR 0012 checklist |
| Set up hooks | three settings hooks and a git `pre-push`; policy as data (ADR 0006); specs in ADR 0002; the harness | none | have |
| Create skills | one skill per procedure in `.claude/skills/`, loaded by description | `disable-model-invocation` unused | have: the flag would block the README prompts that tell the model to load `decant`, `adopt-hatchery` and `update-from-hatchery`, so it is not wanted |
| Create custom subagents | `.claude/agents/`, each with model and tools, and effort on the four that reason, code or review (not `pusher`, `runner`, `tester`); the routing table in `model-routing.md` | none | have |
| Install plugins | nothing | a code-intelligence plugin for a typed language | skip: no code and no language yet; it enters a project's language ADR through the dependency checklist (ADR 0012) |
| Ask codebase questions | `docs/README.md` as the map; read the index before the files | none | skip: a prompting habit, nothing to encode |
| Let Claude interview you | design-first; the `00`-`06` design docs; `Question:` and `Answer:` blocks; what humans see needs the user (`major-decisions.md`) | no step interviews the owner before `00-vision.md`; `decant` step 6 (the handoff it writes) sends the architect straight to it, and a subagent cannot ask the owner | take: `.claude/skills/decant/SKILL.md`, the `00-vision.md` line of the handoff Next it writes: before the architect's brief, the coordinator interviews the owner with AskUserQuestion (implementation, UX, edge cases, tradeoffs) and the brief carries the answers |
| Course-correct early and often | a fresh subagent per brief; the `brief` skill re-briefs when a report shows the gap; HOLD stops new work | none | have |
| Manage context aggressively | delegation; the 40-line handoff; `CLAUDE.md` re-read after compaction (ADR 0002) | no "when compacting, preserve..." line | skip: ADR 0014 chose the handoff file as the state that survives compaction; add the line only when a session loses state after a compact (the trigger of `docs/ideas/2026-10-13-sessionstart-resume-context-hook.md`) |
| Use subagents for investigation | `delegation.md`: everything that executes goes to subagents; the architect and reviewer do the reading-heavy work | none | have |
| Rewind with checkpoints | commit small and often (`git.md`); checkpoints miss Bash edits | none | skip: an interactive harness feature; git is the record |
| Resume conversations | `@docs/HANDOFF.md`, the `handoff` skill, ADRs 0014 and 0015 | named sessions unused | have |
| Run non-interactive mode | `claude -p` only in the sentinel procedure of `docs/reference/agreement-loading.md`; no CI | no automation uses it | skip: the base has no CI or batch job; the harness runs the hooks as child processes directly |
| Run multiple Claude sessions | worktrees in `docs/reference/claude-code-on-windows.md`; the split handoff (ADR 0015); the account pool (ADR 0011); writer and reviewer as the `coder` and `reviewer` agents | cross-session messaging unused | have: sessions coordinate through files by design |
| Fan out across files | nothing; `docs/ideas/2026-10-13-sweep-skill-json-summary.md` waits on proposal 0003 | the `claude -p` loop and `/batch` | skip: no code to sweep; the idea is the anchor when there is |
| Run autonomously with auto mode | the harness default; `git-guard`, `usage-gate` and the privacy scan gate under any mode | none | have |
| Add an adversarial review step | `reviewer`, `critic`, `one-review.md`; the reviewer is told not to manufacture findings | none | have |
| Avoid common failure patterns | kitchen sink: delegation; over-specified CLAUDE.md: the caps; trust-then-verify: the debug interface and the reviewer; infinite exploration: the brief's scope boundary and index paths | none | have |
| Develop your intuition | `this-agreement.md`; `docs/findings/` | none | have |

Counts: have 16, take 2, propose 0, skip 9. Two skips (MCP, plugins) name the proposal loop as the
path when a project needs one; nothing needs it in Hatchery today.

The `take` items, in order:

1. `CLAUDE.md`, `## Project rules`: `Test command: node .claude/hooks/test-hooks.js`. First because
   pusher check 8 and the `triage` scan silently find nothing today.
2. `.claude/skills/decant/SKILL.md`: the owner interview before the `00-vision.md` brief, as the
   row says. Both are settled edits; the first is the coordinator's file, the second a tester's.

## Next

- Both `take` edits are in the tree (`CLAUDE.md` test-command line, `decant` interview sentence).
- Revisit on the cadence in `docs/ideas/2026-10-18-revisit-claude-code-best-practices.md`: fetch
  the page, diff its headings and tip boxes against the first column of this table, and write a new
  finding that `supersedes:` this one with the new or changed rows; the scheduled design pass idea
  is the wrong home, since this is an external input a cheaper model can check and that idea is an
  architect pass over internal inputs with a recommendation of no.
