// SPDX-License-Identifier: GPL-3.0-or-later

// Package clitest builds bin/broodforge once per test package and runs it, so a test proves
// the binary's own output and exit codes (never go run's). A package that uses it adds
//
//	func TestMain(m *testing.M) { os.Exit(clitest.Main(m)) }
//
// which removes the build directory after the tests.
package clitest

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	once     sync.Once
	dir      string
	bin      string
	buildErr error
)

// Main runs the tests and then deletes the binary's directory. It returns m.Run's code.
func Main(m *testing.M) int {
	code := m.Run()
	if dir != "" {
		_ = os.RemoveAll(dir)
	}
	return code
}

// Binary returns the path of the binary, building it on first use with the plan's build line,
// `CGO_ENABLED=0 go build -o <dir>/ ./cmd/broodforge`, at the module root.
func Binary(t testing.TB) string {
	t.Helper()
	once.Do(build)
	if buildErr != nil {
		t.Fatalf("build broodforge: %v", buildErr)
	}
	return bin
}

func build() {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		buildErr = err
		return
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		buildErr = errors.New("not inside the module")
		return
	}
	if dir, err = os.MkdirTemp("", "broodforge-clitest-"); err != nil {
		buildErr = err
		return
	}
	// The binary reads no machine literal (config.MachineLiterals), so no test depends on the
	// user and host names of the machine it runs on.
	cmd := exec.Command("go", "build", "-ldflags=-X github.com/jasondorjeshort/broodforge/internal/config.machineLiteralsOff=true",
		"-o", dir+string(os.PathSeparator), "./cmd/broodforge")
	cmd.Dir = filepath.Dir(gomod)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		buildErr = errors.New(err.Error() + ": " + string(out))
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		buildErr = err
		return
	}
	if len(entries) != 1 {
		buildErr = errors.New("expected one file in " + dir)
		return
	}
	bin = filepath.Join(dir, entries[0].Name()) // broodforge, or broodforge.exe on Windows
}

// Result is one run of the binary.
type Result struct {
	Code   int
	Stdout string
	Stderr string
}

// Environ is the process environment without any BROODFORGE_* entry (a test run inside a
// Broodforge session inherits them), plus extra KEY=VALUE entries.
func Environ(extra []string) []string {
	var out []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), "BROODFORGE_") {
			out = append(out, kv)
		}
	}
	return append(out, extra...)
}

// Run runs the binary with args and returns its exit code and output; it fails the test only
// if the binary cannot be started. env adds KEY=VALUE entries; BROODFORGE_* is removed from
// the inherited environment so only the test decides it (Environ). Each run is logged, so
// `go test -v` shows the commands and their exit codes.
func Run(t testing.TB, env []string, args ...string) Result {
	t.Helper()
	cmd := exec.Command(Binary(t), args...)
	cmd.Env = Environ(env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		r.Code = ee.ExitCode()
	default:
		t.Fatalf("run broodforge %q: %v", args, err)
	}
	t.Logf("$ broodforge %s\nexit %d\nstdout: %q\nstderr: %q", strings.Join(args, " "), r.Code, r.Stdout, r.Stderr)
	return r
}
