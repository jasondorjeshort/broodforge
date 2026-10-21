---
date: 2026-10-21
kind: bug
related: [2026-10-19-guard-gaps-where-git-runs-shell-text.md]
---

# git-guard should apply the git policy of the repository the command runs in

`.claude/hooks/git-guard.js` loads one policy (`load` from `./policy`, which reads the session
project's `.claude/hooks/policy.json` through `policyFile()` in `.claude/hooks/policy.js`) and applies it
to every git command, including one run in a sibling repository.

Seen 2026-10-21: a regenesis session pushed `../droplet-management` (branch `main`) and the push rule
(`git-guard.js` around line 341-429) denied it: "Subagents push only branches matching master", from
regenesis's `branches: ["master"]`. The sibling repo has its own branch naming and its own policy file.
Regenesis worked around it by adding `main` to its own `branches`, which loosens the rule for the session's
own repo too.

Want: for each invocation, use the policy of the repository it runs in. `invocations()` already computes
the resolved dir (`inv.dir`); find that repository's top level and, when it has its own
`.claude/hooks/policy.json`, load that file for the push, commit, delete and cap rules; else fall back to
the session project's policy (or the defaults). Usage thresholds stay the session's.

## Open

- Which repository's policy governs who may commit (the subagent allow-lists): the session's or the target's.
- A target policy that fails to load: deny as today, or fall back to the session's.
- Cache the load per directory so a long pipeline does not reread the file for every invocation.
- A target with no policy file at all: Hatchery defaults (`branches: ["*"]`) would be looser than the
  session's; the session's policy may be the safer fallback.
- The push rule matches the destination branch name, never the remote. With the `main` workaround a pusher
  in a downstream clone may run `git push hatchery HEAD:main` and fast-forward this public repo's `main` to a
  private project's history (a `master`-only rule already let `git push hatchery master` through as a new
  branch). The policy should also decide pushes to a remote other than the repository's own origin; until
  then a clone can set `git remote set-url --push hatchery DISABLED`.
- `pull` counts as a commit (only the runner), yet a downstream push procedure has the pusher run
  `git pull --rebase --autostash` first; on 2026-10-21 the guard denied it (harmless: not behind). Either
  let the pusher pull with `--rebase --autostash`, or have the push procedure hand the rebase back.
