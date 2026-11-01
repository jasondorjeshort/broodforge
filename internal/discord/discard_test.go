// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// TestDiscardTexts: the owner's lines of 04-ux-session-end.md, the confirm in its three counts
// and the two summaries' branch lines.
func TestDiscardTexts(t *testing.T) {
	for n, want := range map[int]string{
		3:                      "Discard session s0001? Its 3 commits stay recoverable.",
		1:                      "Discard session s0001? Its 1 commit stays recoverable.",
		0:                      "Discard session s0001? Its 0 commits stay recoverable.",
		session.CommitsUnknown: "Discard session s0001? Its commits stay recoverable.",
	} {
		if got := discardConfirmText("s0001", n); got != want {
			t.Errorf("confirm %d: %q", n, got)
		}
	}
	now := time.Now()
	wip := summaryText("s0001", &session.EndInfo{By: "Ada", Branch: "b/s0001", Outcome: "pushed", Commits: 1, WIP: "abc",
		Log: []session.CommitLine{{Hash: "abcdef0123", Subject: "WIP: uncommitted changes committed at end"}}}, now)
	if lines := strings.Split(wip, "\n"); len(lines) < 3 ||
		lines[1] != "1 commit · pushed to b/s0001 (WIP: uncommitted changes committed at end)" ||
		lines[2] != "Commits: abcdef0 WIP: uncommitted changes committed at end" {
		t.Errorf("WIP summary:\n%s", wip)
	}
	pr := summaryText("s0001", &session.EndInfo{By: "Ada", Branch: "b/s0001", Outcome: "pull-request", Commits: 2, WIP: "abc",
		URL: "https://github.com/o/r/pull/7"}, now)
	if !strings.Contains(pr, "\nBranch b/s0001 -> pull request #7 https://github.com/o/r/pull/7 (WIP: uncommitted changes committed at end)\n") {
		t.Errorf("WIP request summary:\n%s", pr)
	}
	discarded := summaryText("s0001", &session.EndInfo{By: "Ada", Branch: "b/s0001", Outcome: "discarded", Commits: 2,
		Ref: "refs/aborted/s0001"}, now)
	if lines := strings.Split(discarded, "\n"); len(lines) < 3 || lines[1] != "2 commits · discarded; recoverable on request" ||
		lines[2] != "Kept as refs/aborted/s0001 in the project's canonical." {
		t.Errorf("discard summary:\n%s", discarded)
	}
	if got := summaryText("s0001", &session.EndInfo{By: "Ada", Outcome: "discarded"}, now); got !=
		"Session s0001 ended by Ada · 0 commits · nothing to push; the tree was removed." {
		t.Errorf("empty discard: %q", got)
	}
}

// TestViewConfirmDiscard: the confirm is a venue post with the Discard button; a second replaces
// the first; the end edits the standing one without its button, and none is posted after it.
func TestViewConfirmDiscard(t *testing.T) {
	v, posts := newTestView()
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = testStart }))
	v.ConfirmDiscard(2)
	first := (*posts)[len(*posts)-1]
	if first.Op != OpCreate || first.Body != "Discard session s0001? Its 2 commits stay recoverable." ||
		len(first.Buttons) != 1 || first.Buttons[0].Label != "Discard" || first.Buttons[0].CustomID != "bf:s0001:discard-confirm" {
		t.Fatalf("confirm %+v", first)
	}
	v.ConfirmDiscard(3)
	n := len(*posts)
	if del, again := (*posts)[n-2], (*posts)[n-1]; del.Op != OpDelete || del.MessageID != first.MessageID ||
		again.Op != OpCreate || !strings.Contains(again.Body, "Its 3 commits") {
		t.Fatalf("second confirm: %+v %+v", del, again)
	}
	second := (*posts)[n-1]
	v.Handle(ev(session.EventSessionEnded, time.Second, func(e *session.Event) {
		e.End = &session.EndInfo{By: "Jason", Outcome: "discarded", Commits: 3, Ref: "refs/aborted/s0001"}
	}))
	var edited bool
	for _, p := range (*posts)[n:] {
		if p.Op == OpEdit && p.MessageID == second.MessageID {
			edited = len(p.Buttons) == 0 && p.Body == second.Body
		}
	}
	if !edited {
		t.Errorf("the standing confirm kept its button: %+v", (*posts)[n:])
	}
	n = len(*posts)
	v.ConfirmDiscard(3)
	if len(*posts) != n {
		t.Errorf("a confirm after the end: %+v", (*posts)[n:])
	}
}
