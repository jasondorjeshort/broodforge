// SPDX-License-Identifier: GPL-3.0-or-later

package session

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// maxNameRunes bounds a git name (04, Registration: empty or over 100 characters is refused).
const maxNameRunes = 100

// userIDRE is a Discord user id: a snowflake, decimal digits.
var userIDRE = regexp.MustCompile(`^[0-9]{1,20}$`)

// Register is 04's /register and 05's `register <user id> --name --email`: one registration
// per Discord user id; registering again replaces name and email (R-02). Either way it writes
// registration-changed, with the old pair when there was one.
func (s *Service) Register(user, name, email, actor string) (Registration, error) {
	if !userIDRE.MatchString(user) {
		return Registration{}, badArgs(quote(user) + " is not a Discord user id")
	}
	name = strings.TrimSpace(name)
	switch {
	case name == "" || utf8.RuneCountInString(name) > maxNameRunes:
		return Registration{}, s.refuse(actorOf(actor), "", "register", "name",
			&Error{CodeBadRegistration, "the name must be 1 to 100 characters"})
	case !strings.Contains(email, "@") || strings.ContainsAny(email, " \t\r\n"):
		return Registration{}, s.refuse(actorOf(actor), "", "register", "email",
			&Error{CodeBadRegistration, "the email must contain @ and no spaces"})
	}
	now := s.stamp()
	err := s.cfg.Store.Write(func(tx *store.Tx) error {
		var old string
		if err := tx.Query("SELECT git_name, git_email FROM registration WHERE user_id = ?", []store.Value{store.Text(user)},
			func(r *store.Row) error { old = r.Text(0) + " <" + r.Text(1) + ">"; return nil }); err != nil {
			return err
		}
		if old == "" {
			if _, err := tx.Exec(`INSERT INTO registration (user_id, git_name, git_email, display_name, registered_at)
VALUES (?, ?, ?, '', ?)`, store.Text(user), store.Text(name), store.Text(email), store.Text(now)); err != nil {
				return err
			}
		} else if _, err := tx.Exec("UPDATE registration SET git_name = ?, git_email = ?, registered_at = ? WHERE user_id = ?",
			store.Text(name), store.Text(email), store.Text(now), store.Text(user)); err != nil {
			return err
		}
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindRegistrationChanged, Actor: actorOf(actor),
			Registration: user, Details: details("old", old, "new", name+" <"+email+">")})
		return err
	})
	if err != nil {
		return Registration{}, err
	}
	return ShowRegistration(s.cfg.Store, user)
}

// SetDefault is `register set-default <user id> <credential id>`: the credential a session
// start uses when it names none. It must exist and not be removed.
func (s *Service) SetDefault(user, credID, actor string) (Registration, error) {
	if _, err := ShowRegistration(s.cfg.Store, user); err != nil {
		return Registration{}, err
	}
	cred, err := credentials.Show(s.cfg.Store, credID)
	if err != nil {
		if e, ok := credentials.IsRefusal(err); ok {
			return Registration{}, &Error{e.Code, e.Message}
		}
		return Registration{}, err
	}
	if cred.Status == credentials.StatusRemoved {
		return Registration{}, &Error{CodeCredentialStatus, credID + " is " + cred.Status}
	}
	err = s.cfg.Store.Write(func(tx *store.Tx) error {
		if _, err := tx.Exec("UPDATE registration SET default_credential_id = ? WHERE user_id = ?", store.Text(credID), store.Text(user)); err != nil {
			return err
		}
		_, err := s.cfg.Audit.Append(tx, audit.Event{Kind: audit.KindRegistrationChanged, Actor: actorOf(actor),
			Registration: user, Details: details("defaultCredential", credID)})
		return err
	})
	if err != nil {
		return Registration{}, err
	}
	return ShowRegistration(s.cfg.Store, user)
}

func quote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"` }
