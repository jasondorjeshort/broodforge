# Critic review of proposal 0020

Date: 2026-10-14. Reviewer: critic (Fable 5.1). Sources: the Claude Code and Agent SDK docs at
code.claude.com read today, npm and PyPI registry records, the SDK and engine repositories and their
issue trackers, the GNU GPL FAQ, and runs of the engine installed on the owner's PC (npm install,
`2.1.296`). Secondary press is cited only where marked.

## Verdict in one paragraph

The recommendation (bundle D, mechanism A first) fails its own reversal clause. C3 fails: the
TypeScript Agent SDK is not under an open-source license, and the proposal says "C3 fails ... then
B under D". C22 is worse than the fact check assumed: Anthropic's own Legal and compliance page
forbids exactly what R-39 describes (a developer collecting and storing users' Claude.ai session
tokens), while carving out a platform that hosts the unmodified Claude Code binary into which each
end user signs in with their own subscription through Anthropic's flow. On top of that, the one
feature A was recommended for, an interrupt that "works the same on Windows", has an open segfault
bug on Windows, and the proposal's own per-turn process model already discards everything else A
offers over B. The bundle (TypeScript, discord.js, npm, `node:test`) stands. The mechanism should
be B first, with a permission relay built on the engine's documented `defer` round trip rather than
on a process that waits for a human.

## Claims verified

| Claim | Finding | Source |
|---|---|---|
| C1 | SDK ships for Python and TypeScript only; other languages "run the CLI as a subprocess with the `-p` flag". Parity is not full: Python lacks `SessionStart`/`SessionEnd` callback hooks and several TypeScript-only events; Python `query()` with `can_use_tool` needs a documented "dummy hook keeps the stream open" workaround. The "most third-party front ends use it" sentence is unverifiable; drop it. | agent-sdk/overview, agent-sdk/hooks, agent-sdk/user-input |
| C2 | TypeScript SDK `0.3.300` bundles engine `2.1.300` through per-platform optional packages (`@anthropic-ai/claude-agent-sdk-win32-x64` carries `claude.exe`); `pathToClaudeCodeExecutable` defaults to "Auto-resolved from bundled native binary". Python SDK `0.2.167` bundles `2.1.300` inside 104 to 117 MB wheels; the Windows wheel was missing from PyPI for `0.2.160` to `0.2.163` (fixed `0.2.164`, 2026-10-06). | registry.npmjs.org, pypi.org, python SDK CHANGELOG, python issue #1349 |
| C3 | TypeScript SDK: npm `license: "SEE LICENSE IN README.md"`, no LICENSE file in the repository (GitHub reports none), README: "Use of this SDK is governed by Anthropic's Commercial Terms of Service". Engine (`@anthropic-ai/claude-code`): same npm license string; LICENSE.md is "(c) Anthropic PBC. All rights reserved. Use is subject to Anthropic's Commercial Terms of Service". Python SDK: LICENSE is MIT (Anthropic PBC, 2025), pyproject `license = MIT`, but its README carries the same Commercial Terms sentence and the wheel bundles the proprietary engine. GPL FAQ: pipes, sockets and command-line arguments "are communication mechanisms normally used between two separate programs" (MereAggregation); a GPL program that links a library outside the system-library exception needs a GPLv3 section 7 additional permission from the copyright holders (GPLIncompatibleLibs). Verdict: B is mere aggregation and clean; D+A imports a non-free library into a GPL-3 program; E+A with `cli_path` pointing at the host's engine is the one SDK combination whose code is GPL-compatible. | registry records, repo LICENSE files, gnu.org/licenses/gpl-faq.html |
| C4 | `canUseTool`/`can_use_tool` exist in both; allow returns `updatedInput`/`updatedPermissions`, deny returns `message` and optional `interrupt`. TypeScript works with a string prompt (the docs' own examples); the streaming-only constraint is on `interrupt()`, `setPermissionMode()`, `setModel()`. Never called for auto-approved calls; "can stay pending indefinitely"; "permission prompts don't time out". | agent-sdk/typescript, agent-sdk/user-input, agent-sdk/permissions |
| C5 | `interrupt()` exists, streaming mode only, carried as a control request on stdin; `interrupt_receipt_v1` capability. Windows: TypeScript SDK issue #444 (open since 2026-09-04, label bug): `Query.interrupt()` after init "segfaults claude.exe on Windows (0xC0000005)"; also #359 (open) intermittent "Stream closed" after the first tool call on Windows, #507 (open) 8.3 short paths break fork/resume lookup. Not verified working on Windows. | agent-sdk/typescript, GitHub issues |
| C6 | Per process: `--effort low..max` (also `ultracode`), SDK `effort` option; mid-session in TypeScript streaming mode via `applyFlagSettings({ effortLevel })`; `setPermissionMode`, `setModel` streaming only. Resolves "effort unknown". | cli-reference, agent-sdk/typescript, local `claude --help` |
| C7 | All listed flags exist. `--output-format stream-json` with `-p` requires `--verbose` (observed: `Error: When using --print, --output-format=stream-json requires --verbose`). `--permission-mode` values: `default` (alias `manual`), `acceptEdits`, `plan`, `auto`, `dontAsk`, `bypassPermissions`. Also relevant: `--permission-prompts none`, `--forward-subagent-text`, `--setting-sources`, `--strict-mcp-config`, `--no-session-persistence`, `--session-id`, `--bare` (never reads OAuth). | cli-reference, local run |
| C8 | Observed on the PC: `system/init` carries `session_id`, `model`, `permissionMode`, `tools`, `capabilities`; `assistant` carries `message.model` and `message.usage`; `result` carries `usage`, `modelUsage` (per model: tokens, cache, `costUSD`), `total_cost_usd`, `num_turns`, `is_error`, `permission_denials`. Docs: `usage` is main loop only; `modelUsage` and `total_cost_usd` include subagents and internal calls; per-message `output_tokens` "is a placeholder"; totals are cumulative across resumed sessions since 2.1.277. Subagent messages carry `parent_tool_use_id`; tool_use blocks arrive by default, text only with `--forward-subagent-text`. | local run, agent-sdk/cost-tracking, headless |
| C9 | Not verifiable. The permission tool's input and return shape is undocumented (claude-code issue #24595, closed 2026-03-11 as stale); HTTP acceptance for the permission tool is not stated. The docs do document an alternative: the `defer` round trip (objection 4). | cli-reference, headless, hooks, GitHub |
| C10 | Semi-documented: the TypeScript reference names `control_response` with `requestId`, the `initialize`/`reinitialize` control request, and an open `capabilities` set in `system/init` for feature detection (observed: `interrupt_receipt_v1`, `interrupt_cancel_queued_v1`, `interrupt_send_now_v1`, `msg_lifecycle_v1`, ...). Many behaviours are version-gated ("requires v2.1.2xx"). A and B are exposed to this churn equally; A is B wrapped. | agent-sdk/typescript, local run |
| C11 | Transcripts: `$CLAUDE_CONFIG_DIR/projects/<encoded-cwd>/<id>.jsonl`; since 2.1.223 `--resume <id>` finds the session from any cwd on the machine; a SIGTERM-killed turn is left unfinished and resumable (`CLAUDE_CODE_RESUME_INTERRUPTED_TURN=1` to continue it); the retention sweep deletes transcripts after `cleanupPeriodDays` (30 days default). | agent-sdk/sessions, headless, hooks |
| C12 | TypeScript `env` replaces the subprocess environment ("pass `{ ...process.env, ... }`"); Python `env` merges. Per-query credential is possible in both. | agent-sdk/typescript, agent-sdk/python |
| C13 | Hooks in process under A, as settings-file command hooks under B. `SubagentStart`/`SubagentStop` carry `agent_id`, `agent_type`, `agent_transcript_path`; no model field. Model names come from the stream (`assistant.message.model` with `parent_tool_use_id`) or from the keys of `result.modelUsage`. | agent-sdk/hooks, hooks |
| C14 | Native installer: `%USERPROFILE%\.local\bin\claude.exe`. npm: a `.cmd` shim over `node_modules/@anthropic-ai/claude-code/bin/claude.exe` (observed on the PC); npm needs Node 22 only to install, the binary is native. Git for Windows optional; without it the PowerShell tool is used; `CLAUDE_CODE_GIT_BASH_PATH` honoured. Python SDK refuses `.cmd`/`.bat` CLI paths; Node refuses to spawn `.cmd` without a shell. Sandboxing on native Windows: "Not supported". | setup, python SDK `subprocess_cli.py` |
| C15 | Measured on the PC (owner's full `~/.claude` loaded, Haiku, one turn): 4.1 to 4.8 s wall with stdin closed (API 1.4 to 1.5 s); 6.7 to 7.5 s with stdin left open (the engine waits 3 s for stdin); `--output-format json` run 2.9 s; `claude --version` 0.37 s. Overhead per turn about 2.5 to 3.5 s on Windows. Linux unmeasured. | local runs |
| C16 | SIGTERM: exit 143, turn left unfinished, no `result`, Bash trees killed, `SessionEnd` hooks run, session resumable. SIGINT or `interrupt()` ends the turn with a result. | headless |
| C17 | discord.js `14.27.0`, Apache-2.0, pushed 2026-10-14, Node >= 18. Feature list is standard v14 API; not individually re-verified. | npm, GitHub |
| C18 | Oceanic `1.15.0`, MIT, pushed 2026-10-12. | npm, GitHub |
| C19 | discord.py `2.7.1`, MIT, pushed 2026-09-07; hikari `2.6.0`, MIT, pushed 2026-10-13. | PyPI, GitHub |
| C20 | discordgo BSD-3-Clause, last release `v0.29.0` 2025-05-24, last push 2026-02-14 (slow); disgo Apache-2.0, pushed 2026-10-09. | GitHub |
| C21 | Node 24 is Active LTS (since 2025-10-28); Node 26 becomes LTS 2026-10-28; the PC runs 22.23.2 (Maintenance to 2027-04-30). `node:test` is "Stability: 2 - Stable" with `describe`, `before`, `mock`, `--test-concurrency`. The SDK has peerDependencies (`zod ^4`, `@anthropic-ai/sdk >=0.93`, `@modelcontextprotocol/sdk ^1.29`) a bot must list. | endoflife.date, nodejs.org, npm |
| C22 | See objection 2. | code.claude.com/docs/en/legal-and-compliance |
| C23 | MCP SDKs: TypeScript `1.32.1` (npm MIT; repository moving to Apache-2.0), Python `mcp 2.3.0` MIT, Go `v1.8.0` (MIT moving to Apache-2.0); all GPL-3 compatible; all document a Streamable HTTP server transport. | registries, repo LICENSE and README files |
| C24 | Judgement; I agree with D < E < F for time to first version. Nothing public changes it. | none |

## Objections

1. **Blocking. C3 fails for D+A; the proposal's own clause reverses the mechanism.** The
   TypeScript SDK is a proprietary-terms library imported into a GPL-3 process; N-24 says every
   dependency is GPL-compatible and the proposal says "C3 fails ... then B under D". The engine is
   proprietary under both mechanisms, but under B it is a separate program driven over pipes and
   not distributed by Broodforge, which the GPL FAQ treats as separate programs. Resolves by one
   of: (a) B under D, as the proposal already says; (b) keep A and record a GPLv3 section 7
   additional permission for the Agent SDK in ADR 0001 (the owner is the sole copyright holder, so
   it is possible, but it is a licensing-policy change and the owner's call, not the
   coordinator's); (c) E+A with `cli_path` to the host's engine, the only SDK route whose code is
   MIT. The proposal's sentence "the bundle stays D under B" is right; its claim that C3 can
   reverse "only the mechanism" is wrong, since C3 also favours E over D under A.

2. **Blocking (owner decision, Q1). C22: the design of R-39 is forbidden as written; the
   permitted shape exists and is narrower.** Primary text, Legal and compliance, Authentication
   and credential use: "Anthropic does not permit third-party developers to offer Claude.ai login
   into their own applications, or to route requests through Free, Pro, or Max plan credentials on
   behalf of their users. Moreover, developers may not collect, store, or intermediate Claude.ai
   credentials or session tokens, sign-in to a Claude account must complete through Anthropic's
   own flow." And: "Nor does it prevent an end user from signing in to the unmodified Claude Code
   binary with their own Claude subscription, including where a platform hosts Claude Code",
   under the conditions in "Can customers offer Claude Code in their products?": unmodified
   binary, no authentication method removed, no paying for or intermediating usage, each end user
   authenticates with their own credentials, Commercial Terms accepted. Also: "Advertised usage
   limits for Pro and Max plans assume ordinary, individual usage of Claude Code and the Agent
   SDK", and "Anthropic reserves the right to take measures to enforce these restrictions and may
   do so without prior notice." Secondary press (Gigazine, AlternativeTo, falcao.org, Feb to Apr
   2026) reports server-side enforcement against subscription tokens in third-party harnesses.
   Consequences the proposal must carry: a `setup-token` pasted into Discord and stored by
   Broodforge is a stored session token, prohibited; the per-user `CLAUDE_CONFIG_DIR` into which
   the user completes `claude auth login` themselves is the only subscription shape that resembles
   the carve-out, and the owner's single-user PC instance on the owner's own plan is ordinary
   individual use. The proposal's line "Q1 answered API keys only: ... the config-dir delivery in
   0022 loses its main reason" is backwards: the terms make config-dir the only subscription
   route and the token route an owner-only CI convenience. The carve-out is phrased around a user
   signing in to the unmodified binary; the SDK overview says agents built on the SDK should use
   API keys. B with the host's own installed binary sits closer to the permitted shape than A.
   Resolves by: the owner reads the quoted text and answers Q1; the proposal records which
   credential shapes are offered to whom; 0022 designs only those.

3. **Serious. C5 is contradicted on Windows.** The recommendation leans on "interrupt without
   signals ... works the same on Windows and Linux". TypeScript SDK #444 reports `claude.exe`
   dying with an access violation on `interrupt()` on Windows 11 (open, no fix noted); #359 reports
   stream closure after the first tool call on Windows. The interrupt control request is the same
   engine code path under B's stream-json stdin, so this is an engine bug, not an SDK bug: on
   Windows the only interrupt known to work is terminating the process tree and resuming. Resolves
   by: the spike tests a mid-turn interrupt on Windows before the ADR; the design treats kill and
   resume as the Windows path and `interrupt()`/SIGINT as a Linux fast path; the ADR cites the
   issue numbers.

4. **Serious. The permission relay makes an engine process wait for a human; the docs say not
   to.** Discord answers arrive in minutes or hours. Under A the callback "can stay pending
   indefinitely", a bot restart (R-17) kills the waiting engine and the prompt with it, and #384
   shows the control channel closing while background subagents still need answers. The docs
   recommend, for a host whose user "might take longer to respond than your process can reasonably
   stay running", a `PreToolUse` hook returning `defer`: the process exits with `stop_reason:
   "tool_deferred"` and `deferred_tool_use` (id, name, input) in the result; the host later runs
   `--resume <session-id>` with the same hooks, the same tool call fires `PreToolUse` again, and
   the hook returns `allow` with the answer in `updatedInput` (or `deny`). "There is no timeout or
   retry limit"; the session stays on disk subject to `cleanupPeriodDays`. This is exactly the
   per-turn process model, it is crash-safe, and it is mechanism-neutral (SDK callback hook under
   A, command hook under B). Constraints to state: `defer` works only when the turn has a single
   pending tool call (parallel calls fall through to the normal flow and, in `-p` with no host,
   are denied); it is honoured only in `-p` mode. Resolves by: adopt `defer` as the relay for
   human answers and hooks or rules for policy answers; say what parallel tool calls get (a deny
   that tells the model to retry one at a time); then B needs no MCP permission server and A needs
   no live callback across a human wait.

5. **Serious. The usage rule in "Choices made where the docs were silent" cannot record what
   N-21 asks for.** Per-message `output_tokens` "is a placeholder"; the real output count is on
   the `result`; `result.usage` excludes subagents; `modelUsage` and `total_cost_usd` include
   them but are cumulative across resumed sessions since 2.1.277, so per-turn figures are
   differences between consecutive results. A killed turn has no result: only main-loop input and
   cache tokens are recoverable. Resolves by: the usage record is the per-model delta of
   `modelUsage` when a result arrives; otherwise a record flagged interrupted with deduplicated
   per-message input and cache counts and unknown output; prefer an ending interrupt over a kill
   wherever one works (objection 3).

6. **Serious. Repository content runs on the host before any permission check, and
   subscription sessions cannot use the mode that prevents it.** Headless docs: "Without `--bare`,
   a `-p` session runs the hooks in a project's `.claude/settings.json` and connects the servers
   in its `.mcp.json`, even in a folder you've never trusted", with no trust dialog; the SDK's
   default `settingSources` includes `project`. `--bare` "never reads OAuth credentials", so a
   subscription turn cannot use it, and the docs say `--bare` "will become the default for `-p` in
   a future release". On the droplet, a contributor's pull request adding a settings hook is code
   execution under the session's credential at the next turn, before reach enforcement sees a
   tool call. Resolves by an adapter rule in this proposal, handed to 0022 as a hard requirement:
   `--setting-sources user` (or empty) and `--strict-mcp-config` with an explicit `--mcp-config`
   on every turn, under either mechanism; never rely on the engine's default mode set.

7. **Serious. "Auto mode" names two different things and the mapping is undefined.** Claude Code
   has a permission mode literally called `auto`: a classifier reviews actions, denies on no
   verdict, has model and provider availability rules, and is the built-in starting mode for `-p`
   on versions that do not fetch feature flags (2.1.285+). Broodforge's auto mode (R-26, R-27) is
   the bot answering within the channel's reach. Also: `bypassPermissions` "cannot be used with
   root/sudo privileges" on Linux outside a recognized sandbox, and is inherited by subagents;
   `dontAsk` never calls the host. Resolves by: always pass `--permission-mode` explicitly;
   define Broodforge auto as `default` (or `acceptEdits` for code reach) plus a `PreToolUse`
   hook that applies the reach policy, never `bypassPermissions`; Broodforge prompt mode as the
   same hook returning `defer` for anything the policy does not settle. Record the engine's `auto`
   as a third, later option.

8. **Minor. C4 is misstated.** The callback is not "reachable only in the streaming-input form";
   in TypeScript it works with a string prompt (the docs' own examples). The streaming constraint
   is on `interrupt()` and the setters. Python's `query()` needs the documented dummy-hook
   workaround. Fix the text.

9. **Minor. C9 cannot be verified and should not carry B.** The MCP permission tool's schema is
   undocumented (issue #24595 closed stale), and HTTP acceptance for that tool is unstated. With
   objection 4 the MCP route is unnecessary; drop it or verify it by spike before naming it.

10. **Minor. C15 and the stdin wait are an adapter rule, not a cost.** The 3 s stdin wait doubled
    the measured turn overhead; the adapter must close stdin or use `--input-format stream-json`.
    `--verbose` is required with `stream-json`. State both; cite the measured numbers in the ADR
    and add the droplet's once measured.

11. **Minor. C14: "the engine path is configuration" must name the `.exe`.** The npm shim is a
    `.cmd`; Node will not spawn it without a shell and the Python SDK refuses it outright. Under A
    the SDK ships its own second engine (2.1.300 beside the owner's 2.1.296), so bug reports must
    name which binary ran; under B there is one engine per host, pinned by the owner.

12. **Minor. C13 and R-32: the simplest attribution source is `result.modelUsage` keys.** It
    lists every model that made a request in the turn, subagents included; it also includes
    internal calls (compaction, the auto-mode classifier), so `03-architecture.md` must say
    whether a model that only compacted is a co-author. Hooks add nothing here.

13. **Minor. C11 and N-20: engine state lives in the credential's config dir.** Transcripts,
    `.claude.json` and the retention sweep are under `$CLAUDE_CONFIG_DIR`; R-18 resume depends on
    them. The config-dir rule therefore binds a session's resumability to a credential's
    directory and requires that directory to sit inside the backup root (0021) and inside
    whatever 0022 isolates. Say so.

14. **Minor. C20 and C21 housekeeping.** discordgo is slow (last release May 2025); if F is ever
    revisited, disgo is the primary. The Agent SDK's peerDependencies must be listed by the bot.
    The PC's Node 22 is Maintenance LTS; pin 24.

## Refusals, denials and fail-closed paths, and who lifts them

| Mechanism | Who lifts it, from where, mid-session | Message says so? |
|---|---|---|
| Policy deny inside reach (R-27) via hook or callback | The driver by switching mode (R-28) from Discord; the owner from the debug CLI. Per-turn processes make the next turn pick up the change. | Only if the deny `message` names it; the proposal does not define the text. Objection: define it. |
| `defer` with nobody answering | Nobody automatically: "no timeout or retry limit"; the session waits on disk until the retention sweep deletes it. Broodforge must own a timeout (idle timeout R-15) and the resume. | No engine message. Objection 4 covers it. |
| `dontAsk` / `--permission-prompts none` | Only by starting the next turn in another mode; cannot be lifted inside a turn. | The model is told nobody can approve; the person is not. |
| `bypassPermissions` as root on Linux | Run as a non-root user or inside a recognized sandbox; an engine start error. | Yes: "cannot be used with root/sudo privileges". |
| Engine `auto` mode with no classifier verdict | Nobody per action; the turn stops after ten such denials. | Yes, engine notice. Reason to avoid it as Broodforge's auto. |
| `--bare` not reading OAuth | Do not pass `--bare` for subscription credentials. | Yes, in docs; at runtime an auth failure. |
| Hook timeout (600 s default on `PreToolUse`) | Set `timeout` on the matcher. | The model gets a timeout tool result. |
| Engine path unset, config dir unset (the proposal's two rules) | The owner edits configuration; refuses at instance start. | Not defined. Objection: the refusal must name the key. |
| Conditions that trigger building B | The coordinator. | n/a |

## Hidden coupling into 0022 and 0021

- One process per turn means the credential is handed over per process, as `env` or as a config
  dir. Either is readable from inside the session: the Bash tool inherits the engine's
  environment, and `.credentials.json` is readable by the engine's uid. Neither mechanism hides
  it; on native Windows the engine's sandbox is unsupported. 0022 must say this (N-15) and cannot
  rely on the mechanism choice to fix it.
- The terms (objection 2) decide the delivery option before 0022 starts: config dir with the
  user's own login, not stored tokens.
- A does not block 0022's isolation options: TypeScript offers `spawnClaudeCodeProcess` (custom
  spawn with `command`, `args`, `cwd`, `env`, `signal`) and Python a custom `Transport`, so a
  wrapper such as `systemd-run`, `bwrap` or a different user is possible under A. It is an extra
  integration surface; under B it is the spawn call itself.
- `bypassPermissions` refuses root on the droplet (objection 7); the engine must run as a
  non-root user for any bypass-based mode.
- Project settings and `.mcp.json` load by default (objection 6); 0022 inherits the
  `--setting-sources` rule.
- Engine transcripts live in the credential's config dir (objection 13); 0021's backup root and
  0022's isolation must both contain it.

## The strongest case for B first, and for a non-TypeScript bundle

B first, under D. The license is clean without an exception (1). The terms carve-out is written
around a user signing in to the unmodified binary the host installed (2). On Windows the interrupt
is a process kill either way (3). The per-turn process model the proposal chose already gives up
`interrupt()`, the setters and message injection, which are the only things A offers that B does
not; what remains of A is typed messages and a bundled second engine. The relay that fits Discord
is `defer` plus resume (4), which needs no MCP server and no callback that outlives a human wait.
Agent-drivability favours B: a bug is reproduced by one `claude -p --resume ... --output-format
stream-json` line in a terminal, the recorded JSONL is the fake harness fixture (N-08, N-09), and
a Haiku agent can read both. The cost is real but small: process management on two hosts (needed
for worktrees and git anyway), a JSON line parser, and a type file written from the documented
shapes. Startup overhead is the same per turn under A's per-turn `query()`.

A non-TypeScript bundle. E+A is the only SDK route whose code is GPL-compatible, and it keeps
`sqlite3` in the standard library for 0021; against it stand the Python SDK's Windows history
(hangs, `.cmd` refusal, command-line length, a missing Windows wheel this month), the
`query()` workaround, and no stated owner preference. Under B the language is free, and D keeps
the shortest path with the largest example corpus. F's single binary matters only on the droplet.
I do not recommend changing the bundle.

## Recommendation

Bundle D with mechanism B first: TypeScript on Node 24, discord.js, npm and `node:test`; the
engine the owner installs on each host, driven as one `claude -p --output-format stream-json
--verbose --input-format stream-json` process per turn with stdin closed after the message,
`--setting-sources user`, `--strict-mcp-config`, an explicit `--permission-mode`, and a
`PreToolUse` command hook that applies the reach policy and returns `defer` for anything a person
must answer; interrupt by process-tree kill and resume on Windows, SIGINT on Linux; usage from
`modelUsage` deltas. Build the adapter interface and the recorded-stream fake harness first, as
the proposal says. Keep A as the second implementation, built only if the owner grants a GPL
section 7 exception for the SDK and the spike shows `interrupt()` stable on Windows. Confidence:
high on the C3 and C22 facts, which are quoted from primary pages; moderately high on the
mechanism, since the residual risk, the stream protocol shifting between engine releases, is
shared by both mechanisms and mitigated by the `capabilities` field and a pinned engine version.
