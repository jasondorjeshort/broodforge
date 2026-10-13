# 0005. Four rules that bit downstream and are missing from the base

Date: 2026-10-12
Status: draft

Source: Dystopo task T84 (Hatchery alignment), which brought the base's hooks, rules and skills into
Dystopo and compared the two working agreements line by line. Dystopo's records: its `CLAUDE.md`
(Working in parallel, Rules, Debug before you push), its findings
`docs/findings/2026-10-09-junction-into-main-checkout.md`,
`docs/findings/2026-10-09-branch-deleted-before-merge.md`, and its ADR
`docs/design/decisions/0021-git-guard-exceptions.md` (https://github.com/jasondorjeshort/dystopo).
A Windows note found on the way is a separate finding here,
`docs/findings/2026-10-12-windows-long-path-delete.md`.

## Question

Rules 1 and 2 were written into Dystopo's `CLAUDE.md` the day something went wrong (the two findings
above). Rules 3 and 4 were written before anything broke, as guards: 3 with the privacy hooks on
2026-10-09, 4 with the module map on 2026-10-10, both in Dystopo's "Rules" and "Debug before you push"
sections. None of the four is in the base. Should the base carry them, and where: as working-agreement
text, in a hook, or only as reference material for projects that reach the situation?

1. **Never create a junction or symlink inside a worktree that points at the main checkout.**
   Removing the worktree with a recursive delete follows the link and deletes the target's contents;
   it wiped Dystopo's main `out/` once (2026-10-09).
2. **Never delete a branch before its merge has succeeded.** A task branch deleted after a merge that
   had not gone through made GitHub close the pull request (2026-10-09); recovered by recreating the
   branch at the recorded SHA and `gh pr reopen`.
3. **Never `--no-verify`.** If a hook fires, fix the content. Dystopo's privacy hooks (personal places
   must never reach a public repository) are only as good as the guarantee that they run; the base
   has hooks of its own that a bypass would skip in the same way.
4. **Documentation is part of done, in the same PR.** A change updates the docs it touches (command
   and endpoint lines, the server manual, the design section, the KDoc of a route file, the module
   map) in the same commit series, not in a follow-up. In Dystopo the docs are the agents' interface
   to the project; a stale line costs the next agent a debugging round.

## Constraints that matter here

- Agents do all coding and debugging; a rule that depends on memory fails, a hook does not.
- `git-guard` reads only command text, `agent_type` and `cwd` (ADR 0002 companion, section 1); it can
  run `git` and `gh` but must fail closed.
- The base must stay usable on one branch in the design phase and in the wave model of proposal 0003.
- Windows is a supported host (Dystopo and this repository both run on it).

## Options

### A. Working-agreement text only

Add four lines to `CLAUDE.md` (rules 1 and 2 under Git, 3 under Git, 4 under "Debug interface is part
of every feature" or a new "Done" paragraph). Cost: four lines every turn. Risk: rules 2 and 3 are
exactly the kind an agent under pressure gets wrong once.

### B. Hooks where the rule is mechanical, text for the rest

- Rule 3 in `git-guard` (Dystopo's rules 13 and 14, test cases D18 to D25 in its `test-hooks.js`):
  deny `--no-verify` on `commit`, `push`, `merge`, `rebase`, `am`, `cherry-pick`, `revert`, and
  `git commit -n` (the short form of `--no-verify`), for every caller; deny `git -c
  core.hooksPath=...` and `git config` that unsets or redirects `core.hooksPath` away from the
  project's hooks directory. The base has no `core.hooksPath` today, so the second half is a slot a
  project fills.
- Rule 2 in `git-guard` once branch deletion is allowed at all (today rule 2 and rule 9 deny it for
  everyone, which is safe and sufficient for the single-branch phase). Under proposal 0003's wave
  model, where task branches are deleted after a squash merge: allow `push --delete`, `:ref` and
  `branch -D` only for task branches whose pull request GitHub shows as merged
  (`gh pr list --head <branch> --state merged --json number --limit 1`, run in the command's
  directory; any failure denies). Dystopo implemented and tested this (ADR 0021, E4 and E5, with a
  `GIT_GUARD_MERGED` test seam), and it means the hook, not the caller, decides whether a deletion
  is safe, which also lets cleanup be delegated to an agent.
- Rules 1 and 4 as text: rule 1 in `CLAUDE.md` under Git and in `docs/reference/claude-code-on-windows.md`
  (Git worktrees); rule 4 in `CLAUDE.md` with the list of what "the docs it touches" means for the
  project, which a decanted project fills in.

### C. Reference material only

Put all four in `docs/reference/claude-code-on-windows.md` and a finding each. Cheapest, and loaded only
on demand, which is exactly why it fails: the junction and the branch deletion happened to agents that
had no reason to open a reference file first.

## Recommendation

B. Two of the four are mechanical and have already been enforced and tested in Dystopo's guard; the
other two are short enough for the working agreement. First step: add the `--no-verify` rule (and the
`core.hooksPath` slot) to `git-guard.js` with its test cases, and the junction and documentation lines
to `CLAUDE.md`; carry the merged-PR deletion check into proposal 0003's revision, where git-guard's
rules 3 and 4 are keyed off the phase or the branch.

Two more items from the same adoption that the base should consider while it is there:

- **Match `git-guard` on the PowerShell tool too** (`"matcher": "Bash|PowerShell"`). On Windows
  agents use both shells; matching `Bash` only leaves every git command run through PowerShell
  unguarded. The parser handles PowerShell's `;` and the same `git` word forms.
- **The base harness as a contract.** Dystopo ran Hatchery's unmodified `test-hooks.js` against its
  adapted hooks: 12 of 72 cases differ, all recorded with a reason in its ADR 0021 (rules 3 and 4
  under the wave model; cap messages naming a pusher and a backlog the project does not have; its own
  usage gate). Recording that diff in the adopting project's ADR is a cheap, checkable form of
  "the base wins under `.claude/` unless an ADR records a reason to differ"; the `adopt-hatchery`
  skill could ask for it.

## What would change this recommendation

- If proposal 0003 is rejected and the base never deletes branches, the merged-PR check stays a
  Dystopo-only exception and rule 2 is text in the `commit` skill.
- If Claude Code's hook input gains the shell tool's parsed argv, the text matching in `git-guard`
  (and its false positives on quoted text) should be replaced before more rules are added.
- Evidence that agents hit the `--no-verify` deny on legitimate work (for example a project whose
  hooks are slow) would argue for an owner-set override file, like the usage gate's.

## Claims to verify

- Dystopo's guard and tests: `.claude/hooks/git-guard.js` and `test-hooks.js` in
  https://github.com/jasondorjeshort/dystopo after its PR for T84 merges (113 cases on 2026-10-12).
- `git commit -n` is `--no-verify` (git-commit documentation); `git push -n` is `--dry-run`, so the
  short form is matched for `commit` only.
- `gh pr list --head <branch> --state merged` returns the merged pull request for a squash-merged
  branch (checked in Dystopo on 2026-10-12: `task/T85`, PR #82, allowed; `task/T84`, no PR, denied).
- GitHub closes a pull request whose head branch is deleted before it is merged (GitHub docs, "About
  branches": deleting a branch closes its open pull requests).
- The PowerShell tool passes `tool_input.command` like the Bash tool (Claude Code hooks reference).

## Critic's objections and responses

(none yet)
