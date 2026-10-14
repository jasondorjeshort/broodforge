# 0022. Session isolation and credential storage

Date: 2026-10-14
Status: promoted to ADR 0022

Revised 2026-10-14 after `0022-critic-review.md`, whose verification table is authoritative over
this body. Factual errors are corrected in place; the responses section says what each sentence
said before. Rejected options stay as the record.

## Question

How a channel's reach is enforced on each host (N-14), how the bot token, API keys and the
project git credential are stored (N-12, N-13), how a session's engine process receives its
credential (ADR 0020's two shapes), what a session's own tools can still read of it and of the
host (N-15), where a session's working tree lives, and as what user the bot and the engine run on
the droplet. Decided now because ADR 0020 fixed the process model (one `claude -p` process per
turn; the spawn call is the isolation wrapper) and ADR 0021 fixed the layout the credential store
and the working trees must fit. Coupled because the engine must read the credential while the
agent's Bash tool runs in the same process tree: what the session sees is decided by the
isolation mechanism, not the store. The owner wants a working first version on both hosts within
days; the recommendation names what each host enforces in that version and what is a stated gap.

## Constraints that matter here

- Both hosts from one code base, host differences only behind the named interfaces of
  `../06-platforms.md`; a weaker first-version reach is acceptable on Windows, not on Linux
  (N-01, `../01-requirements.md`, Tensions). Dependencies GPL-compatible and named (N-24).
- Reach is read-only or code in the first version (R-49 later); enforced by the host, not only by
  instructions (N-14); the proposal states per host what the session sees of the credential
  rather than claiming it is hidden (N-15); the owner's `~/.claude` on the PC is never touched.
- From ADR 0020: API key as `ANTHROPIC_API_KEY` in the engine's environment; subscription as a
  per-user `CLAUDE_CONFIG_DIR` the user logs into themselves, Broodforge never reading the token;
  the droplet's direct-message login relay off by default; no stored tokens of any kind (owner,
  2026-10-14); `--setting-sources user`, `--strict-mcp-config` and an explicit `--mcp-config` on
  every process; the `PreToolUse` hook is Broodforge's own CLI, run by the engine, deciding from
  its environment and writing nothing; non-root on the droplet; Windows interrupt is a tree kill.
- From ADR 0021: the instance root holds `config/`, `data/`, `backup/`, `credentials/`; every
  working-tree path is in the store and inside the backup set; a harness config directory inside
  the set is encrypted at rest or excluded by name and listed as unrestorable; the credential-store
  key lives outside the set and is a restore input; the store never holds a secret.
- The engine's own sandbox is unsupported on native Windows and wraps Bash only; the Bash tool,
  the hook and stdio MCP servers inherit the engine's environment unless
  `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` strips credential-like names from it (review, V5, V6);
  `.credentials.json` is a plain file the engine's user reads.
- The engine watches its settings files and reloads `permissions` and `hooks` while a process runs
  (review, V6; the original body said settings are read at process start, which was wrong). A
  settings file the session's user can write is a hook the session can drop mid-turn.
- git refuses to open, clone from or serve (`upload-pack`) a repository owned by another user
  unless `safe.directory` names it; a linked git worktree writes the main `.git` (review, V8).
- Every feature is driven from a terminal: `credential add|remove|list|login`, `session start`
  with a named reach, and a command showing what a session could read (N-05 to N-09, R-50).

## Options

Sub-choices: Linux isolation L1 to L6, Windows isolation W0 to W3, credential storage S1 to S3,
delivery D1 and D2, working-tree placement P1 and P2. Each at the same depth.

### L1. Two unix users on the droplet, filesystem permissions

`broodforge` runs the bot and owns the instance root (`0710`: the engine user traverses to its own
directories, lists nothing); `broodforge-engine` runs every engine process and owns
`harness/<credential id>/` (config dirs) and `sessions/<session id>/` (working trees), each `0750
broodforge-engine:broodforge-engine` with `broodforge` in that group so the bot reads session trees
and bundles and the backup job reads transcripts; `config/`, `data/`, `backup/`, `credentials/`
stay `0700 broodforge`; the canonical checkouts under `projects/<project id>/` are `0750
broodforge:broodforge-engine` so a session's clone can read their objects and refs and nothing
can write them. The spawn crosses users through sudoers, two `NOPASSWD` lines for `broodforge` as
`broodforge-engine`: `/usr/bin/bwrap` and `/usr/bin/kill`, with `env_keep` naming each variable
Broodforge sets (generated from the list in code and printed by the preflight), so no secret is
on a command line; every call is `sudo -n`. The alternative, `AmbientCapabilities=CAP_SETUID
CAP_SETGID` on the bot's unit and Node's `uid`/`gid` spawn options, is rejected because it lets
the bot become any user. Because `sessions/` is the engine user's, creating, bundling and deleting
`sessions/<id>` are engine-user operations through the same route (a Broodforge maintenance
entry run in the sandbox), which the worktree-removal interface, the 0021 removal procedure and
`scratch prune` call. Fail-open guard: a kept name missing from `env_keep` is dropped silently and
a dropped `CLAUDE_CONFIG_DIR` would send every session to one shared `~/.claude`, so the
in-sandbox entry asserts the credential variable is present (exit 1 naming it) and instance start
runs the spawn once with the entry's `--check` flag, refusing to start if any kept name is
missing and printing the sudoers lines expected. Meets: bot secrets, store and other projects
unreadable by any session by permission; the engine's `.credentials.json` unreadable by the bot,
so "never reads the token" holds by construction. Does not meet: sessions share a uid, so one
session's Bash reads another's config dir, its `/proc/<pid>/environ` and its tree. Cost: two
users, two sudoers lines and ownership rules in the operator doc. Who else: every daemon that
drops to a service user; CI runners.

### L2. bubblewrap around each engine process

`bwrap` (LGPL-2.1, packaged by Debian and Ubuntu) per engine process, as `broodforge-engine`:
`--ro-bind` the system directories, the engine binary, Node, Broodforge's install (the hook and
the entry need them) and the project's canonical checkout at its own path (the shared clone's
alternates point there); `--bind` the session's config dir, then `--ro-bind` its `settings.json`
over itself (bwrap binds single files), so the engine writes `.credentials.json` and `projects/`
but nothing inside can rewrite the hook or the deny rules mid-turn; `--bind
sessions/<id>/home` and, for code reach, `--bind sessions/<id>/tree`, for read-only reach
`--ro-bind` the same tree (writes return `EROFS`); `--tmpfs /tmp`, `--proc /proc`, `--dev /dev`;
`--unshare-pid --unshare-ipc --unshare-uts --die-with-parent --new-session`; `--setenv HOME
sessions/<id>/home --setenv PATH <fixed> --chdir <tree>`. No `--clearenv`: the original body had
`--clearenv` with `--setenv` for the credential, which put the key on bwrap's command line (in
`/proc/<pid>/cmdline` for every user and in sudo's log); sudoers `env_reset` with the `env_keep`
list is the whitelist and bwrap passes its environment through. The command is `broodforge
engine-entry -- claude -p ...`, the entry asserting its environment and exec'ing the engine.
Interrupt: bwrap installs no signal handler and `--die-with-parent` is `PR_SET_PDEATHSIG,
SIGKILL`, so a SIGINT to sudo or bwrap kills the engine without a `result`; instead the adapter
walks `pgrep -P` from the sudo child it spawned to the first descendant whose `comm` is `claude`
(host pid namespace) and sends `sudo -n -u broodforge-engine /usr/bin/kill -INT <pid>`; a kill of
the sudo child, which takes the tree down through `--die-with-parent`, is the fallback after a
timeout. Network shared, so a session reaches every listener on the host (constraint on 05:
no unauthenticated localhost HTTP while sessions run; a unix socket under `data/`, which the
engine user cannot traverse, or a token the session cannot read). Meets: with L1, one session
cannot see another's processes (fresh `/proc`), files (not bound) or environment; the canonical
checkout is read-only; the instance root is invisible; a `defer`-resumed process gets the same
binds. Does not meet: the session's own credential is readable by the engine's uid (mitigated by
L5 and the scrub); egress is open; no CPU or memory limit. Cost: one wrapper command built from
the reach; operator steps installing `bwrap`, `socat` and, on Ubuntu 24.04, an AppArmor profile;
the bind list is a living spec in the ADR's companion. Risk: Ubuntu 24.04 restricts unprivileged
user namespaces by AppArmor and ships no profile for `bwrap`; two documented profiles exist with
opposite consequences (L5 below), so the operator doc names one. Debian 12 needs none. Who else:
Flatpak, the engine's own Linux sandbox, CI sandboxes.

### L3. systemd-run transient unit per engine process

`systemd-run --uid=broodforge-engine --pipe --wait --collect -p ProtectSystem=strict -p
ProtectHome=yes -p PrivateTmp=yes -p ReadWritePaths=... -p InaccessiblePaths=<instance root> -p
ProtectProc=invisible -p MemoryMax=... claude ...`. Meets: the filesystem reach of L2 by
directives, a cgroup per turn so a kill takes the whole tree, `systemctl kill --signal=SIGINT` as
a clean interrupt, `MemoryMax=` and CPU caps as properties, `journalctl -u` for an agent, and no
AppArmor profile. Does not meet: `ProtectProc=invisible` hides other users' processes only, so
same-uid sessions still see each other's `environ` unless the droplet's systemd has
`PrivatePIDs=` (257; Ubuntu 24.04 ships 255, Debian 12 ships 252). Cost: a polkit rule or an
unrestricted `sudo systemd-run` lets the bot start a unit as any user, root included; only a
wrapper script as the sudo target bounds it; one more layer between the bot and the stream. The
natural second step is L3 outside L2 (cgroup, signal, limits from L3; binds and `/proc` from L2),
not a replacement. Who else: ad hoc service sandboxes on systemd hosts.

### L4. A container per session

Podman or Docker, one image per project holding the engine, Node, git and the project's toolchain;
a container per session with the tree and config dir mounted. Meets: the strongest boundary,
resource limits, and one day Utopia's tests (R-47). Does not meet the timeline: an image per
project, the engine pinned inside the image as well as on the host, rootless setup, the hook
running inside against a mounted Broodforge. Cost: days; a second place the engine version lives.
Risk: rootless networking and user-namespace mapping of mounted dirs. Who else: hosted agent
runners. Revived by R-47 or a reach the binds cannot express.

### L5. The engine's own sandbox setting and the subprocess scrub

Claude Code's `sandbox` settings key (bubblewrap and `socat` on Linux, seatbelt on macOS,
unsupported on native Windows) wraps the Bash tool only; `Read`, `Edit`, hooks and MCP servers run
outside it. `sandbox.credentials.files` with `"mode": "deny"` blocks a sandboxed read of a named
file, so the session's own `.credentials.json` is hidden from its shell while the engine reads it;
`allowUnsandboxedCommands: false` stops the model retrying a blocked command unsandboxed and
`failIfUnavailable: true` makes a missing sandbox an error rather than a silent downgrade. The
environment is a separate mechanism: `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` strips credential-like
names (`ANTHROPIC_API_KEY`, `CLAUDE_CONFIG_DIR`) from every subprocess the engine starts, on both
hosts, and on Linux runs Bash in its own pid namespace so `/proc` of the engine is unreadable.
Meets: the key and the credentials file hidden from `env` and `cat` in the session's shell (file
only on Linux); egress through an allowlisting proxy later. Does not meet alone: the engine's own
process still holds the credential and any process of the same uid can read its memory or, on
Windows, its environment; a mitigation, not a boundary, which is how N-15 is stated. The original
body said hiding the session's own credential was not achievable under any option; withdrawn.
Cost: lines in the Broodforge-written `settings.json`; the hook's environment names must not look
like credentials (no `TOKEN`, `KEY`, `SECRET`, `PASSWORD` in a `BROODFORGE_*` name) or the scrub
removes them. Risk: whether the inner bwrap starts inside L2 depends on the AppArmor profile: the
Claude Code-documented `/etc/apparmor.d/bwrap` profile (`flags=(unconfined)` with `userns`) lets
it mount, at the price of `/usr/bin/bwrap` becoming a user-namespace entry point for every account
on the box; the `bwrap-userns-restrict` extras profile stacks children into a confinement that
denies capabilities, so the nested sandbox cannot mount and L5 is dead. The profile attaches by
path, so the engine must use `/usr/bin/bwrap`. Spike-gated. Who else: Claude Code's own users.

### L6. Harness permission rules and the hook alone

Deny rules in the written `settings.json` (`Read` and `Edit` by absolute path pattern; `Write`
path rules are ignored by the engine; Bash by command text, which "is not a security boundary")
plus the ADR 0020 hook denying calls whose target lies outside the session tree (for Bash, a
heuristic on the command text). Meets: the instruction layer under every other option and all
that native Windows has today. Does not meet N-14 by itself: a Bash command is written around any
text match, and the model can be talked into it. Cost: nothing beyond the hook. Who else: every
Claude Code user without a sandbox.

### W0. The whole PC instance under a second standard Windows account

A local standard account `broodforge` (`net user broodforge <password> /add`, one admin-console
command; logged into once so the profile exists; Claude Code installed per user for it; Git and
Node machine-wide); the instance root, the key file (`%LOCALAPPDATA%\Broodforge\<instance>.key`
of that account) and the canonical checkouts in its profile; the bot started by a Task Scheduler
task as that account with "run whether user is logged on or not" (the task stores the password;
no interactive desktop needed). Engine and bot are one account, so a session's Bash reads the
bot's secrets, the key file, the store, the other sessions and can rewrite its `settings.json`
mid-turn, exactly as W2 admits; but the owner's `~/.claude`, the owner's checkouts and every other
project under the owner's profile are unreadable by default profile ACLs, which is the one thing
the vision names as never to be touched. Interrupt is `taskkill /T /F /PID` as the same account;
no per-turn credential plumbing, no job-object race, no file-tailed streams. The owner's own
development runs of Broodforge (tests, scratch) stay under the owner's account; only the installed
PC instance runs as `broodforge`. The debug transport (05) is reachable by a session on this host
because it is the same account; stated. Cost: a second account on the PC and a one-time login to
create its profile; the owner's interactive Claude Code and the bot no longer share checkouts
(they were not meant to). Risk: the "Log on as a batch job" right the task needs is granted when
the task is created with a stored password; if a policy removes it the task fails to start, and
the operator doc says where to look. Who else: Windows services under a service account.

### W1. A second local Windows account for the engine only, with a job object

`broodforge-engine`, a local standard account with "Allow log on locally", its own profile and
engine install, Git and Node machine-wide; `harness/` and `sessions/` carry explicit ACEs for it
and nothing else does. The bot spawns through PowerShell `Start-Process -Credential` with
`-RedirectStandardInput <message file> -RedirectStandardOutput <out.jsonl>
-RedirectStandardError`, workable because ADR 0020 closes stdin after one message; the adapter
tails the file. The account's password is an entry in S1. Process-tree cleanup by a job object: a
PowerShell helper (`Add-Type` P/Invoke: `CreateJobObject`, kill-on-close,
`AssignProcessToJobObject` on the engine's PID) holds the handle for the turn, so orphans that
`taskkill /T` misses die with it (V10). Meets: bot secrets, the store, the owner's `~/.claude` and
other projects unreadable by a session (ACLs); a reliable interrupt. Does not meet: sessions share
the account and see each other (L1 without L2). Cost: a day or two of plumbing (V9), a stored
password, file-tailed streams, a second profile on the owner's PC, a helper process per turn.
Risk, corrected after review: `CreateProcessWithLogonW` needs the target account granted access
to the caller's window station and desktop even for `WinSta0\Default`, starts the process in a new
window, and cannot be called from a LocalSystem service, so it is the shape that fails when the
bot runs from a scheduled task with no interactive desktop; `-Credential` needs the Secondary
Logon service (not re-verified); redirection to files, not pipes; the spawn-to-assignment race
(short: the engine spawns tools only after reading stdin). The step after W0, not the first
version. Who else: build systems that run steps as a service account.

### W2. The settings-sources rule, the hook and an explicit config dir alone

The engine runs as the account the bot runs as (the owner's); interrupt is `taskkill /T /F /PID`.
What holds: every process gets an explicit `CLAUDE_CONFIG_DIR` under the instance root,
`--setting-sources user` and `--strict-mcp-config`, so repository hooks never load and the owner's
`~/.claude` is never used; the hook denies calls outside reach; one tree per session; the
credential store is ciphertext. What does not hold, stated: a session's Bash runs as the owner, so
it can read the owner's `~/.claude` (the owner's own token), the store, the credential store and
its key file, every other session's config dir and tree, and every project; the only boundary is
the hook and the model's compliance (L6). Acceptable only because the PC serves the owner and a
few trusted people (`06-platforms.md`) and the requirements allow a weaker first-version Windows
reach. Cost: none. W0 removes "the owner's own token" and "every project" from this list for one
account and one login; W2 stays as what a scratch run under the owner's account is. Who else:
anyone running an agent under their own account.

### W3. WSL2 as the runtime on the PC

The bot and engine run inside a WSL2 distribution; L1 to L3 apply unchanged and one isolation
backend serves both hosts. Costs: the engine, Node, git and the checkouts move inside the WSL
filesystem (a checkout on `/mnt/c` is slow); the owner's interactive Windows Claude Code and the
bot stop sharing a filesystem view without care; the VM stops 60 s after the last process exits
unless kept alive (`vmIdleTimeout`), starting it from a scheduled task without an interactive
session is unverified, and Windows sleep suspending it is unverified (V16);
`claude-code-on-windows.md` stops describing the bot's host. This reinterprets "Windows hosting
is required" (`00-vision.md`). Rejected for the first version with the critic's reasoning
(review, Recommendation): the PC serves the owner and a few trusted people, the requirements allow
a weaker Windows reach, W0 gives the owner's account the protection the vision asked for, and a
VM lifecycle is a cost on a host whose job is to let the owner develop Broodforge while the
droplet does the real work. The owner can reopen it, since it changes what they install; the
trigger is wanting the droplet's reach on the PC.

### S1. One encrypted file, key outside the backup set (both hosts)

`credentials/store.json`: `{ version, entries: { <credential id>: { nonce, ciphertext, tag } } }`,
AES-256-GCM per entry with the id as additional data, Node's `crypto` (no dependency; verified by
run, V13), written to a temporary file and renamed into place, the bot the only writer; a reader
opens, reads and closes, never holding the file, because `fs.rename` on Windows fails with
`EPERM` while another handle holds the target. The key: 32 random bytes in a file whose path the
instance config names, default outside the instance root (Linux `/etc/broodforge/<instance>.key`,
`0400 broodforge`; Windows `%LOCALAPPDATA%\Broodforge\<instance>.key` of the account the bot runs
as), overridable by `BROODFORGE_KEY_FILE`; `credential init` writes it and prints the key once
for the owner's password manager; a missing key file refuses start naming the config key, exit 1.
Holds the Discord bot token, API keys, the project git credential (R-36) and, if W1 ever lands,
its account password as instance-owned entries; a subscription has no entry (nothing to store).
Scratch generates a throwaway key inside its root. Meets N-13 (the backup holds ciphertext),
parity, zero administration, the 0021 restore (step 5 is the key file). Weakness, stated: on the
host, whoever reads as the bot's user reads file and key; that is L1's job on Linux and nobody's
on Windows under W0. Hardening, deferred: `systemd-creds` (`LoadCredentialEncrypted=`, since
systemd 247, so both assumed distributions have it; TPM2 or host key) when the owner wants the key
off the filesystem; DPAPI-wrapping the Windows key file when W1 lands and the wrap buys something.
Who else: restic, age-based secret files.

### S2. The OS secret service

Linux: the D-Bus Secret Service needs a keyring daemon and a session bus unlocked by a user, which
a headless droplet lacks; `systemd-creds` is the headless cousin, listed under S1. Windows: DPAPI
(`ProtectedData`, user scope, built in and verified by run in Windows PowerShell 5.1; no Node
binding without a dependency, V11) ties the ciphertext to the Windows account, which under W0 or
W2 is the account the engine runs as, so it hides nothing from a session; Credential Manager
stores through `cmdkey` but reading back needs the `CredRead` API, so a third-party module or
native code (V11). Meets parity poorly: two backends from day one for no gain on the droplet.
Cost: a dependency per host. Who else: desktop apps.

### S3. Plain file with permissions

`credentials/store.json` in the clear at `0600`. Fails N-13 (the backup carries secrets in the
clear) and is named the weakest; it is S1 with the encryption removed and the same on-host
exposure. Not offered. Who else: many bots; the reason N-13 exists.

### D1. API key in the engine environment

`ANTHROPIC_API_KEY` on the engine process only, never on the bot's environment or a command line
(L2's `env_keep` route on Linux; Node's spawn `env` option on Windows). With the scrub (L5) it is
absent from the session's Bash, hooks and MCP servers on both hosts; readable by any process of the
engine's uid on Linux and of the bot's account on Windows. Revocation: the user's at Anthropic plus
`credential remove`.

### D2. Per-user config dir with the user's own login

`harness/<credential id>/` as `CLAUDE_CONFIG_DIR`. Broodforge writes `settings.json` (the hook, L5
and L6 rules) from its template before every engine process; because the engine reloads it on
change, it is bound read-only inside L2 so the session cannot drop the hook mid-turn (on Windows
it can; stated gap). The engine writes `.credentials.json`, `.claude.json` (login state, per the
docs) and `projects/` (transcripts, which R-18 will need). Backup: in the set, with
`.credentials.json` and `.claude.json` excluded by name in the nightly job and listed in the
restore procedure as "users log in again"; the spike reads what `.claude.json` holds after a
login, and the exclusion is lifted only if it holds no secret. Login: on a host with a shell,
`credential login <id>` prints the exact command (`CLAUDE_CONFIG_DIR=<path> claude auth login`,
prefixed on Linux with the run-as-engine wrapper); with the relay enabled in the instance config, a
Discord command starts the engine's login command as the engine user inside L2, reads the URL from
its output, sends it by direct message (Discord error `50007` is reported as "enable direct
messages from server members and retry"), waits up to a timeout for the code in reply, writes it to
stdin and reports the outcome; the code is never logged or stored. Audit: relay enabled (a config
change), login started, completed or failed, never the code. The headless-login claim and residual
risk are ADR 0020's C25. Every engine process also gets `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`,
`GIT_COMMITTER_NAME` and `GIT_COMMITTER_EMAIL` (the driver, from the registration record): the
author pair is honoured by the engine's Bash (verified by run, V15) and the committer pair is
required because the engine user has no git config and no `EMAIL`, so without it every droplet
commit dies with "unable to auto-detect email address".

### P1. A git worktree off the canonical checkout

`git worktree add` into `<checkout>/.worktrees/<session>` (inside the directory read-only reach
binds) or into `<instance root>/sessions/<session id>` (what ADR 0021 assumed as illustrative;
bindable alone). Either way the session's user needs write access to the canonical `.git` (V8),
so under L1 a session can move `main`, plant a `pre-push` hook or `core.hooksPath` that the bot's
push runs under the git credential, or write objects into the shared store, and read-only reach
cannot bind the checkout read-only. Cheapest in disk and time; the right tool for subagent
worktrees inside a session (R-31). Who else: the Hatchery practice on the owner's PC.

### P2. A per-session shared clone from the canonical checkout, handed back as a bundle

Corrected after review: a plain `git clone <canonical>` by the engine user dies on "dubious
ownership" (the canonical is bot-owned) and so does the bot's `git fetch <session tree>`, because
`upload-pack` refuses a repository owned by another user; lifting the latter with
`safe.directory` would make the bot's git parse a session-written config. The shape that runs:
the engine-side maintenance entry, in the sandbox with the canonical bound read-only at its own
path, runs `git clone --shared -c safe.directory=<canonical> <canonical> sessions/<id>/tree` on
a branch per session (`safe.directory` names the bot-owned canonical, the trusted direction; the
clone's `.git/objects/info/alternates` points at the read-only mount; new objects land in the
session's own `.git`). The bot sets `refs/sessions/<id>` in the canonical to the session's base
commit at start so its gc never prunes what a clone borrows. No hardlinks: `protected_hardlinks`
forbids them across users, and the original sentence "hardlinks where ownership allows" would
never have fired on the droplet. Read-only reach gets the same clone bound read-only, so git
reads work as the engine user and writes return `EROFS`. Handoff: the maintenance entry, inside
the sandbox, runs `git bundle create sessions/<id>/out.bundle <branch>`; the bot, as its own
user, runs `git -C <canonical> fetch sessions/<id>/out.bundle <branch>:refs/sessions/<id>` (a
bundle is objects and refs, no config, no hooks), checks every new commit (author equals the
driver, R-32 trailers present; otherwise it refuses the push and lists the commits with what is
missing, and 03 decides whether a `git interpret-trailers` rewrite is offered before the first
push), and pushes or opens the pull request from the canonical with the project git credential
from S1 in that one git process's environment (`GIT_ASKPASS` or `-c credential.helper='!...'`
reading the environment, never inlining the secret); the engine never sees it. The same bundle
route on Windows for parity. Subagent worktrees are `git worktree add` inside the clone, engine-
owned, so cherry-pick and merge into the session's branch work as R-31 asks. Removal is the 0021
procedure through the engine-side entry. Cost: a working-tree checkout per session (objects
shared, so seconds and the tree's size, not the history's); R-30 and the vision say "worktree"
in git's sense and this reads it as "its own working tree on its own branch, cheap, isolated,
with subagent worktrees inside", which keeps everything the owner asked for except a shared
`refs/`, and the shared `refs/` is the hole; the domain's Worktree record keeps its name. The idea
file's question is answered yes: a clone is preferred because of V8, not speed. Who else: CI
runners that clone per job; git's own `--shared` for throwaway clones.

## Recommendation

Linux: L1 with the two sudoers lines and `sudo -n`, L2 on every engine process with binds from the
reach, the `settings.json` ro-bind, no `--clearenv`, the SIGINT path through `kill` and the
`--die-with-parent` fallback, L6 under them, the scrub variable on every engine process, S1, D1
and D2 with the four git identity variables, P2 with the shared clone and the bundle handoff, the
relay off by default; L5 (`sandbox.enabled`, `credentials.files` deny for `.credentials.json`,
`allowUnsandboxedCommands: false`, `failIfUnavailable: true`) in the written `settings.json` once
the spike shows the inner bwrap starts under the named AppArmor profile, which the operator doc
names as the Claude Code-documented unconfined-userns profile for `/usr/bin/bwrap` on Ubuntu
24.04 (none needed on Debian 12). Enforced in the first version: no session reads the bot token,
the git credential, the store, the key file, another session's credential or tree, or another
project; no session rewrites its hook or deny rules mid-turn; read-only reach cannot write the
project; code reach writes only its own clone; the canonical checkout is never written by a
session and the bot never opens a session's repository. Stated gaps: a session's own credential
is hidden from its shell by engine mechanisms, not by a boundary (any process of the engine's uid
can read it); egress is open and a session reaches every listener on the host (05 is constrained
accordingly); no CPU or memory limit (L3 outside L2 is the next step if a runaway appears).

Windows: W0, S1, D1 and D2 with the scrub and the four identity variables, P2 by the same bundle
route, L6, the written `settings.json`, the relay off. Enforced: no repository hook or MCP server
loads; the owner's `~/.claude`, checkouts and profile are unreadable by a session; one tree per
session; the store and the bot token at rest are ciphertext. Stated gaps: everything a session's
Bash can read as the `broodforge` account (the key file, the store, the other sessions, its own
`.credentials.json`, the debug transport), and it can rewrite its `settings.json` mid-turn; reach
is the hook and the model. Next on Windows: W1 behind the spawn interface if its spike is clean.
W3 is rejected for the first version; the owner may reopen it.

First implementation step: S1 and `credential add|remove|list|login` with a scratch key; the spawn
interface with a same-account backend (Windows, and scratch on both hosts) and the Linux
`sudo -n bwrap` backend built from a reach with the entry's `--check` preflight; the P2 shared
clone, the bundle handoff and the attribution check; then the spike in the ADR's conditions table
on the droplet and the PC, recorded as a finding whose `promoted_to` is the ADR.

## What would change this recommendation

- The operator declines the AppArmor profile on Ubuntu 24.04, or unprivileged bwrap fails under
  it: L3 (systemd does its mounts as root, no user namespace) with a wrapper script as the sudo
  target, and L5 waits.
- The inner bwrap fails under the named profile: L5 is dropped, the scrub stays, and the N-15
  statement loses "and `cat`" on Linux.
- V8 refuted (a worktree needing no canonical `.git` write): P1 instead of P2.
- W1's spike is clean from a scheduled task (window-station grant, file-redirected streams, the
  job object): W1 after W0.
- The owner reopens W3: Windows gets the Linux backend; W0 and W1 are dropped.
- The owner requires egress control: `--unshare-net` with L5's proxy allowlist or L3 with
  `IPAddressDeny`, spike-gated.
- R-47 (project tests on the droplet) arrives: L4 is re-evaluated for the test runner.
- The owner wants the key off the filesystem: the droplet key moves to `systemd-creds`; the
  Windows key file stays.
- An owner requirement that a session must not read its own credential at all: the mechanisms
  here are mitigations; a boundary needs the engine's process and the Bash tool in different
  uids, which no option gives with the real engine; stated, not promised.

## Claims to verify

Results: the table in `0022-critic-review.md`, which corrects V1 (license LGPL-2.1; no profile in
the `apparmor` package), V6 (settings reload mid-run; `Write` path rules ignored), V8 (ownership
refusals on clone and fetch), V9 (window-station grant), V12 (since 247), and verified V11, V13
and V15 by run on the owner's PC. The claims as posed:

- V1. `bubblewrap` on Ubuntu 24.04 and Debian 12: package, license (LGPL-2.0-or-later); whether
  AppArmor restricts unprivileged user namespaces on Ubuntu 23.10 and later and whether the package
  ships a profile that permits `bwrap`; whether a setuid build exists. The flags in L2 exist and do
  what L2 says (`--unshare-pid` gives a fresh `/proc`; `--clearenv`, `--setenv`,
  `--die-with-parent`, `--new-session`). Ubuntu security docs, Debian package page, bwrap man page.
- V2. sudoers `env_keep` or `SETENV` carries variables set by the calling process into a `NOPASSWD`
  command run as another user without putting them on a command line; `sudo` relays SIGINT to its
  child; `/proc/<pid>/environ` is readable by the same uid only. sudoers and proc man pages.
- V3. Node `child_process.spawn` `uid`/`gid` need `CAP_SETUID`/`CAP_SETGID`; systemd
  `AmbientCapabilities=` grants them to a non-root unit. Node docs, systemd.exec.
- V4. `systemd-run` from a non-root user for another uid needs polkit authorization;
  `ProtectProc=invisible` hides other users' processes only; `PrivatePIDs=` from systemd 257;
  `--pipe` relays stdio; Ubuntu 24.04 ships systemd 255, Debian 12 ships 252.
- V5. Claude Code's sandbox: bubblewrap on Linux, unsupported on native Windows, configured in
  `settings.json`; wraps Bash only; can exclude a directory from reads; applies under `-p` with
  `--setting-sources user`; whether it runs inside an outer `bwrap`. code.claude.com, sandboxing.
- V6. Permission rules: `Read`, `Edit`, `Write` deny patterns by absolute path; Bash rules match
  the command string only; settings and hooks are read at process start, not mid-turn; with
  `CLAUDE_CONFIG_DIR` set, `.credentials.json` (`0600` on Linux, plain on Windows), `.claude.json`
  and `projects/` land there; `--setting-sources user` reads that `settings.json`. code.claude.com,
  permissions, hooks, settings; ADR 0020 C25 and C26.
- V7. A Discord bot can open a direct message with a user who shares a server; a user who disabled
  direct messages from server members makes the call fail with a distinct error. Discord API docs.
- V8. A linked worktree's commits write objects to the main `.git/objects`, its branch to the main
  `.git/refs/heads/`, its `HEAD` and index to `.git/worktrees/<name>/`; `git clone <local path>`
  hardlinks objects and copies when `link()` fails; `fs.protected_hardlinks=1` (Ubuntu and Debian
  default) forbids linking another user's files without write access; `--shared` is unsafe if the
  source prunes; NTFS hardlinks work on one volume; `git fetch <path> <branch>` needs read access
  to the clone only. git-worktree, gitrepository-layout, git-clone, kernel sysctl docs.
- V9. `Start-Process -Credential` uses `CreateProcessWithLogonW`, needs the Secondary Logon service
  and "Allow log on locally"; `-RedirectStandardInput/Output/Error` take file paths and work with
  `-Credential`; the engine's native installer installs per user. Microsoft and Claude Code docs.
- V10. Nested job objects since Windows 8; `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`;
  `AssignProcessToJobObject` on a running process; `taskkill /T` walks the live parent chain only.
- V11. `System.Security.Cryptography.ProtectedData` is available in Windows PowerShell 5.1; Node
  has no built-in DPAPI; `cmdkey` cannot read a secret back and reading needs `CredRead`; the
  license of any Node DPAPI or Credential Manager package considered.
- V12. `systemd-creds` and `LoadCredentialEncrypted=` from systemd 250; `$CREDENTIALS_DIRECTORY`
  readable by the unit's user only. systemd.exec, systemd-creds.
- V13. Node `crypto` AES-256-GCM with additional data; `fs.rename` replaces an existing file
  atomically on both hosts. Node docs.
- V14. A push can take a credential from `GIT_ASKPASS` or a one-shot `credential.helper` set on the
  git process, without a file or the command line. git-credential docs.
- V15. Claude Code honours `GIT_AUTHOR_NAME` and `GIT_AUTHOR_EMAIL` in the engine's environment for
  commits its Bash tool makes (plain git behaviour) and does not override them.
- V16. WSL2: systemd supported; the VM stops when the last process exits unless kept alive; Windows
  sleep suspends it; `/mnt/c` checkouts are slow relative to the WSL filesystem. Microsoft docs.

## Critic's objections and responses

1. **Blocking, V6 false: a session can drop the hook and the deny rules mid-turn.** Response:
   confirmed from the settings and hooks docs quoted in the review; the body said "settings are
   read at process start", which was wrong and is replaced in Constraints and D2. On Linux, L2
   now binds the config directory read-write and then `--ro-bind` its `settings.json` over itself
   (bwrap binds single files, verified in source), so the engine writes `.credentials.json` and
   `projects/` while nothing inside can rewrite the file; the engine's install and Broodforge's
   are ro-bound anyway, so the hook binary is fixed too. C26's inline `--settings` route stays as
   the alternative if the spike shows the watcher does not cover it. On Windows under W0 the
   session is the same account and can rewrite the file; stated in W0 and the recommendation as a
   gap. The spike's conditions table has "the session rewrites `settings.json` mid-turn" (C31).
   Resolution: accepted with change.

2. **Blocking, P2 does not run across users and `safe.directory` on the fetch is unsafe.**
   Response: confirmed from `clone.c`, `setup.c` and the git-clone text. P2 is rewritten: the
   engine-side entry runs `git clone --shared -c safe.directory=<canonical>` with the bot-owned
   canonical bound read-only at its own path (the trusted direction: the engine's git parses the
   bot's config, never the reverse); the bot keeps `refs/sessions/<id>` at the base so gc never
   prunes a borrowed object; the handoff is a bundle written inside the sandbox and fetched by the
   bot into `refs/sessions/<id>` in the canonical, so the bot never opens a session's repository
   and no session-written config or hook runs under the git credential. The hardlink sentence is
   dropped (`protected_hardlinks` makes it dead on the droplet). Read-only reach gets the same
   clone bound read-only, so the engine user's git works. The owner's "worktree per session" is
   satisfied by any per-session tree that is cheap, isolated and supports subagent worktrees with
   cherry-picking; the shared clone is all three (objects borrowed, tree checked out, `git
   worktree add` inside it), and what it gives up, a shared `refs/`, is the hole. Resolution:
   accepted with change; the critic's shape adopted.

3. **Serious, the API key on bwrap's command line.** Response: confirmed; `--clearenv --setenv
   ANTHROPIC_API_KEY` contradicted "no secret on a command line" and is removed. sudoers
   `env_reset` with the generated `env_keep` list is the whitelist, bwrap passes its environment
   through, and `--setenv` is kept for `HOME` and `PATH` only. The `--args <fd>` route is rejected
   because it needs `closefrom_override`. Resolution: accepted with change.

4. **Serious, the Linux interrupt is a kill.** Response: confirmed from bwrap's source (no
   handler; `PR_SET_PDEATHSIG, SIGKILL`). The signal path is designed rather than accepting
   kill-and-resume: a second sudoers line for `/usr/bin/kill`, the adapter walking `pgrep -P` from
   its sudo child to the first descendant whose `comm` is `claude`, and `sudo -n -u
   broodforge-engine /usr/bin/kill -INT <pid>`; the `--die-with-parent` kill stays as the
   fallback after a timeout, recorded as interrupted per ADR 0020. The in-sandbox FIFO supervisor
   is rejected as more code for the same result. Spike condition C32: SIGINT mid-turn yields a
   `result` with `modelUsage`. Resolution: accepted with change.

5. **Serious, V1 and V5 are decided by an unnamed operator choice.** Response: accepted. The
   operator doc names the Claude Code-documented unconfined-userns profile for `/usr/bin/bwrap`
   on Ubuntu 24.04, with its cost stated (bwrap becomes a user-namespace entry point for every
   account on a box that has only the two service accounts and the operator's); Debian 12 needs
   none; the engine must use `/usr/bin/bwrap` because the profile attaches by path; the spike runs
   one turn with L5 enabled under it (C33); `failIfUnavailable: true` is in the written settings.
   The reversal clause no longer names the sysctl-off route. Filed for the coordinator as an
   operator step by name. Resolution: accepted with change.

6. **Serious, "not achievable under any option" is wrong; two mechanisms do most of it.**
   Response: accepted; the claim is withdrawn. `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` goes on every
   engine process on both hosts; L5 is rewritten around `sandbox.credentials.files` deny,
   `allowUnsandboxedCommands: false` and `failIfUnavailable: true`; the hook's environment names
   must not look like credentials, and the spike checks the hook still receives them under the
   scrub (C34). The N-15 statement is rewritten: hidden from the session's shell and file tools
   by engine mechanisms, readable by the engine's own process and so by any process of the same
   uid; a mitigation, not a boundary. The "not achievable" bullet is replaced by one that says
   what a boundary would need. Resolution: accepted with change.

7. **Serious, the second Windows account that runs the whole instance was not compared.**
   Response: accepted; W0 is added and weighed against W1 and W2. It is chosen for the first
   version: it protects the one thing the vision names (the owner's `~/.claude`, checkouts and
   profile) with no per-turn plumbing, and W1's `CreateProcessWithLogonW` is the shape that fails
   from a scheduled task with no desktop, which is how the PC instance runs when the owner is not
   logged in. The cost (a second account, a one-time profile login, the engine installed for it) is
   an operator step; the owner's development runs stay under their own account. W1 is the step
   after W0; its risk text is corrected. Resolution: accepted with change.

8. **Serious, attribution is a default and the droplet's commits fail outright.** Response:
   accepted; the spawn sets all four `GIT_AUTHOR_*` and `GIT_COMMITTER_*` variables (both the
   driver; 04 may give the committer an instance identity later), and the bot checks every new
   commit on the fetched bundle (author equals the driver, R-32 trailers present), refusing the
   push and listing the commits otherwise; whether an automatic `git interpret-trailers` rewrite
   is offered before the first push is 03's. Spike condition C35 commits inside the sandbox.
   Resolution: accepted with change.

9. **Serious, the shared network namespace reaches the bot.** Response: accepted; L2 records the
   constraint on 05 (no unauthenticated localhost HTTP while sessions run on the host; on Linux a
   unix socket under `data/`, which the engine user cannot traverse, or a token the session cannot
   read; on Windows under W0 the transport is reachable by a session and the gap is stated).
   `--unshare-net` with a proxy stays the deferred egress step. Resolution: accepted.

10. **Serious, a sudoers mistake fails open to one shared login.** Response: accepted; every call
    is `sudo -n`; the in-sandbox entry (`broodforge engine-entry`) asserts `CLAUDE_CONFIG_DIR` or
    `ANTHROPIC_API_KEY` and exits 1 naming the missing variable; instance start runs the real
    spawn with the entry's `--check` flag and refuses to start if any kept name is missing,
    printing the sudoers lines expected. Resolution: accepted with change.

11. **Minor, L3's costs and merits understated.** Response: accepted; L3 now states the
    root-equivalence of an unwrapped polkit or sudo grant, the signal, cgroup and limit merits,
    and names L3 outside L2 as the second step rather than a replacement. Resolution: accepted.

12. **Minor, `.claude.json` holds login state and is not excluded.** Response: accepted; D2
    excludes it by name alongside `.credentials.json`, lists both as "users log in again", and
    the spike reads what it holds after a login (C36); the exclusion is lifted only if it holds
    no secret. Resolution: accepted with change.

13. **Minor, who creates and removes a session directory on Linux.** Response: accepted; L1 says
    creating, bundling and deleting `sessions/<id>` are engine-user operations through the same
    sudo and bwrap route, called by the worktree-removal interface, the 0021 removal procedure and
    `scratch prune`; scratch defaults to the same-account backend on both hosts. Resolution:
    accepted.

14. **Minor, `HOME` as the session tree pollutes the tree.** Response: accepted;
    `sessions/<id>/home` and `sessions/<id>/tree`, `HOME` the former, cwd the latter. Resolution:
    accepted.

15. **Minor, refusal messages.** Response: accepted in substance: the system-prompt append states
    the reach so an `EROFS` is expected, the hook pre-denies the obvious Bash forms, and the deny
    text should say that a new session is needed after the channel's reach changes. The wording
    itself is what users see, so the change to ADR 0020's deny text goes to 04 and the owner
    rather than being made here. Resolution: accepted with change (wording deferred to 04).

16. **Minor, housekeeping.** Response: accepted on every point: LGPL-2.1; `LoadCredentialEncrypted=`
    since 247 and the version trigger folded into an owner-wanted trigger; store readers open,
    read and close; the relay maps Discord `50007` to "enable direct messages from server members
    and retry"; the askpass or helper snippet reads the secret from that one git process's
    environment. Resolution: accepted.

WSL2 (W3): the critic recommends against it and this proposal agrees; recorded as rejected for
the first version, reopenable by the owner since it changes what they install.
