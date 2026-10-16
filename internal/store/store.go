// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// BusyTimeout is how long a connection waits on a lock before SQLITE_BUSY (ADR 0021).
const BusyTimeout = 5 * time.Second

// ReadPoolSize is the size of the CLI's read pool (ADR 0023: a small read pool).
const ReadPoolSize = 2

// AuditTable is the append-only table the writer's authorizer guards.
const AuditTable = "audit_event"

// Error codes of the store, printed by the CLI as `error: <code>: <message>`.
const (
	CodeNoStore     = "no-store"     // data/store.sqlite does not exist
	CodeStoreSchema = "store-schema" // the store's migrations are not the binary's
	CodeStoreShm    = "store-shm"    // a read-only open cannot use the -shm file (ADR 0021)
	CodeStoreOpen   = "store-open"   // the file cannot be opened as a database
)

// Error is a store refusal with a stable code. The store does not import the control
// package; the CLI turns an Error into its error line.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Path is data/store.sqlite of the instance at root, with forward slashes on every host.
func Path(root string) string { return filepath.ToSlash(filepath.Join(root, "data", "store.sqlite")) }

// pragmas are set on every connection at open, since PRAGMAs are per connection (ADR 0023).
// journal_mode=WAL is persistent in the file and is set by the writer only: a read-only
// connection cannot change it and reads it from the file.
// Each runs on its own: synchronous cannot change inside the savepoint a script would open.
var pragmas = []string{"PRAGMA synchronous=FULL", "PRAGMA foreign_keys=ON"}

func setPragmas(conn *sqlite.Conn) error {
	conn.SetBusyTimeout(BusyTimeout)
	for _, p := range pragmas {
		if err := sqlitex.ExecuteTransient(conn, p, nil); err != nil {
			return err
		}
	}
	return nil
}

// openConn opens a read-write connection to path, creating the file if missing, in WAL mode
// and with the per-connection settings: the writer's, and a test's plain connection.
func openConn(path string) (*sqlite.Conn, error) {
	conn, err := sqlite.OpenConn(path, sqlite.OpenReadWrite|sqlite.OpenCreate|sqlite.OpenWAL)
	if err != nil {
		return nil, openError(path, err)
	}
	if err := setPragmas(conn); err != nil {
		conn.Close()
		return nil, openError(path, err)
	}
	return conn, nil
}

// uriPath makes a file path usable in a file: URI: a Windows drive path gets a leading slash
// and the characters URIs reserve are escaped.
func uriPath(path string) string {
	p := filepath.ToSlash(path)
	p = strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(p)
	if len(p) >= 2 && p[1] == ':' {
		p = "/" + p
	}
	return p
}

// openError maps an open or first-statement failure to a store code.
func openError(path string, err error) error {
	code := sqlite.ErrCode(err)
	if code == sqlite.ResultReadOnlyCantInit || code == sqlite.ResultReadOnlyCantLock ||
		code == sqlite.ResultReadOnlyRecovery || code.ToPrimary() == sqlite.ResultCantOpen {
		if _, serr := os.Stat(path + "-shm"); serr == nil {
			return &Error{CodeStoreShm, path + "-shm: " + err.Error()}
		}
	}
	if p := code.ToPrimary(); p == sqlite.ResultNotADB || p == sqlite.ResultCorrupt {
		return &Error{CodeStoreOpen, path + ": " + err.Error()}
	}
	return fmt.Errorf("open %s: %w", path, err)
}

// auditGuard is the writer's authorizer (ADR 0021's hardening, taken by ADR 0023): it denies
// UPDATE, DELETE, DROP TABLE and ALTER TABLE on audit_event and DROP TRIGGER of a trigger on
// it, and PRAGMA writable_schema, without which SQLite itself refuses a direct write to the
// schema table ("may not be modified"). The schema table is not denied by name: SQLite
// authorizes every CREATE as an INSERT into it and every DROP as a DELETE. The bot's own code
// then cannot change the log even through a statement the triggers would not see. Migrations
// run before it is set.
var auditGuard = sqlite.AuthorizeFunc(func(a sqlite.Action) sqlite.AuthResult {
	// Every other action is allowed; the guard names only what it denies.
	//exhaustive:ignore
	switch a.Type() {
	case sqlite.OpUpdate, sqlite.OpDelete, sqlite.OpDropTrigger, sqlite.OpDropTable, sqlite.OpAlterTable:
		if strings.EqualFold(a.Table(), AuditTable) {
			return sqlite.AuthResultDeny
		}
	case sqlite.OpPragma:
		if strings.EqualFold(a.Pragma(), "writable_schema") {
			return sqlite.AuthResultDeny
		}
	}
	return sqlite.AuthResultOK
})

// ErrClosed is a Write or Query on a store after Close.
var ErrClosed = errors.New("store: closed")

// Store is the bot's one writer connection (ADR 0021: one writer). It is safe for concurrent
// use: calls are serialised. Create opens it.
type Store struct {
	mu   sync.Mutex
	conn *sqlite.Conn
	path string
}

// Path is the file the store writes.
func (s *Store) Path() string { return s.path }

// Close closes the writer connection.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil
	}
	err := s.conn.Close()
	s.conn = nil
	return err
}

// Write runs fn in one IMMEDIATE transaction: committed when fn returns nil, rolled back
// otherwise. Functions given to tx.OnCommit run after the commit, in order.
func (s *Store) Write(fn func(tx *Tx) error) error {
	s.mu.Lock()
	if s.conn == nil {
		s.mu.Unlock()
		return ErrClosed
	}
	tx := &Tx{conn: s.conn}
	err := transaction(tx, fn)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	for _, f := range tx.onCommit {
		f()
	}
	return nil
}

func transaction(tx *Tx, fn func(tx *Tx) error) (err error) {
	end, err := sqlitex.ImmediateTransaction(tx.conn)
	if err != nil {
		return err
	}
	defer end(&err)
	return fn(tx)
}

// Query runs one read on the writer connection, for the bot's own reads.
func (s *Store) Query(query string, args []Value, row func(*Row) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return ErrClosed
	}
	return query1(s.conn, query, args, row)
}

// Tx is one write transaction.
type Tx struct {
	conn     *sqlite.Conn
	onCommit []func()
}

// Exec runs one statement and returns the rows it changed.
func (tx *Tx) Exec(query string, args ...Value) (int, error) {
	if err := query1(tx.conn, query, args, nil); err != nil {
		return 0, err
	}
	return tx.conn.Changes(), nil
}

// Insert runs one INSERT and returns the new row id.
func (tx *Tx) Insert(query string, args ...Value) (int64, error) {
	if err := query1(tx.conn, query, args, nil); err != nil {
		return 0, err
	}
	return tx.conn.LastInsertRowID(), nil
}

// Query reads inside the transaction.
func (tx *Tx) Query(query string, args []Value, row func(*Row) error) error {
	return query1(tx.conn, query, args, row)
}

// OnCommit runs f after the transaction commits, never if it rolls back: how a best-effort
// side effect (the audit file sink) follows the store write it belongs to.
func (tx *Tx) OnCommit(f func()) { tx.onCommit = append(tx.onCommit, f) }

// Querier is anything that reads: the writer, a transaction, the read pool, a snapshot.
type Querier interface {
	Query(query string, args []Value, row func(*Row) error) error
}

// Reader is the CLI's read-only access to a store file: a small pool of read-only
// connections. Each Query holds its own read transaction, never one per command (ADR 0021).
type Reader struct {
	pool *sqlitex.Pool
	path string
}

// Path is the file the reader reads.
func (r *Reader) Path() string { return r.path }

// Close closes every connection of the pool.
func (r *Reader) Close() error { return r.pool.Close() }

// Query runs one statement on a pooled read-only connection.
func (r *Reader) Query(query string, args []Value, row func(*Row) error) error {
	conn, err := r.pool.Take(context.Background())
	if err != nil {
		return err
	}
	defer r.pool.Put(conn)
	return query1(conn, query, args, row)
}

// OpenReadOnly opens the store of the instance at root for reading, as every CLI read does
// without the bot (ADR 0021): it never creates or migrates the file and refuses a store whose
// migrations are not exactly the binary's.
func OpenReadOnly(root string) (*Reader, error) {
	path := Path(root)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, &Error{CodeNoStore, "no store at " + path}
	}
	r, err := OpenFile(path)
	if err != nil {
		return nil, err
	}
	if err := checkSchema(r, false); err != nil {
		r.Close()
		return nil, err
	}
	return r, nil
}

// OpenFile opens any store file read-only (a snapshot, a copy) without checking its schema,
// for backup verify. A missing file is no-store.
func OpenFile(path string) (*Reader, error) {
	path = filepath.ToSlash(path)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, &Error{CodeNoStore, "no store at " + path}
	}
	pool, err := sqlitex.NewPool("file:"+uriPath(path)+"?mode=ro", sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly | sqlite.OpenURI,
		PoolSize: ReadPoolSize,
		PrepareConn: func(conn *sqlite.Conn) error {
			return setPragmas(conn)
		},
	})
	if err != nil {
		return nil, openError(path, err)
	}
	r := &Reader{pool: pool, path: path}
	// The first statement is where a missing -shm or a file that is not a database shows.
	if err := r.Query("SELECT count(*) FROM sqlite_schema", nil, nil); err != nil {
		r.Close()
		return nil, openError(path, err)
	}
	return r, nil
}
