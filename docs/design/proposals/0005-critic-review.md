# Critic review of 0005: Four rules that bit downstream and are missing from the base

Date: 2026-10-13
Reviewer: critic (Opus 5.5)
Evidence: the base at `f747c1f` (`git-guard.js`, `policy.json`, `policy.examples/`,
`.claude/settings.json`, `.claude/rules/docs.md`, `.claude/agents/reviewer.md`,
`docs/reference/claude-code-on-windows.md`, ADRs 0006 and 0007); the downstream build-phase
project's findings, ADR 0021 and its log (Dystopo, at `d6e3b86`).

## Part 1. Claim verification, and what the base already does

| # | Claim or item | Verdict | Evidence |
|---|---|---|---|
| 1 | Rule 1 incident: a junction from a worktree into the main checkout wiped the main `out/` | Confirmed | Downstream finding `2026-10-09-junction-into-main-checkout.md` (`mklink /J`, then a recursive delete of the worktree). |
| 2 | Rule 2 incident: a branch deleted before its merge closed the PR | Confirmed | Downstream finding `2026-10-09-branch-deleted-before-merge.md`. |
| 3 | `git commit -n` is `--no-verify`; `push -n` is `--dry-run` | Confirmed | The guard matches `-n` for `commit` only (rule 13, `git-guard.js`). |
| 4 | `gh pr list --head <b> --state merged` finds squash-merged branches | Confirmed downstream | ADR 0021 E4 and its test seam; the base now has the same check behind `branchDelete.requireMergedPr`. |
| 5 | Rule 2 in the guard | **Done** | ADR 0006: `git.branchDelete` with `requireMergedPr`, fails closed; the task-branch example sets it. |
| 6 | Rule 3 in the guard, plus the `core.hooksPath` slot | **Done** | Rules 13 (`hooks bypass`) and 14 (`hooks path override`), `git.hooksPath` in policy. |
| 7 | Match `git-guard` on PowerShell too | **Done** | `.claude/settings.json`: matcher `Bash|PowerShell`. |
| 8 | The base harness as a contract | **Done in substance** | ADR 0006 Record and Consequences: the harness (325 cases) runs every policy example. |
| 9 | Rule 4 (docs in the same change) | **Done, in a weak place** | `.claude/rules/docs.md`: "A change that alters behaviour ... updates that doc in the same commit"; the `reviewer` checks it. See objection 2. |
| 10 | Rule 1 anywhere in the base | **Absent** | No match for junction or symlink in `CLAUDE.md`, `.claude/rules/`, `.claude/skills/` or the Windows reference. |

## Part 2. Objections

1. **Serious. Rule 1 is mechanical, and the proposal files it as text.** The proposal's own
   constraint is that "a rule that depends on memory fails, a hook does not", yet it puts the
   junction rule in `CLAUDE.md` under Git, where it is neither git nor read at the moment it
   matters. The damage happens at removal, and the base itself prescribes the removal step
   (`claude-code-on-windows.md`: `robocopy <empty> <dir> /MIR`, then `Remove-Item -Recurse
   -Force`). Windows PowerShell 5.1's recursive delete follows junctions, and robocopy includes
   junction points unless told `/XJ` (Microsoft's robocopy reference); whether its purge reaches
   through one I did not test, and the procedure should not depend on that. The hazard is also
   Windows-specific in practice (`rm -rf` does not follow symlinks), which argues against an
   every-turn line in every project. *Resolve:* (a) in the removal procedure, first list reparse
   points under the worktree (`Get-ChildItem -Recurse -Force -Attributes ReparsePoint`), remove
   each link itself (`cmd /c rmdir <link>` removes a junction, not its target), and add `/XJ` to
   the robocopy step; (b) a shell-guard rule that denies `mklink /J|/D`, `New-Item -ItemType
   Junction|SymbolicLink` and `ln -s` when the command's directory is inside a worktree, in the same
   policy-driven style as ADR 0006; (c) one line in the implementer's brief for worktree agents.
   (a) alone would have prevented the incident.

2. **Minor. Rule 4 sits in a path-scoped rule that the forgetful agent never loads.**
   `.claude/rules/docs.md` loads when an agent reads or edits `docs/**` or `CLAUDE.md`; an agent that
   changes code and forgets the docs never opens either, so the rule reaches it only through the
   reviewer. Downstream keeps it every turn in `CLAUDE.md` ("Documentation is part of done") and it
   still slips occasionally (its T75 log: a route used by the checks was not named in the docs; the
   verifier recorded it). The reviewer check is the real enforcement and is adequate. *Resolve:*
   one clause in the coder agent's definition ("update the docs the change touches; the reviewer
   will check"), not another `CLAUDE.md` line.

3. **Minor. The record is out of step with the tree.** ADR 0006 names this proposal as its source
   ("option B, and its two extra items"), while the proposal says "Status: draft" and the proposals
   README says "Awaits the critic". The hook half was decided and built without the critic step
   this repository's own rule requires; the substance holds up (Part 1, rows 5 to 8), so nothing
   needs undoing. *Resolve:* set the status to "promoted in part to ADR 0006; rules 1 and 4
   decided by the answer to this review", answer the objections in its last section, and update
   the README row.

Residual risks of what is already built: the merged-PR check needs `gh` and the network, so
cleanup waits offline (accepted downstream in ADR 0021); the `--no-verify` deny has no owner
override, which the proposal's own "what would change" names as the trigger to revisit; text
matching still reads command text, not argv.

## Recommendation

Most of this proposal is already decided and built correctly through ADR 0006, and the evidence
supports every part that was built: rules 2 and 3, the PowerShell matcher and the harness contract
should stay as they are. What remains is rule 1, and the proposal puts it in the wrong place: make
the worktree removal procedure safe against links (objection 1a, which is what would have saved
the downstream `out/`), add the creation guard if a shell-guard slot is cheap, and keep at most a
brief line rather than an every-turn rule. Leave rule 4 where it is, add one clause to the coder
definition, and close the proposal's status. Confidence: high on Part 1 and objections 2 and 3
(read in the base), moderate on objection 1's robocopy detail (documented default, purge behaviour
untested), high that a removal-time check is the stronger fix.
