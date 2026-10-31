---
date: 2026-10-31
kind: chore
related: [2026-10-23-session-land-pushes-to-main-mid-session.md]  # that one is the feature; this is the agreement departure it creates
---

# Carry-back: when a bot measures the review, the review is the last step

A departure from the base, recorded as a row in ADR 0024 (`../design/decisions/0024-decanted-from-hatchery.md`),
to go back to Hatchery as a request with no diff: the agreement file `.claude/agreement/one-review.md`
is untouched here. Its practice is review, fix, amend, push: "a fix to an unpushed commit is amended
into it" and "a second reading of the same diff by the same model finds what the first did".

ADR 0028 (`../design/decisions/0028-landing-a-session-on-the-default-branch.md`, Decision 6) gates a
land on the default branch by a fact the bot measured: the sha a `reviewer` agent was launched on
equals the sha pushed. A fix after the review amends the commit, moves the sha and voids the fact, so
for a `main` land from a Discord session the review is the last step and a fixed change is reviewed
again on the amended sha. The bot cannot tell a fix from new work without reading content, which it
never does (ADR 0025 rule 2). The cost is a second reviewer run per fixed change on that path; the
interactive checkout keeps the agreement's practice.

Anchor: `.claude/agreement/one-review.md` (the amend sentence); `docs/design/decisions/0028-landing-a-session-on-the-default-branch.md`.

Plan: carry-back (Send) as an agreement request once P34 has landed here and the first `main` land
has run, so the request carries the evidence: one sentence in `.claude/agreement/one-review.md` saying that where a
tool measures the review against a commit, a fix after the review is reviewed again, and the row's
Carry-back cell gets the branch and sha.
