---
date: 2026-10-13
kind: feature
---

# Add a release skill driven by a tag

A downstream project's release flow is tag-driven: the tag creates a draft release, the artifact is downloaded
from that draft and verified, the notes are hand-written, and then the release is published.
Version strings are derived from the tag. Hatchery would carry this as a skill under
`.claude/skills/release/`. It has no release flow yet. [no-check: not yet ported]

Recorded 2026-10-11 from a downstream build-phase project; a candidate once proposal 0003 (`docs/design/proposals/0003-wave-workflow.md`) is decided.
