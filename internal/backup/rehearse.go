// SPDX-License-Identifier: GPL-3.0-or-later

package backup

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Rehearsal is one `backup rehearse` (decisions/0021-backup-and-restore.md, Rehearse). backup
// talks to store only, so the caller supplies what the config and spawn components do.
type Rehearsal struct {
	Root string // the instance whose snapshot is restored
	Path string // the snapshot; "" is the one backup/latest names
	Keep bool   // keep the scratch root on success too
	// CopyConfig copies config/ of src into the scratch root dst as a scratch root named name
	// (config.CopyForScratch).
	CopyConfig func(src, dst, name string) error
	// RemoveTree is the spawn backend's worktree-remove, which deleting a scratch root runs on
	// every session directory.
	RemoveTree func(sessionDir string) error
}

// RehearseResult is what a rehearsal read back from the restored copy: verify's result, the
// number of missing worktrees reconciliation reports, and the scratch root when it was kept.
type RehearseResult struct {
	VerifyResult
	ReconcileMissing int    `json:"reconcileMissing"`
	Root             string `json:"root,omitempty"`
}

// Rehearse runs the companion's steps: (1) a new scratch root with the source's config/ and the
// snapshot as data/store.sqlite; (2) the migrations on the copy (store.Create, which refuses a
// snapshot newer than the binary); (3) reconciliation in report-only mode; (4) the integrity
// check, the migration id and the rows per table of the restored file, and one record read
// from every table. Then the root is deleted unless Keep. On any failure the root is kept and
// returned with the error, so the caller can print it; "" when none was made.
func Rehearse(r Rehearsal) (res RehearseResult, kept string, err error) {
	res, kept, err = rehearse(r)
	return res, kept, slashError(err)
}

func rehearse(r Rehearsal) (RehearseResult, string, error) {
	snap := r.Path
	if snap == "" {
		var err error
		if snap, err = Latest(r.Root); err != nil {
			return RehearseResult{}, "", err
		}
	}
	snap = filepath.ToSlash(snap)
	if fi, err := os.Stat(filepath.FromSlash(snap)); err != nil || !fi.Mode().IsRegular() {
		return RehearseResult{}, "", &store.Error{Code: CodeNoSnapshot, Message: snap + ": no such snapshot file"}
	}
	sc, err := store.NewScratch()
	if err != nil {
		return RehearseResult{}, "", err
	}
	res, err := restore(r, sc, snap)
	if err != nil {
		return RehearseResult{}, sc.Root, err
	}
	if r.Keep {
		res.Root = sc.Root
		return res, sc.Root, nil
	}
	if err := store.DeleteScratch(sc.Root, store.FixedName(sc.InstanceName), r.RemoveTree); err != nil {
		return RehearseResult{}, sc.Root, err
	}
	return res, "", nil
}

func restore(r Rehearsal, sc store.Scratch, snap string) (RehearseResult, error) {
	if err := r.CopyConfig(r.Root, sc.Root, sc.InstanceName); err != nil {
		return RehearseResult{}, err
	}
	dst := store.Path(sc.Root)
	if err := copyFile(snap, dst); err != nil {
		return RehearseResult{}, err
	}
	st, err := store.Create(sc.Root)
	if err != nil {
		return RehearseResult{}, err
	}
	if err := st.Close(); err != nil {
		return RehearseResult{}, err
	}
	v, err := verify(sc.Root, dst)
	if err != nil {
		return RehearseResult{}, err
	}
	rd, err := store.OpenReadOnly(sc.Root)
	if err != nil {
		return RehearseResult{}, err
	}
	defer rd.Close()
	for _, t := range v.Tables {
		q := `SELECT * FROM "` + strings.ReplaceAll(t.Table, `"`, `""`) + `" LIMIT 1`
		if err := rd.Query(q, nil, nil); err != nil {
			return RehearseResult{}, &store.Error{Code: CodeIntegrity, Message: t.Table + ": " + err.Error()}
		}
	}
	findings, err := store.ReconcileFrom(rd, sc.Root, r.Root)
	if err != nil {
		return RehearseResult{}, err
	}
	res := RehearseResult{VerifyResult: v}
	res.Path = snap // the snapshot restored, not the copy, which is deleted
	for _, f := range findings {
		if f.Kind == store.FindingMissing {
			res.ReconcileMissing++
		}
	}
	return res, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(filepath.FromSlash(dst)), 0o700); err != nil {
		return err
	}
	in, err := os.Open(filepath.FromSlash(src))
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(filepath.FromSlash(dst), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, cerr := io.Copy(out, in)
	if cerr == nil {
		cerr = out.Sync()
	}
	return errors.Join(cerr, out.Close())
}
