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
| C25 | The engine's login command, without a TTY and with `CLAUDE_CONFIG_DIR` set, prints the URL and accepts the pasted code on stdin; what it writes; `claude auth login` or `/login` | holds on the PC, piped form, 2026-10-15 and again 2026-10-16: `claude auth login` prints the URL, takes the code on stdin, exits 0, writes `.claude.json`, `.credentials.json`, `backups/`; droplet open |
| C26 | `--settings <file>` is honoured with `--setting-sources user`; whether an empty `--setting-sources` still runs hooks given by `--settings` | holds, 2026-10-16: the `--settings` hook ran beside `--setting-sources user` and beside `--setting-sources ""`; the inline form is closed (Amendments) |
| C27 | The `defer` round trip across a wait: clean exit, `--resume` re-fires `PreToolUse` for the same `tool_use_id`, `updatedInput` or deny honoured | holds on the PC, 2026-10-16: exit 0 with `tool_deferred`, the resume takes no stdin and re-fires the hook for the same id, allow writes the file, deny reaches the model wrapped (Amendments, C39); `updatedInput` untested by design; droplet open |
| C28 | A parallel-tool-call turn in prompt mode: the deny text the model sees; whether `permission_denials` names the calls | holds on the plan's terms, 2026-10-16 (no hang, no crash), but no engine deny: both calls deferred, `result` names the last, the first dropped and re-issued after the resume, `permission_denials` empty (Amendments) |
| C29 | The droplet's per-turn overhead in seconds, measured as C15 (PC: 2.5 to 3.5 s) | open |
| C39 | Pinned engine, `-p --permission-mode default`: a `PreToolUse` hook exiting 2 blocks the call and the `tool_result` carries the hook's stderr verbatim; a hook exiting 1 and a hook past its `timeout` do not block; what the stream shows for each; what the engine's default then does with a `Read`, a `Write` and a `Bash` call when no hook answers in `-p` mode | holds on the PC, 2026-10-16: exit 2 blocks, stderr wrapped with the hook command; exit 1 and the timeout fall through to the engine's default, which allowed the Read, denied the Write without a prompt and ran the safe-listed Bash (Amendments); droplet exit-2 repeat open |

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

`-p` disables AskUserQuestion and plan approval, reported for channels in non-interactive mode (the
`/channels` page, in the 2026-10-16 prior-art finding; S1 checks plain `-p`), so an agent that needs a structured answer asks in prose and ends the turn; the spike
records what the stream carries when the tool is called (S1, `../07-spike-plan.md`).

Must be maintained: the engine pin per host and the capability checks; the transcript fixtures
when the pin moves; the conditions table; the deny text wherever 04 rewords what users see.

Supersede when: the terms change what a hosted subscription login may look like; the engine stops
honouring `defer` or `--setting-sources`; the owner grants the SDK exception and asks for a
long-lived session; a second harness makes the per-turn model the wrong default.

## Amendments

2026-10-14: prompt mode allows read-class tools and defers write tools and Bash; a button per Read would make prompt mode unusable and the engine's own default mode draws the same line; recorded from 03-architecture.

2026-10-14 (review of 03): the read class is `Read`, `Glob`, `Grep`, `LS` and the engine's other file-reading tools, allowed only on paths that resolve under the session tree or the canonical checkout, with `.credentials.json` and the config directory denied everywhere; `WebFetch` and `WebSearch` are not read class: prompt mode defers them, policy mode denies them until a reach allows the network. The hook catches every error and exits 2, because only exit 2 blocks; C39 added for the engine side of that contract. Recorded in `../03-hook-contract.md`.

2026-10-14 (review of 184412c): models used by subagents appear in `modelUsage`, verified by the critic (claim C8, `proposals/0020-critic-review.md`), so the trailer rule covers them.

2026-10-14 (review of 043382f): engine 2.1.296 defaults `--permission-prompts` to `host`, so a call that no hook answers in `-p` with stdin closed may emit a `control_request` and wait for an answer that never comes; observed by the reviewer in help text; the spike confirms (C39, C28 in `../07-spike-plan.md`). If it does, `--permission-prompts none` joins the flag list in Decision, Mechanism, on every process.

2026-10-14 (review of 043382f): `--bare` skips the hooks in settings files on engine 2.1.296, observed by the reviewer in help text, so the deferral of `--bare` for API-key turns is withdrawn and C26 no longer gates it; `--bare` is never used, as Settings sources already says.

2026-10-14 (07-spike-plan): the spike's throwaway runner precedes the adapter interface and the fake harness named in the spike paragraph above, not the other way round; its recorded fixtures are the fake's first input, and `engine run --real` is the runner's production form (`../07-spike-plan.md`).

2026-10-16 (prior-art finding, `../../findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md`, "Terms, re-checked at the source"): the terms were re-read at the primary pages. The Claude Code legal page says a platform may host the unmodified binary when "each end user must authenticate with their own ... Claude subscription plan credentials", and that developers "may not collect, store, or intermediate Claude.ai credentials or session tokens". It also says, verbatim: "Anthropic does not permit third-party developers to offer Claude.ai login into their own applications, or to route requests through Free, Pro, or Max plan credentials on behalf of their users" and "sign-in to a Claude account must complete through Anthropic's own flow", and limits that with "Nor does it prevent an end user from signing in to the unmodified Claude Code binary ... where a platform hosts Claude Code". This design (unmodified binary, one login directory per person, no token read) fits the hosting sentence; a pasted-token or raw-API design does not. Open question for the owner, decided nowhere here: whether the droplet's DM login relay (Decision, credentials), where the bot runs the engine's login command and passes the sign-in link and the pasted code, counts as completing sign-in "through Anthropic's own flow"; the relay is off by default. The support article "Use the Claude Agent SDK with your Claude plan", updated 2026-10-07, says `claude -p`, the Agent SDK and third-party apps remain on subscription limits, that the announced change was paused on 2026-06-15, and that Max and Team plans carry monthly API credits covering `claude -p`. This answers the concern in the Context that headless use might be metered separately; it is a statement as of its date and the supersede triggers above still apply.

2026-10-16 (R-53, Owner 16 in `../04-ux.md`): the sentence "Nobody answering is the channel's idle timeout (R-15)" in Decision is replaced. An unanswered permission prompt is denied after the channel's prompt timeout (default 5 minutes), the turn finishes with its output kept, and the session stays open; the idle timeout of R-15 is a separate timer that still ends the session.

2026-10-16 (spike, `../../findings/2026-10-15-spike-login-claims-for-adr-0020.md`, `../../findings/2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`, `../../findings/2026-10-15-spike-hook-flow-claims-for-adr-0020.md`, every PC row, engine 2.1.296 on Node 22): the table above carries the PC verdicts; C29 and the droplet repeats of C25, C27 and C39 stay open. No PC run emitted a `control_request`, so the `--permission-prompts none` of the 043382f amendment is not added to the command line; it is the first flag to try if a later engine waits. C25's `--console` retry and the plain terminal login were not needed and stay untested on Windows; the relay sentence in Credentials stands there.

2026-10-16 (C39, same finding): exit 2 blocks and the hook's stderr reaches the model wrapped as `PreToolUse:<tool> hook error: [<hook command>]: <stderr>` with the trailing newline kept, and a `deny` reason arrives wrapped the same way without the bracketed command (C27). `../03-hook-contract.md` strips that prefix, the optional bracket and surrounding whitespace before its `Denied by Broodforge:` and `hook:` tests, never a contains-match. Exit 1 and a timeout fall through to the engine's `-p` default, which allowed a `Read`, denied a `Write` without a prompt (a `system/permission_denied` line, then an error result) and allowed and ran a safe-listed `Bash` (`echo ok`): the timeout is a hole in read-only reach and in prompt mode. Decision: the hook stands, the plan's fallback is taken in a better form. The stream marks who decided each call (`tool_result_meta[].permission_decision.source`, `hook` or `config`; undocumented, read as an optional field), so `session` audits `hook bypassed` from the stream, not from a `hook.log` gap, 04 gets a hook-bypassed flow, and the hook's `timeout` is thirty seconds. Recorded in `../03-hook-contract.md` and `../03-architecture.md` (data flow step 5, `EngineEvent`, failure table).

2026-10-16 (S2, same finding): the Windows tree kill and the resume hold without `CLAUDE_CODE_RESUME_INTERRUPTED_TURN`, and a resumed `modelUsage` counts only turns that reached a `result`: the killed turn's tokens are absent from the next cumulative. Usage and attribution above already says "against the previous result"; `../03-architecture.md` said "the sum of the rows written", which under-counts the turn after a kill, and now takes the baseline from the last `result` row of the engine session, the interrupted row standing as its own record outside it. Also seen: the engine refuses a standalone `sleep` in Bash before any hook runs; a tool error with no permission decision gets no decision row.

2026-10-16 (C28, same finding): the sentence "Parallel tool calls in prompt mode fall through to the engine's deny; the adapter posts a one-line notice" in Decision is replaced. The engine defers each call of the message in order, ends the turn with `deferred_tool_use` naming only the last, and drops the rest without a result; after the resume the model re-issues them with new ids, each its own prompt. There is no engine deny text and `permission_denials` is empty. Observed for a deferred pair; a message mixing allowed and deferred calls is unobserved and is read the same way until a fixture shows otherwise. The plan's hook-side rule (deny every call after the first of a message) is not taken: the engine's documented hook input (session id, transcript path, cwd, tool name, input and id) carries no message id, and a denied call would be re-issued the same way, so it saves nothing. Cost: one extra prompt round, one engine process and about 3 s per dropped call; the system prompt append still asks for one call at a time, which lowers how often it happens. `session` records the dropped ids as `dropped` rows; 04's prompt message says the others come back one at a time, wording the owner's. Recorded in `../03-architecture.md` (prompt mode) and `../03-hook-contract.md` (Decision).

2026-10-16 (C26, same finding): `--settings <file>` is honoured beside `--setting-sources user`, and its hooks run even with `--setting-sources ""`. The critic's inline alternative (`--settings` with the JSON on the command line) is closed, not taken: ADR 0022 binds the bot-owned file over the engine's settings path, which needs no flag, and an inline settings string would sit in every process list and sudo log line. `--settings <file>` is the recorded second route if a host cannot bind over the path; the empty-sources half shows that no settings-source flag sheds a hook given that way.

2026-10-16 (ADR 0023, `0023-language-for-the-long-term.md`, the owner's principle of 2026-10-16): the Decision paragraph "Language bundle. TypeScript compiled with `tsc` on Node 24 ... No test framework dependency until one is missed." is replaced in full by ADR 0023's Decision: Go with a pinned toolchain, disgo, zombiezen SQLite, x/sys job objects, the standard library, one static binary per host carrying the bot, the debug CLI, the hook and `engine-entry`. The sentence "Windows needs Git for Windows for the engine's Bash tool" stands (it is the engine's). The Consequences items "one runtime for bot, CLI and tests; the largest example corpus for agents" no longer apply; the rest of the ADR, the mechanism and the command line included, is unchanged.

2026-10-16 (ADR 0023): under "Deferred, with triggers", the item "the Agent SDK (the two conditions above)" is closed: the SDK ships for TypeScript and Python only, a Node sidecar is a second runtime and is rejected, so the second implementation named in Decision, Mechanism ("The Agent SDK is a second implementation designed in the interface and built only when both hold") is withdrawn. The item "mid-turn message injection (the UX doc asks and the SDK is built)" keeps its first trigger and replaces the second with "a long-lived engine process per session is adopted under mechanism B", which would supersede the per-turn process model. The supersede trigger "the owner grants the SDK exception and asks for a long-lived session" reads "the owner asks for a long-lived session".

2026-10-16 (S1, same finding): the Consequences paragraph on AskUserQuestion is confirmed: under plain `-p` the tool is not in `init.tools`, the model's `ToolSearch` for it finds nothing, and it asks in prose and ends the turn; the stream carries no call and no denial. Every stream shape 03 reads was seen, `modelUsage` is cumulative across resumes, the engine emits one `assistant` line per content block sharing one `message.id` (the adapter's deduplication by message id is load-bearing), and `is_error` is absent, not `false`, on a successful `tool_result`. Every `rate_limit_event` carries the account's plan meters; using them is an idea, not a decision here.

2026-10-16 (owner decision, the relay question of the prior-art amendment above): the DM login relay stays off by default and API keys are the droplet fallback for subscriptions, as Decision, credentials, already says; that text stands unchanged. Whether the relay counts as completing sign-in "through Anthropic's own flow" is not decided here but asked of Anthropic once the droplet has real users, and the relay is not enabled before an answer. The PC path (the user logs in at the keyboard with the config directory set) is unaffected. Nothing in the build changes.

2026-10-16 (P7 review, `../08-implementation-plan.md`, P7): Credentials' "an explicit `CLAUDE_CONFIG_DIR` or `ANTHROPIC_API_KEY`" holds per credential kind; on the same-account backend an API-key process gets both, `CLAUDE_CONFIG_DIR` naming the session home's `.claude`, so the owner's `~/.claude` is never used. The body is unchanged; the rule is recorded in ADR 0022 and its companion (Amendments).
