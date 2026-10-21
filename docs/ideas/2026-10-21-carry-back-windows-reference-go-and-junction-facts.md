---
date: 2026-10-21
kind: question
related: [docs/reference/go-on-windows.md, docs/hatchery/reference/claude-code-on-windows.md]
---

# Carry back: language-neutral facts from this project's Windows reference edits

Anchor: `docs/hatchery/reference/claude-code-on-windows.md`. This project's copy of the base's
Windows reference gained a "Go on this PC" section on 2026-10-16 and a finding link as the npm
evidence. The copy was removed in the update to Hatchery `dc896c3` (ADR 0024); the Go section now
lives in `docs/reference/go-on-windows.md`, the project's own.

Most of it is this project's toolchain (MSYS2 gcc for `-race`, the `CGO_ENABLED` lines). Two facts
are about Windows and agents, not Go, and may belong in the base's file:

- A behavioural antivirus scanner quarantined a fresh unsigned build and the Git tools its process
  tree had run, so tools vanished the instant a run ended; check the scanner's quarantine first.
- Directory junctions are not followed by every link resolver (Go's `filepath.EvalSymlinks` is
  one); a check that compares resolved paths must resolve junctions itself.

Question: does the base want either fact in its Windows reference (a request with no diff; the
base's coordinator writes it from this text)?
Answer:
