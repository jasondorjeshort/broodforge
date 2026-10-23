// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	dg "github.com/disgoorg/disgo/discord"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// TestParseSlashModel: the typed forms of /model, /session model|effort and /session start
// model:<alias>.
func TestParseSlashModel(t *testing.T) {
	for text, want := range map[string]struct {
		kind SlashKind
		args string
	}{
		"/model":                    {SlashModel, ""},
		"/model sonnet":             {SlashModel, "sonnet"},
		"/session model fable":      {SlashSessionModel, "fable"},
		"/session effort high":      {SlashSessionEffort, "high"},
		"/session timeout never":    {SlashSessionTimeout, "never"},
		"/session mode acts-freely": {SlashSessionMode, "acts-freely"},
	} {
		s, err := ParseSlash(text)
		if err != nil || s.Kind != want.kind || strings.Join(s.Args, " ") != want.args {
			t.Errorf("%s: %+v %v", text, s, err)
		}
	}
	s, err := ParseSlash("/session start model:sonnet")
	if err != nil || s.Kind != SlashSessionStart || s.Options["model"] != "sonnet" {
		t.Errorf("start: %+v %v", s, err)
	}
	if _, err := ParseSlash("/session start mode:policy"); err == nil {
		t.Error("/session start took an option it does not have")
	}
}

// TestModelCommandsAndSlashOf: the registered commands carry the choice lists, and the
// interactions read back as the typed forms do.
func TestModelCommandsAndSlashOf(t *testing.T) {
	b, err := json.Marshal(Commands())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"name":"model","description":"This session's model`, `"name":"effort"`, `"value":"fable"`, `"value":"max"`} {
		if !strings.Contains(string(b), s) {
			t.Errorf("commands lack %s: %s", s, b)
		}
	}
	cases := []struct {
		data string
		kind SlashKind
		args string
		opts map[string]string
	}{
		{`{"id":"1","name":"model","type":1}`, SlashModel, "", nil},
		{`{"id":"1","name":"model","type":1,"options":[{"name":"model","type":3,"value":"opus"}]}`, SlashModel, "opus", nil},
		{`{"id":"1","name":"session","type":1,"options":[{"name":"model","type":1,"options":[{"name":"model","type":3,"value":"haiku"}]}]}`,
			SlashSessionModel, "haiku", nil},
		{`{"id":"1","name":"session","type":1,"options":[{"name":"effort","type":1,"options":[{"name":"effort","type":3,"value":"low"}]}]}`,
			SlashSessionEffort, "low", nil},
		{`{"id":"1","name":"session","type":1,"options":[{"name":"timeout","type":1,"options":[{"name":"timeout","type":3,"value":"2h"}]}]}`,
			SlashSessionTimeout, "2h", nil},
		{`{"id":"1","name":"session","type":1,"options":[{"name":"start","type":1,"options":[{"name":"model","type":3,"value":"sonnet"}]}]}`,
			SlashSessionStart, "", map[string]string{"model": "sonnet"}},
	}
	for _, c := range cases {
		i := interaction(t, slashJSON("1", testLobby, c.data)).(dg.ApplicationCommandInteraction)
		sl, _, ok := slashOf(i.SlashCommandInteractionData())
		if !ok || sl.Kind != c.kind || strings.Join(sl.Args, " ") != c.args {
			t.Errorf("%s: %+v %v", c.data, sl, ok)
		}
		for k, v := range c.opts {
			if sl.Options[k] != v {
				t.Errorf("%s: option %s = %q, want %q", c.data, k, sl.Options[k], v)
			}
		}
	}
}

func TestModelTexts(t *testing.T) {
	m := headerState{model: "claude-opus-5-5", why: "this channel's default"}
	if got, want := modelQueryText(m), "Model: Opus (this channel's default). Switch with /model opus, sonnet, haiku or fable."; got != want {
		t.Errorf("query\n got %s\nwant %s", got, want)
	}
	m.effort = "high"
	if got, want := modelLine(m), "Model: Opus (this channel's default). Effort: high."; got != want {
		t.Errorf("line %s", got)
	}
	if got := modelLine(headerState{}); got != "Model: the engine's own." {
		t.Errorf("unset %s", got)
	}
	if got, want := notAModelText("gpt"), "Not a model: gpt. Models: opus, sonnet, haiku, fable."; got != want {
		t.Errorf("not a model %s", got)
	}
	if got, want := notAnEffortText("turbo"), "Not an effort: turbo. Efforts: low, medium, high, max."; got != want {
		t.Errorf("not an effort %s", got)
	}
	if got := notAModelText(strings.Repeat("x", 100)); len([]rune(got)) > 100 {
		t.Errorf("a long value is repeated whole: %s", got)
	}
}

// TestViewModelSwitch: the header's model line from the start, then a switch posts 04's line
// and edits the pinned header in place.
func TestViewModelSwitch(t *testing.T) {
	v, posts := newTestView()
	st := *testStart
	st.Model, st.ChannelModel = "claude-sonnet-5-5", "claude-opus-5-5"
	v.Handle(ev(session.EventSessionStarted, 0, func(e *session.Event) { e.Start = &st }))
	header := (*posts)[1]
	if !strings.Contains(header.Body, "\nModel: Sonnet (chosen at start).\nReach: ") {
		t.Errorf("header %q", header.Body)
	}
	v.Handle(ev(session.EventModeChanged, time.Second, func(e *session.Event) { e.Turn, e.Result = 2, "policy" }))
	v.Handle(ev(session.EventModelChanged, time.Second, func(e *session.Event) {
		e.Turn, e.Result, e.Summary = 2, "claude-fable-5-1", "From the next turn: Fable."
	}))
	v.Handle(ev(session.EventEffortChanged, 2*time.Second, func(e *session.Event) {
		e.Turn, e.Result, e.Summary = 2, "max", "From the next turn: effort max."
	}))
	var lines []string
	var last Post
	for _, p := range (*posts)[2:] {
		if p.Op == OpCreate {
			lines = append(lines, p.Body)
		}
		if p.Op == OpEdit && p.MessageID == header.MessageID {
			last = p
		}
	}
	if len(lines) != 3 || strings.Join(lines[1:], "|") != "From the next turn: Fable.|From the next turn: effort max." {
		t.Errorf("venue lines %q", lines)
	}
	if !strings.Contains(last.Body, "\nMode: acts freely (from turn 2). Credential: ") ||
		!strings.Contains(last.Body, "\nModel: Fable (from turn 2). Effort: max.\nReach: ") || len(last.Buttons) != 1 ||
		last.Buttons[0].CustomID != "bf:s0001:end" {
		t.Errorf("header edit %+v", last)
	}
	if got := headerOf(v.events); got.model != "claude-fable-5-1" || got.effort != "max" || got.why != "from turn 2" {
		t.Errorf("headerOf %+v", got)
	}
}
