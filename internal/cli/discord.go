// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// The Discord adapter's commands without a gateway (05-debug-interface-discord.md): `discord
// render` (standalone), `discord inject message|click`, `session outbox` and `session
// transcript`, served by the bot's adapter. `panel post|show` are in panel.go.

// The outcome tokens of `discord render`: the per-session limit held (R-21), or which one broke.
const (
	renderOK         = "edits<=1/s"
	renderPerSession = "edits>1/s"
	renderPerBot     = "requests>50/s"
)

var discordRender = Command{
	Name:   "discord render",
	Class:  ClassStandalone,
	Expect: true,
	Flags: []Flag{
		{Name: "events", Kind: FlagString, Value: "<jsonl>", Usage: "a file of SessionEvents, one per line, as session tail --json prints them"},
		{Name: "clock", Kind: FlagString, Value: "fake|real", Usage: "fake steps a virtual clock from one due post to the next; default real"},
		{Name: "golden", Kind: FlagString, Value: "<dir>", Usage: "render <dir>/events.jsonl on the fake clock and compare with the golden's blessed output"},
		{Name: "update", Kind: FlagBool, Usage: "with --golden: write the golden's output files instead of comparing"},
	},
	Columns: discord.OutboxColumns,
	Local:   runDiscordRender,
}

// injectArgs is `discord inject`'s one positional, every word after it: the kind, then the
// kind's own flags and text, parsed by runDiscordInject (the command tree takes two words).
const injectArgs = "message|click|command"

var discordInject = Command{
	Name:   "discord inject",
	Class:  ClassFile,
	Expect: true,
	Flags:  []Flag{{Name: injectArgs, Kind: FlagString}},
	Local:  runDiscordInject,
}

var sessionOutbox = Command{
	Name:  "session outbox",
	Class: ClassRead,
	List:  true,
	Flags: []Flag{
		{Name: "follow", Kind: FlagBool, Usage: "stream new posts until the session's last one is out"},
		{Name: "since", Kind: FlagInt, Value: "<seq>", Usage: "only posts after this seq"},
		{Name: "channel", Kind: FlagString, Value: "<channel id>", Usage: "only posts to this channel, the panel included; with the id -, any session's"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
	},
	Columns:      discord.OutboxColumns,
	EventColumns: discord.OutboxColumns,
}

var sessionTranscript = Command{
	Name:  "session transcript",
	Class: ClassRead,
	Flags: []Flag{
		{Name: "out", Kind: FlagString, Value: "<file>", Usage: "write the Markdown to this file instead of stdout"},
		{Name: "id", Kind: FlagString, Value: "<session id>"},
	},
}

// discordCheck is `broodforge discord check [--guild <id>] [--json]` (file): the bot token from
// the credential store against the guild, once, over REST; one line per configured channel,
// the notice channel and each venue category, then the message-content intent. Any missing is
// exit 1; the token is never printed. No token, a malformed one or no connection is
// start-discord, the refusal `serve --discord live` gives.
var discordCheck = Command{
	Name:  "discord check",
	Class: ClassFile,
	Flags: []Flag{
		{Name: "guild", Kind: FlagString, Value: "<id>", Usage: "the guild to check; default discord.guildId"},
	},
	Columns: discord.CheckColumns,
	Local:   runDiscordCheck,
}

// CodeDiscordMissing is `discord check` finding a channel, a permission or the intent missing.
const CodeDiscordMissing = "discord-missing"

func init() {
	commands = append(commands, &discordCheck, &discordRender, &discordInject, &sessionOutbox, &sessionTranscript)
	positionals[&discordInject] = []string{injectArgs}
	variadic[&discordInject] = true
	positionals[&sessionOutbox] = []string{"id"}
	positionals[&sessionTranscript] = []string{"id"}
	renderers[&sessionTranscript] = renderTranscript
}

// CodeGoldenDiffers is `discord render --golden` finding the render differ from the blessed files.
const CodeGoldenDiffers = "golden-differs"

// renderExclusive is the one usage error of `discord render`'s input flags: --golden stands alone,
// and one of --events and --golden is needed.
const renderExclusive = "discord render: give --events <jsonl> [--clock fake|real], or --golden <dir> [--update] alone"

func runDiscordRender(_ context.Context, env *Env) error {
	events, golden, clock := env.String("events"), env.String("golden"), env.String("clock")
	switch {
	case golden != "" && (events != "" || clock != ""),
		golden == "" && (events == "" || env.Bool("update")):
		return &usageError{renderExclusive}
	case golden != "":
		events, clock = filepath.Join(golden, discord.GoldenEvents), "fake"
	case clock == "":
		clock = "real"
	}
	if clock != "fake" && clock != "real" {
		return &usageError{"discord render: --clock " + clock + ": want fake or real"}
	}
	evs, err := readReplayEvents(events)
	if err != nil {
		return err
	}
	if golden == "" {
		return printRender(env, discord.Replay(evs, clock == "fake"))
	}
	res, got, err := discord.RenderGolden(evs)
	if err != nil {
		// The events name a session that cannot name a transcript file: the input is bad.
		return control.NewError(control.CodeBadArgs, err.Error())
	}
	dir := filepath.ToSlash(golden)
	if env.Bool("update") {
		// A write failure is an I/O error, `internal`, as scratch and backup report theirs.
		if err := discord.WriteGolden(golden, got); err != nil {
			return err
		}
		b, err := json.Marshal(map[string]string{"result": "golden", "golden": "updated", "dir": dir})
		if err != nil {
			return err
		}
		return env.Out.Record(b, []string{"result", "golden", "dir"})
	}
	want, err := discord.ReadGolden(golden)
	if err != nil {
		return control.NewError(control.CodeBadArgs, err.Error())
	}
	diffs := discord.CompareGolden(got, want)
	if len(diffs) == 0 {
		return printRender(env, res)
	}
	if err := printPosts(env, res); err != nil {
		return err
	}
	if env.JSON {
		// --json: the result line as without --golden, then one golden object per file.
		if err := printResult(env, res); err != nil {
			return err
		}
	}
	var names []string
	for _, d := range diffs {
		names = append(names, d.File)
		if err := printGoldenDiff(env, d); err != nil {
			return err
		}
	}
	// The error line is golden-differs whatever --expect asked: --expect judges the pacing token
	// of an equal render only (05-debug-interface-discord.md, Golden sessions).
	e := control.NewError(CodeGoldenDiffers, dir+": "+strings.Join(names, ", ")+"; --update, then git diff tests/discord/golden/, shows the whole")
	if env.JSON {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		fmt.Fprintln(env.Stderr, string(b))
	} else {
		fmt.Fprintf(env.Stderr, "error: %s: %s\n", e.Code, e.Message)
	}
	return &exitError{ExitFailure}
}

// printGoldenDiff prints one differing file: in text its first differing line as a golden and a
// now record, then `<file>  more  <n>` when further lines differ; in --json one golden object.
func printGoldenDiff(env *Env, d discord.GoldenDiff) error {
	if env.JSON {
		b, err := encodeDiscord(goldenRecord{Golden: d})
		if err != nil {
			return err
		}
		return env.Out.Record(b, nil)
	}
	cols := []string{"file", "line", "side", "text"}
	for _, side := range []struct{ name, text string }{{"golden", d.Golden}, {"now", d.Now}} {
		b, err := encodeDiscord(goldenSide{File: d.File, Line: strconv.Itoa(d.Line), Side: side.name, Text: side.text})
		if err != nil {
			return err
		}
		if err := env.Out.Record(b, cols); err != nil {
			return err
		}
	}
	if d.More == 0 {
		return nil
	}
	b, err := encodeDiscord(goldenSide{File: d.File, Line: "more", Side: strconv.Itoa(d.More)})
	if err != nil {
		return err
	}
	return env.Out.Record(b, cols[:3])
}

// goldenSide is one text record of a differing file: `<file>  <line>  golden|now  <text>`, or
// `<file>  more  <n>`.
type goldenSide struct {
	File string `json:"file"`
	Line string `json:"line"`
	Side string `json:"side"`
	Text string `json:"text"`
}

// goldenRecord is `discord render --golden --json`'s line per differing file.
type goldenRecord struct {
	Golden discord.GoldenDiff `json:"golden"`
}

// readReplayEvents reads a file of ReplayEvents, one JSON object per line; blank and `#` lines
// are skipped.
func readReplayEvents(path string) ([]discord.ReplayEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, control.NewError(control.CodeBadArgs, err.Error())
	}
	defer f.Close()
	var events []discord.ReplayEvent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		var ev discord.ReplayEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, control.NewError(control.CodeBadArgs, fmt.Sprintf("%s line %d: %v", filepath.ToSlash(path), n, err))
		}
		events = append(events, ev)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// printRender prints the posts, then the pacing line whose first column is the --expect token.
func printRender(env *Env, res discord.ReplayResult) error {
	if err := printPosts(env, res); err != nil {
		return err
	}
	return printResult(env, res)
}

// renderToken is the pacing outcome: the per-session limit held, or which one broke.
func renderToken(res discord.ReplayResult) string {
	switch {
	case res.MaxPerSession > 1:
		return renderPerSession
	case res.MaxPerBot > discord.GlobalBudget:
		return renderPerBot
	}
	return renderOK
}

func printPosts(env *Env, res discord.ReplayResult) error {
	env.Out.SetToken(renderToken(res))
	for _, p := range res.Posts {
		b, err := encodeDiscord(p)
		if err != nil {
			return err
		}
		if err := env.Out.Record(b, discord.OutboxColumns); err != nil {
			return err
		}
	}
	return nil
}

func printResult(env *Env, res discord.ReplayResult) error {
	b, err := json.Marshal(struct {
		Result string `json:"result"`
		discord.ReplayResult
	}{renderToken(res), res})
	if err != nil {
		return err
	}
	return env.Out.Record(b, []string{"result", "requests", "maxPerSession", "maxPerBot"})
}

// encodeDiscord is json.Marshal without HTML escaping, so `->` and `<@1001>` print as written.
func encodeDiscord[T discord.Post | discord.Reply | discord.Panel | []discord.Post | transcriptResult | goldenSide | goldenRecord](v T) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}

// injectRequest is `discord.inject`'s wire arguments. Values is a header menu's pick, from
// `click --value <v>`; the adapter refuses a menu's id without one and a button's with one.
type injectRequest struct {
	Kind     string   `json:"kind"`
	Channel  string   `json:"channel"`
	User     string   `json:"user"`
	Roles    string   `json:"roles"`
	Text     string   `json:"text"`
	CustomID string   `json:"customId"`
	Values   []string `json:"values,omitempty"`
}

// runDiscordInject parses `discord inject message --channel <id> --user <id> --roles <ids>
// "<text>"` or `discord inject click --custom-id <id> [--value <v>] --user <id> --roles <ids>`
// and sends it to the bot, which runs the adapter's handlers (refused on a live instance).
func runDiscordInject(ctx context.Context, env *Env) error {
	args := env.Strings(injectArgs)
	if len(args) == 0 || (args[0] != "message" && args[0] != "click" && args[0] != "command") {
		return &usageError{"discord inject: want message, click or command"}
	}
	kind := args[0]
	fs := flag.NewFlagSet("discord inject "+kind, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	channel := fs.String("channel", "", "")
	user := fs.String("user", "", "")
	roles := fs.String("roles", "", "")
	customID := fs.String("custom-id", "", "")
	value := fs.String("value", "", "")
	var texts []string
	rest := args[1:]
	for {
		if err := fs.Parse(rest); err != nil {
			return &usageError{"discord inject " + kind + ": " + err.Error()}
		}
		if fs.NArg() == 0 {
			break
		}
		texts = append(texts, fs.Arg(0))
		rest = fs.Args()[1:]
	}
	switch {
	case *user == "":
		return &usageError{"discord inject " + kind + ": --user is required"}
	case kind != "click" && (*channel == "" || len(texts) != 1):
		return &usageError{"discord inject " + kind + ": --channel <id> and one \"<text>\" are required"}
	case kind == "click" && (*customID == "" || len(texts) != 0):
		return &usageError{"discord inject click: --custom-id <id> is required and takes no text"}
	case kind != "click" && *value != "":
		return &usageError{"discord inject " + kind + ": --value goes with click"}
	}
	r := injectRequest{Kind: kind, Channel: *channel, User: *user, Roles: *roles, Text: strings.Join(texts, ""), CustomID: *customID}
	if *value != "" {
		r.Values = []string{*value}
	}
	req, err := json.Marshal(r)
	if err != nil {
		return err
	}
	addr, err := socketAddress(env.Root)
	if err != nil {
		return err
	}
	result, err := control.Do(ctx, addr, "discord.inject", req, nil)
	if err != nil {
		return err
	}
	return env.Out.Result(result, []string{"reply", "text"})
}

// transcriptResult is `session transcript`'s wire result.
type transcriptResult struct {
	Session  string `json:"session"`
	Markdown string `json:"markdown"`
}

// renderTranscript prints the Markdown as it is, or writes it to --out and prints `written
// <path>`; --json the result object.
func renderTranscript(env *Env, raw json.RawMessage) error {
	var t transcriptResult
	if err := json.Unmarshal(raw, &t); err != nil {
		return err
	}
	if out := env.String("out"); out != "" {
		if err := os.WriteFile(out, []byte(t.Markdown), 0o600); err != nil {
			return control.NewError(control.CodeBadArgs, err.Error())
		}
		b, err := json.Marshal(map[string]string{"result": "written", "path": filepath.ToSlash(out)})
		if err != nil {
			return err
		}
		return env.Out.Record(b, []string{"result", "path"})
	}
	if env.JSON {
		return env.Out.Record(raw, nil)
	}
	env.Out.SetToken(t.Session)
	_, err := io.WriteString(env.Out.w, t.Markdown)
	return err
}

// DiscordHandlers registers the adapter's commands on a control registry: `serve` calls it.
func DiscordHandlers(a *discord.Adapter, svc *session.Service) func(*control.Registry) {
	return func(reg *control.Registry) {
		reg.Handle("discord.inject", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var r injectRequest
			if err := json.Unmarshal(call.Args, &r); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			in := discord.Inbound{Channel: r.Channel, User: r.User, Roles: splitList(r.Roles), Text: r.Text, CustomID: r.CustomID,
				Values: r.Values}
			var rep discord.Reply
			var err error
			switch r.Kind {
			case "click":
				rep, err = a.Click(ctx, in)
			case "command":
				rep, err = a.Command(ctx, in)
			default:
				rep, err = a.Message(ctx, in)
			}
			if err != nil {
				return nil, sessionError(err)
			}
			return encodeDiscord(rep)
		})
		reg.Handle("session.outbox", func(ctx context.Context, call *control.Call) (json.RawMessage, error) {
			var r struct {
				ID, Channel string
				Follow      bool
				Since       int64
			}
			if err := json.Unmarshal(call.Args, &r); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			// --channel: every post to one channel, the panel's included, whichever session
			// (or none) made it; the session id `-` means any.
			if r.Channel != "" {
				var posts []discord.Post
				for _, p := range a.Outbox.Channel(r.Channel) {
					if p.Seq > r.Since && (r.ID == "-" || p.Session == r.ID) {
						posts = append(posts, p)
					}
				}
				if posts == nil {
					posts = []discord.Post{}
				}
				return encodeDiscord(posts)
			}
			if r.ID == "-" {
				return nil, control.NewError(control.CodeBadArgs, "session outbox -: give --channel <id> with it")
			}
			if _, err := svc.Events(r.ID, 0); err != nil {
				return nil, sessionError(err)
			}
			if !r.Follow {
				return encodeDiscord(a.Outbox.Posts(r.ID, r.Since))
			}
			err := a.Outbox.Follow(ctx, r.ID, r.Since, func(p discord.Post) (bool, error) {
				b, err := encodeDiscord(p)
				if err != nil {
					return true, err
				}
				return false, call.Emit(b)
			})
			if err != nil && !errors.Is(err, context.Canceled) {
				return nil, err
			}
			return nil, nil
		})
		reg.Handle("session.transcript", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
			var r struct{ ID string }
			if err := json.Unmarshal(call.Args, &r); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			md, err := a.Transcript(r.ID)
			if err != nil {
				return nil, sessionError(err)
			}
			return encodeDiscord(transcriptResult{Session: r.ID, Markdown: md})
		})
		panelHandlers(a, reg)
	}
}

func runDiscordCheck(ctx context.Context, env *Env) error {
	c, err := config.Load(env.Root, credentialExists)
	if err != nil {
		return configFailure(env, err)
	}
	d := c.Instance.Discord
	guild := env.String("guild")
	if guild == "" {
		guild = d.GuildID
	}
	refuse := func(msg string) error { return control.NewError(stepDiscord, msg) }
	switch {
	case guild == "":
		return refuse("no guild: give --guild <id> or set discord.guildId")
	case d.BotCredentialID == "":
		return refuse("discord.botCredentialId is not set; add the token with credential add --instance --kind bot-token")
	}
	keyPath, err := credentials.KeyPath(env.Root, c.Instance.InstanceName, c.Instance.Credentials.KeyFile, os.LookupEnv)
	if err != nil {
		return refuse(refusalText(err))
	}
	st, err := openStoreAt(env.Root)
	if err != nil {
		return err
	}
	token, err := credentials.BotToken(st, env.Root, keyPath, d.BotCredentialID)
	st.Close()
	if err != nil {
		return refuse(refusalText(err))
	}
	var chans []discord.CheckChannel
	for _, ch := range c.Channels() {
		chans = append(chans, discord.CheckChannel{ID: ch.ID, Reach: ch.Reach})
	}
	if d.NoticeChannelID != "" {
		chans = append(chans, discord.CheckChannel{ID: d.NoticeChannelID, Reach: discord.CheckNotice})
	}
	seen := map[string]bool{}
	for _, p := range c.Projects {
		if cat := p.Venue.Category; cat != "" && !seen[cat] {
			seen[cat] = true
			chans = append(chans, discord.CheckChannel{ID: cat, Reach: discord.CheckCategory})
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	res, err := discord.Check(ctx, string(token), guild, chans)
	if err != nil {
		return refuse(control.OneLine(err.Error()))
	}
	head, err := json.Marshal(map[string]string{"result": "guild", "guild": res.Guild, "name": res.Name})
	if err != nil {
		return err
	}
	if err := env.Out.Record(head, []string{"result", "guild", "name"}); err != nil {
		return err
	}
	for _, l := range res.Lines {
		b, err := json.Marshal(l)
		if err != nil {
			return err
		}
		if err := env.Out.Record(b, discord.CheckColumns); err != nil {
			return err
		}
	}
	if res.Missing > 0 {
		return control.NewError(CodeDiscordMissing, fmt.Sprintf("%d of %d missing", res.Missing, len(res.Lines)))
	}
	return nil
}

// refusalText is a credentials refusal's message without its code, any other error's text.
func refusalText(err error) string {
	if e, ok := credentials.IsRefusal(err); ok {
		return e.Message
	}
	return control.OneLine(err.Error())
}

// discordCredentials is /credential's view of the credential store (discord.Credentials): the
// caller is the user and the audit actor; a refusal becomes the session service's error type.
type discordCredentials struct{ svc *credentials.Service }

func credentialInfo(r credentials.Record) discord.CredentialInfo {
	return discord.CredentialInfo{ID: r.ID, Kind: r.Kind, Label: r.Label, Status: r.Status, LastUsedAt: r.LastUsedAt}
}

func asSessionError(err error) error {
	if e, ok := credentials.IsRefusal(err); ok {
		return &session.Error{Code: e.Code, Message: e.Message}
	}
	return err
}

func (d discordCredentials) Add(user, kind, label string, value []byte) (discord.CredentialInfo, error) {
	r, err := d.svc.AddBy(user, kind, user, label, value)
	return credentialInfo(r), asSessionError(err)
}

func (d discordCredentials) List(user string) ([]discord.CredentialInfo, error) {
	recs, err := d.svc.List(user)
	if err != nil {
		return nil, asSessionError(err)
	}
	out := make([]discord.CredentialInfo, len(recs))
	for i, r := range recs {
		out[i] = credentialInfo(r)
	}
	return out, nil
}

func (d discordCredentials) Remove(user, id string) (discord.CredentialInfo, error) {
	r, err := d.svc.RemoveBy(user, id)
	return credentialInfo(r), asSessionError(err)
}

func splitList(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}
