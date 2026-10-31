# 0028. Companion: the push forms, the `land-*` refusals and lifts, the `config check` warnings, the venue lines

Companion to `0028-landing-a-session-on-the-default-branch.md`; its Decision cites the tables here.
Every human-facing text is the owner's to confirm (04); until then it is marked `wording: owner`. The
tables are living: an edit here says so in the commit message.

## The push forms (Decision 4)

The bot parses the deferred `Bash` call's whole `tool_input.command` (the hook keeps to first words,
`../03-hook-contract.md`). `<remote>` is the project's `repo.remote`, `<default>` its `repo.defaultBranch`,
`<branch>` the session branch.

| Command | Reading |
|---|---|
| `git push`; `git push <remote>`; `git push <remote> HEAD`; `git push <remote> HEAD:<default>`; `git push <remote> <branch>`; `git push <remote> <branch>:<default>`; `git push <remote> HEAD:<branch>` | a land |
| any of the above with `--dry-run` | the land's dry run (outcome `dry-run`) |
| any of the above with `-u`, `--set-upstream`, `-q`, `--quiet`, `-v`, `--verbose` | the option is ignored; the rest is read as above |
| `broodforge session land`; `broodforge session land --dry-run` | a land; its dry run; any other word is `land-bad-push` |
| `--force`, `--force-with-lease`, `--force-if-includes`, `-f`, `--delete`, `-d`, `--mirror`, `--tags`, `--all`, `--no-verify`, `--prune`, any other option | `land-bad-push` naming the form that works |
| a remote other than `<remote>`; a refspec whose source is not `HEAD` or `<branch>`, or whose destination is not `<default>` or `<branch>`; more than one refspec | `land-bad-push` naming the form that works |
| a command that does not begin `git push` or `broodforge session land` (`cd x && git push`, a script) | not a land request: the tree's `pre-push` refuses it, naming the two forms |

## Refusals and lifts (Decisions 1, 3, 4, 5)

Each refusal is an exit-1 line from `session land` (`error: <code>: <text>`), a `deny` on the push word
(`Denied by Broodforge: not landed: <text>; <lift>`) and a venue line; `wording: owner` for every text.

| Code | When | Text (wording: owner) | Lift |
|---|---|---|---|
| `land-not-allowed` | the driver has no role in the channel's `roles.land`, or the channel has none | `Not landed: landing on <default> needs a land role in this channel.` | The owner adds a land role to this channel (`roles.land`, `config reload`); nobody in the session or venue. |
| `land-not-allowed` (subagent) | the request came from a subagent the copy's `git.push.subagents` does not name, or before the V12 probe passes | `Not landed: in this session the push word is the driver process's; a subagent leaves the work committed and reports.` | The project's authors on the default branch (the list); the next session has it. |
| `land-unreviewed` | the review fact does not name the tip | `Not landed: review: none since <sha>[; the last reviewer launch failed: <commitError>]. Commit, launch the reviewer as a call on its own, then push again.` | The session: commit, launch the reviewer on the tip, land again. |
| `land-not-fast-forward` | `<remote>/<default>` is not an ancestor of the tip | `Not landed: <default> moved by <n> commits since this session branched. Fetch <remote>, rebase this tree onto <remote>/<default> (the driver or the runner), review again, then push again.` | The session, as the text says; at the end the fallback. |
| `land-canonical-diverged` | the canonical's default branch and the remote's have diverged | `Not landed: the host's copy of <default> and <remote> have diverged; the owner reconciles it.` | The owner: push the canonical's commits, or `branch -f` / `reset --keep` onto `<remote>/<default>` (`../05-debug-interface-git.md`, The fetch at session start). |
| `land-rejected` | the remote refused the push | `Not landed: push rejected by the remote: <git's one line>. If <default> moved, push again; if the remote refuses the branch, the owner changes the remote or the project's handoff.` | A race: land again. Protection or `denyCurrentBranch`: the owner (the remote's configuration, `handoff` `pull-request`, or the warning's lifts below). |
| `land-bad-push` | the command is not a landing form | `Not landed: Broodforge lands this session's branch onto <default> as a fast-forward, nothing else. Use git push, or git push --dry-run.` | Inside the session: issue a landing form. |
| the handoff's own codes | `handoff-refused-*`, `handoff-failed-*` | the existing texts (ADR 0025 companion; `../05-debug-interface-git.md`) | as today |
| a project hook's deny of a land request | the copy's `PreToolUse` child denied the `Bash` call before the land ran (only when the request did not begin with a landing form; a landing form returns before the chain) | the companion's child-deny text | The project's authors on the default branch; the next session. |
| `pre-push` (the tree) | a push that was not a land request reached git | `Broodforge: session <id> does not push from this tree. To land, run git push as a call on its own, or broodforge session land.` | Inside the session: the two forms. |

## `config check` warnings (Decision 2)

- A `main` project whose channel has no `land` list: `warning: <project> <channel>: handoff main with
  no roles.land; every session there will fall back to <handoffFallback>.` Lift: `roles.land`.
- A path remote that is a non-bare repository (`rev-parse --is-bare-repository` false) whose
  `symbolic-ref HEAD` is the default branch and whose `receive.denyCurrentBranch` is unset or `refuse`:
  `warning: <project>: <remote> is a checkout with <default> checked out; git refuses every push to it
  (receive.denyCurrentBranch). Lifts: a bare repository beside the checkout that the owner pulls from;
  receive.denyCurrentBranch updateInstead on the checkout (still refused while its tree or index
  differs from HEAD); GitHub as the remote.` The first land repeats the inspection and the warning in
  its outcome line. Wording: owner.

## Venue lines (Decision 1; wording: owner)

| Event | Text |
|---|---|
| a done `main` land | `Landed 3 commits on main at <sha> (a reviewer ran on <sha>).` never "reviewed" |
| a done land on a `push` or `pull-request` project | `Pushed 3 commits to <branch>.`; `Pushed 3 commits to <branch>; pull request #<n> <url>.` |
| the end's fallback | `Not landed on main: <refusal text>. Handed off as <branch> instead.` |
| a land from the host | `Landed from the host: <outcome line>.`; the `--as` value is shown as claimed, never as the actor |
| `/session land` placement | a slash command alone, or a header button beside `[End session]`: the owner's call |
