// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// The golden sessions (05-debug-interface-discord.md, Golden sessions; P32): every directory
// under tests/discord/golden/ holds events.jsonl and the output blessed from it. After a
// deliberate render or wording change, re-bless with
//
//	go test ./internal/discord/ -run TestGolden -update
//
// (or `broodforge discord render --golden <dir> --update`) and read `git diff tests/discord/golden/`
// before committing it.

var update = flag.Bool("update", false, "rewrite the golden sessions' output files under tests/discord/golden/")

const goldenRoot = "../../tests/discord/golden"

func goldenDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(goldenRoot)
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(goldenRoot, e.Name()))
		}
	}
	if len(dirs) == 0 {
		t.Fatalf("no golden under %s", goldenRoot)
	}
	return dirs
}

func TestGolden(t *testing.T) {
	for _, dir := range goldenDirs(t) {
		_, got, err := RenderGolden(readReplay(t, filepath.Join(dir, GoldenEvents)))
		if err != nil {
			t.Fatal(err)
		}
		if *update {
			if err := WriteGolden(dir, got); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := ReadGolden(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range CompareGolden(got, want) {
			t.Errorf("%s: line %d differs (%d more after it)\ngolden %s\nnow    %s\n(re-bless with -update and read git diff tests/discord/golden/)",
				filepath.ToSlash(filepath.Join(dir, d.File)), d.Line, d.More, d.Golden, d.Now)
		}
	}
}

// TestGoldenTranscriptOneSession: each blessed transcript is one session's, over its events
// alone, so it holds exactly one `# Session ` heading (a file of several sessions gives one
// transcript each).
func TestGoldenTranscriptOneSession(t *testing.T) {
	seen := 0
	for _, dir := range goldenDirs(t) {
		files, err := ReadGolden(dir)
		if err != nil {
			t.Fatal(err)
		}
		for name, text := range files {
			if !strings.HasPrefix(name, GoldenTranscriptPrefix) {
				continue
			}
			seen++
			n := 0
			for _, l := range strings.Split(text, "\n") {
				if strings.HasPrefix(l, "# Session ") {
					n++
				}
			}
			if n != 1 {
				t.Errorf("%s/%s: %d `# Session ` headings, want 1", filepath.ToSlash(dir), name, n)
			}
		}
	}
	if seen == 0 {
		t.Error("no blessed transcript under " + goldenRoot)
	}
}

func TestCompareGolden(t *testing.T) {
	want := GoldenFiles{GoldenOutbox: "a\nb\nc\n", "transcript-s0001.md": "x\n"}
	got := GoldenFiles{GoldenOutbox: "a\nB\nc\nd", "transcript-s0001.md": "x", "transcript-s0002.md": "y\n"}
	diffs := CompareGolden(got, want)
	if len(diffs) != 2 {
		t.Fatalf("diffs %+v, want two", diffs)
	}
	if d := diffs[0]; d != (GoldenDiff{File: GoldenOutbox, Line: 2, Golden: "b", Now: "B", More: 1}) {
		t.Errorf("outbox: %+v", d)
	}
	if d := diffs[1]; d != (GoldenDiff{File: "transcript-s0002.md", Line: 1, Golden: "-", Now: "y"}) {
		t.Errorf("new transcript: %+v", d)
	}
}

func TestWriteGoldenRemovesStaleTranscript(t *testing.T) {
	dir := t.TempDir()
	for name, text := range map[string]string{GoldenEvents: "{}\n", "transcript-s0002.md": "old\n", "notes.txt": "kept\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := WriteGolden(dir, GoldenFiles{GoldenOutbox: "o\n", "transcript-s0001.md": "t\r\n"}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadGolden(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[GoldenOutbox] != "o\n" || got["transcript-s0001.md"] != "t\n" {
		t.Errorf("read back %q", got)
	}
	for _, name := range []string{GoldenEvents, "notes.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestWriteGoldenStaysInDir: a session id from the events never names a file outside the golden's
// directory: RenderGolden refuses an id that is not [A-Za-z0-9_-]+, and WriteGolden a name that is
// not one plain file in the directory, before it removes anything.
func TestWriteGoldenStaysInDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "a", "gold")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../../../escaped", "sub/x", "C:escaped"} {
		_, files, err := RenderGolden([]ReplayEvent{{Session: id, Event: session.Event{Time: "2026-10-16T10:00:00Z",
			Type: session.EventSessionStarted, Summary: "Session x on X."}}})
		if err == nil {
			err = WriteGolden(dir, files)
		}
		if err == nil {
			t.Errorf("session %q: written, want a refusal", id)
		}
	}
	for _, p := range []string{filepath.Join(root, "escaped.md"), filepath.Join(root, "a", "escaped.md")} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("WriteGolden wrote outside its directory: %s", p)
		}
	}

	// WriteGolden alone refuses such a name before removing a stale transcript.
	if err := os.WriteFile(filepath.Join(dir, "transcript-s0009.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"transcript-../../escaped.md", `transcript-..\escaped.md`, "C:transcript-x.md"} {
		if err := WriteGolden(dir, GoldenFiles{GoldenOutbox: "o\n", name: "x\n"}); err == nil {
			t.Errorf("WriteGolden(%q): nil, want a refusal", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "transcript-s0009.md")); err != nil {
		t.Errorf("a refused write removed a file first: %v", err)
	}
}
