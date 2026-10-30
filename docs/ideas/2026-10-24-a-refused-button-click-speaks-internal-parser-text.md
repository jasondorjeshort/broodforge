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

Plan: tester, one pass; the mechanism is settled, the wording is the owner's. Files:
`internal/discord/adapter.go` (`click`: the three `bad-args` refusals before the kind switch keep
the parser's text as the audit row's message, through `refuse` as now, and return a `refusal`
whose message is the new text instead, so the audit and the reply part; the venue's session, if
any, comes from `sessionOf(in.Channel)` and its state from `svc.Show`); `internal/discord/text.go`
(`badComponentText(session, state string)`, marked `not 04`, drafted: `This button did nothing:
Broodforge could not read it (a bot bug, logged). Session s000a is still open; /session end ends
it.`, the second sentence dropped outside a venue); `internal/cli/discord_test.go` line 63 and the
`inject click` tests, which pin the parser text today; `docs/design/05-debug-interface-discord.md`
(lines 69 and 148: the reply is the drafted text, the audit row the parser's). Verify: `go test
./internal/discord ./internal/cli`, then `audit --kind action-refused` after an `inject click
--custom-id bf:x` shows the parser text in the row and the new text on stdout. The text goes on
the owner's list in `docs/HANDOFF.md` (Waiting on user) with the pass. Out of scope: a post to the
project channel (the second Open point): the inbound log's `refused bad-args` line is the
operator's signal; the owner may ask for more when reviewing the text.
