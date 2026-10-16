---
date: 2026-10-16
kind: bug
---
# Pusher check 7 does not read the policy file's cap exemptions

`.claude/agents/pusher.md` check 7 exempts `docs/design/proposals/` and nothing else, while the
post-edit hook reads `edit.capExempt` from `.claude/hooks/policy.json` (ADR 0006). A project with
a large doc it has exempted passes the hook and fails the push. Seen 2026-10-16 in two siblings:
one hand-edited its `pusher.md` to add four exempt folders (so every base update conflicts on the
agent file), the other has three exempt docs over 700 lines that the base's check 7 would refuse
if it ever took the agent file.

Fix: check 7's first command reads the same `edit.capExempt` globs as the hook (a Node one-liner
over `policy.js`, which already resolves the policy, is the shortest honest path), so the two
enforcement points agree by construction (ADR 0002 wants exactly two). The harness gains a case
that runs the pusher's line on a scratch tree with an exempt file over 300 lines.
