---
paths:
  - "docs/**"
  - "CLAUDE.md"
  - ".claude/agreement/**"
  - ".claude/project/**"
---

# Conventions for docs/

These load automatically when a file matching one of the `paths:` above (`docs/**`, `CLAUDE.md`,
`.claude/agreement/**`, `.claude/project/**`) is read or edited. The map of what goes where is
`docs/README.md`; the reasoning is ADR 0002.

## Size caps (enforced by the pusher, check 7; the post-edit hook warns)

- Any `docs/**/*.md`: at most 300 lines. Split by topic, never by date, and never by "part 2".
  Exempt: `docs/design/proposals/`, because a proposal carries its options, the critic's objections,
  and the responses in one closed record that nothing loads on demand. A project's legacy docs may
  be exempted by path in `.claude/hooks/policy.json` `edit.capExempt`, which the post-edit hook and
  the pusher both read; the patterns' `*` crosses `/`, so name files when a folder's subfolders
  must stay capped. Aim under 300 anyway.
- `docs/HANDOFF.md`, and each `docs/handoff/*.md` where that directory exists: at most 40 lines, at most 120 together. Detail
  goes to an idea or a finding; the handoff links.
- `CLAUDE.md`: at most 40 lines, loading at most 190 (`node .claude/hooks/policy.js --claude-check`);
  `edit.capExempt` naming `CLAUDE.md` lifts both. A new paragraph of facts goes to `docs/reference/` with a pointer.

## Indexes

- A directory under `docs/` is either indexed (its `README.md` has one line per file; adding,
  renaming, resolving or deleting a file updates that index in the same edit; the post-edit hook
  warns about an unlisted file) or listed on demand (its README carries the marker line
  `Listed on demand: node .claude/hooks/queue-list.js <dir>`, the format and never a file name).
  The marker is the hook's switch: with it, the hook checks that an edited file lists without `?`
  instead of checking an index. Only `docs/ideas/`, `docs/operator/` and `docs/findings/` may carry it
  (ADR 0017). Exception: `docs/handoff/` has no README; its index is the `@handoff/<file>.md` lines
  of `docs/HANDOFF.md` (`handoff` skill, `split.md`).
- `docs/design/README.md` lists every ADR of the tree's own `decisions/`. `docs/hatchery/` has its own
  three READMEs (`README.md`, `decisions/README.md`, `reference/README.md`), base-owned; a project
  never edits them.
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
  `area`; optional `supersedes: <file>`, `promoted_to: <path>`. The status is read by its first word
  (`resolved (ADR 0011)` reads `resolved`). Sections in order: Question, What was
  tried, Result, Next (Next may be omitted when resolved). Use the `finding` skill.
- `docs/ideas/YYYY-MM-DD-<slug>.md`: required `date`, `kind: bug | feature | question | chore`;
  optional `related: [<file>, ...]`. Search for an existing idea before adding one (its README,
  "Before adding").
- `docs/operator/YYYY-MM-DD-<slug>.md`: no frontmatter; its README says what the body carries.
- `docs/reference/<topic>.md`: required `verified: YYYY-MM-DD`, `source: <url or "observed">`. When a
  fact changes, fix it in place and bump `verified:`; do not append history.
- `docs/hatchery/decisions/NNNN-*.md`, `docs/design/decisions/NNNN-*.md` and `docs/design/proposals/NNNN-*.md`: no frontmatter; the
  `Date:` and `Status:` lines under the title are the record. Templates are in the `adr` and
  `proposal` skills; status values are in the directory READMEs.

## Personal data (ADR 0018)

- Tracked text (docs, commit messages, tests, fixtures, hook examples) never carries: a home path
  with a user segment in any spelling (drive, root, cygdrive, mnt, or a project-folder slug), an
  email or a person's name other than a listed identity (`privacy.identities`; write the role
  instead: the owner, the coordinator, a downstream project's owner), a host name, a session link,
  a recorded conversation, or a grant file's content.
- Write paths as `~`, `<repo>`, `../<dir>` or `<scratchpad>`. Briefs are not committed and keep
  absolute paths; a tracked file an agent writes from one follows this section.
- A refused push is fixed in the text and folded into the unpushed commit, never lifted; a false
  positive is the owner's (`git.md` of the agreement).

## Content rules

- `HANDOFF.md` in the single-file shape has exactly the sections Now, Next, Waiting on user, Open
  findings; in the split shape it is the map `split.md` describes. In both, the first line after
  the title says it is coordinator state, and it never contains procedure or rationale; it links.
- An `@` import line in `CLAUDE.md`, `docs/HANDOFF.md` or `docs/handoff/*.md` resolves from the
  importing file's directory, so the map writes `@handoff/<file>.md`; a target that does not exist
  loads nothing and says nothing. The post-edit hook and pusher check 7 resolve every such line.
- Reference files state facts and working commands. Findings state evidence and status. Design docs
  state decisions. Do not mix: a provisional result in a design doc reads as truth later.
- Living docs (`CLAUDE.md`, skills, agents, hooks and their examples, rules, READMEs, ideas,
  reference, the handoff) state the practice, not which project it came from: "a downstream
  build-phase project", never its name. The base is copied into projects that have no reason to
  read about their siblings. Provenance belongs in closed records (proposals, decisions, findings)
  and commit messages. One exception: the handoff and an idea may give a sibling checkout's path
  (`../<dir>`) when the next session has to find it; decant resets both.
- The code wins. A doc explains intent; it does not define behaviour. Where a doc contradicts the
  code, the doc is stale: fix it in the same change that found the conflict, and bump `verified:`.
- A change that alters behaviour, an invariant or an entry point a doc describes updates that doc
  in the same commit. Docs are the layer that historically slipped: code moved, docs went wrong.
- A path written in a doc names a file that exists; the post-edit hook warns about one that does
  not. A paragraph or list item that must name a path that is gone or not yet there carries
  `[no-check: <reason>]`; closed records (proposals, decisions, findings) are exempt by policy.
  A moved paragraph leaves a pointer behind.
- Write files that contain apostrophes or backticks with the Write or Edit tool, not a Bash heredoc
  (`docs/hatchery/reference/claude-code-on-windows.md`, Shells). The `git-guard` hook reads
  quoted text and heredoc bodies as data, so text that only mentions a git command is not denied
  unless the command line hands it to a shell.
