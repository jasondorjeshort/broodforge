# 0027 companion: the `setup` steps, the rerun rule, the refusals, `setup host`, `setup login`, `setup service`, `config export`

Companion to `0027-setup-in-few-steps.md` (Decision 1, 4 and 5). The ADR holds the choice, the guild
objects, the single-writer rule, the owner's texts and the amendments; this file holds the command
surface a coder implements from, under `../05-cli-conventions.md`. Source: `../proposals/0027-setup-in-few-steps.md`.

## `broodforge setup`

`broodforge setup [<step>] [--root <path>] [--instance <name>] [--repo <url>] [--project <id>]
[--guild <id>] [--channels yes|no] [--create-role yes|no] [--role <name>] [--from <export>]
[--discord live|dry-run] [--check] [--yes] [--wait <seconds>] [--json]`. Every prompt has a flag; `--yes` takes every shown default
and refuses `needs-input-<step>` where there is no default:

| Flag | The prompt it answers | Default shown | With `--yes` and no flag |
|---|---|---|---|
| `--root`, `--instance` | the root and the instance name (`root`) | `%ProgramData%\Broodforge\<instance>` or `/srv/broodforge/<instance>`; `main` | taken |
| `--repo` | the repository URL (`config`) | none | `needs-input-repo` |
| `--project` | the project id, only when the derived one is invalid (`config`) | derived from the repository name | `needs-input-project` |
| `--guild` | which guild, only when the bot is in several (`guild`) | none | `needs-input-guild` |
| `--channels yes\|no` | create channels in a guild that already has some (`channels`) | no | no: `channels` waits for the owner's channels by name |
| `--create-role yes\|no` | let setup create the role (`guild`, before the link) | yes | yes |
| `--role <name>` | none: the role name `setup` looks up or creates | `Trusted` | taken |
| `--wait <seconds>` | none: the bound on every poll | 900; `0` is unbounded | taken |
| `--from <export>` | none: `config import` first, then only the host-bound and secret steps ask | none | - |
| `--discord live\|dry-run` | none: whether the Discord steps call Discord (`live`) or write the development placeholders and call nothing (`dry-run`, The dry-run form below) | `live` | taken |

Off a terminal every question is refused `needs-input-<step>` naming its flag, so a token on stdin
needs `--yes`, with three exceptions: the git-credential question is deferred, not refused (it writes
nothing; `setup git-credential` adds it later), the token refusal reads `pass the bot token on
stdin with --yes`, not `pass --<flag>`, and `setup git-credential` with empty stdin refuses
`needs-input-git-credential: pass the token on stdin` (`internal/setup/steps.go`).

Names `setup` looks up or creates and never stores (03's config holds ids): the role (`--role`), the
category `Sessions`, the lobby `#<project id>`, the notices channel `#<project id>-notices`. Config values
written with a default and no prompt: timeouts 0, mode `prompt`; `discord.mode` is written `dry-run` and
becomes `live` in the role step of the first run in which every Discord id is present and the role is held.

Output: one line per step, `ok  <step>  <detail>`, `skip  <step>  <already>` or `fix  <step>  <what
the owner does>`; `--json`: one object each, `{"result":"ok|skip|fix","step":"<step>","detail":"<text>"}`.
The `key` step's object adds `"key":"<hex>","path":"<path>"`, the one print, never repeated. A `fix`
that needs a hand pauses: interactive, or with `--wait`, `setup` polls until the step passes or the
wait runs out. `setup <step>` runs one step alone and may replace that step's keys after printing
`old  <value>` and `new  <value>` records (`--json`: `{"step":"<step>","old":<value>,"new":<value>}`)
and asking (`--yes` accepts): the only way `setup` overwrites a present key; for `token`,
`git-credential` and `key` the values are the credential id and the key path, never a secret. One
secret per command: the bot token on `setup`'s stdin, the git credential hidden in the interactive
run or on `setup git-credential`'s stdin alone.

Exit codes: 0 every step `ok` or `skip`; 1 a step failed, its `fix` line on stdout and one stderr
`error: <code>: <message>` line from the refusal table (`--json`: the wire error object); 2 usage.
`--check` prints every step's line, changes nothing, binds nothing, polls nothing, and exits 1 with
`error: check-failed: <steps>` naming the steps that printed `fix`.

The root: `setup` and `setup host` are the exception to 05's missing-root rule; with no `--root` and
no `BROODFORGE_ROOT` the default root applies. Every other command, `setup login`, `setup service`,
`config export` and `config import` included, exits 2 without one. `setup` (not `--check`) is
neither `(file)` nor a client: it joins `serve` in 05's exception list as a listener, binding the
control address before its first write and holding it until exit; its `status` reply is `config:
setup` with `-` for every other line (`--expect setup` matches). `setup --check` is `(file)`.

Steps, in order:

- `root`: the directories.
- `config`: asks the repository URL (`--repo`); the project id is the repository name lowered, runs
  outside `[a-z0-9-]` made `-`, collapsed, cut to 32, asked for (`--project`) when it does not start
  with a letter (03's pattern); writes `instance.json` and `projects/<id>.json` with every key
  commented, each only when absent; the repository URL is kept in the project file as the comment
  key `"//url"`, the one place the schema allows, in the `repo` object, so a rerun after a refused clone needs no `--repo`.
  The question is here because the id names the project file,
  the channels and the clone.
- `key`: `credential init`; `ok  key  <hex>  <path>` (the one print) then the pause until Enter;
  `skip  key  exists` for a valid key (64 hex digits; surrounding whitespace is trimmed, so a CRLF or no
  newline is valid).
- `engine`: path and version.
- `token`: read hidden, or from stdin, stored as the instance's `bot-token`; `skip` when the stored
  one answers `GET /oauth2/applications/@me`; refused, `fix  token  Discord refused the stored
  token: <reason>` and the token prompt again (`--yes` with no stdin token: `needs-input-token`).
  A 401, or a value without a token's shape (no call made), means refused; any other 4xx ends the run
  `refused-token`; a 5xx or 429 is no answer, a waiting `fix` that keeps the stored token.
- `application`: the id from the token's application; the flags ORed with
  `GATEWAY_MESSAGE_CONTENT_LIMITED` (1<<19) by `PATCH /applications/@me`, `skip` when 1<<18 or
  1<<19 is set, `fix` naming the portal switch when the PATCH is refused.
- `guild`: one membership taken; none, the invite link printed and a poll until the bot joins;
  several, a list (disgo `OAuth2.GetCurrentUserGuilds("")`, the empty token meaning the bot's).
  The role question is asked here, before the link is printed, so the link carries Manage Roles
  only on a yes. The invite's permission integer with Manage Roles is 2251800082246736.
- `channels`: the configured ids first, then by name, then create (the ask when the guild has
  channels), the ids written. The lobby is the channel named for the project or the entry setup
  wrote, never any configured id; under `setup channels` gone non-lobby ids are dropped.
- `role`: the configured id first, then by name; absent and the answer no (or the bot already in
  the guild without Manage Roles), a `fix` that waits for the owner to make it; yes, the default,
  create, check it sits below the bot's highest role, assign to the guild's `owner_id`; then, on
  every run, the Manage Roles check below. A role at or above the bot's highest prints a `fix`
  line, then `mismatch-role`; Administrator on the bot prints its own `fix` (remove it in Server
  Settings, Roles; it holds Manage Roles); Discord not answering is a waiting `fix`.
- `git-credential`: the token read hidden, or Enter to defer, which writes nothing; stored as an
  instance credential with no user (`AddGit`), none being registered at setup time; before the
  clone, so a private canonical clones.
- `project`: the canonical cloned from the repository into `projects/<id>/`, the stored git
  credential in that one git process through `GIT_ASKPASS` (ADR 0022's push route); `skip` when
  present with the same remote; refused, `fix  project  <git's line>; add the token with setup
  git-credential and rerun`.
- `defaults`: timeouts 0, mode `prompt`.
- `check`: `config check`, `spawn preflight`, `discord check`, each line echoed.
- `next`: the one closing line.

## The dry-run form (`--discord dry-run`)

The development instance's form (`../05-debug-interface-dev-instance.md`; P27). The five Discord steps
call nothing: `token`, `application` and `guild` print `skip  <step>  dry-run`; `channels`, when the
project file has no channel, writes two placeholder channels, `100` (reach `code`, mode `prompt`,
harness `fake`) and `200` (the same, harness `claude-code`), `start`, `contribute` and `answer` all
`trusted`, `venue.category` left empty, and writes `engine.fakeFixtures` as `fixtures/engine` under the
effective install directory (the parent of `bin/`, as the spawn backend resolves an empty
`paths.install`: the binary built in a checkout); `ok  channels  dry-run 100 fake, 200 claude-code;
fixtures <path>`. No `fixtures/engine` there is `fix  channels  build the binary in a checkout: no
fixtures/engine beside bin/`, not waiting, so the run ends `refused-channels` (the inject route cannot
pass a fixture path, so the form has no half state). `role`, when `roleGroups.trusted` is empty, writes
the placeholder role `["10"]` (`ok  role  dry-run 10`). The ids `100`, `200` and `10` are the scratch
template's, never Discord's (a snowflake has 17 digits or more), which is how a later run tells them
from the owner's. Under the form the `engine` step looks on `PATH` first and at the pinned place
(`%ProgramData%\Broodforge\engine\`) last, the reverse of the live order, so a development root keeps
the owner's own engine after `setup host`. `check` has no `discord check` line; `discord.mode` stays
`dry-run`; `next` prints the dry-run closing text. On a root whose `discord.mode` is `live` the flag is
refused `live-instance` in `Run`, beside the scripted-Discord check, before the control address is bound
and before `root` writes anything; so is a `--from` export whose instance is live. Another value is a
usage error, exit 2, as for `--channels`; set beside `BROODFORGE_SETUP_FAKE_DISCORD`, `dry-run` wins and
the script is not read; `--check` and `setup <step>` take the flag with the same meaning. Its `--check`
fix texts are `the dry-run channels are not configured; run setup --discord dry-run`,
`roleGroups.trusted is not set; run setup --discord dry-run` and, for `next`, `run broodforge setup
--discord dry-run` (interim wording, the owner's review pending). The flag maps to the runner's
no-Discord path (`Env.Discord` nil) plus the writes above; the one rule outside the form is the rerun
rule's placeholder entry: a `--discord live` run replaces the placeholders without asking and without
`old` and `new` records, `--yes` or not, since setup wrote them itself.

## The rerun rule, per key

- A config key: absent, written; present and the host or guild agrees, `skip`; present and they
  disagree, `mismatch-<step>` (the table), replaced only by `setup <step>`. One exception: a
  stored bot token Discord refuses is replaced in the run itself, a new token being a new
  credential id, `discord.botCredentialId` rewritten to it and the old entry `removed`.
- The key file: absent, created; a valid key, `skip`; anything else (a placeholder, a short or
  partial file), `mismatch-key` naming the path. `setup key` replaces a key file only while
  `credentials/store.json` is absent or holds no entry; with entries it refuses `key-in-use`, since
  they would be unreadable under a new key (ADR 0022: the key from the password manager).
- A credential: never overwritten; a new value is a new id, the old one `removed`.
- The channels (category, lobby, notices): a configured id present in the guild is `skip`,
  whatever its name now; a configured id gone from the guild is `mismatch-channels` (`setup
  channels` finds by name or creates, then rewrites the id); no id, by name, then create.
- The role: `roleGroups.trusted`'s first id present is `skip`; gone, or not a list of ids,
  `mismatch-role` (`setup role` finds by name or creates, then rewrites it); no id, by name (found: the id written, `ok`;
  absent: the create branch).
- Manage Roles: every run, `--check` included, reads the bot's guild permissions once the role is
  settled; held, the removal `fix` is printed and polled to `--wait` (`--check`: printed only). A
  Ctrl-C during the wait is caught by the next run.
- A channel or role it found is never deleted or renamed.
- The placeholders of the dry-run form (channel ids `100` and `200`, role id `10`): on a `--discord
  live` run they count as no id, never `mismatch`: `channels` finds by name or creates, writes the
  lobby and drops them as it drops gone non-lobby ids; `role` writes the real id in their place;
  neither asks, prints `old` and `new`, or waits on `--yes`. The one case a plain run replaces a
  present value, because setup wrote it itself as a placeholder and it was never the owner's.

## Refusals

Each is one stderr line `error: <code>: <message>`, exit 1 (`--json`: the wire error object), except
`config import`, which prints one `error: bad-config:` line per problem (`internal/cli/config.go`); a
`fix` line for the step precedes it on stdout where one applies.

| Code | Message | Lifted by, from which shell |
|---|---|---|
| `running` | `a bot answers at <address>; stop it with broodforge stop in this shell (every open session is interrupted) or use setup --check` | the owner, `broodforge stop` from the same shell: the PC's elevated PowerShell, a shell as `broodforge` on Linux; never for `--check` |
| `setup-running` | `another setup holds <address>; finish it or press Ctrl-C there` | the owner, in that terminal |
| `needs-input-<step>` | `pass --<flag>` | a flag, or an interactive run |
| `mismatch-<step>` | `config has <X>, found <Y>; run setup <step> to replace it` | `setup <step>` with the bot stopped: from the elevated PowerShell on the PC (relaunched as `broodforge`, below), `sudo -u broodforge` on Linux |
| `refused-<step>` | `<Discord's reason>; <the portal or server setting>` | the owner in the portal or server, then a rerun; a refusal a rerun can lift is a `fix` line and a poll instead, never this code; a non-waiting `fix` in a normal run also becomes `refused-<step>`, carrying the fix's own detail (`internal/setup/setup.go`) |
| `timeout-<step>` | `still waiting after <n>s: <what to do>` | a rerun, or `--wait 0` |
| `needs-elevation` | `run setup <host\|login> from an elevated PowerShell` or `... with sudo`; on the PC also `<root or pipe> is denied; run this from an elevated PowerShell`, on Linux `<addr> is denied; run this with sudo` | the owner, from that shell |
| `key-in-use` | `credentials/store.json holds <n> entries under the present key; restore it from your password manager or remove them first` | the owner |
| `socket-path` | `<the host's Listen failure>; choose a shorter --root` (pending the owner's review) | a shorter `--root` |
| `refused-project` | `<git's line>; add the token with setup git-credential and rerun` | the owner, then a rerun |
| `live-instance` | `the scripted Discord (BROODFORGE_SETUP_FAKE_DISCORD) is refused on a live instance`; for the flag, `--discord dry-run is refused on a live instance` (both pending the owner's review) | a root whose `discord.mode` is `dry-run` |
| `bad-config` | `<file>: <part and key the schema rejects>` (an export under `config import`, an instance file that does not parse; an OS read error is `internal`) (pending the owner's review) | the owner fixes the file |
| `mismatch-config` | `export <file> has <part: problem>, ...; fix the file or export it again with this version` (a refused `--from` export; interim, pending the owner's review) | the owner fixes the file or exports again, from any shell |
| `no-config` | `<file> is missing; run setup` (the instance file or the project's, on a run that writes; `--check` prints a `fix` line instead) | `setup`, from the shell that runs it |
| `key-unreadable` | `<key path>: <the OS error>` | the owner fixes the file's access or path |
| `bad-args` | `the repository URL carries a credential; pass it without the user and password, and add the token with setup git-credential` (also an unreadable `--from`: `--from: <error>`; an invalid instance name; an invalid `--project`; a guild id the bot is not in; a bad `BROODFORGE_SETUP_FAKE_DISCORD` script) | a rerun with the argument corrected (a plain URL, a valid name) |
| `check-failed` | `<steps>`: `--check` names the steps that printed `fix`; a normal run names `check` when its echoed lines failed | the listed steps' `fix` lines |
| `login-failed` | `the engine's login exited <code>` | a rerun of `setup login` |
| `config-exists` | `<root>/config/instance.json exists` (`config import`) | the owner, an empty root |

## The owner's texts

Accepted by the owner on 2026-10-26 (ADR, Decision 6); four changed by the review fixes of the same
day, marked pending the owner's re-approval. Token: "Bot token (Developer Portal, Bot, Reset Token; not shown as you type):". Key:
"ok  key  <hex>  <path>" (was "key  <hex>  <path>", now a step line) then "Copy the key into your
password manager, then press Enter.". Invite: "fix  guild  Open this link, pick the server, Authorize: <url>" (was "Open this link, pick the
server, Authorize: <url>"; pending the owner's re-approval) then "waiting for the bot to join". Role: "Let setup create the role "Trusted"? This adds Manage
Roles to the invite link; setup asks you to remove it again at the end. [Y/n]"; on no "fix  role
create a role named Trusted in Server Settings, Roles, and give it to yourself; setup is waiting";
on yes "fix  role  remove Manage Roles from the bot's role in Server Settings, Roles; setup is
waiting". Channels: "This server already has channels. Create Sessions, #<project id> and
#<project id>-notices in it? [y/N]". Repository: "Repository URL of the project (its canonical):".
Git credential: "Token for pushes to it (not shown; Enter to add later with setup
git-credential):". Closing: "ok  next  the bot is running. In Discord: /register, then /credential add
kind:subscription; then here: broodforge setup login <credential id>" (was "next  the bot is running.
In Discord: ..."; pending the owner's re-approval). Refusal `running`: "error:
running: a bot answers at <address>; stop it with broodforge stop in this shell (every open session
is interrupted) or use setup --check" ("in this shell" added); the other refusal texts as the table.
Closing under `--discord dry-run` (2026-10-30, interim wording, the owner's review pending): "ok  next
the dry-run instance is ready: broodforge serve --discord dry-run starts it; a rerun with --discord live
adds Discord".

## On the PC: which shell

The root's ACL and the pipe's default security descriptor (`CreateNamedPipe` with no attributes:
full control to SYSTEM, Administrators and the creator, read only for Everyone; the client opens
read-write) keep the owner's unelevated shell out: a read of the root is denied and the pipe is
`connect-failed`. The descriptor stays as it is: Administrators are already admitted and the pipe
authenticates nothing (ADR 0022), so the owner's door is an elevated PowerShell. Reads and `stop`
(`status`, `stop`, `setup --check`, `audit`, every read command) run in it. Writes to the root
(`setup`, `setup <step>`) run as `broodforge`: from an elevated PowerShell `setup` finds its caller
is not `broodforge` and relaunches itself as `broodforge` by `setup host`'s route (rotate the
password, re-register both tasks, `Start-Process -Credential -LoadUserProfile -Wait` in a console
on the desktop), the flags passed through and the prompts answered in that console; stdin is not
carried, so the stdin forms serve tests, scratch roots and Linux. From the unelevated account the
first denied open is `needs-elevation`. A development root under the owner's own account (the
`same-account` backend) needs none of this.

## `setup host [--root <path>] [--instance <name>] [--engine-version <v>] [--from <export>] [--print]`

Elevated or root, idempotent with the bot stopped (a bot answering is `running`, lifted by
`broodforge stop` in the same shell; a rerun then changes nothing but the rotated password and the
re-registered tasks and prints only `ok` and `skip`), ends by running `setup` as the bot's account
with `--root`, `--instance` and `--from` passed through and starting the service; `--from`'s file
is copied to `<root>/data/config-export.json` first, since the owner's profile is unreadable to the
bot's account. `--print` writes the commands and files it would run and change, nothing else (the
debug interface for the part no test runs without elevation); it opens with the line `# password:
generated at run time, withheld here` and writes `<password>` in its place.

Linux: the operator files as code: group and users, root and modes, the sudoers file from the
generated `env_keep` list through a temp file and `visudo -cf`, packages, the AppArmor profile on
Ubuntu 24.04, the engine ELF pinned at `/opt/broodforge/engine/claude`, the binary beside it,
`sudo -u broodforge broodforge setup` on the same terminal, then the unit.

Windows, from an elevated PowerShell (one UAC click), through the built-in cmdlets run as a child
with the script and the password on stdin, never an argument (no new dependency): the account
`broodforge` (`New-LocalUser`, `Add-LocalGroupMember Users`, never expires) with a 32-character
password `setup host` generates; the directory `%ProgramData%\Broodforge\` with inheritance off and
inheritable entries for SYSTEM and Administrators (full control) and `broodforge` (modify), nothing
else, so the instance root `%ProgramData%\Broodforge\<instance>` and the key the inner `setup`'s
`credential init` creates beside it at `%ProgramData%\Broodforge\<instance>.key` inherit them
(`setup host` creates no key file; `credential init`'s `key-exists` rule stands); `bin\` and
`engine\` beneath it with inheritance off again and `broodforge` read and execute only, so no
session replaces the binary or the engine; the binary copied to `bin\`; the engine copied to
`engine\` (W4) or, until W4 holds, the native installer run as `broodforge`; the task `Broodforge`
registered at startup with the generated password and the operator file's three settings, the XML
read back and `PT0S` required (`fix` otherwise); the inner `setup` started as `broodforge` with
`Start-Process -Credential -LoadUserProfile -Wait` in a console on the owner's desktop, which
creates the profile (W3a); then `Start-ScheduledTask`. The password is stored nowhere by
Broodforge: Task Scheduler holds it in its own credential store, readable by SYSTEM and
Administrators only and never by anything running as `broodforge`; the owner never sees it; a rerun
rotates it (`Set-LocalUser`, the tasks re-registered). A DPAPI copy in the owner's profile was
rejected: a second secret file to save one UAC click.

## `setup login`, `setup service`, `config export`, `setup --from`

`broodforge setup login <credential id> [--print]` (Windows elevated, Linux root): the owner's
Claude login under the bot's account until the relay exists. Windows: rotate the password and
re-register the tasks, then run `CLAUDE_CONFIG_DIR=<root>\harness\<id> <engine> auth login` as
`broodforge` with `-LoadUserProfile -Wait` in a console on the desktop; Linux: the same through
`sudo -u broodforge-engine`. On exit 0 it records `credential login <id> --done` over the socket (the
bot must be running); another exit is `login-failed`; a running bot is unaffected by the rotation.

`broodforge setup service` (file): the unit with `KillMode=mixed` and the backup timer, `enable
--now`; or the two tasks alone, rotating the password; `ok  service  <task or unit>`. For a rerun
after the service was removed by hand; `setup host` calls it.

`broodforge config export [--out <file>]` (file) writes `config/` as one JSON document with the
host-bound keys emptied (`engine.path`, `engine.gitBashPath`, `credentials.keyFile`,
`paths.install`, `repo.canonical`, `botCredentialId`, `gitCredentialId`; `engine.fakeFixtures` and
`audit.fileSink`, host paths, written `null`); the credential ids are
emptied, not carried (the token and git credential are asked for again); no secret, never the
store; default `--out` `<root>/data/config-export.json`, outside any repository, the docs'
personal-data rule applying to the file. `setup --from <file>` (`config import <file>` (file) alone,
`config-exists` on a configured root) writes the files and asks only the host-bound and secret
questions; on a configured root it adds absent keys and reruns clean (`setup host` passes it on every
rerun); the export's `instanceName` is the shown default for `--instance`; the file is copied to
`<root>/data/config-export.json`; a bad export under `config import` is `bad-config`; on a second PC with the same application and guild `channels` and `role` find
everything by the imported ids or by name, and the first instance must be retired first (one
application per instance). State and secrets stay 0021's backup and restore.

## Amendments

- 2026-10-26: aligned to P21, P22 and P24 (the code wins): the departures the three passes accepted, the invite integer and the new refusal codes.
- 2026-10-26: this companion now diverges from ADR 0027's own text, and the companion governs. Decision 6 says
  two owner texts await re-approval; now four (see The owner's texts). Decision 4's refusal-code list
  lacks `socket-path`, `live-instance`, `bad-config`, `mismatch-config`, `no-config`, `key-unreadable`
  and `bad-args`. Decision 3's "`fix` otherwise" is now a fix line, then `mismatch-role`, exit 1.
- 2026-10-30: `--discord live|dry-run` and The dry-run form, for the PC development instance (the owner's
  decision; ADR 0027 Amendments; P27), with their entries in the flags, rerun rule, refusals and texts.
