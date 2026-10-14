# 0020. Harness: Claude Code CLI per turn, TypeScript on Node, discord.js, npm

Date: 2026-10-14
Status: accepted

Record: proposal `../proposals/0020-harness-language-library-toolchain.md` and
`../proposals/0020-critic-review.md` (its verification table is authoritative on engine facts).

## Context

Every other design doc is written in this language and against these harness facts
(`../06-platforms.md`, Order). The primary harness must be the real Claude Code program so a
subscription can pay (R-38); both hosts ship in the first version from one code base (N-01); every
dependency is GPL-compatible (N-24); the owner wants a working first version within days.

Verified by the critic (sources in the review): the TypeScript Agent SDK and the engine are under
Anthropic's Commercial Terms, not open source; the Python SDK's code is MIT but its wheel bundles
the proprietary engine; driving the engine over pipes is mere aggregation under the GPL FAQ. The
engine's `interrupt()` segfaults `claude.exe` on Windows (TypeScript SDK issue #444, open). The
`PreToolUse` hook may return `defer` in `-p` mode when a turn has a single pending tool call; the
process exits with `stop_reason: "tool_deferred"` and `--resume` re-fires the hook. `result`
carries `modelUsage` per model, cumulative across resumes since engine `2.1.277`; per-message
`output_tokens` is a placeholder. Without `--bare`, a `-p` session runs the worktree's
`.claude/settings.json` hooks and `.mcp.json` servers; `--bare` never reads OAuth credentials.
Anthropic's Legal and compliance page forbids collecting, storing or intermediating claude.ai
credentials or session tokens and permits an end user signing in to the unmodified binary with
their own subscription, including where a platform hosts it.

The owner decided (2026-10-14, in the session; this ADR is the record): subscriptions stay in the first version as
a per-user `CLAUDE_CONFIG_DIR` the user logs into themselves; no stored tokens of any kind; the
droplet's direct-message login relay is off by default; API keys are the fallback.

Rejected, reasons in the proposal: the Agent SDK in process first (license needs a GPLv3 section
7 exception only the owner can grant; nothing it adds survives the per-turn model; Windows
interrupt broken); Python (SDK Windows record); Go (longest to first version); a long-lived engine
process per session; any pasted or stored token (the terms); the MCP permission-prompt tool
(undocumented schema); the engine's `auto`, `dontAsk` and `bypassPermissions` modes.

## Decision

Mechanism. The Claude Code CLI the owner installs and pins on each host, run as one process per
turn in the session's working tree: `claude -p --output-format stream-json --verbose
--input-format stream-json --permission-mode default --setting-sources user --strict-mcp-config
--mcp-config <file> --resume <engine session id>` (no `--resume` on the first turn), with the
user message written to stdin and stdin closed at once. The engine path is configuration naming
the binary itself (`claude.exe` or the ELF), never a `PATH` lookup or the npm `.cmd` shim; unset
or missing, instance start is refused with the configuration key named, exit 1. Behind the
adapter interface of `../03-architecture.md` [no-check: not yet written], with a fake harness that
replays a recorded `stream-json` transcript. The Agent SDK is a second implementation designed in
the interface and built only when both hold: the owner records a section 7 additional permission
for the SDK in ADR 0001, and a spike shows `interrupt()` stable on Windows.

Language bundle. TypeScript compiled with `tsc` on Node 24 (pinned in `package.json` `engines` and
`.nvmrc`); discord.js 14.x (Apache-2.0); `better-sqlite3` (MIT) as ADR 0021's binding; `npm ci`,
`npm run build`, `npm test` with `node:test`, `npm start -- <args>`; the debug CLI is the same
entry point. Windows needs Git for Windows for the engine's Bash tool
(`CLAUDE_CODE_GIT_BASH_PATH`). No test framework dependency until one is missed.

Permission relay and modes. Broodforge's R-26 mode is named policy mode; its counterpart is prompt
mode; the engine's `auto` is never either. One `PreToolUse` command hook, Broodforge's own CLI
(`broodforge hook pre-tool-use`, a `timeout` set on its matcher), registered in the
Broodforge-written `settings.json` of the session's `CLAUDE_CONFIG_DIR`, decides every tool call
from the session id, mode, reach and pending answer in the process environment. Outside reach:
`deny` in both modes with `Denied by Broodforge: <tool> on <target> is outside this channel's
reach (<reach name>); nobody in this session can allow it; the channel's configuration sets
reach`. Inside reach: a read-class tool (Read, Glob, Grep and the engine's other read-only tools)
is `allow` in both modes, as the engine's own default mode does; otherwise policy mode `allow`,
and prompt mode `defer` for write tools and Bash, Broodforge posts buttons, and the click resumes
the session with the answer for that `tool_use_id` in the environment. The hook
writes nothing to the store. Parallel tool calls in prompt mode fall through to the engine's deny;
the adapter posts a one-line notice and the system prompt append asks for one call at a time.
Nobody answering is the channel's idle timeout (R-15). Mode, model and effort are per-process
flags, so switching takes effect at the next turn.

Interrupt. Windows: terminate the process tree by PID, then resume. Linux: SIGINT to the engine
(ends the turn with a `result`); SIGTERM or a kill only as the fallback.

Usage and attribution. The per-turn usage record is the per-model delta of `result.modelUsage`
against the previous result of the same engine session, `result.usage` kept as a secondary
field. A turn with no `result` is recorded as interrupted with per-message input and cache counts
deduplicated by message id and output unknown. The record is committed before the result is
posted (ADR 0021). Models that worked are the keys of `modelUsage`; for an interrupted turn, the
distinct `assistant.message.model` values seen. Whether a compaction-only model is a co-author
is `03-architecture.md`'s.

Credentials (delivery is 0022's; the shapes are fixed here). API key: `ANTHROPIC_API_KEY` in the
engine process environment only. Subscription: one `CLAUDE_CONFIG_DIR` per registered user per
host, created and owned by Broodforge, holding the Broodforge-written `settings.json` and whatever
the engine writes; the user completes Claude Code's own login into it (on the PC by running it
with the directory set; on the droplet through the relay, which runs the engine's login command,
sends the URL by direct message, and writes the pasted code to the login process's stdin). The
relay is off by default; enabling it is an owner configuration change with an audit event.
Broodforge never reads, stores, logs or relays a token. Every engine process gets an explicit
`CLAUDE_CONFIG_DIR` or `ANTHROPIC_API_KEY`; the owner's `~/.claude` is never used. The directory
sits inside the backup root and the isolation boundary (ADR 0021, harness-owned state).

Settings sources. Every process: `--setting-sources user`, `--strict-mcp-config`, an explicit
`--mcp-config`, `--permission-mode default`. Repository hooks and `.mcp.json` never load. `--bare`
is not used (subscription turns cannot; one command line for both credential kinds).

A message written during a running turn is queued for the next turn. Handed to 0022 as requirements: the credential shapes above and the relay switch; honesty per host
about what a session's Bash tool reads (the environment and `.credentials.json` are readable by
the engine's user; the engine's sandbox is unsupported on native Windows); the settings-sources
rule as the first line of reach; non-root bot and engine on the droplet; the Windows tree kill;
the isolation wrapper is the spawn call (`sudo`, `bwrap`, `systemd-run`, another user).

Conditions the first implementation must check (living table; the coordinator records each
result here when the spike's finding lands, the one amendment this ADR allows):

| Claim | What the spike shows on both hosts | Result |
|---|---|---|
| C25 | The engine's login command, without a TTY and with `CLAUDE_CONFIG_DIR` set, prints the URL and accepts the pasted code on stdin; what it writes; `claude auth login` or `/login` | open |
| C26 | `--settings <file>` is honoured with `--setting-sources user`; whether an empty `--setting-sources` still runs hooks given by `--settings` | open |
| C27 | The `defer` round trip across a wait: clean exit, `--resume` re-fires `PreToolUse` for the same `tool_use_id`, `updatedInput` or deny honoured | open |
| C28 | A parallel-tool-call turn in prompt mode: the deny text the model sees; whether `permission_denials` names the calls | open |
| C29 | The droplet's per-turn overhead in seconds, measured as C15 (PC: 2.5 to 3.5 s) | open |
| C39 | Pinned engine, `-p --permission-mode default`: a `PreToolUse` hook exiting 2 blocks the call and the `tool_result` carries the hook's stderr verbatim; a hook exiting 1 and a hook past its `timeout` do not block; what the stream shows for each; what the engine's default then does with a `Read`, a `Write` and a `Bash` call when no hook answers in `-p` mode | open |

The spike: the adapter interface and the fake harness first, then one turn through the real
engine on each host with a deferred permission answered on resume, an interrupt mid-turn followed
by a resume, a parallel-call turn in prompt mode, printing the `modelUsage` delta and model names.
Its output is a finding (`finding` skill) whose `promoted_to` is this file. An implementation that
finds C27 failing stops and reports; the fallback relays in the proposal are spike-gated.

Out of scope: isolation, credential storage, worktree placement (0022); the adapter interface,
config format and the hook's exact environment names (03); the debug transport (05); the words
users see for the modes (04 and the owner). Follow-up: `01-requirements.md` R-26 to R-28 say
"policy mode".

Deferred, with triggers: the Agent SDK (the two conditions above); mid-turn message injection
(the UX doc asks and the SDK is built); the engine's `auto` as a third mode (a channel wants
classifier-reviewed actions); a `--permission-prompt-tool` server (C27 fails and the SDK is
unavailable); `--bare` for API-key turns (C26 shows `--settings` honoured under it).

## Consequences

Easier: a bug is one `claude -p --resume ...` line and a recorded JSONL is the fixture (N-08,
N-09); no license exception; one runtime for bot, CLI and tests; the largest example corpus for
agents; a bot crash loses no engine process; resume is exercised from the first turn.

Harder or costly: about 3 s of engine start per turn on the PC, unmeasured on the droplet; the
stream's shapes change between releases, so both hosts pin the engine and the adapter reads
`system/init` `capabilities`; a hand-written type file; a Windows interrupt is a kill with output
tokens unknown; prompt mode serialises parallel tool calls; a human wait leaves a transcript on
disk until the engine's retention sweep.

Must be maintained: the engine pin per host and the capability checks; the transcript fixtures
when the pin moves; the conditions table; the deny text wherever 04 rewords what users see.

Supersede when: the terms change what a hosted subscription login may look like; the engine stops
honouring `defer` or `--setting-sources`; the owner grants the SDK exception and asks for a
long-lived session; a second harness makes the per-turn model the wrong default.

## Amendments

2026-10-14: prompt mode allows read-class tools and defers write tools and Bash; a button per Read would make prompt mode unusable and the engine's own default mode draws the same line; recorded from 03-architecture.

2026-10-14 (review of 03): the read class is `Read`, `Glob`, `Grep`, `LS` and the engine's other file-reading tools, allowed only on paths that resolve under the session tree or the canonical checkout, with `.credentials.json` and the config directory denied everywhere; `WebFetch` and `WebSearch` are not read class: prompt mode defers them, policy mode denies them until a reach allows the network. The hook catches every error and exits 2, because only exit 2 blocks; C39 added for the engine side of that contract. Recorded in `../03-hook-contract.md`.
