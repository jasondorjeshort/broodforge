// SPDX-License-Identifier: GPL-3.0-or-later

package control

import (
	"context"
	"encoding/json"
	"strconv"
)

// CmdEcho is the hidden probe every server answers: `broodforge control echo`.
const CmdEcho = "control.echo"

// MaxEchoEvents bounds control.echo's events argument.
const MaxEchoEvents = 1000

// EchoArgs are control.echo's args: text (default "ok") comes back as the result, events
// {"seq":i,"text":text} lines are streamed first (0 to MaxEchoEvents), and a non-empty fail
// makes the call refuse with that code instead, after the events.
type EchoArgs struct {
	Text   *string `json:"text"`
	Events int     `json:"events"`
	Fail   string  `json:"fail"`
}

// EchoEvent is one streamed control.echo event.
type EchoEvent struct {
	Seq  int    `json:"seq"`
	Text string `json:"text"`
}

// EchoResult is control.echo's result.
type EchoResult struct {
	Text string `json:"text"`
}

func echo(ctx context.Context, call *Call) (json.RawMessage, error) {
	var a EchoArgs
	if err := json.Unmarshal(call.Args, &a); err != nil {
		return nil, NewError(CodeBadArgs, err.Error())
	}
	text := "ok"
	if a.Text != nil {
		text = *a.Text
	}
	if a.Events < 0 || a.Events > MaxEchoEvents {
		return nil, NewError(CodeBadArgs, "events must be 0 to "+strconv.Itoa(MaxEchoEvents))
	}
	if a.Fail != "" && !ValidCode(a.Fail) {
		return nil, NewError(CodeBadArgs, "fail must be one kebab-case code, got "+strconv.Quote(a.Fail))
	}
	for i := 1; i <= a.Events; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ev, err := json.Marshal(EchoEvent{Seq: i, Text: text})
		if err != nil {
			return nil, err
		}
		if err := call.Emit(ev); err != nil {
			return nil, err
		}
	}
	if a.Fail != "" {
		return nil, NewError(a.Fail, "echo was asked to fail")
	}
	return json.Marshal(EchoResult{Text: text})
}
