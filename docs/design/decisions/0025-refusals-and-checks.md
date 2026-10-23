# 0025. Companion: the copy procedure, the runtime probe, refusals and lifts, open checks

Companion to `0025-session-workflow-composition.md`; its rules cite the sections here. The refusal
texts are human-facing and the owner's to confirm (04); the checks table is closed by the pass named
on each row. Both tables are living: an edit here says so in the commit message.

## The copy procedure (rule 5)

1. `git -C <canonical> ls-tree --name-only <defaultBranch> -- CLAUDE.md .claude`. `git archive` exits
   128 on a pathspec that matches nothing (observed, git 2.56), so the bot never names a path it has
   not listed.
2. Nothing listed: no archive; `instructions.md` holds the header alone; no `agents.json`, no
   `hooks.json`; the spawn adds `--append-system-prompt-file` only; `session show` prints
   `workflow: none (<defaultBranch> at <commit> has no CLAUDE.md or .claude)`.
3. Else `git -C <canonical> archive <defaultBranch> -- <the listed paths>`, the tar stream extracted
   by the bot itself (`archive/tar`, no tar binary) into `workflow/<session id>/`, then
   `instructions.md`, `agents.json` and `hooks.json` rendered there as the ADR says. The commit and
   date in the header are `git rev-parse <defaultBranch>` and that commit's committer date.
4. `BROODFORGE_WORKFLOW` is set by `spawn` on the engine process (inherited by the hook and its
   children) to the absolute path of `workflow/<session id>`; it is absent when `inject` is `none`
   or nothing was copied. The hook runs no children when it is absent or `hooks.json` is absent.

## The runtime probe (rule 6)

For each `hooks.json` entry, the first word of `command` after the host shell's word splitting (a
quoted word is one word), with `${CLAUDE_PROJECT_DIR}` and `${CLAUDE_PROJECT_DIR:-.}` read as the copy:

- `node`, `python`, `python3`, `bash`, `sh`, `pwsh`, `deno`, `bun`: run `<word> --version` where the
  children run; not found or non-zero fails.
- a word holding a path separator or ending in `.js`, `.py`, `.sh`, `.ps1`, `.exe`: the file must
  exist (a relative word against the copy); a `.js` also needs `node`, a `.py` `python`, a `.ps1`
  `pwsh`, each probed as above.
- anything else: the word must resolve on the children's `PATH` (`exec.LookPath`).

Where: the instance account's environment on the PC; `spawn preflight` inside the sandbox on the
droplet. At `config check` a failure is an error naming the command and the word; at session start
it refuses the start (table below).

## Refusals and lifts

| Where | Text | Lift |
|---|---|---|
| Child deny (hook result; trace `denied: refused by the project's hook <name>: <reason>`) | `Denied by Broodforge: refused by the project's hook <name>: <reason>. The project's rules on its default branch decide this.` | The project's authors change the rule on the default branch; the next session has it. |
| Child over budget | `Denied by Broodforge: the project's hook <name> timed out after <n> seconds. The project's authors fix it on the default branch, or the owner removes <event> from workflow.hooks for this project.` | As the text says; `<n>` is the smaller of the hook's own `timeout` and ten. |
| Child cannot start | `Denied by Broodforge: the project's hook <name> did not start: <first stderr line, else the OS error>. The project's authors fix it on the default branch, or the owner removes <event> from workflow.hooks for this project.` | As the text says. |
| Session start, probe failed (the venue; 04's not-started flow) | `Not started: the project's hook <command> needs <word>, which is not installed where sessions run. The owner installs it, or removes <event> from workflow.hooks in config/projects/<id>.json.` | The owner, either way; then start again. |
| `pre-push` (the session's shell) | `Broodforge: session <id> does not push. Commit in this tree; Broodforge pushes at the handoff when the session ends.` followed by git's own `error: failed to push some refs to '<url>'` | None inside the session by design; the handoff pushes. A session can edit the clone's `core.hooksPath` on both hosts: a refusal, not the boundary (ADR, rule 4). |
| Push-scan hit (handoff refused, the session stays open; 04's handoff-refused flow) | `Not handed off: <n> commits carry personal data. <sha>: <class> at <path:line>. Ask the agent to amend them, then end the session again.` Class and location, never the value, as the base's scan reports. | The text is fixed and the commit amended, never lifted. A false positive is the owner's: `privacy.allow` in `config/instance.json`, beside `privacy.never` and `privacy.identities` (the bot's own keys, the base's `policy.json` `privacy` section's names), extended per project by the `privacy` section of the copy's `.claude/hooks/policy.json`, read as data. |

## Open checks

Replace the proposal's V12, V16 to V18 and V20; each row names the pass that closes it.

| Id | Check | Closed by |
|---|---|---|
| C1 (V12) | A push from a tree with the `pre-push` hook refuses with the line and exit 1, for `git push origin` and for a push naming the canonical's path, git's own line after it. | The `pre-push` pass, a throwaway clone on the PC; the droplet repeat in P14. |
| C2 (V16) | A `PostToolUse` hook's `additionalContext` reaches the model, an exit-2 child's stderr reaches it, neither blocks. | The chain pass: the engine's hooks documentation first, then its first real turn. |
| C3 (V17) | The base's `git-guard` and `post-edit` under Node on the PC finish within ten seconds each, the bot's hook with both within thirty. | The chain pass, timed in its tests. |
| C4 (V18) | The merge route (a deciding hook beside a rewriting one): whether the input is re-judged. | Not needed by the decision; run only if the chain is ever abandoned (the proposal's reversal conditions). |
| C5 (V20) | The `tools` field of an `--agents` definition restricts the subagent. | The loaders pass; until then the field is passed through and not relied on. |
| C6 (V13) | The bot's own hook fires for a subagent's call in a live session. | The first injected session's audit (a `permission decision` row with `agent_type`). |
