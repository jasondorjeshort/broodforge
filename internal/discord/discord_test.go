// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"bufio"
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
		"the task-ant-colony test fails":   false,
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
	if got, want := footer(labelDone, 3, 102*time.Second, u), "Done · turn 3 · 1:42 · 48.2k in / 1.1k out · session 212k · Opus · asks first"; got != want {
		t.Errorf("footer\n got %s\nwant %s", got, want)
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
		"Done · turn 3 · 1:41 · 48.2k in / 1.1k out · session 212k · Opus · asks first"
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
	f, err := os.Open("../../tests/discord/burst.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var events []ReplayEvent
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := sc.Text(); line != "" && !strings.HasPrefix(line, "#") {
			var e ReplayEvent
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				t.Fatal(err)
			}
			events = append(events, e)
		}
	}
	res := Replay(events, true)
	if res.MaxPerSession != 1 || res.MaxPerBot > GlobalBudget {
		t.Errorf("per session %d, per bot %d", res.MaxPerSession, res.MaxPerBot)
	}
	finals := 0
	for _, p := range res.Posts {
		if strings.Contains(p.Body, KeyPrefix) {
			t.Errorf("an unmasked key in %s", p.MessageID)
		}
		if strings.Contains(p.Body, "\nDone · turn 1 · ") {
			finals++
		}
	}
	if finals != 2 {
		t.Errorf("%d final messages, want 2", finals)
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
