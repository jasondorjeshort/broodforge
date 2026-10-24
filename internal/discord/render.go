// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// View renders one session's events into posts (03 Data flow, step 6; 04 from Starting to
// Ending): the venue and its pinned header, one working message per engine process edited in
// place, a delete-and-replace at the turn's end with the footer, the permission prompt with
// its buttons, the mode switch, and the summary with the transcript. It is fed by one
// goroutine; its posts go to offer, which the coalescer paces. Every body passes the mask.
// Handle and Resync hold mu, so a re-sync of the header either precedes the end's edit in the
// coalescer's queue or sees closed and does nothing (P12b).
type View struct {
	mu     sync.Mutex
	closed bool // session-ended or session-interrupted handled: the header keeps no components

	id     string
	venue  string // the private channel; in a dry run its id is the session id
	lobby  string // the project channel
	start  *session.StartInfo
	header string      // the pinned header's message id
	hdr    headerState // the header's mode and model lines, kept current (P12a)
	newID  func() string
	name   func(user string) string
	offer  func(Post)

	events []session.Event // everything seen, for the transcript

	turn      int
	turnStart time.Time
	entries   []entry // the turn's trace and text lines, in order
	segStart  int     // the first entry of the current engine process
	parts     []part  // the current process's working messages; only the last is edited
	allParts  []string
	waiting   bool       // the process ended waiting for an answer
	open      []openCall // the turn's tool calls without a result yet, oldest first
	steps     int        // the turn's text and call events so far, numbering them
	lastCall  int        // the number of the turn's newest call, 0 before one
	lastText  int        // the number of the turn's newest text, 0 before one

	prompt      string // the open prompt's message id
	promptCall  string
	promptTrace string
	dropped     int
	bypass      map[string]session.Event // decisions of calls the hook did not decide
}

type entry struct {
	call string // tool use id; "" for a text line
	text string
}

// openCall is a tool call the working header can name while it runs; parent is the subagent
// call it was made in, "" for the agent's own.
type openCall struct{ id, tool, parent string }

type part struct {
	msg   string
	start int // entry index
}

// ViewConfig is what a View needs besides its session id.
type ViewConfig struct {
	NewID func() string
	Name  func(user string) string
	Offer func(Post)
}

// NewView returns the view of session id.
func NewView(id string, c ViewConfig) *View {
	return &View{id: id, venue: id, newID: c.NewID, name: c.Name, offer: c.Offer, bypass: map[string]session.Event{}}
}

// Lobby is the session's project channel, "" before its session-started event.
func (v *View) Lobby() string { return v.lobby }

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (v *View) post(p Post) {
	p.Session = v.id
	if p.Channel == "" {
		p.Channel = v.venue
	}
	p.Body = Mask(p.Body)
	v.offer(p)
}

func (v *View) create(body string, buttons []Button) string {
	id := v.newID()
	v.post(Post{Op: OpCreate, MessageID: id, Body: body, Buttons: buttons})
	return id
}

func (v *View) who(user string) string {
	switch user {
	case "":
		return "Broodforge" // not 04
	case session.ActorHost:
		return "the host"
	}
	if v.start != nil && user == v.start.Driver {
		return v.start.DriverName
	}
	if v.name != nil {
		return v.name(user)
	}
	return mention(user)
}

// Handle renders one event.
func (v *View) Handle(ev session.Event) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.events = append(v.events, ev)
	t := parseTime(ev.Time)
	switch ev.Type {
	case session.EventSessionStarted:
		v.started(ev)
	case session.EventTurnStarted:
		v.turn, v.turnStart = ev.Turn, t
		v.entries, v.segStart, v.parts, v.allParts, v.waiting, v.dropped = nil, 0, nil, nil, false, 0
		v.open, v.steps, v.lastCall, v.lastText = nil, 0, 0, 0
		v.bypass = map[string]session.Event{}
		v.working(t)
	case session.EventText:
		for _, l := range splitLines(ev.Text) {
			v.entries = append(v.entries, entry{text: l})
		}
		v.steps++
		v.lastText = v.steps
		v.working(t)
	case session.EventToolCall:
		v.entries = append(v.entries, entry{call: ev.ToolUseID, text: firstOf(splitLines(ev.Summary))})
		v.steps++
		v.lastCall = v.steps
		v.open = append(v.open, openCall{id: ev.ToolUseID, tool: ev.Tool, parent: ev.Parent})
		v.working(t)
	case session.EventToolResult:
		line := ev.Summary
		if d, ok := v.bypass[ev.ToolUseID]; ok {
			line = session.BypassLine(session.TraceLine(ev.Tool, ev.Target), d.Decision == session.DecisionAllow, d.DecidedBy)
		}
		v.result(ev.ToolUseID, firstOf(splitLines(line)))
		v.closeCall(ev.ToolUseID)
		v.working(t)
	case session.EventPermissionDecision:
		if ev.Decision == session.DecisionDropped {
			v.closeCall(ev.ToolUseID) // a dropped call never gets a result
		}
		v.decision(ev)
	case session.EventWaitingForAnswer:
		v.wait(ev, t)
	case session.EventTurnEnded:
		v.ended(ev, t)
	case session.EventModeChanged:
		reach := ""
		if v.start != nil {
			reach = v.start.Reach
		}
		v.hdr.apply(ev)
		v.create(modeChangedText(ev.Result, reach), nil)
		v.editHeader()
	case session.EventModelChanged, session.EventEffortChanged, session.EventTimeoutChanged:
		v.switched(ev)
	case session.EventParticipantJoined:
		v.create(ev.Summary, nil)
	case session.EventHandoffRefused:
		msg := ev.Text
		if msg == "" {
			msg = ev.Summary
		}
		v.create(handoffRefusedText(msg), nil)
	case session.EventSessionEnded:
		v.sessionEnded(ev, t)
	case session.EventSessionInterrupted:
		// A restart's interruption posts nothing in 04; the header is not re-synced after it.
		v.closed = true
	case session.EventMessageQueued, session.EventMessageDropped, session.EventParticipantInvited:
		// The hourglass on a queued message is the gateway's (it holds the message's Discord
		// id); a dropped message and an invitation post nothing in 04.
	}
}

// Resync posts the pinned header again as the View's own state has it, menus and End session
// included, so a refused pick drops off the picker's screen (05-debug-interface-discord.md, The
// header's menus). Once the View has handled session-ended or session-interrupted it does
// nothing, so a late re-sync never puts the components back on an ended header.
func (v *View) Resync() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return
	}
	v.editHeader()
}

func (v *View) started(ev session.Event) {
	v.start = ev.Start
	if v.start == nil {
		v.start = &session.StartInfo{DriverName: ev.User, Driver: ev.User}
	}
	v.lobby = v.start.Channel
	v.post(Post{Op: OpChannel, MessageID: v.venue, First: "#" + venueName(v.id, v.start.DriverName)})
	v.hdr.apply(session.Event{Type: session.EventSessionStarted, Start: v.start}) // v.start, never nil
	v.header = v.newID()
	v.post(Post{Op: OpCreate, MessageID: v.header, Pinned: true, Body: headerText(v.id, v.start, v.hdr), Buttons: v.endButton(),
		Selects: headerSelects(v.id, v.hdr)})
}

// editHeader edits the pinned header in place from the current state, with its menus and the
// End session button; nothing before the header exists.
func (v *View) editHeader() {
	if v.header == "" || v.start == nil {
		return
	}
	v.post(Post{Op: OpEdit, MessageID: v.header, Body: headerText(v.id, v.start, v.hdr), Buttons: v.endButton(),
		Selects: headerSelects(v.id, v.hdr)})
}

func (v *View) endButton() []Button {
	return []Button{{Label: "End session", CustomID: CustomID{Kind: IDEnd, Session: v.id}.String(), Style: StyleDanger}}
}

// switched is a model, effort or idle timeout switch: its line (04's `From the next turn:` for
// model and effort), then the pinned header edited in place.
func (v *View) switched(ev session.Event) {
	v.hdr.apply(ev)
	v.create(ev.Summary, nil)
	v.editHeader()
}

// headerSelects are the header's three menus in the order model, effort, mode
// (04-ux-session-controls.md, The header's menus), each marking the current value as its
// default; a menu with nothing marked shows the placeholder 04 gives it.
func headerSelects(id string, m headerState) []Select {
	model := Select{CustomID: CustomID{Kind: IDModel, Session: id}.String()}
	for _, alias := range policy.ModelAliases() {
		aliasID, _ := policy.ResolveModel(alias)
		// A dated id belongs to its prefix's entry, as ModelDisplayName reads it.
		def := m.model != "" && policy.ModelDisplayName(m.model) == policy.ModelDisplayName(aliasID)
		model.Options = append(model.Options, Option{Label: session.ModelShort(aliasID), Value: alias, Default: def})
	}
	if !anyDefault(model.Options) {
		model.Placeholder = modelEngineOwnPlaceholder
		if m.model != "" {
			model.Placeholder = modelPlaceholder(m.model)
		}
	}
	effort := Select{CustomID: CustomID{Kind: IDEffort, Session: id}.String()}
	for _, level := range policy.Efforts {
		effort.Options = append(effort.Options, Option{Label: level, Value: level, Default: m.effort == level})
	}
	if !anyDefault(effort.Options) {
		effort.Placeholder = effortEngineOwnPlaceholder
	}
	acts := m.mode == policy.ModePolicy.String()
	mode := Select{CustomID: CustomID{Kind: IDMode, Session: id}.String(), Options: []Option{
		{Label: modeAsksFirstLabel, Value: modeAsksFirst, Default: !acts},
		{Label: modeActsFreelyLabel, Value: modeActsFreely, Default: acts},
	}}
	return []Select{model, effort, mode}
}

func anyDefault(opts []Option) bool {
	for _, o := range opts {
		if o.Default {
			return true
		}
	}
	return false
}

// headerState is the session's mode, model, effort and idle timeout as the header shows them,
// from its events: the start's, then each switch, so a header edited for one switch shows the
// others.
type headerState struct {
	mode, modeWhy      string
	model, why, effort string
	idle               string // session.IdleText's words: "never", "60 minutes"
}

func (m *headerState) apply(ev session.Event) {
	// Five types of many, so not a switch the exhaustive check would make list them all.
	if ev.Type == session.EventSessionStarted && ev.Start != nil {
		m.idle = ev.Start.IdleTimeout
		m.mode, m.modeWhy = ev.Start.Mode, "this channel's default"
		if ev.Start.Mode != ev.Start.ChannelMode {
			m.modeWhy = "chosen at start" // not 04
		}
		m.model, m.effort, m.why = ev.Start.Model, ev.Start.Effort, "this channel's default"
		if ev.Start.Model != ev.Start.ChannelModel {
			m.why = "chosen at start"
		}
	}
	if ev.Type == session.EventModeChanged {
		m.mode, m.modeWhy = ev.Result, "from turn "+strconv.Itoa(ev.Turn) // not 04: model's words
	}
	if ev.Type == session.EventModelChanged {
		m.model, m.why = ev.Result, "from turn "+strconv.Itoa(ev.Turn)
	}
	if ev.Type == session.EventEffortChanged {
		m.effort = ev.Result
	}
	if ev.Type == session.EventTimeoutChanged {
		m.idle = ev.Result
	}
}

// headerOf is the header state after the events.
func headerOf(evs []session.Event) headerState {
	var m headerState
	for _, ev := range evs {
		m.apply(ev)
	}
	return m
}

// splitLines is a text's lines, each cut to maxLineRunes so any one line fits a message.
func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		r := []rune(l)
		for len(r) > maxLineRunes {
			out = append(out, string(r[:maxLineRunes]))
			r = r[maxLineRunes:]
		}
		out = append(out, string(r))
	}
	return out
}

func firstOf(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

// result puts a call's result line in place of its call line. A call of an earlier process
// (the deferred one, answered) moves to the end, where the resumed process shows it.
func (v *View) result(call, line string) {
	for i, e := range v.entries {
		if e.call != call {
			continue
		}
		if i >= v.segStart {
			v.entries[i].text = line
			return
		}
		v.entries = append(v.entries[:i], v.entries[i+1:]...)
		v.segStart--
		for j := range v.parts {
			v.parts[j].start--
		}
		break
	}
	v.entries = append(v.entries, entry{call: call, text: line})
}

func (v *View) lines(from, to int) []string {
	out := make([]string, 0, to-from)
	for _, e := range v.entries[from:to] {
		out = append(out, e.text)
	}
	return out
}

// closeCall drops a call that has its result, or was dropped, from the open ones, and with it
// every open call made inside it: a subagent's call left without a result when its Agent call
// returned is over too.
func (v *View) closeCall(id string) {
	keep := v.open[:0]
	var inside []string
	for _, c := range v.open {
		switch {
		case c.id == id:
		case c.parent == id:
			inside = append(inside, c.id)
		default:
			keep = append(keep, c)
		}
	}
	v.open = keep
	for _, c := range inside {
		v.closeCall(c)
	}
}

// step is the working header's current step (owner, 2026-10-24), the newest step winning:
// `writing` when the agent's text is newer than every call; else the newest open call's tool
// while one runs (a subagent's call inside its Agent call, the Agent call again once that
// returns); else none, at the turn's start or once every call has its result or was dropped
// and nothing newer came (the agent is between steps, and no step is shown rather than one
// that has finished). A newest open call without a tool name shows none, not an older call.
func (v *View) step() string {
	if v.lastText > v.lastCall {
		return stepWriting
	}
	if n := len(v.open); n > 0 && v.open[n-1].tool != "" {
		return stepRunning(v.open[n-1].tool)
	}
	return ""
}

// endedOnCall is whether the turn's last entry is a tool call or its result rather than the
// agent's text; a turn without entries did not.
func (v *View) endedOnCall() bool {
	n := len(v.entries)
	return n > 0 && v.entries[n-1].call != ""
}

func runes(s string) int { return len([]rune(s)) }

func stopButton(id string) []Button {
	return []Button{{Label: "Stop", CustomID: CustomID{Kind: IDStop, Session: id}.String(), Style: StyleSecondary}}
}

// working renders the current process's working message at t: a new one after a wait, a
// frozen part and a new one when the newest would pass Discord's limit (R-22), else one edit,
// which the coalescer merges with the next (R-21).
func (v *View) working(t time.Time) {
	if v.turn == 0 {
		return
	}
	head := workingHeader(v.turn, t.Sub(v.turnStart), v.step())
	if v.waiting || len(v.parts) == 0 {
		v.waiting = false
		id := v.create(head, stopButton(v.id))
		v.parts = []part{{msg: id, start: v.segStart}}
		v.allParts = append(v.allParts, id)
	}
	for {
		last := &v.parts[len(v.parts)-1]
		body := strings.Join(append([]string{head}, v.lines(last.start, len(v.entries))...), "\n")
		if runes(body) <= MessageLimit {
			v.post(Post{Op: OpEdit, MessageID: last.msg, Body: body, Buttons: stopButton(v.id)})
			return
		}
		k, size := last.start, runes(head)
		for k < len(v.entries) && size+1+runes(v.entries[k].text) <= MessageLimit {
			size += 1 + runes(v.entries[k].text)
			k++
		}
		if k == last.start {
			k++
		}
		frozen := strings.Join(append([]string{head}, v.lines(last.start, k)...), "\n")
		v.post(Post{Op: OpEdit, MessageID: last.msg, Body: frozen})
		id := v.create(head, stopButton(v.id))
		v.parts = append(v.parts, part{msg: id, start: k})
		v.allParts = append(v.allParts, id)
	}
}

func (v *View) decision(ev session.Event) {
	switch ev.DecidedBy {
	case session.DecidedByConfig, session.DecidedByUnknown:
		v.bypass[ev.ToolUseID] = ev
		return
	}
	if ev.ToolUseID == v.promptCall && v.prompt != "" && ev.Decision != session.DecisionDeferred {
		v.post(Post{Op: OpEdit, MessageID: v.prompt, Body: answeredText(ev, v.who(ev.User), v.promptTrace)})
		v.prompt, v.promptCall = "", ""
		return
	}
	if ev.Decision == session.DecisionDropped && ev.DecidedBy == session.DecidedByPolicy {
		v.dropped++
	}
}

// wait posts the prompt, mentioning the driver so a phone buzzes (04), then freezes the
// working message. Queued in that order, the prompt goes first when the working message was
// already sent; when its create is still queued, that create (carrying the frozen content)
// goes out first and the prompt a second later, the per-session limit.
func (v *View) wait(ev session.Event, t time.Time) {
	driver := ""
	if v.start != nil {
		driver = v.start.Driver
	}
	v.promptCall = ev.ToolUseID
	v.promptTrace = session.TraceLine(ev.Tool, ev.Target)
	v.prompt = v.create(promptText(driver, ev.Turn, v.promptTrace, ev.Text, v.dropped),
		promptButtons(v.id, ev.ToolUseID, ev.Tool))
	v.dropped = 0
	if n := len(v.parts); n > 0 {
		last := v.parts[n-1]
		head := waitingHeader(v.turn, t.Sub(v.turnStart))
		body := strings.Join(append([]string{head}, v.lines(last.start, len(v.entries))...), "\n")
		v.post(Post{Op: OpEdit, MessageID: last.msg, Body: body})
	}
	v.parts, v.waiting, v.segStart = nil, true, len(v.entries)
}

// ended is 04's delete-and-replace: every working message of the turn goes, and its final
// content comes back as new messages ending in the footer, so Discord notifies (R-23).
func (v *View) ended(ev session.Event, t time.Time) {
	elapsed := t.Sub(v.turnStart)
	lines := v.lines(0, len(v.entries))
	label := idleLabel(v.endedOnCall())
	switch {
	case ev.Result == session.ResultCompleted:
	case ev.Result == session.ResultError:
		label = labelFailed
		lines = append(lines, turnErrorText(ev.Summary))
	case ev.User != "":
		label = labelStopped
		lines = append(lines, stoppedText(v.who(ev.User), elapsed))
	case ev.Summary == session.ResultInterrupted:
		label = labelFailed
		exit := ""
		if ev.Usage != nil {
			exit = ev.Usage.Exit
		}
		lines = append(lines, turnFailedText(exit))
	default:
		label = labelStopped
		lines = append(lines, "Stopped: "+ev.Summary+".") // not 04
	}
	lines = append(lines, footer(label, ev.Turn, elapsed, ev.Usage))
	for _, chunk := range chunks(lines) {
		v.create(chunk, nil)
	}
	// The final content first, then the deletes, so the venue never shows the turn missing. A
	// working message whose create is still queued still goes out (the coalescer never cancels
	// a create), then its delete.
	for _, id := range v.allParts {
		v.post(Post{Op: OpDelete, MessageID: id})
	}
	v.entries, v.parts, v.allParts, v.segStart, v.waiting = nil, nil, nil, 0, false
	v.open, v.steps, v.lastCall, v.lastText = nil, 0, 0, 0
	v.turn = 0
}

// chunks joins lines into messages within Discord's limit, in order (R-22: split).
func chunks(lines []string) []string {
	var out []string
	cur, size := []string{}, 0
	for _, l := range lines {
		n := runes(l)
		if len(cur) > 0 && size+1+n > MessageLimit {
			out = append(out, strings.Join(cur, "\n"))
			cur, size = nil, 0
		}
		if len(cur) > 0 {
			size++
		}
		cur, size = append(cur, l), size+n
	}
	if len(cur) > 0 {
		out = append(out, strings.Join(cur, "\n"))
	}
	return out
}

// sessionEnded posts the end: 04's idle notice first when the timeout ended it, a failed
// handoff's line, then the summary in the venue and in the project channel with the
// transcript attached there (Owner 11).
func (v *View) sessionEnded(ev session.Event, t time.Time) {
	end := ev.End
	if end == nil {
		end = &session.EndInfo{Outcome: ev.Result}
	}
	// The header once more, without its menus and End session; its text stays as the record.
	v.closed = true
	if v.header != "" && v.start != nil {
		v.post(Post{Op: OpEdit, MessageID: v.header, Body: headerText(v.id, v.start, v.hdr)})
	}
	if end.Reason == "idle" {
		v.create(ev.Summary, nil)
	}
	if strings.HasPrefix(end.Outcome, "failed: ") {
		text := handoffFailedText(v.id, strings.TrimPrefix(end.Outcome, "failed: "), end.Commits)
		v.create(text, nil)
		v.post(Post{Op: OpCreate, Channel: v.lobby, MessageID: v.newID(), Body: text})
	}
	summary := summaryText(v.id, end, t)
	v.create(summary, nil)
	if v.lobby != "" {
		msg := v.newID()
		v.post(Post{Op: OpCreate, Channel: v.lobby, MessageID: msg, Body: summary})
		v.post(Post{Op: OpAttach, Channel: v.lobby, MessageID: msg, File: v.id + ".md", First: v.id + ".md",
			Body: Transcript(v.id, v.events, v.who)})
	}
}
