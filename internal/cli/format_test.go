// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func rows(n int) json.RawMessage {
	var parts []string
	for i := 1; i <= n; i++ {
		parts = append(parts, fmt.Sprintf(`{"id":"s%04d","n":%d}`, i, i))
	}
	return json.RawMessage("[" + strings.Join(parts, ",") + "]")
}

func TestListLimit(t *testing.T) {
	for _, tc := range []struct {
		limit, want int
	}{{0, DefaultLimit}, {5, 5}, {-1, 25}} {
		var b bytes.Buffer
		p := NewPrinter(&b, false, tc.limit)
		if err := p.Result(rows(25), []string{"id", "n"}); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
		if len(lines) != tc.want || lines[0] != "s0001  1" || p.Token() != "s0001" {
			t.Errorf("limit %d: %d lines, first %q, token %q", tc.limit, len(lines), lines[0], p.Token())
		}
	}
}

func TestJSONModeIsJSONL(t *testing.T) {
	var b bytes.Buffer
	p := NewPrinter(&b, true, 0)
	if err := p.Result(json.RawMessage(`[ {"id": "a"}, {"id": "b"} ]`), []string{"id"}); err != nil {
		t.Fatal(err)
	}
	if b.String() != "{\"id\":\"a\"}\n{\"id\":\"b\"}\n" || p.Token() != "a" {
		t.Errorf("got %q token %q", b.String(), p.Token())
	}
}

func TestTextValues(t *testing.T) {
	var b bytes.Buffer
	p := NewPrinter(&b, false, 0)
	rec := json.RawMessage(`{"s":"a\nb\\c","e":"","z":null,"o":{"k": [1, 2]},"t":true,"f":1.5}`)
	if err := p.Record(rec, []string{"s", "e", "z", "missing", "o", "t", "f"}); err != nil {
		t.Fatal(err)
	}
	want := `a\nb\\c  ""  -  -  {"k":[1,2]}  true  1.5` + "\n"
	if b.String() != want {
		t.Errorf("got %q, want %q", b.String(), want)
	}
	b.Reset()
	if err := p.Result(json.RawMessage(`"scalar"`), []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if err := p.Result(json.RawMessage(`null`), []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if err := p.Record(json.RawMessage(`{"a":1}`), nil); err != nil {
		t.Fatal(err)
	}
	if b.String() != "scalar\n{\"a\":1}\n" {
		t.Errorf("got %q", b.String())
	}
}

func TestFormatTime(t *testing.T) {
	at := time.Date(2026, 10, 16, 12, 30, 5, 999, time.FixedZone("x", 2*3600))
	if got := FormatTime(at); got != "2026-10-16T10:30:05Z" {
		t.Errorf("FormatTime = %s", got)
	}
}
