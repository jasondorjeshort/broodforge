// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// Error is a refusal with a stable code (05-debug-interface.md, Git handoff); the CLI maps it
// to its error line. Commits holds the failing commits of handoff-refused-attribution; Hits
// and Checked the hits of handoff-refused-privacy and how many commits the scan read.
type Error struct {
	Code    string
	Message string
	Commits []CommitCheck
	Hits    []PrivacyHit
	Checked int
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Codes this package refuses with; the handoff codes are 05-debug-interface-git.md's.
const (
	CodeBadSession          = "bad-session"
	CodeAlreadyHandedOff    = "already-handed-off"
	CodeCreate              = "worktree-create"
	CodeRefusedBundle       = "handoff-refused-bundle"
	CodeRefusedBundleVerify = "handoff-refused-bundle-verify"
	CodeRefusedFetch        = "handoff-refused-fetch"
	CodeRefusedAttribution  = "handoff-refused-attribution"
	CodeRefusedPrivacy      = "handoff-refused-privacy"
	CodeFailedPush          = "handoff-failed-push"
	CodeFailedPullRequest   = "handoff-failed-pull-request"
	CodeNoCredential        = "no-git-credential"
	CodeCommitMsg           = "commit-msg"
	CodeSessionRef          = "session-ref" // refs/sessions/<id> cannot be read from the canonical
)

var sessionIDRE = regexp.MustCompile(`^s[0-9a-z]{4,}$`)

// ValidSessionID reports whether id is a session id: `s` and at least four base-36 digits.
func ValidSessionID(id string) bool { return sessionIDRE.MatchString(id) }

// SessionID is the id of the session row: `s` plus the base-36 row id, at least four digits
// (03-architecture.md, Git).
func SessionID(row int64) string {
	d := strconv.FormatInt(row, 36)
	if len(d) < 4 {
		d = strings.Repeat("0", 4-len(d)) + d
	}
	return "s" + d
}

// Branch is the session branch, broodforge/<instanceName>/<session id>, local and remote.
func Branch(instance, id string) string { return "broodforge/" + instance + "/" + id }

// SessionRef is the canonical's ref for the session, refs/sessions/<id> (ADR 0022).
func SessionRef(id string) string { return "refs/sessions/" + id }

// SessionDir is <root>/sessions/<id>.
func SessionDir(root, id string) string {
	return filepath.ToSlash(filepath.Join(root, "sessions", id))
}

// TrailersPath is <root>/sessions/<id>/home/trailers, the lines the commit-msg hook appends.
func TrailersPath(sessionDir string) string {
	return filepath.ToSlash(filepath.Join(sessionDir, "home", "trailers"))
}

// Session is what the git steps of one session need: where it lives and its project's repo.
type Session struct {
	Root          string
	ID            string
	Instance      string // instanceName
	Canonical     string // repo.canonical
	Remote        string // repo.remote
	DefaultBranch string // repo.defaultBranch
}

// Dir is the session's directory.
func (s Session) Dir() string { return SessionDir(s.Root, s.ID) }

// Branch is the session branch.
func (s Session) Branch() string { return Branch(s.Instance, s.ID) }

// Worktree is the spawn backend's operand for the session at the base commit.
func (s Session) Worktree(base string) spawn.Worktree {
	return spawn.Worktree{Canonical: s.Canonical, Session: s.Dir(), BaseBranch: s.DefaultBranch,
		Base: base, Branch: s.Branch()}
}

func (s Session) check() error {
	if !ValidSessionID(s.ID) {
		return &Error{Code: CodeBadSession, Message: strconv.Quote(s.ID) + " is not a session id"}
	}
	if s.Root == "" || s.Instance == "" || s.Canonical == "" || s.Remote == "" || s.DefaultBranch == "" {
		return &Error{Code: CodeBadSession, Message: s.ID + ": a session needs its root, instance, canonical, remote and default branch"}
	}
	return nil
}

// run runs the bot's git in the canonical: the bot's environment minus the names spawn strips
// (a session's or another repository's git variables), never a terminal prompt, plus extra.
// A failure is an *Error with code and git's most telling stderr line.
func run(ctx context.Context, code, dir string, extra []spawn.Var, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = spawn.Environment(os.Environ(), append([]spawn.Var{{Name: "GIT_TERMINAL_PROMPT", Value: "0"}}, extra...))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := tellingLine(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", &Error{Code: code, Message: "git " + subcommand(args) + ": " + msg}
	}
	return stdout.String(), nil
}

// subcommand is the git subcommand after any -c options, for messages.
func subcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" {
			i++
			continue
		}
		return args[i]
	}
	return ""
}

// tellingLine is the first line of git's stderr that says what went wrong: a push's first
// lines name the remote ("To <url>") and hints come before errors, so both are skipped.
func tellingLine(stderr string) string {
	first := ""
	for _, l := range strings.Split(stderr, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if first == "" {
			first = l
		}
		if strings.HasPrefix(l, "To ") || strings.HasPrefix(l, "hint:") {
			continue
		}
		return l
	}
	return first
}
