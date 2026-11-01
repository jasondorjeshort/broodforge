# Make the dev-live notice channel private

**Hide the `dev-live` instance's `#<project>-notices` channel from everyone but the bot and you.**
The bot now posts host paths there (the canonical-diverged fix line), and `setup` makes the
channel private only when it creates one. The channel your first live run made predates that,
so every member of the server can still read it. Only the server owner can change it, in the
Discord client. The owning doc is `docs/design/04-ux.md` (Failure flows, the `notice channel`
row).

Pick one of the two routes.

By hand, in the Discord client, as the server owner (about two minutes):

```
Server, right-click #<project>-notices, Edit Channel, Permissions:
  @everyone          View Channel: deny (red X)
  Advanced permissions, add member <the bot>: View Channel, Send Messages: allow
  Advanced permissions, add member <you>:     View Channel, Read Message History: allow
Save Changes.
```

Or let setup make a new one (about five minutes; the channel's old posts go with it): delete
`#<project>-notices` in the Discord client, then in PowerShell in this checkout, with the
dev-live bot stopped:

```
bin\broodforge.exe setup channels --root "$env:LOCALAPPDATA\Broodforge\dev-live"
```

Answer yes to its replace question (the old channel id, then the new one). Then start the bot
again with `serve` as usual.

Check it took: open the server in a browser where you are signed in as another member, or ask a
member, and confirm `#<project>-notices` is not listed. `bin\broodforge.exe discord check --root
"$env:LOCALAPPDATA\Broodforge\dev-live"` still prints `ok` for the `notice` line (the bot can still
see and post there).

When done, delete this file and its row in `docs/operator/README.md`.
