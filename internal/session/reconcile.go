// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Reconcile is ADR 0021's reconciliation at start (03 Startup, step 7): store.Reconcile's
// findings applied, each in its own short transaction with its audit event. A session in a
// running state becomes interrupted (R-17); a worktree's missing_since is set or cleared, its
// state never changed by a path check; an unrecorded directory is listed only. It returns the
// findings, the start-up report.
func Reconcile(st *store.Store, aud *audit.Writer, root string, now time.Time) ([]store.Finding, error) {
	findings, err := store.Reconcile(st, root)
	if err != nil {
		return nil, err
	}
	stamp := store.FormatTime(now)
	for _, f := range findings {
		var werr error
		switch f.Kind {
		case store.FindingInterrupt:
			werr = st.Write(func(tx *store.Tx) error {
				if _, err := tx.Exec("UPDATE session SET state = ?, ended_at = coalesce(ended_at, ?) WHERE id = ?",
					store.Text(StateInterrupted), store.Text(stamp), store.Text(f.Subject)); err != nil {
					return err
				}
				_, err := aud.Append(tx, audit.Event{Kind: audit.KindSessionInterrupted, Actor: ActorInstance, Session: f.Subject,
					Details: details("reason", "reconciliation", "state", f.Detail)})
				return err
			})
		case store.FindingMissing:
			werr = st.Write(func(tx *store.Tx) error {
				_, err := tx.Exec("UPDATE worktree SET missing_since = ? WHERE path = ? AND missing_since IS NULL",
					store.Text(stamp), store.Text(f.Subject))
				return err
			})
		case store.FindingPresent:
			werr = st.Write(func(tx *store.Tx) error {
				_, err := tx.Exec("UPDATE worktree SET missing_since = NULL WHERE path = ?", store.Text(f.Subject))
				return err
			})
		}
		if werr != nil {
			return findings, werr
		}
	}
	return findings, nil
}
