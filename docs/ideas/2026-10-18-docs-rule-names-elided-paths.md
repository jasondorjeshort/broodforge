---
date: 2026-10-18
kind: chore
---

# The docs rule should say an elided path is not read as a path

The first carry-back (ADR 0019, landed from a downstream build-phase project's branch) changed
`post-edit.js` so the path-anchor check skips a token with a `...` segment (`src/.../X.kt`), and
carried a one-line edit to `.claude/rules/docs.md` saying so, next to the `[no-check]` sentence.
The Land review dropped that hunk: a change under `.claude/rules/` is a rule change, refused at
review and filed here (carry-back skill, Land step 4). The behaviour is recorded in the hook
specs (`docs/design/decisions/0002-hook-specs.md`, path anchors); the rule file does not list
the hook's other skips (`*`, `<slug>`) either.

Plan: decide whether `docs.md` names the skipped forms at all. If yes, one sentence after the
`[no-check]` sentence listing `*`, `<slug>` and a `...` segment as tokens the check does not read;
a `tester` edit in Hatchery. If no, delete this idea.
