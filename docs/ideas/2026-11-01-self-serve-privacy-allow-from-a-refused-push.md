---
date: 2026-11-01
kind: feature
related: [2026-11-01-questions-answered-from-discord.md, 2026-10-30-carry-back-privacy-scan-at-commit-time.md]  # the first is the same owner-channel pattern; the second catches the string earlier
---

# Self-serve privacy allow from a refused push

Recorded from the owner's conversation of 2026-11-01 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

Two pushes on 2026-10-31 and 2026-11-01 stopped on false positives of the home rule, and each took
a round trip through the coordinator to the owner and back: the owner said "allow it", the
coordinator edited `privacy.allow`, a runner committed, a pusher relaunched. The bot would post the
hit to the owner channel (file, line, class, the segment it keyed on, never the full string when
the class is a name) with an Allow button and a Fix-it button. Allow edits the list, commits it with
the owner as the actor in the audit, and relaunches the push; Fix-it opens the file in a session.
The allow list stays the owner's: the click is the owner's.

Anchor: `.claude/hooks/privacy-scan.js` (`--json` output, the hit's `match`); `.claude/hooks/policy.json`
(`privacy.allow`); `internal/discord/customid.go` (the buttons); the pusher's check 5 in
`.claude/agents/pusher.md`.

## Open

- A push made by the bot's own landing (ADR 0028) runs the same scan; the button belongs on that
  refusal first, since there is no coordinator in the loop at all.
- A name class hit must never be allowed by a button; only word, host and home, as the owner's
  user allow file already limits.
