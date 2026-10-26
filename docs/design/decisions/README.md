# Decision records

One file per decision: `NNNN-short-title.md`. A decision whose specs would push it past the
150-line cap puts them in a companion `NNNN-<topic>.md` linked from the ADR. Summaries and the
reasoning map are in `../README.md`, Decisions so far; this list is the one-line-per-file index.
Hatchery's own records are in `../../hatchery/decisions/`, never edited here (ADR 0024).

| File | What it is |
|---|---|
| `0020-harness-language-library-toolchain.md` | Claude Code CLI as one process per turn, TypeScript on Node 24, discord.js, npm; policy and prompt modes through a `PreToolUse` hook; credential shapes under the terms |
| `0021-storage.md` | SQLite store with an audit table, file configuration, temp-directory scratch |
| `0021-backup-and-restore.md` | Companion to 0021: snapshot, backup set, restore per host, rehearsal |
| `0022-session-isolation-and-credential-storage.md` | Two users and bubblewrap on the droplet, one service account on the PC, an encrypted store with the key outside the backup set, shared clones handed back as bundles; a table of spike claims |
| `0022-linux-spawn-and-handoff.md` | Companion to 0022: ownership table, sudoers, the bwrap bind list, entry wrapper, signal path, clone and handoff steps |
| `0023-language-for-the-long-term.md` | Go with a pinned toolchain, disgo (arikawa the exit), zombiezen SQLite, x/sys job objects, the standard library, one static binary per host; replaces 0020's language bundle under the owner's principle of 2026-10-16; eight conditions from the review |
| `0024-decanted-from-hatchery.md` | Provenance: the `Hatchery base:` line, the licence, the departure table of base-owned files this project edits |
| `0025-session-workflow-composition.md` | The bot's contract as a precedence header over the repository's workflow injected from a per-session copy of the canonical's default branch; repository hooks as deny-only children of the bot's hook; no push from a session; a push-time scan; the PC keeps an outside gate |
| `0025-refusals-and-checks.md` | Companion to 0025: the copy procedure, the runtime probe, the refusal texts with their lifts, the open checks and the pass that closes each |
| `0026-interrupted-sessions-held-after-a-restart.md` | Interrupted sessions loaded into memory at start so `end` works on them; every service refusal audited; one stderr line per inbound Discord event; every interaction acknowledged; the interrupted wording |
| `0027-setup-in-few-steps.md` | `broodforge setup` does every scriptable step and asks the rest; the bot makes its channels and the `Trusted` role (Manage Roles default yes, removed at the end); `setup host` creates the PC account with a generated password; `setup login`, `config export`, `setup --from`; amends 0022 |
| `0027-setup-commands.md` | Companion to 0027: the `setup` command line and its steps in order, `setup host` per host, `setup login`, `setup service`, `config export` and `setup --from` |

Template:

```
# NNNN. Title

Date: YYYY-MM-DD
Status: proposed | accepted | superseded by NNNN | superseded in part by NNNN

## Context
## Decision
## Consequences
```
