# The PreToolUse hook contract

Companion to `03-architecture.md` (its hook section points here); the commands that drive the
hook from a terminal are `05-debug-interface.md`'s. This is the living spec of what the hook
receives, how it decides and what it returns. The policy table below is one object in `policy`;
changing a cell is a design change recorded here, never configuration.

## How the engine runs it

The engine runs `<install>/bin/broodforge hook pre-tool-use` for every tool call, registered with
an empty matcher and a `timeout` of ten seconds in the bot-owned settings file that ADR 0022
(Decision) places on the engine's settings path, read-only to the session, for both credential
kinds; the hook runs as `broodforge-engine` inside the sandbox on the droplet, as the instance
account on the PC. It has no channel to the bot: nothing under `data/` is bound into the sandbox,
and ADR 0020 made the decision a function of the environment. It reads the engine's JSON on
stdin (`tool_name`, `tool_input`, `tool_use_id`, `cwd`), prints the engine's hook output with
`permissionDecision` `allow`, `deny` or `defer` and a reason, and exits 0.

Failure, and the engine contract this depends on. The engine blocks a call only when the hook
exits 2, and feeds the hook's stderr to the model as the tool result; any other non-zero exit,
and a hook that runs past its timeout, is non-blocking: the call falls through to the engine's
own default permission mode as if no hook were registered. So the hook wraps everything,
malformed stdin and a missing variable included, in one handler that prints `hook: <what>` on
stderr and exits 2: it fails closed on every error it can see. The timeout is the one failure it
cannot catch; the hook does no I/O beyond stdin, the environment and one log line, and ten
seconds is generous. What the engine's default does in `-p` mode when no hook answers, and what
the stream shows for each kind of failure, is spike claim C39 (ADR 0020).

What the bot sees. The hook writes nothing to the store; the stream is the record (ADR 0020).
`session` writes a `permission decision` row per `tool-use` block seen and classifies an error
`tool-result` by its text prefix: `Denied by Broodforge:` is a policy deny; `hook:` is a hook
failure, recorded as `action refused` with actor `hook` and shown as 04's tool-denied notice. A
timeout leaves no prefix in the stream: the row records what the stream shows, and
`sessions/<id>/home/hook.log` (one line per decision, debugging only, never audit) shows which
calls the hook answered, so a call with a result and no log line was decided by the engine alone.

## Environment

Set by `spawn` on the engine process and inherited by the hook; the kept list in
`decisions/0022-linux-spawn-and-handoff.md` is generated from the same array:

| Name | Value |
|---|---|
| `BROODFORGE_SESSION` | the session id |
| `BROODFORGE_MODE` | `policy` or `prompt` |
| `BROODFORGE_REACH` | `read-only` or `code` |
| `BROODFORGE_TREE` | absolute path of `sessions/<id>/tree`; tool-input paths are resolved against it |
| `BROODFORGE_CANONICAL` | absolute path of the project's canonical checkout, read-only inside the session |
| `BROODFORGE_ANSWER` | `<tool_use_id>:allow` or `<tool_use_id>:deny` on a resume after a click; else absent |
| `BROODFORGE_SESSION_ALLOW` | comma-separated tool names a person allowed for the session; absent unless 04 adopts that button |
| `BROODFORGE_HOOK_LOG` | `sessions/<id>/home/hook.log`, one line per decision, for debugging only, never audit |

## Paths

Every path in `tool_input` (`file_path`, `path`, `notebook_path`; a search with no path means
`cwd`) is resolved against `BROODFORGE_TREE`, then canonicalised: symlinks are followed on the
longest prefix that exists, so a link planted in the tree cannot point a check outside it. "Under
the tree" and "under the canonical" are prefix tests on the canonical form. Credential paths are
denied to every class in both reaches and both modes: any file named `.credentials.json`, and
anything under the config directory (`CLAUDE_CONFIG_DIR` when set, else `$HOME/.claude`). This
is the hook's line; on the droplet the binds and the engine's sandbox are the boundary, on the
PC a session's Bash reads the same files (ADR 0022, both stated).

## Policy table

Cells: `allow` is both modes; `ask` is policy mode `allow`, prompt mode `allow` for a tool named
in `BROODFORGE_SESSION_ALLOW` and `defer` for the rest; `web` is prompt mode `defer`, policy mode
`deny`, because no reach allows the network in the first version (ADR 0022 defers egress control;
the engine's own default mode asks for `WebFetch` too). `deny` uses ADR 0020's text.

| Tool class | read-only reach | code reach |
|---|---|---|
| `Read`, `Glob`, `Grep`, `LS`, `NotebookRead` | `allow` under the tree or the canonical, else `deny` | `allow` under the tree or the canonical; elsewhere `allow` too, since Bash reads it anyway, except the credential paths above |
| `TodoWrite`, `Task` | `allow` (a subagent's own calls fire the hook) | `allow` |
| `WebFetch`, `WebSearch` | `web` | `web` |
| `Edit`, `MultiEdit`, `Write`, `NotebookEdit` | `deny` | `ask` when the resolved path is under the tree, else `deny` |
| `Bash` | `deny` | `ask` (the host is the boundary, ADR 0022; the PC's gap stands) |
| anything else, `mcp__*` included | `deny` | `deny` |

## Decision

In order: (1) the cell, after the path checks, is `deny`: deny; an answer cannot lift a reach
deny, so the reach is checked first. (2) `BROODFORGE_ANSWER` names this `tool_use_id`: return
that decision. (3) The cell is `allow`: allow. (4) The cell is `ask` or `web`: by mode as above.
Parallel calls in prompt mode are ADR 0020's. `session reach-report` prints the table's column
for the session's reach beside the bind list the spawn uses.
