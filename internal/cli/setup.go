// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/setup"
)

// setupCmd is `broodforge setup [<step>] [--root <path>] [--instance <name>] [--repo <url>]
// [--project <id>] [--guild <id>] [--channels yes|no] [--create-role yes|no] [--role <name>]
// [--from <export>] [--check] [--yes] [--wait <seconds>] [--json]` (ADR 0027; the companion
// decisions/0027-setup-commands.md; 05-debug-interface-setup.md): one `ok`, `skip` or `fix`
// line per step, exit 0, 1 (one stderr `error: <code>: ...` line) or 2. It is standalone: with
// no --root and no BROODFORGE_ROOT the default root applies (05-cli-conventions.md, the
// exception to the missing-root rule). Not --check, it binds the control address for its
// lifetime, a listener beside serve; --check is (file). `setup token` is this command with the
// step `token`: lookup tries two words first and finds no `setup token`.
var setupCmd = Command{
	Name:  "setup",
	Class: ClassStandalone,
	Flags: []Flag{
		{Name: "instance", Kind: FlagString, Value: "<name>", Usage: "the instance name (default: the root's, else main)"},
		{Name: "repo", Kind: FlagString, Value: "<url>", Usage: "the project's repository, its canonical"},
		{Name: "project", Kind: FlagString, Value: "<id>", Usage: "the project id, when the one derived from the repository is invalid"},
		{Name: "guild", Kind: FlagString, Value: "<id>", Usage: "the guild, when the bot is in several"},
		{Name: "channels", Kind: FlagString, Value: "yes|no", Usage: "create channels in a guild that has some (default no)"},
		{Name: "create-role", Kind: FlagString, Value: "yes|no", Usage: "let setup create the role (default yes)"},
		{Name: "role", Kind: FlagString, Default: "Trusted", Value: "<name>", Usage: "the role setup looks up or creates"},
		{Name: "from", Kind: FlagString, Value: "<export>", Usage: "a config export to start from"},
		{Name: "check", Kind: FlagBool, Usage: "report every step; change, bind and poll nothing"},
		{Name: "yes", Kind: FlagBool, Usage: "take every shown default; refuse where there is none"},
		{Name: "wait", Kind: FlagInt, Default: "900", Value: "<seconds>", Usage: "the bound on every poll; 0 is unbounded"},
		{Name: "step", Kind: FlagString, Value: "<step>"},
	},
	Local: runSetup,
}

func init() {
	commands = append(commands, &setupCmd)
	positionals[&setupCmd] = []string{"step"}
	optional[&setupCmd] = true
}

func runSetup(ctx context.Context, env *Env) error {
	for _, f := range []string{"channels", "create-role"} {
		if v := env.String(f); v != "" && v != "yes" && v != "no" {
			return &usageError{"setup: --" + f + " " + strconv.Quote(v) + ": want yes or no"}
		}
	}
	if env.Int("wait") < 0 {
		return &usageError{"setup: --wait must be 0 or more"}
	}
	step := env.String("step")
	if step != "" && !contains(setup.StepNames(), step) {
		return &usageError{"setup: unknown step " + strconv.Quote(step) + "; want one of " + strings.Join(setup.StepNames(), ", ")}
	}
	if env.String("from") != "" {
		return control.NewError(setup.CodeNotBuilt, "setup --from (config import first) is not built yet")
	}
	tty, echoOff := false, func() func() { return nil }
	if f, ok := env.Stdin.(*os.File); ok && f == os.Stdin {
		tty, echoOff = setup.StdinTerminal()
	}
	o := setup.Options{Root: env.Root, Instance: env.String("instance"), Repo: env.String("repo"),
		Project: env.String("project"), Step: step, Check: env.Bool("check"), Yes: env.Bool("yes"), JSON: env.JSON,
		Wait: time.Duration(env.Int("wait")) * time.Second}
	// paths.install is written empty: the binary's parent directory, as the template says, never
	// a checkout or temporary directory the binary happened to run from.
	return setup.Run(ctx, o, setup.Env{Stdin: env.Stdin, Stdout: env.Out.w, Stderr: env.Stderr, TTY: tty, EchoOff: echoOff,
		LookupEnv: os.LookupEnv, LookPath: exec.LookPath, Checks: setupChecks})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// setupChecks is the check step's `config check` and `spawn preflight`, run in process on root
// with their own output captured: each stdout line echoed as ok, each stderr line and the
// command's error line as a failure, every line prefixed with the command's name.
func setupChecks(ctx context.Context, root string) []setup.Echo {
	var out []setup.Echo
	for _, c := range []*Command{&configCheck, &spawnPreflight} {
		var stdout, stderr bytes.Buffer
		env := &Env{Root: root, Stdin: strings.NewReader(""), Stderr: &stderr, Out: NewPrinter(&stdout, false, -1),
			values: map[string]*flagValue{}}
		err := c.Local(ctx, env)
		lines := func(b *bytes.Buffer, ok bool) {
			sc := bufio.NewScanner(b)
			for sc.Scan() {
				if t := strings.TrimSpace(sc.Text()); t != "" {
					out = append(out, setup.Echo{OK: ok && !strings.HasPrefix(t, "error: "), Text: c.Name + "  " + t})
				}
			}
		}
		lines(&stdout, true)
		lines(&stderr, false)
		if err != nil {
			e := toWire(err)
			out = append(out, setup.Echo{Text: c.Name + "  error: " + e.Code + ": " + e.Message})
		}
	}
	return out
}
