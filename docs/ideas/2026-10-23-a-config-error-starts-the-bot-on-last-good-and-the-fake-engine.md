---
date: 2026-10-23
kind: bug
---

# A config error restarts the bot on the last-good config, and nobody is told

Observed 2026-10-23: the owner renamed a project's `id` without renaming `config/projects/<id>.json`.
`config check` and `discord check` failed with `the file must be named broodforge.json`, but
`serve` started anyway: the audit shows `configuration-invalid` (`using: last-good`) and then
`instance-started` with `config: last-good`, `engine: fake`. The owner started a session and got a
canned fixture reply ("hello") instead of Claude, and could not tell why; only `status` (`config:
last-good from <time>`) and the audit said so. Last-good is meant for a failed `config reload` on a
running bot (03, Configuration); at a cold start it quietly runs the old config, here with the
wrong engine.

Anchors: `internal/config/lastgood.go`, `internal/cli/serve.go` (startup steps 1 to 5),
`docs/design/03-architecture.md` (Startup, Failure modes, `config` row).

Shape to consider: at start, a config that does not load is an error that stops `serve` (the owner
is at the terminal) unless a flag asks for last-good; on a running bot it stays as designed, and the
notice channel gets a post when the bot is on last-good. A session started on a fake engine says so
in its header, so a canned reply is never mistaken for the agent (`fake` shows in `session show`
only).

## Open

- Is a cold start on last-good ever wanted (an unattended restart after a bad edit)? Then it needs a
  loud notice, not a quiet line in `status`.
- Whether the fake engine's header line belongs in `04-ux.md` (what humans see: the owner's call).

Question: at a cold start with an invalid `config/`, should `serve` keep ADR 0021's fallback to
last-good, made loud (the notice-channel post the ADR designed and `startupReport` never built, a
`config: last-good from <time>` clause in the start-up report, and an `engine: fake` line in the
venue header when the channel runs the fake), or refuse unless a flag such as `--last-good` asks
for it? Recommendation: keep the fallback and make it loud. The fallback exists for the unattended
restart (`setup service`, Task Scheduler, systemd), where a refusal is a bot that stays down until
someone looks, while an attended start already has `config check` and the audit row; the loud
parts are the unbuilt half of ADR 0021 and need no new decision, only your wording for the two
lines. A yes makes this a coder pass (`internal/cli/serve.go` steps 2 and 7, the gateway report,
the header text); a no is an ADR 0021 amendment first.
Answer:
