# 0021. Storage: SQLite store with audit table, file configuration, temp-directory scratch

Date: 2026-10-14
Status: accepted

Record: proposal `../proposals/0021-storage.md` and `../proposals/0021-critic-review.md`.
Companion: `0021-backup-and-restore.md` (snapshot, backup set, restore per host, rehearsal).

## Context

Every record in the data model of `../02-domain.md` needs a home, the audit log (N-17) and the
usage records (N-21, N-22) need one relative to it, configuration (R-09, N-03) one relative to
both, and the debug interface needs scratch copies of all of it (N-09). Both hosts from the first
version with zero administration (N-01, N-02), backups as the backstop (N-16, N-19, N-20),
crash safety (N-27), unblocked concurrent sessions (N-26), and a first version within days.
Proposal 0020 (language) is undecided; this decision is language-neutral. Proposal 0022
(isolation, credentials) is unwritten; this decision hands it named constraints and decides
nothing of its own.

Verified by the critic (sources in the review): SQLite is public domain and every binding named
in the proposal is GPL-compatible; in WAL mode readers and the writer do not block each other,
a plain copy of a live database can be inconsistent, `VACUUM INTO` gives a consistent snapshot
that may be incomplete if interrupted, `synchronous=FULL` is durable on power loss, a long
reader starves checkpoints; a trigger refuses `UPDATE` and `DELETE` with a chosen message and
nothing in the file survives a connection that drops it; `better-sqlite3` v13 bundles prebuilt
binaries for both hosts, `node:sqlite` is a release candidate on Node 24 and experimental on
Node 22; Claude Code keeps its own state, credentials file included, as files under its config
directory. Not verified: a Defender slowdown specific to SQLite.

Rejected, reasons in the proposal: plain files (locking, partial lines, indexing ours to write and
test on two hosts); PostgreSQL (a service on each host, in every restore and scratch run); a text
audit log as authoritative (not atomic with the store write); configuration in the store; in-memory
scratch alone; an audit hash chain (tamper-evident only against the off-host anchor the export is).

## Decision

Engine. One SQLite database, `data/store.sqlite` under the instance root, journal mode WAL,
`synchronous=FULL`, `foreign_keys=ON` and a `busy_timeout` on every connection; one table per
record kind in the data model, an index on every "points to" column a total or a lookup uses.
The binding is the language's standard one with bundled or prebuilt natives (for TypeScript the
example is `better-sqlite3`); 0020 names it. Where the binding exposes `sqlite3_set_authorizer`,
the bot's connection may deny `UPDATE`, `DELETE` and `DROP TRIGGER` on the audit table as
optional hardening; it is not relied on.

Schema and migrations. Numbered SQL migration files applied by the bot process only, recorded in
a `migrations` table (id, applied at). A store with a migration id the binary does not know
refuses to open: "store schema N, binary knows M; upgrade the binary or restore an older
snapshot", exit 1. Any second process that opens the store (the debug CLI, if
`05-debug-interface.md` has it read the file directly) never migrates and refuses an unknown
schema with the same message; it holds a read transaction per query, never per command. The
store needs no second writer; 05 may route the CLI through the bot.

Additive columns beyond the domain model: `usage_record` carries `channel_id` and `project_id`
copied at write time; `session` carries `reach` copied at start; `worktree` carries a nullable
`missing_since`; `audit_event` carries a nullable `phase` (`attempted`, `done`, `failed`).

Audit log. Table `audit_event` (`seq` autoincrement primary key, time, kind, phase, actor,
session id, registration id, details as JSON text), written in the same transaction as the
store change it records. Two triggers, `BEFORE UPDATE` and `BEFORE DELETE`, each
`RAISE(ABORT, 'audit_event is append-only (ADR 0021); change this trigger only through a
numbered migration')`. Every action outside the store (push, pull request, Discord post,
worktree deletion) writes `attempted` before and `done` or `failed` after. Scope, stated: the
trigger guards against Broodforge's own bugs; against a tricked agent that can open the file,
the audit log is forensics and the off-host copy is the defence. The audit writer is a sink
interface: the table is the authoritative sink; an optional file sink appends one JSON line per
event, best effort, to a path the instance config names (unset by default); the nightly snapshot
exports `audit.jsonl` beside the snapshot. No retention of audit or usage rows; revisit when the
nightly snapshot no longer fits the backup window.

Configuration. Owner-edited files under `config/`: one instance file, one file per project with
its channels inside; format in `03-architecture.md`. A project file carries an immutable `id` and
a display `name`; store rows point at the id and at Discord's channel ids, never at a row of
config. Config references a credential by the credential id the store uses; the secret lives in
0022's credential store, so `config/` holds no secret. Loaded at start and by `reload`; checked
without loading by `config check`. Validation: one line per error; a credential id the
credential store does not hold is an error naming the id; a store row that references a project
id config no longer defines is a warning. `reload` on failure keeps the old configuration
active. Start on failure loads `data/config-last-good/`, the copy the bot wrote at its last
successful load, writes a `configuration invalid` audit event with the errors, posts to the
instance's notice channel if the last-good config names one, and shows `config: last-good from
<time>` as the first line of `status` until a valid reload; with no last-good copy the start is
refused with the errors, exit 1. Every successful load writes `configuration changed` with the
file hash and the changed top-level keys. A running session keeps the reach and mode it started
with.

Reconciliation at start. Every session in a running state is marked interrupted and reported
(R-17). Every worktree record whose recorded path is absent gets `missing_since` set; it is
cleared when the path is present again; the state is never changed by a path check. `removed`
is set only by the removal procedure (R-35) or by `worktree forget <id>`, which writes its own
audit event. A directory under a worktree root with no record is listed, never deleted;
`worktree adopt <path>` records it. The start-up report lists each missing worktree with whether
it holds unpushed commits. The usage row of a turn is committed before the result is posted; an
interrupted turn gets a row with what the harness reported so far (N-21).

Layout and backup set. Under the instance root: `config/`, `data/` (the store, its `-wal` and
`-shm`, `config-last-good/`), `backup/` (two newest snapshots, `latest` marker, `audit.jsonl`),
`credentials/` (0022's). Worktrees and checkouts live where 0022 places them; every worktree path
is recorded in the store and lies inside the nightly backup's paths. Nothing of Broodforge's is
written outside the backup set; SQLite's temporary files in the OS temp directory are not state.
Snapshot, verify, rehearse and the restore procedure per host: the companion file.

Harness-owned state (constraint handed to 0022). Claude Code's config directory (`~/.claude` or
`CLAUDE_CONFIG_DIR`) with its transcripts, settings and its own credentials file is not Broodforge
state. 0022 states, for its delivery mechanism, where each harness directory lands relative to
the backup set; one inside the set is either encrypted at rest by 0022 or excluded by name and
listed as unrestorable in the restore procedure; the owner's own `~/.claude` on the PC is never
inside the set. If the bot runs as its own user, a direct reader of the live store needs write
access to `-shm`; 0022 and 05 account for it. Transcripts become state only if R-18 is revived.

Scratch (N-09). The instance root is the one setting (flag or environment variable). Scratch
creates a fresh root in the OS temp directory with a minimal generated config, an empty store,
a scratch credential-store entry and a throwaway git repository as the project, prints the path,
and at exit deletes the root only on a zero exit and a successful deletion; otherwise it keeps
the root and prints the path on stderr. `--keep` keeps it on success; `scratch prune
--older-than <days>` removes leftovers; deletion uses the worktree-removal interface, never a
bare recursive delete. In-process unit tests of the data layer may use `:memory:`; the CLI
never honours it.

Debug commands by working name (05 fixes the names and transport): `backup snapshot`, `backup
verify`, `backup rehearse`, `audit`, `config check`, `reload`, `status`, `scratch prune`,
`worktree forget`, `worktree adopt`. Exit codes per N-07.

Out of scope: the config file format and the worktree naming (03), the debug transport (05),
credential storage, delivery and worktree placement (0022), token caps (R-46).

Deferred, with triggers: retention (snapshot outgrows the backup window); a mandatory file sink
(the owner asks for a no-tool audit read); a second writer or several hosts (the R-10 web site
as its own service, or a multi-host vision: PostgreSQL or writes through the bot); journal mode
`DELETE` (WAL shown to misbehave on a host filesystem); B behind the same data-access interface
(0020 picks a language whose only binding cannot build on the owner's Windows).

## Consequences

Easier: zero installation on both hosts; one directory copy is the backup; crash safety is the
library's; totals are one indexed sum; every surface runs against a generated scratch root that
survives a failure as the bug's fixture; the owner reads config in an editor and the audit log
through a command, a tailable file or a greppable export.

Harder or costly: a single-writer store a second bot process could not share; the `sqlite3`
shell is a separate download on Windows; migrations are a discipline from the first schema;
the trigger is not a security boundary and the ADR says so, so the off-host copy of the backup
is load-bearing; last-good config means the bot can run on stale settings after a bad edit until
someone reads `status` or the notice channel; the restore procedure has inputs outside the
backup set that 0022 owns and the rehearsal cannot exercise.

Must be maintained: the migration series; the companion file whenever the layout, 0022's key
handling or the service units change; the additive columns above in the domain doc's successors.

Supersede when: a second concurrent writer process is required, several hosts share one state,
or WAL mode is shown to misbehave on a target filesystem.

## Amendments

2026-10-16 (ADR 0023, `0023-language-for-the-long-term.md`): the Decision sentence "The binding is the language's standard one with bundled or prebuilt natives (for TypeScript the example is `better-sqlite3`); 0020 names it." is replaced: the binding is `zombiezen.com/go/sqlite` (ISC) over `modernc.org/sqlite` (BSD-3-Clause, cgo-free, no native build on either host), named by ADR 0023; the optional `sqlite3_set_authorizer` hardening in the same paragraph is taken through `SetAuthorizer` on the bot's connection; PRAGMAs are per connection and set at open, one writer connection and a small read pool. The deferred item "B behind the same data-access interface (0020 picks a language whose only binding cannot build on the owner's Windows)" is closed: the binding builds everywhere without a C compiler.

2026-10-26 (ADR 0027, `0027-setup-in-few-steps.md`, Decision 7): the Decision sentence "Configuration. Owner-edited files under `config/`: one instance file, one file per project with its channels inside" is read with `broodforge setup` as their creator: a command the owner runs with the bot stopped (it holds the control address while it runs), which writes `instance.json` and `projects/<id>.json`, fills only absent keys, and records the ids of the channels and role it finds or creates; `setup <step>` replaces a present key after showing the old and new values and asking. The bot still never writes them. `config export` and `config import` carry `config/` between hosts with the host-bound keys emptied and no secret. The store gains no second writer: `setup` writes it only while it holds the address (ADR 0022, Amendments from ADR 0027).

2026-10-30 (ADR 0020 Amendments 2026-10-30, the `rate-limit` event; the 2026-10-29 triage): the Decision sentence "one table per record kind in the data model, an index on every "points to" column a total or a lookup uses" gains one table outside the data model: `credential_meter`, migration `0004-credential-meter.sql`, with `credential_id` as its primary key referencing `credential`, `seen_at`, `session_id`, `status`, `rate_limit_type`, `five_hour_utilization`, `five_hour_resets_at`, `seven_day_utilization`, `seven_day_resets_at`, `overage_status` and `is_using_overage`, the event's parsed values as read, zero where a field was absent. It holds one row per credential, the latest `rate-limit` event of any turn paid by it, replaced in place (an upsert) by the bot's writer in its own short transaction when the event arrives in a turn's stream; it is a reading, not an action, so no audit event records it, and the append-only rule stays `audit_event`'s alone. `usage_record` is unchanged and remains the token record (R-44); the meter is read by `credential show --json` and `usage --credential` (`../05-debug-interface.md`). The nightly snapshot carries it as any table; a restore from an older snapshot shows an older `seen_at`, which is the point of the column. Built by P28 (`../08-implementation-plan-engine-stream.md`). The `panel` table (migration `0003-panel.sql`: the pinned panel's message id per project channel, and Discord's id for it once a live create returns it; `../03-architecture.md`, Configuration) is likewise a store table outside 02's data model, landed 2026-10-30.
