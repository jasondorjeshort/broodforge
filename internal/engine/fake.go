// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// FixtureFile is the name of a fixture directory's description.
const FixtureFile = "fixture.json"

// Fixture is fixture.json (fixtures/engine/README.md).
type Fixture struct {
	Claim         string        `json:"claim"`
	EngineVersion string        `json:"engineVersion"`
	Recorded      string        `json:"recorded"`
	DelayMs       int           `json:"delayMs,omitempty"` // between replayed lines; absent is none
	Turns         []FixtureTurn `json:"turns"`
}

// FixtureTurn is one recorded turn's outcome.
type FixtureTurn struct {
	N               int    `json:"n"`
	Exit            int    `json:"exit"`
	EngineSessionID string `json:"engineSessionId"`
}

// ReadFixture reads dir's fixture.json. A directory without one is a bare copy of recorded
// turns (05, Reproduce): no delay, every turn exits 0. A missing directory is an error.
func ReadFixture(dir string) (Fixture, error) {
	var fx Fixture
	if st, err := os.Stat(dir); err != nil {
		return fx, err
	} else if !st.IsDir() {
		return fx, fmt.Errorf("%s: not a directory", dir)
	}
	b, err := os.ReadFile(filepath.Join(dir, FixtureFile))
	if errors.Is(err, fs.ErrNotExist) {
		return fx, nil
	}
	if err != nil {
		return fx, err
	}
	if err := json.Unmarshal(b, &fx); err != nil {
		return fx, fixtureError(filepath.ToSlash(filepath.Join(dir, FixtureFile)), b, err)
	}
	return fx, nil
}

// Turn returns turn n's record and whether fixture.json has one.
func (fx Fixture) Turn(n int) (FixtureTurn, bool) {
	for _, t := range fx.Turns {
		if t.N == n {
			return t, true
		}
	}
	return FixtureTurn{N: n}, false
}

// TurnFile is the path of turn n's recorded stream in dir.
func TurnFile(dir string, n int) string {
	return filepath.Join(dir, "turn-"+strconv.Itoa(n)+".jsonl")
}

// deliverExit sends the turn's last event. Until Interrupt it waits for the reader, as every
// send does; after it, it never blocks: an event the reader left in the slot is dropped to make
// room, so the replay ends even when nobody reads again. The replay is the only sender, so the
// slot is free after the drop.
func deliverExit(events chan EngineEvent, stop <-chan struct{}, exit EngineEvent) {
	select {
	case events <- exit:
	case <-stop:
		select {
		case <-events:
		default:
		}
		events <- exit
	}
}

// ErrReceivedDir wraps a failure to use the directory received-<n>.json goes to.
var ErrReceivedDir = errors.New("received dir")

// ErrNoTurn is returned by StartTurn when the fixture has no file for the turn.
var ErrNoTurn = errors.New("no such turn")

// Fake is the engine that replays a fixture directory (03, The engine adapter): its n-th
// StartTurn plays turn-<n>.jsonl through ParseLine, then an exit carrying fixture.json's exit
// code, and writes the request it received as received-<n>.json.
type Fake struct {
	dir         string
	receivedDir string
	fixture     Fixture

	mu   sync.Mutex
	next int
}

// NewFake opens a fixture directory. receivedDir is where received-<n>.json go; "" is dir.
func NewFake(dir, receivedDir string) (*Fake, error) {
	fx, err := ReadFixture(dir)
	if err != nil {
		return nil, err
	}
	if receivedDir == "" {
		receivedDir = dir
	}
	if st, err := os.Stat(receivedDir); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReceivedDir, err)
	} else if !st.IsDir() {
		return nil, fmt.Errorf("%w: %s is not a directory", ErrReceivedDir, receivedDir)
	}
	return &Fake{dir: dir, receivedDir: receivedDir, fixture: fx, next: 1}, nil
}

var _ Engine = (*Fake)(nil)

// Name is the harness name configuration uses.
func (f *Fake) Name() string { return "fake" }

// Fixture is the directory's fixture.json.
func (f *Fake) Fixture() Fixture { return f.fixture }

// SetNextTurn makes the next StartTurn play turn n.
func (f *Fake) SetNextTurn(n int) {
	f.mu.Lock()
	f.next = n
	f.mu.Unlock()
}

// StartTurn plays the next turn. The events go out one by one, fixture.json's delayMs apart;
// Interrupt stops the replay before the next line and the exit then carries interrupted: true
// and no code.
func (f *Fake) StartTurn(req TurnRequest) (TurnHandle, error) {
	f.mu.Lock()
	n := f.next
	f.next++
	f.mu.Unlock()

	data, err := os.ReadFile(TurnFile(f.dir, n))
	if errors.Is(err, fs.ErrNotExist) {
		return TurnHandle{}, fmt.Errorf("%w: %s", ErrNoTurn, TurnFile(f.dir, n))
	}
	if err != nil {
		return TurnHandle{}, err
	}
	// The request as given, and the flags it sets on the real engine's command line.
	rec, err := json.MarshalIndent(struct {
		TurnRequest
		Args []string `json:"args"`
	}{req, nonNil(RequestArgs(req))}, "", "  ")
	if err != nil {
		return TurnHandle{}, err
	}
	received := filepath.Join(f.receivedDir, "received-"+strconv.Itoa(n)+".json")
	if err := os.WriteFile(received, append(rec, '\n'), 0o644); err != nil {
		return TurnHandle{}, fmt.Errorf("%w: %w", ErrReceivedDir, err)
	}

	turn, _ := f.fixture.Turn(n)
	delay := time.Duration(f.fixture.DelayMs) * time.Millisecond
	// One slot: once Interrupt is called the reader may walk away, so the exit must go in
	// without a reader (deliverExit).
	events := make(chan EngineEvent, 1)
	stop := make(chan struct{})
	var once sync.Once
	go func() {
		defer close(events)
		interrupted := false
		first := true
	replay:
		for _, line := range bytes.Split(data, []byte("\n")) {
			if len(bytes.TrimSpace(line)) == 0 {
				continue // no delay before a blank line or the piece after the last newline
			}
			if !first && delay > 0 {
				t := time.NewTimer(delay)
				select {
				case <-t.C:
				case <-stop:
					t.Stop()
					interrupted = true
					break replay
				}
			}
			first = false
			for _, ev := range ParseLine(line) {
				select {
				case <-stop:
					interrupted = true
					break replay
				default:
				}
				select {
				case events <- ev:
				case <-stop:
					interrupted = true
					break replay
				}
			}
		}
		exit := &ExitEvent{Interrupted: interrupted}
		if !interrupted {
			code := turn.Exit
			exit.Code = &code
		}
		deliverExit(events, stop, EngineEvent{Kind: KindExit, Exit: exit})
	}()
	return TurnHandle{
		Events:    events,
		Interrupt: func() error { once.Do(func() { close(stop) }); return nil },
	}, nil
}

// fixtureError says what is wrong with fixture.json in one sentence, its path slash-separated:
// a syntax error by its line, anything else as the decoder put it.
func fixtureError(path string, src []byte, err error) error {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		off := min(int(se.Offset), len(src))
		return fmt.Errorf("%s is not valid JSON at line %d", path, 1+bytes.Count(src[:off], []byte("\n")))
	}
	return fmt.Errorf("%s: %w", path, err)
}
