---
date: 2026-10-13
kind: feature
---

# Gate Fable launches on a per-model usage meter

Apply the same 50/80 usage rule to `architect` and `critic` launches, which run on Fable, as the
main gate applies to everything else (`.claude/hooks/usage-gate.js`). Why not now: Claude Code
2.1.296 passes the status line (`.claude/statusline.js`) only `rate_limits.five_hour` and
`.seven_day`, with no per-model meter (checked against the docs and a live payload on 2026-10-12).
Until then, run `/usage` by hand before a Fable launch.

2026-10-15: another source, when a switcher such as claude-swap is present: `cswap list --json` gives each account's
per-model weekly windows (`usage.scoped[]`, Fable among them) besides 5h and 7d. Related:
`2026-10-15-usage-gate-hold-vs-account-rotation.md`.

2026-10-16: proposal 0011 considered judging the window from `cswap list --json` at launch time and
withdrew it (critic objection 1): a grant names only `5h` and `7d` (`policy.js` METERS), so the
refusal had no lift, and the status line cannot show a meter it must not fetch. When it ships,
`tool_input.model` is read before the agent file's `model:` (the hooks docs' resolution order).
Besides the payload field below, the meter needs a grant that can name a model window (an ADR
0006 amendment); the owner may widen the trigger below to say so.

Waits: the statusline docs or payload gain a per-model field.
