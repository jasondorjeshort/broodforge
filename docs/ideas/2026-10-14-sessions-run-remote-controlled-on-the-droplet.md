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

Question: Open a proposal now for sessions running remote-controlled on the droplet in place of
the personal wrapper, or hold it until P26 lands the droplet instance?
Recommendation: hold. P26 (`../design/08-implementation-plan-setup.md`) is held with P14 on the
Ubuntu VM; once a Broodforge session runs on the droplet, the wrapper is `/session start` on a
project whose canonical is the host-management checkout, ADR 0022's per-session clone already
answers the clone question, and "which repos a session may clone" is the instance's project list,
not a new mechanism. Retiring the wrapper with plain Remote Control first needs no Broodforge
change and is the owner's alone. A proposal is warranted only if a droplet session is wanted
outside any project's checkout (clone anything on request), which 00-vision lists as later ("a
channel that controls the cloud machine itself").
Answer:
