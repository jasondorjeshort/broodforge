// SPDX-License-Identifier: GPL-3.0-or-later

package discord

import (
	"regexp"
	"strings"
)

// Masked is what a credential-shaped token shows as in every posted text (03 Data flow, step 6).
const Masked = "[masked]"

// KeyPrefix is the Anthropic key shape the inbound scan refuses (03 Data flow, step 1).
const KeyPrefix = "sk-ant-"

var (
	// hidden are Unicode format characters (zero-width space and joiners, bidi marks) and
	// nonspacing marks (U+034F and the combining accents), which show as nothing or ride on a
	// letter and could hide the prefix from a plain match.
	hidden = regexp.MustCompile(`[\p{Cf}\p{Mn}]`)
	// keyShape is the prefix with a key-length tail anywhere, glued to a word or inside
	// Discord's `_` and `__` markup included; or the prefix with any tail where no letter or
	// digit precedes it. So `task-ant-colony`, glued and short, is not a key.
	keyShape = regexp.MustCompile(regexp.QuoteMeta(KeyPrefix) + `[A-Za-z0-9_\-]{8,}|(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(KeyPrefix))
)

// HasKey reports whether an inbound message carries an Anthropic key, once hidden characters
// are removed: the prefix with a key-length tail anywhere, or the prefix after no letter or digit.
func HasKey(text string) bool { return keyShape.MatchString(hidden.ReplaceAllString(text, "")) }

var (
	// sk- prefixed keys (Anthropic's sk-ant-, OpenAI-style sk-), at least eight characters on.
	skKey = regexp.MustCompile(`\bsk-[A-Za-z0-9_\-]{8,}`)
	// A bearer token after its scheme word.
	bearer = regexp.MustCompile(`(?i)\b(bearer)\s+[A-Za-z0-9._~+/=\-]{8,}`)
	// OAuth and forge tokens with a known prefix: GitHub's, Google's, Slack's.
	oauth = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|ya29\.[A-Za-z0-9_\-]{20,}|xox[abprs]-[A-Za-z0-9\-]{10,})`)
	// A candidate long token: letters, digits and the characters base64, JWTs and keys use.
	longRun = regexp.MustCompile(`[A-Za-z0-9+/=_\-.]{41,}`)
	hexOnly = regexp.MustCompile(`^[0-9a-fA-F]+$`)
)

// Mask replaces credential-shaped tokens with [masked] (03 Data flow, step 6; the prior-art
// finding, L1): sk- prefixed keys, bearer and OAuth tokens, and strings over 40 characters that
// are not pure hex, so 40- and 64-character SHAs pass. A path is not a token: a run holding `/`
// or `.` is masked only when one of its segments between them is over 40 and not pure hex.
func Mask(s string) string {
	s = skKey.ReplaceAllString(s, Masked)
	s = bearer.ReplaceAllString(s, "$1 "+Masked)
	s = oauth.ReplaceAllString(s, Masked)
	return longRun.ReplaceAllStringFunc(s, func(run string) string {
		if !strings.ContainsAny(run, "/.") {
			if hexOnly.MatchString(run) {
				return run
			}
			return Masked
		}
		for _, seg := range strings.FieldsFunc(run, func(r rune) bool { return r == '/' || r == '.' }) {
			if len(seg) > 40 && !hexOnly.MatchString(seg) {
				return Masked
			}
		}
		return run
	})
}
