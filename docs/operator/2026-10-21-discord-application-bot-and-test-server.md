# Discord application, bot and test server

**Create the Discord application and its bot, a test server with a project channel, a notice
channel and a category for sessions, give the bot its token through the credential store, and
run `discord check`.** It waits on the owner because it needs his Discord account, the bot
token only he may see, and clicks in the Developer Portal and the Discord client. The step and
its reasoning are owned by `docs/design/05-debug-interface-discord.md` (The Discord adapter; The
live gateway) and `docs/design/03-architecture.md` (Configuration).

Runs as: the owner, in a browser for the portal and in the Discord client, then Git Bash in
`<repo>` on the PC with `bin/broodforge` built (`CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`).
First: nothing; the code is in the tree. Time: about 20 minutes.

## 1. The application and the bot (Developer Portal)

1. At <https://discord.com/developers/applications>, New Application, name it (for example
   Broodforge). On General Information, copy the **Application ID**.
2. On Bot: Reset Token and keep the token in the password manager only (it goes into the
   credential store in step 4, never into a file or a chat). Under Privileged Gateway Intents
   switch on **Message Content Intent**; the other two stay off. Public Bot may be off.
3. The intents the code asks for are Guilds, Guild Messages and Message Content (the integer
   33281); only Message Content needs the switch.

## 2. The test server and the invite

1. In the Discord client, create a server for testing. In it create a category for sessions
   (for example `Sessions`), a text channel for the project (for example `#broodforge`, the
   lobby) and a text channel for notices (for example `#broodforge-notices`). Create a role
   (for example `Trusted`) and give it to yourself.
2. Add the bot with this URL, the application id in place of the placeholder; the permissions
   integer is View Channels, Send Messages, Embed Links, Attach Files, Add Reactions, Read
   Message History, Manage Messages, Pin Messages and Manage Channels:

```
https://discord.com/oauth2/authorize?client_id=<application id>&scope=bot+applications.commands&permissions=2251799813811280
```

3. User Settings, Advanced: switch on **Developer Mode**. Then right-click to Copy ID of: the
   server, the project channel, the notice channel, the sessions category and the role.

## 3. A scratch-shaped root and a running bot

```
bin/broodforge scratch new
bin/broodforge --root <root> serve --discord dry-run
```

`scratch new` prints `<root>`; leave `serve` running in that shell and use a second one below.

## 4. The token into the credential store

The value is read from stdin, never an argument; `read -rs` keeps it out of the history:

```
read -rs BF_TOKEN && printf '%s' "$BF_TOKEN" | bin/broodforge --root <root> credential add --instance --kind bot-token --label bot; unset BF_TOKEN
bin/broodforge --root <root> credential list
```

It prints the new id `cred-...`; the list shows it as `bot-token` with the user `-`.

## 5. The configuration

Stop the bot (`bin/broodforge --root <root> stop`), then edit `<root>/config/instance.json`:

```
"discord": { "mode": "live", "applicationId": "<application id>", "guildId": "<server id>",
             "botCredentialId": "<cred id from step 4>", "noticeChannelId": "<notice channel id>" },
"roleGroups": { "trusted": ["<role id>"] }
```

and `<root>/config/projects/scratch.json`: `"venue": { "category": "<sessions category id>",
"summaryChannel": null }`, and in `channels` the key `100` replaced by `<project channel id>`
(drop `200`, or replace it with a second channel's id).

## 6. Check, then serve live

```
bin/broodforge --root <root> config check
bin/broodforge --root <root> discord check
bin/broodforge --root <root> serve --discord live
```

`discord check` prints `guild  <id>  <name>`, then every channel `present` and `perms:ok`,
the category and the notice channel included, and `message-content  present  intent`; exit 0.
A line `missing` names what to fix (a permission, a channel id, the intent switch). `serve
--discord live` prints `serving  <socket>`; the bot shows online and the slash commands appear
in the server.

When done: delete this file, and send the coordinator the `discord check` output for P12's
Proof (`docs/design/08-implementation-plan.md`); the owner's first live session there is its
second half.
