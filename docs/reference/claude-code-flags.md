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

Check a flag against a new engine version before relying on it:

```
claude --help | grep -A2 -E "^\s+--effort|^\s+--model"
```

The recorded init line of a stream-json run carries `per_turn_effort_active` (the `basic`
fixture, 2.1.296); the adapter does not read it.
