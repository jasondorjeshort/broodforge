// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// Class is how a command reaches the instance (05-cli-conventions.md, Surface).
type Class int

const (
	// ClassWrite commands are control-socket clients only; with no socket they exit 1 with
	// `error: not-running: no bot at <socket>`.
	ClassWrite Class = iota
	// ClassRead commands ask the socket and, with no socket, run Local, which opens
	// data/store.sqlite read-only (ADR 0021).
	ClassRead
	// ClassFile commands, marked `(file)` in 05, run Local on the files under the root and
	// never touch the socket.
	ClassFile
	// ClassStandalone commands (serve, hook, engine-entry) run Local and need no root.
	ClassStandalone
)

// FlagKind is the type of a command flag and of its value in the request's args.
type FlagKind int

const (
	FlagString FlagKind = iota
	FlagInt
	FlagBool
)

// Flag is one command flag. Its args key is the name with dashes removed, camel-cased.
type Flag struct {
	Name    string
	Kind    FlagKind
	Default string // parsed per Kind; "" is the zero value
	Value   string // the placeholder in usage text, e.g. "<n>"
	Usage   string
}

// Command is one entry of the command tree.
type Command struct {
	// Name is the group and command, space-separated ("control echo"); its control-socket
	// cmd is the same with a dot ("control.echo").
	Name  string
	Class Class
	// Hidden commands work but are left out of the usage text.
	Hidden bool
	Flags  []Flag
	// Expect offers --expect <value>, compared with the outcome token.
	Expect bool
	// List offers --limit <n> and --all; lists stop at DefaultLimit rows otherwise.
	List bool
	// Columns are the result's keys in text order; EventColumns the streamed events'.
	Columns      []string
	EventColumns []string
	// Local runs the command without the socket (ClassRead fallback, ClassFile,
	// ClassStandalone). Socket classes leave it nil to mean "socket only".
	Local func(ctx context.Context, env *Env) error
}

// Env is what a running command sees.
type Env struct {
	Root   string
	JSON   bool
	Stdin  io.Reader
	Stderr io.Writer
	Out    *Printer
	values map[string]*flagValue
}

type flagValue struct {
	kind FlagKind
	s    string
	i    int
	b    bool
}

// String, Int and Bool return a command flag's value; a name the command does not declare
// returns the zero value.
func (e *Env) String(name string) string {
	if v := e.values[name]; v != nil {
		return v.s
	}
	return ""
}

func (e *Env) Int(name string) int {
	if v := e.values[name]; v != nil {
		return v.i
	}
	return 0
}

func (e *Env) Bool(name string) bool {
	if v := e.values[name]; v != nil {
		return v.b
	}
	return false
}

// wireCmd is the command's control-socket name.
func (c *Command) wireCmd() string { return strings.ReplaceAll(c.Name, " ", ".") }

// wireArgs renders every declared flag, set or default, as the request's args object.
func (c *Command) wireArgs(e *Env) (json.RawMessage, error) {
	args := map[string]json.RawMessage{}
	for _, f := range c.Flags {
		v := e.values[f.Name]
		var raw []byte
		var err error
		switch f.Kind {
		case FlagString:
			raw, err = json.Marshal(v.s)
		case FlagInt:
			raw = []byte(strconv.Itoa(v.i))
		case FlagBool:
			raw = []byte(strconv.FormatBool(v.b))
		}
		if err != nil {
			return nil, err
		}
		args[camel(f.Name)] = raw
	}
	return json.Marshal(args)
}

// camel turns a flag name into its args key: "dry-run" is "dryRun".
func camel(name string) string {
	parts := strings.Split(name, "-")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// commands is the command tree. A command file registers its commands from its own init(),
// as engine.go does; the table is not edited by a pass, so passes running in parallel never
// collide here.
var commands []*Command

// lookup finds the command named by the first one or two words of args and returns it with
// the number of words it took.
func lookup(args []string) (*Command, int) {
	if len(args) >= 2 {
		name := args[0] + " " + args[1]
		for _, c := range commands {
			if c.Name == name {
				return c, 2
			}
		}
	}
	if len(args) >= 1 {
		for _, c := range commands {
			if c.Name == args[0] {
				return c, 1
			}
		}
	}
	return nil, 0
}
