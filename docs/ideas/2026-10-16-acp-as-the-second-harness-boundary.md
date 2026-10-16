---
date: 2026-10-16
kind: feature
related: []
---

# ACP as the boundary for the second harness

The Agent Client Protocol (ACP), JSON-RPC over stdio, is what OpenAB and `claude-agent-acp` use to
drive Claude Code, Codex, Gemini and Kiro from one chat front end. It is a candidate boundary for
the second harness (R-42, R-43): one adapter speaking ACP would reach several engines instead of
one bespoke adapter per engine.

The first-version `Engine` interface stays bespoke, shaped by the stream-json subset and the
`defer` hook that ADR 0020 settled; ACP has no equivalent of a deferred tool call that exits the
process. This idea is for the day a second harness is wanted: check then whether ACP carries
permission prompts, resume, and per-turn usage well enough to sit behind `Engine`.

Anchor: `docs/design/03-architecture.md`, "The engine adapter" (the `Engine` interface). Source:
`docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md`, lesson L14.

## Open

- Does ACP report per-model usage and a cumulative count the usage-delta rule can use?
- Can the hook's pre-tool-use decision be expressed through ACP's permission request?
