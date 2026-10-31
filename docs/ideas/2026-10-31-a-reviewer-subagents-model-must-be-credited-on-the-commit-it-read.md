---
date: 2026-10-31
kind: question
---

# A reviewer subagent's model must be credited on the commit it read

The attribution check (`internal/cli/worktree.go`, `05-debug-interface-git.md` "Required
trailers") wants a `Co-Authored-By` for every model with non-zero usage from its first turn. In
`s0005` on the dev root (2026-10-31) the session ran on Fable 5.1 and committed with that trailer;
the same turn then launched the `reviewer` on Opus 5.5, which only read the commit. The handoff
was refused `missing:Co-Authored-By: Claude Opus 5.5` until the commit was amended by hand to name
both. Evidence: `docs/findings/2026-10-31-milestone-1-reached-a-channel-200-session-committed-was-reviewed-and-landed.md`.

Two readings. A reviewer's reading is work on the change and earns the trailer (03-architecture,
Attribution, as written); then the session's own commit cannot pass until it is amended or
rewritten to add the reviewer's model (each commit is checked on its own against the models in use
at its committer time), and ADR 0025's review fact is lost by that amend. Or a model
whose usage comes only from a `reviewer` subagent call is not an author, and the check skips it
(not possible from today's rows: `usage_record` holds one row per model per turn with no call or
agent column, so a reviewer-only model is indistinguishable from the session's own use of it; the
parser already reads `parent_tool_use_id`, `internal/engine/parse.go`, and nothing stores it).

## Open

- Which reading the owner wants.
- Whether the engine's session should add trailers for its subagents itself, by the hook, so no
  amend is ever needed.

2026-10-31, ADR 0028 (`../design/decisions/0028-landing-a-session-on-the-default-branch.md`, Waits
on the owner): the answer is needed before the first `main` land. The `trailers` file is per turn
(ADR 0025 rule 10), so under the first reading the commits of the turn that first ran the reviewer
lack its trailer and must be amended after the review, which voids the review fact for them: one
re-review per session, not per land; the `commit-msg` hook would then add subagent trailers at
commit time so no amend is needed. Recommended: the second reading, which needs
`parent_tool_use_id` on the usage row.

Question: credit a reviewer-only model on the commit (as now), or exempt models whose usage is a
`reviewer` call's?
Answer:
