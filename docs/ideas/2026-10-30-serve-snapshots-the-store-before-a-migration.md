---
date: 2026-10-30
kind: feature
---

# `serve` snapshots the store before applying a migration

`serve` on the dev root migrated the store from schema 3 to 4 silently, as a side effect of starting.
A migration is the one store change that cannot be undone from the file itself. `serve` should take a
`backup snapshot` first when migrations are pending, and log one line naming the migration, so a bad
migration costs a restore and the owner can see why the store changed.

Anchor: the migrations in `internal/store/migrate.go` and `internal/store/migrations/`; the snapshot
in `internal/backup/backup.go`.

Plan: coder. In `serve` start-up, before step (4) `store.Create(root)` in `internal/cli/serve.go`: when
`data/store.sqlite` exists, open it with `store.OpenFile`, read `store.SchemaID` and compare with
`store.Known()` (`internal/store/migrate.go`; `migrate` itself runs inside `Create`, so the snapshot must
come first, from outside the store package). When the file is behind, call `backup.Snapshot(root, auditExport)`
(`internal/backup/backup.go`; `auditExport` is in `internal/cli/usage.go`) and log one line to stderr naming
the from and to schema and the snapshot path. A snapshot failure stops `serve` with a start-up step error
rather than migrating unprotected. A new or current store takes no snapshot. Verify: a test that builds a
schema-3 store fixture, starts the serve path against it, and finds the snapshot under `backup/` and the
log line; a second start takes none. Out of scope: other commands that open the store, a restore command.
