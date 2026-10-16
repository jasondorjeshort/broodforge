// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"sync"
)

// EventType names a SessionEvent: 03's five (text, tool-call, tool-result, waiting-for-answer,
// turn-ended) and the lifecycle events `session tail` shows beside them.
type EventType string

const (
	EventSessionStarted     EventType = "session-started"
	EventParticipantJoined  EventType = "participant-joined"
	EventMessageQueued      EventType = "message-queued"
	EventTurnStarted        EventType = "turn-started"
	EventText               EventType = "text"
	EventToolCall           EventType = "tool-call"
	EventToolResult         EventType = "tool-result"
	EventPermissionDecision EventType = "permission-decision"
	EventWaitingForAnswer   EventType = "waiting-for-answer"
	EventTurnEnded          EventType = "turn-ended"
	EventSessionInterrupted EventType = "session-interrupted"
)

// Event is one SessionEvent as `session tail` prints it: `<seq>  <time>  <type>  <summary>`,
// --json every field. Summary is 04's trace line where 04 gives one (`[Read] hello.txt`), the
// text's first 80 characters, or the stop reason (05-debug-interface-session.md).
type Event struct {
	Seq       int64     `json:"seq"`
	Time      string    `json:"time"`
	Type      EventType `json:"type"`
	Summary   string    `json:"summary"`
	Turn      int       `json:"turn,omitempty"`
	User      string    `json:"user,omitempty"`
	ToolUseID string    `json:"toolUseId,omitempty"`
	Tool      string    `json:"tool,omitempty"`
	Target    string    `json:"target,omitempty"`
	IsError   bool      `json:"isError,omitempty"`
	Decision  string    `json:"decision,omitempty"`
	DecidedBy string    `json:"decidedBy,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Text      string    `json:"text,omitempty"`
	Result    string    `json:"result,omitempty"`
}

// eventLog is one session's events in memory: appended by the session's own goroutines,
// read by any number of followers. A follower never blocks the writer and never misses an
// event: it reads the slice from its own position and waits on changed, which is closed and
// replaced on every append (a broadcast).
type eventLog struct {
	mu      sync.Mutex
	events  []Event
	changed chan struct{}
	closed  bool
}

func newEventLog() *eventLog { return &eventLog{changed: make(chan struct{})} }

// add stamps ev with the next seq and appends it.
func (l *eventLog) add(ev Event) Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	ev.Seq = int64(len(l.events)) + 1
	l.events = append(l.events, ev)
	if !l.closed { // a closed log's channel is closed already
		close(l.changed)
		l.changed = make(chan struct{})
	}
	return ev
}

// close wakes every follower for the last time: the session will add no more events.
func (l *eventLog) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		close(l.changed)
	}
}

// since returns the events after seq, the channel that closes on the next append, and whether
// the log is closed.
func (l *eventLog) since(seq int64) ([]Event, <-chan struct{}, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if seq < 0 {
		seq = 0
	}
	var out []Event
	if int(seq) < len(l.events) {
		out = append(out, l.events[seq:]...)
	}
	return out, l.changed, l.closed
}

// last is the seq of the newest event.
func (l *eventLog) last() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int64(len(l.events))
}

// follow hands every event after seq to fn, in order, waiting for new ones, until fn returns
// true, the log closes or ctx ends.
func (l *eventLog) follow(ctx context.Context, seq int64, fn func(Event) (stop bool, err error)) error {
	for {
		evs, changed, closed := l.since(seq)
		for _, ev := range evs {
			seq = ev.Seq
			stop, err := fn(ev)
			if err != nil || stop {
				return err
			}
		}
		if closed {
			return nil
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Events P10 half (b) adds: prompt mode's answers, the mode switch, an invitation, the end.
const (
	EventModeChanged        EventType = "mode-changed"
	EventParticipantInvited EventType = "participant-invited"
	EventHandoffRefused     EventType = "handoff-refused"
	EventSessionEnded       EventType = "session-ended"
	EventMessageDropped     EventType = "message-dropped"
)
