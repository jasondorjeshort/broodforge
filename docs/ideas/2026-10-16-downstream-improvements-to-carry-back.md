---
date: 2026-10-16
kind: chore
---
# Two downstream improvements to the base files, not yet carried back

Found 2026-10-16 by comparing a decanted sibling's `.claude/` against every Hatchery commit: two
of its three locally edited base files are improvements rather than project fits.

- `.claude/agents/pusher.md`: a check 5a, a personal-data scan of the outgoing diff, beside the
  secrets scan. The base has an idea for a private-terms guard hook
  (`docs/ideas/2026-10-13-private-terms-guard-hook.md`); the pusher check is the cheap version
  that exists today.
- `.claude/skills/adr/SKILL.md`: an Amendments section, for the edit a closed ADR may take beyond
  the prediction outcomes ADR 0012 allows. The base ADR 0014 just needed one (the residue idea
  `2026-10-16-handoff-import-check-residue.md` asks for an amendment with nowhere to put it).

Fix: read the sibling's two files (`../broodforge`, `git log -- <path>` there for the reason),
take each as a base change through the ordinary review, and the sibling's next update then stops
conflicting on them. The Amendments section needs a sentence in ADR 0012's item 6 rule ("the one
edit a closed ADR takes") or an amendment of its own.
