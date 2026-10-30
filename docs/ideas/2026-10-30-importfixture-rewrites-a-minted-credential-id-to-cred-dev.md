---
date: 2026-10-30
kind: chore
related: [2026-10-30-real-turns-on-the-dev-root-seed-fixtures.md]
---

# importfixture rewrites a minted credential id to cred-dev

The `dev-read` golden (`tests/discord/golden/dev-read/`) was captured from the dev instance and
carries that instance's minted credential id (`cred-` plus ten lowercase alphanumerics) in the
session-started event, three outbox posts and the transcript's Credential line. The renderer never
looks credentials up, so replay works. The cost is churn: a recapture on another instance, or after
the credential is re-added, changes four places for no change in behaviour.

`importfixture --events` (`internal/tools/importfixture/main.go`, `rewriteEvents`) already rewrites
the captured session id to `s0001` with `rewriteToken`. The reviewer's proposal: rewrite every
whole-token id of the minted shape to `cred-dev` the same way, following `cred-scratch` in the
`burst` golden and the plan's `cred-<dev>`. Doing it re-blesses `dev-read` once.

Plan: tester, one pass. In `rewriteEvents`, beside the session-id rewrite, replace each whole-token
`cred-` plus ten lowercase alphanumerics with `cred-dev` (a regexp; `cred-scratch` and other
non-minted ids stay). Add a case to `TestEventsRewritesEverySessionID` or a sibling test in
`internal/tools/importfixture/main_test.go`: a capture holding a minted id in an event and a post
comes out with `cred-dev` and no minted id. Then run `discord render --golden
tests/discord/golden/dev-read --update` and read the diff: exactly the credential id lines change
(session-started event, three outbox posts, the transcript's Credential line). Verify: `go test
./internal/tools/importfixture ./internal/discord`, and `golden-sessions.sh` reports four `ok`.
