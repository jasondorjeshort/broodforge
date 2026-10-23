# The PreToolUse hook contract

Companion to `03-architecture.md` (its hook section points here); the commands that drive the
hook from a terminal are `05-debug-interface.md`'s. This is the living spec of what the hook
receives, how it decides and what it returns. The policy table below is one object in `policy`;
changing a cell is a design change recorded here, never configuration. A fact marked with a
claim id was observed on engine 2.1.296 on the PC and is recorded in
`../findings/2026-10-15-spike-login-claims-for-adr-0020.md`,
`../findings/2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`,
`../findings/2026-10-15-spike-hook-flow-claims-for-adr-0020.md` and `../findings/2026-10-15-spike-claims-for-adr-0022.md`;
the amendments of 2026-10-16 in ADRs 0020 and 0022 are the decisions this text applies.

## How the engine runs it

The engine runs `<install>/bin/broodforge hook pre-tool-use` for every tool call, a subagent's
included (the engine's documented behaviour; no spike fixture holds a subagent launch call, which the pinned engine names `Agent`; `Task` is kept for an older pin), registered with an empty matcher and a `timeout` of thirty seconds (ten before C39;
the hook does no I/O beyond stdin, the environment and one log line, so the bound is never
reached by a healthy hook and the engine's fallback below is reached less often) in the bot-owned
settings file that ADR 0022 (Decision) places on the engine's settings path, read-only to the
session, for both credential kinds; the hook runs as `broodforge-engine` inside the sandbox on the
droplet, as the instance account on the PC. It has no channel to the bot: nothing under `data/` is
bound into the sandbox, and ADR 0020 made the decision a function of the environment. It reads
the engine's JSON on stdin (`tool_name`, `tool_input`, `tool_use_id`, `cwd`, and
`transcript_path`, which the engine documents for every hook event and the spike hook did not log:
its absence is a hook error, below, so the first real turn proves it), prints the engine's hook
output with `permissionDecision` `allow`, `deny` or `defer` and a reason, and exits 0.

Failure, and the engine contract this depends on (C39, held). The engine blocks a call only when
the hook exits 2, and hands the model the hook's stderr wrapped as `PreToolUse:<tool> hook error:
[<hook command>]: <stderr>`, the trailing newline kept; a `deny` reason arrives wrapped the same
way without the bracketed command (C27). Any other non-zero exit, and a hook that runs past its
`timeout`, is non-blocking: the call falls through to the engine's own default permission mode as
if no hook were registered. In `-p` that default, observed with no `control_request` (so
`--permission-prompts none` is not on the command line), allows a `Read`, denies a `Write` with no
prompt (a `system/permission_denied` line, then an error result `Claude requested permissions to
write to <path>, but you haven't granted it yet.`) and allows and runs a `Bash` command on the
engine's own safe list (`echo ok` ran). So the hook wraps everything, malformed stdin, a missing
variable and a missing `transcript_path` included, in one handler that prints `hook: <what>` on
stderr and exits 2: it fails closed on every error it can see. The timeout is the one failure it
cannot catch and it is a real hole: a safe-listed `Bash` runs in read-only reach, and in prompt
mode with nobody asked. The stream marks every such call (next paragraph), so it is audited, not
silent; ADR 0020's C39 amendment weighs this and keeps the design.

What the bot sees. The hook writes nothing to the store; the stream is the record (ADR 0020).
`session` writes a `permission decision` row per `tool-use` block seen. The decider is the result
line's `tool_result_meta[].permission_decision.source`: `hook` when the hook answered (allow,
deny or an answered defer), `config` when the engine's default decided (a timeout, or a non-2
exit the engine ignored), `unknown` when the field is absent, since the engine does not document
it; the adapter's fixture tests assert the field on every recorded result, so an engine pin that
drops it fails the fixture check instead of silencing the audit below. A `tool-result` whose text
begins `<tool_use_error>` is the tool's own error (the engine refuses a standalone `sleep` in
Bash before any hook runs, S2) and gets no decision row. Any other error result is classified by
its text after stripping `PreToolUse:<tool> hook error: `, then an optional `[<command>]: `, then
surrounding whitespace: `Denied by Broodforge:` is a policy deny; `hook:` is a hook failure,
recorded as `action refused` with actor `hook` and shown as 04's tool-denied notice. A
contains-match is never used: the rest of the text is the model's. A result whose decider is
`config`, or an error result with decider `unknown` and neither prefix, is the hook bypassed:
audited `hook bypassed` with the tool, the outcome (ran, or refused by the engine), the cause
(timed out, or the hook failed) and the text, shown as 04's hook-bypassed flow, the turn
continuing. `result.permission_denials` lists hook and engine denies alike.
`sessions/<id>/home/hook.log` (one line per decision, debugging only, never audit) shows which
calls the hook answered.

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
| `BROODFORGE_ANSWER` | `<tool_use_id>:allow` or `<tool_use_id>:deny` on a resume after a click or a prompt timeout; else absent |
| `BROODFORGE_ANSWER_REASON` | after a prompt timeout only: the deny reason text, written by `session` (04's `the prompt timed out after 5 minutes`) and returned by the hook verbatim; else absent |
| `BROODFORGE_SESSION_ALLOW` | comma-separated tool names a person allowed for the session; absent unless 04 adopts that button |
| `BROODFORGE_HOOK_LOG` | `sessions/<id>/home/hook.log`, one line per decision, for debugging only, never audit |
| `BROODFORGE_WORKFLOW` | absolute path of the session's copy, `workflow/<id>` (ADR 0025 rule 5); absent when nothing was copied or `inject` is `none`: the hook then runs no child (The child chain) |

Under `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` the engine strips `CLAUDE_CONFIG_DIR` from the hook as
it does from Bash (C34), so the hook never sees the config directory by name and no `BROODFORGE_*`
name carries it either (a session's Bash would read it). Every `BROODFORGE_*` name above survives
the scrub. The engine adds `CLAUDE_PROJECT_DIR` (the cwd) and its `CLAUDE_CODE_*` names to the
hook; the hook reads none of them: `BROODFORGE_TREE` is the tree.

## Paths

Every path in `tool_input` (`file_path`, `path`, `notebook_path`; a search with no path means
`cwd`; `Glob`'s `pattern` up to the last separator before its first `*?[{`, per brace alternative,
joined to its search root when relative; a pattern with a `..` segment, a segment matching `..`
without a wildcard, a `{`/`[` group holding a separator, or a `:` after a drive is a hook error)
is first expanded as the engine expands it (engine 2.1.296, its `file_path` function): whitespace
trimmed as JavaScript's `trim()` trims (U+FEFF included, U+200B not), `~`, `~/` and `~\` to the home directory, on a host with drive letters `/<letter>/` to
`<LETTER>:\`; `~user` is a hook error. A relative path resolves against the hook input's `cwd`, as
the engine resolves it, else `BROODFORGE_TREE`; then it is canonicalised: links (symlinks,
and Windows junctions) are followed on the longest prefix that exists, so a link planted in the tree
cannot point a check outside it, twice: `..` applied as text first, as Windows opens a path, and
after the link before it, as POSIX does; the stricter answer wins. A spelling the checks cannot
compare is a hook error: a device or network volume (`\\?\`, `\\.\`, UNC), a drive-relative path,
any `:` after the volume (an NTFS stream). "Under the tree" and "under the canonical" are prefix
tests on the canonical form or, for another spelling of an existing directory (an 8.3 name), the
same file (`os.SameFile`) on an ancestor; a base name is the one its directory lists. Credential paths are
denied to every class in both reaches and both modes, with the reason `Denied by Broodforge: <path>
is a credential file`: any file named `.credentials.json` (any case, trailing dots and spaces
dropped), a hard link to any `.credentials.json` or `.claude.json` the hook can name (the config
directory's, and every `<root>/harness/<id>/` one when ADR 0022's layout yields the root from the
config directory; a hard link elsewhere passes), a `Grep` or `Glob` rooted above the config directory, and
anything under the config directory, which the hook derives from `transcript_path` on stdin
because `CLAUDE_CONFIG_DIR` is scrubbed (C34): the parent of the nearest ancestor directory named
`projects` (the engine writes `<cfg>/projects/<encoded cwd>/<session>.jsonl` for both credential
kinds; a subagent's transcript is expected beneath that, unobserved), canonicalised the same way; no `transcript_path`, or
no such ancestor, is a hook error (`hook: no transcript_path`, exit 2). This is the hook's line; on
the droplet the binds and the engine's sandbox are the boundary, on the PC a session's Bash reads
the same files (ADR 0022, both stated).

## Policy table

Cells: `allow` is both modes; `ask` is policy mode `allow`, prompt mode `allow` for a tool named
in `BROODFORGE_SESSION_ALLOW` and `defer` for the rest; `web` is prompt mode `defer`, policy mode
`deny`, because no reach allows the network in the first version (ADR 0022 defers egress control;
the engine's own default mode asks for `WebFetch` too). `deny` uses ADR 0020's text.

| Tool class | read-only reach | code reach |
|---|---|---|
| `Read`, `Glob`, `Grep`, `LS`, `NotebookRead` | `allow` under the tree or the canonical, else `deny` | `allow` under the tree or the canonical; elsewhere `allow` too, since Bash reads it anyway, except the credential paths above |
| `TodoWrite`, `Agent` (`Task` on an older engine pin) | `allow` (a subagent's own calls fire the hook) | `allow` |
| `WebFetch`, `WebSearch` | `web` | `web` |
| `Edit`, `MultiEdit`, `Write`, `NotebookEdit` | `deny` | `ask` when the resolved path is under the tree, else `deny` |
| `Bash` | `deny` | `ask` (the host is the boundary, ADR 0022; the PC's gap stands) |
| anything else, `mcp__*` included | `deny` | `deny` |

## Decision

In order: (1) the cell, after the path checks, is `deny`: deny; an answer cannot lift a reach
deny, so the reach is checked first. (2) `BROODFORGE_ANSWER` names this `tool_use_id`: return
that decision. (3) The cell is `allow`: allow. (4) The cell is `ask` or `web`: by mode as above.
Parallel calls in prompt mode (C28, observed for a deferred pair; a message mixing allowed and
deferred calls is unobserved): the engine runs the hook once per call, in order; when it has
deferred two or more calls of one message it ends the turn naming only the last in
`deferred_tool_use` and drops the rest, which the model re-issues with new ids after the resume.
The hook has no rule for this: the engine's documented hook input (session id, transcript path,
cwd, tool name, input and id) carries no message id to decide by, and a deny of the later calls
would be re-issued the same way at the same cost; `session` records the dropped calls
(`03-architecture.md`, prompt mode). `session reach-report` prints the table's column for the
session's reach beside the bind list the spawn uses.

## The child chain

ADR 0025 rule 2; the texts and lifts are its companion's (`decisions/0025-refusals-and-checks.md`),
built in `internal/hook/chain.go` and `post.go`. The settings file registers a second entry point
beside the first, `<install>/bin/broodforge hook post-tool-use`, empty matcher, thirty seconds.

When. In code reach, after the Decision above when it is not `deny`, and only when
`BROODFORGE_WORKFLOW` is set and `$BROODFORGE_WORKFLOW/hooks.json` exists; a variable naming no
directory, or a `hooks.json` that does not read or parse, is a hook error. `post-tool-use` runs under the same three conditions.
Which. The entries of the hook's own event, in file order, whose `matcher` matches `tool_name` as
the engine's hooks documentation reads it: empty or `*` matches every tool, anything else is a
case-sensitive regular expression the whole name must match; an invalid one matches nothing and
is logged. How. Serially, each through `$CLAUDE_CODE_GIT_BASH_PATH -c <command>` on a host with
drive letters (the spawn exports it from `engine.gitBashPath`; unset is a cannot-start deny naming
it) and `/bin/sh -c <command>` elsewhere; cwd `BROODFORGE_TREE`; environment the hook's own with
`CLAUDE_PROJECT_DIR` the copy's absolute path; stdin the engine's bytes unchanged. Budget: the
smaller of the entry's `timeout` and ten seconds per child, the rest of twenty for the chain, then
the child is killed. What is taken, `pre-tool-use`: exit 2 (reason: the first stderr line) or an
exit-0 `hookSpecificOutput.permissionDecision` of `deny` (reason: the first line of its
`permissionDecisionReason`), either reason cut at 1000 bytes; an exit with a background job still
holding the pipes is judged by its code once they are closed;
the first such child ends the chain, and the hook returns `deny` with the companion's child-deny
text, `Denied by Broodforge: refused by the project's hook <name>: <reason>. ...`, so What the
bot sees needs no new class. A child over its own budget, or one that cannot start (the shell will
not start, or the shell exits 126 or 127: reason the first stderr line, else the OS error), is a
deny with the companion's over-budget text (`<n>` its own budget) or cannot-start text; a child the
chain's twenty seconds stopped, or never let start, is `Denied by Broodforge: the project's hook
<name> was stopped|did not run: the project's hooks exceeded twenty seconds together before <name>
finished|ran.` Every other exit, `allow`, `ask`,
`defer`, `updatedInput` and any context is dropped: the hook's decision stands unchanged.
`post-tool-use`: every matching child runs; an exit-0 `additionalContext` and an exit-2 child's
stderr are joined by newlines into its own `{"hookSpecificOutput":{"hookEventName":"PostToolUse",
"additionalContext":...}}`, nothing printed when there is none. It exits 0 on every path, an error
a `hook: <what>` line on stderr, since a `PostToolUse` hook cannot block.
Log. One `hook.log` line per child run or skipped matcher: `event`, `child` (the entry's `name`),
`exit`, `ms`, `outcome` (`passed`, `deny`, `over-budget`, `over-chain-budget`, `not-started`, `context`, `skipped`),
`detail` and `dropped` (the field names not taken, `permissionDecision=<value>` for a decision);
the decision line of a child deny carries `trace`, `denied: refused by the project's hook <name>:
<reason>`. A child's reason line is the one part of its output logged, in `detail` and `trace`; its
other output and the tool input never are.
