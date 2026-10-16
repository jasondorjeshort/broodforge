---
verified: 2026-10-16
source: https://github.com/realiti4/claude-swap (README and src at main, 2026-10-16); observed with cswap 0.27.2 on Windows 11
---

# claude-swap (`cswap`)

An unofficial switcher that keeps several Claude Code logins and swaps the active one. Claude Code,
not cswap, enforces each account's limits. Rotating between accounts one person owns is a grey
area under the consumer terms; sharing one account's credentials across people is what the terms
forbid. Nothing in the base requires cswap; what reads it fails open when it is absent.

## Where things live

- Backup root (Windows, macOS): `~/.claude-swap-backup/`; Linux: `$XDG_DATA_HOME/claude-swap/`.
  Settings: `settings.json` there (`cswap config`, `cswap config path`). Auto-switch state:
  `autoswitch_state.json`. Usage cache: `cache/usage.json` (internal; `list --json` is the contract).
- Claude Code's config file: `.claude.json` in `CLAUDE_CONFIG_DIR` when set, else in the home
  directory; a legacy `.config.json` inside the config home wins when it is a regular file that is
  non-empty, or empty and writable with no `.claude.json` beside it (cswap's `paths.py`, mirroring
  Claude Code 2.1.299). Its `oauthAccount` carries `accountUuid`, `emailAddress`,
  `organizationUuid`, `organizationRateLimitTier`, `userRateLimitTier` (among others). cswap
  identifies a profile by email plus `organizationUuid`.
- With `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, Claude Code removes `CLAUDE_CONFIG_DIR` from the
  environment of hooks, the status line and Bash commands (env-vars docs, scrub table), so a
  script in a `cswap run` session then resolves the default profile, not the session's.
- Credentials: `.credentials.json` in the config home (`CLAUDE_CONFIG_DIR`, else `~/.claude`).

## Switching the default login

- `cswap switch` (next), `cswap switch <n|email|alias>`, `cswap switch --strategy best` (most
  headroom; `--model Fable` also counts that model's weekly window), `--strategy next-available`.
  `--json` with `list`, `status`, `switch`; human notices go to stderr.
- Takes effect on the next message in a running session on Windows and Linux; macOS caches the
  keychain entry about 30 seconds. VS Code: close and reopen the Claude Code tab.
- Never `/logout` before a switch: current Claude Code may revoke the refresh token of the account
  being left. Add accounts with `/login` then `cswap add`; re-running `cswap add` refreshes a slot.
- A switch cold-starts the prompt cache: the next turn resends the whole context uncached. One
  user measured 11 to 13 percent of a 5-hour window for that turn on a large context.

## One session per account: `cswap run` (marked experimental in 0.27.2)

- `cswap run <n|email> [-- <claude args>]` launches Claude Code as that account in this terminal
  only. It sets exactly one environment variable, `CLAUDE_CONFIG_DIR`, to
  `<backup root>/sessions/<slot>-<email slug>/` (a preset value is overridden with a warning),
  and scrubs `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN`. Hooks of
  that session inherit the variable (hooks docs) unless the scrub above is set. When the account
  is already the default login, `run` launches plain `claude` with the environment unchanged and
  no profile, which a later `cswap switch` pulls out from under; `--require-session` refuses that.
- The profile gets `settings.json`, `keybindings.json`, `CLAUDE.md`, `skills`, `commands`,
  `agents` from `~/.claude`: copies on Windows, re-synced on every launch; symlinks elsewhere.
  User-scope MCP servers are mirrored; MCP OAuth logins are not. Chat history is per account;
  `--share-history` is not supported on Windows, so `claude --resume` does not cross accounts here.
- `cswap map <n> <dir>` binds a directory; a bare `cswap run` there launches that account.
- On Windows the command is `cswap.exe` (a uv or pipx shim), so a spawn without a shell finds
  it; a `.cmd` shim would not be found that way.

## Automatic rotation: `cswap auto`

- Foreground loop (`--interval`, min 15 s, default 60); `--once` for one tick with an exit code:
  0 switched, 1 error, 2 nothing to do, 3 blocked (no viable target). `--json` emits one event per
  line (`poll`, `switch`, `no-switch`, `all-exhausted`, `error`). `--dry-run` never switches.
- Settings and flags: `autoswitch.threshold` (default 90; the binding 5h or 7d window),
  `autoswitch.strategy` `best` (default) or `consume-first` (stay on the account whose weekly
  window resets soonest), `autoswitch.model` (per-model weekly windows that also trigger a switch,
  e.g. `Fable`), `cooldownSeconds` 300, `hysteresisPct` 10, `includeApiKeyAccounts` false.
- It switches the default login only. Sessions started with `cswap run` keep their account.

## Usage data: `cswap list --json`

- `schemaVersion` (1), `activeAccountNumber`, `accounts[]` with `number`, `email`, `active`,
  `alias`, `disabled`, `usageStatus` (`ok`, `unavailable`, `token_expired`, ...), `usage.fiveHour`
  and `usage.sevenDay` (`pct`, `resetsAt`), per-model weekly windows under `usage.scoped[]`
  (named by Anthropic display name, e.g. `Fable`), pace fields on weekly windows
  (`expectedPct`, `aheadOfPace`, `projectedExhaustionAt`, `willLastToReset`), freshness
  (`usageFetchedAt`, `usageAgeSeconds`, `lastGoodUsage`, `usageError`, `usageRetryAt`),
  `loginExpiresAt`. The README names the scoped entries without their fields (the internal cache
  has `name`, `pct`, `resets_at`); the "ignore unknown fields" promise is written for
  `auto --json`; `list` exit codes are undocumented beyond non-zero with an error object. The
  README omits three row fields 0.27.2 emits (`json_output.account_row`): `organizationUuid`,
  `organizationName`, `isOrganization`; no `accountUuid` is emitted, so email plus organization is
  the join to a session. A row not fetched within five minutes (`STALE_OK_S`) is reported as
  `usageStatus` `unavailable` with `usage` null and the last reading under `lastGoodUsage` with
  `lastGoodAgeSeconds`; `list` does not always refresh such a row on the spot.
- `list` refreshes entries older than its serve TTL and otherwise re-serves cached readings, with
  backoff after a 429. Readings can be hours old; read `usageAgeSeconds`.
- Anthropic's usage endpoint has a per-account request budget, observed at roughly thirty requests
  an hour; a status-line widget polling it every few seconds locked the account out for hours
  (claude-swap issue 220). Do not call cswap from anything that runs per render.
- `cswap list --json | ssh other cswap import-usage - --hold 600` shares readings across machines.

## Not available upstream (checked 2026-10-16)

A `cswap statusline` command (PR 176), an environment for shell hooks (PR 163), pre/post-swap
lifecycle commands (PR 436, closed unmerged) and per-session accounts with pacing (issue 382,
PRs 386 and 387) are all open or closed without a maintainer decision. The stable surfaces are
`list --json`, `auto --once` and its exit codes, `switch --json`, and `run`.
