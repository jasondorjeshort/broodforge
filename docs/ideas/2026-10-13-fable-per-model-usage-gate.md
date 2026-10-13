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

Waits: the statusline docs or payload gain a per-model field.
