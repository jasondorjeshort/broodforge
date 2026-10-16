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
