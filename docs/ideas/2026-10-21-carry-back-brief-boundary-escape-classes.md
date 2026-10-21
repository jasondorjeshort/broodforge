---
date: 2026-10-21
kind: chore
related: [.claude/skills/brief/SKILL.md, docs/design/decisions/0024-decanted-from-hatchery.md]
---

# Carry back: a reviewer brief for a path or name boundary lists the escape classes

Anchor: `.claude/skills/brief/SKILL.md`, the reviewer bullet (departure row in ADR 0024).

For a boundary that decides on paths or names (a hook, a guard, a sandbox list), this project's
reviewer briefs list the escape classes to try: short names, link and junction, UNC and device
prefixes, streams, drive-relative, `~` and shell-style prefixes, whitespace, case, patterns with
metacharacters first; and ask the reviewer to check how the other side (the engine, the shell)
resolves the same string before it compares. The project's PreToolUse hook needed three review
rounds because the first brief named none of these. The base's own guard and privacy scan are
boundaries of the same kind.

Plan: carry back the four lines (`carry-back` skill, Send) without the project's pass name in the
last sentence; then the row in ADR 0024 records `carry-back/brief-escape-classes at <sha7>`.
