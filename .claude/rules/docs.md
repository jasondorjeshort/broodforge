---
paths:
  - "docs/**"
  - "CLAUDE.md"
---

# Conventions for docs/

These load automatically when a file under `docs/` or `CLAUDE.md` is read or edited. The map of what goes where is
`docs/README.md`; the reasoning is ADR 0002.

## Size caps (enforced by the pusher, check 7; the post-edit hook warns)

- Any `docs/**/*.md`: at most 300 lines. Split by topic, never by date, and never by "part 2".
  Exempt: `docs/design/proposals/`, because a proposal carries its options, the critic's objections,
  and the responses in one closed record that nothing loads on demand. Aim under 300 anyway.
- `docs/HANDOFF.md`: at most 40 lines. Detail goes to an idea or a finding; the handoff links.
- `CLAUDE.md`: at most 150 lines. A new paragraph of facts goes to `docs/reference/` with a pointer.

## Indexes

- Every directory under `docs/` has a `README.md` with one line per file. Adding, renaming, resolving,
  or deleting a file updates that index in the same edit. The post-edit hook warns about an unlisted file.
- `docs/design/README.md` lists every ADR; `docs/findings/README.md` shows status per file.
- An index is the one list a doc may carry. Prose never inventories files, fields, routes or
  counts ("the twelve commands", "a 400-line file"): every ordinary commit moves the figure, and a
  later session then "corrects" it. Write the rule that generates the list instead.

## Queues: one file per item, deleted when done

- `docs/ideas/` and `docs/operator/` are queues: one dated file per item, named
  `YYYY-MM-DD-<slug>.md`, deleted in the commit that completes it, never ticked. Git has the
  history; a directory holding only its README is the resting state. Each README owns its format.
- An idea ends in at most one block: `Plan:` (settled, pass-sized, ready to build), `Question:`
  with a blank `Answer:` (the owner's choice; they fill the answer in), or `Waits:` (the owner's
  deferral to a named trigger). No block means untriaged. The `triage` skill writes blocks; only
  the owner writes `Waits:` or deletes an idea for not being wanted.

## Frontmatter

- `docs/findings/YYYY-MM-DD-<slug>.md`: required `date`, `status: open | resolved | superseded`,
  `area`; optional `supersedes: <file>`, `promoted_to: <path>`. Sections in order: Question, What was
  tried, Result, Next (Next may be omitted when resolved). Use the `finding` skill.
- `docs/ideas/YYYY-MM-DD-<slug>.md`: required `date`, `kind: bug | feature | question | chore`;
  optional `related: [<file>, ...]`. Search for an existing idea before adding one (its README,
  "Before adding").
- `docs/operator/YYYY-MM-DD-<slug>.md`: no frontmatter; its README says what the body carries.
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
- The code wins. A doc explains intent; it does not define behaviour. Where a doc contradicts the
  code, the doc is stale: fix it in the same change that found the conflict, and bump `verified:`.
- A change that alters behaviour, an invariant or an entry point a doc describes updates that doc
  in the same commit. Docs are the layer that historically slipped: code moved, docs went wrong.
- A path written in a doc names a file that exists; the post-edit hook warns about one that does
  not. A paragraph that must name a path that is gone or not yet there carries
  `[no-check: <reason>]`. A moved paragraph leaves a pointer behind.
- Write files that contain apostrophes or backticks with the Write or Edit tool, not a Bash heredoc
  (open finding `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`). The `git-guard` hook reads
  quoted text and heredoc bodies as data, so text that only mentions a git command is not denied
  unless the command line hands it to a shell.
