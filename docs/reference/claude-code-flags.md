---
verified: 2026-10-23
source: observed (`claude --help` on Claude Code 2.1.309, the owner's PC)
---

# Claude Code command-line flags the engine adapter passes

The engine adapter builds one command line per engine process (`internal/engine/claudecode.go`,
`Args`). Facts about the flags it depends on, read from the engine's own help:

- `--model <model>`: the model for the process. Takes an alias or a full model id. Broodforge
  passes only an id from `policy`'s model map (`internal/policy/roles.go`, `ResolveModel`).
- `--effort <level>`: the effort level for the process. The engine lists `low`, `medium`, `high`,
  `xhigh`, `max`. Broodforge passes only `policy.Efforts` (`low`, `medium`, `high`, `max`).
- `--resume <id>`: continues an engine session, so a model or effort switch keeps the context
  (ADR 0020).
- `--append-system-prompt-file <file>`: the file's text appended to the default system prompt.
  The help has no line of its own for it; it names `--append-system-prompt[-file]` under `--bare`
  as a way to give context explicitly, and the critic's probe for ADR 0025 found it loads beside
  `--setting-sources user`. Broodforge passes the session's `workflow/<id>/instructions.md` on
  every process (ADR 0025 rule 5). Under `--system-prompt-snapshot on` (the default) the prompt
  is rendered on the conversation's first request and recorded; every later request and resume
  sends the record as-is, even when a later launch passes different text, until a compaction.
- `--agents <json-or-file>`: custom agents, a JSON object, or with `--print` (`-p`) the path to a
  file holding one, keyed by agent name with `description` and `prompt`. Broodforge passes
  `workflow/<id>/agents.json` when the copy defines an agent.
- `--append-system-prompt <prompt>`: the text form; Broodforge no longer passes it.

Check a flag against a new engine version before relying on it:

```
claude --help | grep -A4 -E "^\s+--effort|^\s+--model|^\s+--agents|append-system-prompt"
```

The recorded init line of a stream-json run carries `per_turn_effort_active` (the `basic`
fixture, 2.1.296); the adapter does not read it.
