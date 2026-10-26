# Implementation plan: setup in few steps (ADR 0027)

This file continues `08-implementation-plan.md` under the same conventions: the pass format, the size
words, the standing rules and the `[no-check: ...]` rule. It holds the passes that build ADR 0027, P20 to
P26, ordered so the owner's PC has a working instance soonest: the spike (cheap, and it decides what P23
scripts), the host-neutral `setup`, the Discord steps, the Windows host, the export, the login, the droplet.
P13 of 08 is built here as P23 (its reference file, its Proof and C37 move with it); P14 gains P26 as its
host half and P26 shares P14's hold, lifted on the VM below; P15 to P19 are unchanged. Each pass reads
`decisions/0027-setup-in-few-steps.md` (Decision), `decisions/0027-setup-commands.md` (all) and the ADR 0027
amendments of `decisions/0022-session-isolation-and-credential-storage.md` before its Implements line. The
texts are the companion's (The owner's texts; the refusal table), quoted as they stand; the critic's verification
table (`proposals/0027-critic-review.md`) is authoritative over the proposal body, and the companion over both.

Machines (the owner, 2026-10-26). The Windows spike and every Windows owner step run on the owner's own PC:
the spike against a disposable local account it creates and deletes (P20), never the live `broodforge`
account; no cloud machine. The Linux passes verify first on an Ubuntu VirtualBox VM on the PC, a
transportable stand-in for the droplet, then on the droplet. To stand in faithfully the VM carries what the
droplet operator entries create: Ubuntu 24.04, the `broodforge` and `broodforge-engine` users with the
root's owners and modes, bubblewrap and socat with the AppArmor profile, the pinned engine ELF at
`/opt/broodforge/engine/claude`, Go and gcc for `-race`. P26's `setup host` is what makes that from a bare
Ubuntu, so a VM snapshot taken before it is the pass's own test bed: restore, run, compare.

Standing rules for these passes. `internal/setup/` is a new component [no-check: created by P21]: P21 adds
its row to `03-architecture.md`'s Components table (owns the step runner, the prompts and defaults, the
single-writer bind, the host scripts; talks to `config`, `credentials`, `store`, `spawn`, `git`, `discord`,
`control`), and `.claude/rules/go.md`'s host-branching line gains its `host_windows.go` and `host_linux.go`
(the elevated part and hidden terminal input are host code by nature; no other file of the package
branches). `internal/cli/setup.go` registers its commands in `commands` as every command file does:
`setup` one word with the optional positional `<step>`, `setup host|login|service` two words (`lookup` tries
two words first, so `setup token` falls to the one-word command). Hidden input is `x/sys`'s console-mode or
termios call in the host files; no new module. No pass prints a secret: the token, the git credential, the
key past its one print and the generated password never reach stdout, a log, `--print` (which writes
`<password>`) or a golden file. Every Proof opens with `CGO_ENABLED=0 go build ./...`, the check line `go vet
./... && go tool staticcheck ./... && go tool exhaustive ./...` and `CGO_ENABLED=1 go test -race ./...`,
then its own commands. A real Discord call or engine turn runs only in an owner step; tests use the fakes.
An operator entry whose steps a pass lands is deleted in that pass with its README row
(`../operator/README.md`), the durable part moving to the reference file the pass names.

## P20. The Windows host spike: W3a, W4, W6 (small; the owner runs it)

Implements: proposal 0027 (Claims to verify: W3a, W4, W6. C2 needs no spike, C1 is the route the code takes;
C6's position is P22's owner step; L1 is P26's). Files: `spike/setup-host/` [no-check: written by this pass;
`spike/` goes with P14] with `run.ps1`, the script P23 will generate, written by hand for an account named
`bf-spike` (a name no one mistakes for `broodforge`) and a second standard account `bf-spike2` that only
tests the ACL from outside, in the companion's order: `New-LocalUser` with a
generated password read from stdin, `Add-LocalGroupMember Users`, a directory `%ProgramData%\Broodforge-spike\`
with inheritance off and inheritable entries for SYSTEM, Administrators and `bf-spike`, a root and an `engine\`
read-only to `bf-spike` beneath it (`icacls` of each read back), `claude.exe` copied from the owner's own
install into that `engine\` (W4), `Register-ScheduledTask` with `-User -Password` and
the PC operator file's three settings, `Export-ScheduledTask` read back, `Start-Process -Credential
-LoadUserProfile -Wait` running `whoami` then `claude --version` in a new console (W3a), then `Set-LocalUser
-Password` while a `ping -n 61` started as `bf-spike` is running (W6); `cleanup.ps1`:
`Unregister-ScheduledTask`, `Remove-LocalUser` of both accounts, each profile through `Remove-CimInstance` of its
`Win32_UserProfile` (which drops the hive too), the spike directory with its root and engine copy; `probe-unelevated.ps1` (the ACL read-back as an unelevated user); a `README.md`. The
finding `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` [no-check: written by this pass]
(`finding` skill, one row per claim): W3a holds when the profile exists after the child exits and `whoami`
printed `bf-spike` with no sign-out; W4 when `--version` from the `%ProgramData%` copy prints the pinned
version and `claude -p` with a scratch config directory answers one turn (one real turn, the owner's); W6
when every cmdlet ran from `powershell -NonInteractive -Command -` with the script on stdin and the `ping`
outlived the password change; plus the task XML's `ExecutionTimeLimit` as registered (P23's `fix` depends on
it). Left behind on failure: the account, its profile, the task, the root and the engine copy, every one
named `bf-spike` or under `Broodforge-spike\`, removed by running `cleanup.ps1` again; nothing touches the
owner's profile or `~/.claude`. Proof: `run.ps1` then `cleanup.ps1` from one elevated PowerShell, `probe-unelevated.ps1` from an unelevated one; after cleanup
`Get-LocalUser bf-spike` is an error and the root is gone; the finding's rows filled, each with its
evidence. Depends on: nothing; done before P23's brief is written.

## P21. `setup` host-neutral: the runner, the steps without Discord, the single writer (large; owner sees the prompts)

Implements: ADR 0027 (Decision 1, 2, 4; 6's key, repository, git credential and closing texts; 7's config
rule; Consequences, prediction 1); the companion (`broodforge setup`: the flags, output, exit codes, `--check`,
`--yes`, `setup <step>`, the rerun rule per key, the refusal table, the steps `root`, `config`, `key`, `engine`,
`git-credential`, `project`, `defaults`, `check` without its `discord check` line, `next`); `05-cli-conventions.md`
(Surface, whose listener list gains `setup`; Output; Exit codes); ADR 0022 amendments (Store). Files [no-check:
every file of this pass is new or gains code]:
`internal/setup/setup.go` (`Runner` over the ordered step list, each `Step{Name string; Run(ctx, *State)
Line}`; `Line{Result, Step, Detail}` with `Result` `ok`, `skip` or `fix`, printed as 05's record, `--json`
one object each with the companion's fields; a `fix` that waits polls on the injectable clock until the step
passes or `--wait` (default 900) runs out; a failed step ends the run with one stderr `error: <code>: ...` line,
`--check` with `check-failed`; the Discord steps print `skip  <step>  not built` until P22 registers them); `internal/setup/prompt.go`
(`Prompt(text, default)` and `Hidden(text)` over an injectable terminal; `--yes` takes the default or refuses
`needs-input-<step>` naming the flag; non-interactive stdin: the bot token read from it by P22, nothing else);
`internal/setup/state.go` (the flags, the root default `%ProgramData%\Broodforge\<instance>` when the
`ProgramData` variable is set else `/srv/broodforge/<instance>`, instance `main`; the exception in
`internal/cli/cli.go` that lets `setup` and `setup host` run with no `--root`); `internal/setup/writer.go` (the
bind: before any write, `probeControl`'s check, then `control.Listen` on the address computed from `--instance`
with a registry serving `control echo` and `status` answering `config: setup` and `-` for the rest; a bot answering is `running`
with Decision 6's text, a setup answering `setup-running`, held until exit; `--check` binds nothing); the
steps in `internal/setup/steps.go`: `root` (the directories 03 names under the root; `data/store.sqlite`
through `internal/store/create.go`, whose caller list gains `setup`), `config` (the repository URL prompt is
asked here, because the project id it derives names `projects/<id>.json` and P22's channels; the id rule and
`--project`; `instance.json` and the project file from `internal/config/setuptemplate.go`, a sibling of
`scratchtemplate.go` writing every key with a `//` comment, `privacy.allow` holding `broodforge` and
`broodforge-engine`, written only when the file is absent; a present key the host disagrees with is
`mismatch-<step>`, replaced only by `setup <step>` after `old`, `new` (a secret as its id or path) and a yes),
`key` (`credentials.Init`, the one print as the `ok` line, the pause; `skip  key  exists` for a valid key;
`mismatch-key` for a file that is not one; `setup key` only while the store holds no entry, else `key-in-use`),
`engine` (`%ProgramData%\Broodforge\engine\claude.exe` or `/opt/broodforge/engine/claude`, else the engine on
`PATH`; `--version` parsed as `spawn preflight` parses it; `engine.path` and `engine.version` written; `fix
engine  install Claude Code` otherwise), `git-credential` (hidden, or Enter to defer, `handoff: "none"`; `setup
git-credential` reads stdin alone; the credential through `credentials.Service.Add` on the store `setup` opened,
a new id each time, the old one `removed`), `project` (the clone into `projects/<id>/` through the `git`
package's canonical create, the stored credential in that one git process through `GIT_ASKPASS` as the push
route; `skip` when present with the same remote; a refused clone the companion's `fix`), `defaults`, `check`
(`config check` and `spawn preflight` run in process, each line echoed), `next`; `internal/cli/setup.go`;
`docs/design/05-debug-interface-setup.md` [no-check: written by this pass] (Drive, Observe, Reproduce, Isolation
from the companion, nothing the companion does not say) and its README row; the Command tree of
`05-debug-interface.md`; the `03-architecture.md` row and the `go.md` line of the standing rules;
`tests/repro/setup.sh` and its `tests/README.md` row; `../operator/2026-10-14-credential-store-key-per-host.md`
and `../operator/2026-10-23-allow-the-service-account-in-privacy-allow.md` deleted (`key` and `config` do their
steps). Mid-pass commit after the runner, `root`, `config`, `key` and the bind; then the rest. Proof: `scratch
run -- bash tests/repro/setup.sh` exit 0 (a stub `claude` on the script's
`PATH` printing a version): on an empty temp root `setup --yes --root <r> --instance t --repo <bare> --wait 1`
prints `ok` for every step but the Discord ones (`skip  ...  not built`) and `git-credential` (`skip`,
deferred), exit 0, with `config check --root <r>` passing after it; the same command again prints only `ok`
and `skip` and `git diff --no-index` of the root's `config/` before and after is empty (prediction 1);
`setup --check` on the empty root prints `fix` for every step, creates nothing and exits 1 `check-failed`; `setup --yes` with no
`--repo` exits 1 with `error: needs-input-repo: pass --repo`; with `serve` running on the root, `setup` exits
1 with the `running` text and `setup --check` still runs; with `setup` paused at `key`, `serve` fails
`start-control` and a second `setup` `setup-running`, `status --expect setup` exits 0; `setup --json` carries
the key in the `key` object only. `go test ./internal/setup -run 'TestSteps|TestRerun|TestRefusals|TestSingle
Writer|TestJSON'` and `go test ./internal/cli -run TestProofSetup`. Owner step: `bin/broodforge setup --root
<a development root> --instance dev` under his own account, reading the prompts; wording goes to the handoff's
Waiting on user. Does not: call Discord; write a present key; touch `~/.claude`. Depends on: P12d.

## P22. The Discord steps: token, application, guild, channels, role (medium; owner sees the invite and the role prompts)

Implements: ADR 0027 (Decision 3; 6's token, invite, role and channels texts; 8's C5 and C6 fallbacks); the
companion (`token`, `application`, `guild`, `channels`, `role`; `check` gains its `discord check` line;
`--guild`, `--channels`, `--create-role`, `--role`, `--wait`; the rerun rule's token, channel, role and Manage
Roles entries); `05-debug-interface-discord.md` (The Discord adapter: `discord check` unchanged). Files
[no-check: new files and new code in named ones]:
`internal/discord/setup.go` (`SetupAPI`, the interface `internal/setup` takes: the application behind the
token and its flags, the message-content flag write, the bot's guilds, a guild with its `owner_id`, its
channels and roles, create a channel under a category, create a role, assign a role, the bot's member and its
highest role's position, the bot's guild permissions; `NewSetupAPI(token)` over `restAPI`, which gains
`UpdateCurrentApplication`, `GetCurrentUserGuilds` with the empty bearer token, `GetRoles`, `CreateRole`,
`AddMemberRole`; the invite URL from `PermissionBits`, ORed with Manage Roles on a yes);
`internal/setup/discord.go` (the five steps: `token` read hidden or from stdin, `CheckToken`'s shape first, stored as the
instance's `bot-token` through the store `setup` opened, `discord.botCredentialId` written, `skip` when the
stored one answers the application call, otherwise `fix` with Discord's reason and the prompt again, the new
token a new id, `botCredentialId` rewritten and the old entry `removed`; `application` writes
`discord.applicationId` and ORs `1<<19` unless `1<<18` or `1<<19` is set, `fix` naming the portal switch
when the PATCH is refused; `guild` with the role question before the link, the link, `waiting for the bot to
join` polled to `--wait`, a list when several, `--guild`; `channels` by configured id, then by name, the ask
when the guild has channels, `Sessions`, `#<id>`, `#<id>-notices` created under the category, the ids written
to `discord.guildId`, `noticeChannelId`, the project's `venue.category` and its `channels.<lobby id>`, a
configured id gone from the guild `mismatch-channels`; `role` by configured id, then by name, the no branch's
`fix`, the yes branch's create, position check against the bot's highest role (`fix` otherwise), assign to
`owner_id`; then, on every run, the bot's permissions read and the removal `fix` polled while they carry
Manage Roles; `refused-<step>` and `timeout-<step>` as the table); `internal/setup/fakediscord_test.go` (a
`SetupAPI` fake with a scripted guild: empty, with channels, with the role present, refusing the PATCH, a
role landing above the bot); `internal/discord/setup_test.go` (`TestSetupAPI` against the `restAPI` fake of
`gateway_test.go`, every call's request body recorded); `tests/repro/setup.sh` gains the Discord steps against
the fake through `BROODFORGE_SETUP_FAKE_DISCORD=<script>`, read only when `discord.mode` of the root is not
`live` (a live root refuses it `live-instance`);
`../operator/2026-10-21-discord-application-bot-and-test-server.md` deleted (the portal floor stays as the
`token` prompt's text and the reference file of P23).
Proof: `tests/repro/setup.sh` exit 0 with the Discord steps `ok` on the first run and `skip` on the second,
`channels` creating nothing in the scripted guild with channels under `--channels no`, `role` `fix` on the
scripted no; `setup --yes` with the fake's bot in two guilds and no `--guild` is `needs-input-guild`; a second
run against a scripted guild whose bot still holds Manage Roles prints the removal `fix` again; a configured
channel id the scripted guild lacks is `mismatch-channels`; `discord check` echoed in `check`; `go
test ./internal/setup -run 'TestDiscordSteps|TestRole|TestChannels'` and `go test ./internal/discord -run
TestSetupAPI`. Owner step, the first real run: `setup` on the development root of P21 against his test guild
with a fresh application; the bot joins, the channels and the role appear, Manage Roles is removed, `serve
--discord live` on that root is the vision's setup in few steps on the PC in its development form; the new
role's position (C6) and the count of hand actions go in a finding against the Appendix's 18. Does not:
delete or rename a channel or role; hold the token in memory past the step. Depends on: P21.

## P23. `setup host` and `setup service` on Windows (medium; owner steps; P13 built here)

Implements: ADR 0027 (Decision 5, Windows; 7's account amendment; Consequences, prediction 3); the companion
(`setup host`, Windows; `setup service`, the task; On the PC: which shell); ADR 0022 (Windows, as amended, the
directory ACL included); `03-architecture.md` (Process model, Windows column: the bot task, the nightly backup
task, the Debug CLI row); P13 of 08 (its reference file and C37); P20's finding. Files [no-check: new files]:
`internal/setup/host.go` (`setup host [--root <path>] [--instance <name>] [--engine-version <v>] [--from
<export>] [--print]`: the elevation check first, `needs-elevation` with the table's text, the same code on a
denied root or pipe on Windows; a bot answering is `running`; the script built from a template with the values
filled, run by `powershell.exe -NoProfile -NonInteractive -Command -` as a child with the script then the
password on stdin, never an argument; `--print` writes the script opening with the companion's withheld line,
`<password>` in the password's place, and the files it would change, runs nothing, needs no elevation);
`internal/setup/host_windows.go` (the token's elevation bit through `x/sys/windows`; the 32-character password
from `crypto/rand`, held in memory for the run only; the template: `New-LocalUser broodforge` never expiring,
`Add-LocalGroupMember Users`, `%ProgramData%\Broodforge\` with inheritance off and inheritable entries for
SYSTEM, Administrators and `broodforge` only, the root beneath it and no key file (the inner `setup`'s
`credential init` makes it), `bin\` and `engine\` read-only to `broodforge`, the binary copied to `bin\`, the
engine copied from the owner's install into `engine\` when P20 found W4 holding else the native installer run
as `broodforge`, `--engine-version` checked against `--version`; `--from`'s file copied to `<root>\data\`;
then the inner `setup` as `broodforge` through `Start-Process -Credential -LoadUserProfile -Wait` in a console
on the desktop with `--root`, `--instance` and `--from` passed, then `setup service`; a rerun `Set-LocalUser
-Password` rotates and re-registers; `RelaunchAsAccount`, the same rotate-and-start route, which `setup` and
`setup <step>` on Windows take when the caller is not `broodforge`, flags passed, stdin not carried, the
companion's PC shell rule); `internal/setup/host_linux.go` (the `needs-elevation`
check and a stub refusing `fix  host  held` until P26); `internal/setup/service.go` (`setup service`: on
Windows the task `Broodforge` at startup with the operator file's three settings, action `bin\broodforge.exe
--root <root> serve`, and the task `Broodforge Backup` daily running `backup snapshot`, both `-User
-Password` so a Windows `setup service` alone rotates the password too; `Export-ScheduledTask` read back,
`PT0S` required, `fix  service  ExecutionTimeLimit is <value>` otherwise, P20 says which; `Start-ScheduledTask`;
`ok  service  <task>`); `tests/setup/host-windows.golden.ps1` and `service-windows.golden.ps1` [no-check:
written by this pass] compared by `go test ./internal/setup -run 'TestHostPrint|TestServicePrint'` (the
password `<password>`, the paths placeholders); `docs/reference/pc-host-setup.md` [no-check: written by this
pass] (what `setup host` creates on the PC, how to check it took: `Get-LocalUser`, `Get-ScheduledTask`'s
principal, `icacls` of the root and key, the toolchain-download lift of ADR 0023; `verified:` the owner's
run date) with its README row; `../operator/2026-10-14-pc-broodforge-account-and-task.md` deleted. Proof:
`bin/broodforge setup host --print --root x --instance main` on any machine prints the script opening with the
withheld line and holding no password, exit 0; `setup host` from a non-elevated shell exits 1
`needs-elevation`; the golden tests. Owner steps, on the PC after P20's finding: `setup host` from one elevated
PowerShell, the inner `setup`'s prompts in the console it opens, the token from the password manager; `icacls`
of the directory, the root, the key, `bin\` and `engine\` as the reference file lists; then C37's three
actions recorded as a finding: a turn from the task with no interactive session, `status` over the pipe from
the elevated PowerShell (and `connect-failed` from an unelevated one, the companion's PC shell rule), `stop`
through the task leaving no engine process; `Get-ScheduledTask Broodforge` shows `LogonType Password`, the
profile exists (prediction 3); then `setup --check` from the elevated shell and `setup defaults` relaunching
as `broodforge`. Does not: store the password anywhere; run as `broodforge` anything but `setup` and the
service; touch the owner's `~/.claude`. Depends on: P22, P20's finding.

## P24. `config export` and `setup --from` (small)

Implements: ADR 0027 (Decision 5, export and `--from`; one application per instance); the companion
(`config export`, `setup --from`, `config import`). Files [no-check: new code]: `internal/config/export.go`
(`Export(c) ([]byte, error)`: `config/` as one JSON document, the project files under `projects`, the
host-bound keys the companion lists emptied, comments kept where the parser holds them else dropped;
`Import(root, doc)` writing the files, absent only); `internal/cli/config.go` (`config export [--out <file>]`
`(file)`, default `<root>/data/config-export.json`, `written  <path>`; `config import <file>` `(file)`, refusing
a root whose `config/instance.json` exists, `config-exists`); `internal/setup/state.go` (`--from <file>`:
import first, then the run asks only the steps whose keys are empty: `root`, `key`, `engine`, `token`,
`git-credential`, `project`'s clone; `channels` and `role` find everything by the imported ids or by name in
the same guild; `setup host --from` is P23's copy into the root);
`internal/config/export_test.go` (`TestExportRoundTrip`: a scratch root exported then imported into an empty
root, `config check` passing after the host-bound keys are filled; `TestExportHoldsNoSecret`: the document
holds no `cred-` id, no key hex, no token). Proof: `config export --root <scratch> --out <tmp>/e.json` then
`setup --from <tmp>/e.json --yes --root <r2> --instance t2 --wait 1` on an empty root prints `ok` for the
host-bound steps and `skip` for the imported ones; `config import` on a configured root is
`config-exists`; `grep -c cred- <tmp>/e.json` prints 0. Does not: export the store or a credential; retire
the first instance (the owner's `stop`). Depends on: P22.

## P25. `setup login` (small; owner sees the browser step)

Implements: ADR 0027 (Decision 5, `setup login`; 8: until the relay exists); the companion (`setup login`);
`05-debug-interface.md` (Credentials: `credential login <id> --done`). Files [no-check: new code]:
`internal/setup/login.go` (`setup login <credential id> [--print]`: `needs-elevation` first; the credential
read from the store file for its kind, `bad-kind` unless `subscription`; Windows: rotate the password and
re-register the tasks as P23's service does, then `Start-Process -Credential -LoadUserProfile -Wait` running
`<engine> auth login` with `CLAUDE_CONFIG_DIR=<root>\harness\<id>` in a console on the desktop; Linux: `sudo -u
broodforge-engine` with the same environment, stubbed `fix  login  held` until P26; on the child's exit 0,
`credential.login` with `done` over the socket, the list line printed; a child exit other than 0 is
`login-failed` with its code, nothing recorded; `--print` writes the command with `<password>`);
`tests/setup/login-windows.golden.ps1` [no-check: written by this pass] and `go test ./internal/setup -run
'TestLoginPrint|TestLoginDone'`, the second through `internal/control/controltest` with a `credential.login`
handler recording the call. Proof: the golden test; `setup login cred-x --print` prints the command and exits
0 without elevation; through the harness `setup login <id>` with a stub engine exiting 0 records `usable` and
with one exiting 1 is `login-failed`. Owner step: on the PC instance of P23, `/register` and `/credential add
kind:subscription` in Discord, then `setup login <id>` from an elevated PowerShell, the browser sign-in, the
code pasted back when the callback fails; a first real session from Discord under the service account
follows, the Appendix's 18 counted against the finding of P22. Does not: build the relay; print the login
URL into a log. Depends on: P23.

## P26. The droplet: `setup host` and `setup service` on Linux, `setup login`'s Linux half (medium; HELD with P14)

Implements: ADR 0027 (Decision 5, Linux); the companion (`setup host`, Linux; `setup service`, the unit and
the timer; `setup login`, Linux); proposal 0027 (L1); ADR 0022's companion (Users, ownership, sudoers;
Operator prerequisites); `03-architecture.md` (Process model, Linux column; Signals and leftovers). Files
[no-check: new code and deleted files]: `internal/setup/host_linux.go` (`os.Geteuid` for the elevation check;
the script run by `sh -s` on stdin: the group and the two users, the root and modes as the operator entry's
`install -d` lines, the sudoers file written to a temp file from the `env_keep` list `spawn preflight`
generates and installed only after `visudo -cf` passes (L1), `apt-get install bubblewrap socat`, the AppArmor
profile on Ubuntu 24.04 only, the engine ELF at `/opt/broodforge/engine/claude` staged by the native installer
into a temp `HOME`, the binary at `/opt/broodforge/bin/broodforge`, then `sudo -u broodforge broodforge setup`
on the same terminal, then `setup service`); `internal/setup/service.go` (Linux: the unit `broodforge.service`
with `KillMode=mixed`, `User=broodforge`, `ExecStart` naming the root, and `broodforge-backup.timer` nightly,
`systemctl enable --now` both; `ok  service  <unit>`); `internal/setup/login.go`'s Linux branch;
`tests/setup/host-linux.golden.sh` and `service-linux.golden` [no-check: written by this pass];
`docs/reference/droplet-host-setup.md` [no-check: written by this pass; P14 names it too] (what `setup host`
creates, the check commands of the three droplet operator entries as run, `verified:` the VM's date then the
droplet's); the three droplet operator entries deleted (`../operator/2026-10-14-droplet-users-and-sudoers.md`,
whose stale "not yet built" line goes with it, `../operator/2026-10-14-droplet-bubblewrap-and-apparmor.md`,
`../operator/2026-10-14-droplet-engine-go-and-broodforge.md`). Proof: the golden tests on any host; on the VM
from a snapshot taken before any Broodforge step: `sudo broodforge setup host --print`, then `setup host`,
then the operator entries' own checks (`id`, `stat`, `sudo visudo -c`, `sudo -n -l -U broodforge`, the bwrap
`true` as `broodforge-engine`, `--version` from both users with a minimal `PATH`) all as the entries expect,
`systemctl status broodforge` active, `status` from a shell as `broodforge`; a second `setup host` prints
only `ok` and `skip`; then the same on the droplet, with P14's own Proof (C30, C31, the interrupt) after it.
Does not: install Go or gcc (the build host's, named in the reference file); lift P14's hold on its own.
Depends on: P25, P14's droplet rows; the VM.

## Where each command lands

| Command | Pass |
|---|---|
| `setup` (the runner, `--check`, `--yes`, `--json`, `--wait`, `setup <step>`, the refusals, the bind) | P21 |
| Steps `root`, `config`, `key`, `engine`, `project`, `git-credential`, `defaults`, `check`, `next` | P21 |
| Steps `token`, `application`, `guild`, `channels`, `role`; `check`'s `discord check` line | P22 |
| `setup host`, `setup service` (Windows; `--print` on every host); the relaunch of `setup` as `broodforge` | P23 |
| `config export`, `config import`, `setup --from` | P24 |
| `setup login` (Windows; `--print` and `--done` on every host) | P25 |
| `setup host`, `setup service`, `setup login` (Linux) | P26 |

Not planned here, as ADR 0027 leaves them out: the login relay (the browser step and `setup login` stay), a
release of the binary (the build row stays in every count), the engine's sandbox on Windows (ADR 0022's
trigger).
