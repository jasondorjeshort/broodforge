// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The idle timeout, switched on a running session by the driver (`session timeout`,
// `/session timeout`). It lives on the live session only: the channel's value (0 is never)
// is where a session starts.

// EventTimeoutChanged is the driver's switch of the idle timeout.
const EventTimeoutChanged EventType = "timeout-changed"

// TimeoutText names the forms a timeout may take.
const TimeoutText = "never, a number of minutes, or a duration such as 90m or 2h"

// ParseIdleTimeout reads a timeout as a person gives it: `never`, `off` or `0` is no timeout;
// a bare number is minutes; otherwise a Go duration (`90m`, `2h`) of at least a minute.
func ParseIdleTimeout(s string) (time.Duration, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "never", "off", "none", "0":
		return 0, true
	case "":
		return 0, false
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Minute, n > 0
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < time.Minute {
		return 0, false
	}
	return d.Round(time.Second), true
}

// IdleText is an idle timeout in 04's words, `never` for none.
func IdleText(d time.Duration) string {
	if d <= 0 {
		return "never"
	}
	return humanDuration(d)
}

// TimeoutRequest is `session timeout`.
type TimeoutRequest struct {
	ActRequest
	Timeout string // as ParseIdleTimeout reads it
}

// SetIdleTimeout is the driver's switch of the session's idle timeout, effective now: a session
// waiting for input or an answer starts the new timeout's clock afresh.
func (s *Service) SetIdleTimeout(req TimeoutRequest) (Event, error) {
	d, ok := ParseIdleTimeout(req.Timeout)
	if !ok {
		return Event{}, badArgs("timeout " + quote(req.Timeout) + ": want " + TimeoutText)
	}
	l, _, err := s.acting(req.ActRequest)
	if err != nil {
		return Event{}, err
	}
	if err := s.checkDriver(l, req.ActRequest, "timeout"); err != nil {
		return Event{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	from, to := IdleText(l.idleTimeout), IdleText(d)
	err = s.cfg.Store.Write(func(tx *store.Tx) error {
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindTimeoutChanged, Actor: actorOf(req.As), Session: l.id,
			Details: details("from", from, "to", to)})
		return err
	})
	if err != nil {
		return Event{}, err
	}
	l.idleTimeout = d
	if l.state == StateWaitingForInput || l.state == StateWaitingForAnswer {
		s.armIdleLocked(l)
	}
	summary := "Ends after " + to + " without activity."
	if d <= 0 {
		summary = "Never ends for inactivity."
	}
	return l.log.add(Event{Time: s.stamp(), Type: EventTimeoutChanged, Turn: l.turn + 1, User: req.As, Result: to,
		Summary: summary}), nil
}
