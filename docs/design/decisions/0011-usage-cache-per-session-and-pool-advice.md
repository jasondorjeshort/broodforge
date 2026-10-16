# 0011. The usage cache is per session, the account a label, the pool advice at HOLD

Date: 2026-10-16
Status: accepted

Record: proposal `../proposals/0011-usage-gate-under-account-switching.md` and
`../proposals/0011-critic-review.md`. Evidence: `docs/findings/2026-10-16-account-switcher-practices.md`;
facts: `docs/reference/claude-swap.md`. Amends ADR 0006 (the key table, which is living, and the
`resets_at` source in Writing a grant) and the cache paragraph of `0002-hook-specs.md` section 3.

## Context

The status line cached the meters Claude Code pipes to it in one file, `~/.claude/usage-state.json`,
and the gate judged that file. That assumed one account per machine. With an account switcher
(claude-swap, `cswap`) the login changes under a running session (`cswap switch`, `cswap auto`)
and sessions on different accounts run side by side (`cswap run`, which sets `CLAUDE_CONFIG_DIR`
per session). The shared file then held one account's readings and was judged for another, and
parallel sessions overwrote it. The owner also wanted to know, at HOLD, whether another account had
room, and whether rotation and HOLD fit together at all.

Verified (critic, 2026-10-16): the status-line payload carries `session_id` and `rate_limits`, no
account identity, and Claude Code's status-line docs say to key a per-session cache by
`session_id`; the identity is `oauthAccount` (`accountUuid`, `emailAddress`, `organizationUuid`)
in `.claude.json` under `CLAUDE_CONFIG_DIR` else home, with a legacy `.config.json` precedence;
hooks inherit `CLAUDE_CONFIG_DIR` unless `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` strips it; the
Agent `tool_input` carries `subagent_type` and an optional `model`; `cswap list --json` has the
fields the reference lists and no account uuid; the usage endpoint has a per-account request
budget (one estimate, thirty an hour) that a polling status line exhausted upstream. Practice
found in the survey: switch at session boundaries (a switch cold-starts the prompt cache), let
the switcher hold the cross-account readings, pin parallel sessions to accounts.

## Decision

### The cache: one file per session

- `~/.claude/usage-state/<session_id>.json`, written by the session's status line from the
  `session_id` on stdin, read by the gate from the `session_id` in the PreToolUse payload. Always
  under the user's `~/.claude` (like the grants directory), never under `CLAUDE_CONFIG_DIR`, so
  the location depends on neither the variable nor the scrub. The directory is created on first
  write; files older than one day are deleted when a session's file is created. The old
  `~/.claude/usage-state.json` is neither read nor written, and the new status line deletes it
  once. No `session_id`: the status line writes nothing, the gate takes the stale path.
- Shape: `{ "account": { "uuid", "organizationUuid", "email" } | null, "five_hour": { "used_percentage",
  "resets_at" }, "seven_day": {...}, "updatedAt" }`, epoch seconds. Merge per window as before: a
  window on stdin replaces the cached one, an absent one keeps it. Rule order on a render: no
  window on stdin, nothing is written, whatever the label did; a window present, then if the
  cached and current labels are both known and differ in `uuid` or `organizationUuid`, cached
  windows absent from this payload are dropped, then the merge; the current label is written when
  known, else the cached one is kept (a torn read is a no-op).
- The label: `readAccount()` in `policy.js`, the one reader both scripts use: config home is
  `CLAUDE_CONFIG_DIR` when set else the home directory; `.config.json` in the config home, then
  `.claude.json` there, the first that parses with an `oauthAccount`; returns the three fields or
  `null`. It is display and consistency only, never the key.

### The status line

The segment `acct <email local part>` before the meters, omitted when the label is unknown:
`Opus 5.5 | ctx 25% | acct jdorje | 5h 24% (hold 50) ->12:20 | 7d 72% (hold 50) ->Mon 06:00`.
When the current label and the file's are both known and differ, the meters read `n/a`, as the
gate treats them. The form is the owner's to change (proposed, not reviewed by the owner).

### The gate

- Reads the session's file. The detail names the file's email when there is one:
  `Usage gate (jdorje@example.com): 5h X%, 7d Y%. ...`. When the file's label and the current
  label are both known and differ, the file is no usage data (the stale path, `whenStale`), with
  the reason on stderr; the gate never judges another account's numbers.
- `usage.pool`: `null` (base) or `{ "command": [argv], "timeoutSeconds": 15 }`. Run only when the
  gate is about to deny at HOLD after the grant check: a spawn without a shell, stdin closed,
  stdout parsed as the `cswap list --json` contract (`accounts[]`: `number`, `email`, `disabled`,
  `usageStatus`, `usage.fiveHour.pct`, `usage.sevenDay.pct`, `usageAgeSeconds`; unknown fields
  ignored). Room: not `disabled`, `usageStatus` `ok`, email not the session's label email (every
  row with that email excluded; email is the only join), 5h and 7d each below that meter's hold;
  the row with the largest minimum of hold minus pct is named. Appended to the HOLD deny:
  ` Report to the owner: account <email> (slot <n>) had room <m> min ago: 5h a%, 7d b%. The next
  session is the owner's to start there (cswap switch <n>, or cswap run <n>).`; else ` No other
  account has room (pool readings up to <m> min old); an override grant is still the lift (ADR
  0006).`; on failure (absent program, timeout, non-zero exit, bad JSON) ` No pool reading
  (<reason>).`; with the label unknown the text adds ` (this session's account not excluded)`.
  Hatchery's `policy.json` sets `["cswap", "list", "--json"]`; a decanted project keeps `null`.
- Grants unchanged. ADR 0006, Writing a grant: the weekly `resets_at` comes from this session's file.

### The practice

`CLAUDE.md`, HOLD row: the report after the handoff names the account the gate reported, and the
next session is the owner's to start there; rotation happens at session boundaries. `cswap auto`
runs only when every coordinator session was started with `cswap run --require-session`, else not
at all. The `handoff` skill records the session's account and the one the gate named. Thresholds
unchanged.

### Harness

Group S: `session_id` present and absent; two sessions side by side; label present, absent, from
the legacy file, torn; label change dropping absent windows; a no-window render after a label
change writing nothing; the prune; the old file deleted. Group U: a file for another session only;
label mismatch as stale; every existing U, UX, UO case carries a `session_id`. Group UP: `pool`
null; a scratch policy whose command is `["node", "<stub>", "<fixture>"]` for room, no room,
disabled and non-ok rows, the session's email excluded, unknown label, a missing program, a stub
that sleeps past the timeout, non-zero exit, bad JSON. Never a stub on PATH.

### Out of scope and deferred

- The per-model meter: `docs/ideas/2026-10-13-fable-per-model-usage-gate.md`; revived by a grant
  that can name a model window and a reading the status line can show.
- Judging the pool instead of the account, switching accounts from the hook, a proxy: rejected in
  the proposal (options C, D, E).
- Check while building: a hook inside a subagent carries the parent's `session_id` (docs:
  "current session identifier"). If not, a nested launch takes the stale path, fail-open, said on
  stderr.
- Residual accepted: a torn identity read on the first render after a switch writes the new
  windows under the old label; the gate treats that as stale until the next successful render.

## Consequences

- A switch under a session and parallel sessions are correct by construction; no identity is a
  key, and the scrub cannot break the cache. A new session reads `n/a` until its first response,
  which the gate never needed (it fires after a response).
- At HOLD the owner learns which account has room without running cswap; the gate pays one
  process spawn only on a refused launch, and nothing per render.
- Costs: a directory of small files with a prune rule; one more policy key; a pool parser bound to
  one tool's output shape (the base holds the mechanism, the policy the tool); harness groups to
  keep. `cswap run` on Windows cannot share history, so `--resume` does not cross accounts.
- Supersede when Claude Code's payload carries the account or per-model meters, or when the
  switcher ships a hook surface of its own.
