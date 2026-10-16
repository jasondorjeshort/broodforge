// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"sort"
	"sync"
	"time"
)

// Clock is the injectable clock the idle and prompt timeouts run on: the real one in the bot,
// a manual one in tests (03 session row; 05: the timeouts driven from a test).
type Clock interface {
	Now() time.Time
	// AfterFunc calls f on its own goroutine once d has passed.
	AfterFunc(d time.Duration, f func()) Timer
}

// Timer is a pending AfterFunc; Stop reports whether it stopped the call.
type Timer interface {
	Stop() bool
}

// RealClock is the wall clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

func (RealClock) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }

// ManualClock is a clock that moves only when told: Advance runs every timer that falls due,
// in time order, on the caller's goroutine.
type ManualClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*manualTimer
}

// NewManualClock starts a manual clock at t.
func NewManualClock(t time.Time) *ManualClock { return &ManualClock{now: t} }

type manualTimer struct {
	c       *ManualClock
	at      time.Time
	f       func()
	stopped bool
}

func (c *ManualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *ManualClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &manualTimer{c: c, at: c.now.Add(d), f: f}
	c.timers = append(c.timers, t)
	return t
}

func (t *manualTimer) Stop() bool {
	t.c.mu.Lock()
	defer t.c.mu.Unlock()
	was := !t.stopped
	t.stopped = true
	return was
}

// Advance moves the clock by d and runs the timers due by then.
func (c *ManualClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	var due []*manualTimer
	keep := c.timers[:0]
	for _, t := range c.timers {
		switch {
		case t.stopped:
		case !t.at.After(c.now):
			t.stopped = true
			due = append(due, t)
		default:
			keep = append(keep, t)
		}
	}
	c.timers = keep
	c.mu.Unlock()
	sort.SliceStable(due, func(i, j int) bool { return due[i].at.Before(due[j].at) })
	for _, t := range due {
		t.f()
	}
}

// Pending is how many timers wait to fire.
func (c *ManualClock) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, t := range c.timers {
		if !t.stopped {
			n++
		}
	}
	return n
}
