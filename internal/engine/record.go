// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"
)

// RecordedClaim is fixture.json's claim for a directory engine record started.
const RecordedClaim = "recorded"

var turnFileRE = regexp.MustCompile(`^turn-(\d+)\.jsonl$`)

// Recording is one turn being recorded into a fixture directory (05-debug-interface.md, A turn
// against the fake engine: engine record): the engine's stdout lines verbatim as
// turn-<n>.jsonl, n counting up from the directory's last turn, and its outcome added to
// fixture.json, which carries no delay. Nothing is redacted.
type Recording struct {
	dir     string
	n       int
	resume  string
	fixture Fixture
	file    *os.File

	mu  sync.Mutex
	err error
}

// NewRecording opens dir (made if missing) for its next turn. Resume is the engine session id
// of the previous turn: fixture.json's record, else the previous turn's system/init line.
func NewRecording(dir string) (*Recording, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	fx, err := ReadFixture(dir)
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	last := 0
	for _, e := range ents {
		if m := turnFileRE.FindStringSubmatch(e.Name()); m != nil {
			if k, err := strconv.Atoi(m[1]); err == nil && k > last {
				last = k
			}
		}
	}
	r := &Recording{dir: dir, n: last + 1, fixture: fx}
	if last > 0 {
		if t, ok := fx.Turn(last); ok && t.EngineSessionID != "" {
			r.resume = t.EngineSessionID
		} else {
			r.resume = sessionOf(TurnFile(dir, last))
		}
	}
	f, err := os.OpenFile(TurnFile(dir, r.n), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	r.file = f
	return r, nil
}

// sessionOf is the engine session id of a recorded turn's first init line, "" when none.
func sessionOf(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		for _, ev := range ParseLine(line) {
			if ev.Kind == KindInit && ev.Init.EngineSessionID != "" {
				return ev.Init.EngineSessionID
			}
		}
	}
	return ""
}

// Turn is the number of the turn being recorded.
func (r *Recording) Turn() int { return r.n }

// Resume is the session id the turn resumes, "" for the directory's first turn.
func (r *Recording) Resume() string { return r.resume }

// Path is the turn file being written.
func (r *Recording) Path() string { return filepath.ToSlash(TurnFile(r.dir, r.n)) }

// Write is the engine's Raw sink: one stdout line as written. The first error is kept for Finish.
func (r *Recording) Write(line []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		_, r.err = r.file.Write(line)
	}
}

// Finish closes the turn file and records the turn in fixture.json: its exit code (1 when it
// has none, as an interrupted turn), the engine session id seen, the engine version, the date.
func (r *Recording) Finish(exit *ExitEvent, engineSessionID, engineVersion string) error {
	r.mu.Lock()
	werr := r.err
	r.mu.Unlock()
	if err := r.file.Close(); werr == nil {
		werr = err
	}
	if werr != nil {
		return werr
	}
	code := 1
	if exit != nil && exit.Code != nil && !exit.Interrupted {
		code = *exit.Code
	}
	fx := r.fixture
	if fx.Claim == "" {
		fx.Claim = RecordedClaim
	}
	fx.EngineVersion = engineVersion
	fx.Recorded = time.Now().Format(time.DateOnly)
	turns := []FixtureTurn{{N: r.n, Exit: code, EngineSessionID: engineSessionID}}
	for _, t := range fx.Turns {
		if t.N != r.n {
			turns = append(turns, t)
		}
	}
	sort.Slice(turns, func(i, j int) bool { return turns[i].N < turns[j].N })
	fx.Turns = turns
	b, err := json.MarshalIndent(fx, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(r.dir, FixtureFile), append(b, '\n'))
}

// Abort removes the turn file of a turn that never started.
func (r *Recording) Abort() {
	_ = r.file.Close()
	_ = os.Remove(TurnFile(r.dir, r.n))
}

func writeFileAtomic(path string, content []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".fixture-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(content)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp, path)
	}
	if werr != nil {
		_ = os.Remove(tmp)
	}
	return werr
}
