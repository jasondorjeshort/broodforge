# 0024. Decanted from Hatchery

Date: 2026-10-21
Status: accepted

Hatchery base: 8d6ee28e50b4693b17f90576ea5945f962dd8665
Mechanism: Hatchery ADR 0016 (update by patch over the base-owned set).

## Context

Broodforge was decanted from Hatchery on 2026-10-14, before Hatchery's ADR 0016, and took base
changes by merging `hatchery/main` until 2f98979. This record is written at its first update under
ADR 0016 (`update-from-hatchery`, 2f98979..dc896c3), which is also the one-time migration of
`docs/hatchery/decisions/0016-migration.md`, section 4: `CLAUDE.md` to the template, Hatchery's
records out of this tree's `docs/design/decisions/` and `docs/reference/`, this record.

## Decision

- Base-owned files taken from the base: the set in `.claude/hooks/ownership.json`
  (`node .claude/hooks/ownership.js --check` lists tracked paths the map leaves unclassified); the
  policy file and `settings.local.json` stay the project's.
- Merged: `.claude/settings.json` (no project hook entries; the base's usage gate, guard and
  post-edit hook as shipped), `.gitignore` (the project's). `CLAUDE.md` written from the template
  (title, provenance line, `@docs/HANDOFF.md`, `## Phase`, `Branch:`, ten agreement imports,
  `## Project rules`); the project's rules are the Go and hooks rules under `.claude/rules/` and
  the test commands in `## Project rules`.
- Added: `docs/hatchery/` from the base. Hatchery's records 0001 to 0015 and the Windows reference,
  copies kept by the decant, are removed from `docs/design/decisions/` and `docs/reference/`; the
  copies this project had edited went to carry-back ideas first (`docs/ideas/`, 2026-10-21). The
  project's Go facts from the Windows reference moved to `docs/reference/go-on-windows.md`.
  Citations in living files now name `docs/hatchery/`; closed records keep their text.
- Numbers: this project's ADRs are 0020 to 0023 and this one, and keep their numbers; the next is
  0025 (the `adr` skill numbers from the next free number in this directory).
- Replaced sections: none.
- License: kept GPL-3.0-or-later and Hatchery ADR 0001 (`docs/hatchery/decisions/0001-licensing.md`).
  SPDX headers: every source file (`.claude/rules/go.md`).
- Docs over the 300-line cap: none exempted (`edit.capExempt` is empty).
- Policy (`.claude/hooks/policy.json`): usage holds kept (five-hour 60, weekly 75, no deny line);
  the push scan's placeholder list moved from `git.push.privacyAllow` to `privacy.allow`
  unchanged; `privacy.identities` empty and `git.hooksPath` null until the owner names the commit
  identity (the `decant` skill's identity question); `usage.pool` null; `update.exclude` empty.

## Departures

A living table: one row per base-owned path the project edits and per agreement section it
replaces. The project wins on a rowed path until Hatchery has the change; then base wins and the
row goes. Carry-back is `carry-back/<slug> at <sha7>` once sent (`carry-back` skill), the Hatchery
commit that took the change, `none: <reason>`, or an idea file in this project for a request with
no diff (ADR 0019).

| Path | Reason | Carry-back |
|---|---|---|
| `.claude/skills/adr/SKILL.md` | "Amend without superseding": a dated `## Amendments` entry for a correction that keeps the Decision's core, as ADRs 0020 and 0022 carry | `carry-back/adr-amendments at 09496ad` |
| `.claude/skills/brief/SKILL.md`; `.claude/agents/reviewer.md` | The reviewer bullet lists the escape classes for a path or name boundary | `carry-back/brief-escape-classes at ce41cba` |
| `.claude/agents/reviewer.md` (Rules); `.claude/skills/brief/SKILL.md` (reviewer bullet) | A reviewer finding is a failing test, so the fix is make-it-pass and the case stays as the regression guard; prose findings were fixed and forgotten (evidence 2026-10-30: seven reviews, each with a real defect) | `docs/ideas/2026-10-30-carry-back-a-reviewer-finding-is-a-failing-test.md` |
| `CLAUDE.md` (Project rules); the commit skill's Pushing section | A main-session push runs pusher check 5 in full, the scan and the human read of each outgoing commit, and quotes the result before the push; the base's rule 20 scans every push but only the `pusher` reads | `carry-back/commit-skill-main-session-push-check-5 at de8173f` |
| `.claude/hooks/git-guard.js` | A quoted string that contains git named git, so rule 19 refused expansion stages; this project's fix `5288955` was dropped at the update to `dc896c3` and main runs the base guard | `carry-back/git-guard-quoted-git at fffd564` |
| `.claude/hooks/imports.js` | A backslash, `:` or device-name import escaped the tree on Windows; main runs the base copy | `carry-back/imports-js-backslash-device-outside at 1bf8c6f` |

Given up to base at this update, with what the base's version does not carry:

- `.claude/agents/pusher.md`, `.claude/hooks/git-guard.js`, `policy.js`, `test-hooks.js`, the three
  `policy.examples/`: the project's rule 20 and check 5a, subsumed by Hatchery ADR 0018 (the
  privacy scan, `privacy-scan.js`, pusher check 5).
- `.claude/hooks/shell-parse.js` and the guard's quoted-word test (a quoted string that merely
  contains git does not name git): not in the base; dropped, to be carried back
  (`docs/ideas/2026-10-21-carry-back-quoted-git-does-not-name-git.md`).
- `.claude/skills/commit/SKILL.md`: a main-session push ran pusher check 5a first; the guard's
  rule 20 now scans every push from every caller, so only the human read of a main-session push
  is gone. Kept as a `CLAUDE.md` project rule, to be carried back
  (`docs/ideas/2026-10-21-carry-back-main-session-push-runs-check-5-read.md`).
- `.claude/rules/README.md`: the count of rules this tree ships; `CLAUDE.md`, `## Project rules`,
  names them instead.

## Consequences

Updates use `update-from-hatchery`: the patch from the base line above to `hatchery/main` over the
base-owned set, three-way; a conflict on a rowed path keeps the project text, any other is base
wins. Costs and follow-ups specific to this project: the quoted-git false denies are back until
the carry-back lands; the push scan flags the commit author's address until the owner fills
`privacy.identities` (and then `git.hooksPath` `.claude/hooks/git` with `core.hooksPath`), so the
next push waits on that answer.
