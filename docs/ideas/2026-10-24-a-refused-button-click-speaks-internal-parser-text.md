---
date: 2026-10-24
kind: bug
related: [2026-10-23-session-land-pushes-to-main-mid-session.md]
---

# A refused button click answers in the parser's words

The owner's End session click on s000a got, as an ephemeral reply: "a custom id of 12 bytes starting
"bf" is not a Broodforge custom id". The cause is fixed (base-36 ids, see
`docs/findings/2026-10-24-s000a-post-mortem-end-session-refused-for-a-base-36-id.md`), but the reply
is still what any `bad-args` from `internal/discord/customid.go` would show: internal text, no word
that the session is still open, nothing to do next. The owner read it as an oddity, not as "your
session did not end", and the commits sat unpushed.

A button the bot itself posted can only fail to parse through a bot bug, so the reply could say so in
04's voice (that the click did nothing, the session's state, and the typed fallback, `/session end`)
and the audit row keeps the parser's text.

## Open

- The wording is the owner's (04, what humans see).
- Whether a `bad-args` on a bot-posted component also posts to the project channel, since it is a bug.

Plan: tester, after P12d (its "every refusal audited" step touches the same `click` refusals).
`internal/discord/adapter.go` (`click`: the `bad-args` refusals audit the parser's text and return a
`refusal` with the new; session via `sessionOf`, state via `svc.Show`); `internal/discord/text.go`
(`badComponentText(session, state string)`, marked `not 04`: `This button did nothing: Broodforge
could not read it (a bot bug, logged). Session s000a is still open; /session end ends it.`, second
sentence dropped outside a venue); `internal/cli/discord_test.go` line 63 and the `inject click`
tests; `05-debug-interface-discord.md` 69, 148. Verify: `go test ./internal/discord ./internal/cli`;
`inject click --custom-id bf:x` shows the new text, `audit --kind action-refused` the parser's. Text
goes on the owner's list (`docs/HANDOFF.md`). Not a channel post; the `refused bad-args` inbound
line is P12d's, not present today.
