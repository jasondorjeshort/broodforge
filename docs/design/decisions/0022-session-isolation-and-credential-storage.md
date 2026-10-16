# 0022. Session isolation and credential storage: two users and bubblewrap on the droplet, one account on the PC, an encrypted store, shared clones handed back as bundles

Date: 2026-10-14
Status: accepted

Record: proposal `../proposals/0022-session-isolation-and-credential-storage.md` and
`../proposals/0022-critic-review.md` (its verification table is authoritative on the facts).
Companion: `0022-linux-spawn-and-handoff.md` (ownership, sudoers, bind list, entry wrapper, signal
path, clone and bundle steps; the living spec).

## Context

ADR 0020 fixed one `claude -p` process per turn with the spawn call as the isolation wrapper, the
two credential shapes, no stored tokens, and a `PreToolUse` hook deciding from its environment;
ADR 0021 fixed the instance root and handed over where working trees and config directories land
relative to the backup set. Reach must be enforced by the host (N-14), what a session sees of its
own credential stated per host (N-15), a weaker Windows reach is acceptable and a weaker Linux
reach is not, the owner's `~/.claude` is never touched, and the first version is due within days.

Verified by the critic (sources in the review): the engine reloads `permissions` and `hooks` from
watched settings files while running; `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` strips credential-like
names from the engine's subprocesses on both hosts; the engine's sandbox is Linux-only, Bash-only,
off by default, and can deny a named file; git refuses to clone from, open or serve a repository
owned by another user unless `safe.directory` names it; bwrap handles no signal and
`--die-with-parent` is SIGKILL; sudo logs the full command line; Ubuntu 24.04 restricts
unprivileged user namespaces by AppArmor and ships no `bwrap` profile, Debian 12 does not;
`CreateProcessWithLogonW` fails from a non-interactive service; AES-256-GCM in Node, DPAPI in
Windows PowerShell 5.1 and the engine's honouring of `GIT_AUTHOR_*` were confirmed by runs.

Rejected, reasons in the proposal: ambient capabilities for the spawn; `systemd-run` as the first
layer (same-uid sessions see each other before systemd 257); a container per session (revived by
R-47); the OS secret services; a plain-text store; git worktrees off the canonical (a session
could move `main` or plant a hook the push runs under the credential); `--clearenv` with
`--setenv` for the key; a per-turn cross-account Windows spawn as the first version (W1, the step
after); WSL2 on the PC (rejected for the first version; the owner may reopen it, since it changes
what they install).

## Decision

Linux, enforced by the host. Two users: `broodforge` runs the bot and owns the instance root
(`broodforge:broodforge-engine 0710`, so the engine user can pass through it and nothing more)
with `config/`, `data/`, `backup/`, `credentials/` at `0700` and `settings/` at `0750` with the
engine group; `broodforge-engine` runs every engine process and owns `harness/<credential id>/`
and `sessions/<session id>/` (`0750`, `broodforge` in the engine group); canonical checkouts
under `projects/<project id>/` are `0750 broodforge:broodforge-engine`. The spawn is `sudo -n -u
broodforge-engine /usr/bin/bwrap ... broodforge engine-entry -- claude -p ...`, the environment
carried by sudoers `env_reset` plus an `env_keep` list generated from code, two `NOPASSWD` lines
(`/usr/bin/bwrap`, `/usr/bin/kill`) and nothing else. Binds per reach are the companion's: system
(including the resolver's `/run/systemd/resolve`), engine, Node, Broodforge and the canonical
checkout read-only; `sessions/<id>/home` as `HOME`; `sessions/<id>/tree` read-write for code
reach, read-only for read-only reach; the bot-written `settings/<session id>.json` bound read-only
over the engine's `settings.json`, wherever the credential kind puts it; for a subscription the
config directory read-write with the directories the engine writes per session bound from
`sessions/<id>/cfg/` and the names it loads instructions from bound read-only and empty; fresh
`/proc`, `/dev`, `/tmp`; pid, ipc and uts unshared; network shared. The entry wrapper exits 1
naming the variable unless exactly one of `CLAUDE_CONFIG_DIR` and `ANTHROPIC_API_KEY` is set;
instance start runs the spawn with `engine-entry --check` and refuses to start if a kept name is
missing, printing the sudoers lines expected. Interrupt: find the `claude` descendant of the sudo
child in the host process tree, `sudo -n -u broodforge-engine /usr/bin/kill -INT <pid>`, expect a
`result`; after a timeout kill the sudo child (the tree dies through `--die-with-parent`),
recorded as interrupted per ADR 0020. Creating and deleting `harness/<id>` and `sessions/<id>`
and bundling a session are `engine-entry` maintenance commands through the same route.

Linux, engine mechanisms on top (mitigation, not boundary). Every engine process gets
`CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`; no name on the kept list except the two credential names
contains `TOKEN`, `KEY`, `SECRET` or `PASSWORD`. The written `settings.json` carries
`sandbox.enabled`, a `credentials.files` deny for `.credentials.json`, `allowUnsandboxedCommands:
false` and `failIfUnavailable: true` once C33 holds; until then the block is absent and the key is
hidden from `env` only. The operator installs `bubblewrap`, `socat` and, on Ubuntu 24.04, the
Claude Code-documented unconfined-userns AppArmor profile for `/usr/bin/bwrap`, the path the engine
must use. 05 must not expose an unauthenticated localhost HTTP surface while sessions run: a unix
socket under `data/` or a token sessions cannot read.

Windows. The installed PC instance runs entirely as a local standard account `broodforge`: the
instance root, the key file and the canonical checkouts in its profile, the engine installed for
it, started by a Task Scheduler task with "run whether user is logged on or not". Sessions are
the same account: a session's Bash reads the key file, the store, every config directory, can
rewrite its `settings.json` mid-turn and can reach the debug transport; the owner's profile,
`~/.claude` and checkouts are unreadable by default ACLs. The scrub and the written
`settings.json` without a sandbox block apply; interrupt is the ADR 0020 tree kill. Development
and scratch runs stay under the owner's account with the same-account backend.

Both hosts. The spawn interface has two backends chosen at start, `same-account` (Windows, scratch
on both hosts) and `linux-sudo-bwrap`; nothing else differs. Store: `credentials/store.json`, one
AES-256-GCM entry per credential id with the id as AAD, written by temp file and rename by the bot
only, read by open-read-close; the key is 32 bytes in the file the instance config names (default
`/etc/broodforge/<instance>.key` at `0400 broodforge`; `%LOCALAPPDATA%\Broodforge\<instance>.key`
of the bot's account), `BROODFORGE_KEY_FILE` overrides, `credential init` writes and prints it
once, a missing key refuses start naming the config key (exit 1); it holds the bot token, API keys
and the per-project git credential (R-36). Delivery: `ANTHROPIC_API_KEY` or
`CLAUDE_CONFIG_DIR=harness/<credential id>` on the engine process only; `settings/<session
id>.json` rewritten from the template before every process, outside every session bind;
`.credentials.json` and `.claude.json` excluded from the nightly backup by name as "users log in
again" (C36 may lift `.claude.json`); `credential login <id>` prints the exact command; the relay,
off by default, runs the login through the spawn backend and maps Discord `50007` to "enable
direct messages from server members and retry". Every engine process gets the four `GIT_AUTHOR_*`
and `GIT_COMMITTER_*` variables, all the driver's.

Working trees. One shared clone per session: `git -c safe.directory=<canonical> clone --shared
<canonical> sessions/<id>/tree` on the session's branch, run by the engine side; the bot sets
`refs/sessions/<id>` in the canonical to the base commit at start. Read-only reach gets the same
clone bound read-only. Subagent worktrees are `git worktree add` inside the clone. Handoff: the
engine side writes `sessions/<id>/out.bundle`; the bot fetches it into `refs/sessions/<id>` in the
canonical, checks every new commit (author is the driver; R-32 trailers present) and refuses the
push listing the commits otherwise, then pushes or opens the pull request from the canonical with
the git credential in that one git process's environment through `GIT_ASKPASS`. The bot never opens
a session's repository; no session writes the canonical. The same route on Windows. R-30's
"worktree" is read as a cheap, isolated per-session tree with subagent worktrees inside it.

Debug commands by working name (05 fixes names and transport): `credential init|add|remove|
list|login`, `session start --reach <name>`, `session reach-report <id>` (computed from the bind
list), `worktree bundle <id>`, `worktree handoff <id>`, `spawn preflight`; exit codes per N-07.
Out of scope: the hook's environment names, config format and branch naming (03); the debug
transport (05); the deny text's wording and the committer identity (04 and the owner).

Deferred, with triggers: L3 outside L2 (a runaway, or the owner wants memory and CPU caps);
`--unshare-net` with a proxy allowlist (the owner requires egress control); a container per
session (R-47); W1 (a clean `Start-Process -Credential` spike from a scheduled task);
`systemd-creds` for the droplet key (the owner wants it off the filesystem); WSL2 (the owner
wants the droplet's reach on the PC); an automatic trailer rewrite at handoff (03).

Conditions the first implementation must check (living table; the coordinator records each
result here when the spike's finding lands, the one amendment this ADR allows):

| Claim | What the spike shows | Result |
|---|---|---|
| C30 | Droplet, each reach: `ls`, `env`, `cat` of the instance root, another session's directory and the canonical from inside show nothing beyond the bind list; writes under read-only reach return `EROFS`; `getent hosts api.anthropic.com` resolves and `curl -sS https://api.anthropic.com/` gets an HTTP response from inside | open |
| C31 | Both credential kinds: the session overwrites, `rm`s and `mv`s over `settings.json` mid-turn; each fails (`EROFS` or `EBUSY`) and the hook keeps firing | open |
| C32 | `sudo -n -u broodforge-engine kill -INT <pid>` mid-turn yields a `result` with `modelUsage` | open |
| C33 | Under the named AppArmor profile the engine's sandbox starts inside the outer bwrap; `cat .credentials.json` from Bash is denied; `failIfUnavailable` refuses when the profile is absent | open |
| C34 | Under the scrub, `env` in Bash shows no `ANTHROPIC_API_KEY` or `CLAUDE_CONFIG_DIR` on both hosts and the hook still receives every `BROODFORGE_*` variable | holds on the PC for Bash, 2026-10-16: no `CLAUDE_CONFIG_DIR` in Bash, every `BROODFORGE_*` name in the hook; the hook loses `CLAUDE_CONFIG_DIR` too (Amendments from the spike); key half and droplet open |
| C35 | On the distro's git: a plain `git clone` by the engine user of the bot-owned canonical dies on dubious ownership (the reviewer could not reproduce this on git 2.56 for Windows, whose ownership check differs; the critic's V8 cites `clone.c`), the companion's `-c safe.directory` form succeeds; a commit inside the sandbox has author and committer the driver; the bundle fetches into `refs/sessions/<id>` | holds on the PC half, 2026-10-16: the `-c safe.directory` clone, the commit under the four variables and the bundle fetch into `refs/sessions/<id>` carry the spike identity in all four fields; the dubious-ownership half needs the droplet's two users |
| C36 | What `.claude.json` holds after a login (secret or not) | holds on the PC, 2026-10-15: metadata and a feature-flag cache, no key named like a secret; the exclusion stays (Amendments from the spike); droplet open |
| C37 | PC: the instance under `broodforge` from a scheduled task with no interactive session runs a turn; `taskkill /T` ends the tree; the owner's profile is unreadable from a session's Bash | open |
| C38 | Pinned engine, subscription: a turn's file access under the config directory (`strace -f -e trace=file` on the spawn) names no written directory off the per-session list but the credential's and no read instruction source off the loaded list; a `CLAUDE.md`, `agents/x.md` and a shell snapshot planted in the shared directory by one session are invisible to a second session of the same credential; the token refresh still succeeds under the binds | open |

An implementation that finds C30, C31, C35 or C38's planted files visible stops and reports; an
unlisted name C38 finds is added to its list; C33 failing drops the sandbox block and keeps the
scrub; the spike's output is a finding promoted to this file.

## Consequences

Easier: on the droplet the bot's secrets, the store, the key, other sessions' trees, homes and
transcripts, and other projects are unreadable by a session by permission and bind, not by
instruction; the hook cannot be dropped mid-turn, because its settings file is the bot's and bound
read-only; the bot never runs git against a session-controlled repository; one bundle route and
one spawn interface everywhere; `reach-report` is computed from the bind list the spawn uses.

Harder or costly: two users, two sudoers lines, three packages and an AppArmor profile on the
droplet; a second Windows account the owner creates and logs into once; a checkout and seconds of
clone and bundle per session; the bind list, the two config-directory lists, the `env_keep` list
and sandbox keys track engine releases; sessions sharing a credential still share its credential
files and whatever the two lists miss; the Windows gap list is long; a session's own credential is
a mitigation on Linux and open on Windows; egress is open.

Must be maintained: the companion whenever a path, a variable or the engine's needs change; the
conditions table; the backup exclusions in 0021's companion; the operator entries per host.

Supersede when: the engine's sandbox gains a mode that wraps the engine itself; R-47 brings a
container runtime to the droplet; the owner reopens WSL2; Windows sessions must be isolated from each other.

## Amendments after review (2026-10-14)

- `<root>` is `broodforge:broodforge-engine`, not `broodforge:broodforge`: at `0710` the engine user could not enter it, so bwrap could bind nothing beneath.
- The hook's settings file is `settings/<session id>.json`, bot-owned, bound read-only over the engine's path for both credential kinds: the bot could not write into engine-owned `harness/<id>`, and an API-key session's settings sat in its writable home.
- Subscription sessions get per-session `projects/`, `shell-snapshots/`, `todos/` and empty read-only `CLAUDE.md`, `agents/`, `commands/`, `skills/`, `plugins/` over the shared config directory; C38 added: sessions of one credential could read each other's transcripts and plant instructions.
- `/run/systemd/resolve` bound (`-try`) and a network check in C30: Ubuntu 24.04 resolves through it.
- `worktree-create` and `worktree-remove` bind `sessions/`, not `<sess>`: bwrap refuses a missing source and a mount point cannot be removed.
- `harness-create` and `harness-remove` added: nothing created the engine-owned config directory.
- The clone is `git -c safe.directory=... clone`: `clone -c` writes the new clone's config, which `safe.directory` ignores. C35 records what the reviewer could not reproduce on Windows.
- The `TOKEN`/`KEY`/`SECRET`/`PASSWORD` rule is scoped to the kept list, so `BROODFORGE_KEY_FILE` stands; "exactly one" credential variable here and in the companion.
- `engine-entry` is the one subcommand in `sh`, so it can `exec` into the ELF and the `comm` walk finds `claude`; `DISABLE_AUTOUPDATER=1` set in the spawn to honour the pin.
- Host setup commands live in `docs/reference/droplet-host-setup.md` once run; 0021's restore points there and rewrites the alternates file, which `git worktree repair` does not.

Amendments from the review of 03 to 05 (2026-10-14):

- `--mcp-config` names `<bf>/share/empty/mcp.json` (an empty map shipped with Broodforge), reached through the existing `<bf>` bind; no bind is added.
- The hook environment gains `BROODFORGE_CANONICAL` (the canonical checkout, for the read-class path check); the kept list is generated from the same array, so the companion's sudoers block follows.
- The systemd unit sets `KillMode=mixed` and the spawn starts each `sudo` child detached in its own process group, so `systemctl stop` and a terminal's Ctrl-C reach the bot alone and it interrupts turns through the signal path; a crashed bot's leftover engines are killed with the unit. On Windows a crash leaves engines running until they end: a job object is deferred.
- `credential add --kind subscription` runs `harness-create` and records the credential `needs-login`; the status is the store's only (an auth error sets it, `credential login <id> --done` or the relay's success clears it); the bot never reads the config directory to infer it.
- `credential init` refuses an existing key file; a scratch root ignores `BROODFORGE_KEY_FILE` and names its own key under the root.

## Amendments from the spike (2026-10-16)

Source: `../../findings/2026-10-15-spike-claims-for-adr-0022.md`, the PC rows, engine 2.1.296. The droplet rows and C37 stay open; the `linux-sudo-bwrap` backend and the Task Scheduler install wait on them, host-neutral code does not.

- C34 (PC half): the scrub hides `CLAUDE_CONFIG_DIR` from Bash, as the mitigation line in Decision says, and from the hook as well, while the engine itself runs against the directory; every `BROODFORGE_*` name and the engine's own `CLAUDE_PROJECT_DIR` reach the hook. The hook's credential-path deny therefore derives the config directory from `transcript_path` on stdin (the parent of the nearest ancestor named `projects`, the engine's layout for both credential kinds) and exits 2 when the field is absent, so the first real turn proves the field; no `BROODFORGE_*` name carries the directory, because a session's Bash would read it. The kept list is unchanged: the engine needs `CLAUDE_CONFIG_DIR` and strips it from its children itself. Recorded in `../03-hook-contract.md` (Environment, Paths). The key half waits on a funded key.
- C35 (PC half): the clone, commit and bundle steps of the companion hold with the identity in all four fields; the dubious-ownership half waits on the droplet.
- C36: `.claude.json` after a login holds account and organization identifiers, the account email, install and migration state, and a feature-flag cache that is most of the file; no key is named like a token or secret, and the credential is in `.credentials.json`. The plan's rule ("metadata lifts the exclusion") is not applied: the exclusion in Decision stays, for a different reason than the one it was written with. Every restore ends in a login, because `.credentials.json` is never in the set, and that login rewrites `.claude.json` (C25 saw it created in an empty directory); the file carries personal identifiers the backup set has no use for; and most of it churns daily, so a nightly copy is noise. `machineID` and `userID` were masked and unclassified; they change nothing, since the file is out either way. The listing was of the first account's file, not re-run for the owner's; the verdict rests on key names, which the same engine writes the same way, so it stands. The 0021 companion's "pending spike C36" resolves to this entry.
