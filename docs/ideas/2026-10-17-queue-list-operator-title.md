---
date: 2026-10-17
kind: bug
---
# queue-list.js picks a mid-file line as the title of an operator entry

`node .claude/hooks/queue-list.js operator` showed "check it took" and "1. Backup, outside the
working tree." as the titles of the two entries filed 2026-10-17, not their bold opening lines.
Operator entries carry no frontmatter and no `#` title (`docs/operator/README.md`: an opening bold
line says what the step does), so the title rule in `.claude/hooks/queue-list.js` falls through to
some later line. The contract in `docs/design/decisions/0017-queue-list.md` should say what the
title of an operator entry is (the first bold line, unbolded) and the script and a `Q` case follow.

## Open

- Whether the same fallback misreads an idea or finding whose `#` title is not the first body line.

Plan: `tester`. In `docs/hatchery/decisions/0017-queue-list.md`, state the operator title: the first bold line of the body, unbolded. In `.claude/hooks/queue-list.js` `titleOf` (line 63), operator mode takes the first line matching `^\*\*(.+?)\*\*`; the other modes keep the `#` heading first. `Q` cases: an operator entry with a bold opening line and a later numbered list lists its bold line; an idea whose `#` title follows a blank line still lists the heading (the Open question). Verify: `node .claude/hooks/test-hooks.js`. Out of scope: the other columns.
