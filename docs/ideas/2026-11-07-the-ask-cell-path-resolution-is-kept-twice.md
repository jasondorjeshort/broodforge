---
date: 2026-11-07
kind: chore
---

# The ask cell's path resolution is kept twice

Not sanity checked.

P39 added `internal/session/askpath.go`, a hand-kept copy of the hook's path resolution
(`internal/hook/paths.go`, `facts.go`: `~` and `/<letter>/` expansion, spelling checks, links
followed both ways, under-tree by text or by the same file). `session` needs it so that
`AskCell` agrees with the hook on which calls sit in the `ask` cell, and the import graph of
`tests/lint/lint_test.go` (one row per line of `03-architecture.md`'s Components table) forbids
`session` importing `hook`. The P39 reviewer showed what a looser copy costs: a Write through a
junction out of the tree audited as an `engine` decision instead of `hook bypassed`. Two copies
drift the same way the next time the hook's resolution changes. The copy leaves out the hook's
config-directory and hard-link credential checks (the hook denies those first, so they never reach
the engine). The alternative the coder named: move the resolution into `policy`, which both
packages already import, with one set of tests; the Components table row and the lint graph then
say so.
