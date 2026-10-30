---
date: 2026-10-30
kind: feature
related: [2026-10-30-real-turns-on-the-dev-root-seed-fixtures.md, 2026-10-23-engine-auto-permission-mode-as-a-third-session-mode.md]  # fixtures come from real turns; the auto spike also spends a turn
---

# Engine upgrade canary: replay prompts on the new engine

A moonshot. After a Claude Code auto-update, replay the prompts of the recorded sessions on the dev
instance against the new engine and diff the shape of the event stream (types and fields, not text).
A stream format change, like the `rate_limit_event` that was once an `unknown` event, is then caught
before the live bot meets it. It costs one real turn per recorded session, so the spend is the owner's
to approve.

Anchor: `internal/engine/parse.go`, the fixtures it parses, and `setup engine`.
