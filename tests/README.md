# tests/

Scripts and data for tests that span more than one package. A unit test lives beside its code
as `_test.go` and uses `testing` only; recorded engine streams and other replayed inputs live
under `fixtures/` at the repository root [no-check: created by a later pass]. A script here runs
from the repository root, takes the binary as `bin/broodforge` (`bin/broodforge.exe` on Windows)
built by `CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`, and works against a scratch
root (`docs/design/05-cli-conventions.md`, Scratch), never a live instance.

One line per file below, added with the file.

| File | What it covers |
|---|---|
| `config/valid/` | A valid instance and project pair with `//` comments at several levels; channel `100` inherits every default, `200` overrides them (`config check` exit 0, `config show --effective`) |
| `config/dup-channel/` | Channel `100` in two project files, R-07 (`config check` exit 1 naming both files) |
| `config/unknown-key/` | A channel key that is not 03's, `idleTimeout` (`config check` exit 1 naming the key path) |
| `config/dup-key/` | Channel `100` twice in one project file (`config check` exit 1, `duplicate key channels.100`) |
| `hook/cases.jsonl` | Every cell of `docs/design/03-hook-contract.md`'s policy table in both modes and reaches, answers and the session allow list, hook errors, credential paths (`.credentials.json`, the config directory `transcript` implies, a `Grep` above it), and links out of the tree, one host each through `needs` (`hook cases tests/hook/cases.jsonl`: every line `ok` or `skip`) |
| `hook/linkworld/` | The tree, canonical, outside and config directories `hook/cases.jsonl` resolves against; long names so Windows keeps 8.3 aliases for the short-name cases; its `.credentials.json` files hold no secret, only the name |
| `repro/full-session.sh` | One session against the fake on a scratch root, R-51 (`bin/broodforge scratch run -- bash tests/repro/full-session.sh`, exit 0; node on `PATH`): `fixtures/workflow/`'s files committed to the canonical and both hook events listed, then the request's `--append-system-prompt-file` and `--agents`, `session show`'s `workflow:`, `hooks:` and `review:` lines and `--header` (ADR 0025); `serve` in the background, `register`, a start refused for roles, a policy-mode turn against `fixtures/engine/basic`, its `session tail` and `audit` lines checked, its `session outbox` (venue, header, working message, the final message's footer), `session reach-report`, `usage`, `config reload`, `worktree handoff s0001`, `stop`; the bot's output is `serve.log` under a kept root |
| `repro/defer-resume.sh` | Prompt mode on a scratch root (`bin/broodforge scratch run -- bash tests/repro/defer-resume.sh`, exit 0): `fixtures/engine/defer` waits for an answer (`session show`'s `pending:`), an answer refused for roles, the driver's allow, then a 3 s prompt timeout's deny in `session tail` and `audit`; `session outbox` shows the prompt with its three custom ids, `Allowed by`, `No answer; denied` and the timeout trace line; `fixtures/engine/parallel` leaves one `dropped` call, then `session interrupt` denies its waiting call as `interrupted`; `session end` hands off nothing |
| `discord/burst.jsonl` | Two sessions' turns as `session tail --json` lines with a `session` key: 24 trace events in 3 s for `s0001`, a prompt answered after 5 s and a pasted key for `s0002` (`bin/broodforge discord render --events tests/discord/burst.jsonl --clock fake --expect 'edits<=1/s'`, exit 0; the key shows `[masked]`) |
