// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"strings"
	"sync"

	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// Pin is what the bot records on a reviewer call it deferred (ADR 0028 Decision 3, the
// `reviewer` cell): the sha refs/sessions/<id> holds after the bundle and fetch, and the
// tree's tracked changes at that moment.
type Pin struct {
	Commit string
	// Dirty is the lines of the tree's `git status --porcelain` naming a tracked file (modified,
	// deleted, staged, renamed, unmerged), newline-joined; untracked lines are dropped. "" is a
	// clean tree.
	Dirty string
}

// bundleLocks holds one lock per session, taken by every writer of <sess>/out.bundle: Refresh
// (`worktree check`, the handoff), `worktree bundle`, the reviewer pin, and in P34 the land
// (ADR 0028 Decision 1).
var bundleLocks sync.Map // session id -> chan struct{} of capacity 1

// LockBundle takes the session's bundle lock, waiting no longer than ctx; the returned func
// lets it go.
func LockBundle(ctx context.Context, id string) (unlock func(), err error) {
	v, _ := bundleLocks.LoadOrStore(id, make(chan struct{}, 1))
	ch := v.(chan struct{})
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, &Error{Code: CodeRefusedBundle, Message: "the session's bundle lock: " + ctx.Err().Error()}
	}
}

// PinReview is the bundle at a reviewer's launch, under the session's bundle lock: BundleStep,
// then the canonical's verify and fetch into refs/sessions/<id> as Refresh does, returning the
// ref's sha and the tree's tracked changes read in the same worktree-bundle. Every step runs
// under ctx, so a hung git cannot hold the reviewer's resume.
func PinReview(ctx context.Context, b spawn.Backend, s Session) (Pin, error) {
	unlock, err := LockBundle(ctx, s.ID)
	if err != nil {
		return Pin{}, err
	}
	defer unlock()
	bundle, base, err := BundleStep(ctx, b, s)
	if err != nil {
		return Pin{}, err
	}
	sha, err := fetchBundle(ctx, s, bundle, base)
	if err != nil {
		return Pin{}, err
	}
	return Pin{Commit: sha, Dirty: TrackedChanges(bundle.Status)}, nil
}

// fetchBundle is the canonical half Refresh and PinReview share, for a bundle BundleStep wrote,
// the bundle lock held by the caller: no commits sets the
// ref back to the base; else `bundle verify`, the forced fetch into refs/sessions/<id> and the
// ref's sha.
func fetchBundle(ctx context.Context, s Session, bundle spawn.Bundle, base string) (string, error) {
	if bundle.Commits == 0 {
		if _, err := run(ctx, CodeRefusedFetch, s.Canonical, nil, "update-ref", SessionRef(s.ID), base); err != nil {
			return "", err
		}
		return base, nil
	}
	if _, err := run(ctx, CodeRefusedBundleVerify, s.Canonical, nil, "bundle", "verify", "--quiet", bundle.Path); err != nil {
		return "", err
	}
	if _, err := run(ctx, CodeRefusedFetch, s.Canonical, nil, "fetch", "--quiet", "--no-tags", "--no-write-fetch-head",
		bundle.Path, "+refs/heads/"+s.Branch()+":"+SessionRef(s.ID)); err != nil {
		return "", err
	}
	out, err := run(ctx, CodeRefusedFetch, s.Canonical, nil, "rev-parse", "--verify", "--quiet", SessionRef(s.ID)+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// TrackedChanges keeps the lines of a `git status --porcelain` output that name a tracked
// file: every line but an untracked (`??`) or ignored (`!!`) one.
func TrackedChanges(status string) string {
	var out []string
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "??") || strings.HasPrefix(line, "!!") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
