---
date: 2026-11-02
kind: question
---

# Role lists: read live or fixed at session start

Not sanity checked.

The P34 reviewer found the land gate reading the channel's `roles.land` from the session's copy of
the channel, fixed when the session started, while `worktree handoff` on the host read the live
config; the companion's lift for `land-not-allowed` ("`roles.land`, `config reload`") promises a
reload reaches an open session. The fix made the land gate read the channel's current list on every
land. The other role checks (`start`, `contribute`, `answer`) still read whatever the session or
the venue holds, and nobody has written down which it is or should be: a role added by the owner
mid-session may or may not let a member answer a prompt, depending on the door.

One rule, written once: every role check reads the channel's current configuration at the moment
of the check, so `config reload` is the one lift for every `*-not-allowed`; or every list is fixed
at session start and the lift texts say "the next session". The first is what the land now does and
what the companion promises.

Anchor: `internal/session/service.go` (the land gate's live read), the `contribute` and `answer`
checks beside it, `internal/policy/roles.go`; the lift texts in ADR 0025's and 0028's companions.

## Open

- A test per door that a reload adding the role lifts the refusal for an open session, as
  `TestLandReloadLiftsNotAllowed` does for the land.
