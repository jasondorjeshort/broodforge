---
date: 2026-10-30
kind: feature
---

# Real turns on the dev root seed fixtures

Once the dev credential exists, a real turn on the dev root can be recorded and run through the
import rewrite into a candidate fixture under `fixtures/engine/`, so the fake engine's corpus grows
from use instead of from hand-run spikes. Every turn the owner or an agent drives on the dev root
is a possible fixture; today each one is thrown away.

Anchor: `engine record` (`docs/design/05-debug-interface.md`, "A turn against the fake engine");
`internal/tools/importfixture`; the independent read that P1 requires before a fixture is
committed.

## Open

- Recording every turn keeps conversation text on disk; the rewrite and the read are the privacy
  gate, so a recorded-turn directory must sit outside the tracked tree.
- Which turns are worth a fixture (a new event shape, a refusal) versus noise.

Plan: coder, after the owner's credential step: a recorded-turn directory on the dev root, no flag. The recorder writes each real turn to `<root>/recorded/<turn>/`; the agent then runs importfixture over it and lists each rewritten candidate for the independent read. Nothing is committed by the bot. Depends on the dev credential.
