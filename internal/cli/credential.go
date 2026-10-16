// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
)

// The credential commands (05-debug-interface.md, Credentials). init is (file); list and show
// are reads, answered by the bot or, without one, from the store file; add, remove and login
// are writes the bot serves (CredentialHandlers). No command prints a stored value: init
// prints the new key once, and nothing else prints the key or any value.

// credentialInit is `broodforge credential init (file)`: writes the key file the instance
// names (BROODFORGE_KEY_FILE overrides it except on a scratch root) and prints `<key hex>
// <key file>`, the one time the key is printed. An existing file is key-exists, nothing written.
var credentialInit = Command{
	Name:    "credential init",
	Class:   ClassFile,
	Columns: initColumns,
	Local:   runCredentialInit,
}

// credentialAdd is `broodforge credential add --kind api-key|subscription --user <user id>
// --label <text>`: an API key's value is read from stdin, never an argument; prints the new id.
var credentialAdd = Command{
	Name:  "credential add",
	Class: ClassWrite,
	Flags: []Flag{
		{Name: "kind", Kind: FlagString, Value: "api-key|subscription", Usage: "the credential kind"},
		{Name: "user", Kind: FlagString, Value: "<user id>", Usage: "the registered user it belongs to"},
		{Name: "label", Kind: FlagString, Value: "<text>", Usage: "a label for lists"},
	},
	Columns: []string{"id"},
}

// credentialList is `broodforge credential list [--user <id>]`: `<id>  <kind>  <user id>
// <label>  <status>  <last used or ->`.
var credentialList = Command{
	Name:  "credential list",
	Class: ClassRead,
	List:  true,
	Flags: []Flag{
		{Name: "user", Kind: FlagString, Value: "<id>", Usage: "one user's credentials"},
	},
	Columns: recordColumns,
	Local:   runCredentialList,
}

// credentialShow is `broodforge credential show <id>`: the list line; --json the whole record,
// which holds no store value.
var credentialShow = Command{
	Name:    "credential show",
	Class:   ClassRead,
	Flags:   []Flag{{Name: "id", Kind: FlagString, Value: "<id>"}},
	Columns: recordColumns,
	Local:   runCredentialShow,
}

// credentialRemove is `broodforge credential remove <id>`: marks it removed and deletes its
// value; prints the list line.
var credentialRemove = Command{
	Name:    "credential remove",
	Class:   ClassWrite,
	Flags:   []Flag{{Name: "id", Kind: FlagString, Value: "<id>"}},
	Columns: recordColumns,
}

// credentialLogin is `broodforge credential login <id> [--done]`: prints the command the user
// runs to log the engine into the subscription's config directory; --done records it usable
// and prints the list line. --relay is P12's.
var credentialLogin = Command{
	Name:  "credential login",
	Class: ClassWrite,
	Flags: []Flag{
		{Name: "done", Kind: FlagBool, Usage: "the user has logged in: record the credential usable"},
		{Name: "id", Kind: FlagString, Value: "<id>"},
	},
	Columns: recordColumns,
}

func init() {
	commands = append(commands, &credentialInit, &credentialAdd, &credentialList, &credentialShow,
		&credentialRemove, &credentialLogin)
	positionals[&credentialShow] = []string{"id"}
	positionals[&credentialRemove] = []string{"id"}
	positionals[&credentialLogin] = []string{"id"}
	extraArgs[&credentialAdd] = readCredentialValue
}

var initColumns = []string{"key", "keyFile"}

var recordColumns = []string{"id", "kind", "user", "label", "status", "lastUsedAt"}

// credentialRecord is a credential's one JSON form.
type credentialRecord struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	Provider   string  `json:"provider"`
	User       *string `json:"user"`
	Label      string  `json:"label"`
	Status     string  `json:"status"`
	AddedAt    string  `json:"addedAt"`
	LastUsedAt *string `json:"lastUsedAt"`
	StoreRef   string  `json:"storeRef"`
}

func toRecord(r credentials.Record) credentialRecord {
	opt := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	return credentialRecord{ID: r.ID, Kind: r.Kind, Provider: r.Provider, User: opt(r.User), Label: r.Label,
		Status: r.Status, AddedAt: r.AddedAt, LastUsedAt: opt(r.LastUsedAt), StoreRef: r.StoreRef}
}

type initResult struct {
	Key     string `json:"key"`
	KeyFile string `json:"keyFile"`
}

// loginResult is credential login's result: the command as one string.
type loginResult = string

// readCredentialValue reads an API key from stdin for the request (never a flag); one trailing
// newline is dropped, so `echo` and `printf` give the same value.
func readCredentialValue(env *Env) (map[string]string, error) {
	if env.String("kind") != credentials.KindAPIKey {
		return nil, nil
	}
	b, err := io.ReadAll(io.LimitReader(env.Stdin, credentials.MaxValue+1))
	if err != nil {
		return nil, &usageError{"credential add: cannot read stdin: " + err.Error()}
	}
	b = bytes.TrimSuffix(bytes.TrimSuffix(b, []byte("\n")), []byte("\r"))
	if len(b) == 0 || len(b) > credentials.MaxValue {
		return nil, &usageError{"credential add: --kind api-key reads its value from stdin, 1 to 65536 bytes"}
	}
	return map[string]string{"value": string(b)}, nil
}

func runCredentialInit(_ context.Context, env *Env) error {
	c, err := config.Load(env.Root, credentialExists)
	if err != nil {
		return configFailure(env, err)
	}
	path, err := credentials.KeyPath(env.Root, c.Instance.InstanceName, c.Instance.Credentials.KeyFile, os.LookupEnv)
	if err != nil {
		return credentialError(err)
	}
	key, err := credentials.Init(path)
	if err != nil {
		return credentialError(err)
	}
	out, err := encodeAny(initResult{key, path})
	if err != nil {
		return err
	}
	return env.Out.Result(out, initColumns)
}

func runCredentialList(_ context.Context, env *Env) error {
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	recs, err := credentials.List(r, env.String("user"))
	if err != nil {
		return credentialError(err)
	}
	rows := make([]credentialRecord, len(recs))
	for i, rec := range recs {
		rows[i] = toRecord(rec)
	}
	out, err := encodeAny(rows)
	if err != nil {
		return err
	}
	return env.Out.Result(out, recordColumns)
}

func runCredentialShow(_ context.Context, env *Env) error {
	r, err := openStore(env)
	if err != nil {
		return err
	}
	defer r.Close()
	rec, err := credentials.Show(r, env.String("id"))
	if err != nil {
		return credentialError(err)
	}
	out, err := encodeAny(toRecord(rec))
	if err != nil {
		return err
	}
	return env.Out.Result(out, recordColumns)
}

// credentialError maps a credentials or store refusal to the CLI's error line.
func credentialError(err error) error {
	if e, ok := credentials.IsRefusal(err); ok {
		return control.NewError(e.Code, e.Message)
	}
	return localError(err)
}

// CredentialHandlers registers the bot side of the credential write and read commands on a
// control registry: `serve` (P10) and the tests' harness call it.
func CredentialHandlers(svc *credentials.Service) func(*control.Registry) {
	return func(reg *control.Registry) {
		reg.Handle("credential.add", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct{ Kind, User, Label, Value string }
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			rec, err := svc.Add(a.Kind, a.User, a.Label, []byte(a.Value))
			return recordResult(rec, err)
		})
		reg.Handle("credential.list", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
			var a struct{ User string }
			if err := json.Unmarshal(call.Args, &a); err != nil {
				return nil, control.NewError(control.CodeBadArgs, err.Error())
			}
			recs, err := svc.List(a.User)
			if err != nil {
				return nil, credentialError(err)
			}
			rows := make([]credentialRecord, len(recs))
			for i, rec := range recs {
				rows[i] = toRecord(rec)
			}
			return encodeAny(rows)
		})
		byID := func(f func(id string, done bool) (json.RawMessage, error)) control.Handler {
			return func(_ context.Context, call *control.Call) (json.RawMessage, error) {
				var a struct {
					ID   string
					Done bool
				}
				if err := json.Unmarshal(call.Args, &a); err != nil {
					return nil, control.NewError(control.CodeBadArgs, err.Error())
				}
				return f(a.ID, a.Done)
			}
		}
		reg.Handle("credential.show", byID(func(id string, _ bool) (json.RawMessage, error) {
			return recordResult(svc.Show(id))
		}))
		reg.Handle("credential.remove", byID(func(id string, _ bool) (json.RawMessage, error) {
			return recordResult(svc.Remove(id))
		}))
		reg.Handle("credential.login", byID(func(id string, done bool) (json.RawMessage, error) {
			if done {
				return recordResult(svc.LoginDone(id))
			}
			cmd, err := svc.LoginCommand(id)
			if err != nil {
				return nil, credentialError(err)
			}
			return encodeAny(loginResult(cmd))
		}))
	}
}

func recordResult(rec credentials.Record, err error) (json.RawMessage, error) {
	if err != nil {
		var ce *control.Error
		if errors.As(err, &ce) {
			return nil, ce
		}
		return nil, credentialError(err)
	}
	return encodeAny(toRecord(rec))
}
