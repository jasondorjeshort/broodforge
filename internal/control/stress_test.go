// SPDX-License-Identifier: GPL-3.0-or-later

package control_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/control/controltest"
)

// TestStressUnderGC drives many connections, each carrying many concurrent ids, while a
// goroutine runs the garbage collector in a loop, so goroutine stacks shrink and move and freed
// stack memory is reused often. Memory the kernel reads or writes after a call has returned
// (an OVERLAPPED, an I/O buffer, the handle array of WaitForMultipleObjects) must not live on a
// stack; when one does, this test fails with an invalid handle, a lost connection or a
// corrupted line. Run it with -race -count=20.
func TestStressUnderGC(t *testing.T) {
	stop := make(chan struct{})
	gcDone := make(chan struct{})
	go func() {
		defer close(gcDone)
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
				time.Sleep(time.Millisecond)
			}
		}
	}()
	defer func() { close(stop); <-gcDone }()

	var cancels atomic.Int64
	h := controltest.Start(t, func(r *control.Registry) {
		r.Handle("test.block", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			if err := call.Emit(json.RawMessage(`{"started":true}`)); err != nil {
				return nil, err
			}
			<-ctx.Done()
			cancels.Add(1)
			return nil, ctx.Err()
		})
	})
	const rounds, conns, per, blocks = 5, 8, 300, 5
	for round := range rounds {
		var wg sync.WaitGroup
		for ci := range conns {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stressConn(t, h.Addr, ci, per, blocks)
			}()
		}
		wg.Wait()
		if t.Failed() {
			t.Fatalf("round %d failed", round)
		}
		var cwg sync.WaitGroup
		for range 64 {
			cwg.Add(1)
			go func() {
				defer cwg.Done()
				got, err := control.Do(timeout(t), h.Addr, control.CmdEcho, json.RawMessage(`{"events":5}`), nil)
				if err != nil || string(got) != `{"text":"ok"}` {
					t.Errorf("round %d: Do = %s, %v", round, got, err)
				}
			}()
		}
		cwg.Wait()
	}
	want := int64(rounds * conns * blocks)
	deadline := time.Now().Add(5 * time.Second)
	for cancels.Load() < want && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := cancels.Load(); got != want {
		t.Fatalf("cancelled %d of %d blocked calls after their connections closed", got, want)
	}
}

// stressConn sends per echo requests (every seventh refused) and blocks test.block calls on
// one connection, writing while it reads, and checks every event and final line.
func stressConn(t *testing.T, addr string, ci, per, blocks int) {
	c, err := control.Dial(timeout(t), addr)
	if err != nil {
		t.Error(err)
		return
	}
	defer c.Close()
	go func() {
		w := bufio.NewWriter(c)
		for i := 1; i <= per; i++ {
			fail := ""
			if i%7 == 0 {
				fail = `,"fail":"x-y"`
			}
			fmt.Fprintf(w, `{"id":%d,"cmd":"control.echo","args":{"events":%d,"text":"c%d-%d"%s}}`+"\n", i, i%20, ci, i, fail)
			if i%50 == 0 {
				_ = w.Flush()
			}
		}
		for b := range blocks {
			fmt.Fprintf(w, `{"id":%d,"cmd":"test.block"}`+"\n", 100000+b)
		}
		_ = w.Flush()
	}()
	seen := map[int64]int{}
	final := map[int64]bool{}
	started := 0
	sc := bufio.NewScanner(c)
	sc.Buffer(nil, 1<<20)
	for len(final) < per || started < blocks {
		if !sc.Scan() {
			t.Errorf("conn %d: lines ended (%v) after %d finals, %d blocks started", ci, sc.Err(), len(final), started)
			return
		}
		var l control.Line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			t.Errorf("conn %d: bad line %q", ci, sc.Text())
			return
		}
		if l.ID >= 100000 {
			started++
			continue
		}
		if final[l.ID] {
			t.Errorf("conn %d: line after the final for id %d", ci, l.ID)
		}
		switch l.Kind() {
		case control.LineEvent:
			var e control.EchoEvent
			if err := json.Unmarshal(l.Event, &e); err != nil {
				t.Errorf("conn %d: event %s: %v", ci, l.Event, err)
			}
			seen[l.ID]++
			if e.Seq != seen[l.ID] || e.Text != fmt.Sprintf("c%d-%d", ci, l.ID) {
				t.Errorf("conn %d id %d: event %+v after %d", ci, l.ID, e, seen[l.ID]-1)
			}
		case control.LineResult, control.LineError:
			final[l.ID] = true
			if (l.ID%7 == 0) != (l.Kind() == control.LineError) {
				t.Errorf("conn %d id %d: kind %d (%s)", ci, l.ID, l.Kind(), sc.Text())
			}
			if int(l.ID%20) != seen[l.ID] {
				t.Errorf("conn %d id %d: %d events, want %d", ci, l.ID, seen[l.ID], l.ID%20)
			}
		case control.LineInvalid:
			t.Errorf("conn %d: invalid line %q", ci, sc.Text())
		}
	}
}
