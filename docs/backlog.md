# Backlog

Later than next. `## Ideas` are one line each with no commitment. `## Deferred` items have a trigger;
when it fires, the item moves to `HANDOFF.md` under Next and is cut from here. Soft cap 150 lines:
when near it, delete the oldest ideas (git has them) and turn stale deferred items into findings with
`status: superseded`.

## Ideas

- Fold a "deny `Write|Edit` on `CLAUDE.md` and `.claude/settings.json` from subagents" rule into a
  second PreToolUse hook, using `agent_type` from the hook input.
- A `limits.json` read by hooks, rule, and pusher if a third enforcement point for the doc caps appears.
- Add `fork` to any future `SessionStart` matcher list; the hooks reference lists it alongside
  `startup`, `resume`, `clear`, `compact`.
- When `docs/design/05-debug-interface.md` is written, give it a hooks section pointing at
  `.claude/hooks/test-hooks.js` (`--only`, `--verbose`, `--input <file> <hook>` modes).
- `post-edit` FORMATTERS on Windows: `spawnSync` without a shell cannot run `.cmd` shims such as
  `prettier`; a decanted project filling the table needs the `.exe` or `node <script>` form.

## Deferred

- SessionStart hook (git log, open-finding count, usage line). Why not now: the `@docs/HANDOFF.md`
  import covers continuity with no code. Trigger: a session resumes without knowing where it is, or
  the handoff needs data a file cannot hold. Constraint: never write a baseline on `compact`,
  `clear`, or `fork`.
- Stop hook that blocks once per session when work happened and `HANDOFF.md` is unchanged, with a
  `.claude/state/` directory. Why not now: three unresolved edge cases (critic review of 0002,
  objection 2) and a model turn per block. Trigger: three or more sessions end with a stale handoff
  despite the usage-gate text and the `commit` skill.
- `memory: project` for `coder`. Why not now: a second, unindexed knowledge location. Trigger: the
  same environment fact appears in two findings, or coder briefs keep restating it.
- GitHub Issues for intake. Why not now: one owner; files are discoverable and version with the
  work. Trigger: a second regular contributor, parallel coordinator worktrees, or public users.
