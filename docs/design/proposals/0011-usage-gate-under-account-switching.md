# 0011. The usage gate and status line under an account switcher

Date: 2026-10-16
Status: promoted to ADR 0011

Written by the coordinator from its own research (`docs/findings/2026-10-16-account-switcher-practices.md`),
on the owner's instruction to decide rather than ask; the critic is the independent pass
(`0011-critic-review.md`). Revised after the review; the first draft is in git.

## Question

The gate (`.claude/hooks/usage-gate.js`) and the status line (`.claude/statusline.js`) assume one
account per machine: the status line caches the meters Claude Code pipes to it in
`~/.claude/usage-state.json`, and the gate judges that file against `usage.hold`. With claude-swap
(`cswap`, facts in `docs/reference/claude-swap.md`) the login can change under a running session
(`cswap switch`, `cswap auto`), sessions on different accounts can run side by side (`cswap run`),
and HOLD was set for one account. What should the cache, the gate, the status line and the HOLD
rule do, and which parts of the practice change?

## Constraints that matter here

- The status-line payload carries `rate_limits` for the current login and `session_id`, and no
  account identity. Claude Code's status-line docs say to key a per-session cache by `session_id`.
  The identity is `oauthAccount` in `.claude.json` (`accountUuid`, `emailAddress`,
  `organizationUuid`), in `CLAUDE_CONFIG_DIR` when set, else the home directory.
- `cswap run` sets exactly one variable, `CLAUDE_CONFIG_DIR`; `cswap switch` rewrites the default
  profile in place and the session picks it up on its next message. With
  `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, hooks and the status line do not see `CLAUDE_CONFIG_DIR`.
- The usage endpoint has a per-account request budget (one estimate: thirty an hour). The status
  line runs on every render and must never call cswap; the gate runs once per launch and may, but
  only when the launch is already refused.
- A switch under a running session cold-starts the prompt cache (one anecdote: 11 to 13 percent
  of a 5-hour window). Handoff and a fresh session is the cheaper boundary and is already the HOLD
  response in `CLAUDE.md`.
- The gate fails open (ADR 0006) and nothing here may change that. Every refusal has a documented
  lift. The base must work without cswap, and a project supplies or removes the tool in `policy.json`.
- Thresholds decide what the owner spends; this proposal changes none. What humans see (the status
  line) is the owner's to change; a format is proposed so it can be argued.

## Options

### A. Account-aware cache and status line only

The cache stops being one shared file; the status line names the account; the gate is unchanged
in judgement. HOLD stays per account; the owner rotates by hand at a handoff. Fixes both bugs
(switch under a session, parallel sessions) but leaves the owner to run `cswap list` after every
HOLD to learn which account has room.

### B. A, plus the gate consults the pool at HOLD (recommended)

A policy key `usage.pool` names a command whose output is the `cswap list --json` contract. When a
launch is refused at HOLD, the gate runs it and the deny reports the account with room, for the
owner, with the reading's age. The gate never switches an account and never runs the command on a
launch it allows.

The first draft also judged the launched agent's per-model weekly window from the same output.
Withdrawn after the review (objection 1): a grant cannot name a model window, so the refusal had
no lift, and the status line cannot show the meter. It stays an idea, with what it now waits on.

### C. The gate judges the pool: hold only when no account has room

Rejected: a HOLD that never trips while any account has room is not a pacing rule; it would let one
session drain the pool, and a running session cannot use another account's room without the switch
cost. The pool is advice, not verdict.

### D. The gate switches accounts (`cswap auto --once` from the hook)

Rejected: a hook that changes the default login changes it for every session on the machine, which
no deny message can undo, and pays the cold-cache cost on every session that was running. The
right time to switch is a session boundary, which the gate does not own.

### E. A rotating proxy (claude-rotate) instead of a switcher

Rejected for the base: a server to run, undocumented headers it reads, and a platform scope the
base does not have. A project that adopts one needs none of A or B; its single login is the proxy.

## Recommendation

B, built in two stages, with the practice change recorded in `CLAUDE.md` and the `handoff` skill.

### Stage 1: a cache per session, the account as a label

- Cache: `~/.claude/usage-state/<session_id>.json`, one file per session, written by that session's
  status line from the `session_id` on stdin and read by the gate from the `session_id` in the
  PreToolUse payload. The directory is the user's, like the grants directory, and is not under
  `CLAUDE_CONFIG_DIR`: the file is correct with or without the variable, scrubbed or not. The
  readings a session receives are its own login's by construction, so a switch under the session
  shows on its next render and parallel sessions never share a file. No `session_id` on stdin:
  the status line writes nothing; the gate takes the stale path. The old single file is neither
  read nor written.
- Prune: when a session's file is created, files in the directory older than one day are deleted.
- Merge within the file as now: a window present on stdin replaces the cached one, an absent one
  keeps it, nothing is written when neither is present.
- Label: one function in `policy.js`, shared by both scripts so they cannot disagree, reads the
  identity from the config home (`CLAUDE_CONFIG_DIR` when set, else the home directory):
  `.config.json` in the config home, then `.claude.json`, the first that parses with an
  `oauthAccount`, as `{ uuid, organizationUuid, email }` or `null`. The status line calls it on
  every render and records `account` in the file. When the identity cannot be read, the cached
  label is kept, so a torn read is a no-op. When the payload carries no window, nothing is
  written, label change or not: this rule comes first, so a render between a switch and the next
  response leaves the old label and windows in place for the gate to classify as stale, rather
  than a fresh file with no windows that would judge as clear. When a window is present and both
  the cached and the current label are known and differ (`uuid` or `organizationUuid`), cached
  windows absent from this payload are dropped before the merge: the login changed under the
  session and a stale window is the old account's. The directory is created on first write.
- Status line: the email's local part as its own segment before the meters, since the meters are
  that account's: `Opus 5.5 | ctx 25% | acct jdorje | 5h 24% (hold 50) ->12:20 | 7d ...`.
  Omitted when unknown, like line 1. One form, no fallback. When the current label and the file's
  differ, the meters read `n/a`, as the gate treats them. The owner may change wording or
  placement; nothing else depends on it.
- Gate: reads the session's file; the detail names the label's email when there is one
  (`Usage gate (jdorje@example.com): 5h X%, 7d Y%. ...`). It reads the current identity the same
  way; when both are known and differ, the file is treated as no usage data (the stale path that
  `whenStale` governs, with the reason on stderr), never judged as the other account's numbers.
- Grants: unchanged, in `~/.claude/usage-gate-overrides/`. ADR 0006, Writing a grant: the weekly
  `resets_at` for a project or user grant is read from this session's file.
- Harness group S gains: `session_id` present and absent; two sessions writing side by side; the
  label present, absent, from the legacy file, torn (kept); a label change dropping absent windows;
  a no-window render after a label change writing nothing; the prune. Group U gains: file for
  another session only; label mismatch as stale; and every existing U, UX and UO case gains a
  `session_id`, since without one they would all take the stale path.
- The old `~/.claude/usage-state.json` is neither read nor written; the first run of the new
  status line deletes it, so no project keeps a dead file.
- To check while building: whether a hook firing inside a subagent carries the parent's
  `session_id` (the docs say "current session identifier"; `agent_id` is what differs). If not,
  a nested launch finds no file and takes the stale path: fail-open, and the note on stderr says so.

### Stage 2: the pool at HOLD, `usage.pool`

- Policy: `usage.pool`: `null` (base) or `{ "command": [argv], "timeoutSeconds": 15 }`. The
  command's stdout is parsed as the `cswap list --json` contract (`accounts[]` with `number`,
  `email`, `disabled`, `usageStatus`, `usage.fiveHour.pct`, `usage.sevenDay.pct`,
  `usageAgeSeconds`); unknown fields are ignored. Hatchery's policy sets
  `["cswap", "list", "--json"]`; a decanted project keeps `null` unless it has a switcher. The
  key names what it does, not the tool, as `edit.formatters` does, and it is the harness seam.
- When: only when the gate is about to deny at HOLD (no grant lifts it). Never on an allowed
  launch, never at a deny line, never per render. A spawn without a shell, the timeout from the
  policy, stdin closed.
- Room: an account not `disabled`, with `usageStatus` `ok`, whose email differs from the session's
  label email (every row sharing that email is excluded; `list --json` carries no uuid, so email
  is the join), and whose 5h and 7d readings are each below that meter's hold. The one with the
  most headroom (the largest minimum of hold minus pct over the two meters) is named.
- Deny text appended to the HOLD deny. With room: ` Report to the owner: account <email> (slot
  <n>) had room <m> min ago: 5h a%, 7d b%. The next session is the owner's to start there
  (cswap switch <n>, or cswap run <n>).` Without: ` No other account has room (pool readings up to
  <m> min old); an override grant is still the lift (ADR 0006).` When the command fails (absent,
  timeout, non-zero exit, unparseable): ` No pool reading (<reason>).` so the failure is visible
  where the deny is read. The label unknown: the session's account cannot be excluded, and the
  text says ` (this session's account not excluded)`.
- Harness group UP: `pool` null; a scratch policy whose command is `["node", "<stub>",
  "<fixture>"]`: room, no room, disabled and non-ok rows skipped, the session's email excluded
  (the `cswap run` case: the active row is not the session's), unknown label, a missing program,
  a stub that sleeps past the timeout, non-zero exit, bad JSON. Never a stub on PATH.

### Deferred: the per-model meter

`docs/ideas/2026-10-13-fable-per-model-usage-gate.md` stays, with what it waits on: a grant that
can name a model window (ADR 0006 amendment) and a reading the status line can show without a
per-render call. When it ships, `tool_input.model` is read before the agent file's `model:`.

### The practice

- `CLAUDE.md`, usage table, HOLD row: after the handoff and the commit, the report names the
  account with room when the gate reported one; the next session is the owner's to start there.
  Rotation happens at session boundaries.
- `cswap auto` switches the default login under every plain session, so it runs only when every
  coordinator session was started with `cswap run --require-session`; otherwise not at all.
- `handoff` skill: Now records the account the session ran on (the status line's segment) and
  the account the gate named, so the next session starts there.
- Thresholds unchanged: HOLD 50 stays the per-account pacing line and the handoff trigger; an
  override grant stays the owner's way to spend past it, with one account or several.

### Claims to verify

Verified by the critic on 2026-10-16 (table in `0011-critic-review.md`); the corrections are in
`docs/reference/claude-swap.md`. Still unverified and now irrelevant to the mechanism: whether
meters are scoped per organization (the label records the organization either way), and the
status line's environment under `cswap run` (the cache no longer depends on it).

## What would change this recommendation

- Claude Code adding account identity to the status-line payload: drop the `.claude.json` read.
- claude-swap merging a hook environment, lifecycle commands or a statusline command: take the
  upstream surface over the `.claude.json` read.
- A gate that needs readings from a previous session (none found): the per-session cache would
  need a carry-over rule.

## Critic's objections and responses

1. Model meter has no lift (blocking). Accepted: the meter is withdrawn from Stage 2 and the idea
   records what it waits on. Stage 2 is advice only.
2. Room judged against cswap's active account (serious). Accepted: room excludes every row whose
   email is the session's label email; the deny names the session's account; the `cswap run` case
   is a harness case. Unknown label: not excluded, and the text says so.
3. `accountUuid` alone the wrong key (serious). Accepted with change: the cache is no longer keyed
   by account at all; the label records `uuid` and `organizationUuid` and a label change compares
   both. Org-scoping of meters stays unverified and no longer matters to correctness.
4. The subprocess scrub removes `CLAUDE_CONFIG_DIR` (serious). Accepted: the cache lives under
   `~/.claude` and is keyed by session, so the variable only affects the label's source; with the
   scrub set, a `cswap run` session's label may be the default login's. Recorded in the reference.
5. Per-`session_id` cache (serious). Accepted with one change: the directory is `~/.claude`, not
   the config home, so the file's place does not depend on the variable or the scrub, and the
   grant-writing step finds it. The prune is one day rather than `staleSeconds`, so the last
   sessions' files stay readable for debugging.
6. Stub on PATH unsound (serious). Accepted: the seam is the policy's `command` argv.
7. cswap on every Fable launch (serious). Accepted: the pool command runs only when the gate is
   about to deny at HOLD; failures are in the deny text.
8. `tool_input.model` precedence (minor). Accepted; recorded with the deferred meter.
9. Deny wording (minor). Accepted; the critic's wording, with the failure clause added.
10. Segment fallback unknowable (minor). Accepted: local part, one form.
11. Identity read details (minor). Accepted: first file that parses with `oauthAccount`; torn read
    keeps the cached label; the gate treats a label mismatch as stale.
12. `usage.accounts: "cswap"` bakes in a vendor (minor). Accepted: `usage.pool` with an argv.
13. The `cswap auto` net fires under plain sessions (minor). Accepted: the practice is as stated.

Confirmation pass (the critic, same day, on the revision): all thirteen resolved; three build
details taken into the body: one shared identity reader, the no-window rule before the
label-change rule (a render between a switch and the next response must not write an empty fresh
file), the old cache file deleted once, `session_id` added to the existing harness cases, and the
subagent `session_id` check. One residual accepted: a torn identity read on the first render after
a switch writes the new windows under the old label, which the gate treats as stale (allow under
`whenStale: allow`) until the next successful render, at most a minute or a message later.
