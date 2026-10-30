---
date: 2026-10-29
kind: feature
related: [2026-10-14-sessions-run-remote-controlled-on-the-droplet.md]  # that one is where a session runs; this one is one instance asking another
---

# Cross-host lanes: a session runs a check on the other instance

A moonshot the owner judged far off at filing (2026-10-29). A session on the Linux droplet asks
the Windows instance to run a Windows-only check (a scheduled-task start, a ConPTY repro, a
`process_windows_test.go` case) on the same branch and gets the result back in its venue; the
reverse for a PC session wanting `bwrap`. W6 of the Windows setup spike is the kind of thing only
one host can test, and today the owner is the lane.

This is a departure to argue: `00-vision.md` says many instances per deployment is not a growth
axis, and the two instances it does plan serve different projects (the PC the Broodforge channel,
the droplet the hosted game) with nothing connecting them. A lane
between exactly two instances is narrower than a fleet, but it is still an instance-to-instance
protocol, a major choice (network protocol or API surface) and so a proposal before any code.

Anchor: the per-host process model in `docs/design/03-architecture.md` ("Process model per
host"); the control socket in `internal/control/` is the only cross-process surface now, and it is
local.

## Open

- Transport and trust between two hosts that are deliberately kept apart (secrets stay secret).
- Whether the result is a text report or a recording the fake engine could replay.
- Who pays for the remote run and which audit row records it.

Waits: the first usable version (`00-vision.md`, "The first usable version") has shipped; the owner, 2026-10-29.
