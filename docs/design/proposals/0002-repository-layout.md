# 0002. Repository layout for agent-driven projects

Date: 2026-10-11
Status: promoted to ADR 0002

Revised 2026-10-11 after the critic review (`0002-critic-review.md`). The section bodies keep the
original design for the record, with factual errors corrected in place and marked "[corrected]".
The final design is in `../decisions/0002-repository-layout.md`; the changes and their reasons are
in "Critic's objections and responses" at the end.

## Question

Hatchery is forked to start new projects. Beyond the working agreement, six agents, usage gate,
and design skeleton it already has, what should the base contain so that Claude Code agents work
well in every decanted project: where knowledge and working notes live, which skills ship, which
hooks run, what permissions are pre-approved, and what is deliberately left out?

The design is built around how Claude Code discovers files, not around human filing habits:

| Mechanism | Loaded | Cost model |
|---|---|---|
| `CLAUDE.md` (+ `@imports`) | every session, every subagent | paid on every turn; keep tiny |
| `.claude/rules/*.md` with `paths:` | only when matching files are touched | paid only in scope |
| `.claude/skills/<name>/SKILL.md` | on description match or `/<name>`; subagents on demand through the `Skill` tool when it is in `tools:`, or by `skills:` preload [corrected] | descriptions always indexed, body on demand |
| `.claude/agents/*.md` | when that agent is launched | per launch |
| Hook stdout (`SessionStart`) / block reason (`Stop`) | at the event | per event, main session only |
| Everything under `docs/` | only when a brief or an index points at it | zero until read; needs indexes |
| Claude auto-memory (`~/.claude/...`) | per machine and path | does not travel; cannot be relied on |

Design rule that follows: anything needed every turn goes in `CLAUDE.md`; procedures go in
skills; path-scoped conventions go in rules; state and knowledge go in `docs/` with an index per
directory and a 300-line cap per file; session state is pushed into context by a hook rather
than hoped to be found.

## Constraints that matter here

- Agents code and debug; the main session coordinates and writes briefs. The layout must make
  briefs short: a brief should be able to say "read `docs/reference/README.md`" and be done.
- Language-neutral base. Formatters, linters, CI, and test commands are slots, not content.
- Usage gating: session continuity must survive an abrupt stop at 80%.
- Every directory and file added is something every future agent has to learn. Scale is one
  owner, one coordinator session at a time, a handful of in-flight items. Do not design for more.
- Node is available (hooks are Node); no jq. Python available but not needed.
- GPL-3.0-or-later with SPDX headers on source files, hook scripts included. [corrected: the two
  existing `.js` files already carry the header on line 2.]

## 1. Directory layout

```
CLAUDE.md                   always loaded; rules only; points at everything else (target <= 80 lines)
README.md                   for humans and GitHub; the decant checklist moves to the decant skill
LICENSE, LICENSES/          as today
.claude/
  settings.json             status line, hooks, permissions (committed)
  settings.local.json       per-machine; gitignored (already)
  agents/                   the six subagents (unchanged roles; frontmatter gains skills:)
  skills/<name>/SKILL.md    procedures; supporting files beside SKILL.md (section 3)
  hooks/*.js                hook scripts (section 5)
  scripts/check-docs.js     doc linter run by the handoff skill and the pusher (not a hook)
  rules/docs.md             path-scoped conventions for docs/** (file caps, indexes, frontmatter)
  rules/README.md           how to add a per-language rule once code exists (the slot)
  state/                    hook scratch (session baselines); gitignored
docs/
  HANDOFF.md                where things stand; the only in-flight-work file (section 2)
  backlog.md                ideas and deferred work, one file (section 8)
  design/                   what we are building: 00-06, proposals/, decisions/ (unchanged)
  reference/                durable facts about the world the project depends on, one topic per file
  findings/                 dated, status-tagged results of investigations
```

Rule for what goes where, in the order an agent should ask:

1. Is it a rule Claude must obey on every turn? `CLAUDE.md`. If only when touching certain
   paths, `.claude/rules/<name>.md` with `paths:`.
2. Is it a procedure (steps to do X)? A skill. Never `docs/`, never `CLAUDE.md`.
3. Is it a decision about what we build or why? `docs/design/` (ADR if non-obvious).
4. Is it a fact about a tool, API, platform, or environment that will stay true? `docs/reference/`.
5. Is it the result of an investigation (a bug's root cause, a benchmark, a failed approach)?
   `docs/findings/`, dated and status-tagged. Promote the durable part to `reference/` or
   `design/` when resolved.
6. Is it what is happening right now or next? `docs/HANDOFF.md`. Later than next? `docs/backlog.md`.

Fate of the owner's previous layout:

| Old | New | Why |
|---|---|---|
| `docs/reference/` | `docs/reference/` | Same purpose. Adds `README.md` index and `verified:` frontmatter date per file. |
| `docs/active/` | `docs/HANDOFF.md` | One file, hook-injected, hard length cap. A directory of in-flight notes goes stale and nobody reads it. |
| `docs/active/findings/` | `docs/findings/` | Findings are not transient; they are the evidence trail. Frontmatter status separates open from done. |
| `docs/active/ideas/`, `docs/active/future/` | `docs/backlog.md` | Both are "not now". One file with two sections; an idea that gets a trigger condition becomes deferred work. |
| `docs/active/test/` | none | Test procedures are the debug interface (`05-debug-interface.md`); test results and flake notes are findings. |
| `docs/skills/` | `.claude/skills/` | Claude only auto-discovers skills there. A skill in `docs/` is a doc nobody loads. |

File shapes:

- `docs/reference/<topic>.md`: frontmatter `verified: YYYY-MM-DD`, `source: <url or "observed">`.
  Body: facts, commands that work, gotchas. 300-line cap; split by topic, not by date.
- `docs/findings/YYYY-MM-DD-<slug>.md`: frontmatter `date`, `status: open | resolved | superseded`,
  `area: <free text>`, optional `supersedes: <file>`, `promoted_to: <path>`. Sections: Question,
  What was tried, Result, Next (if open). 150-line cap.
- Every directory under `docs/` has a `README.md`: one line per file, open items marked. The
  `check-docs.js` script fails if a file is missing from its index.

Options considered for the knowledge split: (A) everything under `docs/design/` with subfolders;
rejected because design docs are authoritative and reviewed while findings are provisional, and
mixing them is how stale findings get read as truth. (B) a single `docs/notes/` with frontmatter
`kind:`; rejected because directory is the discovery unit for agents (Glob, README index), and
frontmatter is only visible after reading. (C) as recommended.

## 2. Session continuity

Options: (A) `@docs/HANDOFF.md` import in `CLAUDE.md`. Simplest, but imports load into every
subagent too, and subagents must not inherit the coordinator's in-flight state (it pollutes
their brief and costs tokens on every launch). (B) `SessionStart` hook that prints the handoff.
Fires for the main session only (to verify: SessionStart does not fire for subagents) and also
on `compact`, so the handoff is re-injected exactly when context was just lost. (C) rely on
auto-memory; rejected, it does not travel. Recommendation: B, with a `Stop` hook as a nudge.

`docs/HANDOFF.md`, hard cap 60 lines (the hook warns above 60, the linter fails above 80):

```
# Handoff
Updated: YYYY-MM-DD HH:MM, at commit <short sha>, by <main session | agent name>

## Now            1-3 bullets: what is in flight, which agent had it, exact state (files, failing test)
## Next           ordered, 3-7 bullets, each one actionable in a single brief
## Waiting on user   questions that need a human decision (front-end, wording, scope); or "none"
## Recently done  last 5 commits that matter, one line each with sha; older history is git log
## Open findings  links to docs/findings entries with status: open; or "none"
```

Anything that does not fit moves to `backlog.md` (later) or a finding (evidence). The handoff
never contains procedure or rationale; it links.

`SessionStart` hook (`.claude/hooks/session-start.js`, matcher: all sources) prints to stdout:
the handoff file verbatim, then `git status -sb` (first 15 lines), `git log --oneline -5`, the
count of open findings, and the current usage line if `~/.claude/usage-state.json` is fresh. It
also writes `.claude/state/<session_id>.json` with `{ head, handoff_sha256, started_at }` as the
baseline for the Stop hook, and deletes state files older than 7 days.

`Stop` hook (`.claude/hooks/handoff-check.js`) blocks the stop with a reason when all hold:
`stop_hook_active` is false; the session changed something (tree dirty or `HEAD` differs from
the baseline); `docs/HANDOFF.md` hash equals the baseline; and it has not blocked yet this
session (flag in the state file). Reason text: "This session changed files but
docs/HANDOFF.md is unchanged. Run /handoff now, or reply in one line why nothing needs
recording." Blocking once per session is the deliberate trade: it guarantees at least one update
without nagging every turn. The `handoff` skill and the usage gate's STOP message cover the end
of long sessions. Rejected alternative: block on every stop while stale, because `Stop` fires
after every assistant turn, and a block costs a full extra model turn each time.

Also in continuity: when the usage gate denies at 80%, its reason already says "write a handoff
note"; that text should name `/handoff` so the coordinator runs the skill instead of improvising.

## 3. Skills

Skills ship under `.claude/skills/<name>/SKILL.md` with frontmatter `name` and `description`.
Descriptions are written to trigger: they name the situation, not the file. Supporting files
sit beside `SKILL.md` and are read only when the procedure says so.

| Skill | Verdict | What it does and when it triggers |
|---|---|---|
| `proposal` | accept | Runs the major-choice loop end to end: allocate the next NNNN, write `proposals/NNNN-title.md` from `template.md`, brief the critic (`brief-critic.md`), fold objections into the "Critic's objections and responses" section, then hand to `adr`. Triggers on choosing a language, toolkit, storage, protocol, build system, or core dependency, or on "proposal". |
| `adr` | accept, separate | Write a new ADR from `template.md`, or supersede one (set `Status: superseded by NNNN` on the old, link both ways), and update `decisions/README.md` and `design/README.md` indexes. Triggers on "record this decision", "ADR", "supersede". Kept separate because non-major decisions get ADRs without a proposal (architect rule), and supersession is its own procedure. |
| `handoff` | accept | Rewrite `docs/HANDOFF.md` to the template in section 2 from `git log`, `git status`, open findings, and the session's own knowledge; run `node .claude/scripts/check-docs.js`; report what was dropped to backlog. Triggers on "handoff", "wrap up", usage HOLD/STOP, or the Stop hook's reason. |
| `debug-interface` | accept | For one feature: name the surface (CLI, HTTP, REPL, harness), the commands to drive it, the state it exposes, how a bug is reproduced from a terminal; write that as a section of `05-debug-interface.md`; verify by running the commands and pasting output. Triggers when a feature is designed or a coder is briefed. |
| `finding` | accept | Record an investigation result at `docs/findings/YYYY-MM-DD-slug.md` with the frontmatter in section 1; add it to the index; on resolution, promote the durable fact to `reference/` or `design/` and set `promoted_to`. Triggers on "root cause", "we found that", "benchmark", "this did not work because". |
| `decant` | accept | The README checklist as steps, plus: reset `HANDOFF.md` and `backlog.md` to empty templates, delete `findings/*` and `reference/*` content except READMEs, set the `CLAUDE.md` title and platform list, keep ADRs 0001 and 0002 (they still apply), add the `hatchery` remote. Last step: commit via the runner. Triggers on "/decant" or "new project from Hatchery". |
| `commit` | accept, absorbs `push` | Produces the runner brief: files to stage (from `git status --short`), subject line, body, `Co-Authored-By` for the model that did the work; refuses to brief if the tree contains files nobody described. A final section "Pushing" gives the one-line pusher brief. Triggers on "commit", "push". |
| `push` | reject | The pusher's checks live in `pusher.md` and the brief is one line ("push `main`"). A skill would duplicate the agent file. |
| `brief` | add | Templates for briefing each agent (`templates/<agent>.md`): goal, files to read (by index path), what is ruled out, scope boundary, verification, report format. The coordinator writes briefs every few turns; a template keeps them complete and short. Triggers on "brief the coder", "launch", "delegate". |

Rejected: `review` (already the architect's rules), `usage` (built in), `reference` (folded into
`finding` as promotion), `test` (language-specific; the decanted project adds it).

Preloading via `skills:` frontmatter. [corrected: a subagent with `Skill` in `tools:` loads any
project skill on demand; our agents lacked it only because their `tools:` lists are explicit. The
table below was the original plan and is not adopted; see objection 1.] Preload where the agent is
the author of the artifact the skill produces:

| Agent | `skills:` | Why |
|---|---|---|
| architect | `proposal, adr` | Writes both artifacts; needs the templates without a brief naming them. |
| coder | `debug-interface, finding` | Builds the debug surface for every feature; debugging produces findings. |
| tester | `debug-interface` | Tests drive features through the same surface. |
| critic, runner, pusher | none | Their whole procedure is in their agent file. |

The main session gets everything on demand. Each SKILL.md stays under 100 lines; detail goes in
supporting files.

## 4. Agent memory

`memory: project` gives an agent a persistent directory in the repo that it reads at start and
writes freely (to verify: exact path, whether it is committed by default, and how much of it is
auto-loaded). Options: (A) none; (B) `coder` only, for build and environment quirks; (C) all
agents.

Recommendation: A, none. This design already provides shared, indexed, reviewable memory
(`reference/`, `findings/`, `HANDOFF.md`). Per-agent memory would split knowledge into a second
place that other agents and the human do not read, has no index or status, grows without a cap,
and is written by the agent least able to judge what is durable. The coder's build quirks belong
in `docs/reference/` where the tester and the pusher also find them.

What would change this: evidence of repeated rediscovery, meaning the same fact appears in two
findings, or coder briefs keep having to restate the same environment facts. Then enable
`memory: project` for `coder` alone with a rule in its agent file that memory holds only pointers
into `docs/reference/` and is trimmed on every handoff.

## 5. Hooks

All hook scripts are Node, read JSON on stdin, never throw (a hook failure must not stall a
session), and carry the SPDX header. Added to `.claude/settings.json`:

| Event | Matcher | Script | Purpose |
|---|---|---|---|
| `SessionStart` | all (`startup`, `resume`, `clear`, `compact`, `fork` [corrected]) | `session-start.js` | Inject handoff, git state, open-finding count; record baseline (section 2). |
| `Stop` | none | `handoff-check.js` | Block once per session when work happened and the handoff is unchanged (section 2). |
| `PreToolUse` | `Bash` | `git-guard.js` | Deny destructive git regardless of who runs it: `push --force`/`-f`/`--force-with-lease`, `reset --hard`, `clean -f`, `checkout -- .`/`restore .` on the whole tree, `branch -D`, `rebase` with `-i`. Regex over the full command, so argument order does not matter. Hooks apply to subagent tool calls too (to verify), which makes the runner's and pusher's prose rules enforceable. |
| `PostToolUse` | `Write\|Edit` | `post-edit.js` | Language-neutral checks returned as `additionalContext`, never blocking: a `docs/**` file over 300 lines, `HANDOFF.md` over 60 lines, a new `findings/` file without frontmatter. Contains the formatter slot: a `formatters` table keyed by extension, empty in the base, with a comment telling the decanted project to add `['.rs', 'rustfmt --edition 2021']` or the like. |
| `PreToolUse` | `Agent\|Workflow` | `usage-gate.js` | Exists. Change only the STOP reason text to name `/handoff`. |

Considered and rejected:

- `PreCompact` to force a handoff update: cannot block, and the model is not in control at that
  moment. `SessionStart` with source `compact` re-injects the handoff afterwards, which is the
  real need.
- `SubagentStop` to log agent runs: the coordinator already receives the report; a log file
  nobody reads is noise.
- `UserPromptSubmit` to inject usage or reminders: the status line shows usage, and injecting on
  every prompt is the cost model this design avoids.
- `PreToolUse` denying `Write|Edit` on `CLAUDE.md` and `.claude/settings.json` from subagents:
  attractive and feasible [corrected: hook input carries `agent_id` and `agent_type` inside a
  subagent]. Deferred with a trigger in the ADR; `git-guard` uses `agent_type` for push and
  commit caller rules instead.
- Auto-format on `PostToolUse`: shipped only as the slot above; the base has no language.

## 6. Permissions allowlist

Committed in `.claude/settings.json`. Only commands that cannot change state. [corrected: a
trailing ` *` also matches the bare command, so `Bash(git status)` beside `Bash(git status *)` is
redundant; `Bash(git branch)` is kept because `--list *` does not cover it. `git fetch` writes
remote refs and is removed. The list below is the original; the final list is in the ADR.]

```json
"permissions": {
  "allow": [
    "Bash(git status)", "Bash(git status *)", "Bash(git log *)", "Bash(git diff *)",
    "Bash(git show *)", "Bash(git branch)", "Bash(git branch --list *)", "Bash(git rev-parse *)",
    "Bash(git remote -v)", "Bash(git fetch *)", "Bash(git stash list)", "Bash(git blame *)",
    "Bash(ls *)", "Bash(cat *)", "Bash(head *)", "Bash(tail *)", "Bash(wc *)",
    "Bash(node --version)", "Bash(python --version)",
    "Bash(node .claude/scripts/check-docs.js *)", "Bash(node .claude/scripts/check-docs.js)"
  ],
  "deny": [
    "Read(./.env)", "Read(./.env.*)", "Read(./**/*.pem)", "Read(./**/*.key)"
  ]
}
```

Not pre-approved on purpose: `git add`, `git commit` (the runner is briefed per commit and a
prompt is a useful checkpoint), `git push` (pusher only), anything with `rm`, `curl`, package
managers. Destructive git is handled by the hook rather than deny rules because [corrected] a
deny rule matches the whole command text with wildcards but "doesn't match the same program by
path or inside `sh -c`", and it cannot see which agent is calling. The decanted project extends
the list by hand with `/permissions` or "Yes, and don't ask again" after a few sessions
[corrected: there is no built-in allowlist-mining skill].

## 7. What the base does not contain

- No language toolchain, CI workflow, test runner, formatter, or linter config. Slots: the
  `formatters` table in `post-edit.js`, `.claude/rules/README.md`, and pusher checks 7 and 8,
  which already look for a test and lint command in `CLAUDE.md`.
- No `.claude/rules/<language>.md`; the base cannot know the language. One example rule
  (`docs.md`) shows the format.
- No `.claude/commands/`: legacy mechanism; skills supersede it.
- No `.claude/agent-memory/` or equivalent (section 4).
- No `docs/active/`, `docs/ideas/`, `docs/future/`, `docs/test/`, `docs/journal/`,
  `docs/meetings/`: each is a place for stale text (section 1).
- No `docs/findings/archive/`, no per-platform doc folders, no `templates/` directory at the
  repo root: templates live inside the skill that uses them. Add archive only when the findings
  index itself exceeds the 300-line cap, which this project may never reach.
- No GitHub issue or PR templates, no `CONTRIBUTING.md`, no `CHANGELOG.md`: one owner, agents as
  contributors, git log as the changelog.
- No MCP server configuration: nothing in the process needs one yet.
- No `@imports` in `CLAUDE.md`: every import is paid by every subagent on every launch.

## 8. Tracking in-flight work

Options: (A) files in the repo (`HANDOFF.md` for now and next, `backlog.md` for later);
(B) GitHub Issues through `gh`; (C) both, with issue numbers referenced from the files.

Recommendation: A. Reasons: files are discoverable with the tools agents already use (Glob,
Grep, Read) and by the SessionStart hook; they version atomically with the work they describe;
they travel with a fork and work offline and on a machine without `gh` auth; the scale is a
handful of items, where Issues add a tool call per lookup and a network dependency for no gain.
B is rejected for the base; the owner may use Issues as a personal queue, but anything an agent
must act on is in the repo. C is rejected because two sources of truth drift within a week.

`docs/backlog.md`, cap 150 lines, two sections: `## Ideas` (one line each, no commitment) and
`## Deferred` (one to three lines each: what, why not now, what would trigger it). An item moving
into work is cut from here and pasted into `HANDOFF.md` under Next; nothing is tracked twice.

What would change this: a second regular human contributor, parallel coordinator sessions in
worktrees that need assignment and locking, or a public project where users file bugs. Then
Issues become canonical for intake and `backlog.md` is retired by superseding the ADR.

## Recommendation

Adopt the layout in section 1 and the file shapes given; `docs/HANDOFF.md` injected by a
`SessionStart` hook and guarded once per session by a `Stop` hook; eight skills (`proposal`,
`adr`, `handoff`, `debug-interface`, `finding`, `decant`, `commit`, `brief`) with preloads for
architect, coder, and tester; no agent memory; four hook scripts plus the existing gate; the
read-only allowlist in section 6; a `docs.md` path-scoped rule and a `check-docs.js` linter;
file-based tracking of in-flight work. `CLAUDE.md` shrinks to rules and gains a six-line
"Where things live" section pointing at `docs/HANDOFF.md`, `docs/design/README.md`,
`docs/reference/README.md`, `docs/findings/README.md`, `docs/backlog.md`, and `.claude/skills/`.

Implementation order once promoted: hooks and settings (continuity first), the `docs/` tree with
READMEs and the rule, then skills, then agent frontmatter, then `CLAUDE.md` and README edits.
Each step is one runner commit.

## What would change this recommendation

- If `SessionStart` turns out to fire for subagents, or hooks do not apply to subagent tool
  calls, the continuity and git-guard designs need a different mechanism (likely an `@import`
  plus `omitClaudeMd` on agents that must not see it, or per-agent `disallowedTools`).
- If subagents can invoke skills on demand, drop most preloads and rely on descriptions.
- If `Stop` cannot block in the current release, replace the handoff check with a line in the
  usage gate's reasons and the `commit` skill ("update the handoff before briefing a commit").
- If project-scope agent memory is committed and capped by the tool, revisit `coder` memory.
- If the owner adopts GitHub Issues for intake, section 8 flips and `backlog.md` goes.

Claims the critic should verify, collected: SessionStart does not fire for subagents and does
fire on `compact`; Stop hooks can block with `{"decision":"block","reason":...}` and expose
`stop_hook_active`; PostToolUse `additionalContext` reaches the model; PreToolUse hooks run for
subagent tool calls; `skills:` preload exists for agents and subagents cannot otherwise load
skills; `.claude/rules` `paths:` frontmatter syntax; `memory: project` location and commit
status; permission rule prefix semantics for `Bash(cmd *)` versus bare `cmd`.

## Critic's objections and responses

Numbering follows `0002-critic-review.md` Part 2. Resolutions are recorded in ADR 0002.

1. Preloads are the wrong default. **Accepted, and taken further.** No `skills:` preloads at all;
   every agent gets `Skill` in `tools:` and a one-line pointer naming the skills relevant to it.
   The critic's suggestion to keep `debug-interface` preloaded for the coder is not taken: the
   obligation is already stated in `coder.md`, the skill loads on demand when a feature brief
   arrives, and a bug-fix launch should not pay for it.
2. The Stop hook has three correctness holes. **Accepted.** The Stop hook and `.claude/state/` are
   deferred. The three holes (baseline rewritten on `compact`, deleted handoff, `session_id` across
   `/clear`) are recorded as preconditions in the ADR's deferred list and in `docs/backlog.md`.
3. The `@import` is the better continuity mechanism. **Accepted.** `CLAUDE.md` imports
   `docs/HANDOFF.md`; the first line marks it as coordinator state. The SessionStart hook is
   deferred with the trigger "a session resumes without knowing where it is". The handoff cap is
   tightened to 40 lines (coordinator decision), which bounds the per-launch cost.
4. Usage gating defeats the handoff. **Accepted, third option refined.** The gate blocks only
   `Agent|Workflow`, never `Bash`, so the main session makes wrap-up commits (`HANDOFF.md`, WIP)
   itself at any usage level; the runner rule governs normal work below 50%. Stated in both the
   Usage gating table and the Git section of `CLAUDE.md`, and in the `handoff` and `commit` skills.
5. Node is a silent single point of failure. **Accepted in part.** The `decant` skill verifies
   `node --version` and `gh --version` as step one and says what breaks without them; `README.md`
   names both as prerequisites; `git-guard` gets `"onFailure": "block"` (hooks reference: requires
   2.1.295 or later; installed 2.1.296), so a missing Node blocks `Bash` loudly rather than
   disarming the guard. `post-edit` stays non-blocking. The SessionStart shell fallback is moot
   because the hook is deferred.
6. Four programs encode the same constants; the linter has no home. **Accepted.** `check-docs.js`
   is dropped. Two numbers (300 for any `docs/**/*.md`, 40 for `HANDOFF.md`) live in
   `.claude/rules/docs.md` and are enforced in one place, pusher check 7 (`wc -l`). `post-edit`
   warns with the same numbers but enforces nothing. The critic's "post-edit as the single
   enforcement point" is not taken because a warning is not enforcement. `limits.json` is the
   recorded step if a third point ever appears.
7. Allowlist details. **Accepted with one exception.** Bare `Bash(git status)` dropped; `git fetch`
   dropped; `Read(*.pem)`/`Read(*.key)` simplified; deny reasoning reworded to the documented
   `sh -c` and path bypass. Exception: `Bash(git branch)` stays, because `Bash(git branch --list *)`
   does not match bare `git branch`, and `Bash(git branch *)` would admit `git branch -D`.
   `git ls-files *`, `git branch --show-current`, and `gh --version` added (read-only, used by the
   pusher check and the skills). Noted: `cat`/`head`/`tail` on a `docs/` file trigger the rule.
8. Agent memory trigger already met. **Accepted.** The rejection rests on the unindexed second
   location; the "committed and capped" conditional is dropped; the trigger is repeated rediscovery.
9. Handoff template duplicates the hook. **Accepted.** "Recently done" removed; sections are Now,
   Next, Waiting on user, Open findings.
10. Backlog has a cap and no overflow rule. **Accepted.** Rule written into `backlog.md` and
    `.claude/rules/docs.md`: oldest ideas deleted, stale deferred items become superseded findings.
11. The `brief` skill duplicates the agent files. **Accepted.** One template plus six one-line notes;
    the skill says agent rules live in the agent file and are not restated.
12. Factual and wording fixes. **Accepted.** All corrected in place above and marked. The
    "deny `CLAUDE.md` edits from subagents" hook is feasible but is a `Write|Edit` matcher, not a
    `Bash` one, so it is a separate hook; deferred with a trigger rather than folded into git-guard.
    What is folded into git-guard, because `agent_type` makes it free: deny `git push` from any
    subagent other than `pusher` and `git commit` from any subagent other than `runner`.
13. Issues and `push` residuals. **Accepted.** ADR states the file wins over Issues until superseded.

Changes made on the architect's or coordinator's initiative, not raised by the critic: `CLAUDE.md`
target is 130 lines, not 80 (the current rules do not fit in 80 without losing content); a ninth
skill, `update-from-hatchery`, added at the user's request together with two paste-able prompts at
the top of `README.md`; the hook specifications moved to `../decisions/0002-hook-specs.md` so the
ADR stays under 150 lines; `docs/design/proposals/` is exempt from the 300-line pusher cap, since
this very file (already 332 lines before revision, 400 after) is a closed record that nothing loads
on demand, and trimming it would mean deleting the alternatives or the objections.
