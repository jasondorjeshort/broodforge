---
date: 2026-10-16
kind: chore
related: []
---

# Fixtures need a path rewrite before they move into the repo

The fixtures recorded in the S1 run carry the PC username in paths: the `init` event's `cwd` and
the hook command in the recorded settings. No email or credential appears. `docs/design/07-spike-plan.md`
(the fixtures paragraph) says a person reads each fixture before it moves to `fixtures/engine/<name>/`;
that read is slower than it needs to be, and the public repo would still hold the name, which the
pusher's personal-data check would then flag.

Want: a rewrite step in the move that replaces the home directory and username with a fixed
placeholder (for example `/home/user`), run before the person reads the file, so the read looks only
for what a script cannot know. The fake engine and its tests must not depend on the original paths.

## Open

- Whether the rewrite lives in the spike's `setup.ts` or in a small standalone script.
- Whether any test asserts on a recorded `cwd`; if so, it should compare to the placeholder.
