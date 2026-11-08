---
date: 2026-10-29
kind: feature
related: [2026-10-23-session-land-pushes-to-main-mid-session.md, 2026-10-30-owner-less-weeks.md, 2026-10-30-credential-pool-scheduling-by-headroom.md, 2026-10-30-the-base-tends-itself.md, 2026-10-31-broodforge-deploys-broodforge.md]  # that one is the accept button; this one runs unattended; the pool and base-tending ideas build on this; the last runs a landed change as the live service, which unattended end to end would need
---

# Broodforge works on itself overnight

A moonshot the owner judged far off at filing (2026-10-29); on 2026-11-08 the owner asked for it to
move: P40 (ADR 0029) removed the one blocker a delegating session had, a subagent's prompt ending the
subagent, so a session now delegates in prompt mode under a 60-minute bound. A scheduled session in the Broodforge
channel reads `docs/ideas/` and `docs/findings/`, runs the `triage` skill, builds the `Plan:` ideas
on session branches, and posts each result in the channel as buttons: accept (the `main` handoff),
review (open the venue), discard (delete the branch). The vision already has the owner developing
the bot through the bot; this takes the owner out of the loop until the decision, which the ideas
README reserves for them anyway.

Anchor: the `triage` skill in `.claude/skills/triage/`; session start from a schedule rather than a
person has no entry point yet (`internal/session/service.go` starts sessions from a venue
command); the accept button is the `main` handoff (ADR 0028, built by P34;
`../design/decisions/0028-landing-a-session-on-the-default-branch.md`).

## Open

- R-46's token cap must exist first: an unattended session has nobody to notice spend.
- Whose credential pays for an unattended session; probably an owner-configured API key.
- A nightly session that files findings and ideas feeds its own next run; the ideas README's rule
  that no agent deletes an idea on its own judgement is the brake, and it must hold here.

Waits: the first usable version (`00-vision.md`, "The first usable version") has shipped; the owner, 2026-10-29.
