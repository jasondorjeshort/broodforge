# 0019. Critic review: the carry-back channel

Date: 2026-10-18
Reviewer: critic (Opus by override; process proposal)
Proposal: `0019-carry-back-channel.md` (draft)

## Summary

The core of the recommendation is right: a carried-back change lands in Hatchery as a fresh
Hatchery commit, built from the diff and pushed through Hatchery's own guard and scan, never as a
merge of the sibling's commits. Everything around that core is weaker than the proposal says. PR 1
did not reach Hatchery the way the proposal describes, and the guard rule built on that account
would not have stopped it. Nothing fixes what a carry-back branch is based on, so a sibling's whole
history can enter Hatchery's repository for good. The record lives only on GitHub, in a project
that deleted and recreated its GitHub repositories yesterday. Option C is costed against
a strawman, and the alternative that keeps C's safety and B's queue is never weighed. There are 12
objections: 2 blocking, 5 serious and 5 minor.

## Claims checked

| # | Claim | Result | Evidence |
|---|---|---|---|
| 1 | `refs/pull/<n>/head` stays fetchable after the PR is closed and its branch deleted | Holds in substance | GitHub, "Checking out pull requests locally": "After a pull request is opened, GitHub stores all of the changes remotely"; "The remote `refs/pull/` namespace is read-only"; a deleted fork's PR can still be fetched and merged. GitHub, "Removing sensitive data from a repository": a force-push "will fail to push any refs starting with `refs/pull/`, since GitHub marks those as read-only"; old commits stay reachable "through any pull requests that reference them"; only GitHub Support can "dereference or delete any affected PRs", and only for sensitive data whose risk can't be handled by rotating credentials. Observed: `git ls-remote origin` shows `refs/pull/1/head` and `refs/pull/1/merge`. I did not delete a branch to test this, because that would touch the PR. |
| 2 | GitHub marks a PR merged only when its head commit becomes reachable from the base branch; a local squash never does | Holds | GitHub changelog 2023-09-26: a pull request whose "commits are merged into the base branch by another pull request (or directly)" is "still marked as merged". A local squash makes a new commit, so the head is never reachable. The guard's `mergedOnGitHub` asks `gh pr list --head <b> --state merged` (`git-guard.js` line 287), so `requireMergedPr` would deny every delete after a local landing. |
| 3 | DEP-3 carries `Forwarded:` and `Origin:` for exactly this record | Half right | DEP-3: `Forwarded` holds the URL that proves the patch went upstream. `Origin` records where the patch came from, which for a project's own patch is the project itself. The field that records the landing is `Applied-Upstream`: "the URL or commit identifier of the upstream commit". The row's two stages (PR number, then landing sha) map to `Forwarded`, then `Applied-Upstream`. See objection 10. |
| 4 | `git apply --3way` of `git diff <merge-base> <head>` onto a moved `main` conflicts per file, not whole | Partly false | Experiment in `<scratchpad>/c4` (git 2.56.0.windows.2, fixtures built with `commit-tree`). Same-line edit: `UU f1`, the other files applied, exit 1, per file. Both sides add the same path: `U f4`, per file. **`main` deleted a file the PR edits: `does not exist in index`, nothing applied, whole failure.** **A binary in the PR, without `--binary`: `cannot apply binary patch ... without full index line`, whole failure.** With `--binary`: applied. ADR 0016 records the same thing in its Deferred list ("`--3way` fails whole on most updates") and in the per-path table of its update procedure. See objection 6. |
| 5 | The sibling's push passed its guard because rule 3 matches the branch and never reads the remote | Holds, but describes a different push | `pushTargets` (`git-guard.js` lines 246 to 267) skips `words[0]`, the remote; rule 3 (line 417) checks only the targets. But the push was not to a `hatchery` remote in the sibling's tree. See objection 1. |
| extra | "Hatchery's scan refuses PR 1's commit scanned alone (two `identity` hits)"; "the PR body passes `--text`" | Holds | Rerun: `privacy-scan.js --no-remote 1eb7a71 ^749377e` gives `HIT identity ... author` and `HIT identity ... committer`. `--text` on the body gives `clean, 17 lines`. |

## Objections

1. **Blocking. PR 1 did not travel the way the proposal says, and the rule built on that account would not have stopped it.**
   **Evidence.**
   - The sibling's task text (its `docs/TASKS.md`, T117 Goal) says the change went as "a branch and a draft pull request ... from a fresh clone under the worktrees folder (`git -C`, ...)". The task also says "this project's hooks judge that clone too, so its branch is named `task/T117-<slug>`".
   - The clone (`../<sibling>-wt/<clone>`) has one remote, `origin`, which is Hatchery's URL. It has a local `user.name` and `user.email` set to the sibling's placeholder, and no `core.hooksPath`, so Hatchery's own pre-push hook never ran there.
   - `gh pr view 1 --json isCrossRepository` returns `false`: the PR is a same-repository branch, not a fork.

   **Why the proposed rule fails.** It reads "a push naming the `hatchery` remote whose targets do not all match `carry-back/*` is denied". It does not fire on `git -C <clone> push origin task/...`, which is the exact push that made PR 1. It also misses each of these:
   - a push to a URL;
   - a renamed remote;
   - a bare `git push`, whose remote comes from `branch.<b>.pushRemote`, `remote.pushDefault` or `branch.<b>.remote` (`pushTargets` takes the current branch when there is no word).

   **"`main` is pushed from here only" is not enforced.** In a fresh clone, a sibling whose policy keeps the base default `git.push.branches: ["*"]` (which the proposal keeps) can push the clone's `main` straight into Hatchery's `main`. Rule 3 never checks branches for the main session (line 415). Nothing then runs Hatchery's scan, review or harness.

   **Resolve.**
   - Correct the Constraints and B's text to describe the clone form.
   - Pick one sending form and make the skill use only that form. The natural one is a worktree in the sibling's own repository, cut from a `hatchery/main` sha and pushed to the `hatchery` remote.
   - Key the rule on the resolved push URL, normalised, compared with the `hatchery` remote's URL or a policy key. The rule must also resolve the implicit remote. It must deny any push whose target URL is Hatchery's when the session's project root is not Hatchery, unless the targets all match `carry-back/*`. That covers the `-C <clone>` form.
   - Name harness cases for the clone, URL, implicit-upstream and renamed-remote forms.
   - Optionally add a server-side backstop: a GitHub ruleset with "Restrict creations" on every branch except `main` and `carry-back/**/*`. It works whatever the client does. Rulesets need GitHub Pro or above on a private repository (GitHub docs, "Available rules for rulesets"); I could not check the owner's plan. A ruleset cannot protect `main` here, because the sibling and Hatchery push with the same account.

2. **Blocking. Nothing fixes what a carry-back branch is based on, and the failure is permanent.**
   **Evidence.**
   - A decanted sibling's history starts from Hatchery's history. A branch cut from the sibling's `main` (the obvious thing for an agent to do in the sibling's own tree, which is the form objection 1 recommends) contains every sibling commit since decant.
   - The sibling's rule 20 scans that range, but under the sibling's policy. Its identities and allow list pass the sibling's own placeholder and anything else the sibling lists. So the push succeeds, and the whole sibling history sits under `refs/pull/<n>/head` for good (claim 1, read-only; only Support can remove it, and only for sensitive data).
   - PR 1 was safe only because the clone happened to start at `749377e`. Its one commit's parent is Hatchery's `main` (`git log --format=%P`).
   - The proposal's own constraint says "published once, published for good". No line of the recommendation acts on it.

   **Resolve.**
   - The skill builds the branch from a named `hatchery/main` sha, as a worktree with `-b carry-back/<slug>`.
   - The guard rule from objection 1 also denies a carry-back push when any commit it sends is reachable from the sibling's own `--remotes=origin`, or when the range is more than a small count. Every commit in `rev-list <ref> --not --remotes=hatchery` must have parents only inside the range or in `hatchery/*`.
   - The deny message names the rebuild command.

3. **Serious. The record lives only on GitHub; the exit cost is boilerplate and wrong.**
   **Evidence.**
   - "Hatchery keeps no table: the closed PR list and the landing messages are the record". The departure row says `hatchery PR <n>`, and the landing body says `Landed from PR <n>`.
   - On 2026-10-17 this project rewrote three histories. For the base, "The GitHub repository was deleted and recreated, and the run repeated" (`docs/findings/2026-10-17-history-rewrite-of-three-repos.md`, item 2). The old repositories survive only as `<repo>-backup`.
   - A recreated repository has no PRs. Its numbering restarts at 1, so after a recreation every `PR <n>` cites a different PR or nothing.
   - B also forces that outcome. `refs/pull/*` cannot be rewritten in place (claim 1), so any future rewrite of Hatchery that has to remove something a PR carried must recreate the repository or go through Support.
   - B's exit cost reads "close open PRs, delete `carry-back/*`". It leaves out that the commits stay in `refs/pull/*` for the life of the repository. That is exactly the exit cost ADR 0012 asks for, and exactly what the owner would meet first.

   **Resolve.**
   - The durable reference is git-native. The departure row ends with the landing sha. The landing commit body carries the substance of the PR's account (scanned like any message), not only the number.
   - B's exit cost states that `refs/pull/*` is permanent and that removal goes through Support.
   - "What would change this recommendation" gains a line: a planned rewrite or recreation of Hatchery.

4. **Serious. The strongest alternative is never weighed: C with no remote, plus an issue for the account.**
   C's three listed costs disappear with a small change:
   - **No remote per sibling.** `git fetch <sibling-url> carry-back/<slug>` writes `FETCH_HEAD` and creates neither a remote nor a tracking ref.
   - **Queue, number and account.** The sibling runs `gh issue create --body-file <f>` on Hatchery, after `--text` on the file (the same step B needs for the PR body). Hatchery already has this idea, `docs/ideas/2026-10-13-github-issues-for-intake.md`, which the proposal neither cites nor decides.

   What the result keeps:
   - no foreign object in Hatchery's repository;
   - no guard rule and no pusher exception, so objections 1 and 2 never arise;
   - Hatchery as its only writer;
   - an issue its owner can delete (GitHub docs, "Deleting an issue": a personal repository's owner can delete issues), where a PR needs Support;
   - one channel for every sibling, including one the owner does not control, for which the proposal already falls back to C.

   It also fails safer. With no tracking ref, a mistaken local `git merge FETCH_HEAD` puts the sibling's commit in the range Hatchery's scan reads, and rule 20 refuses it on `identity`. With B, `origin/carry-back/<slug>` is a tracking ref, so a mistaken merge is skipped by `--not --remotes=origin` (the proposal observed exactly this: "clean, 0 commits").

   What E costs:
   - the sibling's repository must be readable by Hatchery, which is true for every sibling today (one account);
   - the GitHub diff view is not attached to the issue, but no agent here reads diffs on the web;
   - there is no `gh pr` review thread; review comments go on the issue.

   **Resolve.** Add this option (call it E) at the same depth as the others and decide between B-tightened and E on the record.

5. **Serious. The queue has the same gap in B as in C, and PR 1 duplicates two open Hatchery ideas.**
   **Evidence.**
   - C is charged with "nothing prompts the coordinator to run" `ls-remote`. But nothing prompts `gh pr list` either: no skill in `.claude/skills/` reads PRs.
   - A PR opened by the same account notifies nobody. GitHub lists "Your own updates, such as when you open ... a pull request" only as an email option the user turns on ("Configuring notifications").
   - The proposal says A has "nothing announces it". The idea route did announce here: Hatchery already holds `docs/ideas/2026-10-16-anchor-check-rejects-elided-paths.md` and `docs/ideas/2026-10-15-statusline-command-cwd-relative.md`, two of PR 1's three changes. `docs/ideas/2026-10-16-downstream-improvements-to-carry-back.md` shows the coordinator reading a sibling checkout. The proposal does not say that landing PR 1 must resolve and delete those ideas.

   **Resolve.**
   - Name the prompt: a step in `triage` (and the handoff's Next) that lists open carry-backs, whichever channel is chosen.
   - Make the landing procedure search `docs/ideas/` for the same change and delete what the landing completes.

6. **Serious. The landing command is weaker than the update command it borrows from, and claim 4 is partly false.**
   **Evidence.**
   - Claim 4's experiment: a PR that edits a file `main` has since deleted fails whole, and so does any binary without `--binary`.
   - The proposal writes `git diff <merge-base> <head> | git apply --3way`. ADR 0016's command carries `-M --binary` and must run in Bash, never PowerShell, where the pipe re-encodes the patch (`0016-update-procedure.md`, step 4).

   **Resolve.** Either option works:
   - Use `git diff -M --binary <mb> <head> | git apply --3way` in Bash, with the per-path fallback and its disposition table.
   - Or have the runner, who may make commits under rule 4, use `git merge --squash --no-commit FETCH_HEAD` (or the tracking ref). Git's merge machinery reports a modify/delete as a per-file conflict and detects renames; this is standard git behaviour, not run here. The runner then commits under Hatchery's identity, so the author is not kept, which is the point.

   Either way, state the fail-whole case and its lift.

7. **Serious. Mechanisms that refuse have no stated lift or wording.**
   - **The new guard rule** (every caller, base-owned, so it ships to every sibling): no deny text, no advice, no policy key. Who lifts a false deny in a sibling while a session runs? Under the guard's rules only the owner, from a terminal, and the message must say so (rule 2's advice, "ask the user", is the pattern). Is the remote name or pattern policy data or hard-coded? A project with a different arrangement would have to depart from the hook, which ADR 0016 item 6 says is a row, never an exclude.
   - **The `--text` step at `gh pr create`**: a hit stops the skill, but the proposal does not say whose lift applies. The answer is the sibling's owner-only lists, from the sibling's main session after asking the user (ADR 0018 item 8), and the skill should say it.
   - **The branch delete**: `git.branchDelete.branches` gains `carry-back/*`, but PR 1's branch is `task/T117-...`. Rule 2 will deny the pusher's delete after "then PR 1 lands". The message, "Deleting or mirroring remote refs is for the user; ask the user to do it", does name the lift, but the proposal's first step does not foresee it.
   - **The pusher check 3 exception**: it is moot for a sibling that keeps no pusher. The sibling that sent PR 1 has none (the proposal says so). The guard rule is the only check there, so objection 1 is the whole of the protection.

   **Resolve.** For each item, give the deny text with its lift. Say whether PR 1 is grandfathered on its branch name. Say whether the rule reads a policy key.

8. **Minor. The one review reads the PR diff, not what lands.**
   The reviewer reads `main...origin/carry-back/<slug>`. Then the coder resolves conflicts against a moved `main` and folds in fixes, and nobody reads the result before the commit. The tracking ref the reviewer needs is also the thing that makes a mistaken merge invisible to the scan (objection 4).
   **Resolve.** The reviewer reads the staged landing (`git diff --cached` after the apply) instead of the PR range. Fixes go back to the writer as usual.

9. **Minor. The ten-year line cannot fail, and no prediction is offered.**
   "A sibling sends a base change by one command, Hatchery lands it by one procedure, the row disappears" is a slogan. Nothing in it can be checked on a date.
   **Resolve.** The ADR carries at least two predictions in ADR 0012's form, for example:
   - "Prediction, check by 2027-04-18: no commit reachable from any `refs/heads/*` of Hatchery's repository has an author or committer outside `privacy.identities`".
   - "Prediction, check by 2027-04-18: every carry-back opened before 2027-01-18 is landed or closed, and its departure row is gone within two updates of the landing".

10. **Minor. The DEP-3 citation names the wrong field.**
    See claim 3. **Resolve.** Cite `Forwarded` for the PR and `Applied-Upstream` for the landing sha.

11. **Minor. B is called "fork-and-PR", but PR 1 is a same-repository branch.**
    `isCrossRepository: false`. The owner's one account can push to both repositories, which is what makes the push possible at all. The first line of "What would change" is the fork case and needs B's fork form spelled out (the contributor pushes to their own fork, so no Hatchery-side guard rule applies).
    **Resolve.** Rename the option and separate the same-account form from the fork form.

12. **Minor. The coordinator executes `gh` writes.**
    "The coordinator closes the PR with one comment". `delegation.md` gives execution to subagents "even when it is one command". `gh` is also outside the guard, so nothing checks who runs it.
    **Resolve.** The pusher, which already deletes the branch, closes the PR after `--text` on the comment file.

## The rejected alternatives, steelmanned

**`gh pr merge --squash` with a pre-merge scan.**
- The steps:
  1. The reviewer reads the PR.
  2. The harness runs on GitHub's test merge, `refs/pull/<n>/merge`, which exists and can be fetched (observed).
  3. `privacy-scan.js --text` runs on the subject and body file.
  4. One command lands the PR: `gh pr merge <n> --squash --subject ... --body-file ... --delete-branch`.
- What it gives:
  - The PR shows as merged, so `requireMergedPr: true` works and the branch goes with it.
  - No local apply, so objection 6 is gone.
  - The squash drops the sibling's author (GitHub's 2019 changelog: the other authors become `Co-authored-by` lines). The `.invalid` placeholder in such a line passes `email`.
  - The content scan is close to Hatchery's anyway. On this machine the never list, the user allow file and the `word` class are user-scoped and shared, so the sibling's scan at its push differs from Hatchery's only in the tracked `privacy.allow` and `privacy.identities`.
- Why it still loses:
  - The landing commit never passes rule 20 or the pusher.
  - Its committer is GitHub's web-flow identity, which no `privacy.identities` lists. ADR 0018's prediction ("no commit on any remote of this tree made after this ADR carries a hit of any class") would be false on the first merge.
  - `gh` is outside the guard, so any agent could run the merge.
  - Fixes cannot be folded in; they go back as sibling commits.

  These are real costs, but they are smaller than the proposal's "a squash on GitHub bypasses every check" suggests. Listing the web-flow identity (an owner decision) answers the second one.

**Idea file only (A).**
- Every sibling is a local checkout. The coordinator can read the built diff read-only (`git -C ../<sibling> show <sha>`), so "a built, tested diff is reduced to prose" overstates the loss.
- Two of PR 1's three changes were already queued here by the idea route.
- Nothing foreign ever enters Hatchery, and no rule, exception or skill is needed.
- Its true cost is latency: those two ideas had sat for days, and the sibling's harness group would be rebuilt rather than reused. That cost is real; it is the argument for E rather than for B.

## Pre-mortem

It is 2036. Hatchery's repository holds read-only `refs/pull/*` from dozens of siblings.

In 2028 one sibling's agent cut `carry-back/<slug>` from the sibling's own `main`, the natural move in a worktree. The sibling's scan passed it under the sibling's own identities and allow list, and the sibling's full history went up. That history held a second account's paths in old blobs, the class this project already rewrote three repositories to remove. In-place rewriting was impossible (`refs/pull/*` is read-only), and Support only acts on credentials, so the owner recreated the repository. That dropped every PR, restarted the numbers, and left every `hatchery PR <n>` in sibling departure tables and every `Landed from PR <n>` pointing at nothing or at the wrong PR.

Three lines should have caught it:
- B's exit cost, "close open PRs, delete `carry-back/*`";
- "Hatchery keeps no table: the closed PR list and the landing messages are the record";
- the guard rule keyed on a remote name with no check of what the branch is based on.

## Recommendation

Keep the proposal's landing: a fresh Hatchery commit, built from the diff and scanned under Hatchery's policy, with fixes folded in and the sibling's author never kept. Replace the transport with E:
- The sibling pushes `carry-back/<slug>` to its own `origin`, cut from a named `hatchery/main` sha.
- It files the account as a Hatchery issue after `--text`.
- Hatchery fetches by URL into `FETCH_HEAD`, and the runner lands with `git merge --squash` (or the `-M --binary` apply), which the reviewer reads staged.
- The departure row ends with the landing sha.

E needs no guard rule and no pusher exception, puts nothing foreign in Hatchery's repository, works unchanged for a sibling the owner does not control, and fails into the scan rather than around it.

If the owner prefers PRs because the owner called PR 1 "by our design", B is acceptable once objections 1, 2 and 3 are fixed: a URL-keyed rule covering the clone form, a base check, and a git-native record. B then costs more moving parts than E for one GitHub view.

Confidence: high (about 85%) that B as written must not be built, because objections 1 and 2 are each enough on their own. Moderate (about 60%) that E beats B-tightened; the deciding question is whether the owner values the PR as the place to discuss a change, which this review cannot settle.

## Sources

- [GitHub Docs: Checking out pull requests locally](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/checking-out-pull-requests-locally)
- [GitHub Docs: Removing sensitive data from a repository](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- [GitHub changelog 2023-09-26: PRs merged indirectly](https://github.blog/changelog/2023-09-26-more-details-provided-when-a-pull-request-is-merged-indirectly-or-is-still-processing-updates/)
- [GitHub changelog 2019-12-19: attribution when squashing](https://github.blog/changelog/2019-12-19-improved-attribution-when-squashing-commits/)
- [GitHub Docs: About pull request merges](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges)
- [GitHub Docs: Available rules for rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets)
- [GitHub Docs: Configuring notifications](https://docs.github.com/en/account-and-profile/managing-subscriptions-and-notifications-on-github/setting-up-notifications/configuring-notifications)
- [GitHub Docs: Deleting an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/administering-issues/deleting-an-issue)
- [DEP-3: Patch Tagging Guidelines](https://dep-team.pages.debian.net/deps/dep3/)
