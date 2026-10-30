// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// Golden sessions (05-debug-interface-discord.md, Golden sessions; P32): a directory holding a
// recorded session's events.jsonl and what the renderer made of them when the golden was
// blessed, outbox.jsonl and one transcript-<session>.md per session in the events. Rendering the
// events again and comparing line by line shows a render or wording regression as a diff.

// GoldenOutbox is a golden's outbox file; a transcript is GoldenTranscriptPrefix<session>.md.
const (
	GoldenOutbox           = "outbox.jsonl"
	GoldenEvents           = "events.jsonl"
	GoldenTranscriptPrefix = "transcript-"
)

// GoldenPost is a Post reduced to the fields a golden compares: no seq (the line's position) and
// no payload (disgo's request body, the library's shape and not the renderer's).
type GoldenPost struct {
	Time      string   `json:"time"`
	Op        Op       `json:"op"`
	Session   string   `json:"session,omitempty"`
	Channel   string   `json:"channel"`
	MessageID string   `json:"messageId"`
	Length    int      `json:"length"`
	First     string   `json:"first,omitempty"`
	Body      string   `json:"body,omitempty"`
	Buttons   []Button `json:"buttons,omitempty"`
	Selects   []Select `json:"selects,omitempty"`
	Pinned    bool     `json:"pinned,omitempty"`
	File      string   `json:"file,omitempty"`
}

// goldenPost reduces p to the compared fields.
func goldenPost(p Post) GoldenPost {
	return GoldenPost{Time: p.Time, Op: p.Op, Session: p.Session, Channel: p.Channel, MessageID: p.MessageID, Length: p.Length,
		First: p.First, Body: p.Body, Buttons: p.Buttons, Selects: p.Selects, Pinned: p.Pinned, File: p.File}
}

// goldenSessionID is what a session id must be to name a transcript file.
var goldenSessionID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// GoldenFiles are a golden's output files by name (outbox.jsonl, transcript-<session>.md), each
// its text with CRLF read as LF.
type GoldenFiles map[string]string

// Names are the files' names in order: outbox.jsonl, then the transcripts by session.
func (g GoldenFiles) Names() []string {
	names := make([]string, 0, len(g))
	for n := range g {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// GoldenDiff is one file that differs: its first differing line (1-based), that line on each
// side ("-" where the side has no such line), and the count of further differing lines.
type GoldenDiff struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Golden string `json:"golden"`
	Now    string `json:"now"`
	More   int    `json:"more"`
}

// RenderGolden replays events on the fake clock and returns the result with the output files a
// golden keeps: the posts reduced to GoldenPost, one per line, and per session in the events its
// transcript over that session's events alone, the names as that session's replay view resolves
// them (the driver's from session-started, anyone else's as a mention), not the registry's.
func RenderGolden(events []ReplayEvent) (ReplayResult, GoldenFiles, error) {
	res := Replay(events, true)
	files := GoldenFiles{}
	var out bytes.Buffer
	for _, p := range res.Posts {
		b, err := encodeGolden(goldenPost(p))
		if err != nil {
			return res, nil, err
		}
		out.Write(b)
		out.WriteByte('\n')
	}
	files[GoldenOutbox] = out.String()

	// The same order Replay renders in: by time, a tie in file order.
	sorted := append([]ReplayEvent(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool { return parseTime(sorted[i].Time).Before(parseTime(sorted[j].Time)) })
	var ids []string
	per := map[string][]session.Event{}
	for _, ev := range sorted {
		id := sessionOr(ev.Session)
		if _, ok := per[id]; !ok {
			ids = append(ids, id)
		}
		per[id] = append(per[id], ev.Event)
	}
	for _, id := range ids {
		if !goldenSessionID.MatchString(id) {
			return res, nil, fmt.Errorf("golden: session id %q is not [A-Za-z0-9_-]+, so it cannot name a transcript file", id)
		}
		n := 0
		v := NewView(id, ViewConfig{NewID: func() string { n++; return "m" + strconv.Itoa(n) }, Name: mention, Offer: func(Post) {}})
		for _, ev := range per[id] {
			v.Handle(ev)
		}
		files[GoldenTranscriptPrefix+id+".md"] = Transcript(id, per[id], v.who)
	}
	return res, files, nil
}

// encodeGolden is json.Marshal without HTML escaping, as the CLI prints a post, so `->` and
// `<@1001>` read as written.
func encodeGolden(v GoldenPost) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// isGoldenOutput is whether name is one of the output files a golden keeps.
func isGoldenOutput(name string) bool {
	return name == GoldenOutbox || (strings.HasPrefix(name, GoldenTranscriptPrefix) && strings.HasSuffix(name, ".md"))
}

// ReadGolden reads dir's output files, CRLF read as LF. A file the golden does not have is
// absent from the result, so it compares as all lines missing.
func ReadGolden(dir string) (GoldenFiles, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := GoldenFiles{}
	for _, e := range entries {
		if !e.Type().IsRegular() || !isGoldenOutput(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = strings.ReplaceAll(string(b), "\r\n", "\n")
	}
	return files, nil
}

// WriteGolden writes files into dir and removes any output file dir holds that files does not
// (a transcript of a session the events no longer hold); events.jsonl is never written.
func WriteGolden(dir string, files GoldenFiles) error {
	// Every name is checked before anything is removed or written: a name that is not one plain
	// file in dir (a separator, `..`, a volume) is refused, so nothing lands outside it.
	for _, name := range files.Names() {
		if filepath.Base(name) != name || !filepath.IsLocal(name) || !isGoldenOutput(name) {
			return fmt.Errorf("golden: %q is not an output file name in %s", name, filepath.ToSlash(dir))
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if _, keep := files[e.Name()]; !keep && e.Type().IsRegular() && isGoldenOutput(e.Name()) {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	for _, name := range files.Names() {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(files[name]), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// CompareGolden compares got (the render now) with want (the blessed files) line by line, by
// position: one GoldenDiff per file that differs, in file-name order. A final newline is not a
// difference; a file one side lacks compares as no lines.
func CompareGolden(got, want GoldenFiles) []GoldenDiff {
	names := map[string]bool{}
	for n := range got {
		names[n] = true
	}
	for n := range want {
		names[n] = true
	}
	all := make([]string, 0, len(names))
	for n := range names {
		all = append(all, n)
	}
	sort.Strings(all)
	var diffs []GoldenDiff
	for _, name := range all {
		gold, now := goldenLines(want[name]), goldenLines(got[name])
		var d *GoldenDiff
		for i := range max(len(gold), len(now)) {
			if i < len(gold) && i < len(now) && gold[i] == now[i] {
				continue
			}
			if d == nil {
				d = &GoldenDiff{File: name, Line: i + 1, Golden: lineAt(gold, i), Now: lineAt(now, i)}
				continue
			}
			d.More++
		}
		if d != nil {
			diffs = append(diffs, *d)
		}
	}
	return diffs
}

// goldenLines splits a file's text into lines, a final newline dropped.
func goldenLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "-"
}
