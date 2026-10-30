// SPDX-License-Identifier: GPL-3.0-or-later

package backup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

// lineExport stands in for audit.Export: one line per audit row.
func lineExport(q store.Querier, w io.Writer) (int, error) {
	n := 0
	err := q.Query("SELECT seq FROM audit_event ORDER BY seq", nil, func(r *store.Row) error {
		n++
		_, err := fmt.Fprintf(w, "{\"seq\":%d}\n", r.Int(0))
		return err
	})
	return n, err
}

func newRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	s, err := store.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	err = s.Write(func(tx *store.Tx) error {
		_, err := tx.Insert("INSERT INTO audit_event (time, kind, actor) VALUES ('2026-10-16T00:00:00Z', 'instance-started', 'instance')")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	var se *store.Error
	if !errors.As(err, &se) || se.Code != code {
		t.Fatalf("got %v, want a %s error", err, code)
	}
}

func TestSnapshotAndVerify(t *testing.T) {
	root := newRoot(t)
	_, err := Verify(root, "")
	wantCode(t, err, CodeNoSnapshot)

	// A leftover from an interrupted run is removed.
	if err := os.MkdirAll(filepath.Join(root, "backup"), 0o700); err != nil {
		t.Fatal(err)
	}
	leftover := filepath.Join(root, "backup", "store-20261015T000000.000Z.sqlite.tmp")
	if err := os.WriteFile(leftover, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Snapshot(root, lineExport)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if res.Result != "ok" || res.AuditEvents != 1 || !strings.HasPrefix(res.Snapshot, Dir(root)+"/store-") {
		t.Errorf("result %+v", res)
	}
	if _, err := os.Stat(leftover); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("leftover .tmp kept: %v", err)
	}
	latest, err := Latest(root)
	if err != nil || latest != res.Snapshot {
		t.Errorf("latest %q %v, want %q", latest, err, res.Snapshot)
	}
	if b, err := os.ReadFile(filepath.Join(root, "backup", AuditExportName)); err != nil || string(b) != "{\"seq\":1}\n" {
		t.Errorf("audit export %q %v", b, err)
	}

	v, err := Verify(root, "")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if v.Result != "ok" || v.Migration != store.Known() || v.Path != res.Snapshot {
		t.Errorf("verify %+v", v)
	}
	rows := map[string]int64{}
	for _, tr := range v.Tables {
		rows[tr.Table] = tr.Rows
	}
	if rows["audit_event"] != 1 || rows["migrations"] != store.Known() || len(rows) != 12 {
		t.Errorf("tables %+v", v.Tables)
	}
	// The snapshot is a rollback-journal file, so a read leaves nothing beside it.
	entries, _ := os.ReadDir(filepath.Join(root, "backup"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-wal") || strings.HasSuffix(e.Name(), "-shm") {
			t.Errorf("%s beside the snapshot", e.Name())
		}
	}

	b, err := os.ReadFile(res.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	truncated := filepath.Join(t.TempDir(), "truncated.sqlite")
	if err := os.WriteFile(truncated, b[:len(b)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Verify(root, truncated)
	wantCode(t, err, CodeIntegrity)
	t.Logf("truncated: %v", err)
	_, err = Verify(root, filepath.Join(t.TempDir(), "absent.sqlite"))
	wantCode(t, err, store.CodeNoStore)
}

func TestSnapshotKeepsTwoNewest(t *testing.T) {
	root := newRoot(t)
	base := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	var made []string
	for i := 0; i < 3; i++ {
		res, err := snapshot(root, lineExport, base.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, filepath.Base(res.Snapshot))
	}
	entries, _ := os.ReadDir(filepath.Join(root, "backup"))
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := []string{AuditExportName, LatestName, made[1], made[2]}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Errorf("backup/ holds %q, want %q", names, want)
	}
	if made[0] != "store-20261016T120000.000Z.sqlite" {
		t.Errorf("name %q", made[0])
	}
}

func TestVerifyRefusesAFileThatIsNotAStore(t *testing.T) {
	root := newRoot(t)
	empty := filepath.Join(t.TempDir(), "empty.sqlite")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Verify(root, empty)
	wantCode(t, err, CodeIntegrity)
	t.Logf("zero bytes: %v", err)
}

func TestSnapshotBusy(t *testing.T) {
	root := newRoot(t)
	lockPath := filepath.Join(root, "backup", LockName)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Snapshot(root, lineExport)
	wantCode(t, err, CodeBusy)
	if !strings.Contains(err.Error(), Dir(root)+"/"+LockName) {
		t.Errorf("busy message %q does not name the lock with forward slashes", err)
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(root, lineExport); err != nil {
		t.Fatalf("after the lock is gone: %v", err)
	}
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("lock kept after a snapshot: %v", err)
	}
}

func TestSlashError(t *testing.T) {
	err := slashError(&os.PathError{Op: "remove", Path: `C:\a\b`, Err: os.ErrPermission})
	if err.Error() != "remove C:/a/b: permission denied" {
		t.Errorf("%q", err)
	}
}
