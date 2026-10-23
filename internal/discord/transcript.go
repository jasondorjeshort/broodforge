// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// Transcript is the session's Markdown transcript (04, Ending a session; Owner 11): every
// message with its writer and time, every trace line, every permission decision and its
// decider, every footer, the summary; masked like every post. who names a user id. not 04:
// the layout (headings per turn, a list for the trace); 04 names only what it holds.
func Transcript(id string, events []session.Event, who func(user string) string) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }
	// blank ends a paragraph: one empty line, never two.
	blank := func() {
		if s := b.String(); s != "" && !strings.HasSuffix(s, "\n\n") {
			b.WriteString("\n")
		}
	}
	var start *session.StartInfo
	var turnStart time.Time
	var queued []string // times of messages waiting for the next turn
	for _, ev := range events {
		t := parseTime(ev.Time)
		switch ev.Type {
		case session.EventSessionStarted:
			start = ev.Start
			line("# Session " + id)
			blank()
			line(ev.Summary)
			if start != nil {
				line("Mode: " + modeName(start.Mode) + ". Reach: " + start.Reach + ". Credential: " + start.Credential + ".")
			}
			line("Started " + ev.Time + ".")
		case session.EventMessageQueued:
			queued = append(queued, ev.Time)
		case session.EventTurnStarted:
			turnStart = t
			blank()
			line("## Turn " + strconv.Itoa(ev.Turn) + " · " + ev.Time)
			blocks := splitBlocks(ev.Text)
			times := queued
			if len(times) < len(blocks) {
				times = append([]string{ev.Time}, times...)
			}
			queued = nil
			for i, blk := range blocks {
				writer, text, _ := strings.Cut(blk, "\n")
				at := ev.Time
				if i < len(times) {
					at = times[i]
				}
				blank()
				line("**" + strings.TrimPrefix(writer, "From: ") + "** · " + at)
				blank()
				for _, l := range strings.Split(text, "\n") {
					line("> " + l)
				}
			}
			blank()
		case session.EventText:
			blank()
			line(ev.Text)
			blank()
		case session.EventToolCall:
			line("- `" + ev.Summary + "`")
		case session.EventPermissionDecision:
			by := ev.DecidedBy
			if ev.User != "" {
				by = who(ev.User) + ", " + ev.DecidedBy
			}
			reason := ""
			if ev.Reason != "" {
				reason = ": " + ev.Reason
			}
			line("  - " + ev.Decision + " (" + by + ")" + reason)
		case session.EventToolResult:
			if strings.Contains(ev.Summary, " -> ") {
				line("  - `" + ev.Summary + "`")
			}
		case session.EventWaitingForAnswer:
			line("- waiting for an answer: `" + session.TraceLine(ev.Tool, ev.Target) + "`")
		case session.EventTurnEnded:
			label := labelDone
			switch {
			case ev.Result == session.ResultCompleted:
			case ev.Result == session.ResultError || (ev.User == "" && ev.Summary == session.ResultInterrupted):
				label = labelFailed
			default:
				label = labelStopped
			}
			blank()
			line(footer(label, ev.Turn, t.Sub(turnStart), ev.Usage))
		case session.EventModeChanged, session.EventModelChanged, session.EventEffortChanged, session.EventParticipantJoined, session.EventParticipantInvited,
			session.EventMessageDropped, session.EventHandoffRefused, session.EventSessionInterrupted:
			blank()
			line("_" + ev.Time + "_ " + ev.Summary)
		case session.EventSessionEnded:
			blank()
			line("## Ended · " + ev.Time)
			blank()
			if ev.End != nil {
				line(summaryText(id, ev.End, t))
			} else {
				line(ev.Summary)
			}
		}
	}
	return Mask(b.String())
}

// splitBlocks splits a turn's input into its From blocks (03 Data flow, step 2).
func splitBlocks(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, "\n\nFrom: ")
	for i := 1; i < len(parts); i++ {
		parts[i] = "From: " + parts[i]
	}
	return parts
}
