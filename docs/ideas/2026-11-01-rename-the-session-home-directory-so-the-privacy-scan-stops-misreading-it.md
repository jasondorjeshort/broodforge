---
date: 2026-11-01
kind: chore
related: [2026-10-30-carry-back-privacy-scan-at-commit-time.md]  # that one moves the scan earlier; this removes a false-positive class at its source
---

# Rename the session `home` directory so the privacy scan stops misreading it

Recorded from the owner's conversation of 2026-11-01, not sanity checked: nobody has read the
anchor against the claim yet.

Every session has a directory named `home` beside its tree (ADR 0022, the engine's `HOME` for the
session), and the push scan's home rule reads any `/home/<segment>` as a home directory with a user
segment. On 2026-10-31 and 2026-11-01 it refused `<sess>/home/land-result` in a critic review and
`/home/trailers` and `/home/hook.log` in P38's docs and code, each lifted by an owner allow entry
for the segment. More code will name files in that directory, so the allow list grows by one entry
per file name. Renaming the directory (`sess-home`, `engine-home`, or similar) ends the class at
the source and lets the allow entries go.

Anchor: `docs/design/decisions/0022-session-isolation-and-credential-storage.md` (the directory's
name and role); `internal/hook/hook.go` (the trailers path beside the tree); `.claude/hooks/policy.json`
(`privacy.allow`, the entries this would retire); `.claude/hooks/privacy-scan.js` (the home rule).

## Open

- Whether the engine needs the directory to be literally named `home`, or only `HOME` to point at it.
- Existing sessions on the live instance hold the old name; a migration or a tolerant lookup.
