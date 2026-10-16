// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// Migration files are numbered SQL files, `NNNN-<slug>.sql`, applied in order by the bot's
// start (Create) only and recorded in the migrations table (ADR 0021). A file is never edited
// once committed; a change is the next number.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

var migrationName = regexp.MustCompile(`^(\d{4})-[a-z0-9-]+\.sql$`)

type migration struct {
	id   int64
	name string
	sql  string
}

// migrations is the embedded series in id order. A malformed name or a gap is a build-time
// mistake and panics at first use, which every store test reaches.
var migrations = sync.OnceValue(func() []migration {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	var ms []migration
	for _, e := range entries {
		m := migrationName.FindStringSubmatch(e.Name())
		if m == nil {
			panic("store: migration file name " + strconv.Quote(e.Name()) + " is not NNNN-<slug>.sql")
		}
		id, _ := strconv.ParseInt(m[1], 10, 64)
		b, err := migrationFiles.ReadFile("migrations/" + e.Name())
		if err != nil {
			panic(err)
		}
		ms = append(ms, migration{id: id, name: e.Name(), sql: string(b)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].id < ms[j].id })
	for i, m := range ms {
		if m.id != int64(i+1) {
			panic(fmt.Sprintf("store: migration %s: want id %04d", m.name, i+1))
		}
	}
	return ms
})

// Known is the highest migration id this binary knows.
func Known() int64 { return int64(len(migrations())) }

const createMigrations = `CREATE TABLE IF NOT EXISTS migrations (
    id         INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
)`

// schemaError is ADR 0021's refusal for a store this binary does not match.
func schemaError(have, known int64, newer bool) error {
	if newer {
		return &Error{CodeStoreSchema, fmt.Sprintf("store schema %d, binary knows %d; upgrade the binary or restore an older snapshot", have, known)}
	}
	return &Error{CodeStoreSchema, fmt.Sprintf("store schema %d, binary knows %d; start the bot once to migrate it", have, known)}
}

// applied reads the migration ids recorded in a store; a store with no migrations table has none.
func applied(q Querier) ([]int64, error) {
	var exists bool
	if err := q.Query("SELECT 1 FROM sqlite_schema WHERE type = 'table' AND name = 'migrations'", nil,
		func(*Row) error { exists = true; return nil }); err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	var ids []int64
	err := q.Query("SELECT id FROM migrations ORDER BY id", nil, func(r *Row) error {
		ids = append(ids, r.Int(0))
		return nil
	})
	return ids, err
}

// SchemaID is the highest migration id recorded in a store, 0 for none.
func SchemaID(q Querier) (int64, error) {
	ids, err := applied(q)
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	return ids[len(ids)-1], nil
}

// checkSchema refuses a store holding a migration id the binary does not know (always) or,
// unless older is allowed, one missing a migration the binary has.
func checkSchema(q Querier, olderOK bool) error {
	ids, err := applied(q)
	if err != nil {
		return err
	}
	known := Known()
	var have int64
	if len(ids) > 0 {
		have = ids[len(ids)-1]
	}
	for _, id := range ids {
		if id < 1 || id > known {
			return schemaError(have, known, true)
		}
	}
	if !olderOK && int64(len(ids)) != known {
		return schemaError(have, known, false)
	}
	return nil
}

// migrate applies every migration the store lacks, each in its own transaction with its row
// in the migrations table, after refusing a newer store. It runs on the writer before the
// authorizer is set, since a migration may change the audit triggers.
func migrate(conn *sqlite.Conn, now time.Time) error {
	if err := sqlitex.ExecuteTransient(conn, createMigrations, nil); err != nil {
		return err
	}
	q := connQuerier{conn}
	if err := checkSchema(q, true); err != nil {
		return err
	}
	ids, err := applied(q)
	if err != nil {
		return err
	}
	done := map[int64]bool{}
	for _, id := range ids {
		done[id] = true
	}
	for _, m := range migrations() {
		if done[m.id] {
			continue
		}
		if err := applyOne(conn, m, now); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	return nil
}

func applyOne(conn *sqlite.Conn, m migration, now time.Time) (err error) {
	end, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		return err
	}
	defer end(&err)
	if err := sqlitex.ExecuteScript(conn, m.sql, nil); err != nil {
		return err
	}
	return query1(conn, "INSERT INTO migrations (id, applied_at) VALUES (?, ?)",
		[]Value{Int(m.id), Text(FormatTime(now))}, nil)
}

// connQuerier reads on a bare connection.
type connQuerier struct{ conn *sqlite.Conn }

func (c connQuerier) Query(query string, args []Value, row func(*Row) error) error {
	return query1(c.conn, query, args, row)
}

// TimeLayout is how every time is stored: UTC RFC 3339 to the second, so text order is time
// order and the CLI prints it as stored.
const TimeLayout = time.RFC3339

// FormatTime is t in TimeLayout, in UTC.
func FormatTime(t time.Time) string { return t.UTC().Format(TimeLayout) }
