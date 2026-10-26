// SPDX-License-Identifier: GPL-3.0-or-later

// Package setup is `broodforge setup` (ADR 0027; its companion decisions/0027-setup-commands.md):
// the ordered steps that make an instance from nothing, each printing one `ok`, `skip` or `fix`
// line, a fix that needs a hand polled to --wait, the prompts and their flags, the rerun rule
// per key, and the single-writer bind of the control address for the run's lifetime. The
// Discord steps (discord.go) call Discord through discord.SetupAPI.
package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Result is a step line's first column.
type Result int

const (
	ResultOK Result = iota
	ResultSkip
	ResultFix
)

func (r Result) String() string {
	switch r {
	case ResultOK:
		return "ok"
	case ResultSkip:
		return "skip"
	case ResultFix:
		return "fix"
	}
	return "fix"
}

// Line is one step line: `<result>  <step>  <detail>` (05-cli-conventions.md, Output), or with
// --json `{"result":...,"step":...,"detail":...}`; the key step's line adds the key and its
// path, the one print.
type Line struct {
	Result Result
	Step   string
	Detail string
	Key    string // the key step's new key, printed once
	Path   string // its file
	// Wait marks a fix that needs a hand: the runner polls the step until it passes or --wait
	// runs out, then refuses timeout-<step>.
	Wait bool
	// After runs once the line is printed: the key step's pause.
	After func() error
	// Note is printed on stderr after a waiting fix, before the poll: "waiting for the bot to
	// join".
	Note string
}

// Step is one step of the run; Run returns the step's line, and a refusal ends the run. A
// step may print lines of its own before (State.emit), as check echoes its commands' lines.
type Step struct {
	Name string
	Run  func(ctx context.Context, s *State) (Line, error)
}

// Error codes of the refusal table (decisions/0027-setup-commands.md, Refusals) built here.
const (
	CodeRunning      = "running"
	CodeSetupRunning = "setup-running"
	CodeNeedsElev    = "needs-elevation"
	CodeKeyInUse     = "key-in-use"
	CodeCheckFailed  = "check-failed"
	CodeNoConfig     = "no-config"
	CodeLiveInstance = "live-instance"
)

func needsInput(step, flag string) error {
	return control.NewError("needs-input-"+step, "pass --"+flag)
}

func mismatch(step, have, found string) error {
	return control.NewError("mismatch-"+step, "config has "+have+", found "+found+"; run setup "+step+" to replace it")
}

func refused(step, msg string) error { return control.NewError("refused-"+step, msg) }

// Options are the command's flags (decisions/0027-setup-commands.md, `broodforge setup`).
type Options struct {
	Root     string // --root or BROODFORGE_ROOT; "" is the default root
	Instance string // --instance; "" is the root's configured name, else main
	Repo     string
	Project  string
	Step     string // `setup <step>`: that step alone
	Check    bool
	Yes      bool
	JSON     bool
	Wait     time.Duration // the bound on every poll; 0 is unbounded
	// The Discord steps' flags: --guild, --channels and --create-role ("yes", "no" or "" for
	// the question), --role ("" is Trusted).
	Guild      string
	Channels   string
	CreateRole string
	Role       string
	// From is --from: a config export (config.Export) imported before the steps, absent keys
	// only, its copy kept at <root>/data/config-export.json.
	From string
}

// Echo is one line of a command the check step runs in process (config check, spawn
// preflight): its text as that command prints it, OK false for an error line.
type Echo struct {
	OK   bool
	Text string
}

// Clock is the runner's time, injectable so a poll is tested without waiting.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time        { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }

// RealClock is the wall clock.
var RealClock Clock = realClock{}

// Env is what the run takes from its process, each part injectable for tests.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer // the step lines
	Stderr io.Writer // the prompts
	// TTY is whether Stdin is a terminal; EchoOff turns its echo off for a hidden answer and
	// returns the restore (nil off a terminal).
	TTY     bool
	EchoOff func() (restore func())
	// LookupEnv reads the environment: ProgramData, BROODFORGE_KEY_FILE.
	LookupEnv func(string) (string, bool)
	// LookPath finds the engine on PATH.
	LookPath func(string) (string, error)
	Clock    Clock
	// Checks runs `config check` and `spawn preflight` on root in process (the check step).
	Checks func(ctx context.Context, root string) []Echo
	// Install is paths.install for a new instance file: the directory holding bin/ and share/.
	Install string
	// Discord builds the Discord calls over a bot token, for one step: discord.NewSetupAPI, or
	// the scripted fake's API. nil leaves Discord out of the run: the five Discord steps print
	// `skip  <step>  no Discord`, for the tests of the other steps.
	Discord func(token string) (discord.SetupAPI, error)
	// FakeDiscord marks Discord as the scripted fake (BROODFORGE_SETUP_FAKE_DISCORD): refused
	// live-instance on a live root, and discord.mode is left dry-run.
	FakeDiscord bool
}

// pollEvery is how often a waiting fix is tried again.
const pollEvery = 2 * time.Second

// Run is one `broodforge setup` invocation: the root and instance resolved (asking when
// neither flag gives them), the control address bound unless --check, then every step in order
// or the one named. A refusal is a *control.Error, whose code the CLI prints.
func Run(ctx context.Context, o Options, env Env) error {
	if env.Clock == nil {
		env.Clock = RealClock
	}
	steps := Steps()
	if o.Step != "" {
		i := slices.IndexFunc(steps, func(s Step) bool { return s.Name == o.Step })
		if i < 0 {
			return fmt.Errorf("unknown step %q", o.Step)
		}
		steps = steps[i : i+1]
	}
	s := &State{opt: o, env: env, check: o.Check, alone: o.Step != "",
		prompt: &prompter{in: newLineReader(env.Stdin), out: env.Stderr, tty: env.TTY, yes: o.Yes, echoOff: env.EchoOff}}
	if o.From != "" {
		if err := s.readExport(); err != nil {
			return err
		}
	}
	if err := s.resolve(); err != nil {
		return err
	}
	if env.FakeDiscord {
		// An instance file that cannot be read cannot show the root is not live: refused. With
		// no instance file, the export --from would import is the instance.
		inst, err := s.instanceDoc()
		if err != nil {
			return toControl(err)
		}
		if inst == nil && s.from != nil {
			inst = s.from.Instance
		}
		if inst != nil && inst.String("discord", "mode") == config.DiscordLive {
			return control.NewError(CodeLiveInstance, "the scripted Discord (BROODFORGE_SETUP_FAKE_DISCORD) is refused on a live instance")
		}
	}
	if !s.check {
		// The root's configured name may differ from --instance: a bot under that name owns
		// the files setup would write.
		if c := s.configured; c != "" && c != s.instance {
			if addr, err := control.Address(s.root, c); err == nil {
				if err := probe(ctx, addr); err != nil {
					return err
				}
			}
		}
		b, err := bind(ctx, s.root, s.instance)
		if err != nil {
			return err
		}
		defer b.close()
		if s.from != nil {
			if err := s.importExport(); err != nil {
				return err
			}
		}
	}
	defer s.close()
	return s.run(ctx, steps)
}

// StepNames is every step's name, in order.
func StepNames() []string {
	var out []string
	for _, s := range Steps() {
		out = append(out, s.Name)
	}
	return out
}

// run runs the steps: a refusal ends a run; --check prints every line and refuses
// check-failed naming the steps that printed fix.
func (s *State) run(ctx context.Context, steps []Step) error {
	for _, st := range steps {
		if err := s.runStep(ctx, st); err != nil {
			return err
		}
	}
	if s.check && len(s.fixed) > 0 {
		return control.NewError(CodeCheckFailed, strings.Join(s.fixed, " "))
	}
	return nil
}

func (s *State) runStep(ctx context.Context, st Step) error {
	line, err := st.Run(ctx, s)
	if err != nil {
		var ce *control.Error
		if !asControl(err, &ce) {
			ce = control.NewError(control.CodeInternal, err.Error())
		}
		if line.Step == "" {
			line = Line{Result: ResultFix, Step: st.Name, Detail: ce.Message}
		}
		if perr := s.emit(line); perr != nil {
			return perr
		}
		if s.check {
			return nil // --check reports the refusal as the step's fix and goes on
		}
		return ce
	}
	if err := s.emit(line); err != nil {
		return err
	}
	if line.Result == ResultFix && !s.check {
		if !line.Wait {
			return control.NewError("refused-"+st.Name, line.Detail)
		}
		if line.Note != "" && s.env.Stderr != nil && !s.opt.JSON { // --json: stderr is the wire object only
			fmt.Fprintln(s.env.Stderr, line.Note)
		}
		if line, err = s.poll(ctx, st, line); err != nil {
			return err
		}
	}
	if line.After != nil && !s.check {
		return line.After()
	}
	return nil
}

// poll runs a waiting fix's step again every pollEvery until it passes, printing its passing
// line, or refuses timeout-<step> when --wait runs out.
func (s *State) poll(ctx context.Context, st Step, fix Line) (Line, error) {
	start := s.env.Clock.Now()
	for {
		if s.opt.Wait > 0 && s.env.Clock.Now().Sub(start) >= s.opt.Wait {
			return fix, control.NewError("timeout-"+st.Name, "still waiting after "+
				strconv.Itoa(int(s.opt.Wait/time.Second))+"s: "+fix.Detail)
		}
		if err := ctx.Err(); err != nil {
			return fix, err
		}
		s.env.Clock.Sleep(pollEvery)
		line, err := st.Run(ctx, s)
		if err != nil {
			return line, err
		}
		if line.Result != ResultFix {
			return line, s.emit(line)
		}
	}
}

// emit prints one line; a fix line under --check is remembered for check-failed.
func (s *State) emit(l Line) error {
	if l.Result == ResultFix && s.check && !slices.Contains(s.fixed, l.Step) {
		s.fixed = append(s.fixed, l.Step)
	}
	if s.opt.JSON {
		return writeJSON(s.env.Stdout, lineJSON{l.Result.String(), l.Step, l.Detail, l.Key, l.Path})
	}
	detail := l.Detail
	if l.Key != "" {
		detail = l.Key + "  " + l.Path
	}
	if detail == "" {
		detail = "-"
	}
	_, err := fmt.Fprintln(s.env.Stdout, l.Result.String()+"  "+l.Step+"  "+detail)
	return err
}

// replace is `setup <step>`'s overwrite of a present key: the old and new records, then the
// question (--yes accepts). Values are an id, a path or a plain setting, never a secret.
func (s *State) replace(step, old, new string) (bool, error) {
	if s.opt.JSON {
		if err := writeJSON(s.env.Stdout, replaceJSON{step, old, new}); err != nil {
			return false, err
		}
	} else if _, err := fmt.Fprintf(s.env.Stdout, "old  %s\nnew  %s\n", old, new); err != nil {
		return false, err
	}
	return s.prompt.Confirm(step, "Replace it? [y/N]")
}

// lineJSON is a step line with --json; replaceJSON the old and new record of `setup <step>`.
type lineJSON struct {
	Result string `json:"result"`
	Step   string `json:"step"`
	Detail string `json:"detail"`
	Key    string `json:"key,omitempty"`
	Path   string `json:"path,omitempty"`
}

type replaceJSON struct {
	Step string `json:"step"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

// writeJSON prints v as one compact line without HTML escaping, so `<credential id>` reads as
// written.
func writeJSON[T lineJSON | replaceJSON](w io.Writer, v T) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// close releases what the run opened.
func (s *State) close() {
	if s.st != nil {
		_ = s.st.Close()
		s.st = nil
	}
}

// openStore is data/store.sqlite, created and migrated when missing (store.Create's caller).
func (s *State) openStore() (*store.Store, error) {
	if s.st == nil {
		st, err := store.Create(s.root)
		if err != nil {
			return nil, err
		}
		s.st = st
	}
	return s.st, nil
}
