---
date: 2026-10-17
status: resolved
area: hooks / git on Windows
promoted_to: docs/design/decisions/0002-hook-specs.md
---

# Git for Windows shows a pre-push hook's stderr only, so the scan's HIT lines were lost

## Question

Why did a push refused by the pre-push hook (ADR 0018) show no `HIT` lines on a Windows clone?

## What was tried

Git 2.56 for Windows, a scratch repository with `core.hooksPath` set to `.claude/hooks/git` and an
outgoing commit that carried a home path. `git push` was run and the output read. The hook ran
`privacy-scan.js --pre-push`, which prints `HIT` lines and the last line on stdout and notices on
stderr (scan contract, section 3).

## Result

The push was refused correctly (the scan exited 1), but the terminal showed none of the stdout
lines: only stderr from the hook reaches the user on this git. Ending the scan call in the hook with
`>&2` sends the `HIT` lines and the last line to stderr, and they show. The fix is in
`.claude/hooks/git/pre-push`; the behaviour is described in the pre-push section of the hook specs.
Not checked on Linux or macOS git, where stdout is expected to show; the redirect is harmless there.

## Next

Omitted: resolved. One related open item stays outside this finding: `privacy.identities` in
`.claude/hooks/policy.json` stays `[]` until the owner chooses the repository identity (real or
noreply); the `identity` class is off, with its notice, until then.
