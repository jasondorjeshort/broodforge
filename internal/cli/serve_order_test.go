// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// TestOpenDiscordOrder: startBot's steps 9 and 10 (ADR 0026): live, the coalescer starts only
// after the gateway has opened (and swept), and not at all when the open fails; a dry run starts
// it at once.
func TestOpenDiscordOrder(t *testing.T) {
	var steps []string
	open := func(context.Context) error { steps = append(steps, "open"); return nil }
	run := func() { steps = append(steps, "run") }
	if err := openDiscord(open, run); err != nil || strings.Join(steps, ",") != "open,run" {
		t.Errorf("live: %v %v", steps, err)
	}
	steps = nil
	failed := func(context.Context) error { steps = append(steps, "open"); return errors.New("no gateway") }
	if err := openDiscord(failed, run); err == nil || strings.Join(steps, ",") != "open" {
		t.Errorf("a failed open: %v %v", steps, err)
	}
	steps = nil
	if err := openDiscord(nil, run); err != nil || strings.Join(steps, ",") != "run" {
		t.Errorf("dry run: %v %v", steps, err)
	}
}

// TestEndLine: session end's line keeps a handoff's count and prints an escape's unknown count
// as null, `-` on the text line.
func TestEndLine(t *testing.T) {
	for _, c := range []struct {
		out  git.Outcome
		want string
	}{
		{git.Outcome{Outcome: git.OutcomeNone, Mode: git.ModePush, Remote: "origin", Commits: 0, Removed: true},
			`{"outcome":"none","mode":"push","remote":"origin","commits":0,"url":null,"request":null,"removed":true}`},
		{git.Outcome{Outcome: session.OutcomeTreeMissing, Mode: git.ModePush, Remote: "origin", Commits: session.CommitsUnknown},
			`{"outcome":"tree-missing","mode":"push","remote":"origin","commits":null,"url":null,"request":null,"removed":false}`},
	} {
		raw, err := encodeWorktree(endLineOf(c.out))
		if err != nil || string(raw) != c.want {
			t.Errorf("end line\n got %s %v\nwant %s", raw, err, c.want)
		}
		plain, _ := encodeWorktree(c.out)
		var a, b map[string]json.RawMessage
		_ = json.Unmarshal(raw, &a)
		_ = json.Unmarshal(plain, &b)
		if len(a) != len(b) {
			t.Errorf("the end line's keys %v differ from git.Outcome's %v", a, b)
		}
	}
	if got := strings.Join(textFields(mustEnd(t, session.OutcomeTreeMissing), sessionEnd.Columns), "  "); got != "tree-missing  origin  -  -  -  -" {
		t.Errorf("text line: %s", got)
	}
}

func mustEnd(t *testing.T, outcome string) json.RawMessage {
	t.Helper()
	raw, err := encodeWorktree(endLineOf(git.Outcome{Outcome: outcome, Mode: git.ModePush, Remote: "origin", Commits: session.CommitsUnknown}))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
