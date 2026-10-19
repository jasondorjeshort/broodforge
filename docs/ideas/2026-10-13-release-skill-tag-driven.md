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

Question: hold this until proposal 0003 is decided and a project has something to release? A release flow is a whole skill, not a pass, and Hatchery ships nothing; I recommend `Waits: proposal 0003 decided`, then a proposal from the sibling that already runs it.
Answer:
