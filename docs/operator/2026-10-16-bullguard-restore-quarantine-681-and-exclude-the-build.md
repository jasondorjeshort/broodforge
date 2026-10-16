# BullGuard: restore quarantine entry 681 and exclude the build

**Restores the files BullGuard quarantined on 2026-10-16 and stops it judging each new build
again.** It waits on the owner because only the owner can operate the BullGuard console and its
exclusion list. Evidence and reasoning:
`docs/findings/2026-10-16-bullguard-sentry-quarantined-the-binary-and-its-process-tree.md`; the
standing rule is in `docs/reference/claude-code-on-windows.md` (Go on this PC).

Until this is done, no agent runs `bin\broodforge.exe`, `go test`, or a scratch root on this PC.

Runs as the owner, at the PC, in the BullGuard window (no command to paste). Takes about 10 minutes.

1. Open BullGuard, then Quarantine, and find the entry dated 2026-10-16 19:29 (folder 681 under
   `C:\ProgramData\BullGuard\SentryQuarantine`). Restore it. That returns Git's `head`, `sed`,
   `seq`, `cp`, `find`, `sh`, `bash` and `grep`, `sleep`, `tail`, and the repository's
   `bin\broodforge.exe`. Alternative: repair or reinstall Git for Windows, and let the next build
   recreate the binary.
2. Add exclusions (file and process, folder level) in BullGuard's settings for:
   - the repository's `bin\` directory;
   - `%LOCALAPPDATA%\Temp\scratch-*` roots (add the `%LOCALAPPDATA%\Temp` folders that hold them
     if wildcards are refused);
   - the Claude scratchpad under `%LOCALAPPDATA%\Temp\claude\`.
3. Check, in Git Bash:

   ```
   ls 'C:/Program Files/Git/usr/bin/'{head,sed,seq,cp,find,sh}.exe
   ```

   All six must print. A rebuilt `bin\broodforge.exe` surviving a `scratch run` confirms the
   exclusion took.

When done: delete this entry. If the exclusions are recorded somewhere else, update
`docs/reference/claude-code-on-windows.md` (drop the "do not run" bullet, note the exclusions).
