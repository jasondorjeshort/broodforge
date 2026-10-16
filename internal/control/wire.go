// SPDX-License-Identifier: GPL-3.0-or-later

package control

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Request is one request line: {"id":<n>,"cmd":"<group>.<command>","args":{...}}.
type Request struct {
	ID   int64           `json:"id"`
	Cmd  string          `json:"cmd"`
	Args json.RawMessage `json:"args,omitempty"`
}

// Error is the wire error object, {"code":"<kebab-case>","message":"<one line>"}. It is also
// the error type handlers return to choose their code; any other error becomes "internal".
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// NewError builds an *Error whose message is folded onto one line.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: OneLine(message)}
}

// Error codes this package produces itself.
const (
	CodeBadRequest     = "bad-request"
	CodeUnknownCommand = "unknown-command"
	CodeInternal       = "internal"
	CodeNotRunning     = "not-running"
	CodeConnectionLost = "connection-lost"
	CodeConnectFailed  = "connect-failed"
	CodeStartControl   = "start-control"
	CodeBadArgs        = "bad-args"
)

var codeRE = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// ValidCode reports whether code is one kebab-case token (05-cli-conventions.md, Output).
func ValidCode(code string) bool { return codeRE.MatchString(code) }

// OneLine folds a message onto one line: CR and LF become spaces.
func OneLine(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r", " ", "\n", " ").Replace(s)), " ")
}

// Line is one line the server sends: an event, a final result or a final error. Kind tells
// which; exactly one of Event, Result (with OK true) or Error (with OK false) is set.
type Line struct {
	ID     int64           `json:"id"`
	OK     *bool           `json:"ok,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
	Event  json.RawMessage `json:"event,omitempty"`
}

// LineKind is what a Line carries.
type LineKind int

const (
	// LineInvalid is a line that matches none of the shapes below.
	LineInvalid LineKind = iota
	// LineEvent is {"id":n,"event":{...}}, sent before the final response of a streaming command.
	LineEvent
	// LineResult is {"id":n,"ok":true,"result":v}.
	LineResult
	// LineError is {"id":n,"ok":false,"error":{...}}.
	LineError
)

// Kind classifies the line.
func (l *Line) Kind() LineKind {
	switch {
	case l.OK == nil && l.Event != nil && l.Error == nil && l.Result == nil:
		return LineEvent
	case l.OK != nil && *l.OK && l.Error == nil && l.Event == nil:
		return LineResult
	case l.OK != nil && !*l.OK && l.Error != nil && l.Result == nil && l.Event == nil:
		return LineError
	}
	return LineInvalid
}

func resultLine(id int64, result json.RawMessage) Line {
	ok := true
	if result == nil {
		result = json.RawMessage("null")
	}
	return Line{ID: id, OK: &ok, Result: result}
}

func errorLine(id int64, e *Error) Line {
	ok := false
	return Line{ID: id, OK: &ok, Error: e}
}

func eventLine(id int64, event json.RawMessage) Line {
	return Line{ID: id, Event: event}
}

// encodeLine renders a line with its terminating newline.
func encodeLine(l Line) ([]byte, error) {
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// maxLine bounds one line in either direction.
const maxLine = 8 << 20
