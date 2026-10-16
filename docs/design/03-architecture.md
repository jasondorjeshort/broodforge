# Architecture

Date: 2026-10-14. Status: first draft, written from `01-requirements.md`, `02-domain.md`,
`06-platforms.md` and ADRs 0020, 0021 (with `decisions/0021-backup-and-restore.md`) and 0022
(with `decisions/0022-linux-spawn-and-handoff.md`); it settles what they handed to it, cites them
for the rest, and where it disagrees the ADR wins. What humans see is `04-ux.md`'s; the debug
commands and wire format `05-debug-interface.md`'s; the hook's environment, path checks and policy table `03-hook-contract.md`'s.

## Components and boundaries

One Go binary, one process per instance (ADRs 0020, 0023), one directory per component under `src/`
[no-check: no code yet] named by the first column. A component talks only to the ones its row
names; everything else goes through `SessionService`. Host differences live in `spawn`,
`worktree-remove`, `control` (the socket path) and nowhere else (`06-platforms.md`, Sharing).

| Component | Owns | Talks to |
|---|---|---|
| `discord` | Gateway, command parsing, role lookup at the moment of an action (N-18), the key-paste scan, rendering of session events into posts and edits written to an outbox sink (the dry-run renderer's output, read over control as `session outbox`), the per-session coalescer (R-21) with an injectable clock, buttons; `discord inject message\|click` feeds its handlers without a gateway, dry-run only | `SessionService`, `config`, `policy` |
| `control` | The local socket the CLI connects to; exposes `SessionService` and the read commands of ADRs 0021 and 0022 as request and response lines; 05 fixes the commands; calls `policy`'s role check as `discord` does | `SessionService`, `store`, `backup`, `policy` |
| `session` | `SessionService`: state machine, participants, turn queue, mode switches, idle and prompt timeouts, the end-of-session sequence; emits `SessionEvent`s | `engine`, `git`, `policy`, `credentials`, `store`, `audit` |
| `engine` | The `Engine` interface, the `claude-code` implementation (one process per turn through `spawn`), the `fake`, the stream parser, usage deltas | `spawn`, `credentials` (secret handed to the spawn only) |
| `hook` | `broodforge hook pre-tool-use`, run by the engine inside the session; a pure function of stdin and environment (`03-hook-contract.md`) | nothing in the bot process |
| `policy` | The reach table, the decision function the hook and `session` share, the deny text, the role check (a pure function of supplied roles and a channel's role list), the model display-name map | nothing (pure) |
| `credentials` | `CredentialStore` over `credentials/store.json`, credential status, harness config directories through `engine-entry`, `credential login`, the DM relay (off by default) (ADR 0022) | `spawn`, `audit` |
| `store` | The SQLite data layer, migrations, reconciliation, scratch roots (ADR 0021) | `audit` (same transaction) |
| `audit` | The sink interface: the `audit_event` table first, the optional file sink second | `store` |
| `git` | Canonical checkouts, `refs/sessions/*`, clone and bundle through the spawn backend, the attribution check, push and pull request | `spawn`, `credentials`, `audit` |
| `spawn` | `SpawnBackend`: `same-account` and `linux-sudo-bwrap`; interrupt; `engine-entry`; preflight | the OS |
| `backup`, `config` | `backup snapshot`, `verify`, `rehearse` (never scheduled by the bot); loading, validation, last-good, `reload`, `config check` | `store`; `credentials` (id existence) |
| `cli` | The one entry point `broodforge <command>`; `serve` runs the bot, everything else is a client of `control` | `control` |

`SessionService` is the boundary N-11 names: `start`, `submit`, `answer`, `setMode`, `interrupt`,
`end`, `invite`, `status`, `subscribe(sessionId)`. The Discord adapter and the control socket call
the same methods and read the same `SessionEvent` stream; a behaviour reachable from Discord only is
a bug. N-26: each session's turn is its own engine process and promise chain, nothing in `session`
awaits another session, and store writes are short transactions (ADR 0021): a stuck engine delays only its own session, until the idle timeout.

Registration and participants (R-01 to R-04, R-13, R-25, N-18). `discord` owns the register
command: one registration per Discord user id; registering again replaces name and email with an
audit event (R-02). `session` owns participants: the driver is the starter, fixed for the session;
a writer becomes a contributor on their first accepted message (R-13). Interrupt and end are
permitted to the channel's `answer` list (owner decision 8 in 04); `setMode` is the driver's.
`discord` looks up the roles held at that moment on every message and click (R-25, N-18), never
writes one (R-04); the role check is `policy`'s pure function of the roles supplied and the
channel's `start`, `contribute` or `answer` list, used by `discord` and `control` alike. An
unregistered writer gets the register flow, an unpermitted one the refused flow (R-03). Inviting: 04.

Credential status (`usable`, `needs-login`, `removed`) is the store's record and nothing else:
a subscription credential is created `needs-login`; an auth error in a `result` sets it;
`credential login <id> --done` on the host, or the relay's success, clears it; an API key is
never `needs-login` (a rejected key is replaced). The bot never reads a config directory to
infer it. A session start naming anything but `usable` is refused (R-41).

## Data flow for one turn

1. A message arrives in a configured venue. `discord` first scans it for an Anthropic key shape
   (the `sk-ant-` prefix): a hit is deleted where the bot may, answered with 04's refusal, never
   submitted, audited `action refused` (reason `key-pasted`, no key text). Then the writer is
   checked as above; a refusal is audited the same way.
2. `SessionService.submit(sessionId, participant, text)`. Idle session: a turn starts. Turn
   running: the message is queued for the next turn (ADR 0020). Each message is one block
   prefixed with a `From:` line of the writer's git name and session role (R-14). The turn's
   requester (R-44) is its first block's writer; the turn row keeps every writer.
3. `session` writes the turn row (`started`) and calls `Engine.startTurn` with a `TurnRequest`.
4. `engine` has `spawn` start the process with ADR 0022's credential delivery, git identity and
   hook variables, writes the stream-json user message, closes stdin, parses stdout into `EngineEvent`s.
5. `session` maps them to `SessionEvent`s (`text`, `tool-call`, `tool-result`,
   `waiting-for-answer`, `turn-ended`) and writes a `permission decision` row per tool call seen:
   allowed by policy, denied by the hook (the deny text's prefix), failed in the hook (the `hook:`
   prefix), deferred, dropped (a parallel call, below), or decided by the engine alone (`decidedBy`
   `config`: the hook was bypassed, `03-hook-contract.md`). The hook writes nothing; the stream is the record.
6. `discord` renders: one working message per turn edited through the coalescer (at most one
   create or edit per second per session and 50 REST requests per second per bot across all, nothing dropped, overflow split or attached, R-20 to
   R-22); a new message for a turn end, a wait or an error (R-23). The outbound mask: all posted
   text (trace lines, prompts, results, transcript) first has credential-shaped tokens replaced by
   `[masked]`: `sk-` prefixed keys, bearer and OAuth tokens, strings over 40 characters that are
   not pure hex (so 40- and 64-character SHAs pass); source:
   `docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md`, L1.
7. On `result` the usage delta is committed with the turn row (`ended`) before `turn-ended` is
   published (N-21). Exit without `result`: `interrupted`, with the per-message counts seen.

Prompt mode: the hook returns `defer` for a call a person must answer; the process exits with
`stop_reason: tool_deferred` and the pending call. `session` records `waiting for answer`, keeps
`engineSessionId`, publishes `waiting-for-answer`; `discord` posts buttons. A click by a
participant the channel lets answer (R-25) calls `SessionService.answer`; `session` starts a new
process with `--resume` and the answer in the hook environment, and the same `tool_use_id` fires
the hook again and gets it. Other clicks are refused; no click in the prompt timeout denies it (R-53).
Parallel calls (ADR 0020, C28, observed for a deferred pair): the engine defers each, ends the turn
naming only the last, and drops the rest; the model re-issues them after the resume, one prompt and
one process each. The `tool-use` ids of that message with no result and not in `deferred` get rows `dropped`; 04 says so.
Button custom ids, parsed by one function in `discord`:
`bf:<session id>:<tool_use_id>:<allow|deny|allow-session>` for a permission answer,
`bf:<session id>:stop`, `bf:<session id>:end`, and `bf:<channel id>:start` on the panel.

## The engine adapter

```go
type Engine interface { Name() string; StartTurn(req TurnRequest) TurnHandle } // "claude-code" | "fake"
type TurnRequest struct {
	SessionID, EngineSessionID string // --resume when EngineSessionID is set
	Cwd         string              // sessions/<id>/tree
	Delivery    Delivery            // Kind "config-dir" (Path) | "api-key" (CredentialID)
	GitIdentity struct{ Name, Email string } // the driver's, all four variables
	HookEnv     HookEnv             // the BROODFORGE_* names of the hook contract
	Input       string              // the From-prefixed blocks
	Answer      *struct{ ToolUseID, Decision string } // Decision "allow" | "deny"
	Model, Effort string            // per process (ADR 0020)
	SystemPromptAppend string       // one call at a time, trailers, reach
	Spawn       SpawnBackend
}
type TurnHandle struct { Events <-chan EngineEvent; Interrupt func() error }
// A sum type is a struct with a Kind enum and one pointer payload per kind (ADR 0023);
// every switch on Kind is checked by the exhaustive linter.
type EngineEvent struct {
	Kind       EventKind // init | assistant | tool-result | result | exit | unknown
	Init       *struct{ EngineSessionID, Model string; Tools, Capabilities []string }
	Assistant  *struct{ MessageID, Model, ParentToolUseID string; Blocks []Block; Usage Usage } // Block: text | tool-use (ID, Name, Input)
	ToolResult *struct{ ToolUseID string; IsError bool; Text, DecidedBy string } // tool_result_meta source
	Result     *struct{ StopReason string; IsError bool; NumTurns int; Deferred *DeferredTool; PermissionDenials []json.RawMessage; ModelUsage map[string]ModelUsage; Usage json.RawMessage }
	Exit       *struct{ Code *int; Signal string; Interrupted bool }
	Unknown    string    // raw line; logged at debug level, never fatal
}
```

Stream-json subset consumed: `system` subtype `init` (session id, model, tools, `capabilities`);
`assistant` (`message.id`, `message.model`, `message.usage`, text and `tool_use` content,
`parent_tool_use_id`; one line per content block, S1); `user` with `tool_result` content and the
optional `tool_result_meta`; `result` (`stop_reason`, `is_error`, `num_turns`, `deferred_tool_use`,
`permission_denials`, `modelUsage`, `usage`). Everything else is `unknown`, partial messages are
not requested, and a `capabilities` set missing a name the adapter relies on is a refusal at
instance start, exit 1.

Usage delta: `modelUsage` is cumulative per engine session over the turns that reached a `result`
(a killed turn's counts are absent from the next cumulative, S2); the turn's record per model is
the cumulative value minus the last `result` row's cumulative for that `engineSessionId` (never a
sum of rows, never an interrupted row), clamped at zero, with `usage` stored beside it. A turn
with no `result` gets the `assistant` counts deduplicated by `messageId`, output unknown, flagged
interrupted (ADR 0020), its own record outside the baseline. Models that worked are the keys of
`modelUsage`, or the distinct `model` values seen when interrupted.

The command line is ADR 0020's; the stdin message is one stream-json `user` line with one text
block; the system prompt append is per process; `--mcp-config` names `<bf>/share/empty/mcp.json`,
an empty map shipped with Broodforge and reached through the existing `<bf>` bind (ADR 0022
companion). The `fake` replays a fixture directory: `turn-<n>.jsonl` files of verbatim stream-json
lines, an optional per-line delay in `fixture.json`, each `TurnRequest` received written as
`received-<n>.json` for assertions; `interrupt()` ends the current file early with `exit` carrying
`interrupted: true`. A recorded real-engine transcript is a fixture as captured (ADR 0020's spike).

## The PreToolUse hook

The whole contract is `03-hook-contract.md`: how the engine runs the hook and reads its exit
codes (exit 2 blocks, everything else falls through, so the hook catches everything and exits 2),
the `BROODFORGE_*` environment, path canonicalisation and the credential-path deny, the policy
table per reach and mode (reads path-checked; web tools never read class), and the decision
order (reach deny before the pending answer). `policy` holds the table.

## Configuration

JSON, parsed by the runtime, no dependency; a key beginning with `//` is ignored at every level,
which is how the owner comments. Files under `config/`: `instance.json` and one
`projects/<project id>.json` per project. `config check`, `reload` and last-good are ADR
0021's. A project id matches `^[a-z][a-z0-9-]{0,31}$`, is set once and never changed; a rename
is a new file plus `worktree forget` for the old rows.

```json
{ "schemaVersion": 1, "instanceName": "pc",
  "credentials": { "keyFile": "C:/Users/broodforge/AppData/Local/Broodforge/pc.key" },
  "discord": { "mode": "live", "applicationId": "…", "guildId": "…", "botCredentialId": "cred-…", "noticeChannelId": "…" },
  "engine": { "path": "C:/Users/broodforge/.local/bin/claude.exe", "version": "2.1.300",
              "gitBashPath": "C:/Program Files/Git/bin/bash.exe", "fakeFixtures": null },
  "spawn": { "backend": "same-account", "interruptTimeoutSeconds": 20 },
  "paths": { "install": "…" }, "audit": { "fileSink": null },
  "relay": { "loginByDirectMessage": false },
  "roleGroups": { "trusted": ["role id"], "members": ["role id"] },
  "defaults": { "idleTimeoutMinutes": 60, "promptTimeoutMinutes": 5, "mode": "prompt", "harness": "claude-code" } }
```

```json
{ "id": "broodforge", "name": "Broodforge",
  "repo": { "canonical": "<root>/projects/broodforge", "remote": "origin", "defaultBranch": "main",
            "forge": "github", "gitCredentialId": "cred-…" },
  "handoff": "pull-request",
  "venue": { "category": "<category id>", "summaryChannel": null },
  "channels": { "<discord channel id>": {
      "reach": "code", "roles": { "start": ["trusted"], "contribute": ["members"], "answer": ["driver"] },
      "mode": "prompt", "harness": "claude-code", "idleTimeoutMinutes": 60, "promptTimeoutMinutes": 5, "model": null, "effort": null } } }
```

Rules: a role list holds group names from `roleGroups` or raw role ids; `"driver"` under `answer`
means the driver only (R-25). A channel inherits `defaults` for a key it omits. `harness` is
`claude-code` or `fake`; `handoff` is `pull-request` (default) or `push`; `venue.category` is
the category id sessions are created under, `venue.summaryChannel` optional, also receiving each
summary and transcript; `discord.mode` is `live` or `dry-run`; `engine.fakeFixtures` is the
`fake`'s fixture directory. The panel message id is recorded per project channel in the store
when `panel post <channel id>` runs: a project has several channels, and the bot never writes
the owner's config files (ADR 0021). `credentials.keyFile` names the store key (ADR 0022; the default
per host and the `BROODFORGE_KEY_FILE` override are the ADR's). `engine.version` must equal the
version parsed from the binary's `--version` output (the leading dotted number of
`2.1.296 (Claude Code)`), else a refusal naming both. A channel id in two project files is a
`config check` error (R-07). Every credential id named must exist (ADR 0021).

## Git: trees, branches, attribution, handoff

Session id: `s` plus the base-36 session row id, at least four digits (`s0001`), unique per
instance. Branch: `broodforge/<instanceName>/<session id>` (two instances pushing one remote
cannot collide; the instance name matches the project-id pattern); tree:
`sessions/<session id>/tree`; the canonical's ref: `refs/sessions/<session id>` (ADR 0022); the
remote branch equals the local one. Subagent worktrees are the engine's own, inside the clone; only the session branch is
bundled, so subagent work reaches the push once merged or cherry-picked into it (R-31); unmerged
subagent branches are listed in the end-of-session summary and removed with the tree.

Attribution (R-32, R-33). Author and committer of every commit: the driver's registration,
through the four git identity variables (ADR 0022). Trailers, one `Co-Authored-By: <name>
<email>` each, in this order: every other participant who wrote a message before the commit's
committer time, in join order; then every model with a usage row in the session before that
time, by display name with `noreply@anthropic.com`. Models used by subagents are contributors
too: `modelUsage` from the engine's result covers them (ADR 0020, Amendments), so every model with non-zero
usage in the session gets a line (an Opus agent that ran a Haiku subagent yields both). The
display name comes from the map in `policy` (model id prefix to name); an unmapped id is used
verbatim. A model that only compacted is a co-author: the stream cannot tell compaction from
work. Two helps, neither the enforcement: the system prompt append lists the exact trailer
lines, and `worktree-create` sets `core.hooksPath` in the clone to a Broodforge-owned
`commit-msg` hook that appends any line of `sessions/<id>/home/trailers` that is missing.

The enforcement is the bot's check at handoff (ADR 0022 companion, step 3): for each commit in
`<base>..refs/sessions/<id>`, author and committer equal the driver and the trailer set computed
for the commit's time is present (extra trailers are fine). A failing commit is listed with what
is missing, the push is refused, and the session stays open in waiting-for-input so the driver
can ask the agent to amend and `end` again. The bot never rewrites history (ADR 0022 defers it).

End of session: interrupt a running turn; when `handoff` is `pull-request` and the credential is
usable, run one closing engine turn with a fixed prompt that returns the pull request title and
body (skipped on `push` projects, on an auth error, or when the credential is marked needs-login;
the fallback title is `Session <id>: <first message, truncated>`, the summary the body); then
bundle, verify, fetch, check, and hand off per the project's `handoff` (default `pull-request`):
`push` pushes `refs/sessions/<id>` to `refs/heads/<session branch>` on the remote; `pull-request`
pushes the same and opens the request through the forge's HTTP API with the project's git
credential in that one call (`forge: "github"` only in the first version; title and body are a 04
flow). No commits: the tree is removed and the session says so. The tree is removed only after
`done` on the push or the request (R-35); a refused or failed handoff keeps it and `status` lists it with its unpushed count.

## Process model per host

| | Linux droplet | Windows PC |
|---|---|---|
| Bot (`broodforge serve`) | user `broodforge`, systemd unit with `KillMode=mixed` (ADR 0022) | account `broodforge`, Task Scheduler, run whether logged on or not |
| Engine, hook, clone, bundle | user `broodforge-engine` through `sudo -n` and bwrap | the same `broodforge` account, `same-account` backend |
| Nightly backup job | systemd timer as `broodforge` | Task Scheduler as `broodforge` |
| Debug CLI | any shell as `broodforge` | the `broodforge` account; the owner's own account for scratch runs |
| Control socket | `data/control.sock`, directory `0700` | named pipe `\\.\pipe\broodforge-<instanceName>` |

Startup, in order; a failure at any step is a refusal naming the step, exit 1: (1) the instance
root from the flag or environment variable (ADR 0021); (2) `config/` or last-good (ADR 0021),
parsed and validated except for credential ids; (3) the key file the config names or
`BROODFORGE_KEY_FILE`, the credential store, then the config's credential ids against it (ADR
0022); (4) `data/store.sqlite` and migrations; (5) the engine binary and version; (6) `spawn
preflight`; (7) reconciliation (ADR 0021); (8) the control socket; (9) the Discord gateway;
(10) the interrupted notice in each interrupted session's venue and the start-up report to the
notice channel if one is configured.

Signals and leftovers. The spawn backend starts every engine (the `sudo` child on Linux) detached
in its own process group, so a terminal's Ctrl-C reaches the bot alone and the bot interrupts each
turn through the backend; `KillMode=mixed` makes `systemctl stop` do the same (SIGTERM to the bot
only, SIGKILL to the rest of the cgroup after `TimeoutStopSec`). After a bot crash, systemd kills
the leftover engines with the unit; on Windows the bot sits in a kill-on-close job object and each engine in its own, so a crash
or `stop` ends them (ADR 0023). The next start records those turns interrupted.

Shutdown (`SIGTERM`, Ctrl-C, or `stop` over the control socket): stop accepting messages and clicks,
interrupt every running turn through the spawn backend, mark every open session interrupted (R-17;
R-18 is later; `end` still runs the handoff, no new turns), post the interrupted notice where the
gateway allows, close the store, exit 0. After thirty seconds: exit 1, open sessions listed on stderr, reconciliation covers them at the next start.

## Debug surface attachment

Every CLI command but `serve` is a control-socket client; with no socket, read commands open the
store file read-only (ADR 0021) and write commands exit 1 saying the bot is not running. Linux: a
unix socket sessions cannot reach (`data/` is unbound and `0700`). Windows: a named pipe a
same-account session can reach, the gap ADR 0022 states. No TCP listener. A scratch root puts the
socket under its root; the pipe name comes from `instanceName`. 05 specifies the line shapes (`cmd` is `<group>.<command>`) and the error codes.

## Failure modes and what the user sees

The last column names a 04 flow, not its wording.

| Where | Failure | Behaviour | Flow |
|---|---|---|---|
| `discord` | Gateway drops, rate limited | disgo reconnects; events buffer per session and coalesce on return; a buffer over its cap drops only intermediate edits, never a new message (N-25) | none |
| `engine` | Binary missing, version or capability mismatch | refusal at start, exit 1 | operator |
| `engine` | Exit without `result` | turn interrupted, partial usage recorded, exit code kept | turn-failed |
| `engine` | `result` with `is_error` | turn ended as error; an auth error also ends the session through the normal ending sequence (summary, bundle handoff), discarding no work, and sets the credential `needs-login` (an API key keeps its status) | credential-needs-login |
| `hook` | Internal error, exit 2 | the engine blocks that call with the hook's stderr as the result, the turn continues, audit `action refused` from the `hook:` prefix | tool-denied notice |
| `hook` | Timeout | non-blocking: the engine's default decides (C39: Read allowed, Write denied, a safe-listed Bash run); the result's `decidedBy` is `config`, audit `hook bypassed` | hook-bypassed |
| `spawn` | Preflight fails; interrupt timeout | refusal at start naming the sudoers block; tree killed and turn interrupted (ADR 0022) | operator; turn-interrupted |
| `credentials`, `git` | Key file missing or store unreadable; id removed or `needs-login`; clone fails | start refused; session start refused naming the credential or the clone error, no tree recorded | session-refused |
| `git` | Bundle, fetch or attribution check fails | handoff refused, tree kept, session stays open, commits listed | handoff-refused |
| `git` | Push rejected, request API error | audit `failed`, tree kept, `status` lists it, retry by `worktree handoff` | handoff-failed |
| `store` | Locked, corrupt, schema newer than the binary | refusal at start, exit 1 (ADR 0021) | operator |
| `config` | Invalid at start or reload | last-good or refusal (ADR 0021) | notice channel |
| `session` | Idle timeout | the end-of-session sequence; a refused handoff keeps the tree | session-ended |
| `backup` | Snapshot or rehearsal fails | the nightly job's exit code; `status` shows the last good snapshot's age | notice channel |
