// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jasondorjeshort/broodforge/internal/session"
)

// /credential (04, Credentials): add, list, remove, default and login, all ephemeral. An API
// key never arrives as a command option: `add kind:api-key` answers with a modal (a form only
// the bot receives), and the key comes back in the modal's submission.

const (
	// CodeNotYours refuses a credential id that is not the caller's.
	CodeNotYours = "not-yours"
	// ReplyModal is the reply that opens the API-key form; its text is the modal's custom id.
	ReplyModal = "modal"
	// APIKeyModalPrefix starts the API-key modal's custom id; the label follows it.
	APIKeyModalPrefix = "bfm:api-key:"
	// APIKeyField is the custom id of the modal's one text input.
	APIKeyField = "key"
	// MaxLabel bounds a label, so the modal's custom id stays within Discord's 100 characters.
	MaxLabel = 80
)

// Credential kinds as /credential add names them, the store's own.
const (
	kindSubscription = "subscription"
	kindAPIKey       = "api-key"
)

// credential runs /credential <subcommand> for the caller.
func (a *Adapter) credential(in Inbound, sl Slash) (Reply, error) {
	if a.cfg.Credentials == nil {
		return Reply{}, a.refuse(in, "command", CodeNotBuilt, "/"+sl.Name+" is not built in the dry-run adapter yet")
	}
	if _, reg, err := a.svc.Registered(in.User); err != nil {
		return Reply{}, err
	} else if !reg {
		return Reply{}, a.refuse(in, "command", session.CodeUnregistered, notRegisteredText)
	}
	sub := ""
	if len(sl.Args) > 0 {
		sub = sl.Args[0]
	}
	switch sub {
	case "add":
		label := strings.TrimSpace(sl.Options["label"])
		if label == "" || utf8.RuneCountInString(label) > MaxLabel || len(APIKeyModalPrefix)+len(label) > 100 {
			return Reply{}, a.refuse(in, "command", CodeBadArgs, "/credential add takes label:<text>, 1 to 80 characters")
		}
		switch sl.Options["kind"] {
		case kindSubscription:
			info, err := a.cfg.Credentials.Add(in.User, kindSubscription, label, nil)
			if err != nil {
				return Reply{}, a.credentialRefusal(in, err)
			}
			return ephemeral(subscriptionReadyText(info.ID)), nil
		case kindAPIKey:
			return Reply{Reply: ReplyModal, Text: APIKeyModalPrefix + label}, nil
		}
		return Reply{}, a.refuse(in, "command", CodeBadArgs, "/credential add takes kind:subscription or kind:api-key")
	case "list":
		infos, err := a.cfg.Credentials.List(in.User)
		if err != nil {
			return Reply{}, a.credentialRefusal(in, err)
		}
		return ephemeral(credentialListText(infos)), nil
	case "remove", "default", "login":
		id := strings.TrimSpace(sl.Options["id"])
		info, err := a.ownCredential(in, id)
		if err != nil {
			return Reply{}, err
		}
		switch sub {
		case "remove":
			if _, err := a.cfg.Credentials.Remove(in.User, id); err != nil {
				return Reply{}, a.credentialRefusal(in, err)
			}
			return ephemeral(credentialRemovedText(id)), nil
		case "default":
			if _, err := a.svc.SetDefault(in.User, id, in.User); err != nil {
				return Reply{}, a.credentialRefusal(in, err)
			}
			return ephemeral(credentialDefaultText(id)), nil
		}
		if info.Kind != kindSubscription {
			return Reply{}, a.refuse(in, "command", CodeBadArgs, id+" is an API key; only a subscription logs in") // not 04
		}
		if a.cfg.Config != nil && a.cfg.Config().Instance.Relay.LoginByDirectMessage {
			// ADR 0022's relay (Discord 50007 and all) is not built: the host login stands in.
			return Reply{}, a.refuse(in, "command", CodeNotBuilt, "the direct-message login relay is not built yet; "+loginOnHostText(id))
		}
		return ephemeral("For " + id + ", " + loginOnHostText(id)), nil // not 04
	}
	return Reply{}, a.refuse(in, "command", CodeBadArgs, "/credential takes add, list, remove, default or login")
}

// ownCredential is the caller's credential id, refused when it is not one of theirs: a
// credential is only ever named by its owner (R-39).
func (a *Adapter) ownCredential(in Inbound, id string) (CredentialInfo, error) {
	if id == "" {
		return CredentialInfo{}, a.refuse(in, "command", CodeBadArgs, "give the credential as id:<id>")
	}
	infos, err := a.cfg.Credentials.List(in.User)
	if err != nil {
		return CredentialInfo{}, a.credentialRefusal(in, err)
	}
	for _, c := range infos {
		if c.ID == id {
			return c, nil
		}
	}
	return CredentialInfo{}, a.refuse(in, "command", CodeNotYours, id+" is not one of your credentials.") // not 04
}

// credentialRefusal audits a refusal of the credential store or the session service like the
// adapter's own; any other error passes as it is.
func (a *Adapter) credentialRefusal(in Inbound, err error) error {
	var se *session.Error
	if !errors.As(err, &se) {
		return err
	}
	if se.Code == session.CodeUnregistered {
		return a.refuse(in, "command", se.Code, notRegisteredText)
	}
	return a.refuse(in, "command", se.Code, se.Message)
}

// IsAPIKeyModal reports whether a modal's custom id is the API-key form's.
func IsAPIKeyModal(customID string) bool { return strings.HasPrefix(customID, APIKeyModalPrefix) }

// apiKeySubmitted is the API-key modal's submission: the key is stored for the caller under
// the label the custom id carries. The key-paste scan does not apply; this is the form 04 sends
// people to. Nothing but the id and the key's last four characters is answered or audited.
func (a *Adapter) apiKeySubmitted(in Inbound, value string) (Reply, error) {
	label, ok := strings.CutPrefix(in.CustomID, APIKeyModalPrefix)
	label = strings.TrimSpace(label)
	if !ok || label == "" {
		return Reply{}, a.refuse(in, "modal", CodeBadArgs, "not the API-key form")
	}
	if a.cfg.Credentials == nil {
		return Reply{}, a.refuse(in, "modal", CodeNotBuilt, "/credential is not built in the dry-run adapter yet")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return Reply{}, a.refuse(in, "modal", CodeBadArgs, "the form held no key")
	}
	info, err := a.cfg.Credentials.Add(in.User, kindAPIKey, label, []byte(value))
	if err != nil {
		var se *session.Error
		if errors.As(err, &se) {
			return Reply{}, a.refuse(in, "modal", se.Code, se.Message)
		}
		return Reply{}, err
	}
	return ephemeral(apiKeyStoredText(info.ID, lastFour(value))), nil
}

func lastFour(s string) string {
	r := []rune(s)
	if len(r) <= 4 {
		return ""
	}
	return string(r[len(r)-4:])
}
