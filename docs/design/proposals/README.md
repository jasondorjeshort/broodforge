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
