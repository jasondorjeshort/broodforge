// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Model and effort (P12a; 05-debug-interface-session.md): chosen at start, else the
// channel's, copied to the session row, switched by the driver from the next turn. Every value
// that reaches the engine is one policy resolves (a model the map knows, a listed effort).

// Events P12a adds.
const (
	EventModelChanged  EventType = "model-changed"
	EventEffortChanged EventType = "effort-changed"
)

// modelPick is a start's model and effort as the row gets them, and the channel's model, by
// which the header says whether the start chose it.
type modelPick struct {
	model, effort, channel string
}

// ModelsText names the models a person may give: `opus, sonnet, haiku, fable`.
func ModelsText() string { return strings.Join(policy.ModelAliases(), ", ") }

// EffortsText names the efforts: `low, medium, high, xhigh, max`.
func EffortsText() string { return strings.Join(policy.Efforts, ", ") }

// ModelShort is a model's short name as 04 writes it, `Sonnet` for claude-sonnet-5-5; an id
// policy does not know is shown as it is.
func ModelShort(id string) string {
	name := policy.ModelDisplayName(id)
	rest, ok := strings.CutPrefix(name, "Claude ")
	if !ok {
		return name
	}
	if f := strings.Fields(rest); len(f) > 0 {
		return f[0]
	}
	return name
}

// startModel resolves a start's model and effort after the registration and role checks. An
// explicit choice (--model, model:) wins: a bad one is bad-args, and the channel's model is
// then read only for the header's parenthesis. Without one, the channel's model; a channel
// value policy does not know (config check refuses it; a last-good file may still hold one)
// refuses the start, audited, so nothing unknown reaches the engine. The effort is the channel's.
func (s *Service) startModel(actor string, ch Channel, typed string) (model, effort, channel string, err error) {
	if ch.Effort != "" && !policy.ParseEffort(ch.Effort) {
		return "", "", "", s.refuse(actor, "", "start", "channel-effort", badArgs("channel "+ch.ID+" has effort "+
			quote(ch.Effort)+" in its configuration; efforts: "+EffortsText()))
	}
	if typed != "" {
		var ok bool
		if model, ok = policy.ResolveModel(typed); !ok {
			return "", "", "", badArgs("--model " + quote(typed) + ": want " + ModelsText() + " or a model id they name")
		}
		channel, _ = policy.ResolveModel(ch.Model)
		return model, ch.Effort, channel, nil
	}
	if ch.Model != "" {
		var ok bool
		if channel, ok = policy.ResolveModel(ch.Model); !ok {
			return "", "", "", s.refuse(actor, "", "start", "channel-model", badArgs("channel "+ch.ID+" has model "+
				quote(ch.Model)+" in its configuration; models: "+ModelsText()))
		}
	}
	return channel, ch.Effort, channel, nil
}

// ModelRequest is `session model`.
type ModelRequest struct {
	ActRequest
	Model string // an alias or a model id policy knows
}

// EffortRequest is `session effort`.
type EffortRequest struct {
	ActRequest
	Effort string
}

// SetModel is SessionService.setModel: the driver's switch of the session row's model,
// effective at the next turn (ADR 0020: a per-process flag; --resume keeps the context).
func (s *Service) SetModel(req ModelRequest) (Event, error) {
	id, ok := policy.ResolveModel(req.Model)
	if !ok {
		return Event{}, badArgs("model " + quote(req.Model) + ": want " + ModelsText() + " or a model id they name")
	}
	return s.setColumn(req.ActRequest, "model", id, audit.KindModelChanged, EventModelChanged,
		"From the next turn: "+ModelShort(id)+".")
}

// SetEffort is SessionService.setEffort: as SetModel, for the effort.
func (s *Service) SetEffort(req EffortRequest) (Event, error) {
	if !policy.ParseEffort(req.Effort) {
		return Event{}, badArgs("effort " + quote(req.Effort) + ": want " + EffortsText())
	}
	return s.setColumn(req.ActRequest, "effort", req.Effort, audit.KindEffortChanged, EventEffortChanged,
		"From the next turn: effort "+req.Effort+".")
}

// setColumn is SetModel's and SetEffort's write: the driver only, an open session only, the row
// and the audit event (`from`, `to`) in one transaction, then the event with the next turn's
// number. column is "model" or "effort", never input.
func (s *Service) setColumn(act ActRequest, column, value string, kind audit.Kind, typ EventType, summary string) (Event, error) {
	l, _, err := s.acting(act)
	if err != nil {
		return Event{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := s.switchOpenLocked(l, act, column); err != nil {
		return Event{}, err
	}
	if err := s.checkDriver(l, act, column); err != nil {
		return Event{}, err
	}
	err = s.cfg.Store.Write(func(tx *store.Tx) error {
		from := ""
		if err := tx.Query("SELECT coalesce("+column+", '') FROM session WHERE id = ?", []store.Value{store.Text(l.id)},
			func(r *store.Row) error { from = r.Text(0); return nil }); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE session SET "+column+" = ? WHERE id = ?", store.Text(value), store.Text(l.id)); err != nil {
			return err
		}
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: kind, Actor: actorOf(act.As), Session: l.id,
			Details: details("from", from, "to", value)})
		return err
	})
	if err != nil {
		return Event{}, err
	}
	if column == "model" {
		l.model = value // the summary's model when no usage names one
	}
	return l.log.add(Event{Time: s.stamp(), Type: typ, Turn: l.turn + 1, User: act.As, Result: value, Summary: summary}), nil
}

// turnModelLocked reads the row's model and effort for the turn starting; l.mu is held. The
// row is the record (05: the spawn reads the row, not the channel).
func (s *Service) turnModelLocked(tx *store.Tx, l *live) error {
	return tx.Query("SELECT coalesce(model, ''), coalesce(effort, '') FROM session WHERE id = ?", []store.Value{store.Text(l.id)},
		func(r *store.Row) error {
			l.turnModel, l.turnEffort = r.Text(0), r.Text(1)
			return nil
		})
}
