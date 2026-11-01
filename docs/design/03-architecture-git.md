# Architecture: git

Companion to `03-architecture.md`, which points here from its Git section (moved by topic on
2026-10-31 when ADR 0028 added landing; the text of the first three sections is the one that stood
there, unchanged). Trees, branches, attribution and the handoff follow ADR 0022 and its companion
(`decisions/0022-linux-spawn-and-handoff.md`); landing follows ADR 0028
(`decisions/0028-landing-a-session-on-the-default-branch.md`); the CLI side is
`05-debug-interface-git.md`'s; what people see is `04-ux.md`'s.

## Trees, branches, attribution

Session id: `s` plus the base-36 session row id, at least four digits (`s0001`), unique per instance. Branch:
`broodforge/<instanceName>/<session id>` (two instances pushing one remote cannot collide; the instance name matches the project-id
pattern); tree: `sessions/<session id>/tree`; the canonical's ref: `refs/sessions/<session id>` (ADR 0022); the remote branch equals
the local one; the base is the canonical's default branch after the fetch at session start (ADR 0022 companion, Clone step 0).
Subagent worktrees are the engine's own, inside the clone; only the session branch is bundled, so subagent work reaches the push once
merged or cherry-picked into it (R-31); unmerged subagent branches are listed in the end-of-session summary and removed with the tree.
Attribution (R-32, R-33). Author and committer of every commit: the driver's registration,
through the four git identity variables (ADR 0022). Trailers, one `Co-Authored-By: <name> <email>` each, in this order:
every other participant who wrote a message before the commit's committer time, in join order; then every model with
usage in the session that the bot had recorded before that time (`model_seen`: the turn's start for the turn's own
model, the launch for a model an `Agent` call pins, else its first `assistant` line; a model with usage the record
missed is owed from its first turn's start, since the usage row lands at the turn's end; ADR 0025 rule 10 as amended
2026-10-31), by display name with `noreply@anthropic.com`. Models used by subagents are contributors too: `modelUsage` from the engine's result
covers them (ADR 0020, Amendments), so every model with non-zero usage in the session gets a line (an Opus agent that
ran a Haiku subagent yields both). The display name comes from the map in `policy` (model id prefix to name); an
unmapped id is used verbatim. A model that only compacted is a co-author: the stream cannot tell compaction from work.
Two helps, neither the enforcement: the system prompt append says once that the `commit-msg` hook adds the trailer
lines and lists none (a list there is stale from turn two, ADR 0025 rule 10), and `worktree-create` sets
`core.hooksPath` in the clone to the Broodforge-owned hooks: `commit-msg` appends any line of the `trailers`
file in the session's `home` directory under `sessions/<id>` that is missing (`pre-push`: ADR 0025 rule 4 as amended
by ADR 0028: a refusal naming the two landing forms, since the tree never pushes). The bot rewrites the `trailers`
file at each first sighting of a model and at every process start, and the `pre-tool-use` hook appends an `Agent`
call's pinned model to it before the subagent starts, so a pinned subagent's first commit carries what the check owes.
Two writers, each by temp file and rename: a line one adds can be lost to the other's rename until the next rewrite
restores it; the check owes from the record, not the file, so the loss is a refusal at the end, never a wrong credit.

## The check at handoff

The enforcement is the bot's check at handoff (ADR 0022 companion, step 3): for each commit in
`<base>..refs/sessions/<id>`, author and committer equal the driver and the trailer set computed
for the commit's time is present (extra trailers are fine). A failing commit is listed with what
is missing, the push is refused, and the session stays open in waiting-for-input so the driver
can ask the agent to amend and `end` again. The bot never rewrites history (ADR 0022 defers it).

## End of session

End of session: interrupt a running turn; when `handoff` is `pull-request` and the credential is
usable, run one closing engine turn with a fixed prompt that returns the pull request title and
body (skipped on `push` projects, on an auth error, or when the credential is marked needs-login;
the fallback title is `Session <id>: <first message, truncated>`, the summary the body); then
bundle, verify, fetch, check, and hand off per the project's `handoff` (default `pull-request`):
`push` pushes `refs/sessions/<id>` to `refs/heads/<session branch>` on the remote; `pull-request`
pushes the same and opens the request through the forge's HTTP API with the project's git
credential in that one call (`forge: "github"` only in the first version; title and body are a 04
flow); `main` runs the land below and, when the land refuses `land-not-fast-forward` or
`land-rejected`, falls back to `handoffFallback` (ADR 0028). Before the bundle, a dirty tree is committed through the spawn backend as one `WIP` commit with the session's attribution, and the end-and-discard keeps the commits on `refs/aborted/<id>` in place of a handoff (`04-ux-session-end.md`). No commits: the tree is removed and the session says so. The tree is removed only after
`done` on the push or the request (R-35); a refused or failed handoff keeps it and `status` lists it with its unpushed count.

## Landing (ADR 0028)

One procedure, `Land` in `git`, run by the bot from the canonical and never opening the tree, that the
end, `/session land`, `session land <id>` and the push word from inside a session all call: fetch and
fast-forward the canonical's default branch as session start does; `Refresh`, `Collect`, the
attribution check and the push scan; for a `main` target the gate (the driver's role in the channel's
`roles.land`; the review fact naming `refs/sessions/<id>` exactly) and `merge-base --is-ancestor
<remote>/<default> refs/sessions/<id>`, false being `land-not-fast-forward` (the bot never rebases:
the session rebases in its tree, reviews again, lands again); the push
`refs/sessions/<id>:refs/heads/<default>` with the askpass helper, never `--force`, audited `land`;
then the canonical's default branch and tracking ref fast-forwarded to the landed sha, the tree
untouched so the engine's next `--resume` sees the files it left. `refs/sessions/<id>` stays, so the
next land hands off only what came after. On a `push` project a land pushes the session branch; on a
`pull-request` project the first land opens the request and later lands fast-forward its branch. The
land, `worktree check` and the reviewer pin share one lock per session (all three write
`<sess>/out.bundle`); a running turn does not block a land. The review fact: `session` answers the
`reviewer` cell's `defer` by bundling under that lock and recording the sha and the tree's cleanliness
on the call before resuming it `allow`; the fact holds when the newest `ok` reviewer row with a
non-empty result and no `commitError` names the tip (`03-hook-contract.md`; ADR 0025 rule 7 as
amended 2026-10-31). The push word: a `Bash` `git push` or `broodforge session land` is the `land`
cell's `defer`; `session` parses the whole command against the ADR 0028 companion's table, runs the
land as the driver (a subagent's request is refused until P37's probe passes), and resumes `allow`
with `updatedInput` carrying the outcome line, or `deny` with the refusal. The actor of a venue land
is the Discord user, of a host land `host` (any `--as` value recorded as claimed); on the PC the
audit cannot tell the owner's terminal from a session's Bash (ADR 0025 rule 8 as amended). Refusal
codes, lifts and texts: `decisions/0028-land-refusals-and-push-forms.md`.
