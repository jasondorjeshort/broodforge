---
date: 2026-11-01
kind: chore
related: [2026-10-30-carry-back-a-reviewer-finding-is-a-failing-test.md]  # that one asks for a finding as a failing test; this lands the test itself
---

# A reviewer's tests ship with the fix

Recorded from the owner's conversation of 2026-11-01, not sanity checked: nobody has read the
anchor against the claim yet.

Three reviews in the wave of 2026-10-31 handed back failing tests in prose (the diverged-notice
command, the concurrent trailer append, the failed credential list), and the coders retyped each
into the tree. The reviewer would instead write its test files straight into the tree under a
`review_test.go` name beside the code, leave them failing, and the fix brief becomes "make these
pass, then rename or merge the file". Nothing is retyped, every finding leaves a regression guard,
and the coordinator can see the finding run. Base-owned, so a Hatchery carry-back.

Anchor: `.claude/agents/reviewer.md` (Rules: it reports and does not fix; a test file is a report in
another form); `.claude/skills/brief/SKILL.md` (the reviewer bullet and the coder fix brief).

## Open

- A reviewer that writes into the tree can no longer be said to have left it untouched; the runner's
  stage-by-name commits make that safe, but the agreement's wording needs the exception.
- Whether the test file stays under the review name or is folded into the package's tests.
