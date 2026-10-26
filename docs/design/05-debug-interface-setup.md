# Debug interface: setup

This is the setup part of `05-debug-interface.md` and shares its Conventions. The surface is the
companion of ADR 0027, `decisions/0027-setup-commands.md`: the flags, the steps in order, the rerun
rule per key, the refusal table and the owner's texts are there and are not repeated here.

## `broodforge setup`

Drive. `broodforge setup [<step>] [--root <path>] [--instance <name>] [--repo <url>] [--project
<id>] [--guild <id>] [--channels yes|no] [--create-role yes|no] [--role <name>] [--from <export>]
[--check] [--yes] [--wait <seconds>] [--json]` runs the steps in the companion's order (`root`,
`config`, `key`, `engine`, `token`, `application`, `guild`, `channels`, `role`, `git-credential`,
`project`, `defaults`, `check`, `next`), one line each: `ok  <step>  <detail>`, `skip  <step>
<already>` or `fix  <step>  <what the owner does>`; `--json` one object each,
`{"result":"ok|skip|fix","step":"<step>","detail":"<text>"}`, the `key` step's object adding
`"key"` and `"path"`, the one print. Every prompt has a flag; `--yes` takes every shown default and
refuses `needs-input-<step>` (`pass --<flag>`) where there is none. `setup <step>` runs that step
alone and may replace its keys after `old  <value>` and `new  <value>` records (`--json`:
`{"step":"<step>","old":<value>,"new":<value>}`) and a yes (`--yes` accepts): the only overwrite of
a present key; for `token`, `git-credential` and `key` the values are the credential id and the key
path. A `fix` that needs a hand polls until the step passes or `--wait` (default 900, `0`
unbounded) runs out, `timeout-<step>`. Exit 0 every step `ok` or `skip`; 1 a step failed, its `fix`
line on stdout and one stderr `error: <code>: <message>` from the refusal table (`--json`: the wire
error object); 2 usage. With no `--root` and no `BROODFORGE_ROOT` the default root applies
(`%ProgramData%\Broodforge\<instance>` or `/srv/broodforge/<instance>`, instance `main`). The bot
token is read hidden on a terminal, or from stdin with `--yes` (off a terminal stdin is otherwise the
key's Enter alone, so without `--yes` the token step is `needs-input-token`); with no token stored, the
token step's `fix` line names the Developer Portal steps. A waiting `fix` may add one stderr line
before its poll (`waiting for the bot to join` after the invite; none under `--json`). Once a run
has filled the Discord ids, `discord.mode` is switched from `dry-run` to `live`. The role goes to
the guild's `owner_id`, not to whoever runs setup; the `role` line names it: `ok  role  <name>
<role id>  held by <member id>`.

Observe. `setup --check` prints every step's line, changes nothing, binds nothing, polls nothing,
and exits 1 with `error: check-failed: <steps>` naming the steps that printed `fix`; it is
`(file)` and runs beside a bot. While a setup runs it holds the control address: `status` answers
`config: setup` with `-` for every other line (`status --expect setup` matches), `serve` on the
root fails `start-control`, a second `setup` is `setup-running`. A bot answering at the address is
`running`, with the owner's text. Afterwards `config check` and `config show` read the files setup
wrote, and `credential list` shows the git credential it added by id. The project file keeps the
repository URL as the comment key `//url` of its `repo` object, which config ignores, so a rerun
after a refused clone needs no `--repo`.

Reproduce. Discord without Discord: `BROODFORGE_SETUP_FAKE_DISCORD=<script>` makes the five
Discord steps and `check`'s `discord check` call a scripted Discord in place of the real one, the
script a JSON `discord.FakeScript` (the token it accepts, the application and its flags, the
guilds the bot is in, the guild it joins on the poll after the invite, and the owner's hand steps
after a number of polls); every change is written back to the file, so a rerun sees what the first
run created, and its `calls` list records every write. It is refused `live-instance` on a root whose
`discord.mode` is `live`, and a run on it leaves `discord.mode` `dry-run`.
`bin/broodforge scratch run -- bash tests/repro/setup.sh` drives the steps against it on roots under
a scratch root, the scratch root's `remote/scratch.git` as the repository and a stand-in engine on
`PATH`: a fresh `--yes` run with the token on stdin (the invite and the Manage Roles removal `fix`,
then `ok`), the rerun (only `ok` and `skip`, `config/` unchanged, nothing written to Discord),
`--check` on an empty root and on a half-written config, `needs-input-repo`, `needs-input-token`
with the portal `fix`, `setup key` on a present key then `key-in-use`, `mismatch-key`, `running`
with a bot on the root, a setup paused at the key (`start-control`, `setup-running`, `status
--expect setup`), `--json`, a server with channels under `--channels no` (`timeout-channels`,
nothing created), two servers without `--guild` (`needs-input-guild`), the bot keeping Manage
Roles (the removal `fix` on two runs), a configured channel gone (`mismatch-channels`), and the
script on a live root (`live-instance`). The Go proofs: `go test ./internal/setup -run
'TestSteps|TestRerun|TestRefusals|TestSingleWriter|TestJSON|TestDiscordSteps|TestRole|TestChannels'`,
`go test ./internal/discord -run 'TestSetupAPI|TestInviteURL|TestFakeSetup'` and `go test
./internal/cli -run TestProofSetup`.

Isolation. No line prints a secret: the key once, in the `key` step's line, never again; the bot
token and the git credential are read hidden or from stdin and stored by id. One secret per
command: the bot token on `setup`'s stdin, the git credential hidden in the interactive run or on
`setup git-credential`'s stdin alone. `setup` binds the control address before its first write and
holds it until exit, so the store keeps one writer (ADR 0022 as amended by ADR 0027); `--check`
binds nothing. A rerun never overwrites a present config key, a valid key file, a stored
credential or a channel or role it found, and never deletes or renames a channel or role. The
Discord steps build their client from the stored token for one step and close it after; the
scripted Discord never runs on a live root.
