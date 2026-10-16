// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// expand mirrors the engine's own path expansion before any check, so the hook judges the file
// the tool will open (engine 2.1.296, the function its Read, Edit and Write apply to
// file_path): surrounding whitespace trimmed as JavaScript's trim() does (jsSpace); `~`, `~/`
// and `~\` to the home directory; and, on a host with drive letters, a `/<letter>/` prefix to
// `<LETTER>:\` (the engine's test is /^\/[a-z]\//i, so `/c\x` and a bare `/c` stay). A `~user` path
// is a hook error: nothing says how the engine reads it. home is "" when unknown, which makes
// any `~` form a hook error too.
func expand(p, home string) (string, error) {
	p = strings.TrimFunc(p, jsSpace)
	if p == "" {
		return "", errors.New("an empty path")
	}
	if p[0] == '~' {
		switch {
		case home == "":
			return "", errors.New("a ~ path with no home directory")
		case p == "~":
			return home, nil
		case p[1] == '/' || p[1] == '\\':
			return home + string(filepath.Separator) + p[2:], nil
		}
		return "", errors.New("a ~user path")
	}
	if drives && len(p) >= 3 && p[0] == '/' && isLetter(p[1]) && p[2] == '/' {
		return strings.ToUpper(p[1:2]) + `:\` + p[3:], nil
	}
	return p, nil
}

// jsSpace is the set JavaScript's String.prototype.trim() removes: WhiteSpace (TAB, VT, FF,
// SP, NBSP, U+FEFF and the Zs category) and LineTerminator (LF, CR, U+2028, U+2029). Unlike
// unicode.IsSpace it holds U+FEFF and not U+0085; U+200B is in neither.
func jsSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000', '\ufeff':
		return true
	}
	return r >= '\u2000' && r <= '\u200a'
}

// drives is whether this host's paths have drive letters: asked of filepath, not of the OS.
var drives = filepath.VolumeName("C:") == "C:"

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// home is the directory `~` names for the engine: Node's os.homedir() reads the same variable
// Go's os.UserHomeDir does on each host.
func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// places are what a decision compares against.
type places struct {
	tree, canon, cfg place
	cfgLine          lineage // the config directory's own ancestors, for "a search above it"
	// credentials are the identities of every credential file the hook can name: the config
	// directory's own and, where ADR 0022's layout makes the instance root derivable, every
	// <root>/harness/<id>/ one; a hard link to any of them is a credential path.
	credentials []os.FileInfo
	home        string
}

// credentialFiles are the files of a config directory that hold an account's secrets or
// identifiers (ADR 0022, C36).
var credentialFiles = []string{credentialName, ".claude.json"}

func newPlaces(tree, canon, cfg string) places {
	p := places{tree: newPlace(tree), canon: newPlace(canon), cfg: newPlace(cfg), cfgLine: newLineage(cfg), home: home()}
	dirs := []string{cfg}
	if root, ok := instanceRoot(cfg); ok {
		if entries, err := os.ReadDir(filepath.Join(root, "harness")); err == nil {
			for _, e := range entries {
				dirs = append(dirs, filepath.Join(root, "harness", e.Name()))
			}
		}
	}
	for _, d := range dirs {
		for _, name := range credentialFiles {
			if fi, err := os.Stat(filepath.Join(d, name)); err == nil {
				p.credentials = append(p.credentials, fi)
			}
		}
	}
	return p
}

// instanceRoot is the instance root a config directory implies under ADR 0022's layout
// (`decisions/0022-linux-spawn-and-handoff.md`, The engine spawn): `<root>/harness/<id>` for a
// subscription, `<root>/sessions/<id>/home/.claude` for an API key.
func instanceRoot(cfg string) (string, bool) {
	up := filepath.Dir
	if filepath.Base(up(cfg)) == "harness" {
		return up(up(cfg)), true
	}
	if filepath.Base(cfg) == ".claude" && filepath.Base(up(cfg)) == "home" && filepath.Base(up(up(up(cfg)))) == "sessions" {
		return up(up(up(up(cfg)))), true
	}
	return "", false
}

// fact resolves one tool_input path both ways and reads the table's facts from the stricter:
// under the tree or the canonical only if both forms are, a credential if either is. A
// relative path resolves against base, the hook input's cwd or else the tree. search marks a
// recursive search root (Grep, and Glob's root and pattern prefix): one that holds the config
// directory reaches the credential as surely as a path inside it.
func (ps places) fact(raw, base string, search bool) (policy.Path, error) {
	exp, err := expand(raw, ps.home)
	if err != nil {
		return policy.Path{}, &Error{"tool_input path is " + err.Error()}
	}
	if err := checkSpelling(exp); err != nil {
		return policy.Path{}, &Error{"tool_input path is " + err.Error()}
	}
	full := join(exp, base)
	f := policy.Path{Target: filepath.ToSlash(strings.TrimSpace(raw)), UnderTree: true, UnderCanonical: true,
		Credential: isCredentialName(filepath.Base(exp))}
	for _, lexical := range []bool{true, false} {
		p, err := walk(full, lexical)
		if err != nil {
			return policy.Path{}, &Error{"cannot resolve a tool_input path: " + errText(err)}
		}
		l := newLineage(p)
		f.UnderTree = f.UnderTree && l.under(ps.tree)
		f.UnderCanonical = f.UnderCanonical && l.under(ps.canon)
		if isCredentialName(realName(p)) || l.under(ps.cfg) {
			f.Credential = true
		}
		self := newPlace(p)
		if self.fi != nil && !self.fi.IsDir() {
			for _, c := range ps.credentials {
				if os.SameFile(self.fi, c) {
					f.Credential = true // a hard link to a credential file
				}
			}
		}
		if search && ps.cfgLine.under(self) {
			f.HoldsCredential = true
		}
	}
	return f, nil
}

// maxAlternatives bounds a Glob pattern's brace expansion.
const maxAlternatives = 64

// globDirs checks a Glob pattern and returns the directory each of its brace alternatives
// names before its first metacharacter (03-hook-contract.md, Paths). It fails closed where the
// literal prefix would not bound what the pattern can reach: a `:` after a drive volume, a
// `{`/`[` group holding a separator, a nested or unbalanced brace, and a segment that can
// match `..` without a wildcard (`..`, `[.][.]`).
func globDirs(pattern string) ([]string, error) {
	if err := checkSpelling(pattern); err != nil {
		return nil, err
	}
	if err := groupsWithoutSeparators(pattern); err != nil {
		return nil, err
	}
	alts, err := braces(pattern)
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(alts))
	for _, alt := range alts {
		for _, seg := range split(alt) {
			if strings.ContainsAny(seg, "*?") {
				continue
			}
			ok, err := path.Match(seg, "..")
			if err != nil {
				return nil, errors.New("a pattern Go cannot read")
			}
			if ok {
				return nil, errors.New("a pattern with a .. segment")
			}
		}
		dirs = append(dirs, globDir(alt))
	}
	return dirs, nil
}

// groupsWithoutSeparators refuses a `[...]` or `{...}` group that holds a separator.
func groupsWithoutSeparators(pattern string) error {
	for _, pair := range []string{"[]", "{}"} {
		depth := 0
		for i := 0; i < len(pattern); i++ {
			switch c := pattern[i]; {
			case c == pair[0]:
				depth++
			case c == pair[1] && depth > 0:
				depth--
			case depth > 0 && (c == '/' || c == '\\'):
				return errors.New("a pattern group holding a separator")
			}
		}
	}
	return nil
}

// braces expands `{a,b}` groups, one level deep.
func braces(pattern string) ([]string, error) {
	i := strings.IndexByte(pattern, '{')
	if i < 0 {
		if strings.IndexByte(pattern, '}') >= 0 {
			return nil, errors.New("a pattern with an unbalanced brace")
		}
		return []string{pattern}, nil
	}
	j := strings.IndexByte(pattern[i:], '}')
	if j < 0 {
		return nil, errors.New("a pattern with an unbalanced brace")
	}
	j += i
	inner := pattern[i+1 : j]
	if strings.ContainsAny(inner, "{") || strings.ContainsAny(pattern[:i], "}") {
		return nil, errors.New("a pattern with a nested brace")
	}
	rest, err := braces(pattern[j+1:])
	if err != nil {
		return nil, err
	}
	var out []string
	for _, a := range strings.Split(inner, ",") {
		for _, r := range rest {
			if out = append(out, pattern[:i]+a+r); len(out) > maxAlternatives {
				return nil, errors.New("a pattern with too many alternatives")
			}
		}
	}
	return out, nil
}

// globMeta are the characters that end a Glob pattern's literal prefix.
const globMeta = "*?[{"

// globDir is the directory a Glob pattern names before its first metacharacter: the prefix
// cut back to its last separator (either slash, which can only make it higher), "" when the
// pattern starts with a metacharacter, the whole pattern when it has none.
func globDir(pattern string) string {
	i := strings.IndexAny(pattern, globMeta)
	if i < 0 {
		return pattern
	}
	j := strings.LastIndexAny(pattern[:i], `/\`)
	if j < 0 {
		return ""
	}
	return pattern[:j+1]
}
