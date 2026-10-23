// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"path"
	"strings"
)

// A hook's command is read as the children's shell reads it (Git Bash on the PC, /bin/sh on the
// droplet, both POSIX): blanks separate words; '...' is literal; "..." keeps \ before $ ` " \ and
// a newline as an escape; \ outside quotes escapes the next character. ${CLAUDE_PROJECT_DIR},
// ${CLAUDE_PROJECT_DIR<op><word>} and $CLAUDE_PROJECT_DIR outside single quotes read as the copy,
// which is what the variable holds when a child runs (ADR 0025 rule 2). No other expansion.

// projectVar is the variable the engine sets for a hook and the chain sets to the copy.
const projectVar = "CLAUDE_PROJECT_DIR"

// word is one shell word: Raw with quotes removed and the variable as written, Expanded with
// the variable read as the copy.
type word struct {
	Raw      string
	Expanded string
}

// splitWords splits command into its words, reading projectVar as dir.
func splitWords(command, dir string) []word {
	var out []word
	var raw, exp strings.Builder
	in := false
	end := func() {
		if in {
			out = append(out, word{raw.String(), exp.String()})
		}
		raw.Reset()
		exp.Reset()
		in = false
	}
	lit := func(s string) { raw.WriteString(s); exp.WriteString(s); in = true }
	// variable reads a projectVar reference at command[i] ('$'), returning its length or 0.
	variable := func(i int) int {
		rest := command[i+1:]
		if strings.HasPrefix(rest, "{"+projectVar) {
			after := rest[1+len(projectVar):]
			if strings.HasPrefix(after, "}") {
				return 2 + len(projectVar) + 1
			}
			for _, op := range []string{":-", ":=", ":+", ":?", "-", "=", "+", "?"} {
				if strings.HasPrefix(after, op) {
					if shut := strings.IndexByte(after, '}'); shut >= 0 {
						return 2 + len(projectVar) + shut + 1
					}
				}
			}
			return 0
		}
		if strings.HasPrefix(rest, projectVar) {
			next := rest[len(projectVar):]
			if next == "" || !(next[0] == '_' || ('0' <= next[0] && next[0] <= '9') || ('a' <= next[0] && next[0] <= 'z') ||
				('A' <= next[0] && next[0] <= 'Z')) {
				return 1 + len(projectVar)
			}
		}
		return 0
	}
	expand := func(i int) int {
		if n := variable(i); n > 0 {
			raw.WriteString(command[i : i+n])
			exp.WriteString(dir)
			in = true
			return n
		}
		lit("$")
		return 1
	}
	for i := 0; i < len(command); {
		c := command[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n':
			end()
			i++
		case c == '\\':
			if i+1 < len(command) {
				if command[i+1] != '\n' {
					lit(command[i+1 : i+2])
				}
				i += 2
				continue
			}
			lit(`\`)
			i++
		case c == '\'':
			shut := strings.IndexByte(command[i+1:], '\'')
			if shut < 0 {
				lit(command[i+1:])
				i = len(command)
				continue
			}
			lit(command[i+1 : i+1+shut])
			i += shut + 2
		case c == '"':
			in = true
			i++
			for i < len(command) && command[i] != '"' {
				switch {
				case command[i] == '\\' && i+1 < len(command) && strings.IndexByte("$`\"\\\n", command[i+1]) >= 0:
					if command[i+1] != '\n' {
						lit(command[i+1 : i+2])
					}
					i += 2
				case command[i] == '$':
					i += expand(i)
				default:
					lit(command[i : i+1])
					i++
				}
			}
			i++ // the closing quote
		case c == '$':
			i += expand(i)
		default:
			lit(command[i : i+1])
			i++
		}
	}
	end()
	return out
}

// scriptSuffixes are the endings that make a word name a file (the companion's probe).
var scriptSuffixes = []string{".js", ".py", ".sh", ".ps1", ".exe"}

func suffix(w string) string {
	low := strings.ToLower(w)
	for _, s := range scriptSuffixes {
		if strings.HasSuffix(low, s) {
			return s
		}
	}
	return ""
}

// namesFile reports whether a word names a file: a path separator or a script suffix.
func namesFile(w string) bool { return strings.ContainsAny(w, `/\`) || suffix(w) != "" }

// hookName is a hooks.json entry's name before a repeat's suffix: the base name of the first
// word that names a file, read with projectVar as dir, else the first word.
func hookName(command, dir string) string {
	ws := splitWords(command, dir)
	for _, w := range ws {
		if namesFile(w.Expanded) {
			return path.Base(strings.ReplaceAll(w.Expanded, `\`, "/"))
		}
	}
	if len(ws) > 0 && ws[0].Expanded != "" {
		return ws[0].Expanded
	}
	return "hook"
}
