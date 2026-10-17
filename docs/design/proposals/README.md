# Proposals

Drafts for major choices. One file per proposal: `NNNN-short-title.md`, numbered to match the
ADR it will become. A proposal is written by the architect, challenged by the critic, then
revised and promoted to `../decisions/NNNN-short-title.md`. Keep the proposal file; it is the
record of the alternatives and the objections.

| File | Status |
|---|---|
| `0002-repository-layout.md` | Promoted to ADR 0002 (`../decisions/0002-repository-layout.md`); revised in place after the critic review. |
| `0002-critic-review.md` | Companion to 0002: the critic's review, answered in the proposal's last section. |
| `0003-wave-workflow.md` | Draft: the build-phase workflow a downstream project converged on (task board with Owns lines, worktree per task, verifier per PR, one-command sweeps, publish after every green sweep). Critic reviewed; awaits the architect's answers. |
| `0003-critic-review.md` | Companion to 0003: the critic's review (four serious, three minor; adopt B revised). Awaits the architect's answers. |
| `0005-rules-that-bit-downstream.md` | Draft: four working rules a downstream project needed that the base lacks (no junctions into worktrees, no branch delete before a verified merge, no `--no-verify`, docs in the same PR). Critic reviewed; awaits the architect's answers. |
| `0005-critic-review.md` | Companion to 0005: the critic's review (one serious, two minor; most of 0005 is already built through ADR 0006). Awaits the architect's answers. |
| `0011-usage-gate-under-account-switching.md` | Under review, revised after the critic: a usage cache per session with the account as a label, the status line naming the account, the gate reporting the pool account with room at HOLD through `usage.pool`, rotation at session boundaries; the per-model meter deferred. |
| `0011-critic-review.md` | Companion to 0011: the critic's review (one blocking, six serious, six minor; all accepted in the revision). |
| `0012-decision-practices-for-long-lived-software.md` | Promoted to ADR 0012 (`../decisions/0012-decision-practices-for-long-lived-software.md`); revised in place after the critic review: evidence corrected, four `CLAUDE.md` lines, surfaces scoped to incompatible change, a skip entry as the fitness lift, the design pass replaced by a trigger scan, dependency default evaluate, Hatchery's own surfaces first (ADR 0013). |
| `0012-critic-review.md` | Companion to 0012: the critic's review (two blocking, seven serious, six minor), all answered in the proposal's last section; every objection accepted, two with a change. |
| `0014-handoff-layout.md` | Promoted to ADR 0014 (`../decisions/0014-handoff-layout.md`); revised in place after the critic review: the import spelling `@handoff/<file>.md`, four hops, a resolving chain check in the hook and the pusher, a 120-line total, the switch trigger in the pusher's terms as a Deferred item, no README and no fixed files under `docs/handoff/`, the split procedure in one skill file. |
| `0014-critic-review.md` | Companion to 0014: the critic review (one blocking, four serious, five minor; claim 2 fails by experiment, the downstream map loads nothing), all answered in the proposal's last section; every objection accepted, five with a change. |
| `0016-base-owned-and-project-owned-files.md` | Promoted to ADR 0016 (`../decisions/0016-base-owned-and-project-owned-files.md`, with three sibling files: the map, the update procedure, the migration); revised in place after two critic passes: an ownership map as data (`ownership.json`, one base-owned set for every project, project-owned queues and indexes, a departure table per project) so an update touches nothing the project wrote; recommends E, one patch-only update from the base-owned allowlist for decanted and adopted projects alike, the rule sections in base-owned `.claude/agreement/<section>.md` files imported by a short project-owned `CLAUDE.md`, and Hatchery's records in a base-owned subtree `docs/hatchery/` so no mode and no number range remain; the base keeps its `policy.json`. The three owner questions are decided (patch-only everywhere, a 40-line `CLAUDE.md` with `edit.capExempt` as the lift, the pool command per project). |
| `0016-critic-review.md` | Companion to 0016: the critic's first review (two blocking, six serious, five minor; option E proposed) and its second pass on E (one blocking, ten serious, four minor; experiments on `git apply --3way`, the guard and the loader), all answered in the proposal's last section; every objection accepted, most with a change. Promoted with the proposal. |
| `0017-queue-directories-list-on-demand.md` | Promoted to ADR 0017; revised in place after the critic review: no per-file index in `docs/ideas/`, `docs/operator/` and `docs/findings/` (the three the owner named); a listing script `.claude/hooks/queue-list.js` (`--status`, `--untriaged`, `--json`, `--check`) for readers and the `handoff`, `triage`, `decant` and `finding` skills, promised in ADR 0013; the hook's index check switched off by a `Listed on demand:` line in the directory's own README, never by a policy key, so no sibling changes until it drops its table, and the hook warns at the edit when a file would list as `?`; this README and `docs/design/README.md` stay curated; each sibling's migration named. Evidence: the finding of 2026-10-17 on queue indexes and the sibling histories. |
| `0017-critic-review.md` | Companion to 0017: the critic's review (five serious, five minor; A right for the three named queues, wrong on the policy-key switch and on taking `design/proposals/`; dystopo's grammar misread), all answered in the proposal's last section; every objection accepted, six with a change. |

Template:

```
# NNNN. Title

Date: YYYY-MM-DD
Status: draft | under review | promoted to ADR NNNN | withdrawn

## Question
## Constraints that matter here
## Options
### A. ...
### B. ...
## Recommendation
## What would change this recommendation
## Critic's objections and responses
```
