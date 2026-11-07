// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// The path resolution AskCell reads a call's target with: a mirror of internal/hook's (paths.go
// and facts.go: expand, checkSpelling, join, walk, lineage), because session may not import
// hook (03-architecture.md, Components). It must answer "under the tree" as the hook does, so a
// call the hook would deny is never read as the engine's own decision. Kept in step with the
// hook's by hand; the hook's credential checks beyond the base name (the config directory, hard
// links) are not mirrored: the hook denies those calls, and its deny text classifies first.

// askMaxLinks bounds the links followed while resolving one path, as the hook's maxLinks.
const askMaxLinks = 64

// askDrives is whether this host's paths have drive letters: asked of filepath, as the hook does.
var askDrives = filepath.VolumeName("C:") == "C:"

// askExpand is the hook's expand: trimmed as JavaScript's trim() does, `~`, `~/` and `~\` to
// home, and on a host with drive letters `/<letter>/` to `<LETTER>:\`; a `~user` path or a
// `~` with no home is an error.
func askExpand(p, home string) (string, error) {
	p = strings.TrimFunc(p, askJSSpace)
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
	if askDrives && len(p) >= 3 && p[0] == '/' && askLetter(p[1]) && p[2] == '/' {
		return strings.ToUpper(p[1:2]) + `:\` + p[3:], nil
	}
	return p, nil
}

func askJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000', '\ufeff':
		return true
	}
	return r >= '\u2000' && r <= '\u200a'
}

func askLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// askSpelling is the hook's checkSpelling: a device or network volume, a drive-relative path or
// a ':' after the volume is a spelling the checks cannot compare.
func askSpelling(p string) error {
	vol := filepath.VolumeName(p)
	drive := len(vol) == 2 && vol[1] == ':'
	rest := p[len(vol):]
	switch {
	case vol != "" && !drive:
		return errors.New("a device or network path")
	case drive && (rest == "" || !os.IsPathSeparator(rest[0])):
		return errors.New("a drive-relative path")
	case strings.ContainsRune(rest, ':'):
		return errors.New("a ':' after the volume")
	}
	return nil
}

func askRooted(p string) bool { return p != "" && os.IsPathSeparator(p[0]) }

// askJoin is the hook's join: p absolute against base, uncleaned; a rooted p takes base's volume.
func askJoin(p, base string) string {
	switch {
	case filepath.IsAbs(p):
		return p
	case askRooted(p):
		return filepath.VolumeName(base) + p
	}
	return base + string(filepath.Separator) + p
}

func askSplit(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool { return r < 0x80 && os.IsPathSeparator(uint8(r)) })
}

// askWalk is the hook's walk: every link (a symlink, or a Windows junction, which Lstat reports
// irregular) on the longest existing prefix followed, `..` applied as text first (lexical, as
// Windows opens a path) or after the link before it (as POSIX does).
func askWalk(abs string, lexical bool) (string, error) {
	if lexical {
		abs = filepath.Clean(abs)
	}
	vol := filepath.VolumeName(abs)
	queue := askSplit(abs[len(vol):])
	resolved := vol + string(filepath.Separator)
	links := 0
	for len(queue) > 0 {
		part := queue[0]
		queue = queue[1:]
		switch part {
		case ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			continue
		}
		next := filepath.Join(resolved, part)
		fi, err := os.Lstat(next)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
				return filepath.Join(append([]string{next}, queue...)...), nil
			}
			return "", err
		}
		if fi.Mode()&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
			resolved = next
			continue
		}
		if links++; links > askMaxLinks {
			return "", errors.New("too many links")
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", err
		}
		if err := askSpelling(target); err != nil {
			return "", err
		}
		if lexical {
			full := filepath.Clean(askJoin(target, resolved) + string(filepath.Separator) + filepath.Join(queue...))
			vol = filepath.VolumeName(full)
			queue = askSplit(full[len(vol):])
			resolved = vol + string(filepath.Separator)
			continue
		}
		if filepath.IsAbs(target) || askRooted(target) {
			tv := filepath.VolumeName(target)
			if tv == "" {
				tv = filepath.VolumeName(resolved)
			}
			resolved = tv + string(filepath.Separator)
			target = target[len(filepath.VolumeName(target)):]
		}
		queue = append(askSplit(target), queue...)
	}
	return resolved, nil
}

// askUnder is the hook's lineage.under: p is dir or beneath it as text, or one of p's existing
// ancestors is dir by identity (another spelling of it, such as an 8.3 name).
func askUnder(p, dir string) bool {
	if rel, err := filepath.Rel(dir, p); err == nil && !filepath.IsAbs(rel) &&
		rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return true
	}
	dfi, err := os.Stat(dir)
	if err != nil {
		return false
	}
	for a := p; ; a = filepath.Dir(a) {
		if fi, err := os.Stat(a); err == nil && os.SameFile(fi, dfi) {
			return true
		}
		if filepath.Dir(a) == a {
			return false
		}
	}
}

// askCredentialName is the hook's credential base name: `.credentials.json` in any case,
// trailing dots and spaces dropped.
func askCredentialName(name string) bool {
	return strings.EqualFold(strings.TrimRight(name, ". "), ".credentials.json")
}

// askHome is the directory `~` names, as the hook reads it.
func askHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// askUnderTree reports whether a tool-input path lands under the tree as the hook resolves it
// (both walks must agree), and whether it names a credential file. ok is false for a path the
// hook refuses as an error (a spelling it cannot compare, a `~user`, a link it cannot read).
func askUnderTree(raw, tree string) (under, credential, ok bool) {
	exp, err := askExpand(raw, askHome())
	if err != nil || askSpelling(exp) != nil {
		return false, false, false
	}
	root, err := filepath.Abs(filepath.FromSlash(tree))
	if err != nil || askSpelling(root) != nil {
		return false, false, false
	}
	if root, err = askWalk(root, true); err != nil {
		return false, false, false
	}
	full := askJoin(exp, root)
	under, credential = true, askCredentialName(filepath.Base(exp))
	for _, lexical := range []bool{true, false} {
		p, err := askWalk(full, lexical)
		if err != nil {
			return false, false, false
		}
		under = under && askUnder(p, root)
		credential = credential || askCredentialName(filepath.Base(p))
	}
	return under, credential, true
}
