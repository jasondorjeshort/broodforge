// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"context"
	"strings"
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
	// EventInternalError is the bot's own failure inside a turn that does not end it (a store
	// write failing mid-turn, 03 Failure modes): Summary is the error (`store: <err>`), unindented;
	// Tool, Target, Parent and Depth are the call's when it happened inside one. The transcript
	// shows it at its depth; Discord posts nothing for it.
	EventInternalError EventType = "internal-error"
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
	// Parent is the subagent call a text, call, decision or result belongs to (the stream's
	// parent_tool_use_id, internal/engine/parse.go), Depth how many such calls it is nested in;
	// the summary is indented by Depth (Indent), so a subagent's calls sit under its Agent line.
	Parent string `json:"parentToolUseId,omitempty"`
	Depth  int    `json:"depth,omitempty"`
	// The payloads the Discord renderer needs beyond the summary (P11): the header's facts on
	// session-started, the footer's on turn-ended, the summary's on session-ended.
	Start *StartInfo `json:"start,omitempty"`
	Usage *TurnUsage `json:"usage,omitempty"`
	End   *EndInfo   `json:"end,omitempty"`
	// Commits is session-ending's count of the tree's commits over its base, nil when it cannot
	// be read: the venue's "Ending: checking and handing off <n> commits..." (04).
	Commits *int `json:"commits,omitempty"`
}

// indentUnit is one level of a subagent's nesting in a summary (04, The turn loop: subagent
// calls are indented under their Agent line).
const indentUnit = "  "

// Indent is s indented depth levels.
func Indent(depth int, s string) string {
	if depth <= 0 {
		return s
	}
	return strings.Repeat(indentUnit, depth) + s
}

// StartInfo is what the venue's pinned header shows (04, Starting and the first message).
type StartInfo struct {
	Channel        string   `json:"channel"`
	ProjectName    string   `json:"projectName"`
	Branch         string   `json:"branch"`
	Driver         string   `json:"driver"`
	DriverName     string   `json:"driverName"`
	Mode           string   `json:"mode"`
	ChannelMode    string   `json:"channelMode"`
	Model          string   `json:"model,omitempty"`        // the row's at start; "" the engine's own
	ChannelModel   string   `json:"channelModel,omitempty"` // the channel's, resolved
	Effort         string   `json:"effort,omitempty"`       // the row's; "" unset
	Reach          string   `json:"reach"`
	Credential     string   `json:"credential"`
	CredentialKind string   `json:"credentialKind"`
	IdleTimeout    string   `json:"idleTimeout"` // 04's words: "60 minutes", "90 seconds" or "never"
	Answer         []string `json:"answer"`
	// Commits and TreeMissing are set on a held session's opening session-interrupted only
	// (ADR 0026): the tree's commit count over its base, nil when it cannot be read, and whether
	// the tree is gone while refs/sessions/<id> is still in the canonical. The notice reads both.
	Commits     *int `json:"commits,omitempty"`
	TreeMissing bool `json:"treeMissing,omitempty"`
}

// The summaries of a session-interrupted event, its reason (05-debug-interface-session.md):
// the bot stopping, reconciliation at start for a session never held in memory, and a held
// session's first event after a restart (ADR 0026).
const (
	InterruptedShutdown       = "shutdown"
	InterruptedReconciliation = "reconciliation"
	InterruptedRestart        = "restart"
)

// TurnUsage is a turn's footer (04, The turn loop): the turn's tokens over every engine process
// it ran (in counts cache reads and writes), the session's so far, the model and the mode the
// turn ran in. OutUnknown is a process without a result (a stop, S2); Exit is the last
// process's exit code, "" when it is not known.
type TurnUsage struct {
	In         int64  `json:"in"`
	Out        int64  `json:"out"`
	OutUnknown bool   `json:"outUnknown,omitempty"`
	SessionIn  int64  `json:"sessionIn"`
	SessionOut int64  `json:"sessionOut"`
	Model      string `json:"model,omitempty"`
	Mode       string `json:"mode"`
	Exit       string `json:"exit,omitempty"`
}

// EndInfo is the summary's facts (04, Ending a session). Commits is the count handed off, or
// CommitsUnknown when a held session ends by one of end's escapes (ADR 0026), whose commits were
// handed off earlier or not at all. Log is those commits, oldest first, when the handoff read
// them; Title is the pull request's, when the handoff built one.
type EndInfo struct {
	By        string       `json:"by"` // who ended it, by name; "" for the idle timeout
	Reason    string       `json:"reason"`
	StartedAt string       `json:"startedAt"`
	Turns     int          `json:"turns"`
	In        int64        `json:"in"`
	Out       int64        `json:"out"`
	Model     string       `json:"model,omitempty"`
	Branch    string       `json:"branch"`
	Outcome   string       `json:"outcome"`
	Commits   int          `json:"commits"`
	Log       []CommitLine `json:"log,omitempty"`
	URL       string       `json:"url,omitempty"`
	Title     string       `json:"title,omitempty"`
	Author    string       `json:"author"`
	CoAuthors []string     `json:"coAuthors"`
}

// CommitLine is one commit the summary lists: its hash and its subject.
type CommitLine struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
}

// CommitsUnknown is EndInfo.Commits for an end by an escape: the count is not known.
const CommitsUnknown = -1

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
	// EventSessionEnding is the end's handoff about to run (04, Ending a session: the venue shows
	// `Ending: checking and handing off <n> commits...` meanwhile); its summary is the count.
	EventSessionEnding EventType = "session-ending"
)
