// SPDX-License-Identifier: GPL-3.0-or-later

package credentials

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// Credential kinds (02-domain.md, R-39) and statuses (03-architecture.md, Credential status).
const (
	KindAPIKey       = "api-key"
	KindSubscription = "subscription"

	StatusUsable     = "usable"
	StatusNeedsLogin = "needs-login"
	StatusRemoved    = "removed"

	// ProviderAnthropic is the one provider of the first version.
	ProviderAnthropic = "anthropic"
)

// MaxValue bounds an API key read from stdin.
const MaxValue = 64 << 10

// actorHost is the audit actor of a command run on the host without --as (05-debug-interface.md).
const actorHost = "host"

// Record is one credential row: never a secret (N-12). StoreRef is the credential's id in the
// store file for an API key and its config directory for a subscription.
type Record struct {
	ID         string
	Kind       string
	Provider   string
	User       string // "" for the instance's own
	Label      string
	Status     string
	AddedAt    string
	LastUsedAt string // "" for never
	StoreRef   string
}

// Harness creates a subscription credential's config directory: the spawn backend's
// harness-create (ADR 0022 companion); Remove undoes a Create whose row was never written.
type Harness interface {
	Create(id string) (dir string, err error)
	Remove(dir string) error
}

// SpawnHarness runs harness-create and harness-remove through a spawn backend on
// <root>/harness/<id>.
type SpawnHarness struct {
	Root    string
	Backend spawn.Backend
}

// Create makes the directory and returns it.
func (h SpawnHarness) Create(id string) (string, error) {
	dir := spawn.HarnessDir(h.Root, id)
	if err := h.Backend.HarnessCreate(dir); err != nil {
		return "", harnessError(err)
	}
	return dir, nil
}

// Remove deletes the directory.
func (h SpawnHarness) Remove(dir string) error { return harnessError(h.Backend.HarnessRemove(dir)) }

// harnessError keeps the backend's code (harness, or preflight-backend for the held one).
func harnessError(err error) error {
	var se *spawn.Error
	if errors.As(err, &se) {
		return &Error{se.Code, se.Message}
	}
	return err
}

// Service is the bot's CredentialStore: the credential rows in the store, the values in the
// encrypted store file, audit events for adds and removes.
type Service struct {
	st         *store.Store
	aud        *audit.Writer
	vault      *Vault
	harness    Harness
	enginePath string
	now        func() time.Time
	mu         sync.Mutex
}

// Config is what a Service needs besides the store.
type Config struct {
	Root       string
	KeyFile    string
	EnginePath string  // engine.path; "" prints `claude` in the login command
	Harness    Harness // nil is SpawnHarness for Root and Spawn
	// Spawn is the configured backend; nil is the same-account directory operations.
	Spawn spawn.Backend
}

// NewService opens the store file with the key; a missing key refuses (ADR 0022).
func NewService(st *store.Store, aud *audit.Writer, c Config) (*Service, error) {
	v, err := OpenVault(StorePath(c.Root), c.KeyFile)
	if err != nil {
		return nil, err
	}
	h := c.Harness
	if h == nil {
		b := c.Spawn
		if b == nil {
			b = &spawn.SameAccount{}
		}
		h = SpawnHarness{Root: c.Root, Backend: b}
	}
	return &Service{st: st, aud: aud, vault: v, harness: h, enginePath: c.EnginePath, now: time.Now}, nil
}

// newID is `cred-<base36>` of 48 random bits.
func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[2:]); err != nil {
		return "", err
	}
	return "cred-" + strconv.FormatUint(binary.BigEndian.Uint64(b[:]), 36), nil
}

// Add records a credential for a registered user. An API key's value goes into the store
// file first, so a row never names a value that is not there; a subscription gets its config
// directory from the harness and starts needs-login.
func (s *Service) Add(kind, user, label string, value []byte) (Record, error) {
	switch kind {
	case KindAPIKey:
		if len(value) == 0 || len(value) > MaxValue {
			return Record{}, &Error{CodeBadValue, "an api-key credential reads its value from stdin, 1 to 65536 bytes"}
		}
	case KindSubscription:
		if len(value) != 0 {
			return Record{}, &Error{CodeBadValue, "a subscription credential has no value; its user logs in"}
		}
	default:
		return Record{}, &Error{CodeBadKind, "kind is " + strconv.Quote(kind) + ", want api-key or subscription"}
	}
	if user == "" || label == "" {
		return Record{}, &Error{CodeBadValue, "a credential needs a user and a label"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	registered := false
	if err := s.st.Query("SELECT 1 FROM registration WHERE user_id = ?", []store.Value{store.Text(user)},
		func(*store.Row) error { registered = true; return nil }); err != nil {
		return Record{}, err
	}
	if !registered {
		return Record{}, &Error{CodeUnknownUser, user + " is not registered"}
	}
	id, err := newID() // 48 random bits; a collision fails the insert below
	if err != nil {
		return Record{}, err
	}
	r := Record{ID: id, Kind: kind, Provider: ProviderAnthropic, User: user, Label: label,
		AddedAt: store.FormatTime(s.now())}
	switch kind {
	case KindAPIKey:
		r.Status, r.StoreRef = StatusUsable, id
		if err := s.vault.Put(id, value); err != nil {
			return Record{}, err
		}
	case KindSubscription:
		dir, err := s.harness.Create(id)
		if err != nil {
			return Record{}, err
		}
		r.Status, r.StoreRef = StatusNeedsLogin, dir
	}
	details, err := json.Marshal(map[string]string{"id": id, "kind": kind, "user": user, "label": label})
	if err != nil {
		return Record{}, err
	}
	err = s.st.Write(func(tx *store.Tx) error {
		if _, err := tx.Insert(`INSERT INTO credential (id, kind, provider, user_id, label, status, added_at, store_ref)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, store.Text(r.ID), store.Text(r.Kind), store.Text(r.Provider),
			store.Text(r.User), store.Text(r.Label), store.Text(r.Status), store.Text(r.AddedAt), store.Text(r.StoreRef)); err != nil {
			return err
		}
		_, err := s.aud.Append(tx, audit.Event{Kind: audit.KindCredentialAdded, Actor: actorHost, Registration: user, Details: details})
		return err
	})
	if err != nil {
		switch kind {
		case KindAPIKey:
			_ = s.vault.Delete(id)
		case KindSubscription:
			_ = s.harness.Remove(r.StoreRef)
		}
		return Record{}, err
	}
	return r, nil
}

// Remove marks a credential removed and deletes its value; the next session start naming it
// is refused (R-41).
func (s *Service) Remove(id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := Show(s.st, id)
	if err != nil {
		return r, err
	}
	if r.Status == StatusRemoved {
		return r, &Error{CodeRemoved, id + " is already removed"}
	}
	details, err := json.Marshal(map[string]string{"id": id})
	if err != nil {
		return r, err
	}
	err = s.st.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec("UPDATE credential SET status = ? WHERE id = ?", store.Text(StatusRemoved), store.Text(id)); err != nil {
			return err
		}
		_, err := s.aud.Append(tx, audit.Event{Kind: audit.KindCredentialRemoved, Actor: actorHost, Registration: r.User, Details: details})
		return err
	})
	if err != nil {
		return r, err
	}
	r.Status = StatusRemoved
	if r.Kind == KindAPIKey {
		if err := s.vault.Delete(id); err != nil {
			return r, err
		}
	}
	return r, nil
}

// LoginCommand is the command the user runs to log the engine into a subscription's config
// directory (spike C25: the engine's `auth login` with CLAUDE_CONFIG_DIR set).
func (s *Service) LoginCommand(id string) (string, error) {
	r, err := s.subscription(id)
	if err != nil {
		return "", err
	}
	engine := s.enginePath
	if engine == "" {
		engine = "claude"
	}
	return "CLAUDE_CONFIG_DIR=" + shellQuote(r.StoreRef) + " " + shellQuote(filepath.ToSlash(engine)) + " auth login", nil
}

// LoginDone records a subscription usable once its user has logged in.
func (s *Service) LoginDone(id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.subscription(id)
	if err != nil {
		return r, err
	}
	err = s.st.Write(func(tx *store.Tx) error {
		_, err := tx.Exec("UPDATE credential SET status = ? WHERE id = ?", store.Text(StatusUsable), store.Text(id))
		return err
	})
	if err != nil {
		return r, err
	}
	r.Status = StatusUsable
	return r, nil
}

func (s *Service) subscription(id string) (Record, error) {
	r, err := Show(s.st, id)
	if err != nil {
		return r, err
	}
	if r.Kind != KindSubscription {
		return r, &Error{CodeNotSubscription, id + " is an " + r.Kind + " credential"}
	}
	if r.Status == StatusRemoved {
		return r, &Error{CodeRemoved, id + " is removed"}
	}
	return r, nil
}

// Secret is an API key's value, for the engine process only (03-architecture.md, engine row).
func (s *Service) Secret(id string) ([]byte, error) {
	r, err := Show(s.st, id)
	if err != nil {
		return nil, err
	}
	if r.Status == StatusRemoved {
		return nil, &Error{CodeRemoved, id + " is removed"}
	}
	if r.Kind != KindAPIKey {
		return nil, &Error{CodeBadKind, id + " is a " + r.Kind + " credential and has no value"}
	}
	return s.vault.Get(r.StoreRef)
}

// List and Show read the bot's own store.
func (s *Service) List(user string) ([]Record, error) { return List(s.st, user) }

func (s *Service) Show(id string) (Record, error) { return Show(s.st, id) }

const recordColumns = "id, kind, provider, coalesce(user_id, ''), label, status, added_at, coalesce(last_used_at, ''), store_ref"

func scan(row *store.Row) Record {
	return Record{ID: row.Text(0), Kind: row.Text(1), Provider: row.Text(2), User: row.Text(3), Label: row.Text(4),
		Status: row.Text(5), AddedAt: row.Text(6), LastUsedAt: row.Text(7), StoreRef: row.Text(8)}
}

// List is every credential, or one user's, in the order added; the bot's store or the CLI's
// read-only one.
func List(q store.Querier, user string) ([]Record, error) {
	query, args := "SELECT "+recordColumns+" FROM credential ORDER BY added_at, rowid", []store.Value(nil)
	if user != "" {
		query = "SELECT " + recordColumns + " FROM credential WHERE user_id = ? ORDER BY added_at, rowid"
		args = []store.Value{store.Text(user)}
	}
	out := []Record{}
	err := q.Query(query, args, func(row *store.Row) error { out = append(out, scan(row)); return nil })
	return out, err
}

// Show is one credential; an id the store does not hold is unknown-credential.
func Show(q store.Querier, id string) (Record, error) {
	var r Record
	found := false
	err := q.Query("SELECT "+recordColumns+" FROM credential WHERE id = ?", []store.Value{store.Text(id)},
		func(row *store.Row) error { r, found = scan(row), true; return nil })
	if err == nil && !found {
		err = &Error{CodeUnknownCredential, "no credential " + id}
	}
	return r, err
}

// Exists is config's id-existence check (ADR 0021): the id is in the store and not removed.
func Exists(q store.Querier) func(id string) bool {
	return func(id string) bool {
		r, err := Show(q, id)
		return err == nil && r.Status != StatusRemoved
	}
}

// shellQuote quotes a word for sh when it holds anything but plain path characters.
func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("/._-:+@=", r))
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// IsRefusal reports whether err is one of this package's coded refusals.
func IsRefusal(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
