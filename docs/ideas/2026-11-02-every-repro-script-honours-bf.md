---
date: 2026-11-02
kind: chore
related: [2026-11-02-redeploy-cannot-replace-the-binary-under-the-running-bot.md]
---

# Every repro script honours `BF`

Not sanity checked.

`tests/repro/full-session.sh` hard-codes the binary under `bin/`; the newer scripts (`restart-end.sh`,
`end-wip-discard.sh`, `stale-canonical.sh`, `ended-category.sh`) take `BF` naming another binary,
and `tests/README.md` says so row by row. With the dev-live bot holding `bin/broodforge.exe` open,
the P34 coder and reviewer each copied the whole tree to a temp directory to run the proof, twice
over, because one script could not be pointed at a temp build.

Anchor: `tests/repro/full-session.sh` (the binary path); `tests/README.md` (its row gains "`BF`
names another binary"); `land.sh` and any later script start from the `BF` form.

## Open

- A shared `tests/repro/lib.sh` for the `BF` default, the `--expect` idiom and the kept-root
  message, or the five-line preamble copied once more. [no-check: the shared file does not exist yet]
