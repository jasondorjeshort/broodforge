# Decision records

One file per decision: `NNNN-short-title.md`. A decision whose specs would push it past the
150-line cap puts them in a companion `NNNN-<topic>.md` linked from the ADR. Summaries and the
reasoning map are in `../README.md`, Decisions so far; this list is the one-line-per-file index.

| File | What it is |
|---|---|
| `0001-licensing.md` | GPL-3.0-or-later by default, LGPL for linkable components (Hatchery) |
| `0002-repository-layout.md` | Where knowledge, state, procedures, rules and hooks live (Hatchery) |
| `0002-hook-specs.md` | Companion to 0002: the hooks and the status line (Hatchery) |
| `0004-adopt-hatchery.md` | How an existing project adopts the practice (Hatchery) |
| `0006-hook-policy-file.md` | Per-project hook choices in `.claude/hooks/policy.json` (Hatchery) |
| `0007-queues-triage-and-one-review.md` | Delete-when-done queues, triage, one review per change (Hatchery) |
| `0008-adr-numbers-and-what-decant-keeps.md` | Hatchery owns 0001 to 0019; what a decant keeps (Hatchery) |
| `0009-adr-numbers-when-both-tests-pass.md` | ADR numbers when a tree is both joined and has a base line (Hatchery) |
| `0020-harness-language-library-toolchain.md` | Claude Code CLI as one process per turn, TypeScript on Node 24, discord.js, npm; policy and prompt modes through a `PreToolUse` hook; credential shapes under the terms |
| `0021-storage.md` | SQLite store with an audit table, file configuration, temp-directory scratch |
| `0021-backup-and-restore.md` | Companion to 0021: snapshot, backup set, restore per host, rehearsal |
| `0022-session-isolation-and-credential-storage.md` | Two users and bubblewrap on the droplet, one service account on the PC, an encrypted store with the key outside the backup set, shared clones handed back as bundles; a table of spike claims |
| `0022-linux-spawn-and-handoff.md` | Companion to 0022: ownership table, sudoers, the bwrap bind list, entry wrapper, signal path, clone and handoff steps |

Template:

```
# NNNN. Title

Date: YYYY-MM-DD
Status: proposed | accepted | superseded by NNNN

## Context
## Decision
## Consequences
```
