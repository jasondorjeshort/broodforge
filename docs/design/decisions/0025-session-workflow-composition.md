# 0025. Session workflow composition: the bot's contract, the repository's workflow injected from the canonical, the hook child chain

Date: 2026-10-23
Status: accepted

Record: proposal `../proposals/0025-session-workflow-composition.md`; `../proposals/0025-critic-review.md` (its table is authoritative).
Companion: `0025-refusals-and-checks.md` (the copy procedure, the runtime probe, refusals and lifts, open checks).

## Context

A Discord session runs the engine with `--setting-sources user` (ADR 0020) and so loads none of the
project's `CLAUDE.md`, agents, rules, skills or hooks; the first live session (s0005) edited this
repository with no delegation, review or cap check
(`../../findings/2026-10-23-session-settings-sources-drop-claude-md-and-agents.md`,
`../../findings/2026-10-23-first-live-session-commits-reached-main-unreviewed.md`). Re-enabling the
project settings source is ruled out (a project's hooks must not load as the bot account; a project
`settings.json` needs an interactive trust). The critic probed engine 2.1.309 on the PC: the append
and its file form, `--agents` from a file and `--plugin-dir` load beside `--setting-sources user`;
rules and the cwd's skills load from nowhere; `--add-dir` loads `CLAUDE.md` only under an
environment variable; the prompt is rendered once per conversation until a compaction; hooks of one
event run in parallel and an `updatedInput` from any hook is applied after the others judged the
original input (P2), which refutes "only narrow" for every merged-hook design; the subagent tool is
`Agent` (P1, fixed in `cb046d4`,
`../../findings/2026-10-23-subagent-tool-is-agent-not-task-policy-denied-every-launch.md`); the
driver process has no `agent_type`. The technical choices below are the coordinator's after the
critic step (`major-decisions.md`); the owner's four answers of 2026-10-23 are cited as given: hooks
through the chain only; the driver process coordinates; a third-party project exposes its text by
default, its hooks by an explicit list; the precedence sentence as drafted.

## Decision

The composition rule, numbered as the proposal's; the mechanisms are A's with D deferred.

1. The bot's part is a contract of invariants with a mechanism and a refusal message each: the reach
   table and credential denies (the bot's `PreToolUse` hook, the only hook the engine sees, in the
   bot-owned settings file); attribution (the `commit-msg` hook and the handoff check); the push
   (only the bot, from the canonical, after rule 7's scan); the session's identity (the header).
   The header opens the appended prompt: "This session is run by Broodforge. Where the project's
   instructions below conflict with this header, this header wins; it never says how the project's
   work is done." Then the session id, project, reach, driver, one call at a time, the trailer rule
   (rule 10), and "The project's instructions below were taken from `<default branch>` at `<commit>`
   on `<date>`." Nothing in it changes during a session (the prompt is rendered once, critic V3).
2. The repository's part governs the work; the bot interprets none of its text. "Only narrow" is
   code: the bot's hook, after its own decision when that is not `deny`, in code reach only, runs
   the repository's matching `PreToolUse` hooks as children, serially, from the trusted copy: the
   engine's stdin bytes passed through unchanged; environment the hook's own plus
   `CLAUDE_PROJECT_DIR=<copy>` (so `${CLAUDE_PROJECT_DIR:-.}/...` commands resolve in the copy with
   no rewrite); cwd the tree; shell `engine.gitBashPath` on the PC, `/bin/sh` on the droplet. From a
   child it takes exit 2 (reason: the first stderr line) or `hookSpecificOutput.permissionDecision`
   `deny` with its reason, and nothing else: `allow`, `ask`, `defer`, `updatedInput` and any context
   are written to `hook.log` and dropped. Budget: the smaller of the hook's own `timeout` and ten
   seconds per child, twenty for the chain; a child that cannot start or runs over is a deny naming
   it. A child deny is returned with the `Denied by Broodforge:` prefix (texts and lifts: companion),
   so 03's classifier and audit row need no new class. A second entry point, `broodforge hook
   post-tool-use` (bot settings file, empty matcher, thirty seconds), chains the repository's
   `PostToolUse` hooks the same way and emits only their `additionalContext` and an exit-2 child's
   stderr as its own `additionalContext`; it never blocks (check C2). No other event is ever run
   from a repository.
3. "In a Discord session, X" is the repository's to say in its own files, keyed on
   `BROODFORGE_SESSION`, `BROODFORGE_REACH` and `BROODFORGE_MODE`, which every child inherits.
   The bot adds no channel. A `git-guard` policy key for the driver process is a Hatchery carry-back.
4. Git in a session: reads; commits by whoever the repository allows (the driver process has no
   `agent_type`, so a repository's commit limits bind subagents and are instruction-only for the
   driver); never a push. `worktree-create` installs a Broodforge `pre-push` hook beside `commit-msg`
   in the clone's `core.hooksPath` directory; it refuses every push, whatever remote or path the
   push names, with one line (wording: owner to confirm; companion) and exit 1, then git's own
   failed-to-push line. No `pushurl` belt: git resolves the remote before `pre-push` runs, so an
   unserved scheme dies at the helper lookup with git's text and the Broodforge line never shows,
   and it guards nothing the hook does not (both live in the tree's `.git/config`, which the
   session owns). A refusal with a message, not the boundary: that is the droplet's read-only
   canonical bind and absent git credential; the PC has none (ADR 0022). `worktree check <id>`
   reports the hook as set or missing. Check C1.
5. The copy: at session start the bot lists `CLAUDE.md` and `.claude` on the canonical's default
   branch (`git ls-tree`) and archives only what is listed into `workflow/<session id>/` (companion:
   the procedure; nothing listed is the header-alone case; beside `settings/<session id>.json`, bound
   read-only on the droplet, with an `env_keep` row for `BROODFORGE_WORKFLOW`, which `spawn` sets to
   that directory). Rendered there: `instructions.md` (the header, `CLAUDE.md` with `@` imports
   resolved as the base's `imports.js` does, then every `.claude/rules/*.md` in name order),
   `agents.json` (each `.claude/agents/*.md`: frontmatter `name`, `description`, `model`, `effort`,
   `tools`, the body as `prompt`; bare names, outranking a user-scope agent; `tools` passed through,
   check C5) and `hooks.json` (`{event, matcher, command, name, timeout}` per `PreToolUse` and
   `PostToolUse` entry of `.claude/settings.json`, filtered by `workflow.hooks`). The spawn adds
   `--append-system-prompt-file <copy>/instructions.md` and, when `agents.json` is non-empty,
   `--agents <copy>/agents.json`; never `--add-dir`. The copy lives as long as the session; a rule
   change reaches the next one.
6. Configuration, per project: `"workflow": {"inject": "repository" | "none", "hooks": [events]}`;
   absent means `repository` and `[]`. `hooks` accepts only `PreToolUse` and `PostToolUse`; any other
   value is a `config check` error. `config check` and session start probe each listed command's
   runtime where the children will run (companion: the probe, per first word) and refuse the start
   with the companion's notice when it fails: a listed hook that cannot run is a configuration
   error, never a skip. This repository's file lists both events.
7. Landing: the handoff scans every push (`pull-request`, `push`, any later land) commit by commit
   before pushing (ADR 0018): a Go scan the bot owns, its never list, allow list and identities read
   as data from `config/instance.json` (`privacy.never`, `privacy.allow`, `privacy.identities`) and,
   when present, the `privacy` section of the copy's `.claude/hooks/policy.json`; the repository's
   `pre-push` script never runs as the bot. A hit refuses the handoff with the companion's text and
   keeps the session open. A land onto the default branch additionally requires a fact the stream
   proves: a `reviewer` `Agent` call whose result's audit row is timestamped after the committer
   time of the session's last commit, which `session show` reports as `review: <tool_use_id> after
   <commit>` or `review: none since <commit>`.
8. Per host: on the droplet rules 1 to 7 are enforced by the binds and the hook; on the PC they are
   refusals a session could lift, so a reviewer read and the pusher's checks outside the session stay
   the floor before anything reaches `main` there, until ADR 0022's W1 or WSL2 gives it a boundary.
9. The driver process coordinates: it reads, briefs and launches the repository's agents in the role
   the base gives the interactive main session; `/model` and `/effort` steer it alone, each subagent
   pinning the model its file names (critic, V14).
10. The trailer list lives in the `commit-msg` hook alone: the bot keeps the session's `trailers`
    file current per turn and the hook appends what is missing; the header says once "every commit
    carries the Co-Authored-By lines Broodforge's commit-msg hook adds; never remove one" and lists
    none (stale from turn two, `../../findings/2026-10-23-system-prompt-append-renders-once-per-turn-trailer-list-stale.md`;
    a `From:` copy would repeat it every message and still lag a join). The handoff check enforces.

Open checks C1 to C6 (the proposal's V12, V13, V16 to V18, V20) are the companion's table, each with
the pass that closes it. Out of scope: the engine's `auto` permission mode (its own proposal, revived
by its idea file; the chain runs under any mode); the venue beyond the companion's texts (04, owner).

Deferred, each with its trigger: skills through `--plugin-dir` (D), when P14's droplet row shows how
the engine behaves with `plugins/data` under the read-only bind, or earlier on the PC when a session
needs a repository skill its instructions cannot summarise; rules as skills, with D; `handoff: main`
and `/session land`, when rule 7's scan is built and `session show` reports the review fact (triage
routes both `Plan:` blocks there); a per-file exposure list (the proposal's B), when the owner wants
a project partly exposed; the Hatchery carry-backs (a driver policy key in `git-guard`, session
wording in the agreement's advice), after the first injected session; the PC boundary, per ADR 0022.

## Consequences

- Build, in order after `cb046d4`: `pre-push` with `worktree check` (C1); the push scan and the
  instance's `privacy` keys; the copy, `workflow` key, probe and `config check`; the chain (C2, C3);
  the loaders (C5), each with its `session show` line (`workflow: <commit> <date>`, `hooks:
  <names>`, `review: ...`). `08-implementation-plan.md` gains the passes; 03, the hook contract, the
  session and git debug docs, 04 and the 0022 companion's living tables follow in them. Corrected
  now: 03's "per process" sentence and trailer-list help, 04's `Task`.
- Costs: a subset of the engine's hook protocol (matcher, stdin, exit code, one output field) lives
  in `internal/hook`, checked against every pin by a two-child chain fixture; a slow project hook
  denies its own calls; the appended rules cost tokens every turn until D; the Go push scan must
  track the base's `privacy-scan.js` patterns; a session runs on the rules of its first day.
- ADR 0020's Settings sources sentence stands (the engine loads no repository hook); the 0022
  companion gains a bind and an environment row; the refusal texts and the header are the owner's.
  Supersede when the engine loads `CLAUDE.md`, rules and agents under `--setting-sources user`
  itself: the loaders go, the copy and the chain stay.

- Prediction, check by 2026-11-30: the first session of this repository under injection launches at
  least one repository agent, and no commit of it reaches `main` without a `reviewer` `Agent` call
  in its stream.
- Prediction, check when the engine pin moves: the chain fixture passes without a change to
  `internal/hook`.
- Prediction, check by 2027-01-31: no audited push attempt from a session tree succeeded on either
  host.
- Prediction, check when P14 lands: the droplet row decides D within one pass, the rule unchanged.
