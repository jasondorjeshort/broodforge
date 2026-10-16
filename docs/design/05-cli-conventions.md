# Debug CLI conventions: surface, wire format, output, exit codes, scratch

Companion to `05-debug-interface.md`, which points here from its first section. Every command
in 05 follows these conventions; a section there writes out only what differs. The hook contract
is `03-hook-contract.md` and the spawn is the ADR 0022 companion; this file is the CLI side only.

Surface. One binary, `broodforge`, the same entry point as the bot (ADR 0020); every command but
`serve`, `hook`, `engine-entry` and the ones marked `(file)` is a client of the control socket
(03, Debug surface attachment). Invocation: `broodforge [--root <path>] [--json] <group>
<command> [flags] [args]`; `--root` and `--json` may also stand among the command's flags, which
precede its arguments. `--root` or `BROODFORGE_ROOT` names the instance root; neither set, exit 2.
Socket: `<root>/data/control.sock` on Linux, `\\.\pipe\broodforge-<instanceName>` on Windows,
always from `instanceName` (a scratch root's is `scratch-<12 hex>`, generated at `scratch new`,
so no path is hashed; the client reads only that key of `config/instance.json`, then of
`data/config-last-good/instance.json` so a bot started on its last-good copy stays reachable,
`error: no-config` or `error: bad-config` naming the first file without it). No socket: a
`(file)` command and every read command open `data/store.sqlite` read-only (ADR 0021); a write
command exits 1 with `error: not-running: no bot at <socket>`.

Wire format, both directions one JSON object per line, UTF-8, newline-terminated. Request:
`{"id":<n>,"cmd":"<group>.<command>","args":{...}}`. Final response: `{"id":<n>,"ok":true,
"result":<value>}` or `{"id":<n>,"ok":false,"error":{"code":"<kebab-case>","message":"<one
line>"}}`. A streaming command (`tail`, `audit --follow`, `send --tail`) sends any number of
`{"id":<n>,"event":{...}}` lines before its final response. `cmd` is the CLI name with the group
as prefix; `args` are the flags, dashes removed, camel-cased, every declared flag sent with its
value or default. A connection may carry several ids at once; its end cancels its calls. The
server's own codes: `bad-request` (with id 0 when the line is not JSON), `unknown-command`,
`internal`; the client's: `not-running` (nothing listens), `connect-failed` (something does but
the connection is refused: access denied, every pipe instance busy for 2 s), `connection-lost`.
The socket authenticates nothing (ADR 0022 states the Windows gap); 05 adds no TCP and no HTTP.

Output. Text mode is one record per line, fixed column order, columns separated by two spaces,
no header; `--json` prints one object for a single result and one object per line (JSONL) for a
list. Short by default (N-10): lists stop at 20 rows unless `--limit <n>` or `--all`; detail is
behind `--verbose` or a `show` command. Times are UTC ISO 8601. A text column is a string
unquoted with `\`, newline, CR and tab escaped, `""` for an empty one, `-` for null or absent,
compact JSON otherwise; events print as records before the result. Every error is one stderr
line, `error: <code>: <message>`; with `--json` the wire error object on stderr instead. A code is one
kebab-case token, a subcode joined by a hyphen (`handoff-refused-attribution`), never a colon.

Exit codes (N-07): `0` done; `1` the command ran and the answer is a refusal or failure, reason
on stderr; `2` usage: unknown command or flag, missing root, malformed stdin. `--expect <value>`,
where offered, compares `<value>` with the command's outcome token: on success the first column
of its first output line (`status`: the `config:` value; `hook try`: the decision), on a refusal
the error code. A match exits 0 even when the outcome is a refusal; a mismatch exits 1 with
`error: expect: wanted <value>, got <actual>`. With `--expect` that line replaces the refusal's
own, and a matched refusal prints none. A usage error is `error: usage: <message>`, never
compared with `--expect`.

Probe and harness. Every server answers the hidden `broodforge control echo [--text <s>]
[--events <n>] [--fail <code>] [--expect <value>]`: `<n>` events `{"seq":i,"text":<s>}` (text
`<s>  <i>`), then the result `{"text":<s>}` (`<s>` defaults to `ok`), or after the events the
refusal `error: <code>: echo was asked to fail`; `<n>` above 1000 or a `<code>` that is not
kebab-case is `error: bad-args`. Until `serve` exists, tests drive the binary
through `internal/control/controltest` (a server on a temporary root whose
`config/instance.json` names `test-<12 hex>`) and `internal/cli/clitest` (builds the binary
with the plan's build line once per test package, into a temporary directory, and runs it);
`go test -v -run TestProof ./internal/cli/` shows each command and its exit code.

Scratch (N-09, ADR 0021). `broodforge scratch new [--project <path or url>]` creates a
root `<temp>/scratch-<12 hex>`, prints its path (`--json`: `{"root":"...","instanceName":
"scratch-<12 hex>"}`); later commands take `--root <path>`. The generated root holds: config with
`discord.mode: "dry-run"`, `defaults.harness: "fake"`, `engine.fakeFixtures` naming the
repository's `fixtures/engine/` [no-check: no code yet], `spawn.backend: "same-account"`,
`credentials.keyFile` under the root (`BROODFORGE_KEY_FILE` is ignored for a scratch root, so a
scratch run never touches the owner's key); a throwaway project `scratch` at `projects/scratch`,
cloned from `--project` (default: generated, one commit on `main`) with a local bare `origin` at
`remote/scratch.git` so `push` works offline; channel `100` (reach `code`, prompt mode) and channel `200` (reach `read-only`, policy
mode), both with `start`, `contribute` and `answer` for role `10` (group `trusted`); credential
`cred-scratch` of kind `api-key` whose value is the fixed string `scratch`, the fake's own; no
registrations.
`broodforge scratch run [--keep] [--project <path or url>] -- <command...>` creates a root, exports
`BROODFORGE_ROOT`, runs the command (a first word `broodforge` is the running binary), deletes the
root on exit 0 unless `--keep`, otherwise keeps it and prints `scratch root kept: <path>` on
stderr; the exit code is the command's. `scratch delete <path>` (`removed  <root>`) and `scratch
prune [--older-than <days>] [--all]` (default 1 day, aged by `config/instance.json`'s modification
time; `removed  <root>` or `refused  <root>  <why>` each, exit 0) remove roots, every
`sessions/<id>` through the worktree-removal interface first, and refuse (`error: not-scratch`) a
root that is not directly under the OS temp directory, is a link, is not named `scratch-<12 hex>`
or whose `instanceName` (read from `config/instance.json` only after the place is checked, with no
last-good recovery) is not that name; a link under `prune` is `refused  <root>  link`. The final
removal of the root runs only after every entry under `sessions/` went through the guard (a link
there, or `sessions/` itself a link, is refused) and the root was verified not to be a link. The scratch form of
every section is the same commands with `--root <scratch root>`; only a difference is written out.

Isolation, shared. The surface reaches the bot process only: a session on Linux has no `data/`,
so no socket; the hook has no socket client. Past the one-time print at `credential init`, no
command prints a credential value, a config-directory file, the store key or a file under
`credentials/`; a store reference prints as its id only. A debug flag never widens a live
session: supplied roles and overrides of reach, fixture or timeouts are `error: live-instance`
when the instance's `discord.mode` is `live`; `--mode` is the driver's own switch and stays.
