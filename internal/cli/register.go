// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/session"
)

// The registration commands (05-debug-interface-session.md, Registration and participants).
// `register <user id>` and `register set-default` are writes the bot serves; `register list`
// and `register show` read the store, through the bot's file or without one. Each prints
// `<user id>  <name>  <email>  <registered at>  <default credential id or ->`.

// registerCmd is `broodforge register <discord user id> --name "<git name>" --email <email>`:
// registering again replaces the pair and writes registration-changed (R-02).
var registerCmd = Command{
	Name:  "register",
	Class: ClassWrite,
	Flags: []Flag{
		{Name: "name", Kind: FlagString, Value: "<git name>", Usage: "the name commits are authored with"},
		{Name: "email", Kind: FlagString, Value: "<email>", Usage: "the email commits are authored with"},
		{Name: "user", Kind: FlagString, Value: "<user id>"},
	},
	Columns: session.RegistrationColumns,
}

// registerList is `broodforge register list`.
var registerList = Command{
	Name:    "register list",
	Class:   ClassRead,
	List:    true,
	Columns: session.RegistrationColumns,
	Local:   runRegisterList,
}

// registerShow is `broodforge register show <user id>`; an unknown id is unregistered.
var registerShow = Command{
	Name:    "register show",
	Class:   ClassRead,
	Flags:   []Flag{{Name: "user", Kind: FlagString, Value: "<user id>"}},
	Columns: session.RegistrationColumns,
	Local:   runRegisterShow,
}

// registerSetDefault is `broodforge register set-default <user id> <credential id>`.
var registerSetDefault = Command{
	Name:  "register set-default",
	Class: ClassWrite,
	Flags: []Flag{
		{Name: "user", Kind: FlagString, Value: "<user id>"},
		{Name: "credential", Kind: FlagString, Value: "<credential id>"},
	},
	Columns: session.RegistrationColumns,
}

func init() {
	commands = append(commands, &registerCmd, &registerList, &registerShow, &registerSetDefault)
	positionals[&registerCmd] = []string{"user"}
	positionals[&registerShow] = []string{"user"}
	positionals[&registerSetDefault] = []string{"user", "credential"}
}

func runRegisterList(_ context.Context, env *Env) error {
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	regs, err := session.Registrations(r)
	if err != nil {
		return err
	}
	out, err := encodeSession(regs)
	if err != nil {
		return err
	}
	return env.Out.Result(out, session.RegistrationColumns)
}

func runRegisterShow(_ context.Context, env *Env) error {
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	reg, err := session.ShowRegistration(r, env.String("user"))
	if err != nil {
		return sessionError(err)
	}
	out, err := encodeSession(reg)
	if err != nil {
		return err
	}
	return env.Out.Result(out, session.RegistrationColumns)
}

// registerHandlers registers register and register.set-default on the bot.
func registerHandlers(svc *session.Service, reg *control.Registry) {
	reg.Handle("register", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
		var a struct{ User, Name, Email string }
		if err := json.Unmarshal(call.Args, &a); err != nil {
			return nil, control.NewError(control.CodeBadArgs, err.Error())
		}
		r, err := svc.Register(a.User, a.Name, a.Email, "")
		if err != nil {
			return nil, sessionError(err)
		}
		return encodeSession(r)
	})
	reg.Handle("register.set-default", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
		var a struct{ User, Credential string }
		if err := json.Unmarshal(call.Args, &a); err != nil {
			return nil, control.NewError(control.CodeBadArgs, err.Error())
		}
		r, err := svc.SetDefault(a.User, a.Credential, "")
		if err != nil {
			return nil, sessionError(err)
		}
		return encodeSession(r)
	})
}
