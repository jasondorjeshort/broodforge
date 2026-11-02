---
date: 2026-11-02
kind: chore
---

# An ADR with a companion is read for agreement before a pass builds it

Not sanity checked.

ADR 0028's Decision 1 and its companion's `config check` warning disagreed on what a `main` end
does when the land is refused; the proposal loop, the critic and the plan all passed over it, and
the P34 coder met it with the code half written. The cost was an architect pass and a coder fix
round in the same sitting, plus five sibling sentences in design docs that restated the decision
and had to be brought into line one by one.

Two base practices would have caught it, both for Hatchery (a carry-back once tried here):
- The `adr` skill, at write and at amendment: when an ADR has a companion file, the architect lists
  every rule the companion restates and checks each against the decision, in the ADR's own
  "Consequences" or the amendment entry. The critic's brief (`brief-critic.md`) asks for the same
  read as a named step.
- The `brief` skill's coder note: a pass whose Implements line names an ADR and a companion asks
  the coder to read both for agreement first and to stop and report a contradiction before writing
  code, as a missing dependency is reported today.

Anchor: `.claude/skills/adr/SKILL.md`; `.claude/skills/proposal/brief-critic.md`;
`.claude/skills/brief/SKILL.md` (per-agent notes, coder); the departure table for this project's
copy of each.

## Open

- Whether the design docs should restate an ADR's rule at all, or carry one sentence and a
  pointer (`.claude/rules/docs.md`, "an index is the one list a doc may carry" is the nearest rule).
