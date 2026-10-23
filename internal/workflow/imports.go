// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The @ import resolver, ported from the base's .claude/hooks/imports.js (ADR 0025 rule 5: CLAUDE.md
// with @ imports resolved as imports.js does), with the copy as the repository root:
//   - an @ counts at line start or after white space;
//   - never inside a fenced block (``` or ~~~, any indent; a ``` line whose info string holds a
//     backtick is not a fence) or a code span (also one crossing a line break in its paragraph);
//   - the token runs to the next white space, with trailing . , ; : ) stripped;
//   - it resolves from the importing file's directory;
//   - imports nest four hops: a file at hop 4 is loaded but its own imports are not.
// A target outside the root (~/..., an absolute path, a ../ climbing above it, a token holding a
// backslash, anything filepath.IsLocal refuses on this host) is never read. The backslash and
// IsLocal rules are stricter than imports.js, which reads a backslash as a separator on Windows
// (a carry-back idea of 2026-10-23); any other change to imports.js is a change here.

// importHops is imports.js's HOPS.
const importHops = 4

// span stands in for code-span text: not white space, so an @ after a span does not count.
const span = '\x00'

// importRef is one @ import: where it stands and what it reaches.
type importRef struct {
	file    string // the importing file, slash path under the root
	line    int    // 1-based
	token   string // without the @
	target  string // slash path under the root, or as written when outside
	outside bool
	exists  bool // a regular file
	isDir   bool
}

type importToken struct {
	line  int
	token string
}

// blankSpans replaces each code span (a run of n backticks up to the next run of exactly n) with
// span characters, keeping line breaks; an unmatched run stays literal.
func blankSpans(text string) string {
	var b strings.Builder
	i := 0
	for i < len(text) {
		if text[i] != '`' {
			b.WriteByte(text[i])
			i++
			continue
		}
		n := 0
		for i+n < len(text) && text[i+n] == '`' {
			n++
		}
		j, closeAt := i+n, -1
		for j < len(text) {
			if text[j] != '`' {
				j++
				continue
			}
			m := 0
			for j+m < len(text) && text[j+m] == '`' {
				m++
			}
			if m == n {
				closeAt = j
				break
			}
			j += m
		}
		if closeAt < 0 {
			b.WriteString(text[i : i+n])
			i += n
			continue
		}
		for _, r := range text[i : closeAt+n] {
			if r == '\n' {
				b.WriteByte('\n')
			} else {
				b.WriteRune(span)
			}
		}
		i = closeAt + n
	}
	return b.String()
}

var fenceRE = regexp.MustCompile("^[ \t]*(`{3,}|~{3,})(.*)$")

type fence struct {
	ch   byte
	n    int
	rest string
}

// fenceOf reads a fence line: ``` or ~~~ (three or more), any indent; a backtick fence's info
// string may not hold a backtick.
func fenceOf(raw string) *fence {
	m := fenceRE.FindStringSubmatch(raw)
	if m == nil || (m[1][0] == '`' && strings.Contains(m[2], "`")) {
		return nil
	}
	return &fence{m[1][0], len(m[1]), m[2]}
}

// isBlank is JavaScript's \s: Unicode white space and the byte-order mark.
func isBlank(r rune) bool { return unicode.IsSpace(r) || r == 0xfeff }

// parseImports is imports.js's parse: every @ token of text with its line.
func parseImports(text string) []importToken {
	var found []importToken
	rows := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	var open *fence
	type row struct {
		k   int
		raw string
	}
	var para []row
	flush := func() {
		if len(para) == 0 {
			return
		}
		raws := make([]string, len(para))
		for i, p := range para {
			raws[i] = p.raw
		}
		for i, line := range strings.Split(blankSpans(strings.Join(raws, "\n")), "\n") {
			for _, tok := range atTokens(line) {
				if cut := strings.IndexRune(tok, span); cut >= 0 {
					tok = tok[:cut]
				}
				tok = strings.TrimRight(tok, ".,;:)")
				if tok != "" {
					found = append(found, importToken{para[i].k + 1, tok})
				}
			}
		}
		para = nil
	}
	for k, raw := range rows {
		f := fenceOf(raw)
		if open != nil {
			if f != nil && f.ch == open.ch && f.n >= open.n && strings.TrimSpace(f.rest) == "" {
				open = nil
			}
			continue
		}
		if f != nil {
			flush()
			open = f
			continue
		}
		if strings.TrimSpace(raw) == "" {
			flush()
			continue
		}
		para = append(para, row{k, raw})
	}
	flush()
	return found
}

// atTokens is the /(^|\s)@(\S+)/g matches of one line: the run of non-blank characters after
// each @ that stands at the line start or after white space.
func atTokens(line string) []string {
	var out []string
	prevBlank := true
	for i := 0; i < len(line); {
		r, w := utf8.DecodeRuneInString(line[i:])
		if r == '@' && prevBlank {
			j := i + w
			for j < len(line) {
				r2, w2 := utf8.DecodeRuneInString(line[j:])
				if isBlank(r2) {
					break
				}
				j += w2
			}
			if j > i+w {
				out = append(out, line[i+w:j])
				i, prevBlank = j, false
				continue
			}
		}
		prevBlank = isBlank(r)
		i += w
	}
	return out
}

var driveRE = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// targetOf is the target of token imported from file (a slash path under the root). A token
// holding a backslash is outside: Windows reads it as a separator, so `.claude\..\..\x` would
// climb out of the copy while the slash form reads it as one name. So is any target that is not
// lexically local on this host (filepath.IsLocal: no climb, no volume, no `:` stream, no device
// name), the one check every read of the copy goes through.
func targetOf(file, token string) (string, bool) {
	if strings.HasPrefix(token, "~") || strings.HasPrefix(token, "/") || strings.Contains(token, `\`) || driveRE.MatchString(token) {
		return strings.ReplaceAll(token, `\`, "/"), true
	}
	rel := path.Clean(path.Join(path.Dir(file), token))
	if rel == ".." || strings.HasPrefix(rel, "../") || !filepath.IsLocal(filepath.FromSlash(rel)) {
		return rel, true
	}
	return rel, false
}

// resolveImports is imports.js's resolve: every import reached from file under root within
// importHops, each file read once, breadth first.
func resolveImports(root, file string) []importRef {
	var out []importRef
	seen := map[string]bool{file: true}
	level := []string{file}
	for hop := 1; hop <= importHops && len(level) > 0; hop++ {
		var next []string
		for _, f := range level {
			text, ok := readText(filepath.Join(root, filepath.FromSlash(f)))
			if !ok {
				continue
			}
			for _, it := range parseImports(text) {
				target, outside := targetOf(f, it.token)
				ref := importRef{file: f, line: it.line, token: it.token, target: target, outside: outside}
				if !outside {
					if fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(target))); err == nil {
						ref.exists, ref.isDir = fi.Mode().IsRegular(), fi.IsDir()
					}
				}
				out = append(out, ref)
				if ref.exists && !seen[target] {
					seen[target] = true
					next = append(next, target)
				}
			}
		}
		level = next
	}
	return out
}

// readText is a file's text without a byte-order mark and with CRLF read as LF; ok is false
// when it cannot be read.
func readText(p string) (string, bool) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	s := strings.TrimPrefix(string(b), string(rune(0xfeff)))
	return strings.ReplaceAll(s, "\r\n", "\n"), true
}
