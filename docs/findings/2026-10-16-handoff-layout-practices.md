---
date: 2026-10-16
status: resolved
area: process / coordination files
promoted_to: docs/design/decisions/0014-handoff-layout.md
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

- towncrier (github.com/twisted/towncrier, fetched): reads "news fragments" instead of "one single
  file which developers all write to"; `<issue>.<type>` in `newsfragments/`, removed by `build`.
- reno (github.com/openstack/reno; docs.openstack.org/reno/latest/user/design.html and usage.html,
  fetched): one file per note "without worrying about merge conflicts", a 16-hex suffix, notes read
  from git history, mistakes hidden with `ignore_notes`. Observed: nova holds 1192 note files.
- changesets (github.com/changesets/changesets/blob/main/docs/common-questions.md, fetched): "Each
  changeset is its own file", random names; on `changeset version` "all the changeset files are removed".
- GitLab, Robert Speicher, 2018-07-03 (about.gitlab.com/blog/solving-gitlabs-changelog-conflict-crisis/):
  constant CHANGELOG conflicts; placeholder lines "lessened, but not solved"; chose one YAML file per
  change in `changelogs/unreleased`, compiled at tagging. A real-release test deleted everything when
  no entries existed (fixed). In 2021 (gitlab.com/gitlab-com/gl-infra/delivery/-/issues/1564, epic 351;
  docs.gitlab.com/development/changelog/) entries came from a `Changelog:` commit trailer: the files
  "don't end up improving the developer experience enough to justify the effort".
- CPython blurb (github.com/python/blurb, fetched): rid Misc/NEWS conflicts; entries folded by
  `blurb release`. Go (github.com/golang/go/blob/master/doc/README.md, fetched): one file per change
  in `doc/next/`, concatenated, then removed. Rust (forge.rust-lang.org/release/release-notes.html,
  fetched): a `relnotes` tracking issue is "the source of truth". Kubernetes
  (kubernetes.dev/docs/guide/release-notes/, fetched): the note lives in the PR description.
- MediaWiki RfC T200392 (phabricator.wikimedia.org/T200392, 2018-07-26) and T201514 (2018-08-08):
  `merge=union` rejected because a removed line is silently re-added; a custom driver "ate a line";
  Gerrit's JGit has no merge drivers; approved 2018-10-31, the bot "still needs to be written" in 2019.

Inference: every project with one file per change has an assembly point (release, cycle end) and a
step that deletes the fragments (reno keeps them); those that left files read commits or PRs.

### 2. Agent setups

Claude Code, all fetched 2026-10-16, pages undated:
- code.claude.com/docs/en/memory, "Import additional files". States: imports recurse to "a maximum
  depth of four hops"; relative paths resolve from the importing file; "Import parsing skips Markdown
  code spans and fenced code blocks"; imports "load at launch" and "don't reduce its context cost";
  each `@path` import counts as a separate file toward a combined-limit warning. No glob or
  directory import is documented.
- code.claude.com/docs/en/context-window, "What survives compaction": root CLAUDE.md "Re-injected
  from disk"; whether its imports are re-expanded is not stated.
- code.claude.com/docs/en/sub-agents, "What loads at startup": a non-fork subagent gets the CLAUDE.md
  hierarchy; Explore and Plan skip it; imports are not mentioned. Observed: this subagent's context
  carries `docs/HANDOFF.md` through the `CLAUDE.md` import.
- code.claude.com/docs/en/agent-teams and interactive-mode (experimental): the team task list is
  `~/.claude/tasks/{team-name}/`, outside the repository, with file locking; "Two teammates editing
  the same file leads to overwrites. Break the work so each teammate owns a different set of files."
- code.claude.com/docs/en/hooks: `InstructionsLoaded` is observational; schema not read.

Other frameworks and published practice:
- agents.md (fetched): the closest file wins; no imports. OpenAI Codex
  (learn.chatgpt.com/docs/agent-configuration/agents-md, fetched): files concatenated root-down.
  OpenAI Cookbook "Using PLANS.md" (developers.openai.com/cookbook/articles/codex_exec_plans,
  fetched): one living ExecPlan per complex task.
- GitHub Copilot CLI (docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions,
  fetched): `@relative/path` includes expand in copilot-instructions.md, AGENTS.md and CLAUDE.md.
  Cursor (cursor.com/docs/context/rules, fetched): an `@file` in a rule is "not inlined into the prompt".
- GitHub Spec Kit (github.com/github/spec-kit/blob/main/spec-driven.md, fetched): one directory per
  feature, `specs/NNN-name/`, on its own branch.
- Anthropic, "Effective harnesses for long-running agents", Justin Young, 2025-11-26
  (anthropic.com/engineering/effective-harnesses-for-long-running-agents): one agent at a time; a
  progress file plus a JSON feature list; each session reads git log and the progress file.
- Anthropic, "Building a C compiler with a team of parallel Claudes", Nicholas Carlini, 2026-02-05
  (anthropic.com/engineering/building-c-compiler): a claim is "a text file to current_tasks/";
  "git's synchronization forces the second agent to pick a different one"; "Merge conflicts are
  frequent, but Claude is smart enough to figure that out".
- beads (github.com/steveyegge/beads, fetched): hash IDs "prevent merge collisions"; Dolt storage.
  mcp_agent_mail (github.com/Dicklesworthstone/mcp_agent_mail): a file per message in git; leases.

Inference: Claude Code and Copilot CLI are the only tools found that inline `@` includes; a Codex or
Cursor reader of the same tree would see a map of import lines, not the streams. Tools built for
many simultaneous agents keep live state outside git or as one file per claim.

### 3. Git on files every contributor edits

- gitattributes (git-scm.com/docs/gitattributes): union leaves added lines "in random order";
  custom drivers live in git config, so every clone installs them; GitLab MRs do not run them
  (gitlab-org/gitlab issue 18830) and neither does Gerrit (T200392).
- Experiment, `git merge-file`: two edits to a handoff's Updated line and Now bullet under `--union`
  exit 0 with two Updated lines and two `## Now` headings; the default driver conflicts. Two appends
  merge cleanly. A deletion beside an insert brings the deleted answered question back, exit 0.
- git-rerere (git-scm.com/docs/git-rerere): replays a recorded resolution when the same conflict
  recurs. Inference: handoff conflicts are new text each time, so there is nothing to replay.
- Linux, deleting the file: Torvalds, commit 9c0ece069b32, 2012-10-01 (via `gh api`):
  `Documentation/feature-removal.txt` "has turned out to be a pure write-only file that causes merge
  conflicts". Splitting MAINTAINERS (LWN, 2017-08-10, lwn.net/Articles/730509/): a script made 54
  files; Torvalds "chose not to go with the result". Observed: still one 928,854-byte file.

### 4. Accounts of splitting a status file

- ehky2882/TRAVEL-GUIDED-TOUR PR 845, merged 2026-09-13 (github.com/ehky2882/TRAVEL-GUIDED-TOUR/pull/845),
  written from Claude Code sessions: three PRs in two days blocked on STATUS.md; "67 commits touched
  STATUS.md in seven days", all on the same three line numbers; replaced by `status/entries/`
  (never edited), `status/owner/<slug>.md` (cleared by `git rm`) and `status/builds/<n>.md`; STATUS.md
  kept only deliberately revised sections (144 KB to 36 KB). Observed 2026-10-16: one STATUS.md
  commit since the split; 279 entry files and 17 owner files.
- Not found: a team that split such a file and went back; MediaWiki and MAINTAINERS were not made.

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
