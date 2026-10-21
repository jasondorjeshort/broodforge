---
date: 2026-10-21
kind: chore
related: [2026-10-21-carry-back-quoted-git-does-not-name-git.md]
---

# Carry back: a main-session push runs pusher check 5 first, read included

Anchor: the Pushing section of `.claude/skills/commit/SKILL.md` and check 5 of
`.claude/agents/pusher.md`.

Before the update to Hatchery `dc896c3` (ADR 0024), this project's commit skill made a main-session
push run the pusher's privacy check first: the mechanical scan and then a human read of each
outgoing commit for names, numbers, addresses and identifiers, with the result quoted before the
push. The owner asked for that after the 2026-10-17 incident
(`docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md`). The base's
commit skill runs the scan on every push through the guard's rule 20 and the pre-push hook, but
only the `pusher` agent does the read; a push from the main session skips it. This project keeps
the rule as a project line in `CLAUDE.md` (`## Project rules`) until the base carries it.

Plan: in a carry-back worktree cut from the current `hatchery/main` (`carry-back` skill, Send), add
to the commit skill's Pushing section that a main-session push performs pusher check 5 in full, scan
and read, and quotes the result before the push; one commit `Carry-back: commit skill: a main-session
push runs pusher check 5, read included`. When it lands, drop the `CLAUDE.md` line here.
