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
