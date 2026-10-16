Goal: <one sentence: the outcome, not the activity>

Read first, in order: `CLAUDE.md`, <index paths, then any specific files>.

Context only this task knows: <facts, decisions, user wording; or "none">

Already ruled out: <what was tried and why it failed, with finding paths; or "nothing">

Scope: do <exact deliverable>. Do not touch <paths or areas>. If the design is silent on
something you need, make the smallest reasonable choice and report it.
Interface: may add <X>; must not change <Y>; or unconstrained because <reason>.
Dependencies: none, or each new one with .claude/skills/proposal/dependency.md answered.

Verify by: <exact commands or observable results; paste output in the report>

Report: <the fields the coordinator needs: files changed, root cause, commands run, what is left,
open questions>. Keep it under <N> lines.

Attribution for the commit: Co-Authored-By: Claude <Model> <noreply@anthropic.com>
