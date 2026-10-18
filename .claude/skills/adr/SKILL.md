---
name: adr
description: Write an architecture decision record or supersede an existing one, and update the design indexes. Use when asked to record a decision, write an ADR, promote a proposal, or supersede or reverse an earlier decision.
---

# ADR

One file per decision in the directory step 1 names, `NNNN-short-title.md`, under 150 lines, written so
someone can implement from it without asking. Non-obvious decisions get an ADR even without a
proposal; major choices arrive here through the `proposal` skill.

## Write a new ADR

1. Number: if promoting a proposal, reuse its `NNNN`; otherwise the highest number across the
   decisions directory and `docs/design/proposals/` plus one. One test names the directory
   (ADR 0016): Hatchery itself, where `git remote | grep -qx hatchery` fails, writes in
   `docs/hatchery/decisions/`, from `0016` up with no ceiling; every other tree writes in
   `docs/design/decisions/`, from `0001` or the next free number, proposals included, and never
   writes in `docs/hatchery/`: it supersedes a Hatchery record by citation from its own ADR
   (`supersedes docs/hatchery/decisions/NNNN-... for this project`).
2. Write from `template.md` in this directory with the Write tool. Status `accepted` (the
   coordinator settles technical decisions after the critic step; `proposed` only when the user
   must decide, which is anything humans see or touch).
3. Context says what forced the decision and what was verified, with sources. Decision is concrete:
   names, paths, data shapes, boundaries, failure modes, and every deferred item with the trigger
   that revives it. Consequences include the costs, not only the benefits.
4. If anything exceeds 150 lines (specs, tables), put it in a sibling file
   `NNNN-<topic>.md` and link it from the ADR.
5. Index: add a line to the 'Decisions so far' list of the directory's index:
   `docs/hatchery/decisions/README.md` in Hatchery, `docs/design/README.md` elsewhere.
6. If promoted from a proposal, set the proposal Status to `promoted to ADR NNNN`.

## Supersede an ADR

1. Before writing, record the outcome of each prediction on its line in the old ADR (the one edit
   a closed ADR takes); then write the new ADR as above. In its Context, name the ADR it replaces
   and what changed.
2. In the old ADR set `Status: superseded by NNNN` and add one line under the status pointing at
   the new file. Do not edit the old body beyond those outcome lines; it is the record.
3. Update the directory's index (step 5 of the main procedure) so the old entry reads "superseded by NNNN".
4. Partial supersession (one section of a long ADR) is still a whole new ADR; say which sections of
   the old one still stand.

## Amend without superseding

Only for tables the ADR itself says are living (ADR 0001 components table is one). Edit the table,
nothing else, and say so in the commit message.
