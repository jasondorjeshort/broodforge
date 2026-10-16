---
date: 2026-10-16
status: resolved
area: process / usage gate under an account switcher
promoted_to: docs/reference/claude-swap.md
---

# How people run Claude Code across several accounts, and what that means for the usage gate

## Question

Three ideas assume an owner who rotates accounts with a switcher such as claude-swap (`cswap`): the usage cache keeps the
previous account's readings after a switch, the HOLD line was never set against the switcher's
threshold, and the status line does not say which account its meters belong to. What practices
exist for using a switcher well, and has anyone solved a usage gate under one?

## What was tried

Web survey on 2026-10-16. Primary sources: the claude-swap README and source at `main`
(`session.py`, `usage_store.py`, `paths.py`), its PRs and issues 176, 163, 436, 382, 386, 387 and
220, and the Claude Code status-line docs. Secondary: two write-ups of daily use, and the
alternative tools claude-rotate (proxy), amux, lpm, ccrotate, teamclaude, swapdex, clauth and the
jx-grxf menu bar `cswap`. Local checks with cswap 0.27.2 on Windows: `cswap --help`, `auto --help`,
`run --help`, `cswap config`, the key
names of `oauthAccount` in `~/.claude.json`, and the cache file under the backup root.

## Result

Facts about cswap and Claude Code are in `docs/reference/claude-swap.md`. The practices:

1. Switch at session boundaries, not under a running session. A switch cold-starts the prompt
   cache; one author measured the first turn after a swap at 11 to 13 percent of a 5-hour window
   on a large context. The upstream per-session proposal (issue 382) moves a session only when its
   account is at the limit or it has been idle longer than the cache lifetime; lpm keeps running
   sessions on their account and chooses an account only for a new session, the first under 90
   percent on both windows. A handoff and a fresh session is cheaper than a swap under a coordinator
   with a large context, and is already what HOLD asks for.
2. Let the switcher be the authority on cross-account usage. `cswap list --json` gives every
   account's 5h, 7d and per-model windows (Fable among them) with freshness and pace fields, from
   its own cache with backoff. The usage endpoint has a per-account budget of roughly thirty
   requests an hour, and a status line polling it is what locked people out (issue 220), so a gate
   may ask cswap at launch time (rare) but a status line must not.
3. Pin projects to accounts (`cswap map`, `cswap run`) rather than rotating one global login when
   sessions run in parallel. Each such session has its own `CLAUDE_CONFIG_DIR`, so anything keyed
   by config home is separated for free. On Windows `cswap run` copies rather than links shared
   config and cannot share history, which the handoff practice sidesteps.
4. Nothing upstream is adoptable as a hook or status line: PRs 176 (statusline), 163 (hook env)
   and 436 (pre/post-swap commands, closed unmerged) have no maintainer response. The stable
   surfaces are `list --json`, `auto --once` with its exit codes, and `run`.
5. Terms: every source that addresses it calls rotating accounts one person owns a grey area
   ("Anthropic hasn't said whether switching accounts when one runs low is allowed", lpm);
   sharing credentials across people is what the consumer terms forbid.

Also established: the status-line payload carries `session_id` and `rate_limits` but no account
identity, and Claude Code's own docs say to key a per-session cache by `session_id` so concurrent
sessions do not read each other's state. The identity is `oauthAccount` in `.claude.json`
(`accountUuid`, `emailAddress`, `organizationUuid`), in `CLAUDE_CONFIG_DIR` when set, else the
home directory. No tool found gates agent launches on usage; swapdex has a status line but reads
machine-wide session logs, not per-account meters.

Decision taken from this: `docs/design/proposals/0011-usage-gate-under-account-switching.md`.

Sources: github.com/realiti4/claude-swap (README, src, PRs 176, 163, 436, issues 220, 382);
bjornjohansen.com/claude-swap; umesh-malik.com/blog/claude-swap-multi-account-switcher-guide;
github.com/doxaras/claude-rotate; amux.io/guides/claude-code-account-rotation;
lpm.cx/claude-code-multiple-accounts; code.claude.com/docs/en/statusline.
