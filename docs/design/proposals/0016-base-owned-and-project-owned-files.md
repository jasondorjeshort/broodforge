# 0016. Base-owned and project-owned files: an update touches nothing the project wrote

Date: 2026-10-17
Status: under review

## Question

Every `update-from-hatchery` conflicts on `.claude/hooks/policy.json` and usually on `CLAUDE.md`
(the owner, 2026-10-17). The evidence is `docs/findings/2026-10-17-base-repo-defaults-and-upstream-merges.md`
(the finding): both files mix base-owned and project-owned text, the base changed them in 5 and 17
commits, and in every surveyed practice that reports fewer conflicts upstream and downstream
stopped writing the same file; frequent merges and `git rerere` shrink a conflict without removing
it. The owner suggested "a default/ directory or something" and accepted the coordinator's reading:
the cause is ownership, not location; a copy the project edits at a path the base ships still
conflicts. What is decided: which paths the base owns, which the project owns, which are mixed by
design, and the mechanism by which an update in either mode (merge for a decanted project, patch
for an adopted one, ADRs 0004 and 0010) leaves every project-owned file untouched. In ten years:
a project that never edits a base-owned file takes every update with no hand resolution, and the
ownership of a path is readable from the tree, not from a skill's prose.

## Constraints that matter here

- Both update modes must come out conflict-free for a project that never edits a base-owned file.
  Merge mode brings the base's whole tree, including the base's own state (its `CLAUDE.md` phase
  line, its handoff, ideas, findings, proposals, its policy); the rule must handle those copies.
- Promised surfaces (ADR 0013): the policy file's keys, `@docs/HANDOFF.md` in `CLAUDE.md`, the
  base line, the hook seams. An incompatible change to any names how each sibling migrates.
- The installed Claude Code (2.1.296, read in the binary) loads `./CLAUDE.md`, `./.claude/CLAUDE.md`
  and unscoped `.claude/rules/*.md` concatenated with no override; `@` imports resolve from the
  importing file, four hops, a missing target loads nothing and says nothing (ADR 0014).
- `policy.js` treats an absent file as `DEFAULTS` with no error and a missing key as the base value
  (lines 125 to 134); the base's own `policy.json` equals `DEFAULTS` plus `usage.pool`, the owner's
  machine-level account-switcher command (ADR 0011).
- `CLAUDE.md` is capped at 150 lines, a constant in `post-edit.js`; the post-edit hook caps and
  resolves `@` lines for `CLAUDE.md` by exact path (lines 196 to 202); pusher check 7 resolves `@`
  lines in `CLAUDE.md` and the handoff files only.
- `merge=ours` needs `git config merge.ours.driver true` in every clone and falls back to a plain
  conflict, silently, where it is unset; `git apply --exclude` and the `:(exclude)` pathspec need
  nothing (the finding, section 2).
- The siblings, observed 2026-10-17: `../broodforge` has no base line and a `hatchery` remote
  (merge mode); `../regenesis`, `../droplet-management` and `../dystopo` carry a `Hatchery base:`
  line (patch mode, ADR 0010; the brief called `../regenesis` decanted, the line decides).
  `CLAUDE.md` is 150, 138, 150 and 89 lines there; none has `.claude/CLAUDE.md` or a merge driver.
- Ruled out before this proposal: frequent merges and `git rerere` alone; a `default/` directory
  of copies the project then edits in place (the same both-sides edit at a new path).

## The ownership map (shared by the options)

Each option below decides the two files; all of them need the same map, so it is stated once.

| Class | Paths | Rule in both modes |
|---|---|---|
| Base-owned | `.claude/**` except `hooks/policy.json` and `settings.local.json`; `LICENSE`, `LICENSES/`; `docs/design/decisions/0001` to `0019` | The project never edits them. Patch mode patches exactly this set. A conflict here means the project edited one: base wins unless its adoption ADR or an ADR of its own names the departure (today's rule, unchanged). |
| Project-owned | `CLAUDE.md`, `.claude/hooks/policy.json`, `README.md`, `docs/HANDOFF.md`, `docs/handoff/`, `docs/design/0*.md`, decisions `0020` up, every file under `proposals/`, `findings/`, `ideas/`, `operator/`, `reference/` (except `claude-code-on-windows.md`) | The base never patches them. Merge mode restores them from `HEAD` mechanically and removes the base's additions under them (the base's own state), so the base's copies never land. |
| Mixed by design | The indexes: `docs/README.md`, `docs/design/README.md`, `docs/design/decisions/README.md`, the queue and findings READMEs; `.gitattributes`; `.claude/settings.json` in an adopted project (union of hook entries); living lines in ADRs 0001 to 0019 (ADR 0008) | Base section first, then a marker line (`## Project decisions`, `# project` in `.gitattributes`), then the project's lines; each side appends to its own section, so the hunks do not touch and git merges them. A conflict that still occurs is hand-merged, and the skill names these files as the only place one is expected. |

`decant` writes the markers; `adopt-hatchery` writes them where it adds an index. A project that
edits a base-owned file is departing: it records why (an ADR, or its adoption ADR's override list)
and takes the hand-merge on that file at each update, as today. The map is promised under ADR
0013 (a new row: carried by the `update-from-hatchery` skill; readers: `update-from-hatchery`,
`decant`, `adopt-hatchery`), so moving a path between classes is a major choice from then on.

## Options

### A. No base `policy.json`; the agreement in a base-owned file; a short project-owned `CLAUDE.md`

Policy. The base deletes `.claude/hooks/policy.json`: its behaviour is `DEFAULTS` already. The key
`usage.pool` stays in the schema (a project may still set it); when it is `null` the gate reads
`~/.claude/usage-pool.json`, same shape as the key's value (`{ "command": argv, "timeoutSeconds" }`),
absent meaning no pool. The switcher is the user's, like the grants and the usage cache that already
live under `~/.claude` (ADR 0006: "`~/.claude` is one user's"), so this is where it belongs; the
owner writes it once (an operator entry). `decant` step 5 and `adopt` step 3 create the project's
file from `policy.examples/` (a `base.json` example is added, the former base file without the
pool); a project with no choices to make may ship none. Order of work: the gate's user-scope read
lands first, the owner writes the file, then the base deletes its policy.

Agreement. The rule sections of today's `CLAUDE.md` (This agreement, Where things live, Delegation,
Debug interface, Major decisions, Usage gating, Model routing, Git, Licensing) move verbatim to a
base-owned file under the patch path list, `.claude/agreement.md`, written so no sentence carries a
project value: thresholds are "the lines in `.claude/hooks/policy.json`" (`node .claude/hooks/policy.js`
prints them), the branch is "the `Branch:` line of `CLAUDE.md`", the licence is "`LICENSE`" with
GPL-3.0-or-later named as the base default. Routing is base text since the models are pinned in
base-owned agent files. The file's first lines say it is edited in Hatchery only and that a project
that must differ states the departure in its own `CLAUDE.md` with an ADR; concatenation has no
override, so a project never contradicts a base rule silently, it names the departure.

Project file. `CLAUDE.md` becomes the project's: title, a provenance line, the import line
`@.claude/agreement.md`, `## Phase`, `Branch:`, `## Project rules`, and `@docs/HANDOFF.md`, at most
40 lines (a new constant beside the 150; the handoff's cap, for the same reason: it is loaded every
turn). The agreement takes the 150. `post-edit.js` caps both by path; check 7 resolves the import
as it resolves every `@` line today and gains one line: `FAIL` when `CLAUDE.md` has no
`@.claude/agreement.md` line, since a loader "says nothing" about a missing import and the
agreement would otherwise vanish unnoticed. `decant` step 4 writes the project file from a template
in the skill instead of editing the base's in place; `adopt` step 6 keeps its contradiction
dialogue but writes only the project file, the rules arriving by the import.

Two placements were weighed for the agreement. `.claude/CLAUDE.md`: no line in the project file can
be lost, but the documentation says "`./CLAUDE.md` or `./.claude/CLAUDE.md`" and both-present
concatenation is verified only by reading one binary, load order is the loader's (project text
before base text), the project file does not show that the agreement exists, `/init` writes
`./CLAUDE.md` without knowing about it, and neither the hook nor check 7 sees the file. An
unscoped rules file (`.claude/rules/agreement.md`) is the same mechanism and `.claude/rules/README.md`
forbids it. The `@` import: documented, relative, one hop (the handoff import is one more, within
four), order chosen by the project file, resolved by the checks that already exist, visible to the
reader, and the one-line hook the finding names as the pattern that cut conflicts elsewhere (GitLab
EE's prepend, Brave's `chromium_src`). The import is taken; the loss of the line is covered by check 7.

Update, step 5 becomes. Patch mode: `git diff $FROM hatchery/main -- .claude LICENSES
':(exclude).claude/hooks/policy.json' | git apply --3way`; `CLAUDE.md` leaves the list; the exclude
stays for ever, because a `$FROM` before the base's deletion commit would otherwise apply that
deletion to the project's file. Merge mode: `git merge --no-ff --no-commit hatchery/main`; then
restore every tracked project-owned path from `HEAD` (`git ls-files -- <paths> | xargs git checkout
HEAD --`, written out in the skill), `git rm` the base's additions under project-owned directories
(`git diff --name-only --diff-filter=A HEAD...MERGE_HEAD -- <dirs>`, index READMEs excepted), and
only then `git diff --name-only --diff-filter=U`: what remains is a mixed index or a recorded
departure. The skill's prose rule shrinks to the map above plus the two commands. The idea
`2026-10-16-update-conflict-rule-wipes-mixed-files.md` is absorbed: its mixed-file gap is the map
(a project's edits to base-owned files are departures, re-applied by its own record), its threshold
cells no longer exist in `CLAUDE.md`, and its `wc -l` gap becomes the two caps with the rule that
the project trims its own file before updating.

Meets the constraints: a project that never edits a base-owned file sees no conflict in either mode;
the base's own copies of project-owned files are restored over or removed mechanically; no promised
key changes. Cost: a coder pass on `post-edit.js`, check 7, the gate's user-scope read and harness
cases; a template edit per skill; one migration edit per sibling. Risks: a project whose
`CLAUDE.md` loses the import line runs without the agreement until the next push (check 7 catches
it there, not at the edit); a departure stated in the project file sits beside the base rule it
departs from, and the model reads both. Who else: Django `global_settings` plus a project file that
"doesn't have to define any settings"; GitLab's one-line prepend; Debian conffiles the package
never edits again (the finding, sections 1 and 4).
Exit cost: paste the agreement back into each `CLAUDE.md` and ship a `policy.json` again; one
update per sibling; delete the user-scope file. The ownership map stays useful whatever follows.
Moving parts: one new base file, one user-scope file read by the gate, one import line per
project, one cap constant, one check 7 line, one pathspec exclude, two restore commands.

Surfaces changed: none incompatibly. Added: the agreement import (`@.claude/agreement.md` in
`CLAUDE.md`; readers: the loader, `post-edit.js`, check 7), the user-scope pool file (reader: the
gate), the ownership map. ADR 0006's "the base ships one" and "the file is the project's" are
amended to "the base ships none; the file is the project's". Migration per sibling, at its next
update, in both modes the same edit: rewrite `CLAUDE.md` to the project shape (keep title, phase,
branch, project rules; replace the rule sections with the import line; drop threshold cells); keep
`policy.json` (merge mode resolves the modify/delete conflict by restoring it, the first and last
conflict on that file); `node .claude/hooks/policy.js` prints; check 7 passes. `../broodforge`:
the one merge-mode case, its 138-line fold becomes about 20. `../regenesis` and
`../droplet-management` at 150: the fold shrinks to the project lines, which must fit 40 or move to
`.claude/rules/`. `../dystopo` at 89, the same; its branch model is policy already and does not move.

### B. Keep both files; automate the project-wins rule

`.gitattributes` gains `.claude/hooks/policy.json merge=ours` and `CLAUDE.md merge=ours`;
`update-from-hatchery` step 2 runs `git config merge.ours.driver true` (idempotent, per clone) and
`decant` step 2 does the same; patch mode adds `--exclude=.claude/hooks/policy.json
--exclude=CLAUDE.md` to `git apply`. Step 5's rule for the two files becomes "nothing to do".

Meets the constraints for `policy.json` exactly: both-sides edits take the project's, a project
that never edited it takes the base's. For `CLAUDE.md` it meets the letter and breaks the point:
the base's edits to the rule sections never reach any project again (the finding: "base edits to
that file never arrive"; "the project wants the base's lines"), so the working agreement forks
silently the day this lands, and every later base rule is re-applied by hand or not at all. The
driver is per clone: a fresh clone that merges by hand gets the old conflict, silently; the guard
must permit `git config merge.ours.driver`; `.gitattributes` is itself a mixed file every sibling
extended, so the attribute lines conflict once. Nothing is done about the base's own copies of
project-owned files, so merge mode still conflicts on `docs/HANDOFF.md` every time and still needs
the clean-adds cleanup. Who else: Pro Git's `database.xml merge=ours`, cruft `skip`, copier
`_skip_if_exists` (the finding, section 3), all for files the downstream never wants upstream's
version of, which `CLAUDE.md` is not.
Exit cost: delete two attribute lines and a config key; then hand-merge every base rule edit that
was skipped while B ran, which grows with time.
Moving parts: two attribute lines, a per-clone config setting the skills must keep setting, two
`--exclude` flags, a guard allowance.

### C. A layered policy: `policy.json` base-owned, `policy.local.json` project-owned

`policy.js` reads `policy.json` (the base's, `DEFAULTS` plus `usage.pool`) and then
`policy.local.json`, each key of the local file replacing the base's; `HOOK_POLICY_FILE` names the
base file and a second seam the local one; `--cap-check` and the printer show the layered result.
Patch mode patches `policy.json` as base-owned; merge mode never conflicts on it since the project
does not edit it. `CLAUDE.md` is not addressed: C pairs with A's agreement half or with B's attribute.

Meets the policy constraint, and gives the base's `usage.pool` a home that ships to every project
on every machine: on a machine without the switcher the gate reports no reading at each HOLD deny
(ADR 0011), noise for a value that is the owner's, not the project's. The base's defaults then exist
twice, in `DEFAULTS` and in the file, and the harness must keep them equal. The promised policy row
("carried by `.claude/hooks/policy.json`") changes incompatibly: every sibling renames its file, the
`$comment` moves, ADR 0006's key table gains a layer column, pusher check 7 and the post-edit hook
read through the loader and need nothing else. Who else: dpkg conffiles plus fragments, systemd
drop-ins, Compose override, with the documented cost that an overlay goes stale silently after a
structural change (the finding, section 1).
Exit cost: fold each local file into one, revert the loader and seam, one update per sibling.
Moving parts: a second policy file per project, a second seam, a layer step in the loader with
its harness cases, a `DEFAULTS`-equals-file check.

### D. Status quo

Step 5 as written: hand-resolve by rule at every update. Cost per update: the policy conflict, the
`CLAUDE.md` conflict, in merge mode the handoff conflict and the clean-adds cleanup; the idea file's
three gaps stand (the literal rule wipes an adopted project's text in mixed files, the threshold
cells are unlisted, the 150 is unreachable for a project at the cap). The owner's report is the
measure: every update, two files. Who else: GitLab CE into EE before the repositories merged,
"days of engineering time" on "busy work" (the finding, section 4).
Exit cost: none. Moving parts: none new; the prose rule keeps growing.

## Recommendation

A. It is the only option under which a project takes every base rule edit and never sees a conflict
on either file, and it costs no promised key: the base's policy is already `DEFAULTS` in code, and
the loader already does what the agreement file needs. B fixes the policy file by the same effect
as A's deletion and breaks the agreement; C adds a second file and an incompatible rename to solve
what deleting one file solves; D is the owner's report. The ownership map is the lasting part: it
makes "what does an update touch" a table and the skill's step 5 two commands, and it handles the
base's own state in merge mode, which neither B nor C does. What would reverse it: the loader
dropping `@` imports or starting to require approval for an in-tree import (then the agreement
moves to `.claude/CLAUDE.md`, the second placement, with the hook and check 7 pointed at it);
the guard refusing the restore command with no safe spelling (then the project-owned set takes
`merge=ours`, B's mechanism, for that set only, with the driver set by the skills).

First step: the ADR (0016), amending 0006 (the base ships no policy; the user-scope pool read) and
adding the map and import rows to 0013. Then one coder brief: the gate's user-scope read, the two
caps and the agreement path in `post-edit.js`, check 7's import line, harness cases, the
`base.json` example; one tester brief for the skills (`update-from-hatchery` step 4 and 5,
`decant` 4 and 5 with the `CLAUDE.md` template, `adopt-hatchery` 3 and 6, the `handoff` note) and
the base's own migration: `.claude/agreement.md`, the short `CLAUDE.md`, delete `policy.json`,
markers in the indexes, an operator entry for `~/.claude/usage-pool.json`. Then each sibling at
its next update. The user is not needed for the decision; the owner writes the pool file.

## What would change this recommendation

- A session with both `./CLAUDE.md` and `./.claude/CLAUDE.md` shows in `/context` that only one
  loads: no effect on A (the import is used), but the fallback placement is gone.
- An in-tree `@.claude/agreement.md` import needs approval or does not resolve: take `.claude/CLAUDE.md`.
- `git checkout HEAD -- <paths>` inside a `--no-commit` merge does not clear the conflict state, or
  the guard denies every spelling of it: `merge=ours` for the project-owned set, driver set by the skills.
- The owner wants `usage.pool` per project rather than per user: the key already allows it; the
  user-scope file is dropped and the base keeps a one-key `policy.json` as a recorded departure of its own.
- A project needs to override a base rule by text, not by policy, more than rarely: the agreement
  gains a parameter (a policy key or a project slot) for that rule; the departure sentence is the stopgap.
- The siblings reach parity and the owner wants one update mode: supersede ADRs 0004 and 0010 to
  patch everywhere from a base line; the map carries over unchanged.

## Claims to verify

- Loader: Claude Code 2.1.296 concatenates `./CLAUDE.md`, `./.claude/CLAUDE.md`, unscoped
  `.claude/rules/*.md` and `CLAUDE.local.md` with no override (the finding, section 1, binary read);
  check with `/context` in a session holding both files.
- Loader: an `@` import whose path begins with `.claude/` resolves relative to `CLAUDE.md`, needs
  no approval inside the repository, counts one hop, and reaches subagents as `@docs/HANDOFF.md`
  does (code.claude.com/docs/en/memory; `.claude/skills/handoff/SKILL.md` line 8).
- Loader: no `.claude/*.md` other than `CLAUDE.md` and `rules/` is loaded on its own.
- `policy.js` lines 125 to 134: ENOENT gives `DEFAULTS` with `error: null`; `test-hooks.js` passes
  in a tree with no `policy.json` (ADR 0006 Consequences: "checked only for loading").
- `post-edit.js` lines 196 to 202 and pusher check 7 line 28: `CLAUDE.md` by exact path; a new path
  is a code change, not a policy key.
- Git: in a `--no-commit` merge, `git checkout HEAD -- <path>` on a conflicted or modify/delete path
  restores `HEAD`'s version to the index and worktree and leaves no unmerged entry; `:(exclude)`
  works in `git diff` (observed in the finding) and in `git checkout`.
- Git: the `git-guard` rules (hook specs, section 1) allow `git checkout HEAD -- <literal paths>`,
  `git rm` of literal paths, and `git config merge.ours.driver true`; which spellings are denied.
- Git: two appends separated by a heading and a blank line merge without conflict; adjacent
  appends conflict (xdiff merge needs non-touching hunks).
- Git: `git apply --3way` of a deletion hunk onto a file the project modified fails or conflicts
  rather than deleting the file; the `:(exclude)` pathspec removes the hunk from the diff.
- `merge=ours` under `git merge` (not only `git apply`) keeps the project's version where both
  sides changed and lets the base's land where only the base changed (the finding: not tested).
- `usage.pool`: a command that cannot run makes the deny say no reading, not crash (ADR 0011; the
  gate's pool code path).
- Siblings: the base line in `../regenesis`, `../droplet-management`, `../dystopo`; none in
  `../broodforge`; `CLAUDE.md` line counts 150, 138, 150, 89; `merge.ours.driver` unset in all
  (observed 2026-10-17 by `grep`, `wc -l`, `git config --get`).
- Counts: 5 base commits on `policy.json`, 17 on `CLAUDE.md` at 48a43e4 (the finding).

## Critic's objections and responses

(none yet)
