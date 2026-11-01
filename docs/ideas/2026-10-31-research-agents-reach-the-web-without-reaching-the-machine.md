---
date: 2026-10-31
kind: feature
---

# Research agents reach the web without reaching the machine

Refiled from session s0009 (2026-10-31), whose file was lost when the session ended with nothing
committed (`2026-10-31-ending-a-session-with-uncommitted-changes-loses-them.md`). Not sanity checked.

In s0009 the owner asked a session to use a subagent to look up the weather on Pikes Peak. The
`general-purpose` subagent launched (the usage gate let it through) and failed at its first step:
the session's reach is `code`, which denies the web tools and anything outside the tree
(`internal/policy/policy.go`, the reach denial; the reach model is ADR 0022,
`docs/design/decisions/0022-session-isolation-and-credential-storage.md`). The owner: "we do want
access to the internet allowed for research agents. We don't particularly want access to the local
machine outside of the sandboxed area."

What it would take: web search and fetch allowed for a research subagent while file and shell
access stay confined to the session tree. The split the policy layer does not have today: reach
governs files and commands, not the network.

## Open

- A new reach value, a flag on `code`, or a grant tied to the agent type the hook sees?
- Whether `curl` and the like through Bash stay denied (the sandbox question, not the tool one).
- Whether this is a proposal (it widens a promised boundary, ADR 0022) or a pass.

Plan: architect, a proposal in `docs/design/proposals/` (the `proposal` skill, then the critic, then an ADR), not a
code pass: it widens ADR 0022's promised boundary, a major choice (`major-decisions.md`). The owner has said they
want it ("we do want access to the internet allowed for research agents", above). Read `internal/policy/policy.go`
(the reach denial, `Decide` and `CellFor`) and `docs/design/03-hook-contract.md` to name the rows it touches (the `ClassWeb` row, denied in both reaches because ADR 0022 defers egress control). The
proposal weighs the three shapes in Open (a new reach value, a flag on `code`, a grant tied to the agent type the
hook sees) and settles whether `curl` and the like through Bash stay denied (recommend yes: the network is the
web tools' only door). It states what stays confined (files and shell in the tree), the ADR 0012 promised-surface
check, and the debug-interface way to drive it (`05-debug-interface.md`). Verify: the proposal file exists, and
the critic brief is written, naming the strongest case against each shape. Out of scope: any code, an engine-level sandbox.
