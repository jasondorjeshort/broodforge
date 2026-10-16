// SPDX-License-Identifier: GPL-3.0-or-later

package hook

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// maxLinks bounds the links followed while resolving one path, as a loop guard.
const maxLinks = 64

// checkSpelling refuses a path the checks below cannot compare (03-hook-contract.md, Paths):
// a device or network volume (`\\?\`, `\\.\`, `\??\`, UNC, `//host/share`), a drive-relative
// path (`D:x`), and any `:` after the volume, which on Windows names an NTFS stream
// (`file::$DATA`). On Linux there is no volume, so the rule is a colon anywhere: refused, not
// read, so it fails closed.
func checkSpelling(p string) error {
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

// rooted reports a path that starts at a root without a volume, such as /tmp/x on Windows.
func rooted(p string) bool { return p != "" && os.IsPathSeparator(p[0]) }

// join makes p absolute against base without cleaning it, so `..` is still there to be
// applied after the links before it are followed. A rooted path takes base's volume.
func join(p, base string) string {
	switch {
	case filepath.IsAbs(p):
		return p
	case rooted(p):
		return filepath.VolumeName(base) + p
	}
	return base + string(filepath.Separator) + p
}

func split(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool { return r < 0x80 && os.IsPathSeparator(uint8(r)) })
}

// walk follows every link on the longest existing prefix of abs (03-hook-contract.md, Paths);
// the part past that prefix is kept as written, so a file about to be created resolves to
// where it would land. A link is anything Lstat reports as a symlink or as irregular: a
// Windows directory junction is irregular, and filepath.EvalSymlinks does not follow it (Go
// 1.27) while os.Readlink does. A link that cannot be read, or whose target is a spelling
// checkSpelling refuses, is an error.
//
// lexical selects how `..` is applied. Windows cleans a path as text before it opens anything,
// so `tree\j\..\x` is `tree\x` whatever j is; POSIX applies `..` to the directory the link
// leads to, so the same path is beside j's target. walk(lexical) cleans first, as Windows
// does; walk(!lexical) applies `..` to the resolved prefix, as realpath does. The hook takes
// both and the stricter answer, so it is right on either host without asking which it is on.
func walk(abs string, lexical bool) (string, error) {
	if lexical {
		abs = filepath.Clean(abs)
	}
	vol := filepath.VolumeName(abs)
	queue := split(abs[len(vol):])
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
		if links++; links > maxLinks {
			return "", errors.New("too many links")
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", err
		}
		if err := checkSpelling(target); err != nil {
			return "", errors.New("a link to " + err.Error())
		}
		if lexical {
			full := filepath.Clean(join(target, resolved) + string(filepath.Separator) + filepath.Join(queue...))
			vol = filepath.VolumeName(full)
			queue = split(full[len(vol):])
			resolved = vol + string(filepath.Separator)
			continue
		}
		if filepath.IsAbs(target) || rooted(target) {
			tv := filepath.VolumeName(target)
			if tv == "" {
				tv = filepath.VolumeName(resolved)
			}
			resolved = tv + string(filepath.Separator)
			target = target[len(filepath.VolumeName(target)):]
		}
		queue = append(split(target), queue...)
	}
	return resolved, nil
}

// canonical is a trusted path (the tree, the canonical, the config directory) resolved.
func canonical(p string) (string, error) {
	if err := checkSpelling(p); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return walk(abs, true)
}

// place is a directory the checks compare against: its canonical path and, when it exists,
// its identity, so another spelling of it (an 8.3 short name, a trailing dot, another case)
// is still it.
type place struct {
	path string
	fi   os.FileInfo
}

func newPlace(p string) place {
	fi, err := os.Stat(p)
	if err != nil {
		fi = nil
	}
	return place{path: p, fi: fi}
}

// lineage is a resolved path with the identity of each of its existing ancestors, itself
// included, for under.
type lineage struct {
	path string
	ids  []os.FileInfo
}

func newLineage(p string) lineage {
	l := lineage{path: p}
	for a := p; ; a = filepath.Dir(a) {
		if fi, err := os.Stat(a); err == nil {
			l.ids = append(l.ids, fi)
		}
		if filepath.Dir(a) == a {
			return l
		}
	}
}

// under reports whether the path is the place or beneath it: as text (filepath.Rel compares as
// the host does, case-insensitively on Windows), or by identity, when one of its existing
// ancestors is the place's directory.
func (l lineage) under(pl place) bool {
	if rel, err := filepath.Rel(pl.path, l.path); err == nil && !filepath.IsAbs(rel) &&
		rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return true
	}
	if pl.fi == nil {
		return false
	}
	for _, fi := range l.ids {
		if os.SameFile(fi, pl.fi) {
			return true
		}
	}
	return false
}

// credentialName is the file the engine keeps a subscription's tokens in.
const credentialName = ".credentials.json"

// isCredentialName compares a base name as Windows opens it: any case, trailing dots and
// spaces dropped.
func isCredentialName(name string) bool {
	return strings.EqualFold(strings.TrimRight(name, ". "), credentialName)
}

// realName is the name the directory holds for an existing p: another spelling (an 8.3 short
// name, a trailing dot) is found by identity among its siblings. A p that does not exist keeps
// its own base name.
func realName(p string) string {
	base := filepath.Base(p)
	fi, err := os.Lstat(p)
	if err != nil {
		return base
	}
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		return base
	}
	for _, e := range entries {
		if e.Name() == base {
			return base
		}
	}
	for _, e := range entries {
		if efi, err := os.Lstat(filepath.Join(filepath.Dir(p), e.Name())); err == nil && os.SameFile(efi, fi) {
			return e.Name()
		}
	}
	return base
}

// configDir derives the engine's config directory from transcript_path: the parent of the
// nearest ancestor directory named projects (03-hook-contract.md, Paths), canonicalised.
func configDir(transcript string) (string, error) {
	if transcript == "" {
		return "", &Error{"no transcript_path"}
	}
	if err := checkSpelling(transcript); err != nil {
		return "", &Error{"transcript_path is " + err.Error()}
	}
	abs, err := filepath.Abs(transcript)
	if err != nil {
		return "", &Error{"transcript_path: " + err.Error()}
	}
	for dir := filepath.Dir(abs); ; {
		if filepath.Base(dir) == "projects" {
			cfg, err := canonical(filepath.Dir(dir))
			if err != nil {
				return "", &Error{"cannot resolve the config directory: " + errText(err)}
			}
			return cfg, nil
		}
		up := filepath.Dir(dir)
		if up == dir {
			return "", &Error{"no projects directory above transcript_path"}
		}
		dir = up
	}
}

// errText is an error without the path it may carry, so stderr names no host path.
func errText(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Op + ": " + pe.Err.Error()
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return le.Op + ": " + le.Err.Error()
	}
	return err.Error()
}
