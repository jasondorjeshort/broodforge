---
date: 2026-10-31
kind: feature
related: []
---

# Role-upgrade evidence digest

Recorded from the owner's conversation of 2026-10-31 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

R-05 of `docs/design/01-requirements.md` says an agent may recommend a role upgrade to the owner
and never apply one; it is marked later and has had no idea file. The audit already holds every
session, commit, refusal, review and spend per person. A nightly digest to a moderator channel
says who landed what, who hit refusals and how often, and recommends upgrades, with the evidence
beside each line. Roles stay the humans' to change in Discord, as `00-vision.md` says.

Anchor: `internal/audit/audit.go` (the rows the digest reads); `internal/store/usage.go` (spend
per person); R-05.

## Open

- A recommendation built from counts is gameable by the troll tier the vision describes: many
  small harmless sessions look like trust.
- Which channel, which role sees it, and whether the person named sees their own line.
- Whether an engine writes the digest (tokens, judgment) or the bot formats counts (free, blunt).
