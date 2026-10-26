// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// Create is clone at session start (ADR 0022 companion): in the canonical, refs/sessions/<id>
// is created at the default branch's commit (refused when it exists), then the backend's
// worktree-create clones the canonical into the session's tree and checks out the session
// branch at that commit. A failed clone deletes the ref again. It returns the base commit.
func Create(ctx context.Context, b spawn.Backend, s Session) (string, error) {
	if err := s.check(); err != nil {
		return "", err
	}
	out, err := run(ctx, CodeCreate, s.Canonical, nil, "rev-parse", "--verify", "--quiet", "refs/heads/"+s.DefaultBranch+"^{commit}")
	if err != nil {
		return "", &Error{Code: CodeCreate, Message: s.Canonical + " has no branch " + s.DefaultBranch}
	}
	base := strings.TrimSpace(out)
	// An empty old value: the ref must not exist yet, so a session id is never reused.
	if _, err := run(ctx, CodeCreate, s.Canonical, nil, "update-ref", SessionRef(s.ID), base, ""); err != nil {
		return "", err
	}
	if err := b.WorktreeCreate(s.Worktree(base)); err != nil {
		_, _ = run(ctx, CodeCreate, s.Canonical, nil, "update-ref", "-d", SessionRef(s.ID), base)
		return "", err
	}
	return base, nil
}

// CloneCanonical is a project's canonical checkout created by `broodforge setup` (ADR 0027):
// `git clone` of url into dir as the bot's own account, the account's credential helpers
// cleared so the askpass helper is the one answer, with the stored git credential (none when
// token is empty) on that one process, as the push carries it (PushEnv). A refusal is an
// *Error with code clone and git's most telling line.
func CloneCanonical(ctx context.Context, askpass, url, dir string, token []byte) error {
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return &Error{Code: CodeClone, Message: err.Error()}
	}
	_, err := run(ctx, CodeClone, parent, PushEnv(askpass, token), "-c", "credential.helper=", "clone", "--quiet", "--", url, dir)
	return err
}

// OriginURL is the URL of remote in the checkout at dir.
func OriginURL(ctx context.Context, dir, remote string) (string, error) {
	out, err := run(ctx, CodeClone, dir, nil, "remote", "get-url", remote)
	return strings.TrimSpace(out), err
}

// SetOriginURL points remote of the checkout at dir to url.
func SetOriginURL(ctx context.Context, dir, remote, url string) error {
	_, err := run(ctx, CodeClone, dir, nil, "remote", "set-url", remote, url)
	return err
}

// HeadBranch is the branch HEAD names in the checkout at dir.
func HeadBranch(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, CodeClone, dir, nil, "symbolic-ref", "--short", "HEAD")
	return strings.TrimSpace(out), err
}

// Base is the session's base commit as the canonical knows it: the merge base of
// refs/sessions/<id> and the default branch. Before the first fetch the ref is the base itself;
// after one it is the fetched tip, whose fork point from the default branch is the base again,
// also when the driver has had commits amended since. No store column holds the base.
func Base(ctx context.Context, s Session) (string, error) {
	out, err := run(ctx, CodeRefusedBundle, s.Canonical, nil, "merge-base", SessionRef(s.ID), "refs/heads/"+s.DefaultBranch)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// SessionTip is refs/sessions/<id> in the canonical and its committer time: the session's last
// commit as of the last bundle (`worktree check` and the handoff fetch it; before the first,
// the base). The review fact of ADR 0025 rule 7 is measured against it. found is false only when
// the canonical holds no such ref (the session was handed off); a canonical that cannot be read
// (moved, not a repository) is an error, CodeSessionRef.
func SessionTip(ctx context.Context, s Session) (sha string, at time.Time, found bool, err error) {
	if err := s.check(); err != nil {
		return "", time.Time{}, false, err
	}
	// for-each-ref prints nothing and exits 0 for a missing ref, and fails for an unreadable
	// repository, which rev-parse --quiet would not tell apart.
	out, err := run(ctx, CodeSessionRef, s.Canonical, nil, "for-each-ref",
		"--format=%(refname) %(objectname) %(committerdate:unix)", SessionRef(s.ID))
	if err != nil {
		return "", time.Time{}, false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || f[0] != SessionRef(s.ID) {
			continue // a ref below refs/sessions/<id>/ would match the pattern too
		}
		n, perr := strconv.ParseInt(f[2], 10, 64)
		if perr != nil {
			return "", time.Time{}, false, &Error{Code: CodeSessionRef, Message: SessionRef(s.ID) + " is not a commit: " + line}
		}
		return f[1], time.Unix(n, 0).UTC(), true, nil
	}
	return "", time.Time{}, false, nil
}

// Discard undoes Create for a session whose start failed after the clone: the tree and its
// bundle go through the backend's worktree-remove, then refs/sessions/<id> is deleted from the
// canonical. Both are attempted; the errors are joined.
func Discard(ctx context.Context, b spawn.Backend, s Session) error {
	if err := s.check(); err != nil {
		return err
	}
	rerr := b.WorktreeRemove(s.Dir())
	_, derr := run(ctx, CodeCreate, s.Canonical, nil, "update-ref", "-d", SessionRef(s.ID))
	return errors.Join(rerr, derr)
}
