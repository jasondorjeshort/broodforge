# Debug interface: the Discord adapter

This is the Discord part of `05-debug-interface.md` and shares its Conventions.

## The Discord adapter

Drive. `broodforge discord check [--guild <id>]` connects once with the bot credential from the
store, prints `guild <id> <name>` then one line per configured channel `<id>  present|missing
<reach>  perms:ok|missing:<names>` (view, send, embed, attach, add-reactions, manage-messages
for pins and deleting a pasted key, manage-channels for a private-channel venue, manage-threads
later, the thread venue is not in the first version), disconnects, exit 1 on any missing; the credential is used for that
connection and never printed. On a dry-run instance the gateway is never opened and the renderer
writes to the outbox; `broodforge discord render --events <jsonl> [--clock fake]` feeds a file of
`SessionEvent`s into the renderer alone and prints the outbox lines with the virtual time of each
create or edit, so the one-per-second coalescer (R-21) is observable without waiting. Inbound
without Discord: `broodforge discord inject message --channel <id> --user <id> --roles <ids>
"<text>"` and `discord inject click --custom-id <id> --user <id> --roles <ids>` (any custom id
of 03: an answer, `stop`, `end` or `start`) run the adapter's own handlers (command parsing, the
key-paste scan, role check, custom-id parsing) with the roles given in place of a guild lookup;
both are refused on a live instance.
Observe. `session outbox`; `discord check --json`; `audit --kind action-refused`.
Reproduce. `discord render --events tests/discord/burst.jsonl --clock fake --expect
'edits<=1/s'` [no-check: no code yet]; an inbound bug is one `discord inject` line.
Isolation. Dry-run opens no connection; `check` is the only command besides `serve` that
connects, and only to the guild it is told. Not exercisable without a guild the owner creates:
the gateway's reconnect and rate-limit paths (N-25), slash-command registration, real button
rendering, private-channel creation (thread creation: later, the thread venue is not in the first version), the direct-message relay (Discord error `50007`)
and what a human sees. There, `serve --discord live` on a scratch-shaped root is the test: the
owner clicks, the agent reads `session tail` and `audit`.
