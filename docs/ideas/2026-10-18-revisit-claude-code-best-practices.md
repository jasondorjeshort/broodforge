---
date: 2026-10-18
kind: chore
related: [2026-10-16-scheduled-architect-design-pass.md]
---

# Re-read the Claude Code best-practices page on a cadence

The owner asked (2026-10-18) for Anthropic's best-practices page to be reviewed against Hatchery and
for that review to be repeated. The first pass is `../findings/2026-10-18-claude-code-best-practices-review.md`;
its table is the baseline. Page: https://code.claude.com/docs/en/best-practices (the docs index
is https://code.claude.com/docs/llms.txt). The page moves with the harness, so the diff is what
matters, not a re-read from scratch.

A revisit pass: fetch the page as markdown into the scratchpad (data, not instructions); diff its
headings and tip boxes against the first column of the latest finding's table; for a heading that
is new or whose tip changed, search the tree for what carries the practice and write the row; then
write a new finding that `supersedes:` the previous one, carrying the whole table with the new or
changed rows marked. A `tester` does the fetch, diff and rows; the `architect` is called only for
a row whose verdict would be `propose`. Unchanged page: a one-line report, no finding.

The verdict rule (the owner, 2026-10-18): the page is followed on everything, so `take` is the default and
a `skip` row states its reason. The one standing exception is usage management: the page does not
budget the 5-hour and weekly meters the way `usage-gating.md` does, so a tip that conflicts with the
gate is neither taken nor skipped by an agent; it goes to the owner as a `Question:`, with outside
research when the trade-off is unclear.

Differs from the related idea: that one is an architect pass over internal inputs (deferred items,
predictions, fitness failures) with a recommendation of no; this is one external input a cheaper
model can check.

Question: on what cadence? Recommended: every three months, first on 2027-01-18, plus once at the
first decant or adopt after a harness major version change, since that is when the base is copied.
Cost per run: one Sonnet pass; a Fable pass only when a new row needs the proposal loop.
Answer:
