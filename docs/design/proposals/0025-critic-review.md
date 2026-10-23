# Critic review of proposal 0025

Date: 2026-10-23. Reviewer: critic (Fable 5.1). Sources: Claude Code 2.1.309 on the owner's PC
(`claude --help`, and nine `claude -p` runs from a throwaway project under `<scratchpad>/probe`,
each with the flags `Args` passes: `--permission-mode default --setting-sources user
--strict-mcp-config --mcp-config <empty map>`; logs kept there), the docs at code.claude.com (hooks,
sub-agents, settings, memory, plugin manifest reference), git 2.56 on the PC, and the repository
(`internal/`, `.claude/hooks/`, the ADRs and findings the proposal cites). Scope per the brief: the
project settings source stays off and `CLAUDE.md` is not pasted without a composition; neither is
re-argued. The `git-guard` hook denied this agent's `git commit` and `git push` in the throwaway
repository as well (it judges by command text, not location), so V12's push half is settled by
git's remote-helper lookup rather than a push; nothing was routed around.

## Verdict in one paragraph

The composition rule is the right shape and most of the engine facts hold: the append and its file
form, `--agents` from a file with `model` and `effort`, two hooks on one matcher, `onFailure:
block`, `--plugin-dir` with agents, skills and hooks, the snapshot, and the subagent's hook input
all behave as the proposal hopes under `--setting-sources user`. Two things break the
recommendation as written. First, rule 2's "a repository rule can only narrow" is prose, and the
mechanism chosen (the repository's hooks merged into the same `PreToolUse` list as the bot's hook)
refutes it: a merged hook returning `allow` with `updatedInput` rewrote a `Write` to a path outside
the project directory after the other hook had judged the original path, and the file was created
there with the stream crediting "the hook". Second, the bot's own hook denies every subagent
launch today: the engine names the call `Agent` in the hook input and the stream, while the policy
table keys the allow on `Task`, so nothing injected would run until that one line moves. Beyond
those: V7 is refuted as written (`--add-dir` loads `CLAUDE.md` only with an environment variable,
and `.claude/rules/` loads from nowhere under these flags); D writes into the config directory the
droplet binds read-only; the `main` landing ideas are not carried, because the project's privacy
scan runs on neither side of a session; and on the PC nothing the proposal calls un-editable is.
The best alternative the proposal did not consider is for the bot's hook to be the only hook the
engine sees and to run the repository's `PreToolUse` hooks itself as children, honouring only a
deny: that makes "only narrow" true by construction and removes the merge semantics from the
trust argument.

## Claims verified

| Claim | Finding | How |
|---|---|---|
| V1 | Holds. `--append-system-prompt "The snapshot word is ALPHA."` beside `--setting-sources user`: the model answered ALPHA. | probe |
| V2 | Holds. `--append-system-prompt-file <file>` is accepted and honoured (the append word came back) although `--help` lists it only inside the `--bare` text. This repository's `CLAUDE.md` with its imports is about 14 KB across a dozen files (`imports.js`), under the Windows limit either way; the file form keeps it out of the process list. | probe, measurement |
| V3 | Holds, and cuts both ways. A resume of the ALPHA session launched with `--append-system-prompt "The snapshot word is BRAVO."` still answered ALPHA. The help text adds "until the conversation is compacted": after a compaction the prompt is rendered again from the flags of that launch, so a changed rule does reach a running session, at a moment nobody chose. The per-turn trailer list in `SystemPromptAppend` is therefore stale from turn two; `03-architecture.md` ("the system prompt append is per process") is wrong. | probe, `--help` |
| V4 | Holds. `--agents <file>` under `-p` with `--setting-sources user`: `init.agents` listed the agent, it launched, its `model: haiku` ran (both models in `modelUsage`), `effort` and `tools` were accepted without error (the `tools` restriction itself untested). The subagent's own `Bash` call reached the settings hook with `agent_type` set to the agent's name and an `agent_id`. The docs list the accepted fields (description, prompt, tools, disallowedTools, model, permissionMode, effort, skills, hooks, maxTurns and more); the file form needs v2.1.281 or later. | probe, docs |
| V5 | Holds. Two `PreToolUse` hooks on `matcher: Bash` in one `--settings` file both ran (the logging hook wrote its line; the deny hook's deny was applied, `permission_decision.source: hook`). A `deny` beside an `allow` for the same call: deny won. A `defer` beside an `allow`: the turn ended `tool_deferred`, so defer won. `onFailure: "block"` on a command that cannot start blocked the `Write` with `PreToolUse:Write hook error: [<command>]: failed; blocking because onFailure is "block"` followed by the shell's not-found line; the same missing command without `onFailure` fell through on an `Edit` to the engine's `-p` default, which denied it with no prompt. The docs say hooks on one event run in parallel; the combination rule for conflicting decisions is undocumented and was only observed. | probe, docs |
| V6 | Holds. `CLAUDE_PROJECT_DIR` in the hook's environment was the engine's start directory with forward slashes, for the driver's and the subagent's calls alike; the hook input's `cwd` was the same directory in backslash form. The docs: the project dir "stays put" when Claude enters a worktree; `cwd` follows. | probe, docs |
| V7 | Refuted as written. `--add-dir <copy>` alone: the copy's `CLAUDE.md` did not load. With `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` it did (the docs name the variable: "By default, CLAUDE.md files from these directories are not loaded"). The path-scoped rule in the copy's `.claude/rules/` and the one in the cwd's did not load in either run, after a `Read` of a matching file (the transcript shows the `Read`); the docs: "Project rules are skipped if you exclude `project` from `--setting-sources`" since v2.1.211. So rules ride on nothing: they are appended unconditionally or become skills. | probe, docs |
| V8 | Holds (the fear is confirmed). The cwd's `.claude/skills/canary-skill` was absent from `init.skills` and the `Skill` tool had no such skill; the owner's user-scope skills were present. | probe |
| V9 | Holds on the PC. A directory with `.claude-plugin/plugin.json` (`name: wf`), `agents/`, `skills/<name>/SKILL.md` and `hooks/hooks.json` passed as `--plugin-dir`: `init.agents` gained `wf:plug-agent` (it launched and answered), `init.skills` gained `wf:plug-skill` (the `Skill` tool ran it), the hooks ran with `${CLAUDE_PLUGIN_ROOT}` substituted to the plugin directory and exported, plus `CLAUDE_PLUGIN_DATA` and `CLAUDE_PROJECT_DIR`; a plugin hook's deny arrived as the same `PreToolUse:Bash hook error: <reason>` line with `source: hook`. Every name is prefixed with the plugin name. The manifest is optional; a `CLAUDE.md` at the plugin root is not loaded (docs, with a validator warning). Loading the plugin created `~/.claude/plugins/data/wf-inline` in the config directory at once (not on first reference). | probe, docs |
| V10 | Holds by code. `policy.js` reads `policy.json` from `__dirname`; `git-guard.js`, `post-edit.js` and `usage-gate.js` `require` their siblings relatively, and take the root from the hook input's `cwd` (the tree), which is what the cap and path checks want. | code |
| V11 | Holds by code. `usage-gate.js`: a missing session file is missing data, allowed with a note unless `usage.whenStale` is `deny`; this repository sets `allow`. In a session `HOME` is the session home, so the file never exists. | code |
| V12 | Holds for `origin`, not for every push. A `clone --shared` of this repository's `main` into the scratchpad wrote `.git/objects/info/alternates` naming the source's objects; `remote.origin.pushurl` set to `broodforge-sessions-never-push://refused` left `git fetch origin` working (exit 0) and `git log` reading through the alternates, and the transport lookup for that scheme fails with `git: 'remote-broodforge-sessions-never-push' is not a git command ... fatal: remote helper '...' aborted session`, the same lookup a push to `origin` performs. A push naming a URL or path instead of the remote (`git push <canonical>`) does not consult `pushurl` at all. The push itself was not run (the guard). | probe, git docs |
| V13 | Holds. The subagent's `Bash` call fired the settings hook with `agent_type` and `agent_id` (V4), and the docs say hooks from settings, managed policy and plugins run inside subagents. Unobserved: the bot's own hook in a real session. | probe, docs |
| V14 | Holds. The `--agents` definition's `model: haiku` ran under `--model sonnet`; `modelUsage` carried both ids. The docs' order: per-invocation `model`, then the definition's, then `CLAUDE_CODE_SUBAGENT_MODEL`, then the main model. | probe, docs |
| V15 | Misread. The companion binds `/usr` read-only in full and sets `PATH` to include `/usr/bin`; ADR 0023's amendment says the droplet spike's Node was `/usr/bin/node` "inside the system bind". A distribution Node is therefore inside the sandbox with no bind change; what a project needs is an operator install row, not a P14 sandbox pass. | ADR 0022 companion, ADR 0023 |

Further facts the probes produced, not in the proposal's list:

| Id | Finding | How |
|---|---|---|
| P1 | The subagent launch is tool `Agent`: in the `tool_use` block, in the hook input's `tool_name` (an empty-matcher hook logged it), and in the base's own `usage-gate` matcher (`Agent|Workflow`). `init.tools` still lists `Task`. `internal/policy` and `03-hook-contract.md` key `ClassAgent` on `TodoWrite` and `Task`, so an `Agent` call falls to "anything else: deny" in both reaches. | probe, code |
| P2 | A merged `PreToolUse` hook on `matcher: Write` returning `permissionDecision: allow` with `updatedInput` whose `file_path` was the parent of the project directory: the file was created there, the tool result reported the rewritten path, `permission_decision` read `{decision: accept, source: hook}`, and the empty-matcher logging hook beside it had received the original in-tree path. Unobserved: the same with a second hook that itself answers `allow` for the original path (the bot's case in policy mode); nothing in the docs says the input is re-judged. | probe |
| P3 | In `-p` default mode with no hook decision the engine allowed `echo` (`source: config, reason_type: subcommandResults`), allowed `Read` (`reason_type: mode`) and denied `Edit` and `Write` without a prompt: C39 as recorded. | probe |
| P4 | The bot's handoff push is `git -c credential.helper= push --quiet <remote> <refspec>` from the canonical (`internal/git/handoff.go`); no `core.hooksPath` is set on the canonical by the bot, and neither `03-architecture.md` nor `05-debug-interface-git.md` mentions a privacy scan on that push. The project's scan is `.claude/hooks/git/pre-push`, "enabled per clone: `git config core.hooksPath`", and rule 20 of `git-guard.js`, which fires on a `git push` command in a session's Bash. | code, docs |
| P5 | `git-guard.js` treats a call with no `agent_type` as the main session; this repository's policy lets the main session commit (`git.commit.subagents` limits subagents only) and push (`mainSession: true`). The driver process in a Discord session has no `agent_type`. | code |

Not settled: V9 on the droplet (the `plugins/data` write under the read-only bind, see objection 7);
V12's push proper; the `tools` restriction of an `--agents` definition; the bot's hook under a
subagent in a live session; P2 with a deciding hook beside the rewriting one.

## Refusals and lifts

| Mechanism | Who lifts it, from where, while a session runs | Does the message say so? |
|---|---|---|
| `pushurl` to an unusable scheme (rule 4) | Nobody in the session by design; on the PC the session itself (`git remote set-url --push`, or a push naming the path, both the same-account gap); the owner from the host. | No: git's text names the scheme (`remote helper '...' aborted session`), not Broodforge, not the handoff command. |
| The precedence header | The owner approves the text; a session cannot change it (V3); a repository cannot (it is the bot's). | Yes, it states what wins. It enforces nothing. |
| The read-only `workflow/<id>/` copy | Droplet: the bind; a rule change reaches the next session. PC: not read-only (same account); the bot's hook denies writes there as outside the tree, the deny text ends "Nobody in this session can allow it", which is true for the hook and false for the PC's Bash. | Partly: the deny says nobody; nothing says "edit the default branch and start a new session". |
| Merged repository hooks with `onFailure: block` | The owner, by editing the project's `workflow` key or installing the runtime, then a new session; nothing mid-session (the engine reloads hooks from the settings file, but the file is the bot's and rewritten per process). | No: the engine's line names the repository's script path and `onFailure`; not who owns it, not the lift. |
| A repository hook's own deny (`git-guard` rules) | The repository's authors, on the default branch, next session. | Its advice is written for an interactive checkout: "ask the user", "from the main session", "Leave the work committed and report". In a Discord session none of those exist. |
| The bot's hook deny of `Agent` (P1) | The project: `internal/policy` and a fixture; nobody in any session. | The deny text says the tool is outside reach, which is false. |

A refusal with no documented lift is an objection; four of the six rows have none in the proposal.

## Objections

1. **Blocking. Rule 2 is stated, not enforced, and the chosen mechanism refutes it (P2).** The
   proposal puts the repository's `PreToolUse` hooks beside the bot's in one settings file and says
   "the bot's hook decides first and a deny is final". Hooks on one event run in parallel (docs),
   there is no first; the decision field combines restrictively (V5: deny over allow, defer over
   allow), which is good, but `updatedInput` is applied from whichever hook returns it, after the
   other hooks judged the original input. A repository hook can move a write, a command or a path
   outside the tree after the bot's hook said allow. Even an honest hook may do this (the engine's
   own docs use `updatedInput` for formatting). Resolves by: the engine never sees a repository
   `PreToolUse` hook. The bot's hook runs them itself, as children, after its own decision, from
   the trusted copy, with the same stdin, and takes from each only exit 2 or a `deny` (a `deny`
   narrows; `allow`, `ask`, `defer` and `updatedInput` are dropped and logged). "Only narrow" is
   then a property of code the repository cannot reach, the parallel race is gone, `onFailure` is
   the bot's own rule, and the audit row names which hook denied. The cost is one loop in
   `internal/hook` and a per-project runtime check at session start; it is smaller than the three
   loaders of A. This is the best alternative the proposal did not consider.

2. **Blocking (for the premise, one line to fix). The bot's hook denies every subagent launch
   (P1).** The tool is `Agent` in the hook input; the policy table allows `Task`. Nothing injected
   by A or D runs until `internal/policy` and `03-hook-contract.md` say `Agent` (and keep `Task`
   for an older pin), with a fixture holding an `Agent` call. The delegation design in 04 (`Task`
   lines indented) and the proposal's Owner 1 assume launches are allowed. Resolves by: the fix
   and fixture first, recorded as a finding; the proposal's "first implementation step" names it.

3. **Serious. Repository hooks are code execution outside the reach table.** In read-only reach
   the bot denies `Bash`; a hook from the repository runs on every tool call as the engine user
   with no reach check, which is exactly the exposure ADR 0020 refused ("a project's hooks must
   not load as the bot account"). The proposal takes "the `hooks` key" whole: a `Stop` hook that
   exits 2 keeps the turn going (docs), which is a loop on the owner's plan; `UserPromptSubmit` and
   `SessionStart` hooks inject text above the header; `PostToolUse` runs the formatter. The
   proposal's answer (Bash runs the project's tests anyway) holds in code reach only, and the
   owner, not the proposal, decides what spends the plan. Resolves by: `PreToolUse` and
   `PostToolUse` only, code reach only, through objection 1's chain; a `Stop` or
   `UserPromptSubmit` hook never; the per-project `workflow` key lists which events, and Owner 2
   is asked about money as well as exposure.

4. **Serious. On the PC nothing the proposal calls un-editable is.** ADR 0022 (Windows) says a
   session "can rewrite its `settings.json` mid-turn", and the engine reloads `hooks` from it while
   running (ADR 0022 Context); the `workflow/<id>/` copy, the bot-owned hooks directory and the
   canonical are the same account. The trust constraint and the ten-year sentence ("no project
   and no session can loosen what the bot guarantees") hold on the droplet only, and milestone 1
   runs on the PC. The proposal names one gap (`git -C <canonical> push`) as if it were the only
   one. Resolves by: a sentence per host in the rule; on the PC the standing rule C (a reviewer
   read and the pusher's checks outside the session) is the floor the landing ideas cannot
   drop, until W1 or WSL2 (ADR 0022's deferred items) gives the PC a boundary.

5. **Serious. The `main` landing ideas are not carried; the proposal says they are "theirs to
   decide".** ADR 0018: every push is scanned commit by commit. In a session no push happens
   (rule 4), so `git-guard` rule 20 never fires; the bot's push from the canonical runs no
   `pre-push` hook (P4); so a `handoff: main` or `/session land` would put commits on the default
   branch with no privacy scan at all, which is the finding of 2026-10-17 repeated by design. The
   caps check (pusher check 7) and the review requirement are instruction-only inside a session
   (objection 11). Resolves by: the bot's handoff check runs the project's pre-push checks (the
   canonical's `core.hooksPath` set to the project's `.claude/hooks/git`, or a Go scan over the
   same patterns) before any push, `pull-request` and `push` included; a land additionally
   requires a fact the stream can prove (a `reviewer` `Agent` call completed after the last
   commit); the `Plan:` blocks of the two ideas depend on both. The `auto` mode idea is
   independent, as the proposal says.

6. **Serious. V7 is refuted as written, so A's rules half has no flag (V7).** `--add-dir` loads
   the copy's `CLAUDE.md` only under `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1`, and
   `.claude/rules/` loads from neither the cwd nor the added directory under these flags. Rules
   are appended unconditionally (this repository: `docs.md`, `go.md`, `hooks.md`, every turn) or
   become skills in D's plugin, which load on demand. Resolves by: the proposal states both and
   picks; `--add-dir` is dropped (it also widens the engine's own file-tool scope to the copy,
   which only the bot's hook then narrows).

7. **Serious. D writes into the config directory the droplet binds read-only (V9).** Loading the
   plugin created `plugins/data/<name>-inline` under the config directory at once; the companion
   binds `<cfg>/plugins` read-only and empty for a subscription session. Whether the engine then
   refuses the plugin, skips the directory or fails the turn is unobserved; the P14 pass finds
   out, not an afternoon on the PC. Also, every plugin agent, skill and command is namespaced
   (`wf:coder`), while the repository's instructions, the `brief` skill and the routing table say
   `coder`; the model used the prefixed name when it was the only one offered, which is not a
   guarantee. Resolves by: a droplet row for `--plugin-dir` before D is chosen; `<sess>/cfg/plugins`
   bound read-write like `projects/`; agents through `--agents` (bare names, V4 holds) and the
   plugin for skills and hooks only, or the header states the prefix.

8. **Serious. The snapshot (V3) is misdescribed in both directions.** "Fixed for the session's
   life" is wrong after a compaction, when the then-current copy is rendered in; and the per-turn
   `SystemPromptAppend` is already stale from turn two, so a participant who joins later is in no
   prompt's trailer list, and `03-architecture.md` says the opposite of what the engine does.
   Resolves by: the finding the proposal promises, written; the header says when the copy was
   taken (commit and date); the trailer list moves to the turn's `From:` block or is stated once
   as the `commit-msg` hook's job alone; the copy stays for the session's life (it does); 03
   corrected.

9. **Serious. `pushurl` is narrower and mute compared with a hook the bot already owns (V12).**
   It stops only a push that names `origin`; a push naming the canonical's path bypasses it and
   on the PC writes the canonical. Its message is git's remote-helper text. The clone already has
   `core.hooksPath` set to a Broodforge-owned directory for `commit-msg`; a `pre-push` there
   refuses every push with a sentence naming the session and the handoff command, and `worktree
   check` can prove both. Both are the same-account gap on the PC. Resolves by: `pre-push` as the
   mechanism, `pushurl` as belt, the message the owner's to word, listed in 04's table.

10. **Serious. Four refusals have no lift and misleading text (the table above).** A missing
    runtime under `onFailure: block` makes every matching call fail for the session's life with
    a line naming a script path; the base's `git-guard` advice tells a Discord session to "ask
    the user" and "leave it to the main session". Resolves by: `config check` and session start
    run each injected hook command's runtime (`node --version` for a `node ...` command) and
    refuse or skip with a notice; the bot's trace line wraps a repository deny as "refused by the
    project's hook <name>; the project's rules on its default branch decide this"; `session show`
    lists the hooks injected and the commit they came from; the repository's own wording stays
    the repository's.

11. **Minor. Owner 1 overstates what the hooks hold (P5).** `git-guard` exempts the main session;
    the driver process has no `agent_type`, so under injection it may still commit (and push, but
    for rule 4) exactly as s0005 did. Delegation and "the runner commits" are instruction-only for
    the driver; the hooks bind subagents. Resolves by: saying so, and a bot-side check (objection
    5) rather than a claim that the hook enforces delegation.

12. **Minor. V15's cost is misplaced.** A distribution Node is inside the `/usr` bind and on the
    sandbox `PATH`; no sandbox change, an operator install row. The rewrite of hook commands must
    also handle the base's `${CLAUDE_PROJECT_DIR:-.}` form, not only `${CLAUDE_PROJECT_DIR}`.

13. **Minor. Boilerplate.** C's "Exit cost: none. Moving parts: none": its moving part is a person
    reading every change, and its exit cost is the one being paid now (eleven defects in s0005).
    The ten-year sentence cannot fail only because it is unqualified; it fails on the PC today
    (objection 4). B's prediction ("s0005's failure class returns per edit") can fail and is fine.

14. **Minor. Hidden couplings to name in the ADR.** The trust argument now rests on the engine's
    undocumented hook-merge semantics (parallel, restrictive-wins, `updatedInput` applied), which a
    pin move can change silently: the fixture suite must hold a two-hook case. A copy "taken at
    session start" under "check in rarely; lose nothing" (00-vision) means a week-long session
    runs on week-old rules, re-rendered at an unplanned compaction (V3). `--agents` sits above
    project and user agents in priority (docs), so an injected name wins over the credential's
    user-scope agent on the PC, which is the right way round and should be said.

## Pre-mortem

Ten years on, the regret is that the repository's hooks were let into the engine's permission
path. It began as three Node scripts from this repository; by year three every third-party
project's `settings.json` ran as the engine user on every tool call in every reach, a `Stop` hook
in one of them held a session open through a weekend of plan usage, and a formatter hook's
`updatedInput` was the route by which a session wrote outside its tree, audited as "allowed by
the hook" with no way to say which. The sentence that should have caught it is rule 2: "a
repository rule can only narrow what the bot allows ... never widen it", written as a property
without a mechanism, beside option A's "merged into the bot's settings file beside the bot's own
hook" and the constraint "the bot's hook decides first and a deny is final", which was never
true of parallel hooks. The smaller regret: the landing ideas were built on "the session having
been held to the rules" while the only push-time check the project had, the privacy scan, ran on
neither side of a session.

## Recommendation

Adopt the composition rule with four amendments: rule 2 names its mechanism (the bot's hook is the
only engine hook; it runs the repository's `PreToolUse` and `PostToolUse` hooks as children, code
reach only, deny-only, from the trusted copy); rule 4 is a Broodforge `pre-push` hook with
`pushurl` as belt; a per-host sentence says the PC enforces none of this and keeps C's gate; the
bot's handoff runs the project's pre-push checks before any push, which the two landing ideas
then build on. Mechanism: instructions by `--append-system-prompt-file` and agents by `--agents`
now (both probed; bare names; rules appended until they are skills), after the one-line `Agent`
fix and its fixture; skills by `--plugin-dir` after a droplet row for the `plugins/data` write;
no `--add-dir`. The exposure default can stay the repository's set for instructions, agents and
skills, which are text; hooks default off for a third-party project until the owner has answered
the money question, since a hook is code. Confidence: high on objections 1, 2, 5 and 7, each
probed or read in code; moderate on the alternative's cost (one loop in `internal/hook`, a
runtime check), which nobody has built; the rule itself I would not change further.
