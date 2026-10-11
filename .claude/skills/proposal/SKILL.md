---
name: proposal
description: Run the major-choice loop (proposal, critic, revision) before an ADR. Use when choosing a language, UI toolkit, cross-platform strategy, data storage, network protocol or API surface, build system, or a dependency the whole project will lean on, or when asked to write a proposal.
---

# Proposal

A major choice is never made in passing. This skill produces `docs/design/proposals/NNNN-title.md`,
gets it challenged by the `critic`, and ends by handing the revised proposal to the `adr` skill.
The architect writes; the coordinator briefs; the critic argues.

## Steps

1. Allocate `NNNN`: the highest number used across `docs/design/proposals/` and
   `docs/design/decisions/` plus one. Proposals and their ADRs share a number.
2. Write `docs/design/proposals/NNNN-<short-title>.md` from `template.md` in this directory (use
   the Write tool). Status: `draft`. Every option gets the same depth of treatment; the
   recommendation names what would reverse it. Claims about tools, licenses, platforms, or APIs
   are listed under "Claims to verify" so the critic can check them.
3. Add the proposal to `docs/design/proposals/README.md` if that index lists files.
4. Set Status to `under review`. Brief the `critic` with `brief-critic.md` filled in. The critic
   writes `NNNN-critic-review.md` beside the proposal.
5. Revise. In "Critic's objections and responses", number every objection, give the response, and
   state the resolution (accepted, accepted with change, rejected because). Correct factual errors
   in the body in place. Do not delete rejected options; they are the record.
6. If any serious or blocking objection remains unresolved, stop and put both positions to the user
   in `docs/HANDOFF.md` under Waiting on user. Otherwise set Status to `promoted to ADR NNNN` and
   run the `adr` skill.

## Rules

- A proposal for something humans see or touch (front end, flows, wording, visuals) is still
  written, but the decision waits for the user; say so in the recommendation.
- Aim under 300 lines; depth goes into options, not prose. Proposals are exempt from the pusher's
  300-line cap because the responses section is added after the body is complete.
- Skip the critic only for a reversal of an existing ADR that the user has explicitly asked for;
  record the reversal with the `adr` skill as a supersession.
