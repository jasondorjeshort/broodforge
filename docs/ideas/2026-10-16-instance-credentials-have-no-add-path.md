---
date: 2026-10-16
kind: feature
---

# Instance credentials have no command to add them

The credential store allows a credential with no user: `credential.user_id` is nullable for the
instance's own credentials (the Discord bot token, the git identity's credential). But
`broodforge credential add` requires `--user` and a registered user
(`internal/cli/credential.go`, `internal/credentials/service.go`), so the CLI has no path to add one.

## Want

`credential add` accepts an instance credential: a way to say "no user" (for example `--instance`,
exclusive with `--user`) that stores the row with `user_id` null, and `credential list` shows it.
P8 (git identity) and P12 (bot token) in `docs/design/08-implementation-plan.md` need it; ADR 0022's
credential section (`docs/design/decisions/0022-session-isolation-and-credential-storage.md`) says
where each kind lives.

## Open

- Which kinds an instance credential may be (api-key only, or a provider token kind of its own).
- Whether the flag shape belongs in `docs/design/05-debug-interface.md` before the pass builds it.
