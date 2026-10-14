# Debug CLI conventions: surface, wire format, output, exit codes, scratch

Companion to `05-debug-interface.md`, which points here from its first section. Every command
in 05 follows these conventions; a section there writes out only what differs. The hook contract
is `03-hook-contract.md` and the spawn is the ADR 0022 companion; this file is the CLI side only.

Surface. One binary, `broodforge`, the same entry point as the bot (ADR 0020); every command but
`serve`, `hook`, `engine-entry` and the ones marked `(file)` is a client of the control socket
(03, Debug surface attachment). Invocation: `broodforge [--root <path>] [--json] <group>
<command> [args]`. `--root` or `BROODFORGE_ROOT` names the instance root; neither set, exit 2.
Socket: `<root>/data/control.sock` on Linux, `\\.\pipe\broodforge-<instanceName>` on Windows,
always from `instanceName` (a scratch root's is `scratch-<12 hex>`, generated at `scratch new`,
so no path is hashed). No socket: a `(file)` command and every read command open
`data/store.sqlite` read-only (ADR 0021); a write command exits 1 with `error: not-running: no
bot at <socket>`.

Wire format, both directions one JSON object per line, UTF-8, newline-terminated. Request:
`{"id":<n>,"cmd":"<group>.<command>","args":{...}}`. Final response: `{"id":<n>,"ok":true,
"result":<value>}` or `{"id":<n>,"ok":false,"error":{"code":"<kebab-case>","message":"<one
line>"}}`. A streaming command (`tail`, `audit --follow`, `send --tail`) sends any number of
`{"id":<n>,"event":{...}}` lines before its final response. `cmd` is the CLI name with the group
as prefix; `args` are the flags, dashes removed, camel-cased. The socket authenticates nothing
(ADR 0022 states the Windows gap); 05 adds no TCP and no HTTP.

Output. Text mode is one record per line, fixed column order, columns separated by two spaces,
no header; `--json` prints one object for a single result and one object per line (JSONL) for a
list. Short by default (N-10): lists stop at 20 rows unless `--limit <n>` or `--all`; detail is
behind `--verbose` or a `show` command. Times are UTC ISO 8601. Every error is one stderr line,
`error: <code>: <message>`; with `--json` the wire error object on stderr instead. A code is one
kebab-case token, a subcode joined by a hyphen (`handoff-refused-attribution`), never a colon.

Exit codes (N-07): `0` done; `1` the command ran and the answer is a refusal or failure, reason
on stderr; `2` usage: unknown command or flag, missing root, malformed stdin. `--expect <value>`,
where offered, compares `<value>` with the command's outcome token: on success the first column
of its first output line (`status`: the `config:` value; `hook try`: the decision), on a refusal
the error code. A match exits 0 even when the outcome is a refusal; a mismatch exits 1 with
`error: expect: wanted <value>, got <actual>`.

Scratch (N-09, ADR 0021). `broodforge scratch new [--keep] [--project <path or url>]` creates a
root in the OS temp directory, prints its path (`--json`: `{"root":"...","instanceName":
"scratch-<12 hex>"}`); later commands take `--root <path>`. The generated root holds: config with
`discord.mode: "dry-run"`, `defaults.harness: "fake"`, `engine.fakeFixtures` naming the
repository's `fixtures/engine/` [no-check: no code yet], `spawn.backend: "same-account"`,
`credentials.keyFile` under the root (`BROODFORGE_KEY_FILE` is ignored for a scratch root, so a
scratch run never touches the owner's key); a throwaway project `scratch` cloned from
`--project` (default: generated) with a local bare `origin` under the root so `push` works
offline; channel `100` (reach `code`, prompt mode) and channel `200` (reach `read-only`, policy
mode), both with `start`, `contribute` and `answer` for role `10` (group `trusted`); credential
`cred-scratch` of kind `api-key` whose value is the fixed string `scratch`, the fake's own; no
registrations. `broodforge scratch run -- <command...>` creates a root, exports
`BROODFORGE_ROOT`, runs the command, deletes the root on exit 0, otherwise keeps it and prints
the path on stderr. `scratch delete <root>` and `scratch prune --older-than <days>` remove roots
through the worktree-removal interface and refuse (`error: not-scratch`) a root that is not under
the OS temp directory or whose `instanceName` does not start with `scratch-`. The scratch form of
every section is the same commands with `--root <scratch root>`; only a difference is written out.

Isolation, shared. The surface reaches the bot process only: a session on Linux has no `data/`,
so no socket; the hook has no socket client. Past the one-time print at `credential init`, no
command prints a credential value, a config-directory file, the store key or a file under
`credentials/`; a store reference prints as its id only. A debug flag never widens a live
session: supplied roles and overrides of reach, fixture or timeouts are `error: live-instance`
when the instance's `discord.mode` is `live`; `--mode` is the driver's own switch and stays.
