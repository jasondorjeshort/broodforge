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
(`%ProgramData%\Broodforge\<instance>` or `/srv/broodforge/<instance>`, instance `main`). Until the
Discord steps are built, `token`, `application`, `guild`, `channels` and `role` print `skip  <step>
not built`.

Observe. `setup --check` prints every step's line, changes nothing, binds nothing, polls nothing,
and exits 1 with `error: check-failed: <steps>` naming the steps that printed `fix`; it is
`(file)` and runs beside a bot. While a setup runs it holds the control address: `status` answers
`config: setup` with `-` for every other line (`status --expect setup` matches), `serve` on the
root fails `start-control`, a second `setup` is `setup-running`. A bot answering at the address is
`running`, with the owner's text. Afterwards `config check` and `config show` read the files setup
wrote, and `credential list` shows the git credential it added by id. The project file keeps the
repository URL as the comment key `//url` of its `repo` object, which config ignores, so a rerun
after a refused clone needs no `--repo`.

Reproduce. `bin/broodforge scratch run -- bash tests/repro/setup.sh` drives the steps without
Discord on roots under a scratch root, the scratch root's `remote/scratch.git` as the repository
and a stand-in engine on `PATH`: a fresh `--yes` run, the rerun (only `ok` and `skip`, `config/`
unchanged), `--check` on an empty root and on a half-written config, `needs-input-repo`, `setup
key` on a present key then `key-in-use`, `mismatch-key`, `running` with a bot on the root, a setup
paused at the key (`start-control`, `setup-running`, `status --expect setup`), and `--json`. The
Go proofs: `go test ./internal/setup -run 'TestSteps|TestRerun|TestRefusals|TestSingleWriter|TestJSON'`
and `go test ./internal/cli -run TestProofSetup`.

Isolation. No line prints a secret: the key once, in the `key` step's line, never again; the bot
token and the git credential are read hidden or from stdin and stored by id. One secret per
command: the bot token on `setup`'s stdin, the git credential hidden in the interactive run or on
`setup git-credential`'s stdin alone. `setup` binds the control address before its first write and
holds it until exit, so the store keeps one writer (ADR 0022 as amended by ADR 0027); `--check`
binds nothing. A rerun never overwrites a present config key, a valid key file, a stored
credential or a channel or role it found.
