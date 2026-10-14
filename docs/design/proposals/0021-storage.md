# 0021. Storage: the store, the audit log, configuration placement, scratch mode

Date: 2026-10-14
Status: promoted to ADR 0021

## Question

Where every record in the domain data model (`../02-domain.md`, Data model) lives, where the
audit log and the usage records live relative to it, where configuration lives relative to all
of that, and how the debug interface gets a scratch copy of the lot. Decided now because
`03-architecture.md` and `05-debug-interface.md` are written against it, and because the owner
wants a working first version on both hosts within days: the choice is weighted toward the
option that is running soonest and needs no administration, with the cost of that weighting
stated. Proposal 0020 (language, harness) is open in parallel; this proposal must hold for any
mainstream language 0020 could pick and says which option a language choice would weaken.
Credential storage is 0022's; here the only rule is that the store never holds a secret (N-13).

## Constraints that matter here

- Both hosts from the first version, one code base, no host-only feature (N-01); `06-platforms.md`.
- GPL-compatible dependencies with licenses named (N-23, N-24, ADR 0001).
- Every feature agent-drivable from a terminal; every surface runs against scratch state that
  touches nothing real (N-05 to N-11, R-50 to R-52).
- Everything the data model lists is held; identity and relations are the domain doc's, not
  redesigned here. The store holds a reference to a credential, never its secret (N-12, N-13).
- Zero administration: the owner installs nothing but Broodforge and its runtime on either host
  (N-02; time-to-first-version).
- Backups are the backstop (N-16, N-19, N-20, N-22): all state under one directory tree the
  nightly backup copies; a restore is a written, rehearsed procedure; the backup never contains
  a credential in the clear.
- Crash-safe: a crash or restart never corrupts the store or the audit log (N-27); a usage record
  is written before the turn's result is shown, failed and interrupted turns included (N-21).
- Append-only audit the owner can read (N-17); totals per user, credential, channel and project
  cheap to compute (R-44, R-45; caps later, R-46).
- Concurrent sessions do not block one another (N-26); one bot process per instance, plus the
  debug CLI as a second process on the same host.
- Configuration editable on the host by file or command, outside the repository (R-09, N-03).

## Options

The platforms doc bundles four sub-choices. The store engine is decided first (A to C); the
audit log, configuration and scratch mode each get their own options after it, judged against
the recommended engine where that matters.

### A. Embedded single-file database (SQLite) for store, usage and audit together

One file, `store.sqlite`, in the instance directory, opened by the bot process in WAL journal
mode; one table per record kind in the data model, foreign keys on, indexes on every "points to"
column the totals need. Schema versions are numbered SQL migration files applied by the bot at
start, recorded in the database, so a backup taken under an older schema restores onto a newer
binary; a store newer than the binary refuses to open (response 14).

- Hosts: the SQLite library is embedded in the binary or the language runtime; nothing is
  installed or started on either host. WAL mode needs a local filesystem (not a network share),
  which both hosts provide.
- Crash: a transaction is atomic and durable on application crash in any mode; with
  `synchronous=FULL` also on power loss. At Broodforge's write rate (a few rows per second at
  most) the fsync per commit is not a cost worth optimizing.
- Concurrency: one writer at a time, readers unblocked by the writer in WAL mode. Writes are
  short rows, not long transactions, so a session's write never waits on another's turn; a stuck
  harness holds no lock. The store needs no second writer: the debug CLI may read the live file
  as a second process (holding a transaction per query, never per command, so checkpoints are
  not starved) or reach the bot over whatever transport `05-debug-interface.md` picks; only the
  bot migrates (response 8). A `busy_timeout` covers the rare contention if a second process
  ever writes.
- Backup: one file plus its `-wal` and `-shm` companions. A plain copy of a live WAL database can
  be inconsistent, so the backup set gets a snapshot written by a Broodforge command (`VACUUM
  INTO` to a temporary name, renamed when complete, two generations kept) before the nightly
  copy runs; restore is copy the snapshot back, start the bot, let migrations run. The snapshot
  is also what the restore rehearsal restores (response 5).
- Totals: an indexed `SUM` per key; the usage row carries `session_id`, `requester`,
  `credential_id` and, copied at write time, `channel_id` and `project_id`, so a per-channel
  total is one index scan and a channel moved to another project later does not rewrite history.
- Inspection: the `sqlite3` shell exists on both hosts (a download on Windows, not installed by
  default), and every language ships or has a binding; the debug interface still gives the
  grep-friendly view.
- Cost: low in tokens (schema plus a thin data-access layer the coder writes once); the migration
  runner is a day of work at most; learning is nil for every agent. Risk: a language whose
  binding needs a native build on Windows (below); a Windows antivirus scanning the database file
  on every write (a known class of slowdown for file-heavy software, not verified for SQLite;
  response 12).
- Who else: Claude Code's own session storage is files, not SQLite, so no precedent there;
  SQLite is the default store of most self-hosted single-process bots and desktop applications.

### B. Plain files: one JSON file per record kind, append-only text audit and usage logs

`projects.json`, `sessions.json` and so on under the instance directory, each rewritten whole
by write-to-temp-then-rename on every change; `audit.jsonl` and `usage.jsonl` appended one line
per event; a lock file for the second process. (The critic's leaner B, one directory per session
with a file per rarely-changing record kind, removes the whole-file rewrite below; its other
costs stand. See "On the case for a leaner B" in the responses.)

- Hosts: no dependency at all; every language reads and writes JSON. Rename-replace is atomic on
  Linux; on Windows it needs `MoveFileEx` with replace semantics, which every mainstream runtime
  already wraps, and is not durable without a flush, so the crash story is still ours to prove
  per host.
- Crash: a record file is either the old or the new version after a crash. An appended log can
  end in a partial line, which the reader must skip; the event it held is lost, which is exactly
  the audit event of the action that was in flight. N-21 (usage written before the result is
  shown) holds if the append is flushed before the Discord post.
- Concurrency: whole-file rewrite serializes every writer behind one lock; turns, permission
  decisions and usage grow without bound, so `turns.json` is rewritten on every message of every
  session. Fine at ten sessions, a design debt at a thousand records per day. Splitting per
  session (one directory per session) fixes that and reinvents a database's layout.
- Backup: a plain copy of the directory; a copy taken mid-rename may miss one file's latest
  version, nothing worse. Restore is copy back. Readable by eye, which no other option offers.
- Totals: scan `usage.jsonl` (hundreds of thousands of lines over a year, a second at most) or
  keep running totals in a file, which is a second source of truth to keep consistent.
- Cost: no library, but Broodforge writes its own locking, partial-line recovery, indexing and
  migration code and tests them on both hosts; that is the time the owner does not have, spent on
  the least interesting part of the bot. Risk: subtle per-host crash bugs found in production.
  Who else: small bots and CLI tools; most grow out of it.

### C. A database server (PostgreSQL) on each host

PostgreSQL installed as a service on the PC and on the droplet; Broodforge connects over a local
socket; one schema as in A.

- Hosts: an installer and a Windows service on the PC, a package and a systemd unit on the
  droplet; two different administration procedures, both the owner's, before the first run.
  Every mainstream language has a mature driver.
- Crash: the best of the three; the server's own journal handles it.
- Concurrency: the best of the three, and more than a one-process bot ever needs.
- Backup: not a file copy. A copy of a live data directory is not a valid backup; the nightly
  job must run `pg_dump` (or a base backup) and the restore must run the server first and load
  the dump into it, which is the rehearsed procedure N-19 asks for, with a server in the loop.
- Scratch mode: a scratch database must be created on a running server for every debug run,
  so even the test suite needs the service installed and up.
- Cost: the highest in setup and in every rehearsal; tokens comparable to A. Who else: anything
  multi-process or multi-host, which the first version is not and the vision never asks for.

### Sub-choice: the audit log

- A1. A table in the store, authoritative, append-only by trigger (`UPDATE` and `DELETE` on it
  raise with a message that names the rule and the lift: the trigger changes only through a
  numbered migration), written in the same transaction as the store change it records, so a
  session start and its audit event commit or roll back together. Actions that happen outside
  the store (push, pull request, Discord post, worktree deletion) get two events, `attempted`
  before and `done` or `failed` after, so a crash between the action and its record leaves a
  visible gap rather than silence (response 9). What the trigger protects against: Broodforge's
  own bugs and a careless shell. What it does not: anyone who can open the file, including a
  tricked agent running as a user that can reach `data/`; the audit log is forensics for that
  case, not tamper-proofing against it, and the defence is the off-host copy (response 2). Read
  through the debug interface (`audit` with a tail, a filter and `--json`) or any SQLite shell.
  The audit writer is a sink interface: the table is the one authoritative sink; an optional
  best-effort file sink appends one JSON line per event to a path the instance config names, so
  the owner gets `tail -f` and 0022 can point it at an OS-level append-only target on the
  droplet; the nightly snapshot command also exports `audit.jsonl` beside the snapshot, so the
  backup is greppable without Broodforge or SQLite and is the off-host anchor a tampered table
  is compared against.
- A2. An append-only text file, authoritative. Readable by eye and by `tail -f`. Loses the
  in-flight line on a crash, is not atomic with the store write (an event without its record or
  a record without its event is possible after a crash), needs its own rotation and its own
  scratch handling. Two write paths with two failure modes to test on two hosts.

### Sub-choice: configuration placement

- P1. Files the owner edits; the store holds runtime state only. One instance file plus one
  file per project (channels inside their project), in a `config/` directory under the instance
  root; format is `03-architecture.md`'s. Read at start and on a `reload` debug command, and
  checked without loading by a `config check` command the owner runs after editing. Validation
  refuses a reload with one line per error and the old configuration stays active; nothing
  half-loads. At start, an invalid `config/` falls back to the last-good copy the bot wrote at
  its previous successful load (`data/config-last-good/`), with a `configuration invalid` audit
  event, a notice in the instance's notice channel when the last-good config names one, and a
  first line in `status` saying so until a valid reload; with no last-good copy (first start)
  the start is refused with the errors (response 3). A reference from config to a credential id
  that 0022's store does not hold is a validation error that names the id. Each load writes a
  `configuration changed` audit event carrying the file's hash and the top-level keys that
  changed. A running session keeps the reach and mode it started with; the session row copies
  reach at start. Records point at configured things by stable ids: a project carries an
  immutable `id` in its file beside its display `name`, and channels are Discord's ids, so a
  rename or a config edit never touches the store; `config check` warns when store rows
  reference a project id config no longer defines (response 7). Secrets (the bot token, the
  project git credential of R-36) are referenced from config by the credential id the store
  uses and live in the credential store of 0022, never in a config file, so `config/` is safe
  in the clear and the owner may keep it in a private git repository.
- P2. Configuration in the store, changed by a command (and later by the web site, R-10).
  Atomic with its audit event, typed at the point of change, one backup artifact. Costs a debug
  command per configurable field from day one, a migration for every config shape change, and
  the owner reads config through a tool. Zero-administration is unaffected either way.
- A hybrid (files as source, parsed copy cached in the store) is two sources of truth and is not
  proposed. The last-good copy under `data/` is not a source: it is read only when `config/`
  fails to validate at start, and the bot says so until the files are fixed.

### Sub-choice: scratch mode (N-09)

- S1. In-memory store only (`:memory:` in SQLite). Fast, isolated, gone at exit. Covers one
  process only: the debug CLI and the bot cannot share it, and worktrees, config and the
  credential store are files regardless, so in-memory scratch covers a third of the state.
- S2. Temp instance directory only. One flag or environment variable names the instance root;
  scratch creates a fresh root in the OS temp directory with a generated minimal config, an
  empty store, a scratch credential-store entry and a throwaway git repository as the project,
  prints the path, and at exit deletes it only on success: the root is kept, and its path printed
  on stderr, whenever the process exits non-zero or the deletion fails; `--keep` keeps it on
  success too; `scratch prune --older-than <days>` removes leftovers; deletion goes through the
  worktree-removal interface of `06-platforms.md`, never a bare recursive delete (response 6).
  Covers everything, costs a few hundred milliseconds, and the kept path is the fixture a bug
  report names (N-08).
- S3. Both: S2 as the debug interface's scratch mode, S1 for in-process unit tests of the data
  layer. Selection: the instance root is the one setting; a path of `:memory:` for the store is
  honoured only by the test harness, never by the CLI.

## Recommendation

A, with A1, P1 and S3: SQLite in WAL mode as the one store for records, usage and audit; the
audit log a trigger-protected table written in the action's transaction, with an optional file
sink and a JSONL export by the nightly snapshot command; configuration in owner-edited files
under the instance root with secrets referenced by id and a last-good fallback at start; scratch
as a generated temp instance directory kept on failure, with in-memory SQLite for unit tests. It
is the only combination that is zero-administration on both hosts, crash-safe by a library's
test suite rather than ours, backed up as a directory copy, and restorable without a server,
which is what days-to-first-version and backups-as-backstop ask for together. The cost of the
fast option, stated: a single-writer store that a second bot process could not share, which the
vision never asks for, and an audit log the owner reads through a shipped command, the file
sink, or the export rather than the authoritative table itself.

Layout under the instance root (names are illustrative; `03-architecture.md` fixes them):
`config/` (P1), `data/store.sqlite` with its WAL companions and `data/config-last-good/`,
`backup/` holding the two newest snapshots, a `latest` marker and the `audit.jsonl` export,
`credentials/` (0022's). Worktrees and project checkouts live wherever 0022 places them; 0021
requires only that every worktree path is recorded in the store and lies inside the nightly
backup's paths, and the restore procedure names those paths (response 15). Nothing of
Broodforge's is written outside the backup set (N-20); SQLite's own temporary files go to the
OS temp directory and are not state.

Harness-owned state, an input 0022 must handle (response 4): Claude Code keeps its own state in
its config directory (`~/.claude` or `CLAUDE_CONFIG_DIR`): session transcripts, settings, and its
own credentials file, which exists in the clear on Windows. None of it is Broodforge state in the
first version; transcripts become state only if R-18 (resume) is revived, and that decision names
the directory and adds it to the backup set. The rule 0021 imposes on 0022's delivery mechanism:
0022 states where each harness directory lands relative to the backup set, and a harness
directory inside the set is either encrypted at rest by 0022 or excluded by name and listed as
unrestorable in the restore procedure; the owner's own `~/.claude` on the PC is never inside the
set. If 0022 runs the bot as its own user on the droplet, a direct reader of the live store
needs write access to the `-shm` file, which 0022 and `05-debug-interface.md` take into account.

Crash and restart rules the store must support: at start, every session in a running state is
marked interrupted and reported (R-17). Worktree reconciliation sets a `missing_since` time on
every worktree record whose path is absent and clears it when the path reappears; the state
(active, pushed, kept with unpushed commits, removed) is never changed by a path check, and
`removed` is set only by the removal procedure (R-35) or by an explicit `worktree forget <id>`
command with its own audit event. The start-up report names every missing worktree and whether
it holds unpushed commits; a directory under a worktree root with no record is listed, never
deleted, and `worktree adopt <path>` records it (response 1). The usage row for a turn is
committed before the turn's result is posted; an interrupted turn gets a usage row with what the
harness reported so far (N-21). No retention: audit and usage rows are never pruned in the first
version; revisit when the nightly snapshot no longer fits the backup window.

Backup contents, first-class: `config/`, the snapshots and `audit.jsonl` under `backup/`, the
credential store as 0022 encrypts it, the project checkouts and worktrees at the paths the
store records. The restore source is the newest verified snapshot, never the live
`store.sqlite` copied beside it. Restore is a numbered procedure per host that starts from an
empty machine and names every input, including the ones outside the backup set (the
credential-store key and anything else 0022 keeps off the host) and the operator steps (runtime,
harness, service unit); worktrees restored to a different path get `git worktree repair` before
the first start (response 5). Rehearsal: a `backup rehearse` debug command restores the newest
snapshot and `config/` into a scratch instance root, runs migrations, runs reconciliation in
report-only mode, opens the data layer and reads one record of each kind, prints counts, and
deletes the scratch root on success; `backup verify` (integrity check and counts on the
snapshot alone) stays as the cheap check. The nightly job runs `backup snapshot`, then `backup
rehearse`, then the host's copy; the bot truncates the WAL (`wal_checkpoint(TRUNCATE)`) after
the snapshot (response 13).

Language sensitivity, so the recommendation holds whatever 0020 picks:

| Language 0020 could pick | A (SQLite) | B (files) | C (PostgreSQL) |
|---|---|---|---|
| TypeScript / Node | `better-sqlite3` (v13 bundles prebuilt binaries for win32-x64 and linux-x64 in the package); `node:sqlite` once 0020 pins Node 24 or later and accepts a release-candidate API (it warns as experimental on Node 22, the owner's version); not weakened | unaffected | driver fine; admin cost unchanged |
| Python | `sqlite3` in the standard library; not weakened | unaffected | driver fine |
| Go | cgo binding needs a C toolchain on Windows; the pure-Go port avoids that at some speed cost; weakened mildly, not blocked | unaffected | driver fine |
| Rust | `rusqlite` with the bundled feature compiles SQLite in; not weakened | unaffected | driver fine |
| C# / .NET, Java | first-party or standard bindings with bundled natives; not weakened | unaffected | driver fine |

B is weakened by no language and C by none; A is weakened only by Go on Windows, where a known
workaround exists. Where the binding exposes `sqlite3_set_authorizer` (Python and `rusqlite`
do; `better-sqlite3` does not), the bot's own connection may deny `UPDATE`, `DELETE` and
`DROP TRIGGER` on the audit table as optional hardening against its own bugs (response 11).

First implementation step: the schema for the data-model table as migration `0001`, the
migration runner, the scratch instance root, and the `backup snapshot`, `backup verify` and
`backup rehearse` commands, with the data-layer tests run on both hosts before any Discord or
harness code.

## What would change this recommendation

- 0020 picks a language whose only SQLite binding needs a native build that fails on the
  owner's Windows toolchain and has no prebuilt: B becomes the first version, behind the same
  data-access interface, with A as the planned replacement.
- A second process must write the store concurrently with the bot on the same host (for
  example the web site of R-10 as its own service): C, or the web site writes through the bot.
- The vision changes to several bot hosts sharing one state: C.
- The owner says reading the audit log must need no tool at all, including after a crash: the
  file sink becomes mandatory rather than optional, the table stays authoritative.
- R-10's web site is pulled into the first version: P2 for channel configuration, P1 for the
  instance and project files.
- The critic shows that SQLite WAL mode misbehaves on either host's filesystem in a way the
  rollback journal does not: journal mode `DELETE`, same recommendation otherwise.

Deferred owner questions: none bears on this choice. Q 1 (credential kind) changes nothing in
the store, since the store holds a reference either way. Q 3 (venue) adds a thread id or a
channel id to the session row, nothing more.

## Claims to verify

Verified by the critic unless marked; sources in `0021-critic-review.md`, Claims verified.

- SQLite is public domain and every binding named above has a GPL-compatible license:
  `better-sqlite3` (MIT), `node:sqlite` (part of Node, MIT), Python `sqlite3` (PSF), Go
  `modernc.org/sqlite` (BSD-3) and `mattn/go-sqlite3` (MIT), `rusqlite` (MIT),
  `Microsoft.Data.Sqlite` (MIT), `sqlite-jdbc` (Apache-2.0). PostgreSQL License is permissive.
- `node:sqlite` is not behind a flag since Node 22.13 but is a release candidate (stability 1.2)
  on Node 24 and 26 and experimental with a runtime warning on Node 22; `better-sqlite3` v13
  ships prebuilt binaries for win32-x64 and linux-x64 inside the npm package. (Corrected from
  the draft's "stable in a current LTS".)
- SQLite WAL mode: readers do not block the writer and vice versa; a plain file copy of a live
  WAL database can be inconsistent; `VACUUM INTO` and the online backup API produce a
  consistent copy, `VACUUM INTO` output may be incomplete if interrupted, and the backup API
  restarts when another process writes; WAL mode requires a local filesystem and shared memory
  and is unsupported on network filesystems; a long-running reader starves checkpoints and the
  WAL grows (sqlite.org, "Write-Ahead Logging").
- `synchronous=FULL` in WAL mode gives durability on power loss; `NORMAL` can lose the last
  transactions on power loss but not corrupt the database (sqlite.org, PRAGMA synchronous).
- SQLite triggers can refuse `UPDATE` and `DELETE` on a table with `RAISE(ABORT, message)`;
  finer controls exist per connection (`sqlite3_set_authorizer`, `PRAGMA query_only`), and
  neither they nor the trigger survive a different connection that drops the trigger.
  (Corrected from the draft's "no finer-grained permission".)
- `sqlite3` command-line shell is available as a download for Windows and a package on Debian
  and Ubuntu; not installed on the owner's PC today.
- Unverified: Windows Defender real-time scanning slows SQLite writes markedly. General evidence
  for file-heavy software only; measure before recommending an exclusion, and scope any
  exclusion to `data/`, never the worktree root.
- On Windows, replacing a file atomically requires `MoveFileEx` with
  `MOVEFILE_REPLACE_EXISTING` (or `ReplaceFile`), which mainstream runtimes wrap, and
  durability needs `FlushFileBuffers`; Linux `rename(2)` is atomic. (Bears on B's crash story.)
- An append to a text file interrupted by a crash can leave a partial last line on both hosts.
  (Bears on A2 and B.)
- PostgreSQL on Windows installs as a service through an installer; a file copy of a live data
  directory is not a valid backup; `pg_dump` is.
- Claude Code stores its own session state as files under its config directory, not in SQLite;
  its credentials file exists in the clear on Windows (bears on "who else" in A and on the
  harness-owned state paragraph).
- Go: `mattn/go-sqlite3` needs cgo and therefore a C compiler on Windows; `modernc.org/sqlite`
  is a cgo-free translation. Unverified: that it passes the SQLite test suite.
- `git worktree repair` fixes worktree links after a move or copy (bears on restore).

## Choices made where the docs were silent

- One database file for records, usage and audit together, not one per concern.
- Usage rows carry `channel_id` and `project_id` copied at write time; the session row carries
  `reach` copied at start; the worktree row carries a nullable `missing_since`; audit events for
  external actions carry a phase (`attempted`, `done`, `failed`). All additive to the domain
  model's relations.
- A project has an immutable `id` in its config file and a display `name`; store rows point at
  the id. Config references a credential by the store's credential id; the project git
  credential of R-36 is a Credential row whose owner is the instance rather than a registration,
  which 0022 stores like any other.
- Secrets referenced from config live in 0022's credential store, so `config/` is safe in a
  backup and in a private repository.
- Start-up reconciliation of sessions and worktrees against the disk, listed above, with
  `worktree forget` and `worktree adopt` as the owner's commands.
- Last-good configuration under `data/` as the fallback at an unattended start.
- Only the bot migrates; a second process that opens the store refuses a schema it does not
  know, naming both versions; a store newer than the binary refuses to start with
  "store schema N, binary knows M; upgrade the binary or restore an older snapshot", exit 1.
- `backup snapshot`, `backup verify`, `backup rehearse`, `audit`, `config check`, `reload`,
  `scratch prune`, `worktree forget` and `worktree adopt` as debug-interface commands by those
  working names; `05-debug-interface.md` fixes the names.
- Scratch keeps its temp root on failure and deletes it on success unless `--keep`; the kept
  path is the bug report.
- No retention of audit or usage rows in the first version.

## Critic's objections and responses

Review: `0021-critic-review.md` (0 blocking, 6 serious, 9 minor; engine choice agreed). Body
corrections are in place above; each response names the section it changed.

1. Reconciliation marks a worktree removed from a path check alone (serious). Right: a path can
   be absent after a partial restore, an unmounted drive or a move, and the one state that exists
   to warn the owner must not be silently converted. Accepted with change: instead of a new
   `missing` state, the worktree row gets a nullable `missing_since` that reconciliation sets and
   clears, so the real state (kept with unpushed commits, in the case that matters) survives the
   absence and is reported with it; `removed` only from the removal procedure or `worktree
   forget`, `worktree adopt` for the unrecorded directory. Recommendation, crash and restart.
2. The audit trigger has no lift, no retention, and does not defend against the stated threat
   (serious). Right on all three, and the draft overclaimed by presenting the trigger as N-17's
   append-only property. Accepted with change. (a) Scoped in A1: the trigger guards against our
   own bugs; against a tricked agent that can open the file the audit log is forensics, and the
   defence is the off-host copy, which is the owner's stated backstop. (c) Lift defined: trigger
   changes only through a numbered migration, and the `RAISE` message says so. (d) Sink
   interface accepted: table authoritative, optional best-effort file sink 0022 can aim at an
   OS-level append-only target. (e) Retention written down: none in the first version, trigger to
   revisit stated. (b) The hash chain is rejected because it is tamper-evident only against an
   anchor the attacker cannot reach, and the nightly `audit.jsonl` export already is that anchor:
   comparing the export against the table detects any change to exported rows, and rows written
   after the last export are unprotected with or without a chain. A chain adds a column, a
   verifier and a migration hazard for no defence the export does not give, and would read as
   the tamper-proofing this proposal declines to promise.
3. Invalid config leaves the bot down after an unattended restart (serious). Right; the draft
   covered reload and not start. Accepted: last-good copy written at every successful load, used
   only at start when `config/` fails validation, announced in the audit log, the notice channel
   and `status`; `config check`; cross-store validation of credential ids; running sessions keep
   reach and mode from start, with reach copied to the session row. P1.
4. Silence on harness state decides 0022 by omission (serious). Right. Accepted, in the shape the
   coordinator asked for: 0021 names harness-owned state as a category and states the constraint
   0022 must satisfy (say where each harness directory lands relative to the backup set; inside
   it means encrypted at rest or excluded and listed as unrestorable), and decides nothing about
   the delivery mechanism. R-18 is explicitly left to its own revival, which is where transcripts
   would become state. Recommendation, harness-owned state.
5. "Copy the tree back, start" does not rebuild a host (serious). Right on every item. Accepted:
   the restore is a numbered procedure per host from an empty machine naming every input, the
   out-of-set ones included; snapshot written to a temporary name and renamed, two generations
   and a `latest` marker; `backup rehearse` restores into a scratch root rather than opening the
   file; `git worktree repair` named; the restore source is the snapshot, not the live copy.
   The procedure text is the ADR's companion file. Option A, backup; Recommendation, backup.
6. Scratch deletes the evidence on failure (serious). Right. Accepted: keep on non-zero exit or
   failed deletion with the path on stderr, `--keep` for success, `scratch prune`, deletion via
   the worktree-removal interface. S2.
7. Natural-name identity makes a rename a migration; two credential reference schemes (minor).
   Accepted with change: immutable project `id` plus display `name`, and one credential scheme
   (the store's id) with the project git credential as an instance-owned Credential row, as
   proposed. The change: a changed id is not refused, because the store cannot tell a rename
   from a removal; `config check` warns about rows that reference a project id config no longer
   defines, which catches both. P1; Choices.
8. The CLI as a second writer pre-decides the transport and adds a second migrator (minor).
   Accepted: only the bot migrates, a second process refuses an unknown schema naming both
   versions, the store needs no second writer and 05 may route the CLI through the bot, the
   `-shm` permission fact handed to 0022. Option A, concurrency; Choices.
9. Atomic with its audit event holds only for store-internal actions (minor). Accepted:
   `attempted` and `done` or `failed` events for every external action; the overclaim reworded. A1.
10. The `node:sqlite` claim is misstated (minor). Accepted: the TypeScript row and the claim
    corrected; `better-sqlite3` first as the example binding, `node:sqlite` once a Node 24 pin
    and a release-candidate API are accepted by 0020, which stays undecided here.
11. "No finer-grained permission than a trigger" is false (minor). Accepted: claim corrected;
    the authorizer listed as optional hardening where the binding exposes it.
12. The Defender exclusion as stated weakens the defence where it matters (minor). Accepted:
    the slowdown is marked unverified, and any exclusion is scoped to `data/`, measured first,
    and recorded as an operator step; the worktree root is never excluded.
13. WAL housekeeping with a second process (minor). Accepted: `VACUUM INTO` for snapshots from
    any process, per-query transactions in second-process readers, `wal_checkpoint(TRUNCATE)`
    after the nightly snapshot.
14. The schema-newer-than-binary refusal is unspecified (minor). Accepted: refuse with the
    message the critic wrote, exit 1. Choices.
15. The default worktree placement pre-decides 0022 (minor). Accepted: 0021 requires only that
    every worktree path is recorded in the store and inside the backup set; placement is 0022's.
    Recommendation, layout.

On the case for a leaner B: accepted as a correction of the record (B's whole-file rewrite is
the naive layout, and runtimes wrap the Windows replace call), and rejected as a reason to pick
B, for the costs the critic names and the proposal keeps: totals need a scan or a second source
of truth, the second-process lock is ours to write and test on two hosts, and a text file is
exactly as tamperable as the trigger. The lesson is taken in A1: the file sink and the export buy
the readable, tailable, greppable properties cheaply.

No serious or blocking objection remains unresolved; the engine choice stands at the critic's
confidence and this proposal is promoted to ADR 0021.
