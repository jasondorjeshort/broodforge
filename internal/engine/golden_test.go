// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// The parser's goldens: fixtures/engine/<name>/events-<n>.jsonl holds every event ParseLine
// yields for turn-<n>.jsonl, one JSON line each, as MarshalJSON writes it. A toolchain move or
// a parser change that alters what is read fails here field by field (P3; ADR 0023, Types and
// JSON). After a deliberate change, regenerate with
//
//	go test ./internal/engine/ -run TestGolden -update
//
// and read the diff before committing it.

var update = flag.Bool("update", false, "rewrite the parser goldens under fixtures/engine/")

// GoldenFile is the path of turn n's golden events in dir.
func goldenFile(dir string, n int) string {
	return filepath.Join(dir, "events-"+strconv.Itoa(n)+".jsonl")
}

func TestGolden(t *testing.T) {
	for _, dir := range fixtureDirs(t) {
		fx, err := ReadFixture(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, turn := range fx.Turns {
			var got bytes.Buffer
			for _, ev := range parseTurn(t, dir, turn.N) {
				line, err := ev.MarshalJSON()
				if err != nil {
					t.Fatal(err)
				}
				got.Write(line)
				got.WriteByte('\n')
			}
			path := goldenFile(dir, turn.N)
			if *update {
				if err := os.WriteFile(path, got.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (generate with -update)", err)
			}
			compareGolden(t, path, got.Bytes(), want)
		}
	}
}

// compareGolden reports, per event, which top-level fields differ.
func compareGolden(t *testing.T, path string, got, want []byte) {
	t.Helper()
	gotLines := bytes.Split(bytes.TrimRight(got, "\n"), []byte("\n"))
	wantLines := bytes.Split(bytes.TrimRight(bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")), "\n"), []byte("\n"))
	if len(gotLines) != len(wantLines) {
		t.Errorf("%s: %d events, golden has %d", path, len(gotLines), len(wantLines))
	}
	for i := range min(len(gotLines), len(wantLines)) {
		if bytes.Equal(gotLines[i], wantLines[i]) {
			continue
		}
		var g, w map[string]json.RawMessage
		if json.Unmarshal(gotLines[i], &g) != nil || json.Unmarshal(wantLines[i], &w) != nil {
			t.Errorf("%s event %d: got %s\nwant %s", path, i+1, gotLines[i], wantLines[i])
			continue
		}
		keys := map[string]bool{}
		for k := range g {
			keys[k] = true
		}
		for k := range w {
			keys[k] = true
		}
		var names []string
		for k := range keys {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			if !bytes.Equal(g[k], w[k]) {
				t.Errorf("%s event %d field %q: got %s, want %s", path, i+1, k, g[k], w[k])
			}
		}
	}
}
