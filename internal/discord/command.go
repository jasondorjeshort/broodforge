// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"errors"
	"regexp"
	"strings"
)

// SlashKind is one of 04's slash commands.
type SlashKind int

const (
	SlashRegister SlashKind = iota
	SlashSessionStart
	SlashSessionStop
	SlashSessionEnd
	SlashSessionMode
	SlashSessionInvite
	SlashCredential
	SlashUsage
)

// Slash is a parsed slash command: its kind, the words after the command name (a mode, a
// credential subcommand) and its named options (`name:<text>`, which may hold spaces).
type Slash struct {
	Kind    SlashKind
	Name    string // as typed: "session mode"
	Args    []string
	Options map[string]string
}

// ErrNotSlash is a message that is not a slash command.
var ErrNotSlash = errors.New("not a slash command")

// slashNames are 04's commands, the longest first so `session start` wins over `session`.
var slashNames = []struct {
	name string
	kind SlashKind
	opts []string // the options it takes; any other is refused
}{
	{"session start", SlashSessionStart, nil},
	{"session stop", SlashSessionStop, nil},
	{"session end", SlashSessionEnd, nil},
	{"session mode", SlashSessionMode, nil},
	{"session invite", SlashSessionInvite, []string{"user"}},
	{"credential", SlashCredential, []string{"kind", "label", "id"}},
	{"register", SlashRegister, []string{"name", "email"}},
	{"usage", SlashUsage, nil},
}

var optionRE = regexp.MustCompile(`^([a-z][a-z-]*):(.*)$`)

// ParseSlash reads a message typed as a slash command (`/register name:Jason Dorje Short
// email:jdorje@gmail.com`), the form `discord inject message` takes in place of Discord's
// interaction. A text without a leading `/` is ErrNotSlash; an unknown command or option is an
// error naming it.
func ParseSlash(text string) (Slash, error) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return Slash{}, ErrNotSlash
	}
	words := strings.Fields(text[1:])
	for _, c := range slashNames {
		n := len(strings.Fields(c.name))
		if len(words) < n || strings.Join(words[:n], " ") != c.name {
			continue
		}
		s := Slash{Kind: c.kind, Name: c.name, Options: map[string]string{}}
		key := ""
		for _, w := range words[n:] {
			if m := optionRE.FindStringSubmatch(w); m != nil && !strings.HasPrefix(m[2], "/") {
				if !contains(c.opts, m[1]) {
					return Slash{}, errors.New("/" + c.name + " has no option " + m[1])
				}
				key = m[1]
				s.Options[key] = m[2]
				continue
			}
			if key == "" {
				s.Args = append(s.Args, w)
				continue
			}
			s.Options[key] = strings.TrimSpace(s.Options[key] + " " + w)
		}
		return s, nil
	}
	name := ""
	if len(words) > 0 {
		name = words[0]
	}
	return Slash{}, errors.New("no command /" + name)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// mentionRE is a user mention as Discord writes it, `<@1002>` or `<@!1002>`.
var mentionRE = regexp.MustCompile(`^<@!?([0-9]{1,20})>$`)

// UserOption reads a user option: a mention or a bare id.
func UserOption(s string) (string, bool) {
	if m := mentionRE.FindStringSubmatch(s); m != nil {
		return m[1], true
	}
	return s, channelIDRE.MatchString(s)
}
