# Proposals

Drafts for major choices. One file per proposal: `NNNN-short-title.md`, numbered to match the
ADR it will become. A proposal is written by the architect, challenged by the critic, then
revised and promoted to `../decisions/NNNN-short-title.md`. Keep the proposal file; it is the
record of the alternatives and the objections.

| File | Status |
|---|---|
| `0002-repository-layout.md` | Promoted to ADR 0002 (`../decisions/0002-repository-layout.md`); revised in place after the critic review. |
| `0002-critic-review.md` | Companion to 0002: the critic's review, answered in the proposal's last section. |
| `0003-wave-workflow.md` | Draft: the build-phase workflow Dystopo converged on (task board with Owns lines, worktree per task, verifier per PR, one-command sweeps, publish after every green sweep). Awaits the critic. |
| `0005-rules-that-bit-in-dystopo.md` | Draft: four working rules Dystopo needed that the base lacks (no junctions into worktrees, no branch delete before a verified merge, no `--no-verify`, docs in the same PR). Awaits the critic. |

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
