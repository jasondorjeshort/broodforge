---
date: 2026-10-14
kind: feature
---

# UX choices that could become user or channel options

The first version fixes these in `docs/design/04-ux.md` (owner decisions, 2026-10-14). The owner
expects some of them to become a user or channel option eventually:

- Venue per channel: private channel per session (v1) or a private thread in the project channel.
- Footer token figures: shown in every footer (v1) or only in `/usage` and the summary.
- Trace lines: full in the final message (v1) or collapsed to counts (`trace: full|brief`).
- The `Allow <tool> this session` button: present (v1) or allow-once and deny only.
- The mode names (`asks first` / `acts freely`), changeable per instance or channel.
- Model and effort per session: `/session model|effort`, driver only, next turn (R-29), where v1
  has channel configuration only.
- Dot-prefix command aliases beside the slash commands.

Question: Which of these, if any, should become a per-channel or per-user option now, rather than
staying 04's fixed decisions?
Two are already settled: model and effort per session landed with R-29
(`../design/04-ux-session-controls.md`, P12a and P12b), and dot-prefix aliases were decided
against (04, decision 1). The other five stand as 04's decisions 3 to 7, each the owner's.
Recommendation: none now. Each is a channel key in 03's configuration plus a renderer or venue
branch; only the thread venue (B) has design weight (the sweep, venue binding, auto-archiving).
Decide after a month of live use on the Broodforge channel shows which fixed choice chafes; the
one picked goes to the architect for a 03 and 04 amendment, then one coder pass per option.
Answer:
