---
date: 2026-11-02
kind: feature
related: [2026-10-31-broodforge-deploys-broodforge.md, 2026-10-31-a-changelog-channel-for-landings.md, 2026-10-29-broodforge-works-on-itself-overnight.md]
---

# A land runs the project's checks before the fast-forward

Moonshot, not sanity checked. A `main` land today needs a reviewer to have read the tip (the
review fact) and nothing else: the bot never builds or tests what it lands. Review is judgment;
the check line is evidence. A project could name its checks (`"land": {"checks": ["CGO_ENABLED=0
go build ./...", "go test ./..."]}` or a script), and `git.Land` would run them on the bundled sha
in a scratch worktree of the canonical between the review fact and the push, refusing
`land-checks-failed` with the first failing command and its last lines. The same runs would give
the deploy idea its "passes build and the goldens" gate for free and the changelog idea a green mark
per landing.

Costs: a land becomes minutes, not seconds, and runs the project's own code on the host as the bot
(the WIP idea of 2026-11-01 already worries about `core.fsmonitor` running as the bot); on Linux the
bwrap sandbox of ADR 0022 bounds it, on the PC nothing does. A failing check mid-session is the
session's to fix (the lift is the text); at the end it falls back like every other refusal.

Anchor: `internal/git/handoff.go` (`Land`, between the fact and step 5); `internal/config/types.go`
(a per-project block); ADR 0028's companion (a new row); `docs/design/03-architecture-git.md`
(Landing).

## Open

- Whether the checks run as the bot on the PC at all, or the PC stays review-only.
- Caching: a sha already checked by a session's own `runner` (the audit shows the command) need not
  run again.
