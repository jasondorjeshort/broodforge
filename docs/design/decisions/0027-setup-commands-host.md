# 0027 companion: `setup host`

Companion to `0027-setup-in-few-steps.md` (Decision 5) and to `0027-setup-commands.md`, split from it
by topic on 2026-11-01 when that file reached its cap: the elevated host part, per host. Source:
`../proposals/0027-setup-in-few-steps.md`.

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
