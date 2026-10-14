# 0020. Harness mechanism, language and runtime, Discord library, build toolchain

Date: 2026-10-14 (revised the same day after `0020-critic-review.md`; C22 answered by the owner
the same day)
Status: promoted to ADR 0020

## Question

How Broodforge drives the Claude Code program, in which language and runtime, with which Discord
library, and with which build, test and run commands on each host. Decided now because every other
design doc (`03-architecture.md`, `05-debug-interface.md`, proposal 0022) is written in this
language and against these harness facts (`06-platforms.md`, Order). The four sub-choices are
coupled: the harness mechanism restricts the language, the language picks the Discord library, the
toolchain follows the language. The owner wants a working first version on both hosts within days;
this proposal weights time to first version above every other cost and says so where it decides.

The critic's verification table in `0020-critic-review.md` is authoritative over this body where
the two differ; the body below has been corrected to it, and the responses section records each
correction.

## Constraints that matter here

- Target hosts: the owner's Windows 11 PC and a Linux droplet, one code base, no host-only feature
  (N-01, N-04); licensing GPL-3.0-or-later with every dependency GPL-compatible and named (N-23,
  N-24); an agent-drivable debug surface for every feature (R-50, N-05 to N-11); agents write all
  the code.
- The primary harness is the real Claude Code program, so that a subscription can pay (R-38,
  `00-vision.md`); a hand-rolled Messages API loop is ruled out as the primary engine and may
  appear later as an API-key-only second harness behind the same interface (R-42).
- Per turn: the speaker's identity goes to the agent with the message (R-14); progress streams at
  about one Discord write per second, coalesced, nothing dropped (R-20, R-21, R-23, N-25); long
  output is split (R-22); permission requests become buttons and the answer goes back (R-24);
  policy mode (the term this proposal picks for R-26's "auto mode", see Process model) answers
  them itself and the driver can switch mode mid-session (R-26, R-28).
- Attribution needs the name of every model that worked, subagents included (R-32); usage is
  recorded per turn, including failed and interrupted turns, before the result is shown (R-44,
  N-21).
- One credential per session, either a subscription login or an API key (R-39, R-40). The
  coordinator's fact check against the Claude Code docs (2026-10-14) established that subscription
  credentials are read only by the Claude Code binary and the surfaces that wrap it. The critic's
  reading of Anthropic's Legal and compliance page (review, objection 2) narrows the fact check:
  a token pasted into Discord and stored by Broodforge is a stored session token, which the terms
  forbid; the permitted shape is each user signing in to the unmodified binary themselves. The
  section "Credential shapes under the terms" carries this; R-39 as written is not implementable.
- Concurrent sessions do not block one another (N-26); a crash never corrupts a worktree, the
  store or the audit log (N-27); no POSIX signals on Windows, stop processes by PID
  (`../../reference/claude-code-on-windows.md`).
- Engine facts the critic verified and this body relies on: the engine on the PC is `2.1.296`
  (npm); the TypeScript SDK `0.3.300` bundles engine `2.1.300` as a second binary; usage totals in
  `result` are cumulative across resumed sessions since `2.1.277`; the `PreToolUse` hook may
  return `defer` in `-p` mode; `--bare` never reads OAuth credentials.

## Options

The harness mechanism is options A to C; the language bundles (language, runtime, Discord library
and alternative, toolchain, harness install) are D to F. Any bundle can use mechanism B; only D and
E can use A.

### A. The Claude Agent SDK in process

What it is: Claude Code packaged as a library (`@anthropic-ai/claude-agent-sdk` for TypeScript,
`claude-agent-sdk` for Python). Broodforge calls `query()` per turn; the SDK spawns the Claude Code
engine it bundles (or, in Python, the engine at `cli_path`) and speaks its stream protocol
internally.

- License (C3, verified): the TypeScript SDK is not open source. npm says `SEE LICENSE IN
  README.md`, the repository has no LICENSE file, and the README says use is governed by
  Anthropic's Commercial Terms of Service. The bundled engine is `(c) Anthropic PBC. All rights
  reserved` under the same terms. The Python SDK's own code is MIT, but its wheel bundles the
  same proprietary engine. A GPL-3.0 program that imports the TypeScript SDK links a non-free
  library (GPL FAQ, GPLIncompatibleLibs) and needs a GPLv3 section 7 additional permission from
  the copyright holder; that is a change to ADR 0001 and the owner's call. The one SDK
  combination whose imported code is GPL-compatible is E with `cli_path` pointing at the host's
  own engine.
- Permission relay: a callback (`canUseTool` / `can_use_tool`) receives each request and returns
  allow (with optional `updatedInput`) or deny (with a `message`, optionally `interrupt`). In
  TypeScript it works with a string prompt; the streaming-input form is required only for
  `interrupt()`, `setPermissionMode()`, `setModel()` and `applyFlagSettings()`. Python's `query()`
  needs the documented dummy-hook workaround to keep the stream open. A pending callback never
  times out, so a bot restart kills the waiting engine and the prompt with it; the `defer` round
  trip (option B, and Process model) is the relay that survives a human wait under A too, as an
  SDK `PreToolUse` hook.
- Interrupt: `interrupt()` on the query, a control request over the engine's stdin, streaming
  form only. On Windows it is not known to work: TypeScript SDK issue #444 (open, bug) reports
  `claude.exe` dying with an access violation (`0xC0000005`) on `interrupt()`; #359 (open)
  reports the stream closing after the first tool call on Windows. The control request is engine
  code, so the bug is the engine's and B inherits it; on Windows the only interrupt known to work
  is terminating the process tree and resuming.
- Usage: a `result` message per turn with `usage` (main loop only), `modelUsage` (per model,
  subagents and internal calls included, cumulative across resumes), `total_cost_usd`,
  `num_turns`, `is_error`, `permission_denials`. Every `assistant` message carries
  `message.model` and `message.usage`, but per-message `output_tokens` is a placeholder. Subagent
  messages carry `parent_tool_use_id`.
- Credential: `env` passed to the query (TypeScript replaces the environment, so pass
  `{ ...process.env, ... }`; Python merges); per turn when each turn is its own `query()`.
- Resume: a `resume` option with the engine's session id (R-18 later). Mid-session, streaming
  form: mode and model setters; effort through `applyFlagSettings({ effortLevel })`; per process
  the `effort` option.
- Cost: lowest code to a first turn; one package install plus the SDK's peerDependencies (`zod`,
  `@anthropic-ai/sdk`, `@modelcontextprotocol/sdk`), which the bot must list; typed messages. The
  SDK pins its own engine, a second binary beside any Claude Code the owner installed, so a bug
  report must name which engine ran.
- Risks: the license above; a breaking SDK release breaks the adapter; the open Windows issues.
- Who else: Anthropic's own headless integrations. (The earlier claim that most third-party
  front ends use it was unverifiable and is withdrawn.)

### B. The Claude Code CLI as a subprocess, stream JSON in and out

What it is: Broodforge spawns `claude -p --output-format stream-json --verbose --input-format
stream-json` per turn (`--verbose` is required with `stream-json` under `-p`), writes the user
message, closes stdin (an open stdin costs a 3 s wait before the turn starts), parses the JSON
lines it emits, and runs the engine the owner installed on the host.

- Permission relay: a `PreToolUse` command hook, Broodforge's own CLI entry point, decides every
  tool call. Inside policy it returns `allow` or `deny` with a message. For a call a person must
  answer it returns `defer`: the engine exits with `stop_reason: "tool_deferred"` and the pending
  call (`deferred_tool_use`: id, name, input) in the result; Broodforge posts the buttons; when
  the click arrives, minutes or hours later, Broodforge runs `--resume <session id>` with the
  same hook, the same tool call fires `PreToolUse` again, and the hook returns the person's
  answer. No timeout or retry limit on the engine's side; the session waits on disk. This is the
  documented route for a host whose user may answer slower than a process should live, it needs
  no MCP server, and it is the per-turn process model. Constraint: `defer` is honoured only in
  `-p` mode and only when the turn has a single pending tool call; parallel calls fall through to
  the engine's own flow, which in `-p` denies them and tells the model. The MCP route
  (`--permission-prompt-tool`) is not used: its tool schema is undocumented (claude-code issue
  #24595, closed stale) and HTTP acceptance for it is unstated (C9).
- Interrupt: Windows, terminate the process tree by PID (`taskkill /T` or a job object, 0022's
  concern) and resume on the next turn; Linux, SIGINT ends the turn with a `result` (SIGTERM
  leaves it unfinished, exit 143, still resumable). A killed turn emits no `result`; the usage
  rule in Process model covers it.
- Usage, subagent models, credential, resume: the same facts as A, because A is this protocol
  wrapped; `--resume <session id>` per turn (since `2.1.223` found from any cwd), env per
  process. Mid-session changes are trivial with a process per turn: `--permission-mode`,
  `--model`, `--effort` are flags.
- Cost: more adapter code than A (process management on two hosts, a JSON line parser, a type
  file written from the documented shapes); no typed messages. Startup: measured on the PC, a
  Haiku turn is 4.1 to 4.8 s wall with stdin closed, of which the API call is 1.4 to 1.5 s, so
  the engine's overhead is about 2.5 to 3.5 s per turn; the droplet is unmeasured (the spike
  measures it). The engine's executable path is configuration naming the `.exe` or ELF binary
  itself, never a `PATH` lookup and never the npm `.cmd` shim, which Node will not spawn without
  a shell (C14); an unset path refuses instance start with a message naming the configuration
  key.
- Risks: the stream's message shapes change with releases (the `capabilities` set in
  `system/init` is the feature-detection hook; many behaviours are version-gated); the engine
  version is whatever the host has, so both hosts pin it; a Windows kill must take the tree or
  the engine's tool subprocesses survive.
- Who else: CI integrations, editor plugins and shell wrappers that drive `claude -p`; the SDK
  overview itself says other languages "run the CLI as a subprocess with the `-p` flag".
- License: the engine is a separate program driven over pipes, like git; the GPL FAQ treats
  pipes, sockets and command-line arguments as communication between separate programs
  (MereAggregation). No Claude Code code is linked into or distributed with Broodforge. This is
  the clean position under N-24 and needs no exception.

### C. Both, behind one adapter interface

What it is: the adapter interface from `03-architecture.md` (start turn, stream events, answer
permission, interrupt, usage) with A and B as two implementations selected by configuration.

- Meets every constraint either implementation meets; the interface itself is required anyway by
  R-42 (a second harness later) and by the debug interface, which needs a fake harness for
  scratch runs (N-09).
- Cost: two implementations to keep green on two hosts; the second rots unless the debug
  interface runs both. Learning: both docs.
- Risk: spending the first days on two routes instead of one working one. Mitigation: the
  interface and a fake harness first, one real implementation second, the other only when a named
  condition (below) triggers it.

### D. TypeScript on Node.js, discord.js, npm

- Language and runtime: TypeScript compiled with `tsc`, Node 24 (Active LTS) pinned in
  `package.json` `engines` and `.nvmrc`; the PC's Node 22 is Maintenance LTS and is upgraded as
  an install step. Mechanism A or B. Process spawning, pipes and timers are host-neutral in Node;
  `child.kill()` on Windows is a hard terminate of the direct child only.
- Discord library: discord.js `14.x` (Apache-2.0, released 2026-10-14): gateway reconnect and
  resume, REST rate-limit queue, message edit, buttons and components, thread creation, channel
  creation with permission overwrites, member role fetch at the moment of an action (N-18).
  Alternative: Oceanic (MIT, maintained), a lighter library with the same gateway and component
  coverage and a smaller community.
- Storage binding, as ADR 0021 asks this proposal to name it: `better-sqlite3` (MIT, prebuilt
  natives for both hosts; verified GPL-compatible in the 0021 review).
- Toolchain, both hosts: `npm ci`; `npm run build`; `npm test` (`node:test`, stable, with
  `describe`, `before`, `mock`; no test framework dependency); `npm start -- <args>`; the debug
  CLI is the same entry point. Windows: Node from the installer or `winget`; Linux: NodeSource or
  `nvm`.
- Harness install: A needs `npm ci` and lists the SDK's peerDependencies; B needs Claude Code
  installed per host (native installer gives `%USERPROFILE%\.local\bin\claude.exe`; npm gives a
  `.cmd` shim over `node_modules/@anthropic-ai/claude-code/bin/claude.exe`, the path the
  configuration names) and, on Windows, Git for Windows for the engine's Bash tool
  (`CLAUDE_CODE_GIT_BASH_PATH`; without it the engine uses its PowerShell tool).
- Time to first version: shortest. The largest body of Discord bot examples agents have seen; one
  runtime for bot, debug CLI and tests; under B the language is free and this stays the shortest.
- Cost and risk: Node's single thread is fine here (the work is I/O); a CPU-heavy feature would
  need a worker. Dependency count is the highest of the three bundles; the lock file pins it.

### E. Python, discord.py, uv

- Language and runtime: Python 3.12 or later, `asyncio`. Mechanism A (`claude-agent-sdk`, MIT
  code; with `cli_path` to the host's engine the import is GPL-compatible, the only SDK route that
  is) or B. Process handling on Windows through `asyncio` subprocesses and
  `CREATE_NEW_PROCESS_GROUP`; no signals to children, terminate by PID as in D.
- Discord library: discord.py (MIT, maintained): gateway reconnect, rate-limit handling, message
  edit, `ui` buttons and views, thread creation, channel creation with overwrites, `fetch_member`
  for fresh roles. Alternative: hikari (MIT, maintained), a lower-level gateway and REST library
  with its own component layer.
- Toolchain, both hosts: `uv sync`; `uv run pytest`; `uv run broodforge <args>`. Windows: Python
  from the installer or `winget`, `uv` from its installer; Linux: `uv` installs its own Python.
- Harness install: as D. The Python SDK refuses `.cmd` and `.bat` CLI paths outright.
- Time to first version: close to D, with Windows against it: the Python SDK's Windows history
  (hangs, the `.cmd` refusal, command-line length limits, a Windows wheel missing from PyPI for
  four releases this month), the `query()` dummy-hook workaround, and no `SessionStart` or
  `SessionEnd` hooks.
- Cost and risk: two runtimes (Node or the native engine for Claude Code, Python for the bot);
  packaging for a service on Windows is less common than for Node; type checking is optional and
  must be enforced by the toolchain to get the same safety as D. `sqlite3` in the standard
  library is a small point for E under 0021.

### F. Go, discordgo, the Go toolchain (the compiled comparison)

- Language and runtime: Go, one static binary per host, goroutines per session. Mechanism B only;
  there is no Agent SDK for Go, so the adapter is written against the stream JSON and the
  `PreToolUse` hook. Rust with serenity and poise is the other compiled candidate and differs in
  build and learning cost, not in these harness facts.
- Discord library: discordgo (BSD-3-Clause) is slow: last release May 2025, last push February
  2026. If F is ever revisited, disgo (Apache-2.0, pushed this month, more complete on components
  and threads) is the primary and discordgo the alternative.
- Toolchain, both hosts: `go build ./...`; `go test ./...`; `go run ./cmd/broodforge <args>`.
  Cross-compiles for the droplet from the PC.
- Harness install: as B in D, on both hosts.
- Time to first version: longest of the three. Every message type is hand-modelled, and the
  Discord ecosystem has fewer worked examples of streamed edits and component flows than D or E.
  With B now the first mechanism for every bundle, F's extra cost over D is the type file and the
  example corpus, not the subprocess route itself.
- Cost and risk: lowest runtime footprint and the simplest service install; a single binary is a
  real advantage on the droplet and none on the PC. The owner's time is spent reading Go diffs for
  a bot whose work is I/O and string handling.

## Process model, under any mechanism

One engine process per turn, in the session's worktree, resumed by the engine's session id from
the previous turn; not one long-lived process per session. Reasons: interrupt is kill-and-resume
on Windows anyway; a crash of the bot leaves no engine process to lose (R-17); resume (R-18) is
exercised from the first turn rather than built later; the `defer` permission round trip is this
model; mode, model, effort and credential are process arguments, so Q 5 costs nothing in the
adapter. Costs: engine startup per turn (about 3 s on the PC), and a message written while a turn
runs is queued for the next turn, not injected (the domain doc's open question; the UX doc may ask
for injection later, which A supports in streaming-input form).

Permission modes. Claude Code has a permission mode literally named `auto` (a classifier reviews
actions, denies on no verdict, stops the turn after ten such denials, and is the engine's own
default for `-p` on some versions). Broodforge's R-26 mode is a different thing, so this proposal
names it **policy mode**: Broodforge answers every permission request itself from the channel's
reach. Its counterpart stays **prompt mode**. The engine's `auto` is recorded as a possible third
mode later, never as Broodforge's policy mode. The word the users see in Discord is `04-ux.md`'s
and the owner's; the requirements doc's "auto mode" is amended when this proposal is promoted.
Mapping, every turn, under either mechanism:

- `--permission-mode default` is passed explicitly on every process; never `bypassPermissions`
  (refuses root on Linux, inherited by subagents), never `dontAsk` (nobody can be asked), never
  the engine's `auto`.
- One `PreToolUse` hook (Broodforge's own CLI entry point, `broodforge hook pre-tool-use` as the
  working name, with a `timeout` set on its matcher) decides every tool call. It learns the
  session id, mode and reach from the process environment Broodforge sets (not secret; the Bash
  tool sees the same environment). Outside the channel's reach: `deny` in both modes, with the
  message `Denied by Broodforge: <tool> on <target> is outside this channel's reach (<reach
  name>); nobody in this session can allow it; the channel's configuration sets reach` (R-27).
  Inside reach: policy mode returns `allow`; prompt mode returns `defer`.
- On resume after a click, Broodforge passes the answer for the pending `tool_use_id` in the
  resumed process's environment; the hook returns `allow` or `deny` for that id and otherwise
  behaves as above. Nothing is written to the store by the hook (ADR 0021: one writer).
- Parallel tool calls in prompt mode fall through to the engine's deny. The adapter sees a turn
  that ended with `permission_denials` and no deferred call, posts a one-line notice, and the
  system prompt append tells the model to make calls that need approval one at a time. The spike
  characterises this.
- Nobody-answers: the engine has no timeout; Broodforge's idle timeout (R-15) ends the session
  and the deferred process state is just a transcript on disk until the engine's retention sweep.

Settings sources, every turn. Without `--bare`, a `-p` session runs the hooks in the worktree's
`.claude/settings.json` and connects its `.mcp.json` servers with no trust dialog, so a pull
request that adds a hook is code execution under the session's credential at the next turn,
before any reach check. `--bare` never reads OAuth credentials, so subscription turns cannot use
it. Rule, both mechanisms: `--setting-sources user` (the user source is the Broodforge-owned
`settings.json` in the session's `CLAUDE_CONFIG_DIR`, where the hook is registered) and
`--strict-mcp-config` with an explicit `--mcp-config` on every process; the engine's default
source set is never relied on. Whether `--settings <file>` is honoured alongside
`--setting-sources user` is a spike claim (C26); if it is, the hook may live there instead.

Usage (N-21, R-44). The per-turn usage record is the per-model delta of `result.modelUsage`
between this turn's result and the previous result of the same engine session (totals are
cumulative across resumes). `result.usage` is main-loop only and is kept as a secondary field.
A turn without a `result` (killed, crashed) is recorded with the interrupted flag and the
per-message input and cache counts from the `assistant` messages seen, deduplicated by message id
(partial messages repeat them), with output tokens unknown; so an ending interrupt (SIGINT on
Linux) is preferred over a kill wherever it works. The record is written before the result is
posted (ADR 0021, reconciliation).

Attribution (R-32). The models that worked in a turn are the keys of `result.modelUsage`; for an
interrupted turn, the distinct `assistant.message.model` values seen. Hooks carry `agent_id` and
`agent_type` but no model, so they add nothing. `modelUsage` includes internal calls (compaction);
whether a model that only compacted is a co-author is `03-architecture.md`'s to say.

Harness-owned state. Transcripts, `.claude.json` and the retention sweep live under the session's
`CLAUDE_CONFIG_DIR`; resume (R-18) depends on them. The config-dir rule therefore binds a
session's resumability to a credential's directory, which must sit inside the backup root (ADR
0021, harness-owned state) and inside whatever 0022 isolates.

Non-root. The bot and the engine run as their own non-root user on the droplet, never as root:
least privilege, and `bypassPermissions` would refuse root even though Broodforge does not use
it. An input to 0022.

## Credential shapes under the terms (C22, owner decision)

Anthropic's Legal and compliance page (quoted in full in the review, objection 2) forbids
third-party developers to offer claude.ai login in their applications, to route requests through
Free, Pro or Max credentials on behalf of their users, and to collect, store or intermediate
claude.ai credentials or session tokens; sign-in must complete through Anthropic's own flow. It
permits an end user signing in to the unmodified Claude Code binary with their own subscription,
including where a platform hosts Claude Code, provided the binary is unmodified, no authentication
method is removed, the platform neither pays for nor intermediates usage, and each user
authenticates with their own credentials. Plan limits assume ordinary individual usage, and
enforcement may come without notice.

Consequences for R-39 and R-40:

- A `claude setup-token` pasted into Discord and stored by Broodforge is a stored session token.
  Not offered to anyone. (The owner's own CI-style token on the owner's own host is the owner's
  to assess outside this design; nothing here depends on it.)
- API keys are unaffected: stored by 0022's credential store, delivered as `ANTHROPIC_API_KEY`
  in the process environment, and the shape the SDK overview itself recommends for agents.
- The only subscription shape that resembles the carve-out, and the one this proposal recommends
  if subscriptions stay in scope: Broodforge keeps one `CLAUDE_CONFIG_DIR` per registered user
  per host, owned by Broodforge's user, holding that user's `settings.json` (the hook, written by
  Broodforge) and whatever the engine writes. The user completes Claude Code's own login into it
  themselves. On the PC the owner runs `claude auth login` with that directory set. On the
  droplet, where users have no shell, Broodforge runs the engine's login command with the
  directory set, relays the login URL it prints to the user by Discord direct message, the user
  signs in at Anthropic in their own browser, pastes the code Anthropic shows back to the direct
  message, and Broodforge writes it to the login process's stdin. Broodforge never reads the
  token; the engine writes it into the directory and reads it on every turn of that user's
  sessions. Sessions bind to one account (R-40 holds by construction).
- Residual risk, stated honestly: Broodforge still relays the one-time authorization code and
  hosts the directory the token lands in, which is "hosting the unmodified binary" by the
  carve-out's own wording but is also the closest a hosted platform comes to intermediating; a
  community droplet serving several users on their own plans is further from "ordinary
  individual usage" than the owner's single-user PC; enforcement, if it comes, falls on the
  user's account, without notice; and the headless login relay (URL printed, code pasted,
  without a TTY) is unverified on either host (C25). Asking Anthropic remains an option.

The owner's question, put by the coordinator: under this shape, do subscriptions stay in the
first version, or is the first version API keys only? Either way both credential kinds stay in
the design (`01-requirements.md`, What waits on the owner). Nothing in the mechanism or bundle
choice changes with the answer; 0022's delivery design does.

The owner's answer (2026-10-14, recorded in ADR 0020): subscriptions stay in the
first version under the per-user `CLAUDE_CONFIG_DIR` shape above; the user completes Claude
Code's own login in a directory Broodforge owns; Broodforge never reads or stores a token; no
setup-tokens. The direct-message login relay for the droplet is a feature that is off by default
and that the owner enables knowingly. API keys are supported as the fallback delivery. "We'll be
careful with that."

## Recommendation

Bundle D with mechanism B first, behind the adapter interface of C: TypeScript on Node 24,
discord.js, `better-sqlite3`, npm and `node:test`; the engine the owner installs and pins on each
host, driven as one `claude -p --output-format stream-json --verbose --input-format stream-json`
process per turn with stdin closed after the message, `--setting-sources user`,
`--strict-mcp-config`, an explicit `--permission-mode default`, and the `PreToolUse` hook of the
Process model applying reach policy and returning `defer` for what a person must answer;
interrupt by process-tree kill and resume on Windows, SIGINT on Linux; usage from `modelUsage`
deltas. It is the shortest path to a turn streaming into Discord on both hosts, its license
position needs no exception, and every harness fact the other docs need (permission relay,
interrupt, per-model usage, subagent names, resume) is available through documented engine
behaviour. Mechanism A is designed in the interface and built only after two conditions hold:
the owner records a GPLv3 section 7 additional permission for the Agent SDK in ADR 0001, and the
spike shows `interrupt()` stable on Windows; until then A offers nothing over B that the per-turn
process model uses. The subscription credential shape is the per-user config dir with the user's
own login, pending the owner's answer on C22; API keys are in either way.

First implementation step: the coder writes the adapter interface and a fake harness that replays
a recorded `stream-json` transcript, then a spike on each host that runs one turn through B with a
permission request deferred and answered on resume, an interrupt mid-turn (kill on Windows, SIGINT
on Linux) followed by a resume, a parallel-tool-call turn in prompt mode, and prints the
`modelUsage` delta and model names seen; the spike's output on both hosts is attached to ADR 0020
and settles C25 to C28 and the droplet's turn latency.

## What would change this recommendation

- The owner grants the section 7 exception and the Windows interrupt spike is clean under A:
  then A becomes the second implementation, still behind the same interface, and a long-lived
  session (message injection, mid-turn setters) becomes possible later.
- The spike shows the `defer` round trip failing on resume on either host: then the permission
  relay falls back to the SDK callback under A (which reopens the license question) or to a
  verified `--permission-prompt-tool` server; both are spike-gated.
- The owner says they will read the code daily and know Python or Go far better than TypeScript:
  then E or F, since the owner develops through the bot and reads its diffs. E's Windows record
  this month counts against it.
- The droplet must run from a single static binary with no runtime installed: F with disgo.
- Owner's C22 answer "API keys only": no change to mechanism or bundle; `--bare` becomes an
  option for the engine (then the settings-sources rule is satisfied by `--bare` itself), and
  0022's config-dir delivery is designed on paper only.
- Owner's Q 3 (threads or private channels): no change; all three libraries do both.
- Owner's Q 5 (switching from Discord): no change under the per-turn process model.
- The engine's stream stops carrying `modelUsage` or `parent_tool_use_id`: the attribution
  source moves to the `assistant.message.model` values seen; no mechanism change.

## Claims to verify

The critic's table in `0020-critic-review.md` verified or refuted C1 to C24 (the coordinator's fact check of
2026-10-14 is summarised in the credentials bullet above, not repeated); the body above is corrected to it.
Still open, for the spike, with web checks by the critic where marked:

- C25. The engine's login command, run without a TTY with `CLAUDE_CONFIG_DIR` set, prints the
  authorization URL and accepts the pasted code on stdin on both hosts; what it writes into the
  directory; whether `claude auth login` or the interactive `/login` is the command.
- C26. `--settings <file>` is honoured together with `--setting-sources user` (critic, docs);
  and whether an empty `--setting-sources` still runs hooks given by `--settings`.
- C27. The `defer` round trip across a kill-free wait on Windows: the deferred process exits
  cleanly, `--resume` with the hook re-fires `PreToolUse` for the same `tool_use_id`, and the
  `updatedInput` or deny is honoured.
- C28. What a parallel-tool-call turn in prompt mode produces in `-p`: the deny text the model
  sees, and whether `permission_denials` on the result identifies the calls.
- C29. The droplet's per-turn overhead in seconds, measured as C15 was on the PC.

## Choices made where the docs were silent

- One engine process per turn, resumed by session id, rather than one per session.
- The engine executable path is configuration naming the binary itself; no `PATH` lookup, no
  `.cmd` shim; an unset or missing path refuses instance start naming the key.
- Broodforge's R-26 mode is called policy mode in the design docs; the engine's `auto` is never
  it.
- The usage record is the `modelUsage` delta when a result arrives, else an interrupted record
  from deduplicated per-message input and cache counts (N-21).
- A message written during a running turn is queued for the next turn in the first version.
- Tests use `node:test`; no test framework dependency until one is missed.
- The harness never runs with the owner's default config dir on the PC; it always gets an
  explicit `CLAUDE_CONFIG_DIR` (subscription) or `ANTHROPIC_API_KEY` (API key) from 0022's
  delivery.
- Every engine process gets `--setting-sources user`, `--strict-mcp-config`, an explicit
  `--mcp-config` and `--permission-mode default`.

## What 0022 must now handle

Handed over as hard requirements, not options:

- Credential delivery per process: `ANTHROPIC_API_KEY` in the environment for API keys; a
  per-user `CLAUDE_CONFIG_DIR` for subscriptions, created and owned by Broodforge, the login
  relayed as described under "Credential shapes" (the owner kept subscriptions in scope; the
  relay is off by default and enabled knowingly). No stored subscription tokens.
- Neither delivery hides the credential from the session's own tools: the Bash tool inherits the
  environment and `.credentials.json` is readable by the engine's user; the engine's sandbox is
  unsupported on native Windows. 0022 says this per host (N-15) and cannot rely on the mechanism
  to fix it.
- The settings-sources rule above, as the first line of reach enforcement: repository hooks and
  `.mcp.json` never load.
- The bot and engine run as a non-root user on the droplet; the Windows process tree kill
  (`taskkill /T` or a job object) is the interrupt; Linux gets SIGINT.
- Each per-user config dir sits inside the backup root and inside the isolation boundary (ADR
  0021, harness-owned state); its `settings.json` is Broodforge-written and carries the hook.
- Under B the isolation wrapper is the spawn call itself (`systemd-run`, `bwrap`, a different
  user); if A is ever built, `spawnClaudeCodeProcess` is the equivalent hook.

## Critic's objections and responses

1. **Blocking, C3 fails for D+A.** Response: confirmed from the primary records quoted in the
   review; the TypeScript SDK is under commercial terms and importing it into a GPL-3 program
   needs a section 7 exception that only the owner can grant, which is a licensing-policy change
   and not this proposal's to make. B is mere aggregation and needs nothing. The body's claim
   that C3 reverses "only the mechanism" was wrong for A (where E is the one GPL-compatible SDK
   route) and right for B (where the bundle is free). Resolution: accepted with change. The
   recommendation is B under D; A is kept in the interface and built only after the owner
   records the exception in ADR 0001 and the Windows interrupt spike is clean. Option A's text is
   corrected.

2. **Blocking, C22, R-39 forbidden as written.** Response: accepted on the facts; the body now
   carries the quoted terms in "Credential shapes under the terms". Stored setup-tokens are out
   for everyone but the owner on the owner's own host, which is outside the design. The one
   subscription shape left is the per-user config dir with the user's own login relayed through
   Discord, written as the recommended path with its residual risk stated and a spike claim
   (C25) for the headless relay. The line "API keys only ... config-dir delivery loses its main
   reason" was backwards and is corrected. Resolution: **accepted with change, owner decided
   2026-10-14**: subscriptions stay in the first version under the per-user config-dir shape
   with the user's own login; no stored subscription tokens of any kind; the droplet's
   direct-message login relay ships off by default and the owner enables it knowingly; API keys
   are the fallback delivery. Recorded in "Credential shapes under the terms" and handed to 0022.

3. **Serious, C5 contradicted on Windows.** Response: accepted; issue #444 and #359 are now
   cited in option A, the "works the same on Windows" sentence is withdrawn, and the design
   treats process-tree kill and resume as the Windows interrupt with SIGINT as the Linux fast
   path. The spike tests both. Resolution: accepted.

4. **Serious, a process waiting for a human.** Response: accepted; the `PreToolUse` `defer`
   round trip is now the permission relay under either mechanism (Process model), with its two
   constraints stated (`-p` only, single pending call) and the parallel-call case given a
   behaviour and a spike claim (C28). B needs no MCP server; A needs no live callback across a
   wait. Resolution: accepted.

5. **Serious, the usage rule cannot record N-21.** Response: accepted; the rule is restated as
   the per-model `modelUsage` delta between consecutive results of the same engine session, with
   an interrupted record from deduplicated per-message input and cache counts and unknown output
   when no result arrives, and a preference for an ending interrupt over a kill. Resolution:
   accepted.

6. **Serious, repository content runs before any permission check.** Response: accepted; the
   settings-sources rule (`--setting-sources user`, `--strict-mcp-config`, explicit
   `--mcp-config`, every process, both mechanisms) is in the Process model and handed to 0022 as
   a hard requirement; `--bare` is noted as available only when the credential is an API key.
   One open point: whether `--settings <file>` works alongside it (C26). Resolution: accepted.

7. **Serious, "auto mode" names two things.** Response: accepted; Broodforge's mode is now
   policy mode, mapped to `--permission-mode default` plus the hook, passed explicitly on every
   process; `bypassPermissions`, `dontAsk` and the engine's `auto` are excluded with the reasons
   given; the deny text is defined; the non-root rule goes to 0022. The user-facing word waits
   for the UX doc and the owner. Resolution: accepted with change (the term is the proposal's
   choice, not the critic's wording).

8. **Minor, C4 misstated.** Response: accepted; option A now says the callback works with a
   string prompt and the streaming constraint is on `interrupt()` and the setters, with Python's
   workaround named. Resolution: accepted.

9. **Minor, C9 unverifiable.** Response: accepted; the MCP permission route is dropped from B,
   with objection 4's relay in its place, and kept only as a spike-gated fallback under "What
   would change this recommendation". Resolution: accepted.

10. **Minor, stdin wait and `--verbose` are adapter rules.** Response: accepted; both are in
    option B's first paragraph with the measured numbers, and the droplet's measurement is spike
    claim C29. Resolution: accepted.

11. **Minor, the engine path must name the `.exe`.** Response: accepted; option B and the
    silent-choices list say the path names the binary, never the `.cmd` shim, and an unset path
    refuses start naming the key. Under B there is one engine per host, pinned by the owner.
    Resolution: accepted.

12. **Minor, `result.modelUsage` keys as the attribution source.** Response: accepted;
    Attribution in the Process model uses the keys, with the stream's `assistant.message.model`
    as the fallback for interrupted turns, and the compaction-only question is handed to
    `03-architecture.md`. Resolution: accepted.

13. **Minor, engine state lives in the credential's config dir.** Response: accepted;
    Harness-owned state in the Process model says the directory sits inside the backup root and
    the isolation boundary, consistent with ADR 0021. Resolution: accepted.

14. **Minor, housekeeping on C20 and C21.** Response: accepted; disgo is F's primary if F is
    revisited, the peerDependencies are listed under A, Node 24 is pinned. Resolution: accepted.

15. **From the review's refusals table: the policy deny message is undefined, and the
    refusal on an unset engine path or config dir does not name the key.** Response: both
    defined above (Process model, deny text; option B, start refusal). The deferred-with-nobody
    case is Broodforge's idle timeout. Resolution: accepted.

Standing after revision: every objection resolved; objection 2 by the owner's decision of
2026-10-14. Promoted to `../decisions/0020-harness-language-library-toolchain.md`.
