---
paths:
  - "docs/**"
---

# Conventions for docs/

These load automatically when a file under `docs/` is read or edited. The map of what goes where is
`docs/README.md`; the reasoning is ADR 0002.

## Size caps (enforced by the pusher, check 7; the post-edit hook warns)

- Any `docs/**/*.md`: at most 300 lines. Split by topic, never by date, and never by "part 2".
  Exempt: `docs/design/proposals/`, because a proposal carries its options, the critic's objections,
  and the responses in one closed record that nothing loads on demand. Aim under 300 anyway.
- `docs/HANDOFF.md`: at most 40 lines. Detail goes to `docs/backlog.md` or a finding; the handoff links.
- `docs/backlog.md`: soft cap 150 lines. Near it, delete the oldest ideas and turn stale deferred
  items into findings with `status: superseded`.

## Indexes

- Every directory under `docs/` has a `README.md` with one line per file. Adding, renaming, resolving,
  or deleting a file updates that index in the same edit. The post-edit hook warns about an unlisted file.
- `docs/design/README.md` lists every ADR; `docs/findings/README.md` shows status per file.

## Frontmatter

- `docs/findings/YYYY-MM-DD-<slug>.md`: required `date`, `status: open | resolved | superseded`,
  `area`; optional `supersedes: <file>`, `promoted_to: <path>`. Sections in order: Question, What was
  tried, Result, Next (Next may be omitted when resolved). Use the `finding` skill.
- `docs/reference/<topic>.md`: required `verified: YYYY-MM-DD`, `source: <url or "observed">`. When a
  fact changes, fix it in place and bump `verified:`; do not append history.
- `docs/design/decisions/NNNN-*.md` and `docs/design/proposals/NNNN-*.md`: no frontmatter; the
  `Date:` and `Status:` lines under the title are the record. Templates are in the `adr` and
  `proposal` skills; status values are in the directory READMEs.

## Content rules

- `HANDOFF.md` has exactly the sections Now, Next, Waiting on user, Open findings, and the first line
  after the title says it is coordinator state. It never contains procedure or rationale; it links.
- Reference files state facts and working commands. Findings state evidence and status. Design docs
  state decisions. Do not mix: a provisional result in a design doc reads as truth later.
- Write files that contain apostrophes or backticks with the Write or Edit tool, not a Bash heredoc
  (open finding `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`). The same goes for any text
  that mentions a destructive git command: the `git-guard` hook matches the Bash command text, so a
  heredoc containing `git push --force` is denied even though nothing runs it.
