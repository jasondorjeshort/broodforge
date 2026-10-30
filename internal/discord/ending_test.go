// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// TestReplaySubagentEnd: tests/discord/subagent-end.jsonl, the subagent fixture's turn as
// `session tail --json` printed it and an end handing off two commits. The subagent's call sits
// indented under its Agent line in the final message (04, The turn loop); the venue says
// `Ending: checking and handing off 2 commits...` while the handoff runs, and that line is deleted
// once the summary is posted after it; the summary lists each commit's short hash and subject
// and the pull request's title (04, Ending a session). P11 remainder, items 1, 2 and 6.
func TestReplaySubagentEnd(t *testing.T) {
	res := Replay(readReplay(t, "../../tests/discord/subagent-end.jsonl"), true)
	var final, ending, summary string
	endingAt, summaryAt, deletedAt := -1, -1, -1
	for i, p := range res.Posts {
		switch {
		case p.Op == OpCreate && strings.Contains(p.Body, "\nIdle · "):
			final = p.Body
		case p.Op == OpCreate && strings.HasPrefix(p.Body, "Ending: "):
			ending, endingAt = p.MessageID, i
		case p.Op == OpCreate && p.Channel == "s0001" && strings.HasPrefix(p.Body, "Session s0001 ended by "):
			summary, summaryAt = p.Body, i
		case p.Op == OpDelete && p.MessageID == ending && ending != "":
			deletedAt = i
		}
	}
	if !strings.HasPrefix(final, "Asking the reviewer to read the change.\n[Agent]\n  [Read] hello.txt\n") {
		t.Errorf("the subagent's call is not indented under its Agent line:\n%s", final)
	}
	if endingAt < 0 || res.Posts[endingAt].Body != "Ending: checking and handing off 2 commits..." {
		t.Fatalf("no ending line in %+v", res.Posts)
	}
	if !(endingAt < summaryAt && summaryAt < deletedAt) {
		t.Errorf("ending %d, summary %d, ending deleted %d: want that order", endingAt, summaryAt, deletedAt)
	}
	want := "Session s0001 ended by Ada Example · 1m · 1 turn · 48 in / 22 out · Opus\n" +
		"Branch broodforge/scratch/s0001 -> pull request #42 https://github.com/example/scratch/pull/42\n" +
		"Title: Session s0001: Have the reviewer read the change.\n" +
		"Commits: a1b2c3d Add config reload · 9f8e7d6 Test last-good fallback\n" +
		"Author: Ada Example. Co-authors: Claude Opus 5.5.\n" +
		"Unmerged subagent branches removed: none."
	if summary != want {
		t.Errorf("summary\n got %s\nwant %s", summary, want)
	}
}

// TestViewEnding: the ending line's cases. None for nothing to hand off; `the commits` when the
// count is unknown; deleted after a refused handoff's text, which keeps the session open, and when
// the bot stops mid-handoff. The
// summary and its transcript also go to venue.summaryChannel, once when it is the lobby itself;
// a bypassed call's line keeps a subagent's indent.
func TestViewEnding(t *testing.T) {
	zero, three := 0, 3
	start := func(summaryTo string) (*View, *[]Post) {
		var posts []Post
		n := 0
		v := NewView("s0001", ViewConfig{NewID: func() string { n++; return "m" + strconv.Itoa(n) }, Name: mention,
			Offer: func(p Post) { posts = append(posts, p) }, SummaryChannel: func(lobby string) string {
				if lobby != "100" {
					t.Errorf("summary channel asked for %q", lobby)
				}
				return summaryTo
			}})
		v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
		return v, &posts
	}
	endings := func(posts []Post) []string {
		var out []string
		for _, p := range posts {
			if strings.HasPrefix(p.Body, "Ending: ") {
				out = append(out, p.Body)
			}
		}
		return out
	}
	ended := ev(session.EventSessionEnded, time.Minute, func(e *session.Event) {
		e.End = &session.EndInfo{By: "Jason", Outcome: "none"}
	})

	v, posts := start("")
	v.Handle(ev(session.EventSessionEnding, time.Second, func(e *session.Event) { e.Commits = &zero }))
	v.Handle(ended)
	if got := endings(*posts); len(got) != 0 {
		t.Errorf("nothing to hand off posted %q", got)
	}

	v, posts = start("")
	v.Handle(ev(session.EventSessionEnding, time.Second, nil))
	if got := endings(*posts); strings.Join(got, "|") != "Ending: checking and handing off the commits..." {
		t.Errorf("unknown count: %q", got)
	}

	v, posts = start("")
	v.Handle(ev(session.EventSessionEnding, time.Second, func(e *session.Event) { e.Commits = &three }))
	line := (*posts)[len(*posts)-1]
	if line.Body != "Ending: checking and handing off 3 commits..." {
		t.Fatalf("ending %q", line.Body)
	}
	v.Handle(ev(session.EventHandoffRefused, 2*time.Second, func(e *session.Event) { e.Text = "1 commit lacks the required attribution." }))
	n := len(*posts)
	if n < 2 || (*posts)[n-2].Op != OpCreate || !strings.HasPrefix((*posts)[n-2].Body, "Not handed off") ||
		(*posts)[n-1].Op != OpDelete || (*posts)[n-1].MessageID != line.MessageID {
		t.Errorf("after the refusal: %+v", (*posts)[n-2:])
	}
	v.Handle(ended) // ended later, by an escape: nothing left to delete
	for _, p := range (*posts)[n:] {
		if p.Op == OpDelete {
			t.Errorf("a second delete: %+v", p)
		}
	}

	// The bot stopping mid-handoff: the Ending line goes with the interruption.
	v, posts = start("")
	v.Handle(ev(session.EventSessionEnding, time.Second, func(e *session.Event) { e.Commits = &three }))
	line = (*posts)[len(*posts)-1]
	v.Handle(ev(session.EventSessionInterrupted, 2*time.Second, func(e *session.Event) { e.Summary = session.InterruptedShutdown }))
	if last := (*posts)[len(*posts)-1]; last.Op != OpDelete || last.MessageID != line.MessageID {
		t.Errorf("after the stop: %+v", last)
	}

	for _, c := range []struct{ to, want string }{{"900", "s0001 100 100 900 900"}, {"100", "s0001 100 100"}, {"", "s0001 100 100"}} {
		v, posts = start(c.to)
		v.Handle(ended)
		var got []string
		for _, p := range *posts {
			if p.Op == OpAttach || (p.Op == OpCreate && strings.HasPrefix(p.Body, "Session s0001 ended by ")) {
				got = append(got, p.Channel)
			}
		}
		if strings.Join(got, " ") != c.want {
			t.Errorf("summaryChannel %q: summary and transcript to %q, want %q", c.to, got, c.want)
		}
	}

	v, posts = start("")
	v.Handle(ev(session.EventTurnStarted, 0, func(e *session.Event) { e.Turn = 1 }))
	v.Handle(ev(session.EventToolCall, time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Tool, e.Target, e.Parent, e.Depth, e.Summary = 1, "r", "Read", "a.go", "ag", 1, "  [Read] a.go"
	}))
	v.Handle(ev(session.EventPermissionDecision, time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Decision, e.DecidedBy, e.Depth = 1, "r", session.DecisionAllow, session.DecidedByConfig, 1
	}))
	v.Handle(ev(session.EventToolResult, time.Second, func(e *session.Event) {
		e.Turn, e.ToolUseID, e.Tool, e.Target, e.Parent, e.Depth, e.Summary = 1, "r", "Read", "a.go", "ag", 1, "  [Read] a.go"
	}))
	last := (*posts)[len(*posts)-1].Body
	if !strings.Contains(last, "\n  [Read] a.go -> ran without Broodforge's check") {
		t.Errorf("the bypassed subagent call lost its indent:\n%s", last)
	}
}

// TestSummaryCommits: the commit list is bounded so the summary stays one message: ten commits,
// then a count, and a long subject or title cut.
func TestSummaryCommits(t *testing.T) {
	var log []session.CommitLine
	for i := range 13 {
		log = append(log, session.CommitLine{Hash: strconv.Itoa(1000000 + i), Subject: "Commit " + strconv.Itoa(i)})
	}
	log[0].Subject = strings.Repeat("x", 100)
	got := summaryText("s0001", &session.EndInfo{By: "Jason", Branch: "b", Outcome: "pull-request", Commits: 13, Log: log,
		URL: "https://github.com/o/r/pull/7", Title: strings.Repeat("t", 300)}, time.Now())
	if !strings.Contains(got, "\nCommits: 1000000 "+strings.Repeat("x", 69)+"... · 1000001 Commit 1 · ") ||
		!strings.Contains(got, " · 1000009 Commit 9 · and 3 more\n") || !strings.Contains(got, "\nTitle: "+strings.Repeat("t", 197)+"...\n") {
		t.Errorf("summary:\n%s", got)
	}
	if runes(got) > MessageLimit {
		t.Errorf("the summary is %d runes", runes(got))
	}
	// A handoff that did not read its commits keeps the count.
	if got := summaryText("s0001", &session.EndInfo{By: "Jason", Branch: "b", Outcome: "failed: x", Commits: 2}, time.Now()); !strings.Contains(got, "\nCommits: 2\n") {
		t.Errorf("count only:\n%s", got)
	}
}
