# 0027. Setup in few steps: `broodforge setup`, bot-made channels, the PC account created by `setup host`

Date: 2026-10-25 (revised 2026-10-25 after `0027-critic-review.md`; the owner's answers 2026-10-26)
Status: promoted to ADR 0027

## Question

How a new owner gets from "has the binary" to a bot online in their server, in as few hand actions
as the Discord Developer Portal, GitHub and a password manager allow, on the PC first and the
droplet second. The owner set the bot up once and must redo it on their main PC; the complaint is
"dozens of steps", none hard: counted uniformly (Step tables), 49 on the PC and 53 on the droplet.
The prediction this proposal is judged by: with `setup` in the binary, no later pass adds an
owner-only step a command could have done, and a rerun on a configured host reports and changes
nothing. Already ruled out: a separate configuration app (a second codebase tracking the config
format; it removes no owner-only step); simple mode as the PC default (the owner, 2026-10-25).

## Constraints that matter here

- Both hosts (`../06-platforms.md`); no new dependency; `setup` must run without a terminal to
  type into. ADR 0021: configuration is owner-edited files with immutable project ids; the bot
  never writes them. ADR 0022: the store is written by the bot only; the key is printed once; the
  owner's `~/.claude` is never touched; the installed PC instance runs as the local standard
  account `broodforge`. `../05-cli-conventions.md`: existing command names and ADR 0020's wire
  format do not change; new subcommands appear under `setup`, `config export`, `config import`.
- A step is one hand action: a page of portal clicks, one command (its UAC or `sudo` prompt
  included), one paste or typed answer, one copy into the password manager, one link opened, one
  browser sign-in. Accepting a shown default is not a step; a "check it took" block is not one
  unless the docs make it the gate before the next (`config check`, `discord check` are).
- Owner choices follow `../../findings/2026-10-25-prior-art-setup-wizards-and-owner-choices.md`:
  a prompt with a shown default for a consequential choice, a flag for every prompt,
  non-interactive refuses an irreversible choice and names the flag, reversible values default in
  config; bot-made channels come from an owner-only, rerunnable step that writes the ids.
- The PC keeps the service account; `same-account` stays the development and scratch backend
  (`../../findings/2026-10-25-prior-art-isolating-agent-sessions-from-the-owner.md`: no runner or
  agent sandbox runs jobs as the Windows owner; the field creates the account itself behind one
  UAC prompt with a password the tool generates and keeps). Out of scope: a session reading its own
  `harness/<id>/.credentials.json` through Bash on the PC (ADR 0022, Windows; closes when the
  engine's sandbox or MXC wraps the engine on Windows, that ADR's trigger); nothing here changes it.
- What the owner types and reads is the owner's decision; this proposal recommends and waits.

## Options

### A. Status quo plus docs

The eight operator files become one ordered checklist per host in `docs/reference/`, the stale
lines fixed (the `spawn preflight` sentence; the systemd unit; the timeouts, a root outside Temp
and the GitHub token as steps). No code. Cost: a tester pass. Risk: the count does not move.
Counts: PC 49, droplet 53. Exit cost: none; the files are deleted. Moving parts: none.

### B. `setup` in the binary, channels still by hand

`broodforge setup` does every scriptable step (the application id, Message Content, the guild,
the invite link, the root, the timeouts, the `env_keep` list, `privacy.allow`) and asks for the
rest; `setup host` does the host part, the PC account included; `config export` and `import`.
Channels, category, role and notice channel are made in the client, Developer Mode on, four ids
copied and pasted in. Cost: one coder pass host-neutral, the Linux host part under P14's hold, the
Windows host part in P13. Risk: eight hand actions survive only to move four ids. Counts: PC 31,
droplet 32. Exit cost: delete the `setup` package; A's files come back. Moving parts: the `setup`
group; the store opened by a command while no bot runs; config written once by a command.

### C. B plus bot-made channels (recommended)

As B, and `setup` creates what the bot needs in the guild given Manage Channels (already in the
invite's set): one category, one lobby per project, one notice channel; it finds an existing one
by name first, writes the ids, and never deletes or renames a channel it finds (the venue deletion
of `../05-debug-interface-discord-live.md` is unchanged). The `Trusted` role is found by name;
absent, a prompt (default yes; `--create-role yes|no`) adds Manage Roles to the invite, creates
it, gives it to the guild's `owner_id`, then waits until the owner has removed Manage Roles again.
Cost: C's REST calls against the `restAPI` fake. Risk: a bot creating channels in someone else's
server (the droplet serves Utopia's) must be told where: setup asks when the guild has channels
and creates nothing without a yes. Counts: PC 18, droplet 20 (the role made by hand, 19 and 21;
the git credential deferred, 15 and 17). Exit cost: delete the `channels` and `role` steps; B's
four prompts return. Moving parts: channel and role creation at setup time only; Manage Roles for
the minutes between the invite and the removal.

### D. C plus a simple mode as the PC default (rejected by the owner, 2026-10-25)

As C, and `setup` on Windows defaults to `spawn.backend: same-account` under the owner's own
account, the engine the owner already has, the bot in a logon task of the owner's (W1). Kept as
the record of why not. Its premise, "a second coder session gains nothing from the service
account", was true of session-to-session isolation and false of what the account is for: in D a
Discord session reads and persists into everything the owner's account can, unattended, under
policy mode, the owner's `~/.claude/.credentials.json` and `settings.json` included (W5), and the
account's value rises with the coder count. It would supersede lines of ADR 0022 (Context,
Decision), `06-platforms.md` (Already constrained) and this proposal's constraints. Its count, once
account creation is code, is 18 on the PC: equal to C. Honest exit cost: `config export`, a fresh
`setup host` under the account, 0021 restore for the store, a new login per credential. Moving
parts: the owner's interactive session as the bot's home; a bot down from a reboot until the owner
signs in; a console window at every logon unless hidden.

## The `setup` surface (B and C)

`broodforge setup [<step>] [--root <path>] [--instance <name>] [--repo <url>] [--project <id>]
[--guild <id>] [--channels yes|no] [--create-role yes|no] [--role <name>] [--from <export>] [--check]
[--yes] [--wait <seconds>] [--json]` (a listener while it runs, `(file)` under `--check`; Review
fixes). One line per step, `ok  <step>  <detail>`, `skip  <step>  <already>` or `fix
<step>  <what the owner does>` (`--json`: one object each), pausing at a `fix` that needs a hand and
polling, bounded by `--wait` (default 900 s), until the step passes. `--check` reports and changes
nothing. `--yes` accepts every shown default and refuses only where there is no default. With a
`<step>` it runs that step alone and may replace that step's keys after printing `old  <value>`,
`new  <value>` (a secret as its id or path) and asking (`--yes` accepts): the "change one setting"
verb of the finding, and the only way `setup` overwrites a present key. Exit 0 all ok, 1 a step
failed (its `fix` line on stdout, one stderr `error: <code>: ...` line), 2 usage. The companion
ADR file `../decisions/0027-setup-commands.md` is authoritative for the flags, the per-step `--json`
fields, the rerun rule per key and the refusal table.

Steps, in order. `root`: the directories. `config`: the repository URL asked here (`--repo`), the
project id from the repository name lowered, runs outside `[a-z0-9-]` made `-`, collapsed, cut to
32, asked for (`--project`) when it does not start with a letter (03's pattern), because the id
names the project file, the channels and the clone; `instance.json` and `projects/<id>.json` with
every key commented, only when absent. `key`: `credential init`, `ok  key  <hex>  <path>` once, a
pause until Enter (`--json`: the one print is this step's object, never repeated). `engine`: path
and version. `token`: read hidden, or from stdin, stored as the instance's `bot-token`; `skip` when
the stored one answers C1; refused by Discord, a `fix` naming the reason and the prompt again, the
new token a new id and the old one `removed`. `application`: the id from the token's application;
the flags ORed with `GATEWAY_MESSAGE_CONTENT_LIMITED` (1<<19) and written back, `skip` when 1<<18
or 1<<19 is set, `fix` naming the portal switch when the PATCH is refused. `guild`: one membership
taken; none, the invite link printed and a poll until the bot joins; several, a list (disgo
`OAuth2.GetCurrentUserGuilds("")`, the empty token meaning the bot's). The role question is asked
here, before the link is printed, so the link carries Manage Roles only on a yes. `channels`: the
configured ids first, then by name, ask when the guild has channels, create what is missing, write
the ids. `role`: the configured id first, then by name; absent and the answer no (or the bot
already in the guild without Manage Roles), a `fix` that waits for the owner to make it; yes, the
default, create, check it sits below the bot's highest role (C6), assign; then, on every run, a
`fix` that waits while the bot still holds Manage Roles. `git-credential`: the token read hidden,
or Enter to defer (`handoff: "none"`), before the clone so a private canonical clones. `project`:
the canonical cloned from the repository into `projects/<id>/` with the stored credential through
`GIT_ASKPASS`. `defaults`: timeouts 0, mode `prompt`. `check`: `config check`, `spawn preflight`,
`discord check`, each line echoed. `next`: the one line that follows. Non-interactive: the bot
token on `setup`'s stdin; `setup git-credential` reads its token on stdin alone.

| Choice | Kind | Flag | Default shown | With `--yes` and no flag |
|---|---|---|---|---|
| Instance root and name | prompt | `--root`, `--instance` | `%ProgramData%\Broodforge\<instance>` or `/srv/broodforge/<instance>`; `main` | taken |
| Which guild (several); create channels in a guild that has some | prompt | `--guild <id>`; `--channels yes\|no` | none; no | `needs-input-guild`; no taken, `channels` waits by name |
| Repository URL; project id when the derived one is invalid | prompt | `--repo`, `--project` | none; derived | `needs-input-repo`, `-project` |
| Create the role (Manage Roles on the invite until removed) | prompt | `--create-role yes\|no` | yes | yes |
| Role name; timeouts, mode, channel names | config | `--role`; none | `Trusted`; 0, `prompt`, `Sessions`, `#<id>`, `#<id>-notices` | taken |

Single writer. At start `setup` (not `--check`) binds the control address (`data/control.sock`;
the pipe `\\.\pipe\broodforge-<instanceName>`) and holds it until exit, a listener beside `serve`,
so `serve` fails `start-control` as for any listener and the vault's single-writer invariant (S1)
holds with one writer at a time; its `status` reply is `config: setup`. A rerun never overwrites a
present config key (a disagreement is `mismatch-<step>`), a valid key file (`skip  key  exists`), a
stored credential (a new token is a new id, the old one `removed`), or a channel or role it found;
the rule per key (a refused stored token replaced in the run; `setup key` only while the store is
empty; configured channel and role ids checked before names; Manage Roles re-checked on every run)
is the companion's. Refusals, exit 1, each one stderr `error: <code>: <message>` line (05) with the
step's `fix` line on stdout where one applies; the companion's table carries the full list
(`key-in-use`, `check-failed`, `login-failed`, `config-exists` added by the review fixes):

| Code | When | Message | Lifted by, from which shell |
|---|---|---|---|
| `running` | a bot answers at the address (never for `--check`) | `error: running: a bot answers at <address>; stop it with broodforge stop in this shell (every open session is interrupted) or use setup --check` | the owner, `broodforge stop` from the same shell (the PC's elevated PowerShell; a shell as `broodforge` on Linux); nothing else while the bot runs |
| `setup-running` | another `setup` holds the address | `error: setup-running: another setup holds <address>; finish it or press Ctrl-C there` | the owner, in that terminal |
| `needs-input-<step>` | `--yes` and no default | `error: needs-input-<step>: pass --<flag>` | a flag, or an interactive run |
| `mismatch-<step>` | config has a value and the host or guild disagrees | `error: mismatch-<step>: config has <X>, found <Y>; run setup <step> to replace it` | `setup <step>`, bot stopped: the elevated PowerShell on the PC (relaunched as `broodforge`), `sudo -u broodforge` on Linux |
| `refused-<step>` | Discord refuses with nothing a poll can lift | `error: refused-<step>: <Discord's reason>; <the portal or server setting>` | the owner in the portal or server, then a rerun |
| `timeout-<step>` | a poll ran past `--wait` | `error: timeout-<step>: still waiting after <n>s: <what to do>` | a rerun, or `--wait 0` |
| `needs-elevation` | `setup host` or `setup login` not elevated or root; on the PC a denied root or pipe | `error: needs-elevation: run setup <host\|login> from an elevated PowerShell` / `... with sudo` | the owner, from that shell |

## `setup host`, `setup login`, `setup service`, `config export`

`broodforge setup host [--root <path>] [--instance <name>] [--engine-version <v>] [--from <export>]
[--print]`, elevated or root, idempotent with the bot stopped (`running` otherwise), ends by
running `setup` as the bot's account with the flags passed through (`--from`'s file copied into
the root first) and starting the service; `--print` writes the commands and files it would run
and change, the password withheld and said so, nothing else (the debug interface for a part no
test runs without elevation). Linux: the operator files as code: group and users, root and modes, the sudoers file
from the generated `env_keep` list through a temp file and `visudo -cf` (L1), packages, the
AppArmor profile on Ubuntu 24.04, the engine ELF pinned at `/opt/broodforge/engine/claude`, the
binary beside it, `sudo -u broodforge broodforge setup` on the same terminal, then the unit.
Windows, from an elevated PowerShell (one UAC click), through the built-in cmdlets run as a child
with the script and the password on stdin, never an argument (no new dependency): the account
`broodforge` (`New-LocalUser`, `Add-LocalGroupMember Users`, never expires) with a 32-character
password `setup host` generates; the directory `%ProgramData%\Broodforge\`, inheritance off,
inheritable entries for SYSTEM, Administrators and `broodforge` only, which the root
`%ProgramData%\Broodforge\<instance>` and the key `%ProgramData%\Broodforge\<instance>.key` inherit
(the key is created by the inner `setup`'s `credential init`, never by `setup host`, so its
`key-exists` rule stands; no session path depends on a profile or carries a user segment; ADR
0022's key default amended by the ADR for 0027); `bin\` and `engine\` beneath it read-only to
`broodforge`; the binary copied to `%ProgramData%\Broodforge\bin\`; the
engine copied to `%ProgramData%\Broodforge\engine\` (W4) or, until W4 holds, the native installer
run as `broodforge`; the task `Broodforge` registered at startup with the generated password and
the operator file's three settings, the XML read back and `PT0S` required (`fix` otherwise); the
inner `setup` started as `broodforge` with `Start-Process -Credential -LoadUserProfile -Wait` in a
console on the owner's desktop, which creates the profile (W3a); then `Start-ScheduledTask`. The
password is stored nowhere by Broodforge: Task Scheduler holds it in its own credential store,
readable by SYSTEM and Administrators only and never by anything running as `broodforge`; the owner
never sees it; a rerun rotates it (`Set-LocalUser`, the task re-registered), as srt does. A DPAPI
copy in the owner's profile was rejected: a second secret file to save one UAC click.

`broodforge setup login <credential id>` (Windows elevated, Linux root): the owner's Claude login
under the bot's account until the relay exists. Windows: rotate the password and re-register the
task, then run `CLAUDE_CONFIG_DIR=<root>\harness\<id> <engine> auth login` as `broodforge` with
`-LoadUserProfile -Wait` in a console on the desktop; Linux: the same through `sudo -u
broodforge-engine`. On exit 0 it records `credential login <id> --done` over the socket; a running
bot is unaffected by the rotation. `broodforge setup service` (the unit with `KillMode=mixed` and
the backup timer, `enable --now`; or the task alone; `ok  service  <task or unit>`) is for a rerun
after the service was removed by hand; `setup host` calls it. `broodforge config export [--out
<file>]` writes `config/` as one JSON document with the host-bound keys emptied (`engine.path`,
`engine.gitBashPath`, `credentials.keyFile`, `paths.install`, `repo.canonical`, `botCredentialId`,
`gitCredentialId`); no secret, never the store; default `--out` `<root>/data/config-export.json`,
outside any repository, the docs' personal-data rule applying to the file (`privacy.identities`,
the Discord ids). `setup --from <file>` (`config import <file>` alone) writes the files and asks
only the host-bound and secret questions; on a second PC with the same application and guild
`channels` finds everything, and the first instance must be retired first (06: one application
per instance). State and secrets stay 0021's backup and restore.

## Step tables

The table of one row per hand action is the Appendix (after the responses, so the body stays
short); `SQ` is the status quo from the operator files, `C` as above with its defaults (the role
created by setup, the git credential given during setup). Its sums: status quo PC 49, droplet 53;
C PC 18, droplet 20; the role made by hand (`--create-role no`), 19 and 21; the git credential
deferred, 15 and 17. Second PC, same application and guild, from an export, the first instance
retired: build, `setup host --from`, token, key, GitHub token paste, `/register`, `/credential
add`, `setup login`, the browser: 9. B is C with the role by hand plus three channel creates,
Developer Mode, four id copies and four pastes: PC 31, droplet 32. D is C with `setup` in place of
`setup host`: 18.

## Recommendation

C. First implementation step: one pass after P12d in `../08-implementation-plan.md`, host-neutral:
`setup` through `check` and `next` against a scratch-shaped root and the `restAPI` fake, `setup
<step>`, `config export` and `--from`; P13 becomes the Windows `setup host` and `setup login` with
the W3a and W4 spikes on a throwaway machine; the Linux `setup host` joins P14. The ADR for 0027
amends ADR 0021 ("owner-edited" to "owner-edited, created once by `setup`"), ADR 0022 ("written by
the bot only" to "by the bot, or by `setup` while it holds the control address"; the Windows key
default; the account created by `setup host`) and retires the PC account operator file.
The owner's calls, answered 2026-10-26 (Owner's answers, below); the texts as decided:

1. Settled 2026-10-25: C, the account kept, simple mode the development and scratch backend.
2. Manage Roles: default on (the owner, 2026-10-26; the proposal had off, as the critic asked):
   setup creates the role and the owner removes Manage Roles at the end, one step; answering no
   costs two instead (the owner makes the role and assigns it). Prompt: "Let setup create the role
   "Trusted"? This adds Manage Roles to the invite link; setup asks you to remove it again at the
   end. [Y/n]". Waiting line on no: "fix  role  create a role named Trusted in Server Settings,
   Roles, and give it to yourself; setup is waiting". On yes: "fix  role  remove Manage Roles from
   the bot's role in Server Settings, Roles; setup is waiting".
3. The names the bot creates: category "Sessions", lobby "#<project id>", notices
   "#<project id>-notices", role "Trusted".
4. The prompts: token "Bot token (Developer Portal, Bot, Reset Token; not shown as you type):";
   key "ok  key  <hex>  <path>" (Review fixes: a step line; the owner's "key  <hex>  <path>") then
   "Copy the key into your password manager, then press Enter.";
   invite "Open this link, pick the server, Authorize: <url>" then "waiting for the bot to join";
   guild with channels "This server already has channels. Create Sessions, #<project id> and
   #<project id>-notices in it? [y/N]"; repository "Repository URL of the project (its canonical):";
   git credential "Token for pushes to it (not shown; Enter to add later with setup
   git-credential):"; the closing line "next  the bot is running. In Discord: /register, then
   /credential add kind:subscription; then here: broodforge setup login <credential id>".
5. The refusal texts of the table above, in particular `running`: "a bot answers at <address>;
   stop it with broodforge stop in this shell (every open session is interrupted) or use setup
   --check" (Review fixes: "in this shell" added; the owner's text had no shell).

## Owner's answers, 2026-10-26

1. C is the decision; the PC keeps the service account; simple mode (`same-account`) stays the
   development and scratch backend. ADR 0022 is amended, not superseded, where this proposal names
   the Windows key default and the account created by `setup host`.
2. Manage Roles: default on. The prompt "Let setup create the role "Trusted"? ..." defaults to
   yes, bracket `[Y/n]`; the invite carries Manage Roles; setup still asks the owner to remove it
   at the end (the "remove Manage Roles" waiting line). The counts above are the default-on ones.
3. and 4. The names and the prompts were not questions to the owner: accepted exactly as quoted.
5. The refusal texts accepted as quoted, the `running` text included.

Recorded as `../decisions/0027-setup-in-few-steps.md`.

## What would change this recommendation

W3a fails: the PC gains one step (sign out and in once as `broodforge`); C stays. W4 fails: `setup
host` runs the native installer as `broodforge`; no step changes. C5 or C6 fails: channels or the
role go back to the client and setup prompts for ids (B there). A release pipeline: "build" and
"copy the binary" leave every count. The login relay: `setup login` and the browser step leave the
first session. Claude Code ships MXC support or srt leaves alpha on Windows: ADR 0022's trigger.

## Claims to verify

Verdicts in brackets are the critic's (`0027-critic-review.md`, Verification, authoritative).
C1 [holds], C2 [not checked, harmless], C5 [holds], C7 [holds], C8 [holds], W2 [holds], S1 [holds;
objection 9 closes the race], L1 [not checked], W1 and W5 [hold; D's record only] stand as the
review states them. The rest:

- C3 [holds]. `GET /users/@me/guilds` through disgo's OAuth2 route with an empty bearer token.
- C4 [holds]. `PATCH /applications/@me` sets 1<<19; OR with the existing flags, skip on 1<<18.
- C6 [holds except the position]. Where a new role lands is undocumented; `--create-role` checks
  the created role sits below the bot's highest before assigning and `fix`es otherwise.
- W3 [refuted]. Replaced by W3a: `Start-Process -Credential <broodforge> -LoadUserProfile` from an
  elevated interactive PowerShell creates the account's profile on first use and runs the child in
  a console on the owner's desktop; ADR 0022 saw `CreateProcessWithLogonW` fail only from a
  non-interactive service. Spike in P13 on a throwaway machine, not the owner's PC.
- W4 [unverified; the earlier citation was wrong]. The native Windows `claude.exe` (Claude Code
  setup docs: a signed executable; auto-update off under `DISABLE_AUTOUPDATER`) runs from
  `%ProgramData%\Broodforge\engine\`. Spike in P13; the fallback is scripted.
- W6. `New-LocalUser`, `Set-LocalUser -Password`, `Register-ScheduledTask -User -Password -Force`
  and `Start-ScheduledTask` run from an elevated PowerShell with the script and the password on
  stdin and no interactive input; a password change does not end the account's running processes.

## Critic's objections and responses

Resolutions: 11 accepted, 8 accepted with change, 0 rejected.

1. Blocking, D breaks the `~/.claude` constraint. Accepted. D is no longer recommended; it stays
   as a rejected option with the lines it would have superseded named (Options, D). The owner's
   reasoning, recorded here so it is not reopened: "we don't want to have claude credentials in
   the session history", and the second prior-art finding shows no runner or agent sandbox
   running jobs as the Windows owner. Body change: Question, Constraints, D, Recommendation.
2. Blocking, the load-bearing claim half right. Accepted. "Gains nothing" struck; D now states the
   trade as the critic put it (a Discord session reads and persists into everything the owner's
   account can, unattended) and that the account's value rises with the coder count.
3. Serious, D buys three or four steps once. Accepted. C recommended; `same-account` stays the
   development and scratch backend. The recount makes it moot: D is 18, equal to C.
4. Serious, the owner's account name in every path. Accepted with change. D is gone, but the point
   stands for C too: the root and key defaults move to `%ProgramData%\Broodforge\`, created by the
   elevated `setup host` with entries for `broodforge` and Administrators only, so no session path
   depends on a profile and none carries a user segment (`broodforge` is in `privacy.allow`, so this
   is for robustness as much as privacy). Amends ADR 0022's Windows key default, recorded in the
   ADR for 0027.
5. Serious, D's exit cost is boilerplate. Accepted. D carries the honest exit; "setup host does
   the move" dropped; every option's exit cost and moving parts made specific.
6. Serious, W3 refuted. Accepted with change. W3a replaces it, and the change goes further than
   the critic's: `setup host` generates the password too, so the password prompt, the password
   manager entry, the sign-out and the engine install all leave the count (seven PC rows to one).
7. Serious, Manage Roles held forever. Accepted. Default off; `--create-role` opt-in with the
   prompt, the removal `fix` that waits until the bot's permissions no longer carry the bit, and a
   position check before assigning (C6). The owner's call 2 states the one-step difference. (The
   owner set the default to on, 2026-10-26: the prompt, the removal `fix` and the position check
   stand; "held forever" stays answered by the `fix` that waits for the removal.)
8. Serious, rerun and the refusal with no lift. Accepted with change. `--check` never refuses; the
   `running` message names `stop` and its effect; a disagreement is `mismatch-<step>`, changes
   nothing, exits 1; the change verb is `setup <step>` (one step alone, old and new printed, a
   yes) rather than flags on setup; `token` re-reads through `setup token` or when the stored
   value fails C1. The refusal table names who lifts each, from where, and while the bot runs.
9. Serious, the single-writer race. Accepted. `setup` binds the control address for its lifetime;
   `serve` fails `start-control` as for any listener; a second `setup` fails `setup-running`;
   `status` answers `setup`. `--check` binds nothing.
10. Serious, bundled counts. Accepted. The Step tables apply one rule to every option and host,
    with the build, the GitHub token (three rows), the login round trip, the code paste and
    `--done`. The quoted numbers moved from 40/41 and 18/13 to 49/53 and 19/21 (18/20 once the
    owner set the role default to yes); the ranking did not.
11. Minor, C3's disgo route. Accepted. Named in the `guild` step.
12. Minor, C4's flag write. Accepted. OR, skip on 1<<18, `fix` when refused; in `application`.
13. Minor, W1's side effects. Accepted with change. D is rejected, so the logon task, the hidden
    console and the sign-in gap are its record only; the `PT0S` read-back applies to the password
    task as well and `setup host` does it.
14. Minor, W4 miscited. Accepted. Recited to the engine docs; spike in P13; the fallback (the
    installer run as `broodforge`) is scripted either way.
15. Minor, `config export` carries personal data; two instances on one application. Accepted.
    Default `--out` under the root's `data/`, the docs rule named; the retire-first sentence added.
16. Minor, drivability gaps. Accepted with change. `--wait <seconds>` bounds the `guild` and
    `role` polls (first "`--yes` without it refuses `needs-input-guild`"; the review fixes gave
    `--wait` a default instead, so `needs-input-guild` is only several guilds and no `--guild`); one secret per command: the bot
    token on `setup`'s stdin, the git credential hidden in the interactive run or on `setup
    git-credential`'s stdin alone; the one-time key print is the `key` step's one object in
    `--json`.
17. Minor, the project id pattern. Accepted. The normalisation is stated in `project`; a derived
    id that still fails is asked for, `--project` is the flag, `needs-input-project` the refusal.
18. Minor, channel creation as its own owner-only verb. Accepted. `setup channels` is the step
    run alone, rerun-safe, and it passes no token step.
19. Minor, the ten-year wish and D's moving part. Accepted with change. The Question now carries
    a prediction that can fail (no later pass adds a scriptable owner step; a rerun changes
    nothing); D's moving part is the owner's interactive session as the bot's home.

## Review fixes, 2026-10-26

An Opus reviewer read the ADR, its companion and this proposal after promotion (findings B1,
S1 to S9, M1 to M4, in the coordinator's brief). What moved in the body above and in the ADR:

- B1: `setup host` creates the `%ProgramData%\Broodforge\` directory and its ACL (SYSTEM,
  Administrators, `broodforge`; `bin\` and `engine\` read-only to the account), never the key file;
  `credential init` creates the key inside it and its `key-exists` rule stands.
- S1: every lift names its shell: the PC's elevated PowerShell (the pipe's default descriptor
  admits Administrators; no DACL change) or a shell as `broodforge` on Linux; on the PC `setup` and
  `setup <step>` relaunch themselves as `broodforge` by `setup host`'s route. The `running` text
  gains "in this shell".
- S2: the repository is asked in `config`; `git-credential` precedes `project`.
- S3: the rerun rule per key (the companion): a refused stored token replaced in the run; a key
  file that is not a key is `mismatch-key` and `setup key` works only on an empty store
  (`key-in-use`); configured channel and role ids checked before names; Manage Roles re-checked on
  every run.
- S4: every refusal is one stderr `error:` line with the `fix` line on stdout; the `--json` fields
  are stated; `setup` and `setup host` take the default root (05's one exception); `(file)` marks
  on `config export|import`, `setup service` and `setup --check`; `setup` is a listener; `status`
  answers `config: setup`; `needs-elevation` names the subcommand.
- S5: `--guild` and `--channels` in the command line; `--yes` takes the `channels` default;
  `--wait` defaults to 900 s, so `needs-input-guild` is only several guilds and no `--guild`.
- S6: `setup host` takes `--root`, `--instance`, `--from` (the file copied into the root); a bot
  running is `running`, idempotence is with the bot stopped.
- S7: `old` and `new` print a secret as its id or path; `setup key` cannot orphan the store.
- S8: the unverified claims and the pass that settles each are in the ADR's Context.
- S9: ADR 0021 carries an amendment entry; `03-architecture.md`'s example paths and Debug CLI row
  follow; the role and channel names are lookup names, not config values.
- M1 to M4: the key is a sibling of the root; the counts above read 18/20; the two prior-art
  findings are resolved to the ADR; the key line is the step's `ok` line; `--print` says the
  password is withheld; the refusal table lives in the companion with its lifters and shells.

## Appendix: the step tables

One row per hand action; `SQ` the status quo from the operator files, `C` as decided with its
defaults (the role created by setup, the git credential given during setup).

| Hand action | SQ PC | SQ droplet | C PC | C droplet |
|---|---|---|---|---|
| Portal: New Application, named | 1 | 1 | 1 | 1 |
| Portal: Bot, Reset Token | 1 | 1 | 1 | 1 |
| Token into the password manager | 1 | 1 | 1 | 1 |
| Portal: Message Content Intent on | 1 | 1 | - | - |
| Portal: copy the Application ID | 1 | 1 | - | - |
| Client: create a server (the droplet's exists) | 1 | - | 1 | - |
| Client: create the sessions category | 1 | 1 | - | - |
| Client: create the lobby channel | 1 | 1 | - | - |
| Client: create the notice channel | 1 | 1 | - | - |
| Client: create the `Trusted` role (C: `--create-role no` only) | 1 | 1 | - | - |
| Client: give the role to yourself (C: `--create-role no` only) | 1 | 1 | - | - |
| Client: remove Manage Roles from the bot's role (setup waits) | - | - | 1 | 1 |
| Paste the application id into the invite URL | 1 | 1 | - | - |
| Open the invite, pick the server, Authorize | 1 | 1 | 1 | 1 |
| Client: Developer Mode on | 1 | 1 | - | - |
| Copy ID: server | 1 | 1 | - | - |
| Copy ID: lobby | 1 | 1 | - | - |
| Copy ID: notice channel | 1 | 1 | - | - |
| Copy ID: category | 1 | 1 | - | - |
| Copy ID: role | 1 | 1 | - | - |
| PC: invent and type the account password | 1 | - | - | - |
| PC: the account password into the password manager | 1 | - | - | - |
| PC: `New-LocalUser` and `Add-LocalGroupMember` (one paste) | 1 | - | - | - |
| PC: sign out, sign in as `broodforge`, sign back | 1 | - | - | - |
| PC: install the engine as `broodforge` | 1 | - | - | - |
| PC: register the task (one paste) | 1 | - | - | - |
| PC: type the password into `Get-Credential` | 1 | - | - | - |
| `setup host` from an elevated shell or `sudo` | - | - | 1 | 1 |
| Droplet: users, root and modes (one paste) | - | 1 | - | - |
| Droplet: assemble the `env_keep` list from the companion | - | 1 | - | - |
| Droplet: `visudo` with the sudoers block | - | 1 | - | - |
| Droplet: packages | - | 1 | - | - |
| Droplet: write the AppArmor profile | - | 1 | - | - |
| Droplet: `apparmor_parser -r` | - | 1 | - | - |
| Droplet: install Go | - | 1 | - | - |
| Droplet: install `gcc` | - | 1 | - | - |
| Droplet: stage and install the engine (one paste) | - | 1 | - | - |
| Droplet: clone the repository | - | 1 | - | - |
| Build the binary (no release exists) | 1 | 1 | 1 | 1 |
| Droplet: install the binary at the shared path | - | 1 | - | - |
| Copy the binary to the droplet | - | - | - | 1 |
| Droplet: write the systemd unit | - | 1 | - | - |
| Droplet: `systemctl enable --now` | - | 1 | - | - |
| Pick and create the root | 1 | - | - | - |
| A shell as `broodforge` | 1 | 1 | - | - |
| `credential init` | 1 | 1 | - | - |
| Key into the password manager | 1 | 1 | 1 | 1 |
| `credential add --instance --kind bot-token` (the command) | 1 | 1 | - | - |
| Paste the token | 1 | 1 | 1 | 1 |
| Edit `instance.json` (ids, credential id, timeouts, `privacy.allow`) | 1 | 1 | - | - |
| Edit `projects/<id>.json` | 1 | 1 | - | - |
| Clone the canonical into `projects/<id>/` | 1 | 1 | - | - |
| Type the repository URL | - | - | 1 | 1 |
| GitHub: mint a token for the canonical | 1 | 1 | 1 | 1 |
| GitHub token into the password manager | 1 | 1 | 1 | 1 |
| `credential add` for it (the command) | 1 | 1 | - | - |
| Paste the GitHub token | 1 | 1 | 1 | 1 |
| `config check` | 1 | 1 | - | - |
| `discord check` | 1 | 1 | - | - |
| Start the bot (`Start-ScheduledTask`; the unit started at enable) | 1 | - | - | - |
| Answer "create channels in this server?" (a server with channels) | - | - | - | 1 |
| First session: `/register` | 1 | 1 | 1 | 1 |
| First session: `/credential add kind:subscription` | 1 | 1 | 1 | 1 |
| A shell as the bot's account for the login | 1 | 1 | - | - |
| Run the login command (`setup login <id>` in C) | 1 | 1 | 1 | 1 |
| Browser sign-in and authorize | 1 | 1 | 1 | 1 |
| Paste the code back (always on the droplet; the PC when the callback fails) | - | 1 | - | 1 |
| `credential login <id> --done` | 1 | 1 | - | - |
| **Sum** | **49** | **53** | **18** | **20** |
