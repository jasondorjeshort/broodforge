# 0021. Guardrails bound damage, not who may act

Date: 2026-10-21
Status: promoted to ADR 0021

## Question

The base was built so agents can run unattended safely; it now decides who may act as well as what
may happen, and the second half has become the cost. On 2026-10-21, in the downstream project
Dystopo: a tested 40-line hook change needed the owner to type terminal commands; a backup branch
could not be deleted; the scan refused the owner's own commits; the carry-back of that change to
Hatchery stopped at the commit, because Hatchery's rules let only its `runner` commit and only its
`pusher` push, agent types that do not exist in a session rooted in another repository. The owner
agreed six items that day (research digest, below) and asked for the whole loop. This changes the
hooks' contract for every downstream project (policy defaults, the meaning of `privacy.identities`,
the usage gate's lift), so it is an incompatible change to promised surfaces (ADR 0013) and runs
as a major choice (ADR 0012). In ten years the right answer looks like this: irreversible acts are
refused by the server and, where the server cannot, by one small stateless client rule each;
reversible acts are audited after the fact; any agent of the owner's may do any reversible thing
that discards nothing uncommitted.

## Constraints that matter here

- Agents write all code and run unattended; the owner should not need a terminal for routine work.
- One Windows machine; every agent and session uses the owner's single git credential (Git
  Credential Manager) and `gh` login, which hold admin on the repositories. Native Windows runs
  Claude Code's commands unsandboxed; WSL2 is sandboxed (claim 18), which the downstream's Android
  toolchain does not use.
- Repositories on GitHub; Hatchery and Dystopo are both public as of 2026-10-21 (claim 22). Dystopo
  was private until days ago, and rulesets on a private repository need GitHub Pro or above (claim
  10): the client guard must bound the irreversible acts alone whenever a repository is private.
- Promised surfaces (ADR 0013): the policy file's keys, types, defaults and meanings; the scan CLI
  and the privacy keys (ADR 0018 sibling, section 5); every base version on the machine shares
  `~/.claude`. An incompatible change carries each sibling's migration (ADR 0016's update procedure).
- Prior decisions in play: ADR 0006 (policy keys, fail modes: guard closed, gate open; grant files
  only on the owner's word), ADR 0007 (one independent review per change; the coordinator reads
  reports, not diffs), ADR 0011 (pool advice at HOLD), ADR 0018 (the scan; its Context ruled out
  trusting the global identity), ADR 0019 (the carry-back lands as a fresh Hatchery commit), ADR
  0020 (trailers, the web committer, no placeholder identity at decant and adopt; on the T120
  branch, not yet on `main`).
- The guard is stateless and reads only the command line and git (ADR 0006); that is its virtue.
- Paying decisions are the owner's (`major-decisions.md`): the usage item changes what stops
  spending; the owner agreed it on 2026-10-21, so it is recorded, not re-asked.

## Evidence

Research digest of 2026-10-21 (scratch, not tracked; sources under Claims to verify). Strong: vendor
defaults bound damage, not role: Claude Code auto mode blocks force pushes, discarding resets and
cleans, amending a pushed or foreign commit, deleting what the session did not create, repointing
remotes, and allows pushing to any branch, leaving branch protection to the server (claims 1 to 3);
the vendor tried and withdrew a created-this-session push rule (claim 1); Copilot's cloud agent is
bound by its credential and the server (5); reflog keeps committed work 90 days, uncommitted work
lost to `reset --hard` or `clean` is gone (11, 12); every incident found was uncommitted work outside
the agent's own space (13); approval fatigue (15). Thin: no published guidance on a committer or
pusher agent (7, an absence); AI-change tiering by path (20); DCO and known identities (14).
Inference, not evidence: a commit/push split adds hand-offs without changing what can happen, since
all agents share one credential and the split is enforced only by a hook or a rule text.

The four 2026-10-21 failures were not captured with their deny lines. A session's guard is its root's
(`${CLAUDE_PROJECT_DIR}`), so a Dystopo-rooted session is judged by Dystopo's policy, whose
`push.branches: ["task/*"]` was never widened to `carry-back/*` as ADR 0019 prescribed; the commit
stop in Hatchery came from Hatchery's rule text read by the agent working there. So item 5 rests on
the digest's ground (the split guards nothing a shared credential cannot do), not on the incidents.

## Options

### A. Status quo: roles and per-verb refusal

Policy names who commits (`runner`) and pushes (`pusher`); the guard refuses `reset --hard`,
`clean`, forced deletes, forced checkout and the like everywhere; `privacy.identities` is the whole
list of allowed identities; every change gets one review before push; HOLD denies launches and only
the owner's grant lifts it. Meets "agents run unattended" only inside Hatchery-rooted sessions; a
session rooted elsewhere has no `runner`, so carry-backs stop. Safety: the refusals are real but most
guard committed, reflog-recoverable state; the owner's own commits are refused when the identity
differs by a byte from the tracked pair. Cost: hand-offs to Haiku and Sonnet agents per commit and
push, the owner's terminal for every lift. Who else: nobody found (claim 7).
Exit cost: every sibling skill written for the runner and pusher hand-offs (commit, carry-back,
decant, adopt, the downstream's wave) is rewritten later instead of now; the count grows each month.
Moving parts: two agent types and their briefs on every commit and push; the owner's terminal as the
lift for each refusal.

### B. The six items, staged (recommended)

1. Server-side rulesets on `main` and `refs/tags/v*`: block non-fast-forward updates and deletion
   (tags: update and deletion); no "require a pull request" (the coordinator's doc edits and the
   light lane push `main`), no bypass actors. Any session or agent commits and pushes any
   non-protected branch. Protected refs are pushed by the main session only: without a per-push word
   from the owner for a range `lane.js` classifies `light` with checks green, or `full` with a
   verifier's merge verdict; any other push of a protected ref waits for the owner's word. `runner`
   and `pusher` stay as optional helpers.
2. Guard rules by damage, stateless. Refused for every caller: a non-fast-forward push (a plain or
   `+` force, or `--force-with-lease` outside `git.push.forceWithLease`, never onto a protected
   ref; the guard pre-empts the server here with its own advice); a local rewrite of commits already
   reachable from a remote-tracking ref (`commit --amend`, `rebase` including `--onto`, `reset` of
   any mode to a non-descendant of the upstream, `branch -f`, `checkout -B`, `switch -C`) unless the
   branch matches `git.push.forceWithLease`; remote deletion of a branch with no merged PR whose tip
   no protected remote branch contains (`gh` with `GIT_GUARD_MERGED` as today); a push deleting or
   moving a `v*` tag; repointing a remote (`remote set-url`, `rename`, `remove`, `config
   remote.<n>.url` or `pushurl`); `--no-verify` and `commit -n`. A discarding command (`reset
   --hard`, `clean` without `-n`, `checkout -- <tree>`, `restore <tree>`, forced checkout or switch,
   `worktree remove --force`) is refused only when `git status --porcelain` in its target (the
   working tree it runs in; the named worktree for `worktree remove`; `--ignored` added for `clean
   -x`) shows work it would discard; the advice gives the count of dirty paths and says commit or
   stash first. A forced local branch delete is allowed when the tip is reachable from a
   remote-tracking ref. Unchanged: rules 10, 12, 14, 15, 19, 20; stash drop and clear (the stash is
   one per repository, shared by every worktree); the downstream's rules against killing processes
   by name and stopping Gradle daemons. Paths compare after `fs.realpathSync.native` (junctions, `C:\`
   against `/c/`, case). Every refusal's advice names its lift and who holds it.
3. Identity. The scan accepts, as author, committer or tagger, the name and email that resolve from
   the pushed repository's own config (`git config --local`, or `--worktree` where set), that name
   with a `users.noreply.github.com` address, GitHub's web committer (ADR 0020 item 3), and the
   extras in `privacy.identities`. An identity that resolves only from global or system scope and is
   not listed stays an `identity` hit whose advice says to set the repository's identity to the one
   the owner uses, never a placeholder. No skill proposes a placeholder identity (generalises ADR
   0020 item 4). The content classes (`email`, `home`, `host`, `word`, `session`, `transcript`,
   `secret`) are unchanged. Fitness check CB1 accepts the same set the scan does.
4. Lanes by path. `node .claude/hooks/lane.js <range>` prints `light` or `full` and the paths that
   decided it. Light: every changed path (both sides of a rename) is a regular file (mode 100644)
   ending `.md` under `docs/` or `.claude/skills/`. Full: anything else; `.claude/hooks/**`, the git
   hooks directory, `.claude/settings.json` and workflows are named as always full. Light with checks
   green (test command, caps, scan) goes to `main` on the implementer's report and is audited after
   the fact. Full keeps a reviewed branch and one fresh-context verifier before `main` (a PR where
   the project has a PR flow; in Hatchery, a pushed branch the coordinator merges after the verifier's
   report, and an ADR 0019 landing with its reviewer); no human is needed for either. Enforced by the
   skills that push `main` (`commit`, `carry-back`, a downstream's wave merge sequence), which run
   `lane.js` and quote its line; a guard-side check is deferred (below).
5. Opt-in strictness. `git.commit.subagents` and `git.push.subagents` default `"any"` (a project sets
   `["runner"]`, `["pusher"]` for the split; `policy.examples/roles.json`); `git.localHistory`:
   `"bounded"` (default, item 2) or `"strict"` (today's per-verb refusal). `usage.mode`: `"warn"`
   (default) or `"block"`. The gate denies at HOLD in both modes, so its sentence reaches the
   coordinator before the launch; in `"warn"` the coordinator may lift it itself by writing a session
   grant (ADR 0006's file) whose `reason` says why the launch is worth it, and the gate prints that
   reason on every use; in `"block"` only the owner's word writes a grant, as today. `usage.deny`,
   when set, stays a ceiling nobody lifts. Hatchery sets no deny, so in `"warn"` it has no hard line.
6. Added: narrow allow-list entries in `.claude/settings.json` for routine verbs (`git add`, `git
   commit`, `git fetch`, `git switch`, `git worktree add`, the test command), never `git:*`;
   `permissions.deny` for `Edit` and `Write` on `/.claude/hooks/**` and `/.claude/settings.json`
   (anchored at the project root, so the live checkout's bound is not edited in place and hook
   changes go through a worktree, which the full lane requires anyway; the owner lifts it from a
   terminal); a post-hoc audit: at wave close or handoff, a reviewer reads everything that reached
   `main` outside a reviewed branch since the handoff's audited sha and reports, and the coordinator
   reads the report; a protected-tag rule (item 1's ruleset, item 2's refusal); a tripwire refusing
   `gh api` writes (an explicit method, or `-f`, `-F`, `--input`) to rulesets, branch protection and
   repository settings, `gh api graphql` mutations, `gh auth token`, and `gh repo
   delete|rename|edit --visibility`, named a tripwire, since agents hold an admin credential; the
   bound for that is a fine-grained token without administration permission, deferred.

Deferred, each with its trigger: (a) an own-worktree allowance for discarding commands, keyed on
`agent_id` (the main session its own key), one record per creation verified against `git worktree
list --porcelain` at use; trigger: a wave's reports show agents forced to stash in their own
worktree more than five times, and a captured subagent hook payload settles what `session_id` and
`agent_id` carry. (b) The guard's lane check on a protected-ref push, with a merge rule (every
non-merge commit, and every merge's non-first parents, already on a remote non-protected branch;
an ADR 0019 landing named by its `Carry-back:` line); trigger: one wave run with skill-side
enforcement. (c) The fine-grained token; trigger: Claude Code or Git Credential Manager lets a
session use a credential other than the machine's. (d) A WSL2 sandbox; trigger: a sibling whose
toolchain runs under WSL2.

Meets the constraints: the irreversible acts (history on the remote, deleted refs, uncommitted
work, a published identity, personal content) stay refused by the server, the client, or both; the
reversible ones become audited. Cost: one script, guard rules rewritten with their harness cases, a
ruleset per repository (an owner step), skills and four agreement sections rewritten, one migration
per sibling. Risks: the status check costs one `git status` per discarding command on Windows; the
deny lines bound the Edit tool, not a shell write (a tripwire like the `gh` rule); a light skill
change lands before anyone reads it (the audit is the backstop). Who else: Claude Code auto mode's
defaults draw the same line (claims 1, 2); Chromium and Ona tier review by path (8, 20).
Exit cost: low for items 2, 5 and 6 (policy values restore today's behaviour: `"strict"`, role
lists, `"block"`); item 3 back to a whole-list `privacy.identities` means listing the identity
again in each project and reverting CB1; rulesets are deleted in a minute; the lane script and audit
step are dropped by editing skills, and siblings drop them at their next update.
Moving parts: rulesets per repository; `lane.js`; two new policy keys and two changed defaults;
allow and deny lines in `settings.json`; the audited sha in the handoff. No new file under `~/.claude`.

### C. Server-side only: drop the client guard

Rulesets on `main` and tags; delete `git-guard.js` except rule 20's scan, which stays as the
`pre-push` git hook; no roles, no local refusals; Claude Code's own permission system (auto mode
or prompts) decides the rest. Meets "unattended" best: nothing client-side refuses. Fails the
private-repository constraint: with no ruleset available, nothing stops a force push to `main`.
Fails the incident pattern: uncommitted work in the main checkout is unprotected unless auto mode
is on, and its classifier missed 17% of 52 real overeager actions (claim 17). Loses the guard's
bypass rules (hooks path, aliases, opaque arguments) which keep the scan itself from being skipped.
Cost: least to keep. Who else: the GitHub Copilot agent (server and credential, no client hook);
Codex (a sandbox with `.git` read-only, on Windows too, claim 4).
Exit cost: rebuilding the guard from git history and its harness, plus re-installing it in every
sibling; moderate.
Moving parts: rulesets; one fewer hook; reliance on Claude Code's mode settings per session.

### D. Keep roles, add bypass lists

Keep runner and pusher, per-verb refusal and the whole-list identities; add lifts for the failures
seen: `git.commit.subagents` accepting any agent whose session is rooted outside the repository;
`git.branchDelete.branches` widened by pattern (`backup/*`); grant files (like the usage gate's)
that lift a named rule for a session; the owner's terminal identity added to each project's list.
Meets the four 2026-10-21 failures one by one. Fails the evidence: every lift is another rule and
another prompt (approval fatigue, claim 15), and the next failure needs a new lift; the role line
still guards nothing a shared credential cannot do. Cost: more policy keys and harness cases than B,
and a grant writer. Who else: GitHub rulesets' bypass lists are the server-side form (claim 9), but
there they bound different credentials, not one.
Exit cost: low in code, high in habit: each lift becomes something a sibling relies on.
Moving parts: grant files for the guard, more keys, the roots test.

### E. Delegate the client side to Claude Code auto mode

Make auto mode the required permission mode in every project; delete the guard's local-history
rules and roles; keep rule 20, the bypass rules and rulesets. Its defaults already encode items 1
and 2 (claim 1), maintained by the vendor. Fails determinism: a classifier, not a rule, with a
measured miss rate (claim 17) and defaults that change by release (it added and withdrew a push
rule between two versions, claim 1); allow rules are settled before it (claim 19), so a broad allow
admits a force push; it is not on for every session (the owner's interactive ones). Cost: lowest
maintenance. Who else: Anthropic makes it the default mode (claim 16). Worth keeping as a check:
the guard's rules should not be looser than its defaults.
Exit cost: re-enabling the rules from git history, and rewriting every sibling skill that assumed
the mode's judgement (what it allows unprompted) back into explicit steps.
Moving parts: a required mode setting per project; dependence on vendor release behaviour.

## Recommendation

B, staged so no step loosens a rule before its replacement holds: (1) rulesets on `main` and `v*`
in Hatchery and each public sibling, an owner step (operator entry with the exact `gh api` call, the
re-check after any visibility change, and the disable-then-re-enable step for the owner's own
rewrites); (2) the deny lines of item 6 and the guard rules of item 2 with the `gh` tripwire,
behind `git.localHistory`; (3) the scan's identity source with CB1; (4) `lane.js` and the skills
that enforce it; (5) the policy defaults, `usage.mode`, `policy.examples/roles.json`; (6) the
agreement sections and skills; (7) the migration. B keeps a deterministic, stateless client rule
for each irreversible act, which C drops and E hands to a classifier, and drops the role rule,
which guards nothing a shared credential cannot do (D keeps it and patches around it). Narrower
than the owner's list in three places, each on the critic's evidence: no own-worktree allowance yet
(deferred a), the repository-scope identity rather than any scope, the lane enforced by skills before
the guard (deferred b). First step: the operator entry for the rulesets.

Follow-up tasks (each a brief; the ADR states the rules):
- T-a Rulesets: operator entry per repository, applied by the owner; a `triage` check that lists a
  repository whose `GET /rulesets` is empty.
- T-b Guard: item 2, the tag rule and the `gh` tripwire in `git-guard.js`; `git.localHistory` in
  `policy.js`; harness cases per rule and per Windows path form; `0002-hook-specs.md` section 1.
- T-c Scan identity: `privacy-scan.js` reads the repository-scope identity, `PV` cases, CB1, the
  0018 sibling's contract rows; decant and adopt set the repository identity and ask only for extras.
- T-d Lane: `.claude/hooks/lane.js` (Node built-ins, harness group `LN`, `fitness.json` coverage).
- T-e Keys: `usage.mode` in `policy.js` and the gate's advice; the new role defaults;
  `policy.examples/roles.json`; ADR 0006's living key table amended in place.
- T-f Agreement and skills: `git.md`, `delegation.md`, `one-review.md`, `usage-gating.md`,
  `model-routing.md`; the `commit`, `carry-back`, `handoff` (the audit and its sha), `brief`,
  `decant`, `adopt-hatchery`, `update-from-hatchery` skills; `runner.md`, `pusher.md`;
  `.claude/settings.json` allow and deny lines.
- T-g Migration (ADR 0016 procedure): per sibling, the policy values that keep its behaviour
  (Dystopo: `usage.mode: "block"` for its hard gate, `carry-back/*` in `push.branches`), the
  rulesets step, the deny lines; Hatchery's own `policy.json` drops its role lists (the owner's
  identity entry stays until T-c lands, then is the owner's to remove).

## What would change this recommendation

- A `branch -D` under the reachable-tip allowance drops unpushed commits once: require the tip on
  the branch's own upstream.
- A repository is private on a plan without rulesets and a non-fast-forward push to `main` gets
  through: the client rule was not enough; make the protected-ref rules strict for agents.
- The audit finds a light-lane skill change that had to be reverted: move `.claude/skills/` to full.
- Claude Code gives hooks a parsed push plan or per-agent credentials: re-weigh E and the token.
- A wave shows more than five forced stashes in agents' own worktrees: build deferred item a.

## Claims to verify

1. Auto mode's default blocks and allows, including "the remote's own branch protection still
   applies", and the withdrawn created-this-session push rule:
   https://code.claude.com/docs/en/permission-modes (strong).
2. Pushing to any branch, the default branch included, is allowed by default: same page (strong).
3. Reversible in-project edits skip the classifier: https://www.anthropic.com/engineering/claude-code-auto-mode;
   "created in the session" is on the page of claim 1 (strong).
4. Codex: workspace-write sandbox, `.git` read-only, Windows supported:
   https://learn.chatgpt.com/docs/agent-approvals-security (strong).
5. Copilot cloud agent pushes to one `copilot/...` branch; rulesets apply:
   https://docs.github.com/en/copilot/concepts/agents/cloud-agent/risks-and-mitigations (strong).
6. "Writes stay single-threaded"; fresh-context reviewer: https://cognition.com/blog/multi-agents-working,
   https://cognition.ai/blog/dont-build-multi-agents (thin: vendor blog, figures unaudited).
7. No published guidance recommends a dedicated committer or pusher agent (thin: an absence).
8. Chromium's Rubber Stamper skips human review for benign changes, never grants OWNERS; clean
   release cherry-picks get a review vote:
   https://chromium.googlesource.com/chromium/src/+/HEAD/docs/code_review_owners.md (strong).
9. Rulesets: block force push, restrict deletions and updates, bypass lists, path rules (private only):
   https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets (strong).
10. Protected branches in private repositories need Pro or Team: GitHub's plans page,
    https://docs.github.com/en/get-started/learning-about-github/githubs-plans (substance holds; what
    happens to a ruleset when a repository goes private is unverified).
11. Reflog expiry 90 days reachable, 30 unreachable: https://git-scm.com/docs/git-gc (strong).
12. Uncommitted changes discarded by `reset --hard`, `checkout -- .`, `clean` are unrecoverable;
    `branch -D` drops the branch's reflog: https://git-scm.com/docs/git-branch (strong).
13. Incidents: the vendor's issue tracker, issues 33850, 34327, 17269 (primary located; strong).
14. The kernel's DCO commit d4563201f33a simplified the identity wording; GCC accepts an established
    online identity: https://gcc.gnu.org/dco.html (medium).
15. Users accept 93% of permission prompts: https://www.anthropic.com/engineering/claude-code-auto-mode (strong).
16. Humans caught 13.6%, the classifier 89%, one planted prompt per session among 1,053 testers:
    https://claude.com/blog/auto-mode-default-in-claude-code (primary; narrow design).
17. Classifier 0.4% false positives on 10,000 actions, 17% false negatives on 52 overeager actions:
    the engineering post of claim 15 (primary; small sample).
18. Sandboxing cut prompts 84%; native Windows runs unsandboxed, WSL2 sandboxed:
    https://www.anthropic.com/engineering/claude-code-sandboxing, https://code.claude.com/docs/en/sandboxing (strong).
19. Allow rules are settled before the classifier:
    https://www.backslash.security/blog/claude-code-auto-mode-catches-a-lot-but-not-everything and the page of claim 1 (holds).
20. Ona auto-approves low-risk PRs by an objective classifier, a person merges:
    https://ona.com/stories/auto-approving-low-risk-prs; https://github.com/touri-ai/touri/pull/297 (thin).
21. A hook inside a subagent carries `agent_id` and `agent_type`; `PreToolUse` may allow with
    `additionalContext`, delivered alongside the tool result: https://code.claude.com/docs/en/hooks
    (holds); a subagent shares the parent's `session_id` (observed, not documented).
22. Verified 2026-10-21 by the architect and the critic: both repositories public, `GET /rulesets`
    returns `[]`, `main` unprotected; Dystopo's policy sets `"any"` roles, `push.branches` and
    `forceWithLease` `["task/*"]`.

## Critic's objections and responses

Review: `0021-critic-review.md` (3 blocking, 8 serious, 9 minor). All resolved; none left open.

1. Blocking, the record keyed on a shared `session_id`. Accepted with change: no record; the status
   check decides every discarding command; the allowance is deferred item a, keyed on `agent_id`.
2. Blocking, the record written before the act in one shared file. Accepted: gone with 1; deferred
   item a verifies against git at use and writes one record per creation.
3. Blocking, the bound's own configuration is editable in place. Accepted: item 6's deny lines,
   anchored at the project root; a shell write is the residual, stated under Risks.
4. Serious, item 3 trusts the global identity on a notice. Accepted with change: repository scope
   only; a global-only identity stays a hit with set-the-repository advice; Evidence says the
   2026-10-21 deny line was not captured.
5. Serious, CB1 breaks. Accepted: CB1 accepts the scan's set (T-c); the identity entry stays until then.
6. Serious, the guard's lane check refuses Hatchery's landings. Accepted: enforcement in skills;
   the guard check is deferred item b with the merge and landing rule written.
7. Serious, skills in the light lane. Accepted in part: light keeps `.claude/skills/**/*.md` because
   the owner named it, now regular files only (no symlinks); agents, rules, agreement and settings are
   full; a reverted light skill change moves skills to full (What would change).
8. Serious, warn mode warns after the act. Accepted with change, the critic's 8(b): the gate denies at
   HOLD in both modes; `"warn"` lets the coordinator write the session grant with a reason (the logged
   override); no gate log; Hatchery's missing deny line stated.
9. Serious, failures not traced. Accepted: Evidence rewritten; item 5 rests on the digest; T-g widens
   the downstream's `push.branches` to `carry-back/*`.
10. Serious, refusals without a lift in the message. Accepted: every new rule's advice names its lift
    and holder; the guard pre-empts the server's rule violation; the operator entry carries the
    ruleset disable-then-re-enable step.
11. Serious, the `gh` rule is a tripwire. Accepted: named so, shapes enumerated (method inference,
    graphql, `gh auth token`); the token is the bound, deferred item c.
12. Minor, verb list and target subject. Accepted: verbs enumerated in item 2; target, not working
    directory; `fs.realpathSync.native`.
13. Minor, claim 10's citation. Accepted: plans page cited; re-check after a visibility change in T-a.
14. Minor, the sandbox dismissed. Accepted: claim 18 strong; deferred item d.
15. Minor, factual slips. Accepted: claims 3, 4, 8, 16, 17 corrected; option C's Codex clause fixed.
16. Minor, who pushes `main`. Accepted: item 1 says when the main session pushes `main` unattended
    and when the owner's word is still needed; the owner may narrow it (ADR Consequences).
17. Minor, new `~/.claude` files. Accepted: both dropped (no record, no gate log).
18. Minor, the audit puts diffs before the coordinator. Accepted: a reviewer reads, the coordinator
    reads its report.
19. Minor, a prediction that cannot fail. Accepted: replaced by the `branch -D` and stash triggers.
20. Minor, boilerplate lines. Accepted: A's and E's exit cost and moving parts rewritten.
