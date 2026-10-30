---
date: 2026-10-30
kind: chore
related: [2026-10-21-carry-back-brief-boundary-escape-classes.md]  # that one makes the brief list escape classes; this one makes the reviewer's output a test; both touch the reviewer bullet and Rules
---

# Carry-back: a reviewer finding is a failing test

A departure from the base, recorded as a row in ADR 0024, to go back to
Hatchery. Two base-owned files changed. `.claude/agents/reviewer.md` gained a Rules bullet: a
finding that can be stated as a test is delivered as one (exact text, file and name, for the coder
to add verbatim; the reviewer has no Write or Edit tool, so writing it is reporting), so the fix is
"make it pass" and the case stays as the regression guard; a finding that cannot be a test says so
in one clause. `.claude/skills/brief/SKILL.md` gained a clause in the reviewer bullet asking for
findings as failing tests.

Why: on 2026-10-30 seven reviews each found a real defect (the summary printing the PR's API URL,
a swallowed `SavePanel` error, ten `git push -f` bypasses of the quoted-git guard, a false byte
count in a proof). Each came back as prose and was re-derived by the coder before it could be
fixed; a test carries the repro across.

Anchor: the Rules in `.claude/agents/reviewer.md`; the reviewer bullet of `.claude/skills/brief/SKILL.md`.

Plan: carry-back (Send) once carry-back/brief-escape-classes has landed in Hatchery, since both touch the reviewer bullet and the reviewer agent's Rules; then the row's Carry-back cell gets the branch and sha.
