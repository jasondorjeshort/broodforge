# Critic review of 0021: storage

Date: 2026-10-14
Reviewer: critic (Fable 5.1)
Proposal: `0021-storage.md`, status under review

## Verdict in one line

The engine choice (SQLite, WAL, one file) is right and I would not reopen it; the mechanisms
around it (reconciliation, the audit trigger, config refusal, scratch cleanup, the restore
procedure) each fail closed without a documented lift, and two silences in the proposal quietly
constrain 0022. Six serious objections, nine minor, none blocking.

## Claims verified

| Claim | Result | Source |
|---|---|---|
| SQLite is public domain | Verified: "All of the code and documentation in SQLite has been dedicated to the public domain by the authors." | sqlite.org/copyright.html |
| better-sqlite3 MIT, prebuilt for win32-x64 and linux-x64 | Verified, and stronger than claimed: v13.0.3 (2026-08-05) ships `prebuilds/win32-x64.node` and `prebuilds/linux-x64.node` inside the npm tarball (Node-API, `engines.node >=22`); the v13.0.0 notes say `prebuild-install` was removed and "prebuilt binaries are published directly with the better-sqlite3 code itself". v12.12.0 still used GitHub release assets per Node ABI. | npm tarball listing; github.com/WiseLibs/better-sqlite3 releases v13.0.0, v12.12.0 |
| `node:sqlite` stable, not behind a flag, in a current LTS | Partly wrong. No flag since 22.13.0, but the module is "Stability 1.2, Release candidate" in Node 24 (from 24.15.0) and Node 26, and "Stability 1.1, Active development" in Node 22. On this PC (Node 22.23.2) `require("node:sqlite")` prints `ExperimentalWarning: SQLite is an experimental feature and might change at any time`. Node 24 and 22 are the LTS lines today; 26 is Current. | nodejs.org/docs/latest-v24.x and latest-v22.x api/sqlite.html; nodejs.org/en/about/previous-releases; observed locally |
| Python `sqlite3` PSF, GPL-compatible | Verified: PSF License v2; Python 2.2 and later marked GPL-compatible in the table. | docs.python.org/3/license.html |
| `modernc.org/sqlite` BSD-3, cgo-free, passes the SQLite test suite | License and cgo-free verified ("Pure-Go SQLite, no cgo"; windows/amd64 and linux/amd64 listed at SQLite 3.53.4). The test-suite claim I could not find on the README, the project page or pkg.go.dev; treat as unverified. | pkg.go.dev/modernc.org/sqlite; gitlab.com/cznic/sqlite |
| `mattn/go-sqlite3` MIT, needs cgo and gcc on Windows | Verified: "you are required to set the environment variable CGO_ENABLED=1"; "To compile this package on Windows, you must have the gcc compiler installed." | github.com/mattn/go-sqlite3 README |
| `rusqlite` MIT, `bundled` compiles SQLite in | Verified. | github.com/rusqlite/rusqlite README |
| `Microsoft.Data.Sqlite` MIT | Verified; natives via `SQLitePCLRaw.bundle_e_sqlite3`. | nuget.org |
| `sqlite-jdbc` Apache-2.0 | Verified (Apache-2.0 plus a BSD-2 notice); natives for Windows and Linux in one JAR. | github.com/xerial/sqlite-jdbc |
| PostgreSQL License permissive | Verified: "similar to the BSD or MIT licenses". | postgresql.org/about/licence |
| WAL: readers and writer do not block each other; not on network filesystems | Verified: "readers do not block writers and a writer does not block readers"; "WAL does not work over a network filesystem". Also found: a long-running reader can starve checkpoints and the WAL grows without bound (bears on objection 13). | sqlite.org/wal.html |
| A plain copy of a live WAL database can be inconsistent | Verified: the WAL file "is part of the persistent state of the database"; copying mid-transaction "might contain some old and some new content, and thus be corrupt". | sqlite.org/wal.html; sqlite.org/howtocorrupt.html s1.2 |
| `VACUUM INTO` and the backup API give a consistent copy | Verified: VACUUM INTO output is "a consistent snapshot of the original database" but "if interrupted by an unplanned shutdown or power loss ... might be incomplete and corrupt"; the backup API restarts from the beginning when another process writes mid-copy. | sqlite.org/lang_vacuum.html; sqlite.org/backup.html |
| `synchronous=FULL` durable on power loss in WAL; NORMAL may lose transactions, not corrupt | Verified: "FULL is atomic, consistent, isolated, and durable (ACID) in WAL mode"; "WAL mode is safe from corruption with synchronous=NORMAL" and "might roll back following a power loss". | sqlite.org/pragma.html#pragma_synchronous |
| Triggers with RAISE(ABORT) refuse UPDATE and DELETE; no finer permission exists | First half verified (returns SQLITE_CONSTRAINT with the given message). Second half is false: `sqlite3_set_authorizer` is invoked at statement compile time with action codes `SQLITE_UPDATE` (table name, column name), `SQLITE_DELETE` (table name) and `SQLITE_DROP_TRIGGER` (trigger, table), and can return SQLITE_DENY; it is per connection. `PRAGMA query_only` makes a connection read-only. Neither survives a different connection, and neither does the trigger (DROP TRIGGER). | sqlite.org/lang_createtrigger.html; sqlite.org/c3ref/set_authorizer.html; sqlite.org/c3ref/c_alter_table.html |
| `sqlite3` shell downloadable for Windows, packaged on Debian and Ubuntu | Verified: `sqlite-tools-win-x64-3540000.zip`; Debian trixie `sqlite3` 3.46.1. Not installed on this PC today (`sqlite3: command not found`); Python 3.10 here links SQLite 3.37.2, Node 22 bundles its own. | sqlite.org/download.html; packages.debian.org; observed |
| Windows Defender real-time scanning slows SQLite writes; directory exclusion is the fix | Not verified for SQLite specifically. General evidence only: Defender scans on open and after close, vendors of file-heavy software recommend exclusions. Plausible, unproven; see objection 12 for why the fix as stated is wrong anyway. | learn.microsoft.com answers; docs.hexagon.com; squeakfoundation vm-dev 2021-12 |
| Atomic replace on Windows needs MoveFileEx(REPLACE_EXISTING) or ReplaceFile plus a flush; Linux rename is atomic | Verified for the Windows half: `MOVEFILE_REPLACE_EXISTING` documented; ReplaceFile documents three partial-failure states and `REPLACEFILE_WRITE_THROUGH` "is not supported". Linux rename atomicity is POSIX and not in dispute. Note that every mainstream runtime already wraps MoveFileEx (bears on the case for B). | learn.microsoft.com MoveFileExW, ReplaceFileA |
| A crash mid-append can leave a partial last line | Verified: "a successful write() may transfer fewer than count bytes" and "does not make any guarantee that data has been committed to disk". | man7.org write(2) |
| PostgreSQL on Windows is an installer plus a service; a live data-directory copy is not a valid backup | Verified: EDB installer, PostgreSQL "runs as a service in the background", asks for a service-account password; "The database server must be shut down in order to get a usable backup." | postgresql.org/download/windows; enterprisedb.com docs; postgresql.org/docs/current/backup-file.html |
| Claude Code stores session state as files, not SQLite | Verified by observation: `~/.claude/projects/<cwd-slug>/<session-uuid>.jsonl`, `history.jsonl`, `sessions/`, no `.sqlite` or `.db` under `~/.claude`. Also observed: `~/.claude/.credentials.json` exists in the clear on Windows (bears on objection 4). | observed on this PC, Claude Code 2.1.x |
| `git worktree repair` exists for moved or copied trees | Verified (bears on objection 5). | git-scm.com/docs/git-worktree |

## Objections

### 1. Start-up reconciliation marks a worktree "removed" from a path check alone, with no lift (serious)

The proposal: "a worktree record whose path is missing is marked removed with an audit event." A
path is missing after a restore that did not include worktrees (the proposal itself says the
backup may miss them), after a drive or mount is absent at boot, after a Windows path mismatch
(case, `\\?\` prefix, junction target gone while the junction remains), or after a user moved the
directory. In each case a worktree in the state "kept with unpushed commits" (R-35, the one state
whose whole purpose is to warn the owner) is silently converted to "removed", and the only trace
is an audit row. R-17 says an interrupted session's worktree "is left intact"; the record of it
should be too. Who can lift it: nobody. There is no "un-remove", no "adopt this directory as
worktree X", and the audit log is append-only so the owner cannot even annotate it.
Resolution: a distinct state `missing` that reconciliation sets and that a later start-up clears
when the path reappears; `removed` only ever set by the removal procedure (R-35) or by an explicit
owner command (`worktree forget <id>`) that writes its own audit event; the start-up report names
every missing worktree and whether it had unpushed commits; a `worktree adopt <path>` command for
the directory-with-no-record case. Each refusal line says which command lifts it.

### 2. The append-only audit trigger has no documented lift, no retention, and does not defend against the stated threat (serious)

Who can lift it: anyone with write access to `store.sqlite`, with `DROP TRIGGER` from the
`sqlite3` shell or any binding, from the same host, while the bot runs (WAL admits a second
writer; the bot's prepared statements re-prepare on SQLITE_SCHEMA). The proposal does not say
this, does not specify the refusal text (RAISE(ABORT, msg) returns whatever message we write; it
should name the rule and the lift), and gives no retention story for a table that grows forever
alongside `usage`. More important: the vision's threat is a tricked agent on the host, and on the
Windows first version 0022 may run the harness as the owner's own user, which can open the file,
drop the trigger, truncate the table, or delete `data/` and `backup/` outright, since the snapshot
lives on the same disk. The trigger is a guard against Broodforge's own bugs and nothing more;
the proposal presents it as the append-only property N-17 asks for.
Resolution: (a) say in the proposal what the trigger protects against and what it does not;
(b) make the property verifiable rather than trusted: each audit row carries a hash of the
previous row (chain), `backup verify` and `audit verify` check the chain and the owner can
compare the exported `audit.jsonl` against an off-box copy; (c) define the lift: trigger changes
only through a numbered migration, never by hand, and the refusal message says so; (d) design the
audit writer as a sink interface (table authoritative, optional line-per-event file sink) so 0022
can point a sink at an OS-level append-only target on the droplet (a root-owned file with
`chattr +a`, or a journald unit) without touching the store; (e) a retention rule, even if it is
"none in the first version, revisit at N rows", written down.

### 3. Configuration that refuses to load leaves the bot down after an unattended restart (serious)

P1: "validation errors refuse the start or reload with one line per error, and nothing
half-loads." Reload is fine: the old config stays active and the owner is at the terminal. Start
is not: the owner edits `config/` while the bot runs, saves a file with an error, does not reload,
and the PC reboots at 3 am (06-platforms: the PC "may sleep or reboot") or the droplet is
restarted by its provider. The bot refuses to start and stays down until a person reads a log.
The brief asks whether "config in files" survives the owner editing while the bot runs; the
answer today is: until the next restart. Who can lift the refusal: the owner, by fixing the file;
but the refusal is on a host nobody is watching. Also unstated: whether a reference from config to
a credential name that 0022's store does not hold is a validation error (it must be, and the
message must name the credential), and what a running session does when a reload changes its
channel's reach or role lists (probably "sessions keep what they started with", which means the
session row must copy reach and mode at start; the domain model holds mode but not reach).
Resolution: on start-up with an invalid `config/`, load the last-good configuration (the store
already records the hash of every loaded config; keep the last-good copy under
`data/config-last-good/` or in the store) and post a loud `configuration invalid, running on
last-good from <date>` audit event and channel message; add a `config check` command the owner
runs after editing; state the cross-store validation and the running-session rule.

### 4. The proposal is silent on harness state, and that silence decides 0022 and breaks N-20 or N-13 (serious)

"Nothing of Broodforge's is written outside the root (N-20); SQLite's own temporary files ... are
not state." But the harness writes session transcripts under `~/.claude/projects/<worktree
slug>/<uuid>.jsonl` (observed), and those are the only thing R-18 (resume, later) can resume
from. On the Windows PC that directory is the owner's own `~/.claude`, shared with their own
sessions (06-platforms). Either transcripts are state, in which case the backup set must include
a directory the proposal does not name and the harness must be pointed at it; or they are not,
in which case R-18 is being decided here by omission. Worse in the other direction: if 0022 picks
`CLAUDE_CONFIG_DIR` per credential and places those directories under the instance root
`credentials/`, the nightly copy picks up Claude Code's own `.credentials.json`, which exists in
the clear on Windows (observed), violating N-13 unless 0022 encrypts the whole directory, which
the harness cannot read. The proposal hands 0022 a layout (`credentials/` under the root, backed
up) without stating the constraint it imposes.
Resolution: one paragraph that names harness-owned state (config dir, transcripts, Claude
Code's own credentials file) as a category; requires 0022 to say for its chosen delivery
mechanism where each lands relative to the backup set; and states the rule 0021 imposes: a
harness directory inside the backup set is either encrypted at rest by 0022 or excluded by name
and listed as unrestorable in the restore procedure.

### 5. "Copy the tree back, start" does not rebuild a host from nothing (serious)

The brief asks whether the backup plus config files let the owner rebuild a host from nothing.
What the written procedure omits: (a) the credential-store key, which N-16 and 0022's leading
option keep outside the backup set, so the restore has a step the proposal does not name; (b)
the Discord bot token and the harness login, both in 0022's store, same gap; (c) git worktree
links: a worktree's `.git` file and the repository's `.git/worktrees/<name>/gitdir` hold absolute
paths, so a restore to a different path or user (a rebuilt droplet) breaks every worktree until
`git worktree repair` runs, and reconciliation would then mark them all missing (objection 1);
(d) the service (Task Scheduler or systemd) and the runtime, which are operator steps the
procedure should at least point to; (e) the snapshot itself: `VACUUM INTO` output "might be
incomplete and corrupt" if interrupted, and the nightly copy can run while the snapshot is being
written, so the snapshot must be written to a temporary name and renamed, with at least two
generations kept; (f) `backup verify` opens the snapshot and counts rows, which proves the file
is readable, not that a restore works; N-19 asks for a rehearsed restore. Also the snapshot
lives inside the instance root on the same disk, so the copy of a live host also carries the
live `data/store.sqlite` and its `-wal`; the procedure must say which file to restore from.
Resolution: a restore procedure with numbered steps, per host, that starts from an empty machine
and names every input (backup tree, key, runtime, service unit); `backup verify` extended to
`backup rehearse`: restore the snapshot and config into a scratch instance root, run migrations,
run reconciliation in report-only mode, start the data layer and read one record of each kind,
then delete; snapshot written atomically with two generations and a `latest` marker.

### 6. Scratch deletes the evidence on failure (serious)

"Scratch deletes its temp root at exit unless `--keep`; the kept path is the bug report." An
agent reproduces a flaky failure once, without `--keep`, and the fixture N-08 wants is gone; the
rerun may not fail. On Windows the deletion is itself a hazard the reference doc documents: a git
repository plus worktree under `%TEMP%`, possibly with a harness child or Defender holding a
handle, possibly needing the `\\?\` procedure; a deletion that fails must not turn a passing run
into a failing one, and a crash leaves orphans in temp forever. Who can lift: the invoker, by
re-running with `--keep`, which is one more run of whatever cost the first run had.
Resolution: keep the root whenever the process exits non-zero or the deletion fails, print the
path on stderr in both cases; `--keep` forces keeping on success; a `scratch prune` command
removes roots older than N days; deletion uses the worktree-removal interface of 06-platforms,
not a bare recursive delete.

### 7. Identity by natural name makes a rename a data migration, and credentials get two reference schemes (minor)

Usage rows copy `project_id` at write time and store rows point at config "by natural ids
(project name, Discord channel id)". Channel ids are Discord's and stable; a project name is the
owner's and will be renamed. After a rename the totals R-45 wants split across two names and
every session and worktree row points at a project the config no longer has. Separately, config
references a credential "by name" while the data model and 0022 identify a credential by "a
credential id the instance assigns"; the project git credential of R-36 has no owning
registration, which the domain model requires of a Credential row. Two naming schemes for one
store, decided in passing.
Resolution: a project carries an immutable `id` in its config file (set once, never edited;
validation refuses a changed id with a message that says why) and a display `name`; rows point
at the id. One reference scheme for credentials, stated here and handed to 0022: config
references a credential by the same id the store uses, and the project git credential is a
Credential row owned by the instance.

### 8. The debug CLI as a second writer pre-decides the debug transport and introduces two migrators (minor)

The proposal assumes the CLI opens the live file as a second process, uses `busy_timeout` "if the
CLI also writes", and runs migrations "at start". 06-platforms says the debug transport (CLI,
local HTTP, REPL) is 05's decision after 0020. Two consequences slip in: (a) a CLI built from a
newer checkout could migrate the schema under a running bot; (b) R-52 wants pending permission
prompts and live session state, which are in the bot's memory, so a file-reading CLI cannot answer
them anyway, which argues for a transport into the bot process. Also, under 0022 the bot may run
as its own unix user on the droplet; a WAL reader needs write access to the `-shm` file, so a CLI
run as the owner may fail to open the store read-only.
Resolution: state that only the bot process migrates and the CLI refuses a schema it does not
recognise, with a message naming both versions; state that the store does not require a second
writer and that 05 is free to route the CLI through the bot; note the `-shm` permission fact for
0022.

### 9. "Atomic with its audit event" holds only for store-internal actions (minor)

Push, pull request, Discord posts and worktree deletion happen outside the transaction. A crash
between the push and the commit of its audit row loses exactly the event N-17 cares about, and
the proposal's "nothing is lost on a crash mid-write" is overclaimed for those kinds.
Resolution: two events for every external action (`attempted` before, `done` or `failed` after)
so a gap is visible; say so in the audit sub-choice.

### 10. The node:sqlite claim is misstated; better-sqlite3 is stronger than claimed (minor)

See the table. `node:sqlite` is a release candidate on Node 24 and 26, experimental with a
runtime warning on Node 22 (the version on this PC). `better-sqlite3` v13 bundles prebuilds for
both hosts inside the package and no longer needs network access or node-gyp on install.
Resolution: correct the TypeScript row to say `better-sqlite3` first, `node:sqlite` once 0020
pins Node 24 or later and accepts a 1.2-stability API.

### 11. "No finer-grained permission than a trigger" is false (minor)

`sqlite3_set_authorizer` can deny UPDATE and DELETE per table and DROP TRIGGER at compile time,
per connection; `PRAGMA query_only` makes a connection read-only. Neither helps against another
connection, so this does not change objection 2, but the bot's own connection can install an
authorizer where the binding exposes it (Python, rusqlite do; `better-sqlite3` does not), which
is a cheap second guard against a code bug.
Resolution: correct the claim; list the authorizer as optional hardening per binding.

### 12. The Defender exclusion as stated weakens the defence where it matters (minor)

The SQLite-specific slowdown is unverified (general evidence only). More important, "an exclusion
for the instance directory" would exclude the worktree root, which is exactly where a tricked
agent writes untrusted code and where real-time scanning is the one host-level defence the Windows
first version has (0022 may offer no other).
Resolution: if an exclusion is ever recommended, scope it to `data/` only and record it as an
operator step with the tradeoff stated; measure first.

### 13. WAL housekeeping with a second process (minor)

A long-lived CLI reader (`audit --follow`, a stuck `backup verify`) starves checkpoints and the
WAL grows without bound (wal.html). The online backup API restarts whenever another process
writes, so a snapshot taken from the CLI while sessions run may never finish; `VACUUM INTO` runs
inside one read transaction and does not have this problem.
Resolution: snapshot by `VACUUM INTO`, not the backup API, when run from a second process; CLI
readers hold transactions per query, never per command; `wal_autocheckpoint` left at default and
a `PRAGMA wal_checkpoint(TRUNCATE)` after the nightly snapshot.

### 14. The schema-version refusal is not specified (minor)

Migrations "applied at start, recorded in the database" cover old-backup-onto-new-binary. The
other direction (a snapshot from a newer schema restored onto an older binary after a rollback)
must refuse to start; the proposal does not say so or what the message says.
Resolution: refuse with "store schema N, binary knows M; upgrade the binary or restore an older
snapshot" and exit 1.

### 15. The default worktree placement pre-decides 0022 (minor)

"the worktree root unless a project configures its own" under the instance root. 06-platforms
lists worktree placement (under the checkout, a per-instance root, or a per-session user's home)
as 0022's question, and the per-session-user option is the one that gives the droplet its reach
enforcement.
Resolution: 0021 requires only that every worktree path is recorded in the store and inside the
backup set; the default location is 0022's.

## The case against the recommendation, and for the alternative

The strongest honest case is not for C, which fails zero-administration on both hosts and makes
every scratch run and every rehearsal depend on a service, but for a leaner B than the proposal
describes: one directory per session holding its records, a JSON file per rarely-changing record
kind, and `audit.jsonl` and `usage.jsonl` as authoritative append-only files. The proposal
weakens B by assuming a single `turns.json` rewritten per message; nobody would build it that
way, and "one directory per session" is the natural layout, not a reinvented database. Every
mainstream runtime already wraps MoveFileEx with replace semantics (`os.replace`, `fs.rename`,
`os.Rename`), so the per-host atomic-replace story is largely the runtime's, not ours. B's real
advantages are the owner's: after a tricked-agent incident the whole state is greppable and
diffable, the audit log is `tail -f`-able, and the backup is readable by eye with no tool, which
is what "backups are the backstop" means to a person at 3 am. B's real costs stand, and they are
the ones that decide it: totals need either a scan or a second source of truth; the second
process needs a lock we write and test on two hosts; the append-only property of a text file is
exactly as weak as the trigger (anyone with the file truncates it); and the token cost of writing,
testing and debugging a store is spent on the part of the project with the least value, by agents
who write SQL against SQLite better than they write file locking for Windows. A is the right
engine. What the alternative teaches is that the owner-facing properties B has for free (readable
by eye, tailable, diffable) should be bought cheaply in A: the JSONL export and an optional
best-effort file sink for the audit log, not a nightly-only export.

## Who can lift each refusal (summary)

| Mechanism | Who lifts it | From where | While running | Message says so |
|---|---|---|---|---|
| Audit trigger refuses UPDATE/DELETE | anyone with the file (DROP TRIGGER) | sqlite3 shell or any binding | yes | not specified (objection 2) |
| Reconciliation marks worktree removed | nobody | none | n/a | no command exists (objection 1) |
| Config refuses start or reload | the owner, by editing the file | the host | reload: yes; start: bot is down | one line per error, but no fallback (objection 3) |
| Scratch deletes temp root | the invoker, with `--keep` | the command line, before the run | n/a | not on failure (objection 6) |
| Schema newer than binary | the owner, by upgrading or restoring older | the host | n/a | not specified (objection 14) |

## Recommendation and confidence

Accept A with A1, P1 and S3; do not reopen the engine or the journal mode, and do not pursue C.
Before the ADR, the proposal needs: the reconciliation states and commands (1), the audit
trigger honestly scoped with a hash chain and a sink interface (2), last-good config on start (3),
a paragraph on harness-owned state that hands 0022 an explicit constraint (4), a restore
procedure that starts from an empty host and a rehearsal that restores rather than opens (5),
keep-on-failure for scratch (6), and the id rule for projects and credentials (7). The minor
items are one-line corrections except 8, which should at least say that only the bot migrates.
Confidence that SQLite in WAL mode is the right store for this project: high, about 0.9; the
owner's constraints (two hosts, days, zero administration, backup as a directory copy, agents
writing the code) all point the same way and the verification found nothing that weakens it.
Confidence that the proposal as written would restore a host from nothing on the first try:
low, which is why five of the six serious objections are about the mechanisms around the store
rather than the store.
