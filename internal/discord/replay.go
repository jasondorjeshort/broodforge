// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"sort"
	"strconv"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// ReplayEvent is one line of a `discord render --events` file: a SessionEvent as `session tail
// --json` prints it, with the session it belongs to (default s0001).
type ReplayEvent struct {
	session.Event
	Session string `json:"session"`
}

// ReplayResult is what the renderer and the coalescer made of a file of events: the posts in
// the order sent, and the limits measured on them.
type ReplayResult struct {
	Posts []Post `json:"-"`
	// Requests is every post; MaxPerSession the most creates and edits one session sent within
	// any one second; MaxPerBot the most posts of all sessions within any one second.
	Requests      int `json:"requests"`
	MaxPerSession int `json:"maxPerSession"`
	MaxPerBot     int `json:"maxPerBot"`
}

// Replay renders events (in time order, a tie in file order; their times are when they happen) through one View
// per session and one Coalescer. With fake, the coalescer runs on a virtual clock that steps
// from one due time to the next, so a burst over seconds replays at once; otherwise it runs on
// the wall clock, events fed as fast as they come, and the replay takes as long as the limits
// make it. Names come from the events' session-started (others show as mentions).
func Replay(events []ReplayEvent, fake bool) ReplayResult {
	events = append([]ReplayEvent(nil), events...)
	sort.SliceStable(events, func(i, j int) bool { return parseTime(events[i].Time).Before(parseTime(events[j].Time)) })
	out := NewOutbox()
	co := NewCoalescer(func(now time.Time, p Post) { out.add(now, p) })
	n := 0
	newID := func() string { n++; return "m" + strconv.Itoa(n) }
	views := map[string]*View{}
	view := func(id string) *View {
		if v := views[id]; v != nil {
			return v
		}
		v := NewView(id, ViewConfig{NewID: newID, Name: mention, Offer: co.Offer})
		views[id] = v
		return v
	}
	if fake {
		var now time.Time
		step := func(until time.Time, all bool) {
			for {
				next, ok := co.NextAt(now)
				if !ok || (!all && next.After(until)) {
					return
				}
				if next.After(now) {
					now = next
				}
				co.Flush(now)
			}
		}
		for _, ev := range events {
			t := parseTime(ev.Time)
			if now.IsZero() {
				now = t
			}
			step(t, false)
			if t.After(now) {
				now = t
			}
			view(sessionOr(ev.Session)).Handle(ev.Event)
			co.Flush(now)
		}
		step(now, true)
	} else {
		for _, ev := range events {
			view(sessionOr(ev.Session)).Handle(ev.Event)
		}
		for co.Pending() > 0 {
			now := time.Now()
			co.Flush(now)
			if next, ok := co.NextAt(now); ok && next.After(now) {
				time.Sleep(next.Sub(now))
			}
		}
	}
	posts := out.Posts("", 0)
	return ReplayResult{Posts: posts, Requests: len(posts), MaxPerSession: maxPerSession(posts), MaxPerBot: maxInWindow(times(posts, ""))}
}

func sessionOr(id string) string {
	if id == "" {
		return "s0001"
	}
	return id
}

func times(posts []Post, id string) []time.Time {
	var ts []time.Time
	for _, p := range posts {
		if id != "" && (p.Session != id || !limited(p.Op)) {
			continue
		}
		if t, err := time.Parse(TimeLayout, p.Time); err == nil {
			ts = append(ts, t)
		}
	}
	return ts
}

func maxPerSession(posts []Post) int {
	seen := map[string]bool{}
	best := 0
	for _, p := range posts {
		if seen[p.Session] {
			continue
		}
		seen[p.Session] = true
		if m := maxInWindow(times(posts, p.Session)); m > best {
			best = m
		}
	}
	return best
}

// maxInWindow is the most times within any half-open second [t, t+1s).
func maxInWindow(ts []time.Time) int {
	sort.Slice(ts, func(i, j int) bool { return ts[i].Before(ts[j]) })
	best, j := 0, 0
	for i := range ts {
		for j < len(ts) && ts[j].Before(ts[i].Add(time.Second)) {
			j++
		}
		if j-i > best {
			best = j - i
		}
	}
	return best
}
