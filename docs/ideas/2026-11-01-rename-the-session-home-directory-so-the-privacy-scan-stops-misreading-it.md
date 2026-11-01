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

Question: rename the session home directory (ADR 0022) at the cost of a design-wide edit and a migration, or keep it
and add an allow entry per file name? The name is fixed in code (`grep -rn '"home"' --include=*.go internal`: the
session service's `stageHome`, `internal/git/git.go`, `internal/hook/hook.go`, `internal/cli/engine.go`, and the check in
`internal/hook/facts.go` that matches `sessions/<id>/home` with `.claude` below it) and in the design docs (`grep -rln`
for the directory name under `docs/design/`); sessions on the live instance hold the old name, so the rename needs a
tolerant lookup or migration, and an ADR 0022 amendment. Recommendation: rename (`engine-home`), because the allow list
is the owner's and grows with every new file name in that directory (`privacy.allow` in `.claude/hooks/policy.json`
already holds `land-result`, `trailers` and `hook.log` for it), and the engine only needs `HOME` to point at it;
schedule it after P34 to P37 (they touch the same session files) as one `coder` pass with the amendment done first by
the `architect`. Otherwise keep the name and accept the entries.
Answer:
