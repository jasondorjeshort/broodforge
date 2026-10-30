---
date: 2026-10-29
kind: feature
related: []
---

# Voice sessions from a Discord voice channel

A moonshot the owner judged far off at filing (2026-10-29). A Discord voice channel attached to a
session venue feeds speech-to-text into the session as turns, and the agent's turn summaries (the
closing message, the footer) come back as speech. The vibe programmer of `00-vision.md` has no
terminal; many have no keyboard habit either, and the community already talks in voice. Text stays
the record: every spoken turn is posted to the venue as text first, so the transcript, the scan
and the audit are unchanged.

Anchor: `internal/discord/gateway.go` handles the text gateway only; the Discord library's voice
support and a speech-to-text dependency are both major choices under `major-decisions.md`
(a dependency the whole project leans on), so a proposal before any code.

## Open

- Who in the voice channel is the driver when several people talk; the text venue has the same
  question and answered it with roles and the primary-author rule.
- Where the speech model runs: on the host (a dependency, and CPU on the droplet) or a paid API
  (the owner's money, their call).
- Permission prompts by voice are a bad fit; buttons stay in the text venue.

Waits: the first usable version (`00-vision.md`, "The first usable version") has shipped; the owner, 2026-10-29.
