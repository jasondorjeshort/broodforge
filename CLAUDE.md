# Broodforge working agreement

Decanted from Hatchery; provenance and departures: `docs/design/decisions/0024-decanted-from-hatchery.md`.
Where work stands right now (coordinator state; subagents follow their brief, not this):

@docs/HANDOFF.md

## Phase

Build: the Go bundle of ADR 0023, pass by pass per `docs/design/08-implementation-plan.md`. The
design docs in `docs/design/` stay authoritative; a change to one goes through the proposal loop.

Branch: main

@.claude/agreement/this-agreement.md
@.claude/agreement/where-things-live.md
@.claude/agreement/delegation.md
@.claude/agreement/debug-interface.md
@.claude/agreement/major-decisions.md
@.claude/agreement/usage-gating.md
@.claude/agreement/model-routing.md
@.claude/agreement/one-review.md
@.claude/agreement/git.md
@.claude/agreement/licensing.md

## Project rules

- Hooks: `node .claude/hooks/test-hooks.js` after any change under `.claude/hooks/`.
- Go, from the repository root: `CGO_ENABLED=0 go build ./...`, then the check line and the test line of
  `.claude/rules/go.md` (`go vet ./... && go tool staticcheck ./... && go tool exhaustive ./...`;
  `CGO_ENABLED=1 go test -race ./...`, gcc per `docs/reference/go-on-windows.md` on the PC).
- Path-scoped rules beyond the base's `docs.md`: `.claude/rules/go.md` (Go files, `go.mod`) and
  `.claude/rules/hooks.md` (`.claude/hooks/`).
- The push scan's placeholders (`user`, `alice`, `bob`) and service accounts (`broodforge`,
  `broodforge-engine`) are `privacy.allow` in `.claude/hooks/policy.json`; a person's name never is.
- A main-session push first runs pusher check 5 (`node .claude/hooks/privacy-scan.js` over the
  outgoing range, then a read of each outgoing commit for names, numbers, addresses and
  identifiers) and quotes the result before pushing.
