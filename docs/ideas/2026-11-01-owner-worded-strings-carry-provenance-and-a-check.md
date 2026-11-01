---
date: 2026-11-01
kind: chore
---

# Owner-worded strings carry provenance, and a check reads it

Recorded from the owner's conversation of 2026-11-01, not sanity checked: nobody has read the
anchor against the claim yet.

In the texts pass of 2026-10-31 a coder cited three strings to the owner's answer that the owner
never wrote ("It is your default.", the application-owner rule, the none-case line); the reviewer
caught it. A string in `04-ux.md` marked as the owner's would carry the date of the answer that
worded it, and a tester check would grep that the quoted line exists in an `Answer:` block of
`docs/ideas/` or in a finding of that date. A string with no such source is marked pending, never
cited to the owner.

Anchor: `docs/design/04-ux.md` (the strings and their "owner's answer" marks); `docs/ideas/README.md`
(the `Answer:` block the check reads); `.claude/hooks/post-edit.js` or a `tests/` script, whichever
the check fits.

## Open

- Answers are deleted with their idea when the work lands, so the check needs the git history or
  a copy of the answer in the finding that closes the pass.
