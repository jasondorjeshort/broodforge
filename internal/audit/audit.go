// SPDX-License-Identifier: GPL-3.0-or-later

package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Kind is an audit event kind: 03's and 02's kinds in kebab-case (05-debug-interface.md,
// Usage and audit queries). The constants are the kinds the design names so far; a pass that
// writes a new kind adds its constant here.
type Kind string

const (
	KindInstanceStarted      Kind = "instance-started"
	KindConfigurationChanged Kind = "configuration-changed"
	KindConfigurationInvalid Kind = "configuration-invalid"
	KindRegistrationChanged  Kind = "registration-changed"
	KindCredentialAdded      Kind = "credential-added"
	KindCredentialRemoved    Kind = "credential-removed"
	KindSessionStarted       Kind = "session-started"
	KindSessionEnded         Kind = "session-ended"
	KindSessionInterrupted   Kind = "session-interrupted"
	KindPermissionDecision   Kind = "permission-decision"
	KindHookBypassed         Kind = "hook-bypassed"
	KindActionRefused        Kind = "action-refused"
	KindCommit               Kind = "commit"
	KindPush                 Kind = "push"
	KindPullRequestOpened    Kind = "pull-request-opened"
	KindHandoffRefused       Kind = "handoff-refused"
	KindWorktreeForgotten    Kind = "worktree-forgotten"
	KindWorktreeAdopted      Kind = "worktree-adopted"
	KindModeChanged          Kind = "mode-changed"
	KindModelChanged         Kind = "model-changed"
	KindEffortChanged        Kind = "effort-changed"
	KindParticipantInvited   Kind = "participant-invited"
)

// Phase marks an action outside the store: attempted before, done or failed after (ADR
// 0021). An event that records a store change has no phase.
type Phase string

const (
	PhaseNone      Phase = ""
	PhaseAttempted Phase = "attempted"
	PhaseDone      Phase = "done"
	PhaseFailed    Phase = "failed"
)

var kebab = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// ValidKind reports whether s is a kind's form: one kebab-case token, such as session-started.
func ValidKind(s string) bool { return kebab.MatchString(s) }

// Event is one audit_event row.
type Event struct {
	Seq          int64
	Time         time.Time
	Kind         Kind
	Phase        Phase
	Actor        string // a Discord user id, "policy", "host" or "instance" (02-domain.md)
	Session      string // "" for none
	Registration string // "" for none
	Details      json.RawMessage
}

// eventJSON is the one JSON form of an event: `audit --json` and backup/audit.jsonl.
type eventJSON struct {
	Seq          int64           `json:"seq"`
	Time         string          `json:"time"`
	Kind         string          `json:"kind"`
	Phase        *string         `json:"phase"`
	Actor        string          `json:"actor"`
	Session      *string         `json:"session"`
	Registration *string         `json:"registration"`
	Details      json.RawMessage `json:"details"`
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// MarshalJSON prints every column, null for an absent one, details as the stored object.
func (e Event) MarshalJSON() ([]byte, error) {
	details := e.Details
	if len(details) == 0 {
		details = json.RawMessage("{}")
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(eventJSON{Seq: e.Seq, Time: store.FormatTime(e.Time), Kind: string(e.Kind),
		Phase: optional(string(e.Phase)), Actor: e.Actor, Session: optional(e.Session),
		Registration: optional(e.Registration), Details: details})
	return bytes.TrimSpace(b.Bytes()), err
}

// Sink receives each event after the store transaction that wrote it commits. The table is
// the authoritative sink and is written by Writer.Append itself; a Sink is a best-effort
// copy (ADR 0021: the optional file sink).
type Sink interface {
	Write(Event) error
}

// FileSink appends one JSON line per event to a file, best effort.
type FileSink struct {
	mu   sync.Mutex
	path string
}

// NewFileSink returns a sink appending to path, the path `audit.fileSink` names.
func NewFileSink(path string) *FileSink { return &FileSink{path: path} }

// Write appends the event as one line.
func (f *FileSink) Write(e Event) error {
	line, err := e.MarshalJSON()
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	file, err := os.OpenFile(f.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	_, werr := file.Write(append(line, '\n'))
	return errors.Join(werr, file.Close())
}

// Writer is the audit writer: it inserts each event in the caller's store transaction, so
// the event commits or rolls back with the change it records, then hands it to the optional
// sink after the commit.
type Writer struct {
	sink    Sink
	now     func() time.Time
	OnError func(error) // a sink failure; nil drops it (best effort)
}

// NewWriter returns a writer with an optional sink (nil for none).
func NewWriter(sink Sink) *Writer { return &Writer{sink: sink, now: time.Now} }

// Append writes ev in tx and returns it with its seq and time.
func (w *Writer) Append(tx *store.Tx, ev Event) (Event, error) {
	if !ValidKind(string(ev.Kind)) {
		return ev, fmt.Errorf("audit: kind %q is not kebab-case", ev.Kind)
	}
	switch ev.Phase {
	case PhaseNone, PhaseAttempted, PhaseDone, PhaseFailed:
	default:
		return ev, fmt.Errorf("audit: unknown phase %q", ev.Phase)
	}
	if ev.Actor == "" {
		return ev, errors.New("audit: an event needs an actor")
	}
	if len(ev.Details) == 0 {
		ev.Details = json.RawMessage("{}")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, ev.Details); err != nil {
		return ev, fmt.Errorf("audit: details: %w", err)
	}
	ev.Details = compact.Bytes()
	ev.Time = w.now().UTC().Truncate(time.Second)
	seq, err := tx.Insert(`INSERT INTO audit_event (time, kind, phase, actor, session_id, registration_id, details)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		store.Text(store.FormatTime(ev.Time)), store.Text(string(ev.Kind)), store.OptText(string(ev.Phase)),
		store.Text(ev.Actor), store.OptText(ev.Session), store.OptText(ev.Registration), store.Text(string(ev.Details)))
	if err != nil {
		return ev, err
	}
	ev.Seq = seq
	if w.sink != nil {
		committed := ev
		tx.OnCommit(func() {
			if err := w.sink.Write(committed); err != nil && w.OnError != nil {
				w.OnError(err)
			}
		})
	}
	return ev, nil
}

// Filter narrows a query (05-debug-interface.md: `audit [--since-seq <n>] [--kind <kind>[,<kind>]]
// [--session <id>] [--limit <n>]`).
type Filter struct {
	SinceSeq int64  // only events after this seq; 0 for all
	Kinds    []Kind // any of these; none for every kind
	Session  string
	// Limit is the number of events; 0 or negative for all. Without SinceSeq the newest Limit
	// events are returned, with it the first Limit after SinceSeq; either way in seq order.
	Limit int
}

const columns = "seq, time, kind, coalesce(phase, ''), actor, coalesce(session_id, ''), coalesce(registration_id, ''), details"

func scan(r *store.Row) (Event, error) {
	t, err := time.Parse(store.TimeLayout, r.Text(1))
	if err != nil {
		return Event{}, fmt.Errorf("audit_event %d: time: %w", r.Int(0), err)
	}
	details := json.RawMessage(r.Text(7))
	if !json.Valid(details) {
		b, _ := json.Marshal(r.Text(7))
		details = b
	}
	return Event{Seq: r.Int(0), Time: t, Kind: Kind(r.Text(2)), Phase: Phase(r.Text(3)), Actor: r.Text(4),
		Session: r.Text(5), Registration: r.Text(6), Details: details}, nil
}

// Query returns the events the filter selects, in seq order.
func Query(q store.Querier, f Filter) ([]Event, error) {
	var conds []string
	var args []store.Value
	if f.SinceSeq > 0 {
		conds = append(conds, "seq > ?")
		args = append(args, store.Int(f.SinceSeq))
	}
	if len(f.Kinds) > 0 {
		marks := make([]string, len(f.Kinds))
		for i, k := range f.Kinds {
			marks[i] = "?"
			args = append(args, store.Text(string(k)))
		}
		conds = append(conds, "kind IN ("+strings.Join(marks, ", ")+")")
	}
	if f.Session != "" {
		conds = append(conds, "session_id = ?")
		args = append(args, store.Text(f.Session))
	}
	query := "SELECT " + columns + " FROM audit_event"
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	newest := f.SinceSeq <= 0 && f.Limit > 0
	if newest {
		query += " ORDER BY seq DESC"
	} else {
		query += " ORDER BY seq"
	}
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, store.Int(int64(f.Limit)))
	}
	var out []Event
	err := q.Query(query, args, func(r *store.Row) error {
		ev, err := scan(r)
		out = append(out, ev)
		return err
	})
	if err != nil {
		return nil, err
	}
	if newest {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out, nil
}

// Export writes every event in seq order as one JSON object per line, every column (the
// companion's snapshot step 2), and returns how many it wrote.
func Export(q store.Querier, w io.Writer) (int, error) {
	n := 0
	err := q.Query("SELECT "+columns+" FROM audit_event ORDER BY seq", nil, func(r *store.Row) error {
		ev, err := scan(r)
		if err != nil {
			return err
		}
		line, err := ev.MarshalJSON()
		if err != nil {
			return err
		}
		if _, err := w.Write(append(line, '\n')); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}
