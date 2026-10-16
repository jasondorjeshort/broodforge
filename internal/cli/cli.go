// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/control"
)

// Exit codes, docs/design/05-cli-conventions.md (N-07).
const (
	ExitOK      = 0
	ExitFailure = 1
	ExitUsage   = 2
)

// RootEnv names the instance root when --root is not given.
const RootEnv = "BROODFORGE_ROOT"

// Error codes the CLI produces itself.
const (
	CodeUsage     = "usage"
	CodeExpect    = "expect"
	CodeNoConfig  = "no-config"
	CodeBadConfig = "bad-config"
)

const usageLine = "usage: broodforge [--root <path>] [--json] <group> <command> [flags]"

// usageError is an exit-2 error: unknown command or flag, missing root, malformed stdin.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// invocation holds the flags every command shares.
type invocation struct {
	root      string
	json      bool
	expect    string
	expectSet bool
	limit     int
	all       bool
	stdout    io.Writer
	stderr    io.Writer
}

// Run dispatches one invocation and returns the process exit code. args excludes the program
// name. --root and --json may come before the command or among its flags; a command's own flags
// follow its name and precede any argument.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	inv := &invocation{stdout: stdout, stderr: stderr}
	global := flag.NewFlagSet("broodforge", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	global.StringVar(&inv.root, "root", "", "")
	global.BoolVar(&inv.json, "json", false, "")
	if err := global.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return ExitOK
		}
		return inv.fail(&usageError{err.Error()})
	}
	rest := global.Args()
	if len(rest) == 0 {
		printUsage(stderr)
		return ExitUsage
	}
	cmd, n := lookup(rest)
	if cmd == nil {
		return inv.fail(&usageError{"unknown command " + strconv.Quote(strings.Join(rest[:min(2, len(rest))], " "))})
	}
	env := &Env{Stdin: stdin, Stderr: stderr, values: map[string]*flagValue{}}
	fs, err := inv.flagSet(cmd, env)
	if err != nil {
		return inv.fail(err)
	}
	if err := fs.Parse(rest[n:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, commandUsage(cmd))
			return ExitOK
		}
		return inv.fail(&usageError{cmd.Name + ": " + err.Error()})
	}
	if fs.NArg() > 0 {
		return inv.fail(&usageError{cmd.Name + ": unexpected argument " + strconv.Quote(fs.Arg(0))})
	}
	if inv.root == "" {
		inv.root = os.Getenv(RootEnv)
	}
	if inv.root == "" && cmd.Class != ClassStandalone {
		return inv.fail(&usageError{"no instance root: pass --root <path> or set " + RootEnv})
	}
	limit := inv.limit
	if inv.all {
		limit = -1
	}
	env.Root, env.JSON = inv.root, inv.json
	env.Out = NewPrinter(stdout, inv.json, limit)
	return inv.finish(env.Out, execute(context.Background(), cmd, env))
}

// flagSet builds the command's flag set: its own flags, the shared --root and --json, and
// --expect, --limit and --all where the command offers them.
func (inv *invocation) flagSet(cmd *Command, env *Env) (*flag.FlagSet, error) {
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&inv.root, "root", inv.root, "")
	fs.BoolVar(&inv.json, "json", inv.json, "")
	if cmd.Expect {
		fs.Func("expect", "", func(s string) error { inv.expect, inv.expectSet = s, true; return nil })
	}
	if cmd.List {
		fs.IntVar(&inv.limit, "limit", 0, "")
		fs.BoolVar(&inv.all, "all", false, "")
	}
	for _, f := range cmd.Flags {
		v := &flagValue{kind: f.Kind}
		env.values[f.Name] = v
		switch f.Kind {
		case FlagString:
			fs.StringVar(&v.s, f.Name, f.Default, f.Usage)
		case FlagInt:
			d := 0
			if f.Default != "" {
				var err error
				if d, err = strconv.Atoi(f.Default); err != nil {
					return nil, fmt.Errorf("%s: flag %s: default %q is not an int", cmd.Name, f.Name, f.Default)
				}
			}
			fs.IntVar(&v.i, f.Name, d, f.Usage)
		case FlagBool:
			fs.BoolVar(&v.b, f.Name, f.Default == "true", f.Usage)
		}
	}
	return fs, nil
}

// execute runs the command by its class.
func execute(ctx context.Context, cmd *Command, env *Env) error {
	switch cmd.Class {
	case ClassWrite:
		return socketCall(ctx, cmd, env)
	case ClassRead:
		err := socketCall(ctx, cmd, env)
		var ce *control.Error
		if cmd.Local != nil && errors.As(err, &ce) && ce.Code == control.CodeNotRunning {
			return cmd.Local(ctx, env)
		}
		return err
	case ClassFile, ClassStandalone:
		if cmd.Local == nil {
			return fmt.Errorf("%s has no local implementation", cmd.Name)
		}
		return cmd.Local(ctx, env)
	}
	return fmt.Errorf("%s: unknown class %d", cmd.Name, cmd.Class)
}

// finish turns the outcome into the exit code, the error line and the --expect comparison.
func (inv *invocation) finish(out *Printer, err error) int {
	expecting := inv.expectSet
	if err == nil {
		if expecting && out.Token() != inv.expect {
			return inv.fail(control.NewError(CodeExpect, "wanted "+inv.expect+", got "+orDash(out.Token())))
		}
		return ExitOK
	}
	var ue *usageError
	if errors.As(err, &ue) {
		return inv.fail(err)
	}
	code := errorCode(err)
	if expecting {
		if code == inv.expect {
			return ExitOK
		}
		return inv.fail(control.NewError(CodeExpect, "wanted "+inv.expect+", got "+code))
	}
	return inv.fail(err)
}

// fail prints the one error line and returns its exit code: `error: <code>: <message>`, or the
// wire error object with --json.
func (inv *invocation) fail(err error) int {
	exit := ExitFailure
	e := toWire(err)
	var ue *usageError
	if errors.As(err, &ue) {
		exit = ExitUsage
	}
	if inv.json {
		b, merr := json.Marshal(e)
		if merr == nil {
			fmt.Fprintln(inv.stderr, string(b))
			return exit
		}
	}
	fmt.Fprintf(inv.stderr, "error: %s: %s\n", e.Code, e.Message)
	return exit
}

func toWire(err error) *control.Error {
	var ue *usageError
	if errors.As(err, &ue) {
		return control.NewError(CodeUsage, ue.msg)
	}
	var ce *control.Error
	if errors.As(err, &ce) && control.ValidCode(ce.Code) {
		return control.NewError(ce.Code, ce.Message)
	}
	return control.NewError(control.CodeInternal, err.Error())
}

func errorCode(err error) string { return toWire(err).Code }

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// printUsage prints the usage line and every visible command.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, usageLine)
	var lines []string
	for _, c := range commands {
		if !c.Hidden {
			lines = append(lines, "  "+strings.TrimPrefix(commandUsage(c), "usage: broodforge "))
		}
	}
	sort.Strings(lines)
	if len(lines) > 0 {
		fmt.Fprintln(w, "commands:")
		for _, l := range lines {
			fmt.Fprintln(w, l)
		}
	}
}

// commandUsage is one command's usage line.
func commandUsage(c *Command) string {
	var b strings.Builder
	b.WriteString("usage: broodforge " + c.Name)
	for _, f := range c.Flags {
		if f.Kind == FlagBool {
			b.WriteString(" [--" + f.Name + "]")
		} else {
			b.WriteString(" [--" + f.Name + " " + f.Value + "]")
		}
	}
	if c.Expect {
		b.WriteString(" [--expect <value>]")
	}
	if c.List {
		b.WriteString(" [--limit <n>|--all]")
	}
	return b.String()
}
