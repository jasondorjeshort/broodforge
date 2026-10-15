# Spike plan, droplet half

Companion to `07-spike-plan.md`: its Order table ranks these claims and its Shared setup names
(`$S`, `$CFG`, `$RUN`, `$FLAGS`, `<bf>`, the hook) apply here. Date: 2026-10-14. Status: plan.
Everything here waits on the droplet operator entries (the files in `docs/operator/` whose slug
starts `droplet-`) and runs as `broodforge` on the droplet, in `sh`. The companion of ADR 0022,
`decisions/0022-linux-spawn-and-handoff.md`, is "the companion" below: its bind block, ownership
table, maintenance commands and Interrupt steps are what the spike builds from.

## Prerequisites and layout

Packages: the operator entries install `bubblewrap` and `socat`; C38 also needs `strace` inside
the sandbox, and `/usr` is bound, so one more line as root, run once with the bubblewrap entry:
`sudo apt-get install -y strace`.

Paths, in the companion's names. `<root>` is the instance root the owner chose; `<sess>` is
`<root>/sessions/spike` (a second session is `<root>/sessions/spike2`, C38); `<cfg>` is `$CFG`,
`<root>/harness/spike`; `<settings>` is `<root>/settings/spike.json`, written by `broodforge`
from the template; `<canon>` is `<root>/projects/spike`, bot-owned as the ownership table says;
`<ecfg>` is `<cfg>` for `sub` and `<sess>/home/.claude` for `key`. `<bf>` is `/opt/broodforge/spike`:
the owner runs `sudo install -d -o broodforge -g broodforge-engine -m 0750 /opt/broodforge/spike`
once, then `broodforge` copies the compiled `spike/` there without root, and the engine user reads
it as a group member. `<engine>` and `<node>` are the engine entry's. `$TREE` is `<sess>/tree`,
the clone `worktree-create` makes of `<canon>`, whose `base` commit carries `hello.txt`.

The bwrap route. `<bf>/bwrap.sh` [no-check: created by the coder] builds the companion's
`sudo -n -u broodforge-engine /usr/bin/bwrap` command from its bind block, with `<bf>` bound in
place of Broodforge's install: `bwrap.sh --reach code|read-only --cred sub|key -- <command>`
is the session sandbox (binds by reach and credential kind, `HOME=<sess>/home`, cwd `<sess>/tree`,
the command run under `engine-entry`); `bwrap.sh --maintenance <dir> -- <command>` is the
maintenance shape (system, Node and `<bf>` binds, no config directory, no `<settings>`,
`HOME=/tmp`, `<dir>` bound read-write), which is how `engine-entry harness-create`,
`worktree-create` and `worktree-bundle` run. `engine-entry` is the companion's `sh` wrapper
copied into `<bf>`. Before any claim: `bwrap.sh --check` runs the `namei -m` walk over every
bind source and `engine-entry --check` and prints the kept names seen; a missing name stops the
spike with the sudoers block to add.

Setup, in order, no engine: `bwrap.sh --maintenance <root>/harness -- engine-entry
harness-create <cfg>`; `node <bf>/setup.js canon --droplet` (creates `<canon>` as `broodforge`
with `git init -b base` and the `hello.txt` commit, through `child_process` as the plan's
Fixture commits paragraph requires); `bwrap.sh --maintenance <root>/sessions -- engine-entry
worktree-create <sess> <canon> spike base` (creates `<sess>/home`, `<sess>/home/.claude`,
`<sess>/cfg/<name>` for every per-session name, and the clone); `<settings>` from the template.
C25 here is the relay shape: `CLAUDE_CONFIG_DIR=<cfg> node <bf>/login.js --spawn bwrap` runs the
engine's login through `bwrap.sh --cred sub` with piped stdio, prints the URL, and writes the
code the owner pastes into the `broodforge` shell to the login's stdin; `.credentials.json` lands
engine-owned under `<cfg>`.

Hook mode and log. `<sess>/home` is engine-owned `0750`: `broodforge` reads it as a group member
and cannot write it, so the mode file `<sess>/home/spike-hook.json` is written through the
sandbox by `<bf>/hookmode.sh <mode> [<tool_use_id>:allow|deny]` (one `bwrap.sh --reach code --
sh -c 'printf ... > <sess>/home/spike-hook.json'`), the only route that can; `hook.log` is
`<sess>/home/hook.log`, named by `BROODFORGE_HOOK_LOG` on the kept list and read from outside.
`$RUN --spawn bwrap --reach <reach> --cred <kind>` builds every engine turn below through
`bwrap.sh`; `--kill-after` and `--deadline` use the companion's Interrupt steps.

## Claims

**C30.** Each reach shows nothing beyond the bind list; read-only writes fail `EROFS`; the
network works. Run, no engine, for each reach: `bwrap.sh --reach code -- sh -c 'ls / <root>
<root>/sessions <root>/harness <root>/projects; ls <root>/config <root>/settings; cat
<root>/config/instance.json; env; grep NSpid /proc/self/status; touch <sess>/tree/x; getent
hosts api.anthropic.com; curl -sS -o /dev/null -w "%{http_code}\n" https://api.anthropic.com/'`
with `<root>/sessions/spike2` present on the host. Evidence: `ls <root>` prints only the
mount-point names; the second session and `config/` are `No such file`; `env` shows `HOME`,
`PATH`, the kept names and nothing else; the `NSpid` line is recorded (C33 compares against it);
`touch` under read-only reach says `Read-only file system`; `getent` prints an address and
`curl` a status. Refuted: stop and report (ADR 0022); the bind block changes.

**C31.** The settings file cannot be replaced mid-turn and the hook keeps firing, both
credential kinds. Run, no engine, for `sub` and `key`: `bwrap.sh --cred sub -- sh -c 'touch
/tmp/f; echo x > <ecfg>/settings.json; rm <ecfg>/settings.json; mv /tmp/f
<ecfg>/settings.json; echo $?'`; then one engine turn per kind, the `key` turn once the owner
funds a key: "Run these commands in Bash, then write done.txt." with the same command string.
Evidence: `Read-only file system` for the write, `Device or resource busy` for `rm` and `mv`;
`hook.log` has the Write line after the failures. Refuted: stop and report (ADR 0022); the
critic's alternative is `--settings` with the JSON inline, which C26 says the watcher covers or
not.

**C32.** `sudo -n -u broodforge-engine /usr/bin/kill -INT <pid>` mid-turn yields a `result`,
and the session resumes after it. Run: `$RUN --spawn bwrap --kill-after 5000 --prompt "Run
\`sleep 30\` in Bash, then say done."`; the spike walks `pgrep -P` from the sudo child to the
first `/proc/<pid>/comm` equal to `claude` and sends the kill; then `$RUN --spawn bwrap
--resume <id> --prompt "Continue."`. Evidence: the `comm` chain printed (`sudo`, `bwrap`,
`bwrap`, `claude`); a `result` with `modelUsage` within the 20 s default; the exit code; the
resumed turn reaches a `result`. Refuted (no `result`): the companion's Interrupt step 4 (kill
the sudo child) is the only path, every Linux interrupt is recorded with output unknown (ADR
0020), and the `kill` sudoers line goes; (no resume): the Linux line of ADR 0020 Interrupt gets
S2's fallback, a new engine session per interrupted turn.

**C27 and C39 on the droplet.** ADR 0020 states both claims for both hosts. Run: C27's four
turns exactly as on the PC through `$RUN --spawn bwrap`, the hook mode set by `hookmode.sh`
between turns; then C39's `exit2` turn only, since exit 1 and the timeout exercise the engine's
default, not the host. Evidence: as on the PC, recorded as droplet rows beside the PC rows in
the finding. Refuted on one host only: the ADR row records both results and the backend for
that host takes the fallback.

**C35, droplet.** A plain clone by the engine user of the bot-owned canonical dies on dubious
ownership; the `-c` form succeeds; the commit and bundle as on the PC. Run, no engine:
`bwrap.sh --reach code -- git clone <canon> /tmp/plain` (the plain clone as the engine user
inside the sandbox, `/tmp` being its tmpfs; `git` under `sudo -u broodforge-engine` is not in
sudoers and is never added), expecting the fatal; the `-c` form is the clone `worktree-create`
already made as `<sess>/tree`; then `bwrap.sh --reach code -- git -C <sess>/tree checkout -b
spike`; `node <bf>/setup.js commit --droplet` (the commit through `bwrap.sh --reach code` under
the four variables, which reach it through `env_keep`); `bwrap.sh --maintenance <sess> --
engine-entry worktree-bundle <sess> spike`; and as `broodforge` in `<canon>`: `git bundle verify
<sess>/out.bundle`, `git fetch <sess>/out.bundle spike:refs/sessions/spike`, `git log -1
--format='%an %ae %cn %ce' refs/sessions/spike`. Evidence: `fatal: detected dubious ownership`
from the first, success from the rest, the four fields from the log. Refuted (the `-c` form
fails too): `safe.directory` goes into `<sess>/home/.gitconfig` written by `worktree-create`, an
amendment to the companion's clone step.

**C38.** A subscription turn writes under `<cfg>` only per-session names and the credential's,
reads instructions only from the loaded list; planted files are invisible to a second session;
the refresh works under the binds. Three turns. Run: `$RUN --spawn bwrap --strace --prompt
"Read hello.txt and reply with its one word."`, where `--strace` puts `strace -f -e trace=file
-o <sess>/home/strace.log` before `<engine>/claude` in the entry; then `<bf>/strace-cfg.sh
<log> <cfg>` prints every path under `<cfg>` with the syscall and open flags, split into written
and read. Planting: a turn from `<sess>` asks Bash to `echo x >> <cfg>/CLAUDE.md; echo x >
<cfg>/agents/x.md; echo x > <cfg>/shell-snapshots/planted.sh; ls <cfg>/shell-snapshots`; then a
turn from `spike2` (its own `worktree-create`, the same `<cfg>`) asks for `cat <cfg>/CLAUDE.md;
ls <cfg>/agents <cfg>/shell-snapshots`. Evidence: the written set minus `projects/`,
`shell-snapshots/`, `todos/`, `.credentials.json`, `.claude.json`, and the read set minus the
loaded list, each empty or naming the additions; session one's first two writes fail (`EROFS`)
and its snapshot write succeeds, into its own `<sess>/cfg/shell-snapshots`; session two's `cat`
prints nothing, `agents` lists nothing, and `shell-snapshots` lists no file from session one
(`planted.sh` absent; its own snapshots may appear); `.credentials.json` mtime before and after.
The refresh cannot be forced: that part stays `open` in the finding's Next until a later turn
observes one. Refuted (a planted file visible): stop and report; an unlisted name goes to its
list (both already in ADR 0022).

**C33.** Under the profile the engine's sandbox starts inside the outer bwrap; `cat
.credentials.json` from Bash is denied; `failIfUnavailable` refuses without the profile. Run:
the settings template with ADR 0022's sandbox block (`sandbox.enabled`, the `credentials.files`
deny, `allowUnsandboxedCommands: false`, `failIfUnavailable: true`), one turn "Run `cat
<cfg>/.credentials.json; grep NSpid /proc/self/status` in Bash." Evidence: the cat denied with
the sandbox's own text, `NSpid` one level deeper than C30's probe printed, nothing on stderr
about an unavailable sandbox. The negative half as written needs root to unload the profile and
does not apply on Debian 12 (it stays `open` there); testable form: on Ubuntu the owner runs
`apparmor_parser -R` once, the turn is repeated expecting the Bash call refused naming the
sandbox, `-r` restores it (5 minutes of the owner's). Refuted (no inner sandbox): ADR 0022
already names it, the block is dropped and the scrub kept.

**C34, key half.** Once the owner funds a key: `ANTHROPIC_API_KEY` in `broodforge`'s
environment for the one call, `$RUN --spawn bwrap --cred key` with the PC half's prompt.
Evidence: the Bash output lacks `ANTHROPIC_API_KEY`, keeps `BROODFORGE_*` and `GIT_*`;
`hook.log` says whether the name reached the hook. Refutations are the PC half's.

**C29.** Per-turn overhead on the droplet. Run: hook `allow`, three Haiku turns "Reply with
ok." through `bwrap.sh`, stdin closed at once, `time` around each. Evidence: wall seconds per
run beside the `result` `duration_api_ms`; the difference is the overhead, compared with C15's
PC figures. A refutation is only a number: ADR 0020 Consequences gets it.
