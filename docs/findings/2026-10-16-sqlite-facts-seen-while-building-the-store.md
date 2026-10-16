---
date: 2026-10-16
status: resolved
area: go / sqlite
promoted_to: docs/reference/go-dependencies.md
---

# SQLite facts seen while building the store

## Question

Which behaviours of `zombiezen.com/go/sqlite` (v1.4.2, over `modernc.org/sqlite`) surprised the P5
coder while building the store and the backup snapshot, and which later pass (the bot's checkpoint in
P10, the rehearsal in P9) would otherwise rediscover?

## What was tried

Observed during P5 on the Windows PC, Go 1.27.2, by building `internal/store` and `internal/backup`
and running their tests. Reported by the coder on 2026-10-16; not re-run for this finding.

## Result

Read-only open of a WAL store leaves files. Opening a WAL-mode store with `sqlite.OpenReadOnly`
(`internal/store/store.go:261`) creates `-shm` and `-wal` beside it when the directory is writable,
and leaves them there after close. A read-only open is therefore not "touches nothing" on disk; a
pass that lists the store directory, or checkpoints on behalf of the bot, must expect both files.

`VACUUM INTO` works on a read-only connection. The snapshot in `internal/backup/backup.go:91` runs
it over the connection `store.OpenReadOnly` returned (line 68), so the live store is never written.
The output file is in `delete` journal mode, not WAL: a snapshot has no `-wal` or `-shm` of its own
and can be copied or renamed as one file.

`PRAGMA synchronous` fails inside `sqlitex.ExecuteScript`. The script runs inside a SAVEPOINT and
the open failed with `SQL logic error: Safety level may not be changed inside a transaction`. Each pragma is its own statement, run one at a time in the
loop at `internal/store/store.go:56` over the list at line 52, on every connection (pragmas are per
connection).

A truncated store opens without error. A copy cut to a few KB opens cleanly and fails only at its
first statement with "database disk image is malformed". Open is not a validity check, which is why
`verify` runs `PRAGMA integrity_check` (`internal/backup/backup.go:209`, called at line 260) and
maps anything but the single row `ok` to `CodeIntegrity`.
