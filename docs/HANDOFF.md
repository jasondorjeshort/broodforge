# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-26 by main session (P20 spike run, W6 open; Go installed)

## Now

- Unpushed on `main`: ADR 0027 `d891592` through companion `bfb56b3`, then the P20 verdicts and the
  Go reference. Code commits were reviewed with fixes amended in; the companions (`6f006b8`,
  `bfb56b3`) and today's two docs commits go to the reviewer before the owner's requested push.
- P20 spike run by the owner: every claim holds but W6's last step (`Start-ScheduledTask` never
  started the task, 0x41303; cause unknown) and the unreported console row. P23 waits on W6.

## Next

1. Coder: W6 per the P20 finding's Next (read back the task, a trigger `Start-ScheduledTask`
   honours, `schtasks /run`), then an owner re-run of the spike. Then P23: coder.
2. Reviewer: `6f006b8`, `bfb56b3` and today's docs commits; then a fresh pusher.
3. P25 `setup login` (after P23); P26 Linux on the Ubuntu VM then the droplet (held with P14).
4. Tester: the repro needs `</dev/null` (it reads stdin to EOF); triage the 2026-10-23, 2026-10-24
   and 2026-10-26 ideas (the last: `goLive` after a role timeout, needs a test).
5. Earlier queue: s000a post-mortem Next; milestone 1 blocker (`engine run --real`, prompt mode,
   subagents); s0008's open items; the last-good cold start; the net-diff finding; carry-backs.

## Waiting on user

- P20: whether a console window appeared on the desktop during `run.ps1` (at the W6 re-run).
- Texts the passes added (see `05-debug-interface-setup.md`): the `fix token` portal line, the
  `fix application` intent line, the role-above-the-bot and Administrator lines, the channels-wait
  and guild-picker prompts, the `mismatch-config` export message, the invite and closing line
  prefixes, the `socket-path`, `live-instance` and `bad-config` texts; and `next` says "the bot is
  running" before P23 starts one.
- Earlier: the `pre-push` line (P15); P12a wording; "Never ends for inactivity."; the switch post;
  the hook no-answer phrase; R-17; `Waits:` blocks; commit name and email; `C:\elsewhere` to delete.

## Open findings

- `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` (W6 open; Next 1)
- `docs/findings/2026-10-24-interrupted-session-cannot-be-ended-after-a-restart.md`; `2026-10-24-s000a-post-mortem-*.md`, `2026-10-24-background-subagents-*.md`, `2026-10-24-session-friction-*.md`; `2026-10-23-subagents-in-a-session-end-before-writing.md`, `...-first-live-*.md`
- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md`; `2026-10-15-spike-*.md` (droplet open); `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md`
