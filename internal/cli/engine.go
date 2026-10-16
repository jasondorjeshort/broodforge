// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/engine"
)

// Error codes of engine run.
const (
	CodeEngineExit = "engine-exit" // the turn's exit code was not 0
	CodeBadFixture = "bad-fixture" // the fixture directory or its fixture.json cannot be read
	CodeNoTurn     = "no-turn"     // the fixture has no turn-<n>.jsonl
	// CodeBadReceivedDir: received-<n>.json cannot be written there (missing, not a directory, read-only).
	CodeBadReceivedDir = "bad-received-dir"
)

// Outcome tokens of engine run, compared by its --expect.
const (
	tokenResult   = "result"
	tokenNoResult = "no-result"
)

// engineRun is the adapter alone, no store, no session (05-debug-interface.md, A turn against
// the fake engine): `broodforge engine run --fake <fixture dir> [--turn <n>] [--cwd <dir>]
// [--received-dir <dir>] --prompt <text> [--expect <value>]`. It streams the turn's
// EngineEvents as JSONL on stdout, in text mode too, and exits with the turn's exit code (any
// non-zero code is 1, `error: engine-exit: ...`). Its outcome token for --expect is `result`
// when the turn produced a result event and `no-result` otherwise; a non-zero exit is the
// refusal engine-exit. --real is P7's.
var engineRun = Command{
	Name:  "engine run",
	Class: ClassStandalone,
	Flags: []Flag{
		{Name: "fake", Kind: FlagString, Value: "<fixture dir>", Usage: "replay this fixture directory"},
		{Name: "turn", Kind: FlagInt, Default: "1", Value: "<n>", Usage: "the turn-<n>.jsonl to play"},
		{Name: "cwd", Kind: FlagString, Value: "<dir>", Usage: "the request's working directory"},
		{Name: "received-dir", Kind: FlagString, Value: "<dir>", Usage: "where received-<n>.json goes (default: the fixture directory)"},
		{Name: "prompt", Kind: FlagString, Value: "<text>", Usage: "the turn's input"},
	},
	Expect: true,
	Local:  runEngine,
}

func init() { commands = append(commands, &engineRun) }

func runEngine(_ context.Context, env *Env) error {
	dir := env.String("fake")
	if dir == "" {
		return &usageError{"engine run: --fake <fixture dir> is required"}
	}
	if env.String("prompt") == "" {
		return &usageError{"engine run: --prompt <text> is required"}
	}
	n := env.Int("turn")
	if n < 1 {
		return &usageError{fmt.Sprintf("engine run: --turn %d: turns count from 1", n)}
	}
	fake, err := engine.NewFake(dir, env.String("received-dir"))
	if errors.Is(err, engine.ErrReceivedDir) {
		return control.NewError(CodeBadReceivedDir, err.Error())
	}
	if err != nil {
		return control.NewError(CodeBadFixture, err.Error())
	}
	fake.SetNextTurn(n)
	h, err := fake.StartTurn(engine.TurnRequest{Cwd: env.String("cwd"), Input: env.String("prompt")})
	if errors.Is(err, engine.ErrNoTurn) {
		return control.NewError(CodeNoTurn, err.Error())
	}
	if errors.Is(err, engine.ErrReceivedDir) {
		return control.NewError(CodeBadReceivedDir, err.Error())
	}
	if err != nil {
		return err
	}
	token := tokenNoResult
	var exit *engine.ExitEvent
	var printErr error
	for ev := range h.Events {
		switch ev.Kind {
		case engine.KindResult:
			token = tokenResult
		case engine.KindExit:
			exit = ev.Exit
		case engine.KindInit, engine.KindAssistant, engine.KindToolResult, engine.KindUnknown:
		}
		if printErr != nil {
			continue // drain the turn
		}
		line, err := ev.MarshalJSON() // not json.Marshal, which would escape HTML again
		if err == nil {
			err = env.Out.Event(line, nil)
		}
		printErr = err
	}
	if printErr != nil {
		return printErr
	}
	env.Out.SetToken(token)
	switch {
	case exit == nil:
		return control.NewError(CodeEngineExit, "the turn ended without an exit event")
	case exit.Interrupted:
		return control.NewError(CodeEngineExit, "the turn was interrupted")
	case exit.Code == nil:
		return control.NewError(CodeEngineExit, "the engine exited without a code")
	case *exit.Code != 0:
		return control.NewError(CodeEngineExit, fmt.Sprintf("the engine exited %d", *exit.Code))
	}
	return nil
}
