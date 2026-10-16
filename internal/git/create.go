// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"errors"
	"strings"

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
