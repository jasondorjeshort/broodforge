// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
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

// FetchTimeout is the deadline of the fetch at session start and its fast-forward.
const FetchTimeout = 30 * time.Second

// FetchResult is the fetch at session start (ADR 0022 companion, Clone at session start, step
// 0). Commit is the default branch's commit the session starts from; Fetched when the fetch
// succeeded (zero when it did not); Ahead the local branch's commits past the remote's, left
// alone; Stale a fetch or fast-forward that failed or ran past its deadline, Error git's
// telling line or the reason, and the start goes on from the local branch.
type FetchResult struct {
	Commit  string
	Fetched time.Time
	Ahead   int
	Stale   bool
	Error   string
}

// Fetch is step 0 of clone at session start, in the canonical: `git -c credential.helper=
// fetch --quiet --no-tags --no-write-fetch-head <remote> +refs/heads/<db>:refs/remotes/<remote>/<db>`
// with the push's askpass and token (PushEnv), under FetchTimeout, then the local branch and the
// fetched ref compared with `merge-base --is-ancestor` both ways. Behind is fast-forwarded
// (`merge --ff-only` while HEAD is the branch, else `update-ref <ref> <new> <old>`); equal is
// nothing; ahead is left alone with its count. Neither containing the other is an *Error with
// code canonical-diverged naming both commits. A fetch or fast-forward that fails or times out
// is a Stale result and no error. The canonical is never reset or cleaned.
func Fetch(ctx context.Context, s Session, askpass string, token []byte) (FetchResult, error) {
	if err := s.check(); err != nil {
		return FetchResult{}, err
	}
	branch := "refs/heads/" + s.DefaultBranch
	tracking := "refs/remotes/" + s.Remote + "/" + s.DefaultBranch
	fctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	stale := func(err error) (FetchResult, error) {
		r := FetchResult{Stale: true, Error: messageOf(err)}
		if errors.Is(fctx.Err(), context.DeadlineExceeded) {
			r.Error = "no answer from " + s.Remote + " before the deadline"
		}
		// The commit the session starts from, read outside the spent deadline (the caller's
		// may be spent too).
		rctx, rcancel := context.WithTimeout(context.WithoutCancel(ctx), waitDelay)
		defer rcancel()
		if out, err := run(rctx, CodeFetch, s.Canonical, nil, "rev-parse", "--verify", "--quiet", branch+"^{commit}"); err == nil {
			r.Commit = strings.TrimSpace(out)
		}
		return r, nil
	}
	if _, err := run(fctx, CodeFetch, s.Canonical, PushEnv(askpass, token), "-c", "credential.helper=", "fetch", "--quiet",
		"--no-tags", "--no-write-fetch-head", s.Remote, "+"+branch+":"+tracking); err != nil {
		return stale(err)
	}
	fetched := time.Now().UTC()
	local, err := run(fctx, CodeFetch, s.Canonical, nil, "rev-parse", "--verify", "--quiet", branch+"^{commit}")
	if err != nil {
		return stale(&Error{Code: CodeFetch, Message: s.Canonical + " has no branch " + s.DefaultBranch})
	}
	remote, err := run(fctx, CodeFetch, s.Canonical, nil, "rev-parse", "--verify", "--quiet", tracking+"^{commit}")
	if err != nil {
		return stale(&Error{Code: CodeFetch, Message: s.Remote + " has no branch " + s.DefaultBranch})
	}
	local, remote = strings.TrimSpace(local), strings.TrimSpace(remote)
	if local == remote {
		return FetchResult{Commit: local, Fetched: fetched}, nil
	}
	behind, err := isAncestor(fctx, s.Canonical, local, remote)
	if err != nil {
		return stale(err)
	}
	if behind {
		head, herr := HeadBranch(fctx, s.Canonical)
		if herr == nil && head == s.DefaultBranch {
			_, err = run(fctx, CodeFetch, s.Canonical, nil, "merge", "--ff-only", "--quiet", tracking)
		} else {
			_, err = run(fctx, CodeFetch, s.Canonical, nil, "update-ref", branch, remote, local)
		}
		if err != nil {
			return stale(err)
		}
		return FetchResult{Commit: remote, Fetched: fetched}, nil
	}
	ahead, err := isAncestor(fctx, s.Canonical, remote, local)
	if err != nil {
		return stale(err)
	}
	if !ahead {
		return FetchResult{Commit: local, Fetched: fetched}, &Error{Code: CodeCanonicalDiverged,
			Message: s.Canonical + " " + s.DefaultBranch + " is at " + local + ", " + s.Remote + " at " + remote + "; neither contains the other"}
	}
	out, err := run(fctx, CodeFetch, s.Canonical, nil, "rev-list", "--count", tracking+".."+branch)
	if err != nil {
		return stale(err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return stale(&Error{Code: CodeFetch, Message: "git rev-list --count: " + strings.TrimSpace(out)})
	}
	return FetchResult{Commit: local, Fetched: fetched, Ahead: n}, nil
}

// isAncestor is `git merge-base --is-ancestor a b` in dir: exit 0 true, exit 1 false, anything
// else an *Error with git's telling line.
func isAncestor(ctx context.Context, dir, a, b string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "merge-base", "--is-ancestor", a, b)
	cmd.Env = spawn.Environment(os.Environ(), []spawn.Var{{Name: "GIT_TERMINAL_PROMPT", Value: "0"}})
	cmd.WaitDelay = waitDelay
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &ee) && ee.ExitCode() == 1 && ctx.Err() == nil:
		return false, nil
	}
	msg := tellingLine(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	return false, &Error{Code: CodeFetch, Message: "git merge-base: " + msg}
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

// SessionTip is refs/sessions/<id> in the canonical: the session's last commit as of the last
// bundle (`worktree check`, the handoff and a reviewer's launch fetch it, Refresh; before the
// first, the base). The review fact of ADR 0025 rule 7 is measured against it. found is false
// only when the canonical holds no such ref (the session was handed off); a canonical that
// cannot be read (moved, not a repository) is an error, CodeSessionRef.
func SessionTip(ctx context.Context, s Session) (sha string, found bool, err error) {
	if err := s.check(); err != nil {
		return "", false, err
	}
	// for-each-ref prints nothing and exits 0 for a missing ref, and fails for an unreadable
	// repository, which rev-parse --quiet would not tell apart.
	out, err := run(ctx, CodeSessionRef, s.Canonical, nil, "for-each-ref",
		"--format=%(refname) %(objectname)", SessionRef(s.ID))
	if err != nil {
		return "", false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 || f[0] != SessionRef(s.ID) {
			continue // a ref below refs/sessions/<id>/ would match the pattern too
		}
		return f[1], true, nil
	}
	return "", false, nil
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
