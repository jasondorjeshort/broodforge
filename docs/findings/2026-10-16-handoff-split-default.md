---
date: 2026-10-16
status: resolved
area: process / coordination files
promoted_to: docs/design/decisions/0015-handoff-shape-at-decant.md
---

# Every recorded handoff split followed days on one file and an observed failure

## Question

When should a project adopt ADR 0014's per-stream handoff: from the start, or at a trigger
(the owner: decanted projects run parallel sessions from day one)? Is a map of `@` import lines
or a README listing the files the better index? Builds on `2026-10-16-handoff-layout-practices.md`
("the prior finding"). Recommends nothing; "States" is the source, "Inference" is mine.

## What was tried

2026-10-16: WebSearch, WebFetch; `curl` of the Claude Code changelog (head 2.1.301), the Task
Master and beads changelogs and the npm registry; `gh api` for histories. Siblings read-only with
`git --no-optional-locks -C`, no fetch. "Own" = non-merge commits absent from `git -C hatchery log --all`.
Per repository: `git log --since=2026-10-02 --no-merges --format='C %H %cs' --name-only` (awk);
`git log --merges --format=%h%x20%cs -- docs/HANDOFF.md docs/handoff/`, again with `--full-history`,
each merge's `docs/HANDOFF.md` at merge-base and both parents taken by `git show` into the
scratchpad and replayed with `git merge-file -p` (exit code = conflicts); `git reflog` filtered on
`rebase (continue)`, each commit checked with `git diff-tree --name-only -r`; for regenesis the
brief's `--diff-filter=A` lines, `ls`, `wc -l -c`. Codex and Windsurf docs redirected to
learn.chatgpt.com and docs.devin.ai and were read there. No source failed.

### 1. Local numbers (14 days to 2026-10-16; broodforge has a live session, its tree is dirty)

| Repo | Commits: all / own | Handoff commits: path-limited / own | Own, handoff only | Merges (brief) | Full-history merges whose handoff replay conflicts |
|---|---|---|---|---|---|
| regenesis | 593 / 482 | 59 / 46 | 20 | 3 | 5 of 22: 4 base updates, 1 branch that had deleted the file |
| broodforge | 122 / 50 | 37 / 20 | 11 | 1 | 2 of 4, both `Merge hatchery/main` |
| droplet-management | 93 / 93 | 5 / 5 | 0 | 0 | none; handoff only since 2026-10-15 |
| dystopo | 399 / 399 | 71 / 71 | 45 | 0 | none; one human author, handoff commits over 6 days |
| hatchery (scale) | 95 | 30 | 12 | 2 | 0 of 2 |

- regenesis took the base's single file 2026-10-12 15:50 (e4d61c7e), split 2026-10-16 07:23
  (af2b8d27): 3.6 days, 256 own commits, 35 touching the handoff, 18 nothing else. Since (to
  12:02): 13 own, 11 touching the handoff paths, 2 only those. Handoff commit authors (base
  included): Dorje, two spellings, 44; one other local identity 14; another 1. Nine author names on all commits.
- Stream files ever added 8 (seven imported, one README), 8 exist, none closed. Loaded per launch:
  map 14 lines plus seven files 132 = 146 lines, 8,199 bytes, over ADR 0014's 120; the other four
  handoffs, single files, are 34 to 40 lines, 1,747 to 2,668 bytes. From 07:23 to 09:58
  (b5a88b50) no stream loaded (the spelling fix ADR 0014 records).
- This regenesis clone's reflog (from 2026-09-04): 91 entries from `pull --rebase` runs on
  2026-10-12 to 16; 13 `rebase (continue)` in all, one on a handoff commit, 2026-10-16 09:42,
  after the split. Other three siblings: none. The project's other clones are not visible here.
- Its ADR 0017 (owner's decision, no critic) states 48 of 438 commits in a week touched the
  handoff and "that rebase conflicted on the handoff routinely"; no reason for not starting split;
  supersede "when parallel sessions on this repo stop (one stream again, and the single file is simpler)".

### 2. Accounts of when others split (dated)

- ehky2882/TRAVEL-GUIDED-TOUR (prior finding: PR 845, merged 2026-09-13; `gh api` today): STATUS.md
  first committed 2026-08-25, 161 commits in 19 days, split after three PRs blocked in two days.
- marcin-urbanski/claude-skills (github.com/marcin-urbanski/claude-skills/pull/14, merged
  2026-10-07; repo created 2026-09-30): one `docs/handoff.md` for 7 days, then a gitignored
  `docs/handoffs/<slug>.md` per thread plus `_overview.md`. States the cause: the handoff was
  written "inside the worktree, where it dies with it". No conflict count cited.
- AnastasiyaW/claude-code-config (github.com/AnastasiyaW/claude-code-config/blob/main/rules/session-handoff.md):
  rule added 2026-04-08; 2026-04-14 "offer two-mode choice (single-file default, multi-session
  opt-in)"; one file means "last writer wins and earlier handoffs vanish"; per-chat files with an
  append-only `INDEX.md`, no auto-delete, manual 7- and 14-day cleanup.
- beads (CHANGELOG.md): one JSONL; v0.9.0 2025-10-12; hash IDs 2025-10-31 against collisions; a
  merge driver 2025-11-05 "Eliminates most git merge conflicts"; Dolt migration by 2026-01-14.
- Task Master (CHANGELOG.md, npm times): one `tasks.json`; `move` for "merge conflicts ... when
  multiple team members have created tasks on different branches" (0.15.0, 2025-05-24); tags in the
  same file for "parallel (agentic) development ... without conflicts" (0.17.0, 2025-06-15).
- Split by design from the start: Spec Kit, one `specs/NNN-name/` per feature on its own branch
  (github.com/github/spec-kit/blob/main/spec-driven.md); Kiro; the C compiler's `current_tasks/`
  and Codex ExecPlans (prior finding). Each unit is a piece of work read on demand.
- Birgitta Böckeler, 2025-10-15 (martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html), one
  user: spec-kit "created a LOT of markdown files for me to review", "very verbose and tedious",
  "felt like overkill" on a mid-sized feature. Not found: a team that started split and folded back.

### 3. Guidance on parallel sessions (fetched 2026-10-16, pages undated)

- code.claude.com/docs/en/agent-teams: "Two teammates editing the same file leads to overwrites.
  Break the work so each teammate owns a different set of files." And: "For sequential tasks,
  same-file edits, or work with many dependencies, a single session or subagents are more effective."
- code.claude.com/docs/en/best-practices, common-workflows: worktrees "so edits don't collide".
  Codex (learn.chatgpt.com/docs/environments/git-worktrees): "without interfering with each other";
  Cursor (cursor.com/docs/configuration/worktrees): "several agents on the same repo without
  conflicts". None names a shared status file or says which files a task edits.
- Simon Willison, 2025-10-05 (simonwillison.net/2025/Oct/5/parallel-coding-agents/): separate checkouts, no status file.

### 4. Index mechanisms (docs fetched 2026-10-16)

| Tool | Inlines `@path` in instruction files | Relative base, depth | A fork reading the map sees |
|---|---|---|---|
| Claude Code (code.claude.com/docs/en/memory) | yes, at launch | importing file; four hops | the streams |
| Copilot CLI (docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions) | in copilot-instructions.md, AGENTS.md, CLAUDE.md, nested; not in GEMINI.md or `*.instructions.md` | not stated | the streams only if the base is the importing file (untested) |
| Gemini CLI (geminicli.com/docs/reference/memport/, updated 2026-04-10) | in GEMINI.md, or CLAUDE.md via `context.fileName` | importing file per its API reference; 5 | the streams; a missing target leaves an error comment |
| Codex (learn.chatgpt.com/docs/agent-configuration/agents-md) | none documented; AGENTS.md root to cwd, 32 KiB | n/a | inert lines |
| Cursor (cursor.com/docs/context/rules) | "not inlined into the prompt"; read when needed | n/a | lines the agent may follow |
| Windsurf (docs.devin.ai/desktop/cascade/memories) | none documented; 12,000 characters per rule file | n/a | inert lines |
| Aider (aider.chat/docs/usage/conventions.html) | only files named by `--read` or `read:` | n/a | the map only |

- No tool documents a directory or glob import; globs in Cursor, Windsurf and Copilot scope when a
  rule applies. No tool loads a README index. Claude Code's memory page, of a CLAUDE.md naming
  AGENTS.md in words: "Claude sees AGENTS.md only if it decides to open the file."
- Seen in the wild: a SessionStart hook choosing the file by branch (claude-skills); an index read
  at start by instruction (claude-code-config). regenesis runs an import map and a README both.

### 5. Context cost

- memory page: imports "don't reduce its context cost, because imported files also load at
  launch"; "target under 200 lines per CLAUDE.md file"; a warning when files "add up past a
  combined limit", and "Each CLAUDE.md, rules file, and `@path` import counts as a separate file".
  No number is documented. Changelog: 2.1.169, the threshold "scales with the model's context
  window"; 2.1.281 counts "instruction files together, so many mid-sized files and @-imports are caught".
- Subagents load the CLAUDE.md hierarchy (prior finding), so every launch pays the handoff again.

## Result

1. Timing. States: every dated account of a shared status or handoff file that was split went
   through a single-file period first: 3.6 days (regenesis), 7 (claude-skills), 19
   (TRAVEL-GUIDED-TOUR); beads and Task Master kept one file and patched IDs, drivers or tags.
   Each switch followed an observed failure (blocked PRs, routine rebase conflicts, a lost
   worktree handoff), not a count fixed in advance. The layouts split from the start (Spec Kit,
   Kiro, claim files, ExecPlans) hold pieces of work read on demand, not every-session state.
   claude-code-config made the single file the default and multi-session an opt-in.
2. For the owner's position. States: vendor guidance partitions files per agent before any
   conflict (agent teams, worktrees); the C compiler's claim directory was there from day one.
   Inference: where several sessions write one branch from the first day, the record puts the
   split within days; if conflicts come as ADR 0017 reports ("routinely"), a two-conflict
   trigger fires in the first week, so waiting costs days of conflicts and a migration commit.
3. Against it. States: single-writer setups keep one file (the long-running harness, the
   claude-code-config default). dystopo: one human author, 71 handoff commits in 6 days, no
   recorded handoff conflict. Of 7 handoff replay conflicts in the siblings' merges, 6 are base
   updates and one a deleted file; none is two sessions on one branch. This regenesis clone's
   reflog shows no rebase stop on a handoff commit before the split (other clones unseen). Costs
   seen after splitting: 146 lines per launch against 34 to 40, past the base's 120 within a day
   with no stream closed; 2.5 hours with no stream loaded; stale-file cleanup rules
   (claude-code-config); review fatigue from many files (Böckeler).
4. Index. States: an import map loads with documented certainty only in Claude Code; Copilot CLI
   and Gemini CLI expand nested `@` (Copilot's relative base undocumented); Codex, Cursor,
   Windsurf and Aider show the lines inert. A README index loads in no tool; every tool, Claude
   Code included, reads it only if the agent chooses to. No tool imports a directory or glob.
5. One writer, split from the start. Inference: in Claude Code a map with one stream costs the
   map's own lines (aim under 15) and one more file toward the combined warning, plus the silent
   miss on a wrong spelling; in a tool that does not inline, the stream is not read at all.

## Next

- Still open after ADR 0015: whether nested imports re-expand after `/compact`; the combined
  instruction-file limit (no number documented); Copilot CLI's relative base for a nested `@` (a two-file test).
