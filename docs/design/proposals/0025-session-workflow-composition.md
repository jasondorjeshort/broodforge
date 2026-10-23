# 0025. Session workflow composition: the bot's contract and the repository's workflow in one Discord session

Date: 2026-10-23
Status: promoted to ADR 0025

## Question

A Discord session runs under two sets of instructions that have never been separated. The bot's
part (who is speaking, the reach, one call at a time, the trailers, how a session ends and hands
off) applies to every project and is today the short `SystemPromptAppend`. The repository's part
(its `CLAUDE.md` and imports, agents, skills, path rules, hooks: here, delegation, one review, the
caps, who commits and pushes) differs per project and may be absent. With `--setting-sources
user` on every engine process (ADR 0020) the session gets only the bot's part, and the first live
session (s0005) edited this repository with no delegation, no review and no cap check
(`../../findings/2026-10-23-session-settings-sources-drop-claude-md-and-agents.md`,
`../../findings/2026-10-23-first-live-session-commits-reached-main-unreviewed.md`). Milestone 1
(edit Broodforge from its own Discord channel) waits on this. To decide: how the two parts
compose (which wins on a conflict, where each is stated, whether a repository can say "in a
Discord session, X"), which repository files reach a session and who decides, what a session may
do with `git`, and what a project with no `.claude/` gets. In ten years: a new project gets a
correct session on the day it is configured, with no bot change per project, and on a host with
an isolation boundary (the droplet today) no project and no session can loosen what the bot
guarantees; on a host without one (the PC, ADR 0022) the bot says so and a gate outside the
session stands in.

The three ideas of 2026-10-23 (the engine's `auto` mode as a third session mode, `handoff: main`,
`/session land`) are consumers of this answer, not parts of it. The `auto` idea is independent.
The two landing ideas depend on rule 7 below (a push-time scan and a proven in-session review)
and cannot be built until it is.

## Constraints that matter here

- Target platforms (`../06-platforms.md`), licensing (ADR 0001), agent-drivable debug interface
  for every feature, agents write all code.
- Ruled out: re-enabling the project settings source. A project's hooks and `.mcp.json` must not
  load as the bot account (ADR 0020, Settings sources), and a project `settings.json` is ignored
  in a workspace never trusted interactively, which the bot cannot do (the first finding above).
- Ruled out: pasting `CLAUDE.md` into `SystemPromptAppend` without deciding the composition.
- Trust, per host. A session may edit anything in its tree, `.claude/` and `CLAUDE.md` included,
  so text or code that governs the session must come from somewhere the session cannot write.
  On the droplet that is a bot-owned file bound read-only, like the hook's settings file (ADR
  0022, Decision). On the PC nothing is un-editable: the session, the bot and the canonical are
  one account, and the engine reloads `hooks` from a rewritten settings file while running (ADR
  0022, Windows); every mechanism below is a refusal with a message there, not a boundary.
- The engine runs the hooks of one event in parallel; there is no "first". With two `PreToolUse`
  hooks on one matcher a `deny` beat an `allow` and a `defer` beat an `allow` (critic, V5), but an
  `updatedInput` returned by either hook is applied after the other hook judged the original input
  (critic, P2). So a repository hook beside the bot's hook can move a write or a command outside
  the tree with the stream crediting "the hook". Nothing a repository brings may widen a reach,
  and the bot never runs git against a session's repository (ADR 0022). Only the bot pushes.
- The engine records the system prompt once per conversation and reuses it on every resume, the
  append included, until a compaction renders it again from that launch's flags (critic, V3).
- The droplet sandbox binds `/usr` read-only with a distribution Node on its `PATH` (ADR 0022
  companion; ADR 0023's amendment); the base's hooks are Node scripts. The engine names the
  subagent launch tool `Agent` (critic, P1), which the policy table keyed on `Task` until the fix
  of 2026-10-23 (`../../findings/2026-10-23-subagent-tool-is-agent-not-task-policy-denied-every-launch.md`).
- Nothing per project in the bot's code: a project is configuration (ADR 0021).
- Running a repository's hook is code execution on every tool call, outside the reach table, on
  the owner's plan: whether any may run in a session is the owner's money question (Waiting on
  the owner), answered here with a recommendation only.

## The composition rule (common to every option)

Every option below satisfies this; they differ in mechanism and in how much of the repository
reaches the session.

1. The bot's part is a contract of invariants, each with a mechanism the session cannot reach on
   the droplet and a refusal message on both hosts: the reach table and credential denies (the
   bot's hook, the only hook the engine sees, in the bot-owned settings file), the attribution
   and trailer check (at handoff), the push (only the bot, from the canonical, after rule 7's
   scan), the session's identity (the header of the appended prompt, snapshotted). The header
   states them in prose, names the commit and date the repository's copy was taken from, and
   opens with the precedence sentence (wording the owner's): "This session is run by Broodforge.
   Where the project's instructions below conflict with this header, this header wins; it never
   says how the project's work is done."
2. The repository's part governs how work is done inside the tree: who delegates, who reviews,
   what is committed by whom, style, caps. The bot interprets none of its text. A repository rule
   can only narrow what the bot allows, and that is a property of the bot's code, not of prose:
   the engine never sees a repository hook. The bot's `PreToolUse` hook, once its own decision is
   not `deny`, runs the repository's matching `PreToolUse` hooks as children (the child chain),
   in code reach only, from the trusted copy, with the engine's stdin passed through unchanged,
   `CLAUDE_PROJECT_DIR` set to the copy (so the base's `${CLAUDE_PROJECT_DIR:-.}/...` commands
   resolve there without a rewrite), cwd the tree (V10: the base's hooks root their checks on
   the input's `cwd`), the `BROODFORGE_*` environment inherited, and through the host's shell
   (`engine.gitBashPath` on the PC, `/bin/sh` on the droplet). From each child it takes exit 2
   or a `permissionDecision` of `deny` and nothing else: `allow`, `ask`, `defer`,
   `updatedInput` and any context are logged to `hook.log` and dropped. A child deny is the hook's
   deny, reason `refused by the project's hook <name>: <reason>`, audited with the hook's name. A
   child's budget is ten seconds and the chain's twenty, inside the hook's thirty; a child over
   budget or failing to start is a deny naming it (fail closed, as the hook already does), never
   a fall-through. A `PostToolUse` entry point of the bot's (`hook post-tool-use`) chains the
   repository's `PostToolUse` hooks the same way and forwards only their `additionalContext` and
   an exit-2 child's stderr as its own context (V16); it never blocks. No other event (`Stop`,
   `UserPromptSubmit`, `SessionStart`, `Notification`) is ever run from a repository.
3. "In a Discord session, X" is the repository's to say, in its own files, keyed on facts the bot
   already exports: `BROODFORGE_SESSION`, `BROODFORGE_REACH` and `BROODFORGE_MODE` are set in the
   engine process, its shells, the bot's hook and every child (`../03-hook-contract.md`,
   Environment), and the header names the session. A repository `CLAUDE.md` may carry a section
   for sessions; its hooks may branch on the variables; the base's `git-guard`, which exempts a
   call with no `agent_type` as the main session (critic, P5), may grow a policy key for a
   session's driver process (a carry-back). The bot adds no channel and no file format.
4. What a session may run as `git`: any read; commits in the tree by whoever the repository's
   own rules allow (the bot's identity variables and `commit-msg` hook apply to every committer,
   subagents included; the driver process has no `agent_type`, so a repository's commit limits
   bind its subagents and delegation is instruction-only for the driver); never a push. The
   refusal is a Broodforge `pre-push` hook in the clone's bot-owned `core.hooksPath` directory
   (beside `commit-msg`), whose message names the session and the handoff command (wording the
   owner's, listed in 04's table), with `remote.origin.pushurl` set to a scheme no helper serves
   as the belt (V12: alternates and `git fetch origin` unaffected). Both live in the tree's
   `.git/config`, which the session owns on both hosts, so they are refusals with a message, not
   the boundary: on the droplet the boundary is the canonical's read-only bind and the absent
   git credential; on the PC there is none (ADR 0022). `worktree check` proves both are set.
5. Text and code are taken from the canonical checkout's default branch at session start and
   copied into a bot-owned `workflow/<session id>/` beside `settings/<session id>.json`, bound
   read-only on the droplet (`git archive <defaultBranch> -- CLAUDE.md .claude` into it; the
   import lines of `CLAUDE.md` resolve inside the copy). The copy lives as long as the session,
   so a week-long session (00-vision, "check in rarely") runs on the rules of its first day: a
   rule change reaches the next session, and the header says which commit this one has.
6. Per host. On the droplet rules 1 to 5 are enforced by the binds and the bot's hook. On the PC
   they are refusals a session could lift, so C's gate (a reviewer read and the pusher's checks
   outside the session before anything reaches `main`) is the floor the landing ideas cannot
   drop there, until W1 or WSL2 (ADR 0022's deferred items) gives the PC a boundary.
7. Landing. The bot's handoff scans every push (`pull-request`, `push`, and any later land)
   commit by commit before pushing, as ADR 0018 requires: a Go scan the bot owns, whose never
   list and allow list are data read from the instance's configuration and, when present, from
   the copy's `.claude/hooks/policy.json` privacy section; the repository's `pre-push` script is
   never run as the bot (ADR 0020). A land onto the default branch additionally requires a fact
   the stream can prove (the candidate: a `reviewer` `Agent` call that completed after the last
   commit); the two landing ideas' `Plan:` blocks name theirs.

## Options

### A. Inject from the canonical copy through engine flags, hooks through the chain

Instructions: `CLAUDE.md` with its `@` imports resolved by the bot (the base's `imports.js` rule:
relative to the importing file, a missing target loads nothing), then every `.claude/rules/*.md`
unconditionally (no flag loads them under these settings sources, V7; this repository's three cost
tokens every turn until they become skills), appended after the header through
`--append-system-prompt-file` (V2: honoured, keeps the text out of the process list). Agents:
every `.claude/agents/*.md` of the copy translated to the engine's `--agents` JSON (`description`,
`prompt`, `model`, `effort`, `tools`: V4) and passed as a file; bare names, and an injected name
outranks a user-scope agent of the credential's config directory (docs via the critic, 14).
Hooks: the `hooks` key of the copy's `.claude/settings.json`, `PreToolUse` and `PostToolUse`
entries only, written as `workflow/<session id>/hooks.json` for the child chain (rule 2) and
named to the hook by `BROODFORGE_WORKFLOW` (one row in the environment table and the sudoers
`env_keep` list). Skills: none (V8: the cwd's `.claude/skills/` is dropped), so the repository's
procedures (`brief`, `commit`, `proposal`) are unavailable and the agents that name them run
without them; D closes that.
Meets the constraints: the text is trusted (copied from the default branch), the engine sees one
hook per event, nothing per project in code. A project with no `.claude/`: the header alone,
which is s0005's session and worked. Cost: three small loaders in Go (imports, agents, the hooks
list), the chain in `internal/hook` (a matcher on `tool_name`, stdin pass-through, exit code and
one JSON field read), a second hook entry point, and at `config check` and session start a probe
of each listed command's runtime (`node --version` for a `node ...` command) that refuses or
skips with a notice (10). Risks: a project hook over budget denies its own calls for the session
(the owner turns hooks off per project); the base's usage gate sees no meters in a session and
allows with a note (V11), so HOLD is the bot's own meters. Comparable: the engine's `--bare` help
lists these flags as the way to provide context explicitly.
Exit cost: delete the loaders, the chain and the copy step; sessions fall back to the header.
Moving parts: the copy directory and its bind, the loaders, the chain and its budget, the
`workflow` key per project (`inject`, `hooks`), `session show` and `reach-report` listing what
was injected, from which commit, and which hooks ran.
Git in a session: rule 4; here, the repository's `git-guard` additionally limits commits to the
`runner` subagent by `agent_type` (V4: set for a subagent's calls). Third party with no
`.claude/`: the header only.

### B. Inject only a declared per-project list from configuration

The same loaders and chain as A, but the project file names what is taken: `"workflow":
{"instructions": ["CLAUDE.md"], "agents": [".claude/agents/*.md"], "hooks": ["PreToolUse"]}`;
nothing not listed reaches the session, and a project with no key gets the header only.
`config check` verifies every pattern matches something on the default branch.
Meets the constraints as A does, and answers the exposure question in the owner's favour: a
third-party project's instructions reach a session only when the owner lists them. Cost: A's
plus the schema and check. Risk, and the reason it is not recommended as the default for text:
s0005's failure class returns per project and per edit. A new agent file, a new import, a renamed
rule is silently not loaded until the owner edits configuration, and the session behaves as if
the rule never existed, which is exactly what went wrong. The list also puts a per-project fact
(what the project's workflow consists of) in the bot's configuration, where the project cannot
maintain it. For hooks the argument reverses: a hook is code, and an explicit list is the right
shape for an opt-in (Recommendation).
Exit cost: A's plus removing the keys. Moving parts: A's plus the schema.
Git in a session: rule 4, and only the hooks listed. Third party with no `.claude/`: the header.

### C. The session runs only the bot's part; the gate is at landing

No injection. The session works under the header alone, commits to its branch, and the bot's
handoff (today's `push` and `pull-request`) is the only exit. Anything reaching `main` passes
the project's own workflow outside the session: a reviewer read and the pusher's checks from an
interactive session that loads the rules, which is the standing rule in the handoff file today.
Meets the constraints trivially (nothing new runs). The cost is in use: a session of this
repository does not delegate, does not review, does not know the caps, and the defects surface
after the fact (eleven in s0005, found by two reviews). The two landing ideas cannot be built on
it: a direct land on `main` from a session with no rules is the s0005 push automated. For a
third-party project with its own workflow the session ignores it just the same.
Exit cost: the one being paid now, a person's reading of every session's change; leaving C is
adding A or D. Moving parts: that person, every time.
Git in a session: rule 4. Third party with no `.claude/`: the header, same as every project.
C is the floor rather than an alternative: until injection is built it is the rule everywhere,
and on the PC it stays the rule after it (rule 6).

### D. The repository's skills as a plugin built from the canonical copy

Instructions and agents as in A. Skills, which no other route gives (V8), through one engine
loader: the bot lays the copy's `.claude/skills/` out as a plugin directory (a manifest and
`skills/`) and passes `--plugin-dir <copy>/plugin` (V9: the skill appeared in `init.skills` and
the `Skill` tool ran it). Hooks stay with the chain, never the plugin's `hooks.json`; agents stay
on `--agents`, because every plugin name is prefixed (`wf:coder`) while the repository's
instructions and routing table say `coder`. The rules under `.claude/rules/` can become skills
here and stop being appended every turn.
Risks: loading a plugin writes `plugins/data/<name>-inline` into the config directory at once,
which the droplet binds read-only and empty for a subscription session (ADR 0022 companion);
whether the engine then refuses the plugin, skips the write or fails the turn is unobserved, so
D needs a droplet row (`<sess>/cfg/plugins` bound read-write like `projects/`, or a refusal) in
the P14 pass before it is chosen there. The plugin format is the engine's and less exercised in
`-p` than the flags; a pin move breaks every project's skills at once.
Exit cost: delete the layout step; the skills go, A stands. Moving parts: the layout step, the
same copy directory and `workflow` key as A, the droplet bind row. Git in a session: rule 4 and
the chain, as A. Third party with no `.claude/`: the header.

## Recommendation

The composition rule (1 to 7), A now, D for skills after its droplet row.

- Decide the rule as the ADR; it is the part that outlives any flag. Rule 2's chain is taken on
  its merits: it is the only mechanism found that makes "only narrow" a property of the bot's
  code (the engine's merge semantics are undocumented and apply `updatedInput` from any hook),
  and it removes those semantics from the trust argument altogether, so the fixture suite needs
  the single-hook contract C39 already has plus a chain case, not a two-engine-hook case. Its
  costs: a subset of the engine's hook protocol reimplemented in Go (matcher, stdin, exit code,
  one field) and tracked against the pin; a second entry point for `PostToolUse`; a budget that
  can make a slow project hook deny its own calls; a new environment row in the companion.
- Exposure default: text on, hooks opt-in. Instructions, rules and agents (and skills under D)
  from every project alike, taken from the default branch; `workflow.inject: none` turns a
  project off. Hooks run only for a project whose `workflow.hooks` lists the events (B's shape),
  code reach only, and only once the owner has answered the money question; this repository
  lists `PreToolUse` and `PostToolUse` the day it is answered yes. Reason: text default off
  reproduces s0005 for every project the owner forgets to switch on, while a hook is code
  execution on the plan and is the owner's to spend.
- Mechanism: `--append-system-prompt-file`, `--agents` from a file, the chain, rules appended;
  no `--add-dir` (it loads `CLAUDE.md` only under an environment variable, loads no rules, and
  widens the engine's own file-tool scope to the copy, V7). D after the droplet row.
- Rule 4's `pre-push` and `pushurl` and rule 7's scan go first: the push hole exists today in
  policy mode, and no push of the bot's is scanned today (critic, P4).
- C stays the rule everywhere until the build lands, and on the PC after it (rule 6).

Two further decisions are the owner's, answered here with a recommendation only (Waiting on the
owner): Owner 1, who coordinates inside a session, recommended the session's driver process in
the role the base gives the interactive main session (it reads, briefs and launches the
repository's agents; the session's `/model` and `/effort` steer it alone, each subagent pinning
its own model as its file says, V14), noting that for the driver the repository's commit limit is
instruction-only (rule 4) and the bot-side proven-review fact of rule 7 is what a land can rely
on. Owner 2, which files a third-party project exposes, recommended as the exposure default
above, since a project's text is what its own developers get and its hooks are the money
question's.

First implementation steps, after the ADR: the `Agent` fix with its fixture (done outside this
proposal, the finding above); `pre-push` and `pushurl` with their `worktree check` lines and the
Go push scan in the handoff; the copy directory, the `workflow` key and `config check`; the
chain with its budget, runtime probe, audit row and trace line; then the loaders, each with its
`session show` line.

## What would change this recommendation

- The owner answers the money question no: hooks are never injected; rules 2's chain is not
  built; delegation, review and caps are instruction-only in every session and rule 7's proven
  review is the only in-session fact a land can use.
- The owner chooses off-by-default exposure for text (Owner 2): B's list becomes the required
  key and `config check` refuses a project that has `.claude/` and no `workflow` key, so the
  omission is loud rather than silent.
- The chain's reimplemented subset drifts from the engine (a hook input field the base's hooks
  need and the bot's hook does not pass): a per-pin fixture of the chain, or the merge route with
  `updatedInput` stripped if a later engine lets a settings file forbid it.
- The droplet row for D fails and cannot be bound: skills stay unavailable in sessions; rules
  stay appended; the repository's procedures are summarised in its instructions instead.
- A later engine loads `CLAUDE.md`, rules and agents under `--setting-sources user` on its own:
  the loaders go, the copy directory and the chain stay (trust), the rule stands.

## Claims to verify

Engine 2.1.309 on the owner's PC unless noted. V1 to V15 were checked by the critic
(`0025-critic-review.md`, Claims verified, authoritative over this list): V1, V2, V4, V5, V6, V8,
V9 (PC), V10, V11, V13 and V14 hold; V3 holds with the compaction re-render; V7 is refuted as
written; V12 holds for `origin` only; V15 was this proposal's misreading. Still open:

- V12. The push itself from a tree with `pushurl` set, and the `pre-push` hook's refusal, run in
  a throwaway clone (the critic's guard stopped short of the push).
- V13. The bot's own hook under a subagent's call in a live session (the settings-file hook was
  observed; the bot's was not).
- V16. The engine's `PostToolUse` output contract: a hook's `additionalContext` reaches the
  model, exit 2's stderr reaches it, neither blocks (the engine's hooks documentation; a probe).
- V17. The chain's budget against real hooks: the base's `git-guard` and `post-edit` under Node
  on the PC inside ten seconds each, and the bot's hook inside thirty with both.
- V18. P2 with a deciding hook beside the rewriting one (the merge route, kept only as the
  fallback above): whether the input is re-judged.
- V19. D on the droplet: the `plugins/data` write under the read-only bind (P14).
- V20. The `tools` restriction of an `--agents` definition restricts the subagent.

## Critic's objections and responses

1. Rule 2 stated, not enforced; the merged-hook mechanism refutes it (P2). Response: correct, and
   the undocumented merge semantics were the wrong thing to rest trust on. The child chain is
   adopted as rule 2 with its order (the bot's decision first, children only when it is not
   deny, code reach only), its output filter (exit 2 or `deny` only), its budget (fail closed) and
   its costs stated in the Recommendation. Resolution: accepted with change.
2. The bot's hook denies every `Agent` launch (P1). Response: fixed outside the proposal in
   `internal/policy` and `03-hook-contract.md` with a finding; named as the first implementation
   step and in the constraints. Resolution: accepted.
3. Repository hooks are code execution outside the reach table. Response: the chain runs only
   `PreToolUse` and `PostToolUse` children, in code reach only; every other event is excluded by
   rule 2; hooks are opt-in per project by event, and whether any may run at all is the owner's
   money question, listed below. Resolution: accepted with change.
4. On the PC nothing is un-editable. Response: the trust constraint is now per host, rule 6
   makes C the floor on the PC until W1 or WSL2, and the ten-year sentence is qualified.
   Resolution: accepted.
5. The landing ideas are not carried; no push of the bot's is scanned. Response: rule 7 adds a
   bot-owned Go scan before every push with the lists read as data, and a stream-proven review
   fact for a land. The critic's first form (the canonical's `core.hooksPath` running the
   project's `pre-push`) is not taken: that runs a repository script as the bot account, which
   ADR 0020 refuses. Resolution: accepted with change.
6. V7 refuted; `--add-dir` loads no rules. Response: `--add-dir` dropped; rules appended
   unconditionally in A, skills in D. Resolution: accepted.
7. D writes into the read-only config directory; plugin names are prefixed. Response: D narrowed
   to skills, agents stay on `--agents`, a droplet row (V19) before D is chosen there.
   Resolution: accepted with change.
8. The snapshot misdescribed. Response: the constraint and rule 5 now say "until a compaction
   renders it again"; the copy is per session so its text does not change at that moment, the
   bot's per-turn header does; the header names its commit. The stale trailer list is the
   critic's observation to record as a finding and corrects `03-architecture.md` as an ADR
   consequence; the `commit-msg` hook and the handoff check are the enforcement already.
   Resolution: accepted with change.
9. `pushurl` is narrower and mute than a `pre-push` hook. Response: `pre-push` is the mechanism,
   `pushurl` the belt. Added, because it holds on both hosts: both live in the tree's
   `.git/config`, which the session owns on the droplet too, so neither is the boundary; rule 4
   names the real one per host. Resolution: accepted with change.
10. Four refusals without lifts. Response: the runtime probe at `config check` and session start,
    the trace wrapping `refused by the project's hook <name>`, and `session show` listing the
    injected hooks and commit are in A and rule 2; the repository's own wording stays its own
    (rule 3 lets it address sessions). Resolution: accepted.
11. Owner 1 overstates what the hooks hold (P5). Response: rule 4 and the Recommendation say the
    driver process has no `agent_type`, delegation is instruction-only for it, and rule 7's fact
    is what a land relies on; rule 3 offers the base a policy key for the driver as a carry-back.
    Resolution: accepted.
12. V15 misplaced; the `:-.` form. Response: Node is inside the `/usr` bind, an operator install
    row; the chain sets `CLAUDE_PROJECT_DIR` for the child so neither form needs a rewrite.
    Resolution: accepted.
13. Boilerplate in C and the ten-year sentence. Response: both rewritten. Resolution: accepted.
14. Hidden couplings. Response: the merge-semantics coupling is removed by the chain rather than
    named; the week-old-rules fact and the `--agents` priority are stated in rule 5 and A.
    Resolution: accepted with change.

## Waiting on the owner

All four answered by the owner on 2026-10-23, each as recommended; recorded in ADR 0025.

- The money question: whether any repository hook may run inside a session at all (code
  execution on every tool call, on the plan, outside the reach table). Recommended: yes, through
  the chain only, `PreToolUse` and `PostToolUse`, code reach, opt-in per project by event; this
  repository opted in. Answered: yes, exactly so; this repository opts in (ADR 0025, rules 2 and 6).
- The precedence header's wording (rule 1), and the `pre-push` refusal line (rule 4), both read
  by people. Answered: the precedence sentence accepted as drafted; the `pre-push` line is drafted
  in the ADR (rule 4) and marked for the owner to confirm.
- Owner 1: who coordinates inside a session. Recommended: the driver process, in the base's
  main-session role, launching the repository's agents. Answered: the driver process (ADR 0025,
  rule 9).
- Owner 2: which files a third-party project exposes. Recommended: its text (instructions,
  rules, agents, skills) by default with `inject: none` as the switch; its hooks only by the
  money question's answer and an explicit per-project list. Answered: text by default,
  `inject: none` the switch, hooks only by an explicit per-project list (ADR 0025, rules 5 and 6).
