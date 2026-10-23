---
date: 2026-10-23
status: open
area: sessions / process
---

# The first live session's commits reached main unreviewed

## Question

Did the process hold for the first real session driven from Discord (s0005, Opus 5.5, real engine,
a scratch root with the project's canonical clone), whose four commits the owner pushed to `main` by
hand? The commits: idle timeout never by default with `/session timeout`; prompt timeout never by
default; a vision principle; two ideas.

## What was tried

- The session trace and audit read.
- Two independent reviews: the first of the working tree and then commit d531b6e, the second of the
  range 1e44f39..origin/main. Builds, vet, staticcheck, exhaustive and the race test were run on
  `git archive` copies.
- The settings-source canary (`2026-10-23-session-settings-sources-drop-claude-md-and-agents.md`).

## Result

Held:

- Discord prompts reached Discord and were answered (3 of 80 decisions were asked, the rest allowed
  by policy and audited).
- Commits carry the owner as author and committer and the correct model trailers, no session link;
  the privacy scan is clean.
- Build, vet, staticcheck, exhaustive and the race tests pass on the final tree.
- Zero (the old default) was safe in all consumers; existing positive timeouts behave as before.

Not held:

1. No independent review before the push. The session committed itself and used no subagent (73 tool
   calls: 29 Edit, 21 Read, 14 Grep, 7 Bash, 2 Write; none Task) because the project's agreement and
   agents never loaded (the settings-source finding).
2. Size caps exceeded on main: the handoff 48 lines (cap 40), `03-architecture.md` 303 and `04-ux.md`
   312 (cap 300). Unseen because the post-edit hook did not load in the session and the push was by
   hand, so only the pusher's check 7 would catch it.
3. `ParseIdleTimeout` in `internal/session/timeout.go` accepts a bare minute count that overflows
   `time.Duration`: 307445735 became 26 seconds and 153722868 became negative, shown as "never".
4. `l.idleTimeout` is read after the unlock in `internal/session/actions.go` while `SetIdleTimeout`
   writes it under the lock (no race seen in 60 `-race` runs).
5. Docs not updated: `05-debug-interface.md` command tree, `05-debug-interface-discord.md`
   subcommand lists, `02-domain.md` ("the channel's idle timeout"), `03-architecture.md` ("until the
   idle timeout"), the `events.go` comment "60 minutes", and the `05-debug-interface-session.md`
   promise of `<n> minutes` while `1h30m15s` is accepted.
6. The new vision text says a session stays open until a person ends it, but every bot restart
   interrupts open sessions (R-17) and resume is later (R-18).
7. Both ideas filed that day end in a `Waits:` block, which only the owner writes, and their quoted
   owner words could not be checked against the trace.
8. The session rewrote the handoff, whose shape rules it did not know.
9. New human-facing wording ("Never ends for inactivity.", the refusal text, the slash option text)
   was not listed as waiting on the owner.
10. `TestProofHookPreToolUse` fails when `BROODFORGE_SESSION` is set (idea of 2026-10-23).
11. `session show --json` gives `"until":""` for a pending call with no deadline.

## Next

- Items 3, 4, 5, 10 and 11: fixed and reviewed in 9a65371 and aadfec2 (with the owner's
  decision of 2026-10-23: every timeout is capped at a year with a notice, and never is a
  one-year timer). Item 2: ff5b45f. Items 8 and 9: the handoff of 53fcc18.
- Items 6 and 7 are the owner's.
- The design question behind items 1 and 2 is the idea named in the settings-source finding
  (`docs/ideas/2026-10-23-sessions-load-none-of-the-projects-instructions-agents-or-hooks.md`), a
  proposal, not this finding.
