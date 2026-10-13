---
date: 2026-10-13
kind: chore
---

# Test the first install on a real device before a release is called done

A downstream project's first install on a real device, a release build with an empty profile and UI only, found
a dead end that no emulator sweep had reached. Emulator sweeps cover the paths they script; a fresh
install on hardware reaches what a user reaches first. The step belongs in the checklist of the
release and sweep skills, if those are built (see `2026-10-13-release-skill-tag-driven.md` and
`2026-10-13-sweep-skill-json-summary.md`). Nothing in Hatchery carries it yet.

Recorded 2026-10-11 from a downstream build-phase project; a candidate once proposal 0003 (`docs/design/proposals/0003-wave-workflow.md`) is decided.
