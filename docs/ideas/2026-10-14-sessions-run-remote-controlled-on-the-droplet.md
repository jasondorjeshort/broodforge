---
date: 2026-10-14
kind: feature
related: [2026-10-29-cross-host-lanes-run-a-check-on-the-other-instance.md]  # that one is one instance asking another; this one is where a session runs
---

# Sessions run remote-controlled off box, from the host-management checkout

Development has largely moved off the owner's PC, but at least one contributor still writes on
it. A temporary setup there had `claude` in the root tell people to type a personal wrapper name
instead, which ran a single session as that person in their checkout. The owner wants that
replaced: those sessions should be remote-controlled and run somewhere else, probably not in the
game's checkout but in the host-management checkout, and a session should clone a repo as needed
when dev work is on the table.

Anchor: the hosting principle and the "explicitly later" list in `docs/design/00-vision.md`. This
is the same shape as a Broodforge session (one person, one checkout, one credential, running on
the dev box), so once Broodforge sessions exist on the droplet, the personal wrapper is a
Broodforge session opened from Discord or a remote-controlled Claude Code session on the droplet.

ADR 0022 settled the per-session clone, and the Open question about the clone source is answered
there.

## Open

- Which repos a session may clone.
- Whether the personal wrapper is retired by Broodforge or by plain Remote Control first.
