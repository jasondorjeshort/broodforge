---
name: adr
description: Write an architecture decision record or supersede an existing one, and update the design indexes. Use when asked to record a decision, write an ADR, promote a proposal, or supersede or reverse an earlier decision.
---

# ADR

One file per decision in `docs/design/decisions/NNNN-short-title.md`, under 150 lines, written so
someone can implement from it without asking. Non-obvious decisions get an ADR even without a
proposal; major choices arrive here through the `proposal` skill.

## Write a new ADR

1. Number: if promoting a proposal, reuse its `NNNN`; otherwise the highest number across
   `decisions/` and `proposals/` plus one, within the range the tree is allowed (ADR 0008):
   Hatchery itself uses `0001` to `0019`; a project decanted from Hatchery starts at `0020`, so its
   first ADR is `0020` or the next free number above it, never one in Hatchery's range even when
   that is free; a project that adopted Hatchery keeps its own numbering. A proposal takes the
   number of the ADR it will become. Which case this tree is: Hatchery itself has no `hatchery`
   remote; decanted means `git merge-base HEAD hatchery/main` succeeds; adopted means a
   `Hatchery base:` line exists under `docs/design/decisions/`. If both tests pass (an adopted
   project whose history was later joined to the base), the `Hatchery base:` line wins and the tree
   numbers as adopted (ADR 0009).
2. Write from `template.md` in this directory with the Write tool. Status `accepted` (the
   coordinator settles technical decisions after the critic step; `proposed` only when the user
   must decide, which is anything humans see or touch).
3. Context says what forced the decision and what was verified, with sources. Decision is concrete:
   names, paths, data shapes, boundaries, failure modes, and every deferred item with the trigger
   that revives it. Consequences include the costs, not only the benefits.
4. If anything exceeds 150 lines (specs, tables), put it in a sibling file
   `NNNN-<topic>.md` and link it from the ADR.
5. Index: add a line to the "Decisions so far" list in `docs/design/README.md`. If
   `decisions/README.md` lists files, add it there too.
6. If promoted from a proposal, set the proposal Status to `promoted to ADR NNNN`.

## Supersede an ADR

1. Write the new ADR as above. In its Context, name the ADR it replaces and what changed.
2. In the old ADR set `Status: superseded by NNNN` and add one line under the status pointing at
   the new file. Do not edit the old body; it is the record.
3. Update `docs/design/README.md` so the old entry reads "superseded by NNNN".
4. Partial supersession (one section of a long ADR) is still a whole new ADR; say which sections of
   the old one still stand.

## Amend without superseding

Only for tables the ADR itself says are living (ADR 0001 components table is one). Edit the table,
nothing else, and say so in the commit message.
