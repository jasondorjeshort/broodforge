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
