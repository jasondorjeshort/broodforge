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

- From Dystopo (2026-10-11), each a candidate for a skill, hook or rule once proposal 0003 is
  decided: a private-terms guard (a gitignored `private-terms.local.json` of names, boxes and
  coordinates; a commit hook and a scanner for PR text and built artifacts); a `verify` agent and
  the rule that the coordinator reads reports, not diffs; a release skill (tag-driven draft, verify
  the artifact downloaded from the draft itself, hand-written notes, publish; version strings
  derived from the tag); a `sweep` skill (one command, JSON summary, contact sheet; findings become
  the next wave's sweep task); the lesson that the first install on a real device, release build,
  empty, UI only, found a dead end no emulator sweep could; per-agent port and emulator allocation
  in briefs; the Haiku runner executes sweeps and escalates to Sonnet only on failure.
- Usage gate thresholds: Dystopo ran at 75% with the owner's clarification that only new
  implementers stop there while in-flight verification and merges finish. Hatchery's 50/80 is
  untested on a build-phase project; decide with proposal 0003 whether the gate needs a phase or
  an override for in-flight work (Dystopo uses an environment variable the coordinator sets).
- `git-guard` rules 3 and 4 (only the pusher pushes, only the runner commits) fit the single-branch
  phase; in the wave model every implementer pushes its own `task/*` branch and opens a PR. Key the
  rules off a phase setting or the branch name when 0003 is promoted.
- ADR numbers collide between Hatchery and its projects (both can write `0003` next); candidate fix:
  Hatchery reserves `0001`-`0019`, projects start at `0020` (`adr` and `decant` skills). ADR 0004 defers it.
- The main session may push on the owner's instruction (CLAUDE.md says so); say it in the `commit`
  skill's Pushing section too, so a coordinator does not launch a pusher for a push the owner asked for.

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
- Fable-specific usage gate (same 50/80 rule, applied to `architect` and `critic` launches). Why not
  now: Claude Code 2.1.296 passes the status line only `rate_limits.five_hour` and `.seven_day`, with
  no per-model meter (checked docs and live payload, 2026-10-12); until then run `/usage` by hand
  before a Fable launch. Trigger: the statusline docs or payload gain a per-model field.
- GitHub Issues for intake. Why not now: one owner; files are discoverable and version with the
  work. Trigger: a second regular contributor, parallel coordinator worktrees, or public users.
