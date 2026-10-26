---
date: 2026-10-26
kind: bug
---

# `setup` stays dry-run after a role timeout, while `next` says the bot is running

`goLive` switches `discord.mode` from `dry-run` to `live` only when this run wrote a Discord key.
If a run writes `roleGroups.trusted` and then times out waiting for the owner to hold the role
(`internal/setup/discord.go`, the role step's timeout), the next run finds every key present and
writes nothing, so the mode stays `dry-run`, while `next` prints "the bot is running".

Found by review from reading, not reproduced; needs a test first (a scripted Discord whose owner
does not hold the role on the first run, then does on the second; assert `discord.mode` after the
second run).

Searched `docs/ideas/`, `docs/findings/` and `docs/HANDOFF.md` for overlap: none.
