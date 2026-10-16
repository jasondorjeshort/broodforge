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
	cmd := exec.Command("go", "build", "-o", dir+string(os.PathSeparator), "./cmd/broodforge")
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

// Run runs the binary with args and returns its exit code and output; it fails the test only
// if the binary cannot be started. env adds KEY=VALUE entries; BROODFORGE_ROOT is removed from
// the inherited environment so only the test decides the root. Each run is logged, so
// `go test -v` shows the commands and their exit codes.
func Run(t testing.TB, env []string, args ...string) Result {
	t.Helper()
	cmd := exec.Command(Binary(t), args...)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), "BROODFORGE_ROOT=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, env...)
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
