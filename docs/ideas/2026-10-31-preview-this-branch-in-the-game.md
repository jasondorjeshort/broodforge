---
date: 2026-10-31
kind: feature
related: [2026-10-31-zero-token-buttons-from-the-repository-workflow.md, 2026-10-29-the-game-files-its-own-tickets.md, 2026-10-29-cross-host-lanes-run-a-check-on-the-other-instance.md]  # the first is the button this rides on; the second is the game reporting in; the third runs a check elsewhere, not a game
---

# Preview this branch in the game

Recorded from the owner's conversation of 2026-10-31 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

For the hosted game, one button in the venue deploys the session branch to a per-session preview
instance of the game, and the player who asked for the change tries it in play before anyone lands
it. The hosting half (a preview instance per branch, its lifetime, its data) lives in the game's
repository as one of the fixed actions of the related idea. The bot half is the action and a link
to the preview in the venue, and a teardown when the session ends.

Anchor: `internal/git/handoff.go` (the session end, where a preview would be torn down); the
actions file of the zero-token-buttons idea; the game's repository, not here.

## Open

- A preview instance runs the branch's code on the host: the same hole the vision names for the
  cloud box, now opened on every session, so the preview must run under the session's isolation.
- A game with persistent state needs a copy of it per preview or a throwaway world.
- Who may press it: the driver, or anyone in the channel who can see the venue.
