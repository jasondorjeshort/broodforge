// SPDX-License-Identifier: GPL-3.0-or-later

package credentials

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Error is a refusal with a stable code, mapped to the CLI's error line by the caller; this
// package does not import control.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Codes this package refuses with.
const (
	CodeKeyExists         = "key-exists"
	CodeKeyMissing        = "key-missing"
	CodeBadKey            = "bad-key"
	CodeNoKeyFile         = "no-key-file"
	CodeStoreUnreadable   = "store-unreadable"
	CodeUnknownCredential = "unknown-credential"
	CodeUnknownUser       = "unknown-user"
	CodeBadKind           = "bad-kind"
	CodeBadValue          = "bad-value"
	CodeRemoved           = "removed"
	CodeNotSubscription   = "not-subscription"
)

// KeyEnv overrides the configured key file, except on a scratch root (ADR 0022).
const KeyEnv = "BROODFORGE_KEY_FILE"

// KeySize is the AES-256 key length.
const KeySize = 32

// scratchPrefix starts every scratch root's instanceName (05-cli-conventions.md, Scratch).
const scratchPrefix = "scratch-"

// KeyPath is the store key's file for the instance at root: BROODFORGE_KEY_FILE when set,
// except on a scratch root, which names its own key under the root; else the instance file's
// credentials.keyFile. A relative path is taken from the root. Neither set is no-key-file.
func KeyPath(root, instanceName, configured string, lookup func(string) (string, bool)) (string, error) {
	p := configured
	if !strings.HasPrefix(instanceName, scratchPrefix) {
		if v, ok := lookup(KeyEnv); ok && v != "" {
			p = v
		}
	}
	if p == "" {
		return "", &Error{CodeNoKeyFile, "set credentials.keyFile in config/instance.json or " + KeyEnv}
	}
	if !filepath.IsAbs(p) && !(p[0] == '/' || p[0] == '\\') {
		p = filepath.Join(root, p)
	}
	return filepath.ToSlash(p), nil
}

// Init writes a new random key to path, refusing a file that exists, and returns it as the 64
// hex digits the file holds (then a newline), for the one print at `credential init`.
func Init(path string) (string, error) {
	if _, err := os.Lstat(path); err == nil {
		return "", &Error{CodeKeyExists, path}
	}
	var key [KeySize]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	text := hex.EncodeToString(key[:])
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return "", &Error{CodeKeyExists, path}
	}
	if err != nil {
		return "", err
	}
	_, werr := f.WriteString(text + "\n")
	if err := errors.Join(werr, f.Close()); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	// Read-only for its owner (ADR 0022: 0400); on Windows this sets the read-only attribute.
	if err := os.Chmod(path, 0o400); err != nil {
		return "", err
	}
	return text, nil
}

// ReadKey reads the key file: 64 hex digits, surrounding whitespace ignored.
func ReadKey(path string) ([KeySize]byte, error) {
	var key [KeySize]byte
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return key, &Error{CodeKeyMissing, "no key file at " + filepath.ToSlash(path)}
	}
	if err != nil {
		return key, &Error{CodeKeyMissing, err.Error()}
	}
	text := bytes.TrimSpace(b)
	if len(text) != 2*KeySize {
		return key, &Error{CodeBadKey, filepath.ToSlash(path) + ": want 64 hex digits"}
	}
	if _, err := hex.Decode(key[:], text); err != nil {
		return key, &Error{CodeBadKey, filepath.ToSlash(path) + ": want 64 hex digits"}
	}
	return key, nil
}
