---
date: 2026-11-01
kind: bug
---

# `session reach-report` prints no row for the reviewer cell

P33 added a `reviewer` cell to the policy table (`internal/policy/policy.go`). `session reach-report`
builds its rows from `ClassRows` (`internal/session/actions.go`, around line 794) and prints nothing
for that cell. `docs/design/03-hook-contract.md` (around line 188) says the report prints the table's
column, so the report and the table disagree.

Reproduce: run `session reach-report` on any session and look for a `reviewer` row; there is none.

Fix with P34, or sooner. The open design point is whether `CellFor` grows a row kind, so the report can
name the cell the same way the hook decides on it; the architect decides that, then a coder builds it.

Anchor: `internal/session/actions.go` (`ClassRows`) and `internal/policy/policy.go` (the `reviewer` cell).

## Open

- Does `CellFor` return a row kind, or does `ClassRows` list the cells itself? The first keeps one
  source of truth; the second is a smaller change.
- A test that fails when a cell exists in the table and not in the report.
