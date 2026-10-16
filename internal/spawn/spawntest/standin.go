// SPDX-License-Identifier: GPL-3.0-or-later

// Package spawntest is a stand-in engine for tests: a test binary that calls Main from its
// TestMain becomes, when the spawn runs it with Env set, a process that answers --version,
// records what it was given and replays a recorded turn's stdout, so the spawn, the
// claude-code engine and the CLI's --real commands are proven without a real engine turn.
package spawntest

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Environment names the stand-in reads. They pass the spawn's strip, which removes only the
// engine's and Broodforge's own prefixes.
const (
	Env        = "BFTEST_STANDIN"         // set: run as the stand-in
	EnvVersion = "BFTEST_STANDIN_VERSION" // --version prints "<this> (Claude Code)"; default 2.1.296
	EnvFixture = "BFTEST_STANDIN_FIXTURE" // a turn-<n>.jsonl copied to stdout
	EnvLog     = "BFTEST_STANDIN_LOG"     // where the Record goes, as JSON
	EnvSleep   = "BFTEST_STANDIN_SLEEP"   // a duration slept after the fixture, before exiting
	EnvExit    = "BFTEST_STANDIN_EXIT"    // the exit code; default 0
)

// Record is what one stand-in run saw: never an environment value except the names below.
type Record struct {
	Args     []string          `json:"args"`
	Cwd      string            `json:"cwd"`
	EnvNames []string          `json:"envNames"`
	Values   map[string]string `json:"values"` // CLAUDE_CONFIG_DIR, BROODFORGE_*, GIT_* only
	Stdin    string            `json:"stdin"`
	Settings string            `json:"settings"` // <CLAUDE_CONFIG_DIR>/settings.json as found
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
	stdin, _ := io.ReadAll(os.Stdin)
	if path := os.Getenv(EnvLog); path != "" {
		writeRecord(path, string(stdin))
	}
	if path := os.Getenv(EnvFixture); path != "" {
		if b, err := os.ReadFile(path); err == nil {
			os.Stdout.Write(b)
		}
	}
	if d, err := time.ParseDuration(os.Getenv(EnvSleep)); err == nil {
		time.Sleep(d)
	}
	code, _ := strconv.Atoi(os.Getenv(EnvExit))
	os.Exit(code)
}

func writeRecord(path, stdin string) {
	cwd, _ := os.Getwd()
	r := Record{Args: os.Args[1:], Cwd: filepath.ToSlash(cwd), Values: map[string]string{}, Stdin: stdin}
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
