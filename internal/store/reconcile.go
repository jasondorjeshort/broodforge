// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Session states that reconciliation leaves alone; every other state is a session the bot
// was running when it stopped (02-domain.md, Session).
const (
	SessionEnded       = "ended"
	SessionInterrupted = "interrupted"
	WorktreeRemoved    = "removed"
)

// Finding kinds of a reconciliation report, the first column of `reconcile` (ADR 0021,
// Reconciliation at start).
const (
	FindingInterrupt  = "interrupt"  // a session in a running state: marked interrupted
	FindingMissing    = "missing"    // a worktree whose recorded path is absent: missing_since set
	FindingPresent    = "present"    // a worktree marked missing whose path is back: missing_since cleared
	FindingUnrecorded = "unrecorded" // a directory under sessions/ no session record names: listed only
)

// Finding is one line of a reconciliation report.
type Finding struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject"`           // the session id, worktree path or directory
	Session string `json:"session,omitempty"` // the session it belongs to, "" for none (absent in JSON, - in text)
	Detail  string `json:"detail,omitempty"`  // the session's state, or missing_since; "" for none
}

// Reconcile computes ADR 0021's reconciliation without writing anything: the report-only
// mode of `reconcile` and of the backup rehearsal. The bot's start applies the same findings
// (P10). A worktree's state is never changed by a path check; only missing_since moves.
func Reconcile(q Querier, root string) ([]Finding, error) { return ReconcileFrom(q, root, root) }

// ReconcileFrom is Reconcile of a store restored at root from an instance at recordedRoot (the
// backup rehearsal): a recorded worktree path under recordedRoot is checked at the same place
// under root, so the source's own trees never count as present. Findings name the recorded
// paths. A path outside recordedRoot is checked as recorded.
func ReconcileFrom(q Querier, root, recordedRoot string) ([]Finding, error) {
	from := recordedRoot
	if abs, err := filepath.Abs(recordedRoot); err == nil {
		from = abs
	}
	from = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(from)), "/")
	var out []Finding
	sessions := map[string]bool{}
	err := q.Query("SELECT id, state FROM session ORDER BY id", nil, func(r *Row) error {
		id, state := r.Text(0), r.Text(1)
		sessions[id] = true
		if state != SessionEnded && state != SessionInterrupted {
			out = append(out, Finding{Kind: FindingInterrupt, Subject: id, Session: id, Detail: state})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = q.Query("SELECT path, coalesce(session_id, ''), coalesce(missing_since, '') FROM worktree WHERE state != ? ORDER BY path",
		[]Value{Text(WorktreeRemoved)}, func(r *Row) error {
			path, session, since := r.Text(0), r.Text(1), r.Text(2)
			_, statErr := os.Stat(filepath.FromSlash(rebase(path, from, root)))
			switch {
			case errors.Is(statErr, os.ErrNotExist):
				out = append(out, Finding{Kind: FindingMissing, Subject: path, Session: session, Detail: since})
			case statErr == nil && since != "":
				out = append(out, Finding{Kind: FindingPresent, Subject: path, Session: session, Detail: since})
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	var unrecorded []string
	for _, e := range entries {
		if e.IsDir() && !sessions[e.Name()] {
			unrecorded = append(unrecorded, filepath.ToSlash(filepath.Join(dir, e.Name())))
		}
	}
	sort.Strings(unrecorded)
	for _, p := range unrecorded {
		out = append(out, Finding{Kind: FindingUnrecorded, Subject: p})
	}
	return out, nil
}

// rebase maps a recorded path under from to the same place under to; from is absolute,
// slash-separated and compared without regard to case, since a Windows path may be recorded in
// another case than given. Any other path is returned as it is.
func rebase(path, from, to string) string {
	p := filepath.ToSlash(path)
	if len(p) > len(from) && p[len(from)] == '/' && strings.EqualFold(p[:len(from)], from) {
		return filepath.ToSlash(filepath.Join(to, p[len(from)+1:]))
	}
	return path
}
