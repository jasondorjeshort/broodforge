---
date: 2026-10-16
status: resolved
area: environment / windows antivirus
promoted_to: docs/reference/claude-code-on-windows.md
---

# BullGuard Sentry quarantined the binary and its process tree

## Question

At 19:28:31 local, the second a reviewer's `bin/broodforge.exe scratch run -- bash exp.sh`
returned, files vanished: `C:\Program Files\Git\usr\bin\{head,sed,seq,cp,find,sh}.exe`, the
repository's `bin\broodforge.exe`, and every file the run's children wrote into the reviewer's
scratchpad (serve log, outbox and tail JSON, a transcript, an audit copy). `grep`, `tail`, `sleep`
and `bash` were in use and survived on disk. What took them?

## What was tried

Read-only queries, 2026-10-16:

- `root/SecurityCenter2` lists BullGuard Antivirus (productState 262160) and Windows Defender.
  `Get-MpThreatDetection` shows no Defender detection in 24 hours.
- BullGuard processes were running: Core, FileScanner, Firewall, Sentry, Helper, Tray, Update.
- Listed `C:\ProgramData\BullGuard\SentryQuarantine\681\` (written 19:29:53).
- Ruled out: a delete in the reviewer's script (it had none); the project's own deletion paths.

## Result

BullGuard's Sentry (behavioural) component judged the freshly built unsigned binary and its
process tree malicious and quarantined the tree and its outputs, as one entry, 681. The entry
holds `broodforge.exe`, `bash.exe`, `cp.exe`, `find.exe`, `grep.exe`, `head.exe`, `sed.exe`,
`seq.exe`, `sh.exe`, `sleep.exe`, `tail.exe`, the scratch outputs and an `activity.log`. Beside it,
in the same minute, `Sentry.log`, `Detection.log`, `SentryEye.log` and a `.bglog` under
`Logs\Sentry` were written. The files were taken, not deleted: they are in the listing.

The two repro runs three minutes earlier lost nothing. The binary is rebuilt from source on every
pass, so each new build is judged again until an exclusion exists. Git's own tools are collateral
because they ran as the binary's children.

Resolution is the owner's. The owner uninstalled BullGuard on 2026-10-16 instead of restoring entry
681, so Git for Windows is reinstalled rather than restored
(`docs/operator/2026-10-16-bullguard-restore-quarantine-681-and-exclude-the-build.md`). Until that
is done no agent runs the binary, `go test` or a scratch root on this PC. The standing rule is
promoted to `docs/reference/claude-code-on-windows.md`.
