---
date: 2026-10-13
kind: feature
---

# Add a private-terms guard for commits, PR text and artifacts

A downstream project keeps names, boxes and coordinates it does not want public in a gitignored
`private-terms.local.json`. A commit hook and a scanner would check PR text and built artifacts
against that list. Hatchery would add the hook beside the existing ones in `.claude/hooks/`, and the
scanner with it. Nothing like it exists here yet.

Recorded 2026-10-11 from a downstream build-phase project; a candidate once proposal 0003 (`docs/design/proposals/0003-wave-workflow.md`) is decided.

Question: delete this as covered by ADR 0018, or keep it for scanning built artifacts? Since it was filed, `.claude/hooks/privacy-scan.js` scans every push against the never list (`~/.claude/privacy-never.txt`, literal terms) and `--text <file>` scans PR text. What is left is scanning release artifacts, which Hatchery does not build. I recommend deleting it, or `Waits: the first project with a release artifact` if you want the residue kept.
Answer:
