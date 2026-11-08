// SPDX-License-Identifier: GPL-3.0-or-later

// Package spawntest is a stand-in engine for tests: a test binary that calls Main from its
// TestMain becomes, when the spawn runs it with Env set, a process that answers --version,
// records what it was given and replays a recorded turn's stdout, so the spawn, the
// claude-code engine and the CLI's --real commands are proven without a real engine turn.
package spawntest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Environment names the stand-in reads. They pass the spawn's strip, which removes only the
// engine's and Broodforge's own prefixes.
const (
	Env        = "BFTEST_STANDIN"         // set: run as the stand-in
	EnvVersion = "BFTEST_STANDIN_VERSION" // --version prints "<this> (Claude Code)"; default 2.1.296
	EnvFixture = "BFTEST_STANDIN_FIXTURE" // a turn-<n>.jsonl copied to stdout; unreadable, exit FixtureUnreadable
	EnvLog     = "BFTEST_STANDIN_LOG"     // where the Record goes, as JSON
	EnvSleep   = "BFTEST_STANDIN_SLEEP"   // a duration slept after the fixture, before exiting
	EnvExit    = "BFTEST_STANDIN_EXIT"    // the exit code; default 0
	// EnvWait set: after each control_request line of the fixture the stand-in waits for a
	// control_response line on stdin, or stdin's end, before the next line, as the engine waits
	// for an answer under --permission-prompt-tool stdio.
	EnvWait = "BFTEST_STANDIN_WAIT"
	// EnvHold "sleep": after the fixture the stand-in sleeps EnvSleep and exits without waiting
	// for stdin's end (an engine that exits, or hangs, whatever its stdin does). Unset, it
	// waits for stdin's end, then sleeps EnvSleep, then exits.
	EnvHold = "BFTEST_STANDIN_HOLD"
)

// FixtureUnreadable is the stand-in's exit code when EnvFixture names a file it cannot read.
const FixtureUnreadable = 3

// Record is what one stand-in run saw: never an environment value except the names below.
type Record struct {
	Args     []string          `json:"args"`
	Cwd      string            `json:"cwd"`
	EnvNames []string          `json:"envNames"`
	Values   map[string]string `json:"values"` // CLAUDE_CONFIG_DIR, BROODFORGE_*, GIT_* only
	Stdin    string            `json:"stdin"`
	// StdinClosed is set once stdin reached its end before the record was written.
	StdinClosed bool   `json:"stdinClosed"`
	Settings    string `json:"settings"` // <CLAUDE_CONFIG_DIR>/settings.json as found
}

// stdinLog is the stand-in's stdin as read so far; the record is rewritten on every change,
// so a process killed early still leaves what it read.
type stdinLog struct {
	mu     sync.Mutex
	cond   *sync.Cond
	text   strings.Builder
	lines  []string
	closed bool
	path   string
}

func (s *stdinLog) read(r io.Reader) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		s.mu.Lock()
		if line != "" {
			s.text.WriteString(line)
			s.lines = append(s.lines, line)
		}
		if err != nil {
			s.closed = true
		}
		s.writeLocked()
		s.mu.Unlock()
		s.cond.Broadcast()
		if err != nil {
			return
		}
	}
}

// waitResponse waits for a control_response line at index from or later, or stdin's end, and
// returns the index after the line it found.
func (s *stdinLog) waitResponse(from int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		for i := from; i < len(s.lines); i++ {
			if strings.HasPrefix(s.lines[i], `{"type":"control_response"`) {
				return i + 1
			}
		}
		from = len(s.lines)
		if s.closed {
			return from
		}
		s.cond.Wait()
	}
}

func (s *stdinLog) waitClosed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for !s.closed {
		s.cond.Wait()
	}
}

func (s *stdinLog) write() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeLocked()
}

func (s *stdinLog) writeLocked() {
	if s.path != "" {
		writeRecord(s.path, s.text.String(), s.closed)
	}
}

// Main runs the stand-in and exits when Env is set; otherwise it returns at once.
func Main() {
	if os.Getenv(Env) == "" {
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		v := os.Getenv(EnvVersion)
		if v == "" {
			v = "2.1.296"
		}
		os.Stdout.WriteString(v + " (Claude Code)\n")
		os.Exit(0)
	}
	in := &stdinLog{path: os.Getenv(EnvLog)}
	in.cond = sync.NewCond(&in.mu)
	in.write()
	go in.read(os.Stdin)
	if path := os.Getenv(EnvFixture); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			// An unreadable fixture (an MSYS path such as /c/... handed on unconverted) is the
			// caller's mistake: say so and exit, rather than print nothing and wait on a stdin a
			// prompt-mode adapter closes only at a result that will never come.
			os.Stderr.WriteString("stand-in: " + EnvFixture + ": " + err.Error() + "\n")
			in.write()
			os.Exit(FixtureUnreadable)
		}
		replay(b, in, os.Getenv(EnvWait) != "")
	}
	if os.Getenv(EnvHold) != "sleep" {
		in.waitClosed()
	}
	if d, err := time.ParseDuration(os.Getenv(EnvSleep)); err == nil {
		time.Sleep(d)
	}
	in.write()
	code, _ := strconv.Atoi(os.Getenv(EnvExit))
	os.Exit(code)
}

// replay copies the fixture to stdout, whole, or line by line waiting for an answer after
// each control_request line.
func replay(b []byte, in *stdinLog, wait bool) {
	if !wait {
		os.Stdout.Write(b)
		return
	}
	next := 0
	for _, line := range bytes.SplitAfter(b, []byte("\n")) {
		os.Stdout.Write(line)
		if bytes.HasPrefix(line, []byte(`{"type":"control_request"`)) {
			next = in.waitResponse(next)
		}
	}
}

func writeRecord(path, stdin string, closed bool) {
	cwd, _ := os.Getwd()
	r := Record{Args: os.Args[1:], Cwd: filepath.ToSlash(cwd), Values: map[string]string{}, Stdin: stdin, StdinClosed: closed}
	for _, kv := range os.Environ() {
		name, value, _ := strings.Cut(kv, "=")
		if name == "" {
			continue
		}
		r.EnvNames = append(r.EnvNames, name)
		up := strings.ToUpper(name)
		if up == "CLAUDE_CONFIG_DIR" || strings.HasPrefix(up, "BROODFORGE_") || strings.HasPrefix(up, "GIT_") {
			r.Values[name] = value
		}
	}
	sort.Strings(r.EnvNames)
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		if b, err := os.ReadFile(filepath.Join(dir, "settings.json")); err == nil {
			r.Settings = string(b)
		}
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile(path, b, 0o644)
}

// ReadRecord reads a stand-in's record.
func ReadRecord(path string) (Record, error) {
	var r Record
	b, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	return r, json.Unmarshal(b, &r)
}

// Has reports whether names holds name (case-insensitively, as Windows compares).
func (r Record) Has(name string) bool {
	for _, n := range r.EnvNames {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}
