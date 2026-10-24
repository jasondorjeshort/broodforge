// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// The rate limits of 03 Data flow, step 6 (R-20 to R-22): at most one create or edit per
// second per session, and 50 REST requests per second per bot across every session.
const (
	PerSession   = time.Second
	GlobalBudget = 50
	GlobalWindow = time.Second
)

// Coalescer holds every session's posts until the limits let them out. Nothing is dropped: an
// edit to a message already queued merges into the queued create or edit (the latest content
// and components win), a delete drops the message's queued edits, and every other post keeps its place.
// Time comes from the caller (Flush, NextAt), so a test or `discord render --clock fake` steps
// a virtual clock and Run steps the injectable one.
type Coalescer struct {
	mu     sync.Mutex
	queues map[string]*sessionQueue
	sent   []time.Time // the global window's sends, oldest first
	// after is the session served last: every pass starts with the one after it, so under a
	// full budget the sessions take turns instead of the lowest ids always going first.
	after string
	sink  func(now time.Time, p Post)
	wake  chan struct{}
}

type sessionQueue struct {
	posts []*Post
	last  time.Time // the last create or edit sent
	any   bool
	done  bool // the renderer's last post is queued: tell the outbox once the queue empties
}

// NewCoalescer returns a coalescer handing every post it lets out to sink.
func NewCoalescer(sink func(now time.Time, p Post)) *Coalescer {
	return &Coalescer{queues: map[string]*sessionQueue{}, sink: sink, wake: make(chan struct{}, 1)}
}

// limited is whether a post counts against the per-session limit: every message write does;
// a channel's creation and a delete count against the global budget only.
func limited(op Op) bool {
	switch op {
	case OpCreate, OpEdit, OpAttach:
		return true
	case OpChannel, OpDelete:
	}
	return false
}

// Offer queues a post for its session ("" is the bot's own, such as the panel).
func (c *Coalescer) Offer(p Post) {
	c.mu.Lock()
	q := c.queues[p.Session]
	if q == nil {
		q = &sessionQueue{}
		c.queues[p.Session] = q
	}
	switch p.Op {
	case OpEdit:
		for _, held := range q.posts {
			if held.MessageID == p.MessageID && (held.Op == OpCreate || held.Op == OpEdit) {
				// The later components win with the content: a switch's menus over the start's,
				// an end's none over a switch's (05-debug-interface-discord.md, The header's menus).
				held.Body, held.Buttons, held.Selects = p.Body, p.Buttons, p.Selects
				c.mu.Unlock()
				return
			}
		}
	case OpDelete:
		// The message's queued edits are moot; a queued create still goes out (with the latest
		// content merged in), so a turn shorter than the limit still shows its working
		// message, then the delete.
		keep := q.posts[:0]
		for _, held := range q.posts {
			if held.MessageID == p.MessageID && (held.Op == OpEdit || held.Op == OpAttach) {
				continue
			}
			keep = append(keep, held)
		}
		q.posts = keep
	case OpChannel, OpCreate, OpAttach:
	}
	q.posts = append(q.posts, &p)
	c.mu.Unlock()
	c.poke()
}

// Done marks the session's last post queued: the outbox learns it once the queue empties.
func (c *Coalescer) Done(id string) {
	c.mu.Lock()
	q := c.queues[id]
	if q == nil {
		q = &sessionQueue{}
		c.queues[id] = q
	}
	q.done = true
	c.mu.Unlock()
	c.poke()
}

func (c *Coalescer) poke() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *Coalescer) pruneLocked(now time.Time) {
	i := 0
	for i < len(c.sent) && !c.sent[i].After(now.Add(-GlobalWindow)) {
		i++
	}
	c.sent = c.sent[i:]
}

func (c *Coalescer) ids() []string {
	ids := make([]string, 0, len(c.queues))
	for id := range c.queues {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Flush sends every post the limits allow at now, one per session per pass so no session
// starves another, and returns the sessions whose last post went out.
func (c *Coalescer) Flush(now time.Time) []string {
	type out struct{ p Post }
	var sends []out
	var finished []string
	c.mu.Lock()
	c.pruneLocked(now)
	for progress := true; progress; {
		progress = false
		for _, id := range c.rotated() {
			q := c.queues[id]
			if len(q.posts) == 0 {
				continue
			}
			head := q.posts[0]
			if limited(head.Op) && q.any && now.Before(q.last.Add(PerSession)) {
				continue
			}
			if len(c.sent) >= GlobalBudget {
				break
			}
			q.posts = q.posts[1:]
			if limited(head.Op) {
				q.last, q.any = now, true
			}
			c.sent = append(c.sent, now)
			c.after = id // the next pass, and the next flush, start after it
			p := *head
			p.finish()
			sends = append(sends, out{p: p})
			progress = true
		}
	}
	for _, id := range c.ids() {
		if q := c.queues[id]; q.done && len(q.posts) == 0 {
			finished = append(finished, id)
			delete(c.queues, id)
		}
	}
	c.mu.Unlock()
	for _, s := range sends {
		c.sink(now, s.p)
	}
	return finished
}

// NextAt is when the next post may go out, false when nothing waits.
func (c *Coalescer) NextAt(now time.Time) (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(now)
	var next time.Time
	found := false
	for _, q := range c.queues {
		if len(q.posts) == 0 {
			if q.done {
				return now, true
			}
			continue
		}
		t := now
		if limited(q.posts[0].Op) && q.any && q.last.Add(PerSession).After(now) {
			t = q.last.Add(PerSession)
		}
		if !found || t.Before(next) {
			next, found = t, true
		}
	}
	if found && len(c.sent) >= GlobalBudget {
		if free := c.sent[0].Add(GlobalWindow); free.After(next) {
			next = free
		}
	}
	return next, found
}

// Pending is how many posts wait.
func (c *Coalescer) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, q := range c.queues {
		n += len(q.posts)
	}
	return n
}

// Run sends posts as the limits allow on clock until ctx ends; onDone hears of each session
// whose last post went out.
func (c *Coalescer) Run(ctx context.Context, clock session.Clock, onDone func(id string)) {
	fire := make(chan struct{}, 1)
	for {
		now := clock.Now()
		for _, id := range c.Flush(now) {
			onDone(id)
		}
		var timer session.Timer
		if next, ok := c.NextAt(now); ok {
			d := next.Sub(now)
			if d <= 0 {
				continue
			}
			timer = clock.AfterFunc(d, func() {
				select {
				case fire <- struct{}{}:
				default:
				}
			})
		}
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-c.wake:
		case <-fire:
		}
		if timer != nil {
			timer.Stop()
		}
	}
}

// rotated is the session ids in order, starting with the first after the one served last.
func (c *Coalescer) rotated() []string {
	ids := c.ids()
	i := sort.SearchStrings(ids, c.after)
	if i < len(ids) && ids[i] == c.after {
		i++
	}
	return append(ids[i:], ids[:i]...)
}
