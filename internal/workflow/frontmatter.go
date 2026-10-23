// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"strconv"
	"strings"
)

// The frontmatter reader for agent files: the YAML subset they use, read by hand (no YAML
// dependency, ADR 0023). A field is `key: value` at the left margin; a value in double or single
// quotes is unquoted; `|` or `>` takes the indented lines after it as a block (kept or folded);
// `[a, b]` or `- item` lines are a list; an indented line after a plain value continues it. A
// value is never split at ": ", so `description: Runs tests: fast` reads whole, as agent files
// are written. Anything else is ignored.

// field is one frontmatter value: a scalar or a list.
type field struct {
	s      string
	list   []string
	isList bool
}

// str is a scalar field's value, a list's items joined by ", ".
func (f field) str() string {
	if f.isList {
		return strings.Join(f.list, ", ")
	}
	return f.s
}

// items is a list field's items, or a scalar's comma-separated items (`tools: Read, Grep`).
func (f field) items() []string {
	if f.isList {
		return f.list
	}
	var out []string
	for _, p := range strings.Split(f.s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// frontmatter splits text into its frontmatter fields and its body; ok is false when text does
// not open with a `---` line closed by another.
func frontmatter(text string) (map[string]field, string, bool) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return nil, "", false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if t := strings.TrimRight(lines[i], " \t"); t == "---" || t == "..." {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, "", false
	}
	body := strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	return fields(lines[1:end]), body, true
}

func indented(l string) bool { return l != "" && (l[0] == ' ' || l[0] == '\t') }

func fields(lines []string) map[string]field {
	out := map[string]field{}
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if indented(l) || strings.TrimSpace(l) == "" || strings.HasPrefix(l, "#") {
			continue
		}
		key, rest, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		key, rest = strings.TrimSpace(key), stripComment(rest)
		// The lines that belong to this key: indented, blank, or a "- " item.
		j := i + 1
		for j < len(lines) && (indented(lines[j]) || strings.TrimSpace(lines[j]) == "" ||
			strings.HasPrefix(lines[j], "- ") || lines[j] == "-") {
			j++
		}
		more := lines[i+1 : j]
		i = j - 1
		switch {
		case rest == "" && hasItem(more):
			var items []string
			for _, m := range more {
				if t := strings.TrimSpace(m); strings.HasPrefix(t, "-") {
					items = append(items, unquote(stripComment(strings.TrimPrefix(t, "-"))))
				}
			}
			out[key] = field{list: items, isList: true}
		case strings.HasPrefix(rest, "|") || strings.HasPrefix(rest, ">"):
			out[key] = field{s: block(more, rest[0] == '>')}
		case strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]"):
			var items []string
			for _, p := range strings.Split(rest[1:len(rest)-1], ",") {
				if p = unquote(strings.TrimSpace(p)); p != "" {
					items = append(items, p)
				}
			}
			out[key] = field{list: items, isList: true}
		default:
			parts := []string{rest}
			for _, m := range more {
				if t := stripComment(m); t != "" {
					parts = append(parts, t)
				}
			}
			out[key] = field{s: unquote(strings.TrimSpace(strings.Join(parts, " ")))}
		}
	}
	return out
}

// stripComment is a value without its YAML trailing comment (a # at the start or after a blank,
// outside quotes) and the blanks around it. A block scalar's own lines keep theirs, as in YAML.
func stripComment(s string) string {
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t'):
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

func hasItem(lines []string) bool {
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "-") {
			return true
		}
	}
	return false
}

// block is a block scalar's lines without their common indentation: kept as lines (|) or folded
// into one line per paragraph (>), trailing blank lines dropped.
func block(lines []string, fold bool) string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	cut := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if cut < 0 || n < cut {
			cut = n
		}
	}
	trimmed := make([]string, len(lines))
	for k, l := range lines {
		if len(l) >= cut && cut >= 0 {
			trimmed[k] = l[cut:]
		} else {
			trimmed[k] = strings.TrimSpace(l)
		}
	}
	if !fold {
		return strings.Join(trimmed, "\n")
	}
	var paras []string
	var cur []string
	for _, l := range trimmed {
		if strings.TrimSpace(l) == "" {
			paras = append(paras, strings.Join(cur, " "))
			cur = nil
			continue
		}
		cur = append(cur, l)
	}
	paras = append(paras, strings.Join(cur, " "))
	return strings.Join(paras, "\n")
}

// unquote reads a double-quoted value with Go's (JSON-like) escapes, a single-quoted one with
// ” as a quote, and returns anything else as it is.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
		return s[1 : len(s)-1]
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	}
	return s
}
