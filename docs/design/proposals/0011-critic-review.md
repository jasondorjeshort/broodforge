# Critic review of proposal 0011: the usage gate and status line under an account switcher

Date: 2026-10-16
Reviewer: critic (Fable 5.1)
Proposal: `0011-usage-gate-under-account-switching.md` (status: under review)

Scope note. The proposal was written by the coordinator rather than the architect, on the owner's
instruction; this review is the only independent pass before the ADR. The architect's answer step
in the `proposal` skill still applies. Not re-argued, per the brief: the thresholds, option D (the
gate switching accounts), option E (a proxy).

## Claims to verify

| # | Claim | Result | Evidence |
|---|---|---|---|
| 1 | `cswap run` sets only `CLAUDE_CONFIG_DIR` | Verified, with two caveats | `session.py` at `main`: `env["CLAUDE_CONFIG_DIR"] = str(session_dir)` after filtering `AUTH_OVERRIDE_ENV_VARS` (`ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN` and their `_FILE_DESCRIPTOR` variants). Caveats: when the slot is already the default login, `run` launches plain `claude` with the environment unchanged (no `CLAUDE_CONFIG_DIR`, no scrub); a preset `CLAUDE_CONFIG_DIR` is overridden with a warning. |
| 2 | Hooks and the status line see `CLAUDE_CONFIG_DIR` | Verified for hooks; inferred for the status line; one documented exception | hooks docs: "A hook process inherits the parent environment, apart from the `OTEL_*` exporter variables ... and, when `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB` is set to `1`, the variables it strips." env-vars docs, scrub table: `CLAUDE_CONFIG_DIR` | Removes it. The status-line page states no inheritance rule; it says Claude Code sets `COLUMNS` and `LINES` before running the script and the env-vars page says `CLAUDECODE` and `CLAUDE_CODE_CHILD_SESSION` are set in status-line subprocesses, so the status line gets an environment Claude Code builds from its own; this session's Bash subprocess carries `CLAUDE_CODE_SESSION_ID`, `CLAUDE_PID`, `CLAUDE_EFFORT`, which is the same spawn path. Not tested under `cswap run`: it has never run on this machine (no `~/.claude-swap-backup/sessions/`). |
| 3 | `.claude.json` in `CLAUDE_CONFIG_DIR` else home; legacy `.config.json` rule | Verified; the reference states the legacy rule too narrowly | `paths.py`: base is `CLAUDE_CONFIG_DIR` when set else `Path.home()`; `_pick_global_config` docstring: "Claude Code 2.1.299's choice". The legacy file wins when it is a regular file and (non-empty, or empty and writable with no `.claude.json`). Claude Code docs (claude-directory): "If you set `CLAUDE_CONFIG_DIR`, every `~/.claude` path on this page lives under that directory instead", and `~/.claude.json` is on that page. |
| 4 | No account field in the status-line payload | Verified | Full JSON schema on code.claude.com/docs/en/statusline (2026-10-16): no account, email or organization field; `agent.name` is the `--agent` name. `session_id` is present, and the page's cache example says to key per-session caches by it. |
| 5 | `cswap list --json` fields; additive contract | Partly verified | README "JSON output for scripting": `schemaVersion` (1), `activeAccountNumber`, `accounts[]` with `number`, `email`, `active`, `alias`, `disabled`, `usageStatus` (`ok`, `unavailable`, `token_expired`), `usage.fiveHour`/`sevenDay` (`pct`, `resetsAt`), `usageFetchedAt`, `usageAgeSeconds`, `lastGoodUsage`, `usageError`, `usageRetryAt`, `loginExpiresAt`. "Per-model `scoped` entries" are mentioned with no field names; the internal cache (`cache/usage.json` on this machine) has `scoped[]` with `name`, `pct`, `resets_at`. The "ignore unknown fields" sentence is written for `auto --json`, not `list`. `list` exit codes are undocumented beyond "an error object with a non-zero exit code". No `accountUuid` in the output; `organizationUuid` only in the internal cache. |
| 6 | `auto --once` exit codes 0, 1, 2, 3 | Verified | `cswap auto --help`, 0.27.2: 0 switched, 1 error (network trouble, lock contention), 2 no action, 3 blocked. `cswap run --help` labels `run` `[EXPERIMENTAL]`. |
| 7 | About thirty usage requests an hour per account | Verified as one estimate | Issue 220: the reporter wrote "~28-30 req/h budget", unmeasured; the lockout came from a widget retrying every 2-second render after a 429; the maintainer only labelled the issue. |
| 8 | A switch costs 11 to 13 percent of a 5-hour window | Verified as one anecdote | bjornjohansen.com: "I've seen 11-13% of the 5-hour quota go on the first request after a swap"; no context size or method. |
| 9 | Switch takes effect on the next message on Windows and Linux | Verified, with an inconsistency | umesh-malik.com: "On Linux and Windows the new account applies on your next message"; the same post says Windows uses "a protected credential store", which does not fit its own file-re-read explanation. macOS keychain cache about 30 s. |
| 10 | `tool_input.subagent_type` in the PreToolUse payload | Verified | hooks docs, PreToolUse, Agent table: `prompt`, `description`, `subagent_type`, and optional `model` ("Optional model alias to override the default"). The agent files here use aliases (`fable`, `opus`, `sonnet`, `haiku`), so a case-insensitive match against `scoped[].name` `Fable` works. |

Also checked: `~/.claude.json` is 97 KB with 20 keys under `oauthAccount` (among them `accountUuid`,
`emailAddress`, `organizationUuid`, `organizationRateLimitTier`, `userRateLimitTier`); read plus
parse costs 0.7 ms (50 iterations). Command hooks have a 600-second default timeout on most events
(hooks docs, UserPromptSubmit section), so a 15-second child is within budget. Claude Code here is
2.1.296; the agent `model:` values are aliases.

## Objections

### 1. The model meter is a refusal nothing documented can lift (blocking for Stage 2)

Stage 2 adds a HOLD on the launched agent's per-model window, judged by the `sevenDay` thresholds.
The lift for HOLD is a grant, and a grant names meters from a fixed list: `policy.js` has
`METERS = ['5h', '7d']`, `classifyGrant` makes a grant malformed when `meters` names anything else,
and `holdCovered` requires every tripped label to be in the lifted set. A tripped `Fable` label is
never lifted, so the only lifts are `usage.overrideEnv` (set at session start, the owner's
instruction) or editing the policy. The deny would still end with "Lifting this early is the
user's decision (ADR 0006)", pointing at a mechanism that cannot lift it. CLAUDE.md says the grant
is the owner's way to spend past HOLD; the proposal's own practice section says "the grant is
still the lift". Second half: the status line never calls cswap, so it cannot show the model
meter; the owner sees no `HOLD` on the line while the gate denies every architect and critic
launch, and the spec's promise that ` HOLD lifted` tracks the gate's verdict no longer holds.

Resolve: define the lift before the meter ships. Either a grant with no `meters` lifts every
tripped meter including model windows and `meters` may name a model (`"Fable"`), with the deny
saying so and ADR 0006's grant paragraph amended; or defer the model meter (the idea's `Waits:` is
still valid) and ship Stage 2 as pool advice only. Either way the deny must say the status line
does not show this meter.

### 2. "Room" is judged against cswap's active account, which is not the session's account under `cswap run` (serious)

The proposal defines room as "an account other than the active one". cswap's `active` is the
default login. A `cswap run` session's account is by design not the default login, so the pool
advice in that session can name the session's own account as the one with room (cswap's reading
may be older and lower than the cache's), or skip the default-login account that has room.
This is the parallel-sessions case the proposal exists for. Further, `list --json` carries no
`accountUuid`, so the only join between the cache and the pool is the email, and cswap's own
identity is email plus `organizationUuid` (reference, Where things live).

Resolve: exclude the account whose email equals the cache's `account.email` (and say in the deny
which account the readings are for); state the join rule and what happens when two rows share an
email; add the `cswap run` case to the UA harness group.

### 3. `accountUuid` alone is probably the wrong identity (serious; the org-scoping is unverified)

cswap identifies a profile by email plus `organizationUuid`, and the live `oauthAccount` carries
`organizationRateLimitTier` and `userRateLimitTier`, which reads as limits that are scoped per
organization membership. One person with one Anthropic account in two organizations (a personal
Max and a team seat) then has one `accountUuid` and two quota pools; the mismatch drop never
fires between them, and the original bug (the previous pool's readings judged after a switch)
returns in exactly the configuration a switcher is for. I could not verify from Claude Code's
docs that the meters are per organization; the key names and cswap's choice both point that way.

Resolve: key the cache on `accountUuid` plus `organizationUuid`, show the email, and add a
harness case for the same uuid under two organizations. If the architect can show the meters are
per user regardless of organization, keep the uuid and record the evidence in the reference.

### 4. The subprocess environment scrub removes `CLAUDE_CONFIG_DIR` (serious)

The env-vars page's scrub table lists `CLAUDE_CONFIG_DIR` | Removes it, for "the subprocesses it
starts, such as Bash commands, hooks, and stdio MCP servers". With `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`,
a `cswap run` session's gate and status line resolve the config home to `~/.claude`, read the
default login's `.claude.json` and the home cache, and judge the wrong account silently. The
proposal's "What would change this recommendation" names only "evidence that hooks do not
inherit `CLAUDE_CONFIG_DIR`"; the evidence is a documented setting, and the scrub is the kind of
hardening a project adopts without re-reading this ADR.

Resolve: either document the incompatibility where the scrub would be set (the reference and the
`usage.accounts` key's description) and have the status line say `acct ?` when it cannot tell, or
key the cache in a way that does not depend on the variable (objection 5).

### 5. A per-`session_id` cache is the stronger Stage 1; the proposal lists it only as the fallback (serious)

The strongest alternative. Key the cache by `session_id` (`<config home or ~/.claude>/usage-state/<session_id>.json`),
which both scripts already receive on stdin (the gate reads it for grants). Then: the readings a
session's status line receives are by construction its own login's, so a switch under the
session shows on the next render with no identity read and no drop rule; parallel sessions never
share a file, with or without `CLAUDE_CONFIG_DIR`, scrub or not; and it is what Claude Code's own
status-line page prescribes for a cache ("unique across sessions so concurrent sessions ... don't
read each other's cached state", using `session_id`). The gate only ever fires after an API
response in the same session (an Agent call comes from an assistant message), so the per-session
file is already written by the time the gate runs, modulo the 300 ms debounce race that exists
today. What it costs: the cross-session carry-over (a new session's line reads `n/a` until its
first response, which the gate never needed); the coordinator reads `seven_day.resets_at` for a
grant from the session's file instead of a fixed path (ADR 0006, Writing a grant, needs the edit
either way, since the proposal moves the file too); and a prune rule (delete files older than
`staleSeconds` on write). The account read from `.claude.json` stays, as display only: the
segment and the deny's email, where a torn read or a missing field costs nothing.

Resolve: make per-session keying the Stage 1 mechanism and the account a label, or show a need
for cross-session carry-over that the gate has. If the account key stays, objections 3, 4 and 11
all have to be answered on their own.

### 6. A stub `cswap` on PATH is not a sound harness seam on Windows (serious)

The hooks spawn without a shell (`post-edit.js`, and the idea `2026-10-13-post-edit-formatters-cmd-shims.md`
records that a `.cmd` shim then cannot be the command). A harness cannot write a `.exe`; a `.cmd`
stub needs `shell: true`, which makes `cmd.exe` quoting and `PATHEXT` order part of the test. And
the real `cswap.exe` is on PATH on the development machine (`C:\ProgramData\uv\bin`), so any case
that fails to put the scratch directory first calls the real tool, which may fetch from the usage
endpoint during a test run and spend the budget objection 7 protects. ADR 0006's own pattern for
this is an environment seam read from Claude Code's environment (`GIT_GUARD_MERGED`,
`POST_EDIT_FORMATTERS_JSON`).

Resolve: a seam that replaces the command (`USAGE_GATE_POOL_COMMAND`, a JSON argv such as
`["node", "stub.js", "fixture.json"]`), with the missing-binary case an argv that does not exist
and the timeout case a script that sleeps; never PATH. Objection 12 gives the policy shape that
makes the seam the same mechanism as the real setting.

### 7. Every Fable launch, not only HOLD, puts a network-and-lock CLI in the launch path (serious)

The gate runs cswap "after the per-account verdict is HOLD or the agent's model has a scoped
window to check". Whether a model has a scoped window is known only by running cswap, so every
architect and critic launch runs `cswap list --json`: a refresh of due accounts over the network
(the internal cache has `nextPollAt` per account), a lock (`auto --once` documents "lock
contention" as an error class), and up to 15 s of stall. Two sessions launching at once contend.
Failure fails open with a note on stderr, which for a hook that exits 0 without JSON nobody sees,
so the model meter is best-effort in a way the owner cannot observe, and the status line cannot
show it (objection 1).

Resolve: run cswap at HOLD only (advice), and take the model reading, if the meter ships at all,
from a source the status line can also show; or state in the ADR that the model meter is
advisory and name where its failures are visible.

### 8. The per-invocation `model` overrides the agent file (minor)

The hooks docs list an optional `model` in the Agent `tool_input`, and the sub-agents page gives
the resolution order: the per-invocation parameter first, then the frontmatter, then
`CLAUDE_CODE_SUBAGENT_MODEL`, then the main model. The proposal reads only the file, so a
`general-purpose` launch with `model: fable` escapes the meter and a `coder` launched with
`model: sonnet` is judged by the Fable window.

Resolve: `tool_input.model` first, then the file's `model:`; say that the environment overrides
and `inherit` are out of reach.

### 9. The deny addresses the wrong reader (minor)

The agent that reads the deny can neither run `cswap switch` (machine-wide, and the practice
forbids it under a session) nor start the next session. "Hand off, then start the next session
with cswap switch <n>, or cswap run <n> in a mapped directory" reads as an instruction to the
agent, and "mapped directory" is cswap jargon. The "No other account has room" clause does not say
the grant is still the lift, though the practice section does. The room clause should carry the
reading's age, since cswap readings can be hours old.

Resolve: "Report to the owner: account <email> (slot <n>) had room <m> min ago: 5h a%, 7d b%.
The next session is the owner's to start there (cswap switch <n>, or cswap run <n>)." and, on no
room, "No other account has room (cswap, readings up to <m> min old); an override grant is still
the lift (ADR 0006)."

### 10. The segment's fallback is unknowable at render time (minor)

"A second account with the same email local part: the segment shows the full address" needs the
status line to know the other accounts, and it reads one `.claude.json` and must not call cswap.

Resolve: pick one form (the full address, or the local part with no promise), or a policy key for
the form. The wording is the owner's in any case; the proposal says so.

### 11. The identity read: the legacy rule, the torn read, and the gate's blindness to a mismatch (minor)

Three small things in one place. (a) The reference states the legacy `.config.json` rule as
"non-empty regular file"; cswap's mirror of Claude Code 2.1.299 also takes an empty, writable
`.config.json` when no `.claude.json` exists. For an identity read the write-target rule is not
needed: try `.config.json` then `.claude.json` and take the first that parses with an
`oauthAccount`. (b) Claude Code rewrites `.claude.json` often (it keeps five backups of each
rewrite); a torn read parses as unknown. The proposal keeps the cached windows then, good, but if
that render writes `account: null`, the next render with a known account drops the windows as "a
cache without an account". On an unknown identity, keep the cached account field too, so a torn
read is a no-op. (c) The gate is "unchanged in judgement": after `cswap switch` it judges the
file's account until the status line next writes, so the first launch after a switch can be
judged on the previous account's numbers, in either direction. The gate can read the same
identity and treat a mismatch as "no usage data" (the stale path, which `whenStale` already
governs) instead of judging another account's readings silently. Per-session keying (objection
5) removes (b) and (c).

Reading a 97 KB file per render is fine: 0.7 ms measured; the line runs at most once per 300 ms
and every 60 s when idle.

### 12. `usage.accounts: "cswap"` bakes one tool into the base schema (minor)

The loader rejects unknown values, so a second switcher is a schema change; the base script
would parse one vendor's output; `list --json`'s stability promise is written for `auto --json`,
its exit codes are undocumented, and `run` is marked experimental in 0.27.2's own help. The
proposal's own rule is that the base must work without cswap and the project turns it off.

Resolve: name the key by what it does and let the project supply the tool, as `edit.formatters`
does: `usage.pool: null | { "command": [argv], "timeoutSeconds": 15 }`, output parsed as the
documented `list --json` contract. The base then holds mechanism, Hatchery's policy holds
`["cswap", "list", "--json"]`, and the harness seam of objection 6 is the same key.

### 13. The practice's "net" fires under the sessions it is meant to protect (minor)

"Where [`cswap auto`] runs, its threshold stays above HOLD as a net" conflicts with the reference:
`auto` "switches the default login only. Sessions started with cswap run keep their account." So
the net switches the login under every plain session, which is the mid-session switch the
proposal forbids; it is harmless only for `cswap run` sessions. The same-account fast path of
`cswap run` (claim 1) launches a plain session that a later switch pulls out from under, which is
what `--require-session` guards.

Resolve: the practice says `cswap auto` runs only when every coordinator session was started with
`cswap run --require-session`, otherwise not at all; and the handoff records the start command.

## Recommendation

Adopt Stage 1 with the cache keyed by `session_id` under the config home (Claude Code's own
guidance, no identity dependence, no drop rule, scrub-proof) and the account read from
`.claude.json` as a label only (uuid plus organization recorded, email shown), with the gate
treating a label mismatch as stale rather than judging it. Reduce Stage 2 to pool advice at HOLD
through a policy-declared command (`usage.pool`), joined to the cache by email with the session's
own account excluded, with the deny reworded for the agent that reads it and the reading's age in
it. Defer the model meter until the grant schema can name a model window and the status line can
show it; the idea's `Waits:` stays. Confidence: high on objections 1, 4, 6 and 10 (each rests on
a document or on the code as it stands); moderate on 3 (the org-scoping of the meters is
inferred from field names and cswap's identity choice, not from Claude Code's docs) and on 5 (I
have not measured the debounce race, and the carry-over may have a use I did not find). If the
architect keeps the account key, objections 3, 4 and 11 need their own answers; if the model
meter ships, objection 1 is the one that cannot be answered by wording.
