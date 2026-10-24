// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"errors"
	"regexp"
	"strings"
)

// CustomIDKind is what a button or a header menu does (03-architecture.md, Data flow: Custom ids).
type CustomIDKind int

const (
	// IDAnswer is a permission answer: `bf:<session id>:<tool_use_id>:<allow|deny|allow-session>`.
	IDAnswer CustomIDKind = iota
	// IDStop is the working message's Stop: `bf:<session id>:stop`.
	IDStop
	// IDEnd is the header's End session: `bf:<session id>:end`.
	IDEnd
	// IDStart is the panel's Start session: `bf:<channel id>:start`.
	IDStart
	// IDModel, IDEffort and IDMode are the header's select menus (P12b): `bf:<session
	// id>:model|effort|mode`; the pick is the interaction's value, never in the id.
	IDModel
	IDEffort
	IDMode
)

// CustomID is one button's or menu's custom id, parsed by the one function 03 names
// (ParseCustomID).
type CustomID struct {
	Kind      CustomIDKind
	Session   string // IDAnswer, IDStop, IDEnd, IDModel, IDEffort, IDMode
	Channel   string // IDStart
	ToolUseID string // IDAnswer
	Decision  string // IDAnswer: allow, deny or allow-session
}

// The answers a permission prompt's buttons carry (04, Permission prompts).
const (
	DecisionAllow        = "allow"
	DecisionDeny         = "deny"
	DecisionAllowSession = "allow-session"
)

var (
	sessionIDRE = regexp.MustCompile(`^s[0-9]{4,}$`)
	channelIDRE = regexp.MustCompile(`^[0-9]{1,20}$`)
	toolUseRE   = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)
)

// sessionKinds are the three-part forms `bf:<session id>:<word>`.
var sessionKinds = map[string]CustomIDKind{"stop": IDStop, "end": IDEnd, "model": IDModel, "effort": IDEffort, "mode": IDMode}

// ErrBadCustomID is a custom id that is not one of 03's forms.
var ErrBadCustomID = errors.New("not a Broodforge custom id")

// ParseCustomID reads a button's or menu's custom id; anything not in 03's forms is
// ErrBadCustomID.
func ParseCustomID(s string) (CustomID, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 3 || parts[0] != "bf" {
		return CustomID{}, ErrBadCustomID
	}
	if len(parts) == 4 {
		id := CustomID{Kind: IDAnswer, Session: parts[1], ToolUseID: parts[2], Decision: parts[3]}
		switch id.Decision {
		case DecisionAllow, DecisionDeny, DecisionAllowSession:
		default:
			return CustomID{}, ErrBadCustomID
		}
		if !sessionIDRE.MatchString(id.Session) || !toolUseRE.MatchString(id.ToolUseID) {
			return CustomID{}, ErrBadCustomID
		}
		return id, nil
	}
	if len(parts) != 3 {
		return CustomID{}, ErrBadCustomID
	}
	if kind, ok := sessionKinds[parts[2]]; ok {
		if !sessionIDRE.MatchString(parts[1]) {
			return CustomID{}, ErrBadCustomID
		}
		return CustomID{Kind: kind, Session: parts[1]}, nil
	}
	if parts[2] != "start" || !channelIDRE.MatchString(parts[1]) {
		return CustomID{}, ErrBadCustomID
	}
	return CustomID{Kind: IDStart, Channel: parts[1]}, nil
}

// String is the custom id as a button or a menu carries it.
func (c CustomID) String() string {
	switch c.Kind {
	case IDAnswer:
		return "bf:" + c.Session + ":" + c.ToolUseID + ":" + c.Decision
	case IDStop:
		return "bf:" + c.Session + ":stop"
	case IDEnd:
		return "bf:" + c.Session + ":end"
	case IDStart:
		return "bf:" + c.Channel + ":start"
	case IDModel:
		return "bf:" + c.Session + ":model"
	case IDEffort:
		return "bf:" + c.Session + ":effort"
	case IDMode:
		return "bf:" + c.Session + ":mode"
	}
	return ""
}

// IsMenu reports whether the id is one of the header's select menus, whose interaction carries
// the one value picked.
func (c CustomID) IsMenu() bool {
	switch c.Kind {
	case IDModel, IDEffort, IDMode:
		return true
	case IDAnswer, IDStop, IDEnd, IDStart:
	}
	return false
}
