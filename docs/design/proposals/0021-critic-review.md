# 0021, critic review: guardrails bound damage, not who may act

Date: 2026-10-21
Reviewer: the critic (Fable), read-only in the `T121` worktree. Read in order: `CLAUDE.md` and the ten
agreement sections, the proposal, ADRs 0006, 0007, 0011, 0012, 0013, 0016, 0018 with its sibling,
0019, 0020 (on the T120 branch), the hooks (`git-guard.js`, `policy.js`, `policy.json`,
`usage-gate.js`, `statusline.js`, `test-hooks.js` group CB), `settings.json`, `runner.md`,
`pusher.md`, the hook specs section 1, the downstream project's live policy, the research digest.
Not re-argued, as briefed: a history rewrite of existing commits; a PR flow for carry-backs.

Verdict in one line: the six items are the right direction; items 1, 5 (roles) and 6 are ready to
specify; item 2's "created this session" record, item 3's trust in the global identity and item 4's
guard-side lane check are each specified in a way that fails on this machine's own facts and need
one change each before an ADR.

## 1. Claims verified

| # | Finding | Source read | Status |
|---|---|---|---|
| 1 | Verbatim on the page: blocked by default are force push; `git reset --hard`, `checkout -- .`, `restore .`, `clean -fd`, `stash drop/clear` ("presumes would discard uncommitted changes"); `commit --amend` when HEAD was not created in this session, and from v2.1.198 when HEAD is pushed (a message-only reword of the session's own commit is not blocked); "deleting or tearing down a stateful resource Claude didn't create in the session"; "changing where pushes go with `git remote set-url` or `git remote add`, unless you named the new remote"; "irreversibly destroying files that existed before the session". Allowed: "pushing to any branch of the repository you're working in, including the default branch ... The remote's own branch protection still applies." Also: "from v2.1.203 until v2.1.211, only routine default-branch pushes and pushes to the branch you started on or to branches Claude created were allowed by default" (the vendor tried a created-this-session push rule and withdrew it). Claude Code "runs `git status` itself before a command that would discard uncommitted work ... and shows the classifier whether staged, modified, or untracked work is present". | code.claude.com/docs/en/permission-modes | strong, holds |
| 2 | As above. | same | strong, holds |
| 3 | "File writes and edits inside the project directory are allowed without a classifier call" is on the engineering post. "Created this session" is not on that post; it is on the docs page of claim 1 (amend, stateful resources). | anthropic.com/engineering/claude-code-auto-mode | holds, cite the docs page for the phrase |
| 4 | The page exists. `workspace-write` keeps network off; `<writable_root>/.git` is read-only (and the resolved git dir of a pointer file). But: "Windows supports MXC, with `elevated` and `unelevated` as legacy fallbacks"; under WSL2 the Linux sandbox. Option C's "Codex (sandbox instead; not on Windows)" is wrong. | learn.chatgpt.com/docs/agent-approvals-security | holds except the Windows clause |
| 5 | "Copilot cloud agent only has the ability to push to a single branch" (a new `copilot/` branch, or the PR's branch when mentioned there); "subject to any branch protections and required checks"; the requester cannot approve the PR. | docs.github.com, cloud-agent risks-and-mitigations | strong, holds |
| 6 | "multi-agent systems work best today when writes stay single-threaded"; the generator-verifier loop "works best when the coding and review agents do not share any context beforehand"; "Devin Review catches an average of 2 bugs per PR, of which roughly 58% are severe". The `.ai` URL redirects to `.com`. Note: single-threaded writes is an argument for one integration point, not for many agents pushing `main` (objection 7). | cognition.com, both posts | thin as stated, holds |
| 7 | One search for a dedicated committer or pusher subagent found guides on tool-restricted subagents and none recommending a push-only or commit-only agent. An absence; holds as far as one search goes. | web search | thin, holds |
| 8 | "Rubber Stamper applies the Bot-Commit label to conforming CLs, allowing them to bypass code review"; "supports various benign files, clean cherry-picks, and clean reverts"; "never provides OWNERS approval, by design". The same page says clean cherry-picks on release branches get CR+1, that is a review vote, not an exemption. | chromium code_review_owners.md | holds, with that nuance |
| 9 | Available rules: restrict creations, restrict updates, restrict deletions ("selected by default"), block force pushes ("enabled by default"), linear history, each applying to branches and tags; bypass by role, team or app; push rulesets restrict paths (private or internal repositories only). | docs.github.com about-rulesets, available-rules | strong, holds |
| 10 | The cited page as fetched today carries no plan sentence for repository rulesets; it says organization rulesets are for Team and Enterprise. The plans page lists "Protected branches" under "Advanced tools and insights in private repositories" for Pro and Team. A search index still returns the sentence "Rulesets are available in public repositories with GitHub Free and GitHub Free for organizations, and in public and private repositories with GitHub Pro, GitHub Team, and GitHub Enterprise Cloud" attributed to the docs, but it is not on the live page. Substance holds (private needs Pro or above); the citation must move to the plans page or state the gap. Not verified: what GitHub does to an existing ruleset when a Free repository goes private. | docs.github.com about-rulesets, githubs-plans; search | holds in substance, citation wrong |
| 11 | `gc.reflogExpire` "defaults to 90 days"; `gc.reflogExpireUnreachable` "defaults to 30 days", "generally created as a result of using `git commit --amend` or `git rebase`". | git-scm.com/docs/git-gc | strong, holds |
| 12 | `git branch -d/-D`: "If the branch currently has a reflog then the reflog will also be deleted." The unrecoverability of uncommitted work under `reset --hard` and `clean` is git's documented behaviour. | git-scm.com/docs/git-branch | strong, holds |
| 13 | The three primary issues exist in the vendor's issue tracker (numbers 33850, 34327, 17269; all closed; titles match the digest: uncommitted work destroyed in the main worktree, `reset --hard` at startup, reset on task failure). | gh api on the vendor's repository | primary located, holds |
| 14 | Kernel commit d4563201f33a (2023-02-26) "Documentation: simplify and clarify DCO contribution example language": nicknames were always accepted; wording simplified. GCC: "A known identity can be the committer's real, birth or legal name, but can also be an established (online) identity"; not "an anonymous id or false name". | gh api, gcc.gnu.org/dco.html | medium, holds |
| 15 | "in practice users accept 93% of them anyway". | engineering post | strong, holds |
| 16 | Primary found: the vendor's own blog: 1,053 paid testers; humans caught 13.6% (143 of 1,053), auto mode 89% (937); humans blocked about 17% early, about 5% after 50 or more prompts. Setup: one planted dangerous prompt per session in a study environment, nothing dangerous ever ran. That bounds what it shows (one prompt, no real work). | claude.com/blog/auto-mode-default-in-claude-code | now primary; narrower than the digest implies |
| 17 | On the engineering post: full pipeline false positives 0.4% on 10,000 real actions; false negatives 17% on 52 real overeager actions, 5.7% on 1,000 synthetic exfiltrations. The third-party post adds an arXiv stress test at 81% miss, not production traffic. | engineering post; backslash.security | located; the 17% is on 52 cases |
| 18 | "we've found that sandboxing safely reduces permission prompts by 84%" (internal usage). The docs say outright: "The sandbox runs on macOS, Linux, and WSL2. On native Windows, Claude Code runs commands unsandboxed." So the second half is primary, not secondary, and a sandbox is available on this machine through WSL2 (objection 14). | engineering post; code.claude.com/docs/en/sandboxing | first strong; second strong, the proposal under-rates it |
| 19 | The post: "Allow rules are settled before the classifier is consulted, so with that rule in place our force-push was never graded at all." The docs agree and add: on entering auto mode "broad allow rules that grant arbitrary code execution are dropped" (`Agent`, `Monitor`, and whole-tool Bash rules) while "narrow rules like `Bash(npm test)` stay in effect"; whether `Bash(git:*)` counts as broad is not stated. | backslash.security; permission-modes docs | holds; item 6's "never `git:*`" is right |
| 20 | Ona: "engineers cannot self-classify"; "Ona Automation evaluates every PR against objective criteria"; "a person always clicks the merge button"; median lead time down 74%. The touri PR 297 exists, merged 2026-10-18: "tier pull requests by risk so the review bot can merge only low-risk changes". | ona.com; gh api | thin, holds; note that both keep a human or a bot merge, never the author's push |
| 21 | Hooks docs: `agent_id` "unique identifier for the subagent or in-process teammate the hook fires in"; `agent_type` present when the hook fires inside a subagent; settings hooks "also run inside subagents". `session_id` is defined only as "current session identifier"; nothing says whether a subagent carries the parent's. Observed in this review (a subagent): `CLAUDE_CODE_SESSION_ID` equals the parent session's id (the scratchpad directory's). PreToolUse: `permissionDecision` allow/deny/ask/defer; `additionalContext` "added to Claude's context alongside the tool result", allowed with `"allow"`, ignored for `"defer"`; `onFailure: "block"` needs v2.1.295. | code.claude.com/docs/en/hooks; this session | holds on `agent_id`/`agent_type`/`additionalContext`; `session_id` sharing observed, not documented |
| 22 | Re-run today from each repository directory: both repositories public (`visibility: PUBLIC`), `GET /rulesets` returns `[]` for both, `branches/main/protection` returns 404 "Branch not protected" for both. The downstream policy sets `commit.subagents: "any"`, `push.subagents: "any"`, `push.branches: ["task/*"]`, `forceWithLease: ["task/*"]`, `branchDelete: task/* with requireMergedPr`, `commit.protectedBranches: ["main","master"]`, `hooksPath: tools/git-hooks`, per-meter holds 50/75 with `deny: 100`, and one real identity. | `gh repo view`, `gh api` | holds |

Also checked against the tree: `test-hooks.js` CB1 compares `git log main` identities with `privacy.identities` and `privacy.allow` only (objection 5); `statusline.js` prunes only `.json` and `.tmp` files older than a day in `~/.claude/usage-state/`, so a `gate-log.jsonl` there survives (objection 17 is about the surface row, not loss); `post-edit.js` reads the policy and guards no file; the `commit` skill and `git.md` let the main session push only "when the user asked for the push in so many words" (objection 16).

## 2. Refusals, and who lifts each

For every mechanism in B that refuses, denies or fails closed:

| Mechanism | Who lifts | From where | While a session runs | Message says so |
|---|---|---|---|---|
| Ruleset: non-fast-forward or deletion on `main`, update or deletion on `v*` | the owner; any holder of the admin credential, which every agent is | GitHub web UI or `gh api` (the guard's item-6 rule refuses the agent path) | yes | no: git prints GitHub's rule-violation text, naming no lift; the guard does not pre-empt it |
| Guard: force or lease push outside `git.push.forceWithLease` (rule 1, kept) | the owner, by the policy pattern or a terminal push | `policy.json` edit; terminal | yes | yes (rule 1's advice names the pattern) |
| Guard: local rewrite of commits reachable from a remote-tracking ref | the owner via `git.push.forceWithLease` | `policy.json` | yes | not drafted; must name the key |
| Guard: discarding command outside an own worktree with a dirty status | the agent, by committing or stashing; the owner from a terminal | the worktree | yes | not drafted; should list the count of dirty paths |
| Guard: remote deletion of a branch not created, not merged, not contained | the owner | terminal | yes | rule 2's advice says "ask the user"; the new conditions must be named |
| Guard: lane check on a protected-ref push | the agent, by pushing the branch first; the owner from a terminal | the branch | yes | not drafted; must print `lane.js`'s deciding paths |
| Guard: `gh api` writes to rulesets, protection, visibility; `gh repo delete/rename/edit --visibility` | the owner | terminal or web UI | yes | not drafted |
| Guard: tag deletion or move | the owner | terminal, after disabling the tag ruleset | yes | not drafted; the operator entry must carry the disable-then-re-enable step, since the owner's own rewrites (ADR 0018, out of scope) need it |
| Guard: `--no-verify`, `commit -n` (rule 13, kept) | nobody | n/a | n/a | yes |
| Guard: malformed policy | whoever fixes the file | Edit tool | yes | yes |
| Rule 20: scan identity hit under item 3 | the owner: repository identity, or an extra in `privacy.identities` | repository config; `policy.json` | yes | yes today; the scope notice is on a passing push, not a refusal (objection 4) |
| Usage gate in `"block"` mode at HOLD | the owner, by a grant file or `usage.overrideEnv` | `~/.claude/usage-gate-overrides/` | yes | yes (ADR 0006 grammar) |
| Usage gate at `usage.deny` | nobody by design | n/a | n/a | yes |
| Usage gate in `"warn"` mode | nothing is refused | n/a | n/a | the sentence arrives with the tool result, after the launch (objection 8) |
| Fitness check CB1 after T-g removes the identity entry | the owner (`privacy.identities`) or an ADR amendment | `policy.json` | yes | yes, but it would fire on every push (objection 5) |

Two refusals have no documented lift today in the proposal: the server's rule violation (the owner must know to disable the ruleset) and the lane refusal; the rest need their advice sentences written (objection 10).

## 3. The strongest case against B as specified, and the better alternative

B's thesis is right: a shared admin credential makes a role split advisory, the server is the only
non-bypassable bound, and the reflog makes most local rewrites reversible. Where B goes wrong is in
adding two new client mechanisms with their own memory and judgement, a state file keyed on a host
identifier and a lane proxy, to a guard whose whole virtue (ADR 0006) is that it is stateless,
reads only the command line, and needs no file of the repository it runs in.

The "created this session" record exists to let an agent discard uncommitted work in its own
worktree without a status check. Remove it and nothing else in item 2 changes: the status check
already decides every other case, and the only cost is that an agent with a dirty own worktree
runs `git stash` (or names the files, which rule 7 allows today) before `reset --hard`. That is
rarer than the cost of the record: a PreToolUse hook with a side effect, a file shared by every
subagent of a session (they share the id), a sibling-worktree hole, a race on one file, and a
dependence on an undocumented `session_id` semantics. Item 2 without the record is deterministic,
stateless and reaches the same evidence (every incident was uncommitted work outside the agent's
space, and a status check catches each).

The lane proxy in the guard ("every outgoing commit is already on a remote non-protected branch")
cannot pass Hatchery's own landing: a local merge commit or a carry-back landing is on no remote
branch. `lane.js` as a script is good and cheap; its enforcement belongs in the skills (the pusher's
checks, the `wave` merge sequence) until a definition of the proxy that admits merges and landings
has run for a wave. Item 3 should accept the repository-scope identity, which ADR 0018 already sets
from the tracked list, and keep refusing an unlisted global one; that fixes the owner's refusal if
it came from a clone with no repository identity, and keeps ADR 0018's reason intact (28 commits
leaked exactly the global identity nobody set).

So the better alternative is B staged: land items 1, 5 (roles and `localHistory`), 6 and the
repository-scope form of item 3 now, each reversible by a policy value; run one wave; then bring
item 2's own-worktree allowance (if a wave shows agents stashing too often) and item 4's guard
enforcement as their own short proposals with the open facts settled (a captured subagent hook
payload; the merge-commit rule). Options C and E stay rejected for the reasons the proposal gives;
E's defaults match item 2 and are worth stating as a check on the guard's rules, not a replacement.

## 4. Pre-mortem

Ten years on, the guard is the part of the base nobody dares touch: it carries a state directory
under `~`, a lane classifier and an identity resolver, each with harness groups, and twice it
silently changed meaning when the host changed what `session_id`, `cwd` or `agent_type` carried in
a subagent (once every worktree became "foreign", and strict mode returned by accident; once every
path became "created" because a fallback was widened in a fix). The real loss was elsewhere: a
machine set up for a second account pushed that account's global identity for a month before
anyone read the "config scope" notice, because the notice went to an agent's stderr on a passing
push and the audit step read diffs, not identities. The lines that should have caught it are in
Risks ("subagents must share the parent's `session_id`, claim 21", marked "to check" and built on
anyway) and item 3's "shows on the first push" (shows to whom?); and ADR 0018's own Context,
quoted in Constraints and then overruled by a notice.

## 5. Objections

1. **Blocking: the "created this session" record is keyed on an identifier every subagent of a
   session shares.** Evidence: claim 21 (the docs define `session_id` only as "current session
   identifier" and add `agent_id` for subagents; observed here, a subagent's `CLAUDE_CODE_SESSION_ID`
   equals the parent's). In a wave of five to eight implementers, every worktree any of them
   created is "this session's" for all of them and for the coordinator, so a discarding command in
   a sibling's worktree is allowed without the status check. The incident the proposal cites (an
   agent leaving its worktree) is covered; the parallel-wave case, which the downstream runs every
   day, is not. Resolves: drop the record (section 3) or key it on `agent_id` with the main session
   as its own key; never on `session_id` alone.
2. **Blocking: the record is a claim written before the act, in one shared file.** It is written at
   `PreToolUse` of `worktree add`, `branch`, `checkout -b`, `switch -c`; a command that then fails
   (a path in use, a branch that exists on the remote, a denial by a later rule) leaves a record
   saying created. Parallel subagents with one `session_id` read-modify-write one file, the shape
   ADR 0006 refused for grants. Resolves: no record (preferred); else one file per record, written
   by a `PostToolUse` hook from the tool result, or verified at use time against git (`git worktree
   list --porcelain`, the branch reflog's "Created from" entry).
3. **Blocking: the bound's own configuration is unguarded.** `policy.json`, `git-guard.js`, `lane.js`
   and `settings.json` in the session's checkout are editable by any agent with the Edit tool in one
   call, and the guard runs from that working tree, not from a commit (ADR 0020 item 5 covers only
   the pushed branch's copy; `post-edit.js` reads the policy and guards nothing). Today the role
   rules share this weakness; under B the client rule becomes the only bound whenever a repository is
   private, so a rule the refused party can rewrite in place is a notice. Resolves: `permissions.deny`
   lines for `Edit(.claude/hooks/**)`, `Write(.claude/hooks/**)`, `Edit(.claude/settings.json)` in
   item 6 (deny rules apply in every mode, the docs say), so hook changes go through a worktree,
   which is item 4's full lane anyway; the lift is the owner editing `settings.json` from a terminal,
   said in the operator entry.
4. **Serious: item 3 trusts the global identity on the strength of a notice, and the proposal does
   not say what refused the owner's commits on 2026-10-21.** ADR 0018 item 4 sets the repository
   identity from the first tracked entry, so in a repository set up by decant or adopt the
   repository-scope identity equals the list by construction; if the refusal came from another clone
   or a differing global value, the fix is the repository scope, not trust in any scope. The
   `--show-origin` notice goes to the pusher's stderr on a passing push, which under B nobody reads
   before the commit is public. Resolves: accept the identity that resolves from the repository's own
   config (`.git/config` or the worktree's), the noreply form of it, and the web committer; an
   unlisted global or system identity stays a hit with advice to set `user.email` in the repository
   (the proposal's last "what would change" bullet, applied now). State the 2026-10-21 deny line.
5. **Serious: T-g breaks CB1.** `test-hooks.js` CB1 checks every author and committer of `git log
   main` against `privacy.identities` and `privacy.allow` only (read today). "The identity entry
   removable by the owner" makes CB1 FAIL on every Hatchery commit; pusher check 8 reports it as
   design work and the test command fails. Resolves: T-c or T-e amends CB1 to accept the configured
   identity as the scan will, or the entry stays and the proposal says so.
6. **Serious: the guard's lane check refuses Hatchery's own landings.** "Refuses a `full` range
   unless every outgoing commit is already on a remote non-protected branch": a local `merge --no-ff`
   (the coordinator merging a verified branch) and an ADR 0019 carry-back landing (by design a fresh
   commit on `main`, and always a hook or skill change, so `full`) are on no remote branch.
   Resolves: define the proxy as "every non-merge commit is on a remote non-protected branch, and a
   merge commit's non-first parents are", name the carry-back landing (its `Carry-back:` line, the
   ADR 0019 reviewer as its verifier), or keep enforcement in the skills until then (section 3).
7. **Serious: `.claude/skills/**` in the light lane.** Skills are agent instructions that drive git
   (`commit`, `carry-back`, the downstream's `release`), the process half of what the hooks are the
   mechanism half of; the digest's own consensus puts automation high, and T-f itself routes the
   skill edits of this change through the full loop. A skill landing on `main` on the implementer's
   word is the same risk class as a hook. Symlinks (mode 120000) are not addressed. Resolves: light is
   `docs/**/*.md` only, regular files; skills, rules, agents and agreement are full. The "what would
   change" bullet already foresees this narrowing; take it before the first incident, not after two.
8. **Serious: `usage.mode: "warn"` warns after the act and leaves the rule and the hook disagreeing.**
   `additionalContext` arrives "alongside the tool result", so the HOLD sentence reaches the
   coordinator after the launch it describes; the agreement's HOLD row still says "no new work", so
   the hook allows what the rule forbids. The digest's "a warning with a logged override" implies a
   recorded choice; the design logs the launch, not a decision. Resolves: either (a) warn mode plus
   a HOLD row for it that says what the coordinator does with the context (finish launched work, no
   new implementers: the downstream's own wording), so hook and rule agree; or (b) keep block and let
   the coordinator write a session grant with a reason on the owner's standing instruction (ADR 0006
   already defines the file), which is a logged override in one sentence of the agreement and no key.
   Hatchery's own `deny: null` means warn mode leaves it with no hard line at all; say so.
9. **Serious: the four 2026-10-21 failures are not traced to the rules that fired.** The guard a
   session runs is `${CLAUDE_PROJECT_DIR}/.claude/hooks/git-guard.js` with that root's policy; a
   session rooted in the downstream is judged by the downstream's policy, whose `push.branches` is
   `["task/*"]`, which ADR 0019 item 3.6 said to widen to `carry-back/*` and which was not. If that
   is what stopped the carry-back, Hatchery's roles were not the cause and item 5 fixes nothing a
   policy line does not. Resolves: quote each failure's deny prefix in Evidence; keep item 5 on the
   digest's grounds (the split guards nothing a shared credential cannot do), not on the incident's.
10. **Serious: several new refusals have no lift in their message** (section 2): the server's rule
    violation, the lane refusal, the local-rewrite refusal, the dirty-status refusal, the `gh` rule,
    the tag rule. Resolves: each rule's advice names the lift and who holds it, as rule 20's does; the
    guard pre-empts the server for a forced or deleting push to a protected ref with the same advice;
    the rulesets operator entry carries the disable-then-re-enable step for the owner's own rewrites.
11. **Serious: the `gh` rule is presented as the answer to the admin credential and is a tripwire.**
    `gh api` with `-f`, `-F` or `--input` is a POST with no `-X`; `gh api graphql` mutations
    (`deleteRef`, `updateRepository`) do the same work; `gh auth token` hands the credential to
    `curl`, outside any rule; `gh api -X DELETE repos/.../git/refs/heads/<b>` deletes an unprotected
    branch around rule 2. Resolves: call it a tripwire in the ADR, enumerate the shapes (method
    inference, graphql, `gh auth token`), and keep the fine-grained token as the bound with its trigger.
12. **Minor: item 2's verb list is incomplete and two tests name the wrong subject.** "Local rewrite"
    must say whether it covers `branch -f`, `checkout -B`, `switch -C` (rule 18 today), `reset
    --soft`/`--mixed` to an older commit (allowed today; ADR 0018 item 8 relies on `reset --soft` to
    the last pushed sha, which stays fine), `rebase --onto`. `worktree remove --force` and `branch -D`
    run from outside the worktree, so "the working directory resolves inside" must read "the target
    path or branch". Junctions need `fs.realpathSync`, not `path.resolve` (the downstream's rule on
    junctions inside worktrees). Resolves: enumerate in T-b.
13. **Minor: claim 10's citation and the private-visibility question.** The sentence is not on the
    cited page today (section 1). Also unverified: what GitHub does to a ruleset when a Free
    repository goes private (the downstream was private until days ago and may be again). Resolves:
    cite the plans page; the operator entry says to re-check `GET /rulesets` after any visibility
    change, and the `triage` check of T-a lists a repository whose ruleset list is empty.
14. **Minor: claim 18 under-rates its own source, and dismisses the sandbox in one clause.** The docs
    say native Windows is unsandboxed and WSL2 is sandboxed; a read-only main checkout under the
    sandbox is the exact bound the incidents want. It is no fit for the downstream (Android tooling on
    Windows) but is for Hatchery. Resolves: name it as a deferred option with a trigger (a sibling
    whose toolchain runs under WSL2), and mark claim 18's second half strong.
15. **Minor: factual slips in the evidence.** Codex runs on Windows (claim 4); "created this
    session" is on the docs page, not the engineering post (claim 3); claim 16's primary is the
    vendor blog with a one-planted-prompt design; claim 17's 17% is on 52 cases; Chromium's clean
    cherry-picks get a review vote (claim 8). Resolves: correct the strengths in the proposal.
16. **Minor: item 1 silently changes who pushes `main`.** `git.md` and the `commit` skill let the main
    session push only "when the user asked for the push in so many words"; the downstream's ADR 0026
    lets the coordinator push only its own doc edits. "Protected refs are pushed by the main session
    only" reads as unattended pushes of `main` by the coordinator. The digest's agreed item says "work
    branches". Resolves: say in item 1 and T-f whether the owner's word is still needed per push of
    `main`, and record the owner's answer.
17. **Minor: two new files under `~/.claude` without a surface row or a prune rule.** The guard-state
    directory and the gate log sit beside files ADR 0013 lists as promised surfaces because every base
    version on the machine shares `~/.claude`. Resolves: a row each (directory, shape, prune bound) in
    T-e, and a prune rule for guard-state (the log is append-only and survives the status line's prune,
    which touches `.json` and `.tmp` only).
18. **Minor: the audit step puts diffs in front of the coordinator.** ADR 0007: the coordinator reads
    the report, not the diff. Resolves: the audit is a reviewer brief at wave close over the light-lane
    range since the handoff's audited sha, reporting findings as follow-ups; the coordinator reads that.
19. **Minor: a prediction that cannot fail.** "An agent destroys committed work that the reflog or the
    remote cannot restore inside its own worktree" needs `branch -D` of an unpushed, never-checked-out
    branch (claims 11, 12); it will not fire. Resolves: "a `branch -D` under the created-branch
    allowance drops unpushed commits once", or "a wave logs more than N stashes forced by the status
    check" as the trigger to revisit the record.
20. **Minor: boilerplate lines.** Option A's "Exit cost: none now" and "Moving parts: none new" are
    the status quo's and say nothing (A's real exit cost is the sentence after); option E's exit cost
    omits re-training sibling skills written for a mode. Resolves: one concrete line each.

## 6. Recommendation and confidence

Adopt B with items 1, 5 and 6 as written plus objection 3's deny lines; item 3 in its
repository-scope form (objection 4) with CB1 amended (5); item 2 without the state record, the
status check deciding every discarding command outside the target's own cleanliness (1, 2, 12);
item 4 as `lane.js` plus skill-side enforcement, the guard's lane check deferred until the
merge-commit rule is defined and one wave has run (6, 7); `usage.mode` either defined per objection
8(a) or replaced by 8(b). Sequence as the proposal has it, with the operator entry for rulesets
first and the migration naming CB1, the downstream's `carry-back/*` pattern and its wave rule for
light-lane merges. Confidence: high that objections 1, 2, 3, 5 and 6 describe faults that would
show in the first wave (each rests on a file or a command read today); moderate that the staged
form is the right size, since the owner may prefer the own-worktree allowance over stashing, in
which case objection 1's `agent_id` key is the fallback.
