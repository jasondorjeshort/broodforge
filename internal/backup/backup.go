// SPDX-License-Identifier: GPL-3.0-or-later

package backup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Error codes of snapshot and verify, beside the store's.
const (
	CodeIntegrity  = "integrity-check" // PRAGMA integrity_check did not return ok, or the file is not a database
	CodeNoSnapshot = "no-snapshot"     // backup/latest is missing or names no file
	CodeBusy       = "backup-busy"     // another snapshot holds backup/snapshot.lock
)

// LockName is the file a running snapshot holds in backup/.
const LockName = "snapshot.lock"

// Keep is how many snapshots stay after a snapshot (the companion's step 4).
const Keep = 2

// LatestName is the marker file naming the newest complete, checked snapshot.
const LatestName = "latest"

// Exporter writes the audit_event table of q as JSON lines and returns how many it wrote.
// The caller passes audit.Export; backup talks to store only (03-architecture.md).
type Exporter func(q store.Querier, w io.Writer) (int, error)

// AuditExportName is the audit export beside the snapshots.
const AuditExportName = "audit.jsonl"

const tmpSuffix = ".tmp"

// Dir is backup/ of the instance at root, with forward slashes.
func Dir(root string) string { return filepath.ToSlash(filepath.Join(root, "backup")) }

// snapshotName is `store-<UTC timestamp>.sqlite`; the milliseconds keep two snapshots in one
// second apart and the names sort in time order.
func snapshotName(t time.Time) string {
	return "store-" + t.UTC().Format("20060102T150405.000Z") + ".sqlite"
}

func isSnapshot(name string) bool {
	return strings.HasPrefix(name, "store-") && strings.HasSuffix(name, ".sqlite")
}

// SnapshotResult is what a snapshot made.
type SnapshotResult struct {
	Result      string `json:"result"` // "ok"
	Snapshot    string `json:"snapshot"`
	AuditEvents int    `json:"auditEvents"`
}

// Snapshot runs the companion's snapshot steps 1 to 4 from any process
// (decisions/0021-backup-and-restore.md, Snapshot): VACUUM INTO a .tmp file over a read-only
// connection, the audit export from that file, its integrity check, then the renames, the
// latest marker and the pruning to the two newest. Step 5, the checkpoint, is the bot's alone
// and is not done here. A leftover .tmp from an interrupted run is deleted first.
// Two runs at once are refused: the second finds backup/snapshot.lock and is backup-busy.
func Snapshot(root string, export Exporter) (SnapshotResult, error) {
	res, err := snapshot(root, export, time.Now())
	return res, slashError(err)
}

func snapshot(root string, export Exporter, now time.Time) (res SnapshotResult, err error) {
	live, err := store.OpenReadOnly(root)
	if err != nil {
		return res, err
	}
	dir := Dir(root)
	unlock, err := lock(dir)
	if err == nil {
		defer unlock()
		err = removeTmp(dir)
	}
	if err != nil {
		live.Close()
		return res, err
	}
	name := snapshotName(now)
	final := dir + "/" + name
	tmp := final + tmpSuffix
	auditTmp := dir + "/" + AuditExportName + tmpSuffix
	defer func() {
		if err != nil {
			_ = os.Remove(tmp)
			_ = os.Remove(auditTmp)
		}
	}()
	err = live.Query("VACUUM INTO ?", []store.Value{store.Text(filepath.FromSlash(tmp))}, nil)
	if cerr := live.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return res, err
	}

	snap, err := store.OpenFile(tmp)
	if err != nil {
		return res, err
	}
	n, err := exportAudit(snap, auditTmp, export)
	if err == nil {
		err = integrity(snap)
	}
	if cerr := snap.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return res, err
	}

	if err := os.Rename(filepath.FromSlash(tmp), filepath.FromSlash(final)); err != nil {
		return res, err
	}
	if err := os.Rename(filepath.FromSlash(auditTmp), filepath.FromSlash(dir+"/"+AuditExportName)); err != nil {
		return res, err
	}
	if err := writeLatest(dir, name); err != nil {
		return res, err
	}
	if err := prune(dir); err != nil {
		return res, err
	}
	return SnapshotResult{Result: "ok", Snapshot: final, AuditEvents: n}, nil
}

func exportAudit(q store.Querier, path string, export Exporter) (int, error) {
	f, err := os.OpenFile(filepath.FromSlash(path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, err
	}
	n, err := export(q, f)
	if err == nil {
		err = f.Sync()
	}
	return n, errors.Join(err, f.Close())
}

// lock creates backup/ if needed and takes backup/snapshot.lock with O_EXCL; the returned
// function releases it. A lock left by a crashed run blocks snapshots until it is deleted,
// which the refusal says.
func lock(dir string) (func(), error) {
	if err := os.MkdirAll(filepath.FromSlash(dir), 0o700); err != nil {
		return nil, err
	}
	path := dir + "/" + LockName
	f, err := os.OpenFile(filepath.FromSlash(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil, &store.Error{Code: CodeBusy, Message: "another snapshot holds " + path + "; delete it if none is running"}
	}
	if err != nil {
		return nil, err
	}
	_, werr := fmt.Fprintf(f, "%d\n", os.Getpid())
	if err := errors.Join(werr, f.Close()); err != nil {
		_ = os.Remove(filepath.FromSlash(path))
		return nil, err
	}
	return func() { _ = os.Remove(filepath.FromSlash(path)) }, nil
}

// slashError prints the paths of a file-system error with forward slashes, as every path the
// CLI prints is.
func slashError(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return &os.PathError{Op: pe.Op, Path: filepath.ToSlash(pe.Path), Err: pe.Err}
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return &os.LinkError{Op: le.Op, Old: filepath.ToSlash(le.Old), New: filepath.ToSlash(le.New), Err: le.Err}
	}
	return err
}

// removeTmp deletes what an interrupted snapshot left.
func removeTmp(dir string) error {
	entries, err := os.ReadDir(filepath.FromSlash(dir))
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), tmpSuffix) {
			if err := os.Remove(filepath.Join(filepath.FromSlash(dir), e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeLatest replaces backup/latest with the snapshot's file name, through a rename so the
// marker always names a complete snapshot.
func writeLatest(dir, name string) error {
	path := filepath.Join(filepath.FromSlash(dir), LatestName)
	if err := os.WriteFile(path+tmpSuffix, []byte(name+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(path+tmpSuffix, path)
}

// prune deletes every snapshot older than the Keep newest.
func prune(dir string) error {
	entries, err := os.ReadDir(filepath.FromSlash(dir))
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && isSnapshot(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for i := 0; i < len(names)-Keep; i++ {
		if err := os.Remove(filepath.Join(filepath.FromSlash(dir), names[i])); err != nil {
			return err
		}
	}
	return nil
}

// Latest is the path of the snapshot backup/latest names.
func Latest(root string) (string, error) {
	dir := Dir(root)
	b, err := os.ReadFile(filepath.FromSlash(dir + "/" + LatestName))
	if errors.Is(err, os.ErrNotExist) {
		return "", &store.Error{Code: CodeNoSnapshot, Message: "no " + dir + "/" + LatestName}
	}
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(b))
	if name == "" || strings.ContainsAny(name, `/\`) || !isSnapshot(name) {
		return "", &store.Error{Code: CodeNoSnapshot, Message: dir + "/" + LatestName + " names no snapshot"}
	}
	return dir + "/" + name, nil
}

// integrity runs PRAGMA integrity_check; anything but the one row "ok" is CodeIntegrity with
// the result (its first rows) as the message.
func integrity(q store.Querier) error {
	var rows []string
	err := q.Query("PRAGMA integrity_check", nil, func(r *store.Row) error {
		rows = append(rows, r.Text(0))
		return nil
	})
	if err != nil {
		return &store.Error{Code: CodeIntegrity, Message: err.Error()}
	}
	if len(rows) == 1 && rows[0] == "ok" {
		return nil
	}
	if len(rows) > 3 {
		rows = append(rows[:3], "...")
	}
	return &store.Error{Code: CodeIntegrity, Message: strings.Join(rows, "; ")}
}

// TableRows is one table of a verified file and its row count.
type TableRows struct {
	Table string `json:"table"`
	Rows  int64  `json:"rows"`
}

// VerifyResult is what verify found: the integrity result, the migration id and the row
// count of every table.
type VerifyResult struct {
	Result    string      `json:"result"` // "ok"
	Path      string      `json:"path"`
	Migration int64       `json:"migration"`
	Tables    []TableRows `json:"tables"`
}

// Verify opens a snapshot read-only, default the one backup/latest names, runs the integrity
// check and reads the migration id and the row count per table (the companion's Verify). It
// proves the file is readable, not that a restore works. path "" means latest.
func Verify(root, path string) (VerifyResult, error) {
	res, err := verify(root, path)
	return res, slashError(err)
}

func verify(root, path string) (VerifyResult, error) {
	if path == "" {
		var err error
		if path, err = Latest(root); err != nil {
			return VerifyResult{}, err
		}
	}
	path = filepath.ToSlash(path)
	r, err := store.OpenFile(path)
	var se *store.Error
	if errors.As(err, &se) && se.Code == store.CodeStoreOpen {
		return VerifyResult{}, &store.Error{Code: CodeIntegrity, Message: se.Message}
	}
	if err != nil {
		return VerifyResult{}, err
	}
	defer r.Close()
	if err := integrity(r); err != nil {
		return VerifyResult{}, err
	}
	res := VerifyResult{Result: "ok", Path: path, Tables: []TableRows{}}
	if res.Migration, err = store.SchemaID(r); err != nil {
		return res, err
	}
	if res.Migration == 0 { // an empty file, or a database that is not a store
		return VerifyResult{}, &store.Error{Code: CodeIntegrity, Message: path + ": no migrations recorded; not a store"}
	}
	var tables []string
	err = r.Query("SELECT name FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite\\_%' ESCAPE '\\' ORDER BY name", nil,
		func(row *store.Row) error {
			tables = append(tables, row.Text(0))
			return nil
		})
	if err != nil {
		return res, err
	}
	for _, t := range tables {
		var n int64
		q := `SELECT count(*) FROM "` + strings.ReplaceAll(t, `"`, `""`) + `"`
		if err := r.Query(q, nil, func(row *store.Row) error { n = row.Int(0); return nil }); err != nil {
			return res, &store.Error{Code: CodeIntegrity, Message: t + ": " + err.Error()}
		}
		res.Tables = append(res.Tables, TableRows{Table: t, Rows: n})
	}
	return res, nil
}
