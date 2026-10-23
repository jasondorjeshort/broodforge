// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
)

// DefaultLimit is how many rows a list prints without --limit or --all (N-10).
const DefaultLimit = 20

// Printer is the one formatter (05-cli-conventions.md, Output). Text mode prints one record
// per line, its columns in a fixed order separated by two spaces, no header; JSON mode prints
// each record as one compact object per line, so a single result is one object and a list is
// JSONL. It remembers the outcome token for --expect: the first column of the first line, as
// text mode renders it, whichever mode is printing.
type Printer struct {
	w     io.Writer
	json  bool
	limit int // rows per list; negative means all

	token    string
	tokenSet bool
}

// NewPrinter returns a printer writing to w. limit 0 means DefaultLimit, negative means all.
func NewPrinter(w io.Writer, jsonMode bool, limit int) *Printer {
	if limit == 0 {
		limit = DefaultLimit
	}
	return &Printer{w: w, json: jsonMode, limit: limit}
}

// Token is the outcome token: the first column of the first line printed, or "" if none was.
func (p *Printer) Token() string { return p.token }

// SetToken makes s the outcome token whatever was printed, for a command whose first line is
// not its outcome (engine run streams events and its outcome is whether a result came).
func (p *Printer) SetToken(s string) { p.token, p.tokenSet = s, true }

// Event prints one streamed event as a record.
func (p *Printer) Event(raw json.RawMessage, cols []string) error { return p.Record(raw, cols) }

// Result prints a final result: an array as a list of records (cut at the limit), null as
// nothing, anything else as one record.
func (p *Printer) Result(raw json.RawMessage, cols []string) error {
	raw = bytes.TrimSpace(raw)
	switch {
	case len(raw) == 0 || string(raw) == "null":
		return nil
	case raw[0] == '[':
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return err
		}
		for i, r := range rows {
			if p.limit >= 0 && i >= p.limit {
				break
			}
			if err := p.Record(r, cols); err != nil {
				return err
			}
		}
		return nil
	}
	return p.Record(raw, cols)
}

// Record prints one record: an object's cols in order (the whole object as compact JSON when
// cols is empty), or any other value as its one column.
func (p *Printer) Record(raw json.RawMessage, cols []string) error {
	fields := textFields(raw, cols)
	if !p.tokenSet {
		p.token, p.tokenSet = fields[0], true
	}
	var line string
	if p.json {
		var b bytes.Buffer
		if err := json.Compact(&b, raw); err != nil {
			return err
		}
		line = b.String()
	} else {
		line = strings.Join(fields, "  ")
	}
	_, err := io.WriteString(p.w, line+"\n")
	return err
}

func textFields(raw json.RawMessage, cols []string) []string {
	var obj map[string]json.RawMessage
	if len(cols) == 0 || json.Unmarshal(raw, &obj) != nil || obj == nil {
		return []string{textValue(raw)}
	}
	out := make([]string, len(cols))
	for i, c := range cols {
		// A column named `<key>:` is labelled: `<key>: <value>`, one column (worktree check's
		// `privacy: <sha>`).
		if key, ok := strings.CutSuffix(c, ":"); ok {
			out[i] = c + " " + textValue(obj[key])
			continue
		}
		out[i] = textValue(obj[c])
	}
	return out
}

var controlEscapes = strings.NewReplacer("\\", `\\`, "\n", `\n`, "\r", `\r`, "\t", `\t`)

// textValue renders one JSON value as a column: a string unquoted with \, newline, CR and tab
// escaped (an empty string as ""), null or a missing key as -, anything else as compact JSON.
func textValue(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "-"
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			if s == "" {
				return `""`
			}
			return controlEscapes.Replace(s)
		}
	}
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return string(raw)
	}
	return b.String()
}

// FormatTime is the one time format of every output: UTC, ISO 8601, to the second.
func FormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }
