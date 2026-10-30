// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// fixturesRoot is the repository's fixtures/engine/, from this package's directory.
const fixturesRoot = "../../fixtures/engine"

// fixtureDirs is every directory under fixtures/engine/: a new fixture replays without a test
// edit.
func fixtureDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(fixturesRoot)
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(fixturesRoot, e.Name()))
		}
	}
	if len(dirs) == 0 {
		t.Fatal("no fixtures under " + fixturesRoot)
	}
	return dirs
}

// turnLines is a recorded turn's non-blank lines.
func turnLines(t *testing.T, dir string, n int) [][]byte {
	t.Helper()
	data, err := os.ReadFile(TurnFile(dir, n))
	if err != nil {
		t.Fatal(err)
	}
	var lines [][]byte
	for _, l := range bytes.Split(data, []byte("\n")) {
		if l = bytes.TrimSpace(l); len(l) > 0 {
			lines = append(lines, l)
		}
	}
	return lines
}

// parseTurn is every event ParseLine yields for a recorded turn, in order.
func parseTurn(t *testing.T, dir string, n int) []EngineEvent {
	t.Helper()
	var evs []EngineEvent
	for _, l := range turnLines(t, dir, n) {
		evs = append(evs, ParseLine(l)...)
	}
	return evs
}

// oracle is what a line should yield, read independently of the parser with map[string]any:
// the kinds in order, and for tool results whether is_error is true.
func oracle(t *testing.T, line []byte) (kinds []EventKind, toolErrors []bool) {
	t.Helper()
	var o map[string]any
	if err := json.Unmarshal(line, &o); err != nil {
		return []EventKind{KindUnknown}, nil
	}
	typ, _ := o["type"].(string)
	sub, _ := o["subtype"].(string)
	switch {
	case typ == "system" && sub == "init":
		return []EventKind{KindInit}, nil
	case typ == "assistant":
		return []EventKind{KindAssistant}, nil
	case typ == "result":
		return []EventKind{KindResult}, nil
	case typ == "rate_limit_event":
		return []EventKind{KindRateLimit}, nil
	case typ == "user":
		msg, _ := o["message"].(map[string]any)
		content, _ := msg["content"].([]any)
		for _, c := range content {
			b, _ := c.(map[string]any)
			if b["type"] == "tool_result" {
				kinds = append(kinds, KindToolResult)
				toolErrors = append(toolErrors, b["is_error"] == true)
			}
		}
		if len(kinds) > 0 {
			return kinds, toolErrors
		}
	}
	return []EventKind{KindUnknown}, nil
}

// TestEveryFixtureReplays parses every recorded turn and compares the parser with the oracle,
// line by line, so a toolchain or parser change that alters what is read fails here first
// (ADR 0023, Types and JSON).
func TestEveryFixtureReplays(t *testing.T) {
	for _, dir := range fixtureDirs(t) {
		fx, err := ReadFixture(dir)
		if err != nil {
			t.Fatal(err)
		}
		files, _ := filepath.Glob(filepath.Join(dir, "turn-*.jsonl"))
		if len(files) != len(fx.Turns) || len(fx.Turns) == 0 {
			t.Errorf("%s: %d turn files, %d turns in fixture.json", dir, len(files), len(fx.Turns))
		}
		for _, turn := range fx.Turns {
			inits := 0
			for i, line := range turnLines(t, dir, turn.N) {
				wantKinds, wantErrs := oracle(t, line)
				evs := ParseLine(line)
				if len(evs) != len(wantKinds) {
					t.Fatalf("%s turn %d line %d: %d events, want %d", dir, turn.N, i+1, len(evs), len(wantKinds))
				}
				for j, ev := range evs {
					if ev.Kind != wantKinds[j] {
						t.Errorf("%s turn %d line %d: kind %s, want %s", dir, turn.N, i+1, ev.Kind, wantKinds[j])
					}
					if _, err := ev.MarshalJSON(); err != nil {
						t.Errorf("%s turn %d line %d: %v", dir, turn.N, i+1, err)
					}
					switch ev.Kind {
					case KindInit:
						inits++
						if ev.Init.EngineSessionID != turn.EngineSessionID {
							t.Errorf("%s turn %d: init session %q, fixture.json says %q", dir, turn.N, ev.Init.EngineSessionID, turn.EngineSessionID)
						}
						if len(ev.Init.Capabilities) == 0 || len(ev.Init.Tools) == 0 || ev.Init.Model == "" {
							t.Errorf("%s turn %d: init without tools, capabilities or model", dir, turn.N)
						}
					case KindAssistant:
						if ev.Assistant.MessageID == "" || ev.Assistant.Model == "" {
							t.Errorf("%s turn %d line %d: assistant without message id or model", dir, turn.N, i+1)
						}
					case KindToolResult:
						if ev.ToolResult.IsError != wantErrs[j] {
							t.Errorf("%s turn %d line %d: isError %v, want %v", dir, turn.N, i+1, ev.ToolResult.IsError, wantErrs[j])
						}
						if ev.ToolResult.ToolUseID == "" {
							t.Errorf("%s turn %d line %d: tool result without an id", dir, turn.N, i+1)
						}
					case KindResult:
						if len(ev.Result.ModelUsage) == 0 || ev.Result.StopReason == "" {
							t.Errorf("%s turn %d line %d: result without modelUsage or stop reason", dir, turn.N, i+1)
						}
						if ev.Result.StopReason == "tool_deferred" && (ev.Result.Deferred == nil || ev.Result.Deferred.ToolUseID == "") {
							t.Errorf("%s turn %d: tool_deferred result names no call", dir, turn.N)
						}
					case KindRateLimit:
						if ev.RateLimit.Status == "" || ev.RateLimit.SevenDay.ResetsAt == 0 || ev.RateLimit.FiveHour.ResetsAt == 0 {
							t.Errorf("%s turn %d line %d: rate-limit without status or windows: %+v", dir, turn.N, i+1, ev.RateLimit)
						}
					case KindExit, KindUnknown:
					}
				}
			}
			if inits != 1 {
				t.Errorf("%s turn %d: %d init events, want 1", dir, turn.N, inits)
			}
		}
	}
}

// TestToolResultMeta: every recorded tool result carries tool_result_meta with a source, so
// decidedBy is set, except a <tool_use_error> the engine raised before any hook ran
// (interrupted turn 1, S2).
func TestToolResultMeta(t *testing.T) {
	var exceptions []string
	for _, dir := range fixtureDirs(t) {
		fx, err := ReadFixture(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, turn := range fx.Turns {
			for _, ev := range parseTurn(t, dir, turn.N) {
				if ev.Kind != KindToolResult {
					continue
				}
				tr := ev.ToolResult
				if strings.HasPrefix(tr.Text, "<tool_use_error>") {
					if tr.DecidedBy != "" {
						t.Errorf("%s turn %d: %s: a tool_use_error with decidedBy %q", dir, turn.N, tr.ToolUseID, tr.DecidedBy)
					}
					exceptions = append(exceptions, filepath.Base(dir)+"/"+TurnFile("", turn.N))
					continue
				}
				if tr.DecidedBy != "hook" && tr.DecidedBy != "config" {
					t.Errorf("%s turn %d: %s: decidedBy %q, want hook or config", dir, turn.N, tr.ToolUseID, tr.DecidedBy)
				}
			}
		}
	}
	if len(exceptions) != 1 || exceptions[0] != "interrupted/turn-1.jsonl" {
		t.Errorf("tool_use_error results in %q, want only interrupted/turn-1.jsonl", exceptions)
	}
}

// TestProofEvents is the P3 Proof on the parser: basic turn 1 has two unknown lines (three
// before P28 read its rate_limit_event), and
// parallel turn 1 defers the second of two tool-use blocks that share one message id (C28).
func TestProofEvents(t *testing.T) {
	unknown := 0
	for _, ev := range parseTurn(t, fixturesRoot+"/basic", 1) {
		if ev.Kind == KindUnknown {
			unknown++
		}
	}
	if unknown != 2 {
		t.Errorf("basic turn 1: %d unknown events, want 2", unknown)
	}

	var uses []Block
	var ids []string
	var deferred *DeferredTool
	for _, ev := range parseTurn(t, fixturesRoot+"/parallel", 1) {
		if ev.Kind == KindAssistant {
			for _, b := range ev.Assistant.Blocks {
				if b.Kind == BlockToolUse {
					uses = append(uses, b)
					ids = append(ids, ev.Assistant.MessageID)
				}
			}
		}
		if ev.Kind == KindResult {
			deferred = ev.Result.Deferred
		}
	}
	if len(uses) != 2 || ids[0] != ids[1] {
		t.Fatalf("parallel turn 1: tool uses %v under message ids %v, want two under one", uses, ids)
	}
	if deferred == nil || deferred.ToolUseID != uses[1].ID {
		t.Errorf("parallel turn 1: deferred %+v, want the second id %s", deferred, uses[1].ID)
	}
}

// commandsChanged is the system/commands_changed lines of basic turn 1, two real ones recorded
// on engine 2.1.296 (25626 bytes each).
func commandsChanged(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(TurnFile(fixturesRoot+"/basic", 1))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, `{"type":"system","subtype":"commands_changed"`) {
			out = append(out, strings.TrimSpace(line))
		}
	}
	if len(out) != 2 {
		t.Fatalf("basic turn 1 has %d commands_changed lines, want 2", len(out))
	}
	return out
}

// headOf is the oracle for an unknown event's head: whole runes of the line while they fit
// in UnknownHeadBytes.
func headOf(line string) string {
	n := 0
	for _, r := range line {
		if n+utf8.RuneLen(r) > UnknownHeadBytes {
			break
		}
		n += utf8.RuneLen(r)
	}
	return line[:n]
}

func TestParseLineNeverRefuses(t *testing.T) {
	cc := commandsChanged(t)[0]
	if len(cc) != 25626 {
		t.Fatalf("commands_changed line is %d bytes, want 25626", len(cc))
	}
	// A line whose 200th byte is inside a rune: the head stops before the rune.
	cut := strings.Repeat("a", UnknownHeadBytes-1) + "—" + "tail"
	for _, tc := range []struct {
		line, typ, subtype, head string
	}{
		{cc, "system", "commands_changed", headOf(cc)},
		{"not json", "", "", "not json"},
		{"[1,2]", "", "", "[1,2]"},
		{`"s"`, "", "", `"s"`},
		{`{"type":"system","subtype":"thinking_tokens"}`, "system", "thinking_tokens", `{"type":"system","subtype":"thinking_tokens"}`},
		{`{"type":"user","message":{"content":"a prompt"}}`, "user", "", `{"type":"user","message":{"content":"a prompt"}}`},
		{cut, "", "", strings.Repeat("a", UnknownHeadBytes-1)},
	} {
		evs := ParseLine([]byte(tc.line))
		if len(evs) != 1 || evs[0].Kind != KindUnknown {
			t.Errorf("%.40s: %+v, want one unknown", tc.line, evs)
			continue
		}
		want := UnknownEvent{Type: tc.typ, Subtype: tc.subtype, Bytes: len(tc.line), Head: tc.head}
		if got := *evs[0].Unknown; got != want {
			t.Errorf("%.40s: %+v, want %+v", tc.line, got, want)
		}
		if tc.line == cc && len(evs[0].Unknown.Head) != UnknownHeadBytes {
			t.Errorf("commands_changed head is %d bytes, want %d", len(evs[0].Unknown.Head), UnknownHeadBytes)
		}
		if len(tc.head) > UnknownHeadBytes || !strings.HasPrefix(tc.line, tc.head) {
			t.Errorf("%.40s: head %q is not a prefix of at most %d bytes", tc.line, tc.head, UnknownHeadBytes)
		}
	}
	if h := headOf(cc); len(h) < UnknownHeadBytes-utf8.UTFMax+1 || !utf8.ValidString(h) {
		t.Errorf("commands_changed head is %d bytes or not whole runes: %q", len(h), h)
	}
	if evs := ParseLine([]byte("  \r")); len(evs) != 0 {
		t.Errorf("blank line: %+v, want none", evs)
	}
	// A field of the wrong type is left empty, the rest is read; unknown fields are ignored.
	evs := ParseLine([]byte(`{"type":"system","subtype":"init","session_id":"s1","model":5,"tools":["Read"],"new_field":{"x":1}}`))
	if len(evs) != 1 || evs[0].Kind != KindInit || evs[0].Init.EngineSessionID != "s1" || evs[0].Init.Model != "" || evs[0].Init.Tools[0] != "Read" {
		t.Errorf("init with a mistyped field: %+v", evs)
	}
	// is_error absent is success; a source-less meta entry gives no decidedBy.
	evs = ParseLine([]byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"a","content":[{"type":"text","text":"x"},{"type":"text","text":"y"}]}]},"tool_result_meta":[{"id":"a","permission_decision":{"decision":"accept"}}]}`))
	if len(evs) != 1 || evs[0].ToolResult.IsError || evs[0].ToolResult.Text != "x\ny" || evs[0].ToolResult.DecidedBy != "" {
		t.Errorf("tool result: %+v", evs[0].ToolResult)
	}
}

// TestUnknownHeadCutsAtRuneBoundary (P29): a 2-, 3- or 4-byte rune across byte 200 is left out whole; 199-, 200- and 201-byte lines.
func TestUnknownHeadCutsAtRuneBoundary(t *testing.T) {
	for _, r := range []string{"é", "—", "😀"} {
		for pre := UnknownHeadBytes - len(r) - 1; pre <= UnknownHeadBytes+1; pre++ {
			line := strings.Repeat("a", pre) + r + "tail"
			u := ParseLine([]byte(line))[0].Unknown
			if h := u.Head; len(h) == 0 || len(h) > UnknownHeadBytes || !utf8.ValidString(h) || h != headOf(line) || u.Bytes != len(line) {
				t.Errorf("%d-byte rune after %d bytes: head %d bytes, valid %v, want %d", len(r), pre, len(h), utf8.ValidString(h), len(headOf(line)))
			}
		}
	}
	for _, n := range []int{199, 200, 201} {
		if h := ParseLine([]byte(strings.Repeat("b", n)))[0].Unknown.Head; len(h) != min(n, UnknownHeadBytes) {
			t.Errorf("%d-byte line: head %d bytes", n, len(h))
		}
	}
}

func TestMarshalFlatAndUnescaped(t *testing.T) {
	ev := EngineEvent{Kind: KindToolResult, ToolResult: &ToolResultEvent{ToolUseID: "a", IsError: true, Text: "<tool_use_error>&</tool_use_error>"}}
	b, err := ev.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"type":"tool-result","toolUseId":"a","isError":true,"text":"<tool_use_error>&</tool_use_error>"}`; string(b) != want {
		t.Errorf("got %s\nwant %s", b, want)
	}
	b, _ = EngineEvent{Kind: KindUnknown, Unknown: &UnknownEvent{Type: "x", Subtype: "y", Bytes: 25, Head: `{"type":"x","subtype":"y"`}}.MarshalJSON()
	if want := `{"type":"unknown","eventType":"x","subtype":"y","bytes":25,"head":"{\"type\":\"x\",\"subtype\":\"y\""}`; string(b) != want {
		t.Errorf("got %s\nwant %s", b, want)
	}
	if _, err := (EngineEvent{Kind: "bogus"}).MarshalJSON(); err == nil {
		t.Error("an event of no kind marshalled")
	}
}

// resultOf is the last result of a recorded turn.
func resultOf(t *testing.T, dir string, n int) *ResultEvent {
	t.Helper()
	var res *ResultEvent
	for _, ev := range parseTurn(t, dir, n) {
		if ev.Kind == KindResult {
			res = ev.Result
		}
	}
	if res == nil {
		t.Fatalf("%s turn %d: no result", dir, n)
	}
	return res
}

// TestUsageDeltaResumed: basic is one engine session over three turns; each turn's record is
// its cumulative minus the previous result's, limits left out.
func TestUsageDeltaResumed(t *testing.T) {
	dir := fixturesRoot + "/basic"
	var baseline map[string]ModelUsage
	for n := 1; n <= 3; n++ {
		cur := resultOf(t, dir, n).ModelUsage
		u := UsageOf(parseTurn(t, dir, n), baseline)
		if u.Interrupted || u.Cumulative == nil {
			t.Fatalf("turn %d: interrupted", n)
		}
		for model, fields := range cur {
			if _, ok := u.Delta[model]["contextWindow"]; ok {
				t.Errorf("turn %d: contextWindow in the delta", n)
			}
			for f, v := range fields {
				if notSummed[f] {
					continue
				}
				want := max(0, v-baseline[model][f])
				if got := u.Delta[model][f]; got != want {
					t.Errorf("turn %d %s %s: %v, want %v", n, model, f, got, want)
				}
			}
		}
		if n > 1 && u.Delta[firstModel(cur)]["inputTokens"] >= cur[firstModel(cur)]["inputTokens"] {
			t.Errorf("turn %d: the delta is not below the cumulative; modelUsage stopped being cumulative?", n)
		}
		baseline = u.Cumulative
	}
}

func firstModel(m map[string]ModelUsage) string {
	for k := range m {
		return k
	}
	return ""
}

// TestUsageInterrupted (S2): interrupted turn 2 has no result, so its record is the assistant
// counts deduplicated by message id and it is not a baseline; turn 3, which resumes it, is
// measured against the session's last result, of which there is none, so its delta is its
// whole cumulative.
func TestUsageInterrupted(t *testing.T) {
	dir := fixturesRoot + "/interrupted"
	fx, err := ReadFixture(dir)
	if err != nil {
		t.Fatal(err)
	}
	baselines := map[string]map[string]ModelUsage{} // engine session id: last result's cumulative
	var records []TurnUsage
	for _, turn := range fx.Turns {
		u := UsageOf(parseTurn(t, dir, turn.N), baselines[turn.EngineSessionID])
		if u.Cumulative != nil {
			baselines[turn.EngineSessionID] = u.Cumulative
		}
		records = append(records, u)
	}
	if records[0].Interrupted || !records[1].Interrupted || records[2].Interrupted {
		t.Fatalf("interrupted flags %v %v %v, want false true false", records[0].Interrupted, records[1].Interrupted, records[2].Interrupted)
	}
	row := records[1].Delta
	if len(row) != 1 || row["claude-haiku-5-5"]["inputTokens"] <= 0 {
		t.Errorf("turn 2 row %v, want input counts for the one model", row)
	}
	if _, ok := row["claude-haiku-5-5"]["outputTokens"]; ok {
		t.Error("turn 2 row has output tokens, which a turn without a result cannot know")
	}
	cur := resultOf(t, dir, 3).ModelUsage
	for model, fields := range cur {
		for f, v := range fields {
			if !notSummed[f] && records[2].Delta[model][f] != v {
				t.Errorf("turn 3 %s %s: %v, want the whole cumulative %v", model, f, records[2].Delta[model][f], v)
			}
		}
	}
}

// TestInterruptedRowDeduplicates (S1): parallel turn 1 writes its one message as two lines
// sharing one usage; the row counts it once.
func TestInterruptedRowDeduplicates(t *testing.T) {
	evs := parseTurn(t, fixturesRoot+"/parallel", 1)
	var once Usage
	lines := 0
	for _, ev := range evs {
		if ev.Kind == KindAssistant {
			once = ev.Assistant.Usage
			lines++
		}
	}
	row := InterruptedRow(evs)["claude-haiku-5-5"]
	if lines != 2 || row["inputTokens"] != float64(once.Input) || row["cacheReadInputTokens"] != float64(once.CacheRead) || row["cacheCreationInputTokens"] != float64(once.CacheWrite) {
		t.Errorf("%d assistant lines, row %v, want one message's usage %+v", lines, row, once)
	}
}

// TestRateLimit: the recorded line of recorded-read turn 1 (allowed_warning, seven-day 0.68)
// is one rate-limit event; the top-level utilization and resetsAt and seven_day_overage_included
// are not read; a line with no unifiedWindows, or no rate_limit_info, is one event of zeros (P28).
func TestRateLimit(t *testing.T) {
	var got []RateLimitEvent
	for _, ev := range parseTurn(t, fixturesRoot+"/recorded-read", 1) {
		if ev.Kind == KindRateLimit {
			got = append(got, *ev.RateLimit)
		}
	}
	want := RateLimitEvent{Status: "allowed_warning", RateLimitType: "seven_day",
		FiveHour: Window{Utilization: 0.09, ResetsAt: 1792248000}, SevenDay: Window{Utilization: 0.68, ResetsAt: 1792728000},
		OverageStatus: "rejected"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("recorded-read turn 1: rate-limit events %+v, want one %+v", got, want)
	}
	b, err := EngineEvent{Kind: KindRateLimit, RateLimit: &got[0]}.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if w := `{"type":"rate-limit","status":"allowed_warning","rateLimitType":"seven_day","fiveHour":{"utilization":0.09,"resetsAt":1792248000},"sevenDay":{"utilization":0.68,"resetsAt":1792728000},"overageStatus":"rejected","isUsingOverage":false}`; string(b) != w {
		t.Errorf("got %s\nwant %s", b, w)
	}
	for _, line := range []string{
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour","isUsingOverage":true,"utilization":0.5,"resetsAt":7}}`,
		`{"type":"rate_limit_event"}`,
		`{"type":"rate_limit_event","rate_limit_info":{"unifiedWindows":{"five_hour":"x"}}}`,
	} {
		evs := ParseLine([]byte(line))
		if len(evs) != 1 || evs[0].Kind != KindRateLimit || evs[0].RateLimit == nil {
			t.Fatalf("%s: %+v, want one rate-limit event", line, evs)
		}
		r := *evs[0].RateLimit
		if r.FiveHour != (Window{}) || r.SevenDay != (Window{}) {
			t.Errorf("%s: windows %+v %+v, want zeros", line, r.FiveHour, r.SevenDay)
		}
	}
	if r := ParseLine([]byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour","isUsingOverage":true}}`))[0].RateLimit; r.Status != "allowed" || r.RateLimitType != "five_hour" || !r.IsUsingOverage || r.OverageStatus != "" {
		t.Errorf("no unifiedWindows: %+v", r)
	}
}

// TestRateLimitMalformedNeverRefuses: a rate_limit_event whose info or fields have the wrong
// type is still one rate-limit event, the mistyped fields at their zero value, and a bad field
// beside good ones leaves the good ones read (ADR 0023: the stream is never refused).
func TestRateLimitMalformedNeverRefuses(t *testing.T) {
	for _, info := range []string{
		`null`, `"x"`, `[1,2]`,
		`{"unifiedWindows":null}`,
		`{"unifiedWindows":{"five_hour":null,"seven_day":null}}`,
		`{"unifiedWindows":{"five_hour":{"utilization":null}}}`,
		`{"unifiedWindows":{"seven_day":{"resetsAt":"soon"}}}`,
		`{"status":5}`,
		`{"isUsingOverage":"yes"}`,
		`{"unifiedWindows":{"five_hour":{"resetsAt":1.5}}}`,
	} {
		line := `{"type":"rate_limit_event","rate_limit_info":` + info + `}`
		evs := ParseLine([]byte(line))
		if len(evs) != 1 || evs[0].Kind != KindRateLimit || evs[0].RateLimit == nil || *evs[0].RateLimit != (RateLimitEvent{}) {
			t.Errorf("%s: %+v, want one zero rate-limit event", line, evs)
		}
	}
	line := `{"type":"rate_limit_event","rate_limit_info":{"status":5,"rateLimitType":"five_hour","overageStatus":"rejected","isUsingOverage":"yes",` +
		`"unifiedWindows":{"five_hour":{"utilization":0.3,"resetsAt":"soon"},"seven_day":{"utilization":0.5,"resetsAt":1792728000}}}}`
	evs := ParseLine([]byte(line))
	want := RateLimitEvent{RateLimitType: "five_hour", OverageStatus: "rejected",
		FiveHour: Window{Utilization: 0.3}, SevenDay: Window{Utilization: 0.5, ResetsAt: 1792728000}}
	if len(evs) != 1 || evs[0].Kind != KindRateLimit || evs[0].RateLimit == nil || *evs[0].RateLimit != want {
		t.Errorf("bad fields beside good: %+v, want %+v", evs, want)
	}
}
