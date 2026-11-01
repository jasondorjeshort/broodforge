# Make the Ended sessions category on dev-live

**Run `setup channels` on the `dev-live` instance so it creates the category `Ended sessions` and
writes its id to `venue.endedCategory`.** An ended session's channel now moves there. The dev-live project was set up before the category
existed, so its key is empty: each end there leaves the channel under Sessions and writes one
`warning: discord: moving the venue of <id>: ...` line to the bot's stderr. The instance is yours;
agents never touch it. The owning doc is `docs/design/04-ux-session-end.md` (Ending a session).

What has to be true first: a `bin\broodforge.exe` built from a checkout that has this change, and
the dev-live bot stopped (setup refuses `running` while it answers).

In PowerShell, in this checkout, as yourself (about five minutes):

```
bin\broodforge.exe stop --root "$env:LOCALAPPDATA\Broodforge\dev-live"
bin\broodforge.exe setup channels --root "$env:LOCALAPPDATA\Broodforge\dev-live"
bin\broodforge.exe discord check --root "$env:LOCALAPPDATA\Broodforge\dev-live"
```

`setup channels` asks nothing when only this category is missing; its line ends `, Ended sessions
<id>`. `discord check` then prints a second `present  category  perms:ok` line, that id. Start the bot
again with `serve` as usual.

Check it took: start a session in the lobby, end it, and watch its channel move under Ended
sessions after the summary; it is still deleted ten minutes later. The lock is expected to fail
(`warning: discord: lock <id>: ...` on the bot's stderr, the channel still writable), since the bot
lacks Manage Roles (ADR 0027); whether to drop the lock or restore Manage Roles is your call.

When done, delete this file and its row in `docs/operator/README.md`.
