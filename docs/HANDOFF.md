# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-31 by main session (wave launched after triage and the six answers)

## Now

- Account `jdorje`. Pushed at `38ebc03`. Ahead: five earlier commits, the triage commits (`b027eaa`..`85355dd`), the
  answers commit. A pusher stopped on `behind 2` earlier today (a parallel session): the rebase rewrote hashes under the live bot.
- Triage 2026-10-31: 22 ideas read, 8 Plans, 14 Questions, no deletions; scan 0 fired, 0 due, tests green with `-race`.
  Six answers recorded the same day (texts, trailer, WIP end, discard, ended category, auto); five are Plans now.
- Wave in flight (lanes, each its own agent, disjoint files): L1 `tester` auto spike (finding); L2 `coder` the five
  texts and the credential default; L3 `coder` P38; L4 `tester` three chores (run skill, `delegation.md` sentence,
  `pre-commit` scan). Then: `reviewer` over the triage and answer commits, `pusher`.
- Live instance `%LOCALAPPDATA%\Broodforge\dev-live` (the owner's; agents never touch it). `dev`: dry-run, bot stopped,
  schema 4 (P38 makes it 5), `cred-2ha7apqowv`, four real turns granted, none spent.

## Next

1. Auto mode (the owner's priority): on the spike's pass, `architect` amends ADR 0020 and 04's mode rows (third
   value of `mode`, project and channel), then `coder`. On a fail, the idea waits for a later engine.
2. Owner-answered Plans, `coder`, sequential (shared files): WIP commit at end plus the two-button discard
   (`end-a-session-without-handing-off`, `ending-a-session-with-uncommitted-changes`); the Ended sessions category.
3. P33 to P37 (`08-implementation-plan-landing.md`) after P38; P34 waits for Next 2 to clear `handoff.go`. Real turns
   only on the owner's word. Then the carry-back Send for the two chores of L4 (ADR 0024 rows name the ideas).
4. `tester`: the two session leftovers (after P38 leaves `turn.go`); the P11 remainder, serve snapshot and the
   session-start link button (`coder`, Plans in `docs/ideas/`). Held: the web-reach proposal (Fable budget).
5. W6: the owner re-runs `spike/setup-host/` (README); then P23, P25. Hatchery: land the five carry-back branches.

## Waiting on user

- `Question:` blocks in `docs/ideas/` (fourteen new today, all later-wave features; none blocks the wave). Wording
  the owner sees on sight: discard confirm and summary lines, the WIP end summary line (in the answers). The
  `Waits:` ideas stay. Older: 0028's venue lines; dry-run and setup texts; four Discord strings; two canonical rows
  and refuse-or-warn; subagent indent; the `pre-push` line; P12a; "Never ends for inactivity."; the switch post;
  the hook no-answer phrase; R-17; `C:\elsewhere`; W6's console window.

## Open findings

`node .claude/hooks/queue-list.js findings --status open`. Live: `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` (Next 5); the 2026-10-23 and 2026-10-24 session findings (the handoff import gap seen again today); `2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (owner).
