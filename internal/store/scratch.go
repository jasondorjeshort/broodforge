// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// Scratch roots (ADR 0021, Scratch; 05-cli-conventions.md, Scratch): a fresh instance root in
// the OS temp directory, named by its instanceName, `scratch-<12 hex>`. This file makes the
// directory and removes it; the callers write the configuration (config) and the store
// (Create) into it. Removal runs every session directory through the worktree-removal
// interface first and refuses anything that is not a scratch root.

// CodeNotScratch refuses a deletion of anything but a scratch root.
const CodeNotScratch = "not-scratch"

// ScratchPrefix starts every scratch root's instanceName.
const ScratchPrefix = "scratch-"

var scratchNameRE = regexp.MustCompile(`^scratch-[0-9a-f]{12}$`)

// IsScratchName reports whether name is a scratch root's instanceName and directory name.
func IsScratchName(name string) bool { return scratchNameRE.MatchString(name) }

// Scratch is one scratch root.
type Scratch struct {
	Root         string `json:"root"` // forward slashes
	InstanceName string `json:"instanceName"`
}

// NewScratch makes an empty <temp>/scratch-<12 hex> (0700), a fresh name each time.
func NewScratch() (Scratch, error) {
	tmp := os.TempDir()
	for range 5 {
		var b [6]byte
		if _, err := rand.Read(b[:]); err != nil {
			return Scratch{}, err
		}
		name := ScratchPrefix + hex.EncodeToString(b[:])
		dir := filepath.Join(tmp, name)
		err := os.Mkdir(dir, 0o700)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return Scratch{}, err
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			_ = os.Remove(dir)
			return Scratch{}, err
		}
		return Scratch{Root: filepath.ToSlash(abs), InstanceName: name}, nil
	}
	return Scratch{}, errors.New("no free scratch name in " + filepath.ToSlash(tmp))
}

// CheckScratchPlace refuses (not-scratch) a root that is not a directory directly under the OS
// temp directory named scratch-<12 hex>, or that is a link. It reads nothing inside the root,
// so it runs before anything does.
func CheckScratchPlace(root string) error {
	shown := filepath.ToSlash(root)
	refuse := func(why string) error { return &Error{CodeNotScratch, shown + ": " + why} }
	if !filepath.IsAbs(root) {
		return refuse("not an absolute path")
	}
	clean := filepath.Clean(root)
	name := filepath.Base(clean)
	if !IsScratchName(name) {
		return refuse("the directory is not named " + ScratchPrefix + "<12 hex>")
	}
	fi, err := os.Lstat(clean)
	if err != nil {
		return refuse(err.Error())
	}
	if isLink(fi.Mode()) || !fi.IsDir() {
		return refuse("a link or not a directory")
	}
	parent, err := os.Stat(filepath.Dir(clean))
	if err != nil {
		return refuse(err.Error())
	}
	tmp, err := os.Stat(os.TempDir())
	if err != nil || !os.SameFile(parent, tmp) {
		return refuse("not directly under the OS temp directory " + filepath.ToSlash(os.TempDir()))
	}
	return nil
}

// isLink is a symbolic link, a junction or another reparse point.
func isLink(m fs.FileMode) bool { return m&(fs.ModeSymlink|fs.ModeIrregular) != 0 }

// CheckScratch is CheckScratchPlace, then the name: instanceName, which nameOf reads from the
// root's config only once the place is checked, must be the directory's name.
func CheckScratch(root string, nameOf func(root string) (string, error)) error {
	if err := CheckScratchPlace(root); err != nil {
		return err
	}
	clean := filepath.Clean(root)
	name, _ := nameOf(clean) // none readable: "", refused below
	if name != filepath.Base(clean) {
		return &Error{CodeNotScratch, filepath.ToSlash(root) + ": instanceName " + quoteName(name) + " is not the directory's name"}
	}
	return nil
}

// FixedName is a nameOf for a root whose instanceName the caller made itself.
func FixedName(name string) func(string) (string, error) {
	return func(string) (string, error) { return name, nil }
}

func quoteName(s string) string {
	if s == "" {
		return `""`
	}
	return `"` + s + `"`
}

// DeleteScratch checks root with CheckScratch, then runs every entry under sessions/ that is a
// directory or a link through removeTree (the spawn backend's worktree-remove, whose
// session-directory guard refuses a link), and only then removes the root. sessions/ itself
// being a link is refused here, as the guard refuses it. Any refusal or failure stops before
// the root is removed, and the root stays.
func DeleteScratch(root string, nameOf func(string) (string, error), removeTree func(sessionDir string) error) error {
	if err := CheckScratch(root, nameOf); err != nil {
		return err
	}
	clean := filepath.Clean(root)
	sessions := filepath.Join(clean, "sessions")
	fi, err := os.Lstat(sessions)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	case isLink(fi.Mode()) || !fi.IsDir():
		return &Error{CodeNotScratch, filepath.ToSlash(sessions) + ": sessions/ is a link or not a directory"}
	default:
		entries, err := os.ReadDir(sessions)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() && !isLink(e.Type()) {
				continue
			}
			if err := removeTree(filepath.ToSlash(filepath.Join(sessions, e.Name()))); err != nil {
				return err
			}
		}
	}
	return os.RemoveAll(clean)
}

// ScratchEntry is one scratch root found in the temp directory, with the time it was
// created: the modification time of its config/instance.json, written once at creation, or
// of the directory when that file is missing. Link is set for an entry that is a link (a
// junction or symbolic link named like a root), which no deletion follows.
type ScratchEntry struct {
	Root    string
	Name    string
	Created time.Time
	Link    bool
}

// ListScratch lists the entries in the OS temp directory named scratch-<12 hex>, directories
// and links alike, by name.
func ListScratch() ([]ScratchEntry, error) {
	tmp := os.TempDir()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return nil, err
	}
	var out []ScratchEntry
	for _, e := range entries {
		link := isLink(e.Type())
		if (!e.IsDir() && !link) || !IsScratchName(e.Name()) {
			continue
		}
		dir := filepath.Join(tmp, e.Name())
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		entry := ScratchEntry{Root: filepath.ToSlash(abs), Name: e.Name(), Link: link}
		fi, err := os.Lstat(filepath.Join(dir, "config", "instance.json"))
		if link || err != nil {
			if fi, err = e.Info(); err != nil {
				continue // gone since ReadDir
			}
		}
		entry.Created = fi.ModTime()
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
