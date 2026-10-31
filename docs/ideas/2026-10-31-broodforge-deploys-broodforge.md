---
date: 2026-10-31
kind: feature
related: [2026-10-23-handoff-main-lands-a-session-on-the-default-branch.md, 2026-10-29-broodforge-works-on-itself-overnight.md]  # the first lands the change; this one runs it; the second would need this to be unattended end to end
---

# Broodforge deploys Broodforge

Recorded from the owner's conversation of 2026-10-31 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

The owner develops the bot through the bot, but a change landed on the default branch still needs
a hand to build the binary and restart the service. A landing that passes build, the check line
and the golden sessions builds the binary, swaps it in place, restarts the service, and the open
sessions come back as held sessions the way ADR 0026 already makes them after any restart. The
previous binary stays beside the new one as the rollback. This is the last mile of the vision's
first-usable-version sentence: from then on the owner develops Broodforge through Broodforge.

Anchor: `cmd/broodforge/main.go` (the one binary, which is also the sandbox entry wrapper, ADR
0023); `docs/design/decisions/0026-interrupted-sessions-held-after-a-restart.md` (the restart
survives sessions); `docs/design/08-implementation-plan-golden-sessions.md` (the check that would
gate the swap); the `main` handoff of the related idea (the trigger).

## Open

- Who restarts the service on each host: the Task Scheduler task on the PC, systemd on the
  droplet; whether the running binary can replace itself on Windows while it runs.
- A migration in the new binary runs against the live store; the snapshot idea
  (`2026-10-30-serve-snapshots-the-store-before-a-migration.md`) is a prerequisite.
- A bad binary that starts and then fails: what decides to roll back, and how the owner hears.
