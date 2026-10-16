# BullGuard: reinstall Git for Windows and finish the uninstall

**Replaces the Git tools BullGuard quarantined on 2026-10-16 and completes BullGuard's removal.**
The owner uninstalled BullGuard instead of restoring quarantine entry 681, so the restore path is
gone (the quarantine's `.bin` files are the product's own format) and Git for Windows must be
reinstalled. It waits on the owner because the install needs a UAC prompt and a reboot. Evidence
and reasoning: `docs/findings/2026-10-16-bullguard-sentry-quarantined-the-binary-and-its-process-tree.md`;
the standing rule is in `docs/reference/claude-code-on-windows.md` (Go on this PC).

Until this is done, no agent runs `bin\broodforge.exe`, `go test`, or a scratch root on this PC.

Runs as the owner, at the PC, in an elevated-capable shell. Takes about 10 minutes.

1. Reinstall Git for Windows over the top (accept the UAC prompt). Either the installer from
   git-scm.com, or:

   ```
   winget install --id Git.Git -e --source winget --force
   ```

2. Reboot once, so the BullGuard uninstall completes and Security Center stops listing it.
3. Check, in Git Bash:

   ```
   ls 'C:/Program Files/Git/usr/bin/'{head,sed,seq,cp,find,sh}.exe
   ```

   All six must print.
4. Tell the coordinator "runs are clear".

Defender remains. If a later build is quarantined by Defender, file the same class of entry.

When done: delete this entry and its line in the index, and drop the "do not run" bullet in
`docs/reference/claude-code-on-windows.md`.
