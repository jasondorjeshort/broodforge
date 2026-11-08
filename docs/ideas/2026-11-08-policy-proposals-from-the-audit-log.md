---
date: 2026-11-08
kind: feature
---

# Policy proposals from the audit log

Moonshot, not sanity checked. Every click a person makes on a permission prompt is a
`permission-decision` row (`internal/store`, decider `participant`, the tool and its input class). The
policy table (`03-hook-contract.md`) is edited by hand and has not changed since the live sessions
began, so the same `Bash` shapes are allowed by hand session after session, and now a subagent's too
(ADR 0029).

The idea: the bot reads its own rows and proposes. "`Bash echo …` was allowed 40 times across 9
sessions and never denied; add an allow for `echo` in reach `code`?" posts to the owner's notice channel
with two buttons; an accept writes the row to the project's policy (`.claude/hooks/policy.json`, or a
project-owned table the hook reads) on a branch with a commit the reviewer reads, never straight to
`main`; a decline is remembered so the proposal is not repeated. The prompt load falls over time by the
owner's own answers, and the table carries the evidence for every row it gains (the count and the
sessions). The reverse proposal is the safer one to start with: a tool that was denied every time it was
asked becomes a `deny` with a message, so the model stops asking.

Anchors: the `permission-decision` rows and `audit --kind`; `internal/policy/policy.go` (the table is
code today, a data file would have to come first); the notice channel (`docs/operator/`); ADR 0028 (a
proposed row lands by the same review-then-land path as any change).

## Open

- The table is Go code today; a proposal needs a data form the hook reads (a policy file per project,
  a major choice under ADR 0012: the hook contract's surface).
- Input classes: `echo hi` and `echo $(cat ~/.ssh/id_rsa)` are both `Bash echo`; the proposal must be
  over the class the hook already distinguishes (the git-guard's reading of quoted text, the path
  classes), never over a prefix.
- Whose answers count: the owner's only, or any participant with the role.
