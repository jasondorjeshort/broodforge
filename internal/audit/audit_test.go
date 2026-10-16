// SPDX-License-Identifier: GPL-3.0-or-later

package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Create(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func appendAll(t *testing.T, s *store.Store, w *Writer, evs ...Event) {
	t.Helper()
	err := s.Write(func(tx *store.Tx) error {
		for _, ev := range evs {
			if _, err := w.Append(tx, ev); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
}

func TestAppendQueryAndFileSink(t *testing.T) {
	s := newStore(t)
	sinkPath := filepath.Join(t.TempDir(), "audit.log")
	w := NewWriter(NewFileSink(sinkPath))
	w.now = func() time.Time { return time.Date(2026, 10, 16, 12, 0, 0, 500, time.UTC) }

	appendAll(t, s, w,
		Event{Kind: KindInstanceStarted, Actor: "instance"},
		Event{Kind: KindSessionStarted, Actor: "1001", Session: "s0001", Registration: "1001", Details: json.RawMessage(`{ "channel": "100" }`)},
		Event{Kind: KindPush, Phase: PhaseAttempted, Actor: "instance", Session: "s0001"},
		Event{Kind: KindPush, Phase: PhaseDone, Actor: "instance", Session: "s0001"},
	)
	// A rolled-back write leaves neither a row nor a sink line.
	boom := errors.New("boom")
	err := s.Write(func(tx *store.Tx) error {
		if _, err := w.Append(tx, Event{Kind: KindCommit, Actor: "1001"}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}

	all, err := Query(s, Filter{})
	if err != nil || len(all) != 4 {
		t.Fatalf("all: %d %v", len(all), err)
	}
	got, _ := all[1].MarshalJSON()
	want := `{"seq":2,"time":"2026-10-16T12:00:00Z","kind":"session-started","phase":null,"actor":"1001","session":"s0001","registration":"1001","details":{"channel":"100"}}`
	if string(got) != want {
		t.Errorf("event JSON\n%s\nwant\n%s", got, want)
	}

	newest, _ := Query(s, Filter{Limit: 2})
	if len(newest) != 2 || newest[0].Seq != 3 || newest[1].Seq != 4 {
		t.Errorf("newest 2: %+v", newest)
	}
	after, _ := Query(s, Filter{SinceSeq: 1, Limit: 2})
	if len(after) != 2 || after[0].Seq != 2 || after[1].Seq != 3 {
		t.Errorf("after 1, limit 2: %+v", after)
	}
	pushes, _ := Query(s, Filter{Kinds: []Kind{KindPush, KindCommit}, Session: "s0001"})
	if len(pushes) != 2 || pushes[0].Phase != PhaseAttempted || pushes[1].Phase != PhaseDone {
		t.Errorf("pushes: %+v", pushes)
	}

	b, err := os.ReadFile(sinkPath)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(b)), "\n"); len(lines) != 4 || lines[1] != want {
		t.Errorf("sink lines %q", lines)
	}

	var buf bytes.Buffer
	n, err := Export(s, &buf)
	if err != nil || n != 4 || buf.String() != string(b) {
		t.Errorf("export %d %v:\n%s\nwant the sink's lines", n, err, buf.String())
	}
}

func TestAppendRefusesBadEvents(t *testing.T) {
	s := newStore(t)
	w := NewWriter(nil)
	for _, ev := range []Event{
		{Kind: "Not_Kebab", Actor: "a"},
		{Kind: KindPush, Phase: "maybe", Actor: "a"},
		{Kind: KindPush},
		{Kind: KindPush, Actor: "a", Details: json.RawMessage(`{not json`)},
	} {
		err := s.Write(func(tx *store.Tx) error { _, err := w.Append(tx, ev); return err })
		if err == nil {
			t.Errorf("%+v was accepted", ev)
		}
	}
	if evs, _ := Query(s, Filter{}); len(evs) != 0 {
		t.Errorf("rows written: %+v", evs)
	}
}
