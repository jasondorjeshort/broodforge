# 0022 companion: the Linux spawn, the entry wrapper, the signal path, clone and handoff

Companion to `0022-session-isolation-and-credential-storage.md`; the ADR's Decision section points
here. This is the living spec: edit it when a path, a variable, a bind or the engine's needs
change, in the same commit, and keep `session reach-report` computed from the same list. The
commands that build a host to this table live in the operator entries until the owner has run
them, then in `../../reference/droplet-host-setup.md` with the run date as `verified:`
[no-check: written when the droplet operator entries are done]; the restore procedure in
`0021-backup-and-restore.md` points there.

## Users, ownership, sudoers

| Path | Owner and mode |
|---|---|
| `<root>` (instance root) | `broodforge:broodforge-engine` `0710` |
| `<root>/config`, `data`, `backup`, `credentials` | `broodforge:broodforge` `0700` |
| `<root>/settings`, holding one `<session id>.json` per session | `broodforge:broodforge-engine` `0750`; the files `0640` |
| `<root>/projects/<project id>` (canonical checkout) | `broodforge:broodforge-engine` `0750`, recursively |
| `<root>/harness`, `<root>/sessions` | `broodforge-engine:broodforge-engine` `0750` |
| `<root>/harness/<credential id>` (config dir) | `broodforge-engine:broodforge-engine` `0750` |
| `<root>/sessions/<session id>/{home,cfg,tree}` | `broodforge-engine:broodforge-engine` `0750` |
| `/etc/broodforge/<instance>.key` | `broodforge:broodforge` `0400` |
| `<engine>`, `<node>`, `<bf>` | `root:root` `0755`, recursively |

`broodforge` is a member of group `broodforge-engine` (reads trees, bundles and transcripts);
`broodforge-engine` is in no group of the bot's. The engine group on `<root>` is what lets the
engine user pass through the root to `harness/`, `sessions/`, `projects/` and `settings/`, and
`0710` keeps it from listing the root; the four `0700` directories are the ones it must not
enter. Every source in the bind list below must be traversable by `broodforge-engine`:
`spawn preflight` runs `namei -m` on each as that user and fails on the first that is not.
Sudoers, installed by the operator and printed by `spawn preflight`:

```
Defaults:broodforge env_reset
Defaults:broodforge env_keep += "<every name in the kept list, space separated>"
broodforge ALL=(broodforge-engine) NOPASSWD: /usr/bin/bwrap
broodforge ALL=(broodforge-engine) NOPASSWD: /usr/bin/kill
```

The kept list, generated from one array in code: `ANTHROPIC_API_KEY`, `CLAUDE_CONFIG_DIR`,
`CLAUDE_CODE_SUBPROCESS_ENV_SCRUB`, `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`,
`GIT_COMMITTER_EMAIL`, and every hook variable 03 names, `BROODFORGE_WORKFLOW` (the session's
workflow copy, ADR 0025 rule 5, set only when something was copied) among them. No name on the kept list other than the
two credential names contains `TOKEN`, `KEY`, `SECRET` or `PASSWORD`, so the scrub strips the
credential from the engine's subprocesses and nothing else; a bot-side name such as
`BROODFORGE_KEY_FILE` is never kept and is unaffected. Every call is `sudo -n`; a password prompt
is a refusal, never a hang.

## The engine spawn

Built from the session's reach by the `linux-sudo-bwrap` backend. `<cfg>` is
`<root>/harness/<credential id>`; `<sess>` is `<root>/sessions/<session id>`; `<settings>` is
`<root>/settings/<session id>.json`, written by the bot from the template by temp file and rename
before every process; `<ecfg>`, the directory the engine reads its user settings from, is `<cfg>`
for a subscription session and `<sess>/home/.claude` for an API-key session (no
`CLAUDE_CONFIG_DIR`, so the engine uses `$HOME/.claude`); `<canon>` is the project's canonical
checkout, `<engine>` the directory of the pinned ELF, `<bf>` Broodforge's install, `<node>` the
Node install. The block is one command. The lines from `--bind <cfg>` to the `CLAUDE.md` line
are omitted for an API-key session; `--bind <sess>/tree` becomes `--ro-bind` for read-only reach.

```
sudo -n -u broodforge-engine /usr/bin/bwrap \
  --ro-bind /usr /usr --symlink usr/lib /lib --symlink usr/lib64 /lib64 \
  --symlink usr/bin /bin --symlink usr/sbin /sbin \
  --ro-bind /etc /etc --tmpfs /etc/broodforge \
  --ro-bind-try /run/systemd/resolve /run/systemd/resolve \
  --ro-bind <engine> <engine> --ro-bind <node> <node> --ro-bind <bf> <bf> \
  --ro-bind <canon> <canon> \
  --bind <sess>/home <sess>/home \
  --bind <sess>/tree <sess>/tree \
  --bind <cfg> <cfg> \
  --bind <sess>/cfg/projects <cfg>/projects \
  --bind <sess>/cfg/shell-snapshots <cfg>/shell-snapshots \
  --bind <sess>/cfg/todos <cfg>/todos \
  --ro-bind <bf>/share/empty/dir <cfg>/agents \
  --ro-bind <bf>/share/empty/dir <cfg>/commands \
  --ro-bind <bf>/share/empty/dir <cfg>/skills \
  --ro-bind <bf>/share/empty/dir <cfg>/plugins \
  --ro-bind <bf>/share/empty/CLAUDE.md <cfg>/CLAUDE.md \
  --ro-bind <settings> <ecfg>/settings.json \
  --tmpfs /tmp --proc /proc --dev /dev \
  --unshare-pid --unshare-ipc --unshare-uts --die-with-parent --new-session \
  --setenv HOME <sess>/home --setenv PATH <engine>:<node>/bin:/usr/bin:/bin \
  --setenv DISABLE_AUTOUPDATER 1 \
  --chdir <sess>/tree \
  -- <bf>/bin/broodforge engine-entry -- <engine>/claude -p ...
```

The trailing `...` is ADR 0020's flag list. Rules: no `--clearenv`, no `--setenv` of any secret;
the credential and the git identity reach the process through `env_keep`. A bind over a path
inside an earlier bind comes after it, which is why `<settings>` is last among the binds; bwrap
creates a missing mount point itself (a directory or an empty file, as the engine user, on the
writable bind beneath), so `<cfg>/projects` or `<sess>/home/.claude/settings.json` need not
exist before the spawn. Nothing under `<root>` is bound except the lines above, so `config/`,
`data/`, `backup/`, `credentials/`, every other `harness/`, `sessions/` and `settings/` entry and
every other project are absent inside. `/etc` is bound read-only in full for name resolution and
certificates, with `/etc/broodforge` hidden by the tmpfs over it; `/run/systemd/resolve` is bound
because Ubuntu 24.04 points `/etc/resolv.conf` into it (`-try`, since Debian without
systemd-resolved has no such directory). `DISABLE_AUTOUPDATER` keeps the pinned ELF from trying to
replace itself. The `defer`-resumed process (ADR 0020) is spawned with the same command.

Config directory. Sessions sharing a credential share `<cfg>` read-write, because the engine
refreshes `.credentials.json` and rewrites `.claude.json` there and the bot does not depend on how
(in place or by rename). Two lists in code neutralise the rest. The per-session list is every
directory the engine writes during a turn that the credential does not need, bound from
`<sess>/cfg/<name>`: transcripts (`projects/`), shell snapshots (sourced by the Bash tool) and
todos are the session's own. The loaded list is every name the engine reads instructions or code
from, bound read-only from `<bf>/share/empty/` (an empty `dir/` and an empty `CLAUDE.md`,
shipped by Broodforge), so no session can plant a memory file, agent, command, skill or plugin
for a later session of the same user. C38 completes both lists on the pinned engine: a name the
spike finds written goes to the first, a name it finds read goes to the second. What stays shared
is the credential files and whatever neither list names.

Operator prerequisites: `bubblewrap` and `socat` installed; on Ubuntu 24.04 the Claude
Code-documented `/etc/apparmor.d/bwrap` profile (`flags=(unconfined)` with `userns`, attached to
`/usr/bin/bwrap`); the engine as the native ELF, Node and Broodforge installed at paths both users
can read.

## `broodforge engine-entry`

`<bf>/bin/broodforge` is a POSIX `sh` script that dispatches `engine-entry` to a shell
implementation and every other subcommand to Node, because the entry wrapper must `exec` into the
engine, which Node cannot, so that the engine is the process directly below the inner `bwrap`.
It runs as `broodforge-engine` inside the sandbox. Checks: exactly one of `CLAUDE_CONFIG_DIR` and
`ANTHROPIC_API_KEY` is set, `HOME` is set and writable, cwd is writable for code reach; on failure
it prints `engine-entry: <VARIABLE> missing; check sudoers env_keep (spawn preflight prints it)`
to stderr, exit 1, nothing spawned. `--check` prints every kept name it sees, one per line, and
exits 0; the bot runs the full spawn with `--check` at instance start (`spawn preflight` runs it
on demand) and refuses to start if a kept name is absent, printing the sudoers block above.

Maintenance commands run as the Broodforge binary through the same `sudo -n -u broodforge-engine /usr/bin/bwrap`
route, each in a sandbox with the system and Broodforge binds above, no config directory,
no `<settings>`, and `HOME` set to `/tmp`:

- `engine-entry harness-create <cfg>` and `harness-remove <cfg>` bind `<root>/harness`
  read-write and create (`0750`) or delete the config directory; `credential add` and
  `credential remove` run them for a subscription credential, and the login relay runs only
  after `harness-create`.
- `engine-entry worktree-create <sess> <canon> <branch> <base>` binds `<root>/sessions`
  read-write (the session directory does not exist yet, and bwrap refuses a missing source) and
  `<canon>` read-only; it creates `<sess>` with `home/`, `home/.claude/`, `cfg/<name>` for every
  name on the per-session list, then runs the clone below.
- `engine-entry worktree-bundle <sess> <branch>` binds `<sess>` read-write and `<canon>` read-only.
- `engine-entry worktree-remove <sess>` binds `<root>/sessions` read-write (a directory that is
  itself the mount point cannot be removed from inside) and deletes `<sess>`. This is the Linux
  implementation of the worktree-removal interface (ADR 0021).

## Interrupt

1. The adapter holds the pid of the `sudo` child it spawned.
2. Walk the host process tree: `pgrep -P <pid>` repeatedly, taking the first descendant whose
   `/proc/<pid>/comm` is `claude` (the chain is `sudo`, `bwrap`, `bwrap` as namespace pid 1,
   `claude`; the entry wrapper exec'd into the engine). The name is `claude` because the engine
   is the native ELF; the npm shim would show `node`, which is one reason ADR 0020 names the ELF.
3. `sudo -n -u broodforge-engine /usr/bin/kill -INT <pid>`; wait for the `result` (ADR 0020).
4. After the interrupt timeout (instance config; default 20 s): find the outer `bwrap` process
   (the `sudo` pid's child, from `pgrep -P <sudo pid>`) and run `sudo -n -u broodforge-engine
   /usr/bin/kill -KILL <bwrap pid>` through the sudoers kill line, never against the sudo pid;
   sudo then exits on its own, `--die-with-parent` kills the tree, and the turn is recorded as
   interrupted per ADR 0020.

Never signal `sudo` or `bwrap` with SIGINT expecting a `result`; both end in a SIGKILL of the engine.

## Clone at session start

0. The bot, in `<canon>`, before the workflow copy (ADR 0025 rule 5) and the clone: `git -c
   credential.helper= fetch --quiet --no-tags --no-write-fetch-head <remote> +refs/heads/<default
   branch>:refs/remotes/<remote>/<default branch>` (as the handoff's fetch runs), `<remote>` and the
   branch the project's `repo.remote` and `repo.defaultBranch`, with the push's `GIT_ASKPASS` helper
   and, when the project names `repo.gitCredentialId`, its value on that one process (step 4 below),
   nothing else fetched, a deadline of thirty seconds. Then the two branches are compared
   (`merge-base --is-ancestor` both ways): the local branch behind the fetched ref is fast-forwarded,
   `git merge --ff-only` while HEAD is that branch, so the checkout stays coherent, else `git
   update-ref <ref> <new> <old>`; equal is nothing; the local branch ahead (it contains the remote's
   commit, as when the owner or a script commits in the canonical without pushing) is left as it is
   and the session starts from it, the `session-started` row carrying `canonical` `ahead <n>`, the
   count of local commits past the remote's, with no venue line; diverged (neither contains the
   other) refuses the start, `canonical-diverged`, nothing cloned, the row ended `refused:
   canonical-diverged`; the owner reconciles the canonical by hand. A fetch that fails or runs past
   its deadline (offline, a credential refused, a `repo.gitCredentialId` the store lacks, which skips
   the fetch with the reason `credential-missing`, a fast-forward refused by a dirty checkout) does
   not refuse: the session starts from the local branch, the row carries `canonical` `stale: <git's
   line or the reason>` (`fetched` otherwise), the venue's first post says so (`../04-ux.md`,
   `canonical-stale`, the owner's wording) and `status --json` shows the last fetch per project.
   The canonical is never reset or cleaned, and the bot's git still never opens a session's tree.
1. The bot, in `<canon>`: `git update-ref refs/sessions/<id> <base commit>`; creates
   `<sess>` through `worktree-create`, which runs as the engine user:
2. `git -c safe.directory=<canon> clone --shared --branch <base branch> <canon> <sess>/tree`,
   then `git -C <sess>/tree checkout -b <session branch>`. The `-c` goes before `clone`: there it
   is command-scope configuration, reaches git's own subprocesses through `GIT_CONFIG_PARAMETERS`
   and counts as protected for `safe.directory`; `clone -c key=value` instead writes the new
   clone's local config, which git ignores for `safe.directory`. The exception names the bot-owned
   canonical only; the clone's `.git` is engine-owned and needs none.
3. Record the tree path in the store (ADR 0021). Read-only reach: the same clone, bound read-only.

The canonical's objects are borrowed through `.git/objects/info/alternates`, an absolute path;
the bot never runs `git gc --prune` with a date that could remove objects reachable from
`refs/sessions/*`, and `refs/sessions/<id>` is deleted only after the session's tree is removed.

## Handoff at session end (and `worktree handoff <id>`)

1. `worktree-bundle` (engine user, in the sandbox): `git -C <sess>/tree bundle create
   <sess>/out.bundle <base commit>..<session branch>`; a session with no commits writes nothing
   and reports so.
2. The bot, in `<canon>`: `git bundle verify <sess>/out.bundle`, then `git fetch
   <sess>/out.bundle <session branch>:refs/sessions/<id>`.
3. Attribution check on `refs/sessions/<id>` since the base: every commit's author name and email
   equal the driver's registration; every commit carries the R-32 trailers 03 specifies. A
   failing commit is listed in the channel with what is missing and the push is refused.
4. Push or pull request from `<canon>`: `git push origin refs/sessions/<id>:refs/heads/<session
   branch>` with `GIT_ASKPASS` set to a Broodforge helper that reads the credential from that one
   git process's environment (`BROODFORGE_GIT_CREDENTIAL`, set only on this spawn, never kept);
   audit `attempted` before, `done` or `failed` after (ADR 0021).
5. Removal per R-35 through `worktree-remove`; then `git update-ref -d refs/sessions/<id>`.

The bot never runs git with `<sess>/tree` as the repository: no `fetch <path>`, no `-C <sess>`,
no `safe.directory` for anything engine-owned. On Windows the same five steps run with the
`same-account` backend (no sudo, no bwrap), so the code path is one.

## Amendments

- 2026-10-14, review of the spike harness: Interrupt step 4 sent SIGKILL to the `sudo` process,
  but sudo runs as full root, so the bot's user cannot signal it and the kill fails with EPERM.
  Step 4 now targets the outer `bwrap` process through the sudoers kill line. Spike claim C32
  checks it: `sudo -n -u broodforge-engine sleep 60 & kill -KILL $!` (expected EPERM), then the
  bwrap-pid form (expected to succeed).
- 2026-10-16, ADR 0023 (`0023-language-for-the-long-term.md`): Broodforge is one static Go
  binary, so the runtime bind and the shell wrapper go. The sentence "`<bf>/bin/broodforge` is a
  POSIX `sh` script that dispatches `engine-entry` to a shell implementation and every other
  subcommand to Node, because the entry wrapper must `exec` into the engine, which Node cannot"
  is replaced: `<bf>/bin/broodforge` is the binary, `engine-entry` is a subcommand of it that
  runs its checks and `syscall.Exec`s the engine, so the engine is still the process directly
  below the inner `bwrap` and the `comm` walk still finds `claude`. In the spawn block the line
  `--ro-bind <node> <node>` is dropped and `PATH` becomes `<engine>:/usr/bin:/bin`; in the
  ownership table the row `<engine>, <node>, <bf>` loses `<node>`; "Maintenance commands run as
  Node" reads "run as the binary"; the operator prerequisite "the engine as the native ELF, Node
  and Broodforge installed" loses Node. `<bf>` holds the binary and `share/`; the droplet builds
  it from the pinned source (ADR 0023). The Windows gap "a crashed bot's leftover engines run on
  until they end" in the 0022 ADR's 03-review amendments is closed by ADR 0023's job object.
- 2026-10-16, source the P7 review (`../08-implementation-plan.md`, P7): "no `CLAUDE_CONFIG_DIR`"
  for an API-key session (The engine spawn, `<ecfg>`) is this backend's rule, and "exactly one of
  `CLAUDE_CONFIG_DIR` and `ANTHROPIC_API_KEY`" is `engine-entry`'s check on Linux. The same-account
  backend sets both for an API key, `CLAUDE_CONFIG_DIR` naming `<sess>/home/.claude`, because on
  the same account the engine would otherwise read the owner's `~/.claude`; `<ecfg>` is the same
  directory either way.
- 2026-10-30, the idea file of 2026-10-24 on a session starting from a stale canonical and the
  2026-10-29 triage: nothing fetched the remote into the canonical, so a session branched from
  whatever the default branch was last set to (on the live root, four commits behind, then twelve).
  Clone at session start gains step 0 above, written in place as this living spec allows: the
  fetch to the remote-tracking ref, the fast-forward (`merge --ff-only` while checked out,
  `update-ref` otherwise; not `fetch --update-head-ok`, which leaves the checkout incoherent, and
  not `pull`, which needs an upstream and runs a merge), the ahead case left alone (review of
  2026-10-30: `tests/repro/full-session.sh` commits the workflow fixture into the canonical without
  pushing, and one local commit of the owner's must not stop every session), the diverged refusal
  and the stale start.
  The fetch comes before the copy, so a session's rules are the remote's too. Built by P30
  (`../08-implementation-plan-canonical.md`); the venue line and refuse-or-warn on divergence are
  the owner's, pending in `../04-ux.md`.
