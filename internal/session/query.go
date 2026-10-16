// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The reads below take any store.Querier: the bot's writer, or the CLI's read-only store when
// no bot runs (ADR 0021), so both print the same records.

// ListRow is one line of `session list`: `<id>  <state>  <channel>  <driver>  <mode>  <reach>
// <turns>  <started at>`.
type ListRow struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Channel   string `json:"channel"`
	Driver    string `json:"driver"`
	Mode      string `json:"mode"`
	Reach     string `json:"reach"`
	Turns     int64  `json:"turns"`
	StartedAt string `json:"startedAt"`
}

// ListColumns are ListRow's text columns.
var ListColumns = []string{"id", "state", "channel", "driver", "mode", "reach", "turns", "startedAt"}

// List is every session, or those in one state, by id.
func List(q store.Querier, state string) ([]ListRow, error) {
	query := `SELECT s.id, s.state, s.channel_id, s.driver_id, s.mode, s.reach,
    (SELECT count(*) FROM turn t WHERE t.session_id = s.id), s.started_at FROM session s`
	var args []store.Value
	if state != "" {
		query += " WHERE s.state = ?"
		args = append(args, store.Text(state))
	}
	query += " ORDER BY s.rowid"
	out := []ListRow{}
	err := q.Query(query, args, func(r *store.Row) error {
		out = append(out, ListRow{ID: r.Text(0), State: r.Text(1), Channel: r.Text(2), Driver: r.Text(3), Mode: r.Text(4),
			Reach: r.Text(5), Turns: r.Int(6), StartedAt: r.Text(7)})
		return nil
	})
	return out, err
}

// Participant is one participant of a session.
type Participant struct {
	User     string `json:"user"`
	Role     string `json:"role"`
	JoinedAt string `json:"joinedAt"`
}

// Worktree is the session's tree.
type Worktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	State  string `json:"state"`
}

// Pending is the call a waiting session waits on (P10 half b fills it).
type Pending struct {
	ToolUseID string `json:"toolUseId"`
	Tool      string `json:"tool"`
	Target    string `json:"target"`
	Since     string `json:"since"`
	Until     string `json:"until"`
	// Dropped is how many other calls of its message the engine dropped (C28): 04 says so.
	Dropped int `json:"dropped"`
}

// Detail is `session show`.
type Detail struct {
	ID              string        `json:"id"`
	State           string        `json:"state"`
	Channel         string        `json:"channel"`
	Project         string        `json:"project"`
	Driver          string        `json:"driver"`
	Mode            string        `json:"mode"`
	Reach           string        `json:"reach"`
	Harness         string        `json:"harness"`
	Credential      string        `json:"credential"`
	EngineSessionID *string       `json:"engineSessionId"`
	StartedAt       string        `json:"startedAt"`
	EndedAt         *string       `json:"endedAt"`
	EndOutcome      *string       `json:"endOutcome"`
	Turns           int64         `json:"turns"`
	Participants    []Participant `json:"participants"`
	Worktree        *Worktree     `json:"worktree"`
	Pending         *Pending      `json:"pending"`
}

func optional(r *store.Row, i int) *string {
	if r.IsNull(i) {
		return nil
	}
	s := r.Text(i)
	return &s
}

// Show reads one session from the store; an id it does not hold is unknown-session.
func Show(q store.Querier, id string) (Detail, error) {
	var d Detail
	found := false
	err := q.Query(`SELECT id, state, channel_id, project_id, driver_id, mode, reach, harness, credential_id,
    engine_session_id, started_at, ended_at, end_outcome, (SELECT count(*) FROM turn WHERE session_id = s.id)
FROM session s WHERE id = ?`, []store.Value{store.Text(id)}, func(r *store.Row) error {
		found = true
		d = Detail{ID: r.Text(0), State: r.Text(1), Channel: r.Text(2), Project: r.Text(3), Driver: r.Text(4),
			Mode: r.Text(5), Reach: r.Text(6), Harness: r.Text(7), Credential: r.Text(8), EngineSessionID: optional(r, 9),
			StartedAt: r.Text(10), EndedAt: optional(r, 11), EndOutcome: optional(r, 12), Turns: r.Int(13)}
		return nil
	})
	if err != nil {
		return d, err
	}
	if !found {
		return d, &Error{CodeUnknownSession, "no session " + id}
	}
	d.Participants = []Participant{}
	err = q.Query("SELECT user_id, role, joined_at FROM participant WHERE session_id = ? ORDER BY joined_at, rowid",
		[]store.Value{store.Text(id)}, func(r *store.Row) error {
			d.Participants = append(d.Participants, Participant{User: r.Text(0), Role: r.Text(1), JoinedAt: r.Text(2)})
			return nil
		})
	if err != nil {
		return d, err
	}
	err = q.Query("SELECT path, branch, state FROM worktree WHERE session_id = ? AND parent_path IS NULL ORDER BY created_at LIMIT 1",
		[]store.Value{store.Text(id)}, func(r *store.Row) error {
			d.Worktree = &Worktree{Path: r.Text(0), Branch: r.Text(1), State: r.Text(2)}
			return nil
		})
	return d, err
}

// Show is the bot's `session show`: the store's record, with the pending answer from memory.
func (s *Service) Show(id string) (Detail, error) {
	d, err := Show(s.cfg.Store, id)
	if err != nil {
		return d, err
	}
	if l := s.lookup(id); l != nil {
		l.mu.Lock()
		if l.pending != nil {
			p := *l.pending
			d.Pending = &p
		}
		l.mu.Unlock()
	}
	return d, nil
}

// Registration is one registration as `register list|show` print it: `<user id>  <name>
// <email>  <registered at>  <default credential id or ->`.
type Registration struct {
	User              string  `json:"user"`
	Name              string  `json:"name"`
	Email             string  `json:"email"`
	RegisteredAt      string  `json:"registeredAt"`
	DefaultCredential *string `json:"defaultCredential"`
}

// RegistrationColumns are Registration's text columns.
var RegistrationColumns = []string{"user", "name", "email", "registeredAt", "defaultCredential"}

const registrationColumns = "user_id, git_name, git_email, registered_at, default_credential_id"

func scanRegistration(r *store.Row) Registration {
	return Registration{User: r.Text(0), Name: r.Text(1), Email: r.Text(2), RegisteredAt: r.Text(3), DefaultCredential: optional(r, 4)}
}

// Registrations lists every registration in the order registered.
func Registrations(q store.Querier) ([]Registration, error) {
	out := []Registration{}
	err := q.Query("SELECT "+registrationColumns+" FROM registration ORDER BY registered_at, rowid", nil,
		func(r *store.Row) error { out = append(out, scanRegistration(r)); return nil })
	return out, err
}

// ShowRegistration is one registration; an id the store does not hold is unregistered.
func ShowRegistration(q store.Querier, user string) (Registration, error) {
	var reg Registration
	found := false
	err := q.Query("SELECT "+registrationColumns+" FROM registration WHERE user_id = ?", []store.Value{store.Text(user)},
		func(r *store.Row) error { reg, found = scanRegistration(r), true; return nil })
	if err == nil && !found {
		err = &Error{CodeUnregistered, user + " is not registered"}
	}
	return reg, err
}
