// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/policy"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// Every string a person sees in Discord is built here, from 04-ux.md, so the wording is read
// and changed in one file. A string 04 does not give word for word is marked `not 04:`; the
// pass report lists them for the owner.
//
// not 04: names. 04 writes a first name ("driver Jason", "Allowed by Jason", "Stopped by
// Jason", "ended by Jason"); these strings show the full registered git name, which is all the
// store holds. The header, prompt answers, Stop line and summary carry it. Owner's call.

// MessageLimit is Discord's content limit; near it a working message is frozen and a new one
// continues it (R-22), and a final message is split.
const MessageLimit = 2000

// maxLineRunes bounds one line, so a line plus a header always fits in one message.
const maxLineRunes = 1900

// modeName is the mode's name people see (04, Owner 7).
func modeName(mode string) string {
	if mode == policy.ModePolicy.String() {
		return "acts freely"
	}
	return "asks first"
}

// modelShort is the footer's model name: `Opus` for claude-opus-5-5 (04, The turn loop); an id
// policy does not know is shown as it is.
func modelShort(id string) string {
	name := policy.ModelDisplayName(id)
	rest, ok := strings.CutPrefix(name, "Claude ")
	if !ok {
		return name
	}
	if f := strings.Fields(rest); len(f) > 0 {
		return f[0]
	}
	return name
}

// tokens is a token count as 04 writes it: 950, 48.2k, 212k, 1.2M.
func tokens(n int64) string {
	switch {
	case n < 1000:
		return strconv.FormatInt(n, 10)
	case n < 100_000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	case n < 1_000_000:
		return fmt.Sprintf("%dk", (n+500)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}

// clock is a turn's elapsed time as 04 writes it: 0:12, 1:42; past an hour 1:02:03.
func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int64(d / time.Second)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// span is a session's length as the summary writes it: 1h 12m; not 04: 12m under an hour and
// 45s under a minute.
func span(d time.Duration) string {
	s := int64(d / time.Second)
	switch {
	case s >= 3600:
		return fmt.Sprintf("%dh %dm", s/3600, s/60%60)
	case s >= 60:
		return fmt.Sprintf("%dm", s/60)
	}
	return fmt.Sprintf("%ds", s)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

// mention is a user mention as Discord sends it; Discord shows it as @<name>.
func mention(user string) string { return "<@" + user + ">" }

// channelRef is a channel mention as Discord sends it; Discord shows it as #<name>.
func channelRef(id string) string { return "<#" + id + ">" }

// venueName is a private channel's name (04, Venue): `s0001-jason`, the driver's first name
// lowercased, letters and digits only.
func venueName(id, driverName string) string {
	first := ""
	if f := strings.Fields(driverName); len(f) > 0 {
		first = strings.ToLower(f[0])
	}
	var b strings.Builder
	for _, r := range first {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return id
	}
	return id + "-" + b.String()
}

// roleList names a channel's role list for people: `the driver`, a group name capitalised
// (`Trusted`), a raw role id as a role mention. not 04: the joining with commas.
func roleList(list []string) string {
	if len(list) == 0 {
		return "nobody"
	}
	out := make([]string, len(list))
	for i, e := range list {
		switch {
		case e == policy.RoleDriver:
			out[i] = "the driver"
		case channelIDRE.MatchString(e):
			out[i] = "<@&" + e + ">"
		default:
			out[i] = strings.ToUpper(e[:1]) + e[1:]
		}
	}
	return strings.Join(out, ", ")
}

// credentialKindName is the header's credential kind: 04's `Claude subscription`; not 04:
// `API key` for a key.
func credentialKindName(kind string) string {
	if kind == "api-key" {
		return "API key"
	}
	return "Claude subscription"
}

// headerText is the venue's pinned header (04, Starting and the first message).
// not 04: the name in it is the full registered name (see the top of this file).
func headerText(id string, st *session.StartInfo) string {
	why := "this channel's default"
	if st.Mode != st.ChannelMode {
		why = "chosen at start" // not 04
	}
	return "Session " + id + " on " + st.ProjectName + ", branch " + st.Branch + ", driver " + st.DriverName + ".\n" +
		"Mode: " + modeName(st.Mode) + " (" + why + "). Credential: " + st.Credential + " (" + credentialKindName(st.CredentialKind) + ").\n" +
		"Reach: " + st.Reach + ". Ends after " + st.IdleTimeout + " without activity. Who may answer prompts: " + roleList(st.Answer) + ".\n" +
		"Write what you want done. Everyone who writes here is handed to the agent by name."
}

// workingHeader is the working message's first line (04, The turn loop).
func workingHeader(turn int, elapsed time.Duration) string {
	return "Working · turn " + strconv.Itoa(turn) + " · " + clock(elapsed)
}

// waitingHeader replaces it on a working message frozen while a call waits for an answer.
// not 04.
func waitingHeader(turn int, elapsed time.Duration) string {
	return "Waiting for an answer · turn " + strconv.Itoa(turn) + " · " + clock(elapsed)
}

// Footer labels: 04's Done and Stopped; not 04: Failed, for a turn that ended in an error.
const (
	labelDone    = "Done"
	labelStopped = "Stopped"
	labelFailed  = "Failed"
)

// footer is a turn's last line (04, The turn loop; Owner 5).
func footer(label string, turn int, elapsed time.Duration, u *session.TurnUsage) string {
	if u == nil {
		u = &session.TurnUsage{}
	}
	out := tokens(u.Out) + " out"
	if u.OutUnknown {
		out = "out unknown"
	}
	parts := []string{label, "turn " + strconv.Itoa(turn), clock(elapsed), tokens(u.In) + " in / " + out,
		"session " + tokens(u.SessionIn)}
	if u.Model != "" {
		parts = append(parts, modelShort(u.Model))
	}
	parts = append(parts, modeName(u.Mode))
	return strings.Join(parts, " · ")
}

// stoppedText is 04's Stop line, above the footer.
// not 04: the name in it is the full registered name (see the top of this file).
func stoppedText(by string, elapsed time.Duration) string {
	return "Stopped by " + by + " after " + clock(elapsed) + ". What was done so far stays in the tree. Say what to do next."
}

// turnFailedText is 04's turn-failed flow; not 04: `exit unknown` when no code was seen.
func turnFailedText(exit string) string {
	if exit == "" {
		exit = "unknown"
	}
	return "The agent's process ended without a result (exit " + exit + "). Nothing in the tree was lost; tokens so far are recorded. Send your message again, or end the session."
}

// turnErrorText is a turn that ended in an error with a reason. not 04: 04's turn-failed
// wording with the reason in place of the exit code.
func turnErrorText(reason string) string {
	return "The agent's turn ended in an error (" + reason + "). Nothing in the tree was lost; tokens so far are recorded. Send your message again, or end the session."
}

// promptText is the permission prompt (04, Permission prompts): the mention so a phone buzzes,
// the call's trace line, the full command, and the parallel-calls line when the engine dropped
// some (C28).
func promptText(driver string, turn int, trace, command string, dropped int) string {
	lines := []string{mention(driver) + " Permission needed · turn " + strconv.Itoa(turn), trace}
	if command != "" {
		lines = append(lines, "Command: "+command)
	}
	if dropped > 0 {
		lines = append(lines, "The agent asked for "+strconv.Itoa(dropped+1)+" actions at once; this is the last. The others will be asked again one at a time.")
	}
	return strings.Join(lines, "\n")
}

// promptButtons are the prompt's three buttons (04, Owner 6).
func promptButtons(id, toolUseID, tool string) []Button {
	cid := func(d string) string {
		return CustomID{Kind: IDAnswer, Session: id, ToolUseID: toolUseID, Decision: d}.String()
	}
	return []Button{
		{Label: "Allow", CustomID: cid(DecisionAllow), Style: StyleSuccess},
		{Label: "Deny", CustomID: cid(DecisionDeny), Style: StyleDanger},
		{Label: "Allow " + tool + " this session", CustomID: cid(DecisionAllowSession), Style: StyleSecondary},
	}
}

// answeredText is what a prompt becomes once answered (04): `Allowed by Jason`, `Denied by
// Jason`, `No answer; denied`. not 04: the trace line kept under it, and `No answer; <cause>`
// for a call dropped because the session ended or the bot stopped.
// not 04: the name in it is the full registered name (see the top of this file).
func answeredText(ev session.Event, by, trace string) string {
	var first string
	switch {
	case ev.DecidedBy == session.DecidedByPromptTimeout:
		first = "No answer; denied"
	case ev.Decision == session.DecisionDropped:
		first = "No answer; " + ev.Reason
	case ev.Decision == session.DecisionAllow:
		first = "Allowed by " + by
	default:
		first = "Denied by " + by
	}
	return first + "\n" + trace
}

// modeChangedText is the mode switch's post (04, Policy mode); not 04: the asks-first form.
func modeChangedText(mode, reach string) string {
	if mode == policy.ModePolicy.String() {
		return "From the next turn: acts freely. Broodforge answers permission prompts itself, inside this\nchannel's reach (" + reach + "). Switch back with /session mode asks-first."
	}
	return "From the next turn: asks first. Permission prompts come here for an answer. Switch with /session mode acts-freely."
}

// handoffRefusedText is 04's handoff-refused flow around the refusal's own message.
func handoffRefusedText(message string) string {
	return "Not handed off: " + strings.TrimSuffix(message, ".") + ". Ask the agent to amend them, then end the session again."
}

// handoffFailedText is 04's handoff-failed flow; not 04: the refusal code as the one line.
func handoffFailedText(id, code string, commits int) string {
	return "Push rejected by the remote: " + code + ". Your " + plural(commits, "commit is", "commits are") +
		" kept on the host as " + id + "; the owner can retry with broodforge worktree handoff " + id + "."
}

// summaryText is the session's summary (04, Ending a session).
// not 04: the name in it is the full registered name (see the top of this file).
func summaryText(id string, end *session.EndInfo, ended time.Time) string {
	by := end.By
	if by == "" {
		by = "the idle timeout" // not 04
	}
	head := "Session " + id + " ended by " + by
	if end.Commits == 0 && (end.Outcome == "none" || end.Outcome == "") {
		return head + " · 0 commits · nothing to push; the tree was removed."
	}
	started, err := time.Parse(time.RFC3339, end.StartedAt)
	length := ""
	if err == nil {
		length = " · " + span(ended.Sub(started))
	}
	model := ""
	if end.Model != "" {
		model = " · " + modelShort(end.Model)
	}
	lines := []string{head + length + " · " + plural(end.Turns, "turn", "turns") + " · " + tokens(end.In) + " in / " + tokens(end.Out) + " out" + model}
	switch {
	case end.Outcome == "pull-request":
		lines = append(lines, "Branch "+end.Branch+" -> pull request "+pullNumber(end.URL)+end.URL)
	case end.Outcome == "pushed":
		lines = append(lines, "Branch "+end.Branch+" -> pushed") // not 04
	default:
		lines = append(lines, "Branch "+end.Branch+" -> not handed off ("+end.Outcome+")") // not 04
	}
	lines = append(lines, "Commits: "+strconv.Itoa(end.Commits)) // not 04: 04 lists each commit's hash and subject
	co := "none"                                                 // not 04
	if len(end.CoAuthors) > 0 {
		co = strings.Join(end.CoAuthors, ", ")
	}
	lines = append(lines, "Author: "+end.Author+". Co-authors: "+co+".", "Unmerged subagent branches removed: none.")
	return strings.Join(lines, "\n")
}

// pullNumber is `#42 ` from a pull request URL ending in /pull/42, "" otherwise.
func pullNumber(url string) string {
	i := strings.LastIndex(url, "/pull/")
	if i < 0 {
		return ""
	}
	n := url[i+len("/pull/"):]
	if _, err := strconv.Atoi(n); err != nil {
		return ""
	}
	return "#" + n + " "
}

// Inbound replies and refusals (04, Registration; The project channel and the panel;
// Credentials). Every refusal is ephemeral.
const (
	notRegisteredText = "You are not registered. Run /register with your name and email first; it takes ten seconds."
	keyPastedText     = "Never paste a key in chat. Use /credential add kind:api-key, which opens a private form."
	lobbyHintText     = "Start a session with the button above or /session start; this channel is the lobby."
)

func registeredText(name, email string) string {
	return "Registered. Commits you drive are authored as " + name + " <" + email + ">.\nNext: add a credential with /credential add, then start a session from a project channel."
}

func updatedText(oldName, oldEmail, name, email string) string {
	return "Updated: was " + oldName + " <" + oldEmail + ">, now " + name + " <" + email + ">."
}

func startRolesText(channel string) string {
	return "Your roles do not allow starting a session in " + channelRef(channel) + ". Ask a server admin."
}

// contributeRolesText is a venue message refused for roles. not 04: adapted from the invite's.
func contributeRolesText(channel string) string {
	return "Your roles do not allow contributing in " + channelRef(channel) + "."
}

func couldNotStartText(reason string) string { return "Could not start a session: " + reason + "." }

// startedText is the start's ephemeral reply. not 04.
func startedText(id string) string { return "Session " + id + " is open in " + channelRef(id) + "." }

// panelText is the project channel's panel (04, The project channel and the panel).
func panelText(projectName, handoff string, start []string) string {
	ending := "ending it makes a pull request"
	if handoff == "push" {
		ending = "ending it pushes the branch" // not 04
	}
	return "Work on " + projectName + " with Claude Code: one session is one branch; " + ending + ". May start: " + roleList(start) + "."
}
