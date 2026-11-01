---
date: 2026-11-01
kind: feature
related: [2026-10-29-broodforge-works-on-itself-overnight.md, 2026-10-31-the-simple-questions-channel.md]  # the first drains Plans unattended; this drains Questions; the second is the owner's framing of the channel shape
---

# Questions answered from Discord

Recorded from the owner's conversation of 2026-11-01 as a moonshot, not sanity checked: nobody has
read the anchor against the claim yet.

The owner answered six `Question:` blocks in a coordinator chat on 2026-10-31 in about twenty
minutes, one at a time, with the coordinator explaining each. The bot would post each block to an
owner channel as it is written: the question, the recommendation, and Accept, Edit and Later
buttons. Accept writes the recommendation as the `Answer:` line and commits the idea file; Edit
opens a modal whose text becomes the answer; Later leaves it. Triage then drains without a session,
and the owner reads the backlog where they already are. The owner, on this: "the sort of thing we
want, where a 'simple questions' channel exists with simple in-out answers from a sandboxed haiku(?)
agent that's got an independent briefing that it's talking to players not devs"; the two shapes
share the channel pattern, one facing the owner, one facing players.

Anchor: `docs/ideas/README.md` (the block format the post renders and the answer writes);
`.claude/skills/triage/SKILL.md` (where blocks are written); `internal/discord/customid.go` (the
buttons); `docs/design/04-ux.md` (a new owner channel beside the notice channel).

## Open

- The answer is a commit to the project's default branch by the bot, outside any session; which
  identity and which review it gets (ADR 0028 measures a land by its review).
- A question whose answer needs an explanation first, as all six did today: the post carries the
  recommendation's reason, or a Discuss button opens a short session.
