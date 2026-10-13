---
date: 2026-10-13
kind: bug
related: [2026-10-13-usage-gate-stale-deny-not-liftable.md]
---

# A malformed policy makes the usage gate use the base defaults with only a stderr note

`.claude/hooks/usage-gate.js` and `.claude/statusline.js` fall back to the loader defaults when
`policy.json` fails validation (ADR 0006 fail modes), and say so only on stderr, which nobody reads
in a hook. Since 59fd8ce the default deny is `null`, so a project with a real ceiling loses it on a
typo. The same happens when a policy uses a key or value its loader does not know yet: a project
that copies `"deny": null` before taking the loader that accepts it is silently stricter, the
reverse silently looser. `git-guard.js` fails closed and loud on the same error; the gate does not.

## Open

- Whether the status line should show a flag (for example `POLICY?`) while the policy is malformed.
- Whether `update-from-hatchery` should run `node .claude/hooks/policy.js` as a post-merge check
  that fails the update, so policy and loader always move together.
