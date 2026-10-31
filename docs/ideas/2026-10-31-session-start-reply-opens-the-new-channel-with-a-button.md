---
date: 2026-10-31
kind: feature
---

# `/session start` takes the driver to the new channel in one click

The driver wants the new session channel brought to the front when they run `/session start`.
Discord gives a bot no way to move a client to a channel (that is client-side), and the owner
does not want it to depend on a local connection (`discord://` deep link). What ticket bots do,
and what is enough here, is server-side: an ephemeral reply with a link button, plus a mention
in the new channel.

## Today

`start` in `internal/discord/adapter.go` answers with `ephemeral(startedText(...))`, and
`startedText` in `internal/discord/text.go` is "Session <id> is open in <#channel>." The
`<#channel>` mention is already a clickable link, so the jump exists as text. `Reply` carries
only `Reply` and `Text`, so it cannot hold a button.

## Want

- The ephemeral start reply carries a link-style button, "Open session", whose URL is
  `https://discord.com/channels/<guild>/<venue channel>`. The panel's Start button reaches the
  same `start`, so it gets the same reply.
- The first message in the new venue channel mentions the driver, so the channel shows an
  unread or mention badge and notifies on every device.

## Open

- `Reply` has no buttons and `Button` (`internal/discord/outbox.go`: `Label`, `CustomID`, `Style`)
  uses a `CustomID`; a link button has a URL and no custom id. The gateway's `answer` (`internal/discord/gateway.go`)
  needs a path for it, and the fake gateway and `Reply`'s JSON (the debug surface) must show it.
- Whether `SessionStarted` (`internal/discord/adapter.go`) already mentions the driver in the
  header; if so only the button is new.
- The reply waits `venueWait` for the channel; if it times out there is no channel id and so no
  button, and the text-only fallback stays.
