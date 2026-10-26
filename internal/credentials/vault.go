// SPDX-License-Identifier: GPL-3.0-or-later

package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// StorePath is credentials/store.json of the instance at root (ADR 0022, Both hosts).
func StorePath(root string) string {
	return filepath.ToSlash(filepath.Join(root, "credentials", "store.json"))
}

// vaultVersion is the store file's format.
const vaultVersion = 1

// vaultFile is credentials/store.json: one AES-256-GCM entry per credential id, the id as the
// additional data, so an entry moved to another id does not open.
type vaultFile struct {
	Version int                   `json:"version"`
	Entries map[string]vaultEntry `json:"entries"`
}

type vaultEntry struct {
	Nonce  []byte `json:"nonce"`  // base64 in the file
	Sealed []byte `json:"sealed"` // ciphertext and tag
}

// Vault is the encrypted store file. It is read by open-read-close on every call and written
// by temp file and rename, by the bot only.
type Vault struct {
	path string
	aead cipher.AEAD
	mu   sync.Mutex
}

// OpenVault opens the store file at path with the key in keyFile. A missing store file is an
// empty store; a missing key file is key-missing.
func OpenVault(path, keyFile string) (*Vault, error) {
	key, err := ReadKey(keyFile)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// A crash between a write's temp file and its rename leaves the temp file, sealed values
	// and all; the bot is the only writer, so at open any is stale.
	if stale, err := filepath.Glob(filepath.Join(filepath.Dir(path), tempPattern)); err == nil {
		for _, f := range stale {
			_ = os.Remove(f)
		}
	}
	return &Vault{path: path, aead: aead}, nil
}

// Entries counts the entries of the store file at path without opening any: what a new key
// would make unreadable (`setup key`, key-in-use). A missing file holds none.
func Entries(path string) (int, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, &Error{CodeStoreUnreadable, err.Error()}
	}
	var f vaultFile
	if err := json.Unmarshal(b, &f); err != nil {
		return 0, &Error{CodeStoreUnreadable, path + ": not a version 1 credential store"}
	}
	return len(f.Entries), nil
}

// tempPattern names a write's temp file beside the store file.
const tempPattern = ".store-*.json"

func (v *Vault) read() (vaultFile, error) {
	f := vaultFile{Version: vaultVersion, Entries: map[string]vaultEntry{}}
	b, err := os.ReadFile(v.path)
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, &Error{CodeStoreUnreadable, err.Error()}
	}
	if err := json.Unmarshal(b, &f); err != nil || f.Version != vaultVersion {
		return f, &Error{CodeStoreUnreadable, v.path + ": not a version 1 credential store"}
	}
	if f.Entries == nil {
		f.Entries = map[string]vaultEntry{}
	}
	return f, nil
}

func (v *Vault) write(f vaultFile) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(v.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return err
	}
	_, werr := tmp.Write(append(b, '\n'))
	serr := tmp.Sync()
	if err := errors.Join(werr, serr, tmp.Close()); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), v.path); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}

// Put seals value under id, replacing any entry it had.
func (v *Vault) Put(id string, value []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	f, err := v.read()
	if err != nil {
		return err
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	f.Entries[id] = vaultEntry{Nonce: nonce, Sealed: v.aead.Seal(nil, nonce, value, []byte(id))}
	return v.write(f)
}

// Get opens id's entry. A missing entry is unknown-credential; one that does not open (another
// key, another id's entry, a changed byte) is store-unreadable.
func (v *Vault) Get(id string) ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	f, err := v.read()
	if err != nil {
		return nil, err
	}
	e, ok := f.Entries[id]
	if !ok {
		return nil, &Error{CodeUnknownCredential, "no value for " + id + " in the credential store"}
	}
	if len(e.Nonce) != v.aead.NonceSize() {
		return nil, &Error{CodeStoreUnreadable, id + ": bad nonce"}
	}
	value, err := v.aead.Open(nil, e.Nonce, e.Sealed, []byte(id))
	if err != nil {
		return nil, &Error{CodeStoreUnreadable, id + ": the entry does not open with this key"}
	}
	return value, nil
}

// Delete removes id's entry; a missing one is not an error.
func (v *Vault) Delete(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	f, err := v.read()
	if err != nil {
		return err
	}
	if _, ok := f.Entries[id]; !ok {
		return nil
	}
	delete(f.Entries, id)
	return v.write(f)
}
