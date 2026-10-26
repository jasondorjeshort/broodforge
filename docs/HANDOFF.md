# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-26 by main session (setup in few steps: ADR 0027, P20-P22 and P24 built)

## Now

- `main` is 7 ahead of origin, unpushed: ADR 0027 `d891592`, plan `0338526`, spike `9fa0532`, P21
  `fab65e1`, P22 `4acc5e8`, P24 `ed5f0d9`, companion `6f006b8`. Every code commit had its reviewer
  read and its fixes amended in; the companion commit (`6f006b8`, docs) has not been reviewed yet.
- Go is not installed on this PC: the passes used a go1.27.2 zip at `%LOCALAPPDATA%\Temp\bf-gotool-Kf7H`
  (nothing on PATH) and gcc from `C:\msys64\mingw64\bin`; `docs/reference/go-on-windows.md` is stale.
- The owner's spike (P20, `spike/setup-host/README.md`) has not been run; P23 waits on its verdicts.

## Next

1. Owner: run the P20 spike and paste its `claim` lines; the tester fills the finding. Then P23
   (`setup host` and `setup service` on Windows, `08-implementation-plan-setup.md`): coder.
2. Reviewer: `6f006b8` (companion) before the push; then the pusher, when the owner asks.
3. P25 `setup login` (after P23); P26 Linux on the Ubuntu VM then the droplet (held with P14).
4. Tester: `docs/reference/go-on-windows.md` (Go not installed here; the temp toolchain); the repro
   needs `</dev/null` (it reads stdin to EOF); triage the 2026-10-23 and 2026-10-24 ideas.
5. Earlier queue: s000a post-mortem Next; milestone 1 blocker (`engine run --real`, prompt mode,
   subagents); s0008's open items; the last-good cold start; the net-diff finding; carry-backs.

## Waiting on user

- Go on this PC: install it (`winget install GoLang.Go`) or keep the temp toolchain.
- Texts the passes added (see `05-debug-interface-setup.md`): the `fix token` portal line, the
  `fix application` intent line, the role-above-the-bot and Administrator lines, the channels-wait
  and guild-picker prompts, the `mismatch-config` export message; and `next` says "the bot is
  running" before P23 starts one.
- Earlier: the `pre-push` line (P15); P12a wording; "Never ends for inactivity."; the switch post;
  the hook no-answer phrase; R-17; `Waits:` blocks; commit name and email; `C:\elsewhere` to delete.

## Open findings

- `docs/findings/2026-10-26-spike-windows-setup-host-claims.md` (verdicts empty; Next 1)
- `docs/findings/2026-10-24-interrupted-session-cannot-be-ended-after-a-restart.md`; `2026-10-24-s000a-post-mortem-*.md`, `2026-10-24-background-subagents-*.md`, `2026-10-24-session-friction-*.md`; `2026-10-23-subagents-in-a-session-end-before-writing.md`, `...-first-live-*.md`
- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md`; `2026-10-15-spike-*.md` (droplet open); `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md`
