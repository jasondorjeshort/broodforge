// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// LastGoodDir is data/config-last-good/ of the instance at root: the copy the bot wrote at its
// last successful load, laid out as config/ is (ADR 0021).
func LastGoodDir(root string) string { return slashJoin(root, "data", "config-last-good") }

// LoadLastGood reads and validates the last-good copy. With no copy it is a *NotFoundError.
func LoadLastGood(root string, exists CredentialExists) (*Config, error) {
	recoverLastGood(root)
	return LoadDir(LastGoodDir(root), exists)
}

// LastGoodTime is when the last-good copy was written: its instance file's modification time.
func LastGoodTime(root string) (time.Time, error) {
	recoverLastGood(root)
	st, err := os.Stat(filepath.Join(LastGoodDir(root), "instance.json"))
	if err != nil {
		return time.Time{}, err
	}
	return st.ModTime().UTC(), nil
}

// WriteLastGood replaces data/config-last-good/ with the files of c, byte for byte as read.
// The new copy is written beside the old one and renamed into place, so a failure part way
// leaves the previous copy (or, between the two renames, the previous copy under a
// config-last-good.old-* name, which the next read renames back: recoverLastGood).
func WriteLastGood(root string, c *Config) error {
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(data, 0o700); err != nil {
		return err
	}
	staleNew, _ := filepath.Glob(filepath.Join(data, "config-last-good.new-*"))
	for _, s := range staleNew {
		_ = os.RemoveAll(s)
	}
	tmp, err := os.MkdirTemp(data, "config-last-good.new-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	for _, f := range c.Files {
		p := filepath.Join(tmp, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return err
		}
		if err := writeSynced(p, f.Raw); err != nil {
			return err
		}
	}
	final := LastGoodDir(root)
	stale, _ := filepath.Glob(filepath.Join(data, "config-last-good.old-*"))
	for _, s := range stale {
		_ = os.RemoveAll(s)
	}
	old := ""
	if _, err := os.Stat(final); err == nil {
		old = final + ".old-" + filepath.Base(tmp)
		if err := renameRetry(final, old); err != nil {
			return err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := renameRetry(tmp, final); err != nil {
		if old != "" {
			_ = os.Rename(old, final)
		}
		return err
	}
	if old != "" {
		return os.RemoveAll(old)
	}
	return nil
}

func writeSynced(path string, b []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// recoverLastGood finishes a WriteLastGood cut off between its two renames: with no
// data/config-last-good/ and exactly one config-last-good.old-* beside it, that copy is
// renamed back. Anything else is left alone.
func recoverLastGood(root string) {
	final := LastGoodDir(root)
	if _, err := os.Stat(final); !errors.Is(err, fs.ErrNotExist) {
		return
	}
	olds, _ := filepath.Glob(filepath.Join(root, "data", "config-last-good.old-*"))
	if len(olds) == 1 {
		_ = os.Rename(olds[0], final)
	}
}

// Rename retries: 5 tries, 50 ms apart.
const (
	renameTries = 5
	renamePause = 50 * time.Millisecond
)

// renameRetry is os.Rename retried while it fails for permission. Observed once in ten `serve`
// starts on the PC (2026-10-16, P10): the rename of the freshly written last-good directory
// failed with `Access is denied`, unreproduced, most likely another process (a scanner or the
// indexer) holding a file just written in it. On Windows ERROR_ACCESS_DENIED is fs.ErrPermission;
// a sharing violation has no host-neutral sentinel and is not retried. Any other error, or the
// fifth failure, is returned as is.
func renameRetry(from, to string) error {
	var err error
	for i := 0; i < renameTries; i++ {
		if i > 0 {
			time.Sleep(renamePause)
		}
		if err = os.Rename(from, to); err == nil || !errors.Is(err, fs.ErrPermission) {
			return err
		}
	}
	return err
}

// LastGoodInstanceName is instanceName of the last-good copy, read alone as InstanceName reads
// config/'s: a client whose config/ names another instance than the bot runs as (an edit not
// yet reloaded, or one the bot refused) still finds the bot under the name it started with.
func LastGoodInstanceName(root string) (string, error) {
	return readInstanceName(slashJoin(LastGoodDir(root), "instance.json"))
}
