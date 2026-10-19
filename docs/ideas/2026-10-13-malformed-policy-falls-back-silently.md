---
date: 2026-10-13
kind: bug
related: [2026-10-13-usage-gate-stale-deny-not-liftable.md]
---

# A malformed policy makes the usage gate use the base defaults with only a stderr note

`.claude/hooks/usage-gate.js` and `.claude/statusline.js` fall back to the loader defaults when
`policy.json` fails validation (ADR 0006 fail modes), and say so only on stderr, which nobody reads
in a hook. Since 5632bef the default deny is `null`, so a project with a real ceiling loses it on a
typo. The same happens when a policy uses a key or value its loader does not know yet: a project
that copies `"deny": null` before taking the loader that accepts it is silently stricter, the
reverse silently looser. `git-guard.js` fails closed and loud on the same error; the gate does not.

## Open

- Whether the status line should show a flag (for example `POLICY?`) while the policy is malformed.
- Whether `update-from-hatchery` should run `node .claude/hooks/policy.js` as a post-merge check
  that fails the update, so policy and loader always move together.

Question: which shape from Open do you want: a `POLICY?` flag on the status line while `policy.json` fails validation, a failing `policy.js` check in `update-from-hatchery`, or both? Still true in `.claude/hooks/usage-gate.js` (header, lines 14-18): the gate falls back to the defaults with a stderr note. The flag is something you see and the fallback decides whether launches spend money, so it is yours. I recommend both: the flag catches a typo the day it happens, the update check catches loader and policy drifting apart.
Answer: 2026-10-19, both: the POLICY? flag on the status line and the failing policy.js check in update-from-hatchery.
