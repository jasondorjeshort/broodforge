// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

func TestMask(t *testing.T) {
	sha40 := strings.Repeat("a1b2c3d4e5", 4)
	sha64 := strings.Repeat("0123456789abcdef", 4)
	for in, want := range map[string]string{
		"key sk-ant-api03-abcdefghijkl done":           "key [masked] done",
		"Authorization: Bearer abc.def-ghi_jkl123":     "Authorization: Bearer [masked]",
		"token ghp_0123456789abcdefghijABCDEFGHIJ end": "token [masked] end",
		"commit " + sha40 + " and " + sha64:            "commit " + sha40 + " and " + sha64,
		"blob " + strings.Repeat("Zz9", 14):            "blob [masked]",
		"[Read] internal/discord/render_test.go":       "[Read] internal/discord/render_test.go",
		"path a/" + strings.Repeat("Qx", 21) + "/b.go": "path [masked]",
		"plain words stay as they are":                 "plain words stay as they are",
	} {
		if got := Mask(in); got != want {
			t.Errorf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
	zwsp := "\xe2\x80\x8b" // U+200B, a format character
	for text, want := range map[string]bool{
		"here sk-ant-xyz":                  true,
		"sk-ant-api03-abc":                 true,
		"key:sk-ant-abc":                   true,
		"sk" + zwsp + "-ant-api03-abc":     true,
		"s" + zwsp + "k-ant-api03-abc":     true,
		"_sk-ant-api03-abcdefgh_":          true, // Discord italics
		"__sk-ant-api03-abcdefgh__":        true, // underline
		"keysk-ant-api03-abcdefgh":         true, // glued to a word
		"sk-a͏nt-api03-abcdefgh":           true, // U+034F, a nonspacing mark
		"kéysk-ant-api03-abcdefgh":        true,
		"the task-ant-colony test fails":   false,
		"multitask-ant-ok":                 false,
		"sk-other and ask-ant-eater":       false,
		"nothing here at all, not one key": false,
	} {
		if got := HasKey(text); got != want {
			t.Errorf("HasKey(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestCustomID(t *testing.T) {
	for _, s := range []string{"bf:s0001:toolu_01Nm:allow", "bf:s0001:toolu_01Nm:deny", "bf:s0001:toolu_01Nm:allow-session",
		"bf:s0001:stop", "bf:s0001:end", "bf:100:start"} {
		id, err := ParseCustomID(s)
		if err != nil {
			t.Errorf("ParseCustomID(%q): %v", s, err)
			continue
		}
		if id.String() != s {
			t.Errorf("round trip %q -> %q", s, id.String())
		}
	}
	for _, s := range []string{"", "bf", "bf:s0001", "xx:s0001:stop", "bf:s0001:toolu:maybe", "bf:100:stop", "bf:s0001:start",
		"bf:s0001:a:b:allow", "bf:s1:end"} {
		if _, err := ParseCustomID(s); err == nil {
			t.Errorf("ParseCustomID(%q) accepted", s)
		}
	}
}

func TestParseSlash(t *testing.T) {
	s, err := ParseSlash("/register name:Jason Dorje Short email:jdorje@gmail.com")
	if err != nil || s.Kind != SlashRegister || s.Options["name"] != "Jason Dorje Short" || s.Options["email"] != "jdorje@gmail.com" {
		t.Errorf("register: %+v %v", s, err)
	}
	s, err = ParseSlash("/session mode acts-freely")
	if err != nil || s.Kind != SlashSessionMode || len(s.Args) != 1 || s.Args[0] != "acts-freely" {
		t.Errorf("mode: %+v %v", s, err)
	}
	s, err = ParseSlash("/session invite user:<@1002>")
	if u, ok := UserOption(s.Options["user"]); err != nil || s.Kind != SlashSessionInvite || !ok || u != "1002" {
		t.Errorf("invite: %+v %v", s, err)
	}
	if _, err := ParseSlash("hello"); err != ErrNotSlash {
		t.Errorf("plain text: %v", err)
	}
	if _, err := ParseSlash("/register nickname:x"); err == nil {
		t.Error("an unknown option was accepted")
	}
	if _, err := ParseSlash("/frobnicate"); err == nil {
		t.Error("an unknown command was accepted")
	}
}

func TestTextFormats(t *testing.T) {
	for n, want := range map[int64]string{950: "950", 48200: "48.2k", 31000: "31.0k", 212000: "212k", 1_200_000: "1.2M"} {
		if got := tokens(n); got != want {
			t.Errorf("tokens(%d) = %s, want %s", n, got, want)
		}
	}
	if got := clock(102 * time.Second); got != "1:42" {
		t.Errorf("clock = %s", got)
	}
	if got := span(72 * time.Minute); got != "1h 12m" {
		t.Errorf("span = %s", got)
	}
	if got := modelShort("claude-opus-5-5-20261001"); got != "Opus" {
		t.Errorf("modelShort = %s", got)
	}
	u := &session.TurnUsage{In: 48200, Out: 1100, SessionIn: 212000, Model: "claude-opus-5-5", Mode: "prompt"}
	if got, want := footer(idleLabel(false), 3, 102*time.Second, u), "Idle · turn 3 · 1:42 · 48.2k in / 1.1k out · session 212k · Opus · asks first"; got != want {
		t.Errorf("footer\n got %s\nwant %s", got, want)
	}
	if got, want := footer(idleLabel(true), 5, 621*time.Second, u), "Idle · ended on a tool call, no closing message · turn 5 · 10:21 · 48.2k in / 1.1k out · session 212k · Opus · asks first"; got != want {
		t.Errorf("footer ended on a call\n got %s\nwant %s", got, want)
	}
	for step, want := range map[string]string{"": "Working · turn 5 · 0:00", stepRunning("Bash"): "Working · turn 5 · 0:00 · running Bash", stepWriting: "Working · turn 5 · 0:00 · writing"} {
		if got := workingHeader(5, 0, step); got != want {
			t.Errorf("workingHeader(%q) = %s, want %s", step, got, want)
		}
	}
	u = &session.TurnUsage{In: 31000, OutUnknown: true, SessionIn: 195000, Model: "claude-opus-5-5", Mode: "prompt"}
	if got, want := footer(labelStopped, 3, 48*time.Second, u), "Stopped · turn 3 · 0:48 · 31.0k in / out unknown · session 195k · Opus · asks first"; got != want {
		t.Errorf("stopped footer\n got %s\nwant %s", got, want)
	}
	if got := session.BypassLine("[Bash] echo ok", true, session.DecidedByConfig); got != "[Bash] echo ok -> ran without Broodforge's check: the permission check timed out on the host." {
		t.Errorf("bypass: %s", got)
	}
}

// TestCoalescerPerSession: a burst of edits to one message goes out at most once a second,
// the last content wins, and a delete drops the edits still queued.
func TestCoalescerPerSession(t *testing.T) {
	var sent []Post
	var at []time.Time
	co := NewCoalescer(func(now time.Time, p Post) { sent, at = append(sent, p), append(at, now) })
	t0 := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	co.Offer(Post{Session: "s1", Op: OpCreate, MessageID: "m1", Body: "v0"})
	co.Flush(t0)
	for i := 1; i <= 30; i++ {
		co.Offer(Post{Session: "s1", Op: OpEdit, MessageID: "m1", Body: "v" + strconv.Itoa(i)})
		co.Flush(t0.Add(time.Duration(i) * 100 * time.Millisecond))
	}
	for now := t0.Add(3 * time.Second); co.Pending() > 0; now = now.Add(time.Second) {
		co.Flush(now)
	}
	for i := 1; i < len(at); i++ {
		if at[i].Sub(at[i-1]) < PerSession {
			t.Fatalf("posts %d and %d %v apart", i-1, i, at[i].Sub(at[i-1]))
		}
	}
	if last := sent[len(sent)-1]; last.Body != "v30" {
		t.Errorf("last content %q, want v30", last.Body)
	}
	if len(sent) > 5 {
		t.Errorf("%d posts for a 3 s burst", len(sent))
	}
	co.Offer(Post{Session: "s1", Op: OpEdit, MessageID: "m1", Body: "stale"})
	co.Offer(Post{Session: "s1", Op: OpDelete, MessageID: "m1"})
	co.Flush(t0.Add(time.Minute))
	if p := sent[len(sent)-1]; p.Op != OpDelete {
		t.Errorf("after a delete: %s %q", p.Op, p.Body)
	}
}

// TestCoalescerGlobalBudget: 120 sessions each with one post wait for the bot's 50 a second.
func TestCoalescerGlobalBudget(t *testing.T) {
	var at []time.Time
	co := NewCoalescer(func(now time.Time, _ Post) { at = append(at, now) })
	t0 := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 120; i++ {
		co.Offer(Post{Session: "s" + strconv.Itoa(1000+i), Op: OpCreate, MessageID: "m" + strconv.Itoa(i), Body: "x"})
	}
	now := t0
	for co.Pending() > 0 {
		co.Flush(now)
		next, ok := co.NextAt(now)
		if !ok {
			break
		}
		now = next
	}
	if len(at) != 120 {
		t.Fatalf("%d of 120 sent", len(at))
	}
	if m := maxInWindow(at); m > GlobalBudget {
		t.Errorf("%d posts in one second", m)
	}
	if last := at[len(at)-1].Sub(t0); last < 2*time.Second {
		t.Errorf("120 posts done after %v", last)
	}
}

func ev(typ session.EventType, at time.Duration, f func(*session.Event)) session.Event {
	e := session.Event{Type: typ, Time: time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC).Add(at).Format(time.RFC3339Nano)}
	if f != nil {
		f(&e)
	}
	return e
}

func newTestView() (*View, *[]Post) {
	var posts []Post
	n := 0
	v := NewView("s0001", ViewConfig{NewID: func() string { n++; return "m" + strconv.Itoa(n) }, Name: mention,
		Offer: func(p Post) { posts = append(posts, p) }})
	return v, &posts
}

var testStart = &session.StartInfo{Channel: "100", ProjectName: "Broodforge", Branch: "broodforge/pc/s0001", Driver: "1001",
	DriverName: "Jason", Mode: "prompt", ChannelMode: "prompt", Reach: "code", Credential: "cred-7k2m",
	CredentialKind: "subscription", IdleTimeout: "60 minutes", Answer: []string{"driver"}}

// TestViewHeaderAndTurn: 04's header, the working message, the delete-and-replace with the
// footer last, and the prompt's buttons and answer.
func TestViewHeaderAndTurn(t *testing.T) {
	v, posts := newTestView()
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
	header := "Session s0001 on Broodforge, branch broodforge/pc/s0001, driver Jason.\n" +
		"Mode: asks first (this channel's default). Credential: cred-7k2m (Claude subscription).\n" +
		"Model: the engine's own.\n" +
		"Reach: code. Ends after 60 minutes without activity. Who may answer prompts: the driver.\n" +
		"Write what you want done. Everyone who writes here is handed to the agent by name."
	if p := (*posts)[1]; p.Body != header || !p.Pinned || p.Buttons[0].CustomID != "bf:s0001:end" {
		t.Errorf("header %+v", p)
	}
	v.Handle(ev(session.EventTurnStarted, time.Second, func(e *session.Event) { e.Turn = 3 }))
	v.Handle(ev(session.EventText, 2*time.Second, func(e *session.Event) {
		e.Turn, e.Text = 3, "Let me look at how the config loader reads the file."
	}))
	v.Handle(ev(session.EventToolCall, 3*time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Summary = 3, "t1", "[Read] src/config/load.ts"
	}))
	v.Handle(ev(session.EventToolCall, 4*time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Tool, e.Target, e.Summary = 3, "t2", "Bash", "npm install left-pad", "[Bash] npm install left-pad"
	}))
	v.Handle(ev(session.EventWaitingForAnswer, 5*time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Tool, e.Target, e.Text = 3, "t2", "Bash", "npm install left-pad", "npm install left-pad --save"
	}))
	var prompt Post
	for _, p := range *posts {
		if strings.HasPrefix(p.Body, "<@1001> Permission needed") {
			prompt = p
		}
	}
	if prompt.Body != "<@1001> Permission needed · turn 3\n[Bash] npm install left-pad\nCommand: npm install left-pad --save" ||
		len(prompt.Buttons) != 3 || prompt.Buttons[2].Label != "Allow Bash this session" ||
		prompt.Buttons[2].CustomID != "bf:s0001:t2:allow-session" {
		t.Errorf("prompt %+v", prompt)
	}
	v.Handle(ev(session.EventPermissionDecision, 6*time.Second, func(e *session.Event) {
		e.ToolUseID, e.Decision, e.DecidedBy, e.User = "t2", session.DecisionAllow, session.DecidedByParticipant, "1001"
	}))
	if p := (*posts)[len(*posts)-1]; p.MessageID != prompt.MessageID || p.Op != OpEdit || p.Body != "Allowed by Jason\n[Bash] npm install left-pad" || len(p.Buttons) != 0 {
		t.Errorf("answered prompt %+v", p)
	}
	v.Handle(ev(session.EventToolResult, 7*time.Second, func(e *session.Event) {
		e.ToolUseID, e.Summary = "t2", "[Bash] npm install left-pad"
	}))
	v.Handle(ev(session.EventTurnEnded, 102*time.Second, func(e *session.Event) {
		e.Turn, e.Result = 3, session.ResultCompleted
		e.Usage = &session.TurnUsage{In: 48200, Out: 1100, SessionIn: 212000, Model: "claude-opus-5-5", Mode: "prompt"}
	}))
	var final Post
	deletes := 0
	for _, p := range *posts {
		switch {
		case p.Op == OpDelete:
			deletes++
			if final.Body == "" {
				t.Error("a working message was deleted before the final content was posted")
			}
		case p.Op == OpCreate && strings.HasPrefix(p.Body, "Let me look"):
			final = p
		}
	}
	want := "Let me look at how the config loader reads the file.\n[Read] src/config/load.ts\n[Bash] npm install left-pad\n" +
		"Idle · ended on a tool call, no closing message · turn 3 · 1:41 · 48.2k in / 1.1k out · session 212k · Opus · asks first"
	if final.Body != want {
		t.Errorf("final\n got %q\nwant %q", final.Body, want)
	}
	if deletes != 2 { // the first process's working message and the resumed one's
		t.Errorf("%d deletes, want 2", deletes)
	}
}

// TestViewOverflow: a working message near Discord's limit is frozen and continued (R-22).
func TestViewOverflow(t *testing.T) {
	v, posts := newTestView()
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
	v.Handle(ev(session.EventTurnStarted, time.Second, func(e *session.Event) { e.Turn = 1 }))
	for i := 0; i < 100; i++ {
		v.Handle(ev(session.EventToolCall, time.Second, func(e *session.Event) {
			e.Turn, e.ToolUseID, e.Summary = 1, "t"+strconv.Itoa(i), "[Read] internal/some/long/path/to/a/file"+strconv.Itoa(i)+".go"
		}))
	}
	msgs := map[string]bool{}
	for _, p := range *posts {
		if runes(p.Body) > MessageLimit {
			t.Fatalf("a %d-character post", runes(p.Body))
		}
		if strings.HasPrefix(p.Body, "Working") {
			msgs[p.MessageID] = true
		}
	}
	if len(msgs) < 2 {
		t.Errorf("%d working messages for 100 lines", len(msgs))
	}
}

// TestReplayBurst: tests/discord/burst.jsonl keeps every session to one create or edit a second.
func TestReplayBurst(t *testing.T) {
	res := Replay(readReplay(t, "../../tests/discord/burst.jsonl"), true)
	if res.MaxPerSession != 1 || res.MaxPerBot > GlobalBudget {
		t.Errorf("per session %d, per bot %d", res.MaxPerSession, res.MaxPerBot)
	}
	finals := 0
	for _, p := range res.Posts {
		if strings.Contains(p.Body, KeyPrefix) {
			t.Errorf("an unmasked key in %s", p.MessageID)
		}
		if strings.Contains(p.Body, "\nIdle · turn 1 · ") {
			finals++
		}
	}
	if finals != 2 {
		t.Errorf("%d final messages, want 2", finals)
	}
}

func readReplay(t *testing.T, path string) []ReplayEvent {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var events []ReplayEvent
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			var e ReplayEvent
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				t.Fatal(err)
			}
			events = append(events, e)
		}
	}
	return events
}

// TestReplayTurnEnd: tests/discord/turn-end.jsonl. The working header names the step (a call
// outstanding by its tool, the agent's text as writing, none between steps), and a completed
// turn's footer says Idle, with `ended on a tool call, no closing message` when its last entry
// is a call (owner, 2026-10-24).
func TestReplayTurnEnd(t *testing.T) {
	res := Replay(readReplay(t, "../../tests/discord/turn-end.jsonl"), true)
	heads := map[string][]string{}
	finals := map[string]string{}
	for _, p := range res.Posts {
		first, _, _ := strings.Cut(p.Body, "\n")
		switch {
		case p.Op != OpCreate && p.Op != OpEdit:
		case strings.HasPrefix(first, "Working"):
			heads[p.Session] = append(heads[p.Session], first)
		case strings.Contains(p.Body, "\nIdle · "):
			finals[p.Session] = p.Body
		}
	}
	for id, want := range map[string][]string{
		"s0001": {"Working · turn 1 · 0:00", "Working · turn 1 · 0:02 · writing", "Working · turn 1 · 0:04 · running Grep",
			"Working · turn 1 · 0:06", "Working · turn 1 · 0:08 · running Grep", "Working · turn 1 · 0:10"},
		"s0002": {"Working · turn 1 · 0:00", "Working · turn 1 · 0:02 · running Bash", "Working · turn 1 · 0:04",
			"Working · turn 1 · 0:06 · writing"},
	} {
		if got := strings.Join(heads[id], "\n"); got != strings.Join(want, "\n") {
			t.Errorf("%s working headers\n got %q\nwant %q", id, heads[id], want)
		}
	}
	for id, want := range map[string]string{
		"s0001": "I will look for the callers.\n[Grep] load( in internal/\n[Grep] Load( in cmd/\n" +
			"Idle · ended on a tool call, no closing message · turn 1 · 0:12 · 12.0k in / 3.0k out · session 12.0k · Opus · asks first",
		"s0002": "[Bash] go test ./...\nAll tests pass.\nIdle · turn 1 · 0:08 · 21.4k in / 610 out · session 21.4k · Haiku · acts freely",
	} {
		if finals[id] != want {
			t.Errorf("%s final\n got %q\nwant %q", id, finals[id], want)
		}
	}
}

// TestViewStep: parallel calls show the newest still running, a turn without entries ends
// plain Idle, and the transcript's footer follows the view's.
func TestViewStep(t *testing.T) {
	v, posts := newTestView()
	last := func() string {
		first, _, _ := strings.Cut((*posts)[len(*posts)-1].Body, "\n")
		return first
	}
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
	v.Handle(ev(session.EventTurnStarted, 0, func(e *session.Event) { e.Turn = 2 }))
	if got := last(); got != "Working · turn 2 · 0:00" {
		t.Errorf("turn start: %s", got)
	}
	v.Handle(ev(session.EventToolCall, time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Tool, e.Summary = 2, "a", "Read", "[Read] a.go"
	}))
	v.Handle(ev(session.EventToolCall, time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Tool, e.Summary = 2, "b", "Grep", "[Grep] x in ."
	}))
	if got := last(); got != "Working · turn 2 · 0:01 · running Grep" {
		t.Errorf("two calls: %s", got)
	}
	v.Handle(ev(session.EventToolResult, 2*time.Second, func(e *session.Event) { e.ToolUseID, e.Summary = "b", "[Grep] x in ." }))
	if got := last(); got != "Working · turn 2 · 0:02 · running Read" {
		t.Errorf("one call left: %s", got)
	}
	v.Handle(ev(session.EventToolResult, 3*time.Second, func(e *session.Event) { e.ToolUseID, e.Summary = "a", "[Read] a.go" }))
	if got := last(); got != "Working · turn 2 · 0:03" {
		t.Errorf("no call left: %s", got)
	}
	v.Handle(ev(session.EventTurnEnded, 4*time.Second, func(e *session.Event) { e.Turn, e.Result = 2, session.ResultCompleted }))
	v.Handle(ev(session.EventTurnStarted, 5*time.Second, func(e *session.Event) { e.Turn = 3 }))
	v.Handle(ev(session.EventTurnEnded, 6*time.Second, func(e *session.Event) { e.Turn, e.Result = 3, session.ResultCompleted }))
	var footers []string
	for _, p := range *posts {
		if p.Op == OpCreate && strings.Contains(p.Body, "Idle · ") {
			lines := strings.Split(p.Body, "\n")
			footers = append(footers, lines[len(lines)-1])
		}
	}
	want := []string{"Idle · ended on a tool call, no closing message · turn 2 · 0:04 · 0 in / 0 out · session 0 · asks first",
		"Idle · turn 3 · 0:01 · 0 in / 0 out · session 0 · asks first"}
	if strings.Join(footers, "\n") != strings.Join(want, "\n") {
		t.Errorf("footers\n got %q\nwant %q", footers, want)
	}
	tr := Transcript("s0001", v.events, mention)
	for _, w := range want {
		if !strings.Contains(tr, "\n"+w+"\n") {
			t.Errorf("transcript lacks %q:\n%s", w, tr)
		}
	}

	// Calls that never get a result do not stay running, and the newest step wins.
	call := func(id, tool, parent string) session.Event {
		return ev(session.EventToolCall, time.Second, func(e *session.Event) {
			e.Turn, e.ToolUseID, e.Tool, e.Parent, e.Summary = 1, id, tool, parent, "["+tool+"] "+id
		})
	}
	result := func(id string) session.Event {
		return ev(session.EventToolResult, 2*time.Second, func(e *session.Event) { e.Turn, e.ToolUseID, e.Summary = 1, id, "[x] "+id })
	}
	decided := func(id, decision, by string) session.Event {
		return ev(session.EventPermissionDecision, time.Second, func(e *session.Event) {
			e.Turn, e.ToolUseID, e.Decision, e.DecidedBy = 1, id, decision, by
			if by == session.DecidedByParticipant {
				e.User = "1001"
			}
		})
	}
	text := ev(session.EventText, 2*time.Second, func(e *session.Event) { e.Turn, e.Text = 1, "Next." })
	waiting := ev(session.EventWaitingForAnswer, time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Tool, e.Target = 1, "b", "Bash", "npm test"
	})
	for name, c := range map[string]struct {
		events []session.Event
		want   string
	}{
		// The engine kept the message's last call (C28): the other is dropped and never has a result.
		"dropped parallel call": {[]session.Event{call("a", "Read", ""), call("b", "Bash", ""),
			decided("a", session.DecisionDropped, session.DecidedByPolicy), decided("b", session.DecisionDeferred, session.DecidedByPolicy),
			waiting, decided("b", session.DecisionAllow, session.DecidedByParticipant), result("b")}, "Working · turn 1 · 0:02"},
		"orphaned child": {[]session.Event{call("ag", "Agent", ""), call("c", "Bash", "ag"), call("d", "Read", "c"), result("ag")},
			"Working · turn 1 · 0:02"},
		"text after a stale call":  {[]session.Event{call("a", "Read", ""), text}, "Working · turn 1 · 0:02 · writing"},
		"call after text":          {[]session.Event{call("a", "Read", ""), text, call("b", "Grep", "")}, "Working · turn 1 · 0:01 · running Grep"},
		"result for an unknown id": {[]session.Event{result("zz")}, "Working · turn 1 · 0:02"},
		"newest call without tool": {[]session.Event{call("a", "Read", ""), call("b", "", "")}, "Working · turn 1 · 0:01"},
	} {
		v, posts := newTestView()
		v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
		v.Handle(ev(session.EventTurnStarted, 0, func(e *session.Event) { e.Turn = 1 }))
		for _, e := range c.events {
			v.Handle(e)
		}
		got := ""
		for _, p := range *posts {
			if first, _, _ := strings.Cut(p.Body, "\n"); strings.HasPrefix(first, "Working") {
				got = first
			}
		}
		if got != c.want {
			t.Errorf("%s: header %q, want %q", name, got, c.want)
		}
	}
}

// TestCoalescerFairness: 60 sessions with five posts each under the bot's 50 a second. Without
// rotation the lowest ids won every pass and the last ten waited 5 s for their first post; with
// it every session's first post is out within one window of the first.
func TestCoalescerFairness(t *testing.T) {
	first := map[string]time.Time{}
	co := NewCoalescer(func(now time.Time, p Post) {
		if _, ok := first[p.Session]; !ok {
			first[p.Session] = now
		}
	})
	t0 := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		for k := 0; k < 5; k++ {
			id := "s" + strconv.Itoa(1000+i)
			co.Offer(Post{Session: id, Op: OpCreate, MessageID: id + "-" + strconv.Itoa(k), Body: "x"})
		}
	}
	for now := t0; co.Pending() > 0; {
		co.Flush(now)
		next, ok := co.NextAt(now)
		if !ok {
			break
		}
		now = next
	}
	if len(first) != 60 {
		t.Fatalf("%d of 60 sessions posted", len(first))
	}
	for id, at := range first {
		if d := at.Sub(t0); d > GlobalWindow {
			t.Errorf("%s's first post %v after the first", id, d)
		}
	}
}
