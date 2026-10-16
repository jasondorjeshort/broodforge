---
date: 2026-10-16
status: open
area: process / coordination files
---

# Per-change, per-claim or per-stream files, or no file, end shared-file conflicts, at a cost

## Question

How do projects and agent frameworks keep a shared "current state / what is next" file for many
parallel writers without constant merge conflicts? Evidence for the proposal on Hatchery's
`docs/HANDOFF.md` and its critic, after a downstream project split its handoff by workstream (its
ADR 0017: a near-static map of bare `@docs/handoff/<stream>.md` lines plus fixed `live.md` and
`waiting.md`, no commit hash, edited in the work commit). This finding recommends nothing.

## What was tried

Web survey on 2026-10-16 (WebSearch, WebFetch), `gh api` for repository state, and a local
`git merge-file` test with git 2.56.0.windows.2. A rebase-level test was not run: the `git-guard`
policy denies `commit` and `push` to this agent even in the scratchpad. Read the downstream ADR 0017
and its `docs/handoff/` (observed: a map of six bare `@` lines, stream files of 4 to 19 lines).
"States" is the source; "Inference" is mine. Undated pages are marked "fetched".

### 1. Changelog fragments

- towncrier README (github.com/twisted/towncrier, fetched; docs 26.9.0). States: "Rather than reading
  the Git history, or having one single file which developers all write to and produce merge
  conflicts, towncrier reads 'news fragments'". Files are `<issue>.<type>` in `newsfragments/`;
  `towncrier build` appends them to the news file and removes them "with `git rm`" (tutorial page).
- reno (github.com/openstack/reno README; docs.openstack.org/reno/latest/user/design.html and
  usage.html, fetched). States: "Reno stores each release note in a separate file to enable a large
  number of developers to work on multiple patches simultaneously ... without worrying about merge
  conflicts"; `reno new <slug>` adds a 16-hex suffix so authors need no special naming; `reno report`
  scans git history, and a note stays with the release that first introduced it; mistaken notes are
  hidden with `ignore_notes`, not deleted. Observed: openstack/nova `releasenotes/notes` holds 1192 files.
- changesets FAQ (github.com/changesets/changesets/blob/main/docs/common-questions.md, fetched).
  States: "Each changeset is its own file", random names "to avoid collisions"; on `changeset version`
  "all the changeset files are removed. This is so we only ever use a changeset once."
- GitLab, Robert Speicher, 2018-07-03 (about.gitlab.com/blog/solving-gitlabs-changelog-conflict-crisis/).
  States: constant CHANGELOG conflicts when MRs added entries at the same spot; placeholder lines
  "lessened, but not solved"; release managers hand-removed unrelated entries on every cherry-pick.
  Chosen: one YAML file per change in `changelogs/unreleased`, compiled at tagging and removed from
  stable and master. A real-release test found the tool deleted everything when no entries existed (fixed).
- GitLab 2021: delivery issue 1564 (gitlab.com/gitlab-com/gl-infra/delivery/-/issues/1564, cutover
  planned 2021-05-22) and epic 351. States: entries now come from a commit subject with a `Changelog:`
  trailer (docs.gitlab.com/development/changelog/); reason: not writing the same thing twice; entry
  files "don't end up improving the developer experience enough to justify the effort".
- CPython blurb (github.com/python/blurb, fetched). States: "designed to rid CPython core development
  of the scourge of Misc/NEWS conflicts"; entries at `Misc/NEWS.d/next/<section>/<UTC>.gh-issue-<n>.<nonce>.rst`,
  collisions "unthinkably unlikely"; `blurb release` folds `next/` into `<version>.rst`.
- Go (github.com/golang/go/blob/master/doc/README.md, fetched). States: one Markdown file per change
  under `doc/next/`, paths mirror packages; `relnote generate` concatenates them sorted by path at
  cycle end; then `doc/next` is removed and reseeded from `initial/`.
- Rust (forge.rust-lang.org/release/release-notes.html, fetched). States: a `relnotes` label makes
  triagebot file a tracking issue, "the source of truth"; in the first three weeks of beta the release
  owner runs the relnotes tool, inserts the output into RELEASES.md, and closes the issues.
- Kubernetes (kubernetes.dev/docs/guide/release-notes/, fetched). States: the note lives in the PR
  description; Prow labels gate the merge; cherry-pick automation copies the original PR's note.
- MediaWiki RfC T200392 (phabricator.wikimedia.org/T200392, 2018-07-26; written 2013) and T201514
  (2018-08-08). States: the cost was rebasing RELEASE-NOTES; `merge=union` rejected because "if one
  side removes a line and the other adds in the same place, it will re-add the removed line", silently;
  a custom driver "ate a line on the first trial run"; Gerrit's JGit has no merge drivers; approved
  2018-10-31, the bot "still needs to be written" in 2019; task open.

Inference: every project that adopted one file per change also has an assembly point (release,
cycle end) and a step that deletes the fragments; reno keeps them and its directory only grows.
Those that left files (GitLab 2021, Rust, Kubernetes) have tooling that reads commits or PRs.

### 2. Agent setups

Claude Code, all fetched 2026-10-16, pages undated:
- code.claude.com/docs/en/memory, "Import additional files". States: "Imported files can recursively
  import other files, with a maximum depth of four hops"; relative paths resolve from the importing
  file; "Import parsing skips Markdown code spans and fenced code blocks"; imports "load at launch"
  and "don't reduce its context cost"; a startup warning fires when files "add up past a combined
  limit", and "Each CLAUDE.md, rules file, and `@path` import counts as a separate file". The import
  section documents no glob or directory import.
- code.claude.com/docs/en/context-window, "What survives compaction". States: "Project-root CLAUDE.md
  and unscoped rules | Re-injected from disk". Whether its imports are re-expanded is not stated.
- code.claude.com/docs/en/sub-agents, "What loads at startup". States: a non-fork subagent gets
  "every level of the CLAUDE.md hierarchy the main conversation loads"; "The built-in Explore and Plan
  agents skip this"; `omitClaudeMd: true` drops it. Imports are not mentioned. Observed: this
  subagent's context carries `docs/HANDOFF.md` through the `CLAUDE.md` import. A subagent `memory`
  field gives `.claude/agent-memory/<name>/` (project scope, in git); the first 200 lines or 25KB of
  its MEMORY.md load; the main session's auto memory does not reach subagents.
- code.claude.com/docs/en/agent-teams and interactive-mode (experimental). States: the team task list
  is `~/.claude/tasks/{team-name}/`, outside the repository; "Task claiming uses file locking"; one
  mailbox JSON file per agent; `CLAUDE_CODE_TASK_LIST_ID` shares a list across sessions on a machine;
  "Two teammates editing the same file leads to overwrites. Break the work so each teammate owns a
  different set of files."
- code.claude.com/docs/en/hooks. States: `InstructionsLoaded` is observational; its matcher values
  include `include` and `compact`. Its input schema was not read.

Other frameworks and published practice:
- agents.md (fetched). States: "The closest AGENTS.md to the edited file wins"; no import syntax; no
  task state. OpenAI Codex (learn.chatgpt.com/docs/agent-configuration/agents-md, fetched): one file
  per directory, concatenated root-down until `project_doc_max_bytes` (32 KiB); no imports.
- OpenAI Cookbook, "Using PLANS.md" (developers.openai.com/cookbook/articles/codex_exec_plans,
  fetched). States: one ExecPlan per complex task with living sections Progress, Surprises &
  Discoveries, Decision Log, Outcomes & Retrospective; "it should always be possible to restart from
  only the ExecPlan".
- GitHub Copilot CLI (docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions,
  fetched). States: instruction files are combined with no precedence order; `@relative/path`
  includes expand in copilot-instructions.md, AGENTS.md and CLAUDE.md, not in `*.instructions.md`.
- Cursor (cursor.com/docs/context/rules, fetched). States: one `.mdc` per rule in `.cursor/rules`;
  an `@file` in a rule is read by tools on demand, "not inlined into the prompt".
- GitHub Spec Kit (github.com/github/spec-kit/blob/main/spec-driven.md, fetched). States: one
  directory per feature, `specs/NNN-name/` with spec.md, plan.md, tasks.md, on its own branch.
- Anthropic, "Effective harnesses for long-running agents", Justin Young, 2025-11-26
  (anthropic.com/engineering/effective-harnesses-for-long-running-agents). States: one agent at a
  time; `claude-progress.txt` plus a JSON feature list, since the model is "less likely to
  inappropriately change or overwrite JSON files"; each session reads git log and the progress file.
- Anthropic, "Building a C compiler with a team of parallel Claudes", Nicholas Carlini, 2026-02-05
  (anthropic.com/engineering/building-c-compiler). States: an agent claims a task "by writing a text
  file to current_tasks/"; "git's synchronization forces the second agent to pick a different one";
  it pulls, merges, pushes, removes the lock; "Merge conflicts are frequent, but Claude is smart
  enough to figure that out"; READMEs and progress files "updated frequently".
- beads (github.com/steveyegge/beads, fetched). States: "replaces messy markdown plans with a
  dependency-aware graph"; "Hash-based IDs ... prevent merge collisions in multi-agent/multi-branch
  workflows"; Dolt storage with cell-level merge. mcp_agent_mail (github.com/Dicklesworthstone/mcp_agent_mail):
  one markdown file per message plus inbox and outbox copies in git; advisory file leases.

Inference: Claude Code and Copilot CLI are the only tools found that inline `@` includes; a Codex or
Cursor reader of the same tree would see a map of import lines, not the streams. Tools built for
many simultaneous agents keep live state outside git (task list with file locks, Dolt) or as one
file per claim.

### 3. Git on files every contributor edits

- gitattributes (git-scm.com/docs/gitattributes). States: union "take[s] lines from both versions
  ... tends to leave the added lines in the resulting file in random order and the user should verify
  the result. Do not use this if you do not understand the implications." Custom drivers are defined
  in git config, so every clone installs them; GitLab MRs do not run them (gitlab-org/gitlab issue
  18830) and neither does Gerrit (T200392). The page names merge, revert and cherry-pick, not rebase.
- Experiment, `git merge-file`: two sessions' edits to a handoff's Updated line and Now bullet, under
  `--union`, exit 0 with two Updated lines and two `## Now` headings; the default driver conflicts.
  Two appends to a list merge cleanly. One session deleting an answered question while another
  inserts next to it brings the answered question back, with exit 0.
- git-rerere (git-scm.com/docs/git-rerere). States: replays a recorded hand resolution when the same
  conflict recurs, for "long lived topic branches"; rebase uses it. Inference: handoff conflicts are
  new text each time, so there is nothing to replay.
- Linux, deleting the file: Torvalds, commit 9c0ece069b32, 2012-10-01 (via `gh api`):
  `Documentation/feature-removal.txt` "has turned out to be a pure write-only file that causes merge
  conflicts and has no actual redeeming features."
- Linux, splitting proposed: LWN, 2017-08-10 (lwn.net/Articles/730509/). States: MAINTAINERS takes
  "on the order of 100 patches in each development cycle"; a script split it into 54 files; Torvalds
  "tried it out but chose not to go with the result". Observed: still one 928,854-byte file.

### 4. Accounts of splitting a status file

- ehky2882/TRAVEL-GUIDED-TOUR PR 845, merged 2026-09-13 (github.com/ehky2882/TRAVEL-GUIDED-TOUR/pull/845),
  written from Claude Code sessions. States: three PRs in two days blocked on STATUS.md; "67 commits
  touched STATUS.md in seven days", all on the same three line numbers; replaced by
  `status/entries/<UTC>-<branch>.md` (never edited), `status/owner/<slug>.md` (cleared by `git rm`)
  and `status/builds/<n>.md`, written by a script that refuses to overwrite; a hand-kept branch table
  was deleted as derivable from git; STATUS.md kept only deliberately revised sections (144 KB to
  36 KB); a start script prints the owner items to every session. Observed 2026-10-16: one STATUS.md
  commit since the split; 279 entry files and 17 owner files.
- Not found: a team that split such a file and went back. MediaWiki and Linux MAINTAINERS are splits
  proposed and not made.

## Result

Patterns found, with the conditions under which the sources use them:

1. One file per change, assembled and deleted at a release (towncrier, changesets, blurb, Go, GitLab
   2018). Many writers, a high commit rate, a tool as the reader at a fixed point. Costs: an assembly
   step, a deletion step that can misfire, and drift if fragment names are fixed.
2. One file per change kept and read from git history (reno). Needs a reading tool; grows forever.
3. No file: commit trailers, PR fields or tracker issues (GitLab 2021, Kubernetes, Rust). Writers never
   conflict; needs collection tooling; nothing loads it into a session.
4. One file per claim, created to claim and deleted when done (C compiler `current_tasks/`). Push is
   the lock; fits short-lived claims and a directory that is near-empty at rest.
5. One file per stream or feature with living sections (Codex ExecPlans, Spec Kit, ADR 0017). One
   writer per file at a time; the reader must be pointed at the file (AGENTS.md line, branch, map).
6. Append-only entries plus a small deliberately edited core (TRAVEL-GUIDED-TOUR). Many writers;
   growth is unbounded unless pruned.
7. Live state outside git in a lock-aware store (Claude Code task list, beads, agent mail). Many
   simultaneous agents with tooling; the Claude list is machine-local and not reviewed in a PR.
8. One file, git made to cope: union only for unordered append-only lists with no deletions; custom
   drivers only where every clone and the server run them; rerere only for recurring conflicts.
9. Delete the file and derive its content (Linux 2012, the TRAVEL branch table) where git already
   knows it or nobody reads it.

Inference: in every account the conflicts came from parallel writers on the same lines; the
one-writer setup (the long-running harness) kept a single progress file. Only patterns 5, 6 and 7
have a reader at session start; the fragment patterns assume a tool that reads at release time.

Claude Code facts confirmed from the docs: nested imports to four hops; code spans and fences
skipped; imports cost full context and count toward the combined-size warning; Explore and Plan load
no CLAUDE.md. Not confirmed: re-expansion of imports after compaction; a mid-session edit by another
session reaching a running one. ADR 0017 says "up to five hops"; the docs fetched today say four.

## Next

For the critic to settle:

1. Hatchery's reader is every session and subagent at launch, not a release tool. Which pattern fits
   that reader, and what is the context cost per launch when every open stream loads?
2. Who deletes a closed stream file, and what catches a missed delete? Patterns 1 and 4 delete by
   tool or by the claimant; ADR 0017 defers its check.
3. Are fixed-name files (`live.md`, `waiting.md`, the map) the next hot spot? TRAVEL-GUIDED-TOUR gave
   each owner item its own file, cleared by `git rm`, and the union test shows a deleted answer
   returning.
4. What does a project with one coordinator and one stream, as the base itself runs, gain from a split?
5. Import lines are inert in Codex, AGENTS.md readers and Cursor; does that matter for decanted projects?
6. Unverified by test: imports re-read after `/compact`; union under `git pull --rebase` (the guard
   denied the commits this test needs; a runner can run it).
7. Sources looked for and not read: gnulib `git-merge-changelog` (HTTP 429), Aider's and Devin's
   in-flight-state docs, Kubernetes `kep.yaml`, and whether PEP 0 is generated from PEP headers.
