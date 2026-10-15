---
date: 2026-10-15
kind: chore
related: [.claude/agents/pusher.md, docs/design/decisions/README.md]
---

# Carry back to Hatchery: pusher check 5a and the ADR 0009 index row

Two base improvements made here on 2026-10-15 that the base does not have yet. Both are edits to
files where the base wins on the next `update-from-hatchery`, so they are taken into the base
verbatim or they come back as conflicts.

1. `.claude/agents/pusher.md`, check 5a: a personal-data scan of the outgoing diff and messages
   before every push (emails with a dotted domain, home paths carrying a person's username, host
   names, people other than the owner, account ids; the project's own service-account paths are
   allowed). The owner's rule: an independent Sonnet or Haiku read for doxing before every release.
   Numbered 5a so ADR 0006's "check 7" keeps its number.
2. `docs/design/decisions/README.md`: the base at `ea20b96` has no row for ADR 0009; this project
   added one by hand in its second merge. The base's own index should carry it.

Plan: in a Hatchery session, copy check 5a and the Report clause from this project's `pusher.md`
verbatim, add the 0009 row to the base's decisions index, commit through the base's runner. Then
this file is deleted in the commit that merges that base change here.
