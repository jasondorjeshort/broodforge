# Install the always-up dev-live task

**Register the Task Scheduler task `Broodforge-dev-live`, which starts the dev-live bot at
machine boot with nobody logged on and restarts it when it fails.** It waits on you because a
boot task can only be registered from an elevated shell (a normal shell is refused `Access is
denied`), and an agent never raises UAC. The script and its procedure are `scripts/dev-task.ps1`
and the run skill's section "2c. The always-up task" (`.claude/skills/run/SKILL.md`); the instance is
`docs/design/05-debug-interface-dev-instance.md`'s live form.

True first: `bin\broodforge.exe` is built and the dev-live root exists (both are, today). The
dev-live bot may be running by hand: `install` then registers the task without starting a
second bot beside it, and says so.

In a normal PowerShell window in this checkout, as yourself (UAC asks once; the window that
opens is elevated and stays open):

```
Start-Process powershell -Verb RunAs -ArgumentList '-NoProfile -ExecutionPolicy Bypass -NoExit -File "<repo>\scripts\dev-task.ps1" install -Instance dev-live'
```

`<repo>` is this checkout's full path; `scripts\dev-task.ps1 install -Instance dev-live` run from
a normal shell prints the line with it filled in. Confirm the `user` line names your own
account (the task runs as whoever the elevated window is). The task logs on as S4U, "run whether
user is logged on or not" with no password stored. If it reports that the bot does not come up
under S4U, run the same line with ` -Logon Password` appended before the closing quote; that
asks for your Windows account password once, which Task Scheduler alone keeps.

A `warn  task \Broodforge-serve also starts dev-live` line names the boot task you made by hand
on 2026-10-31 (its action carries a `<you>` placeholder and its last run failed). Two boot
tasks on one instance race for its pipe; disable or delete that one in the same elevated window:
`Disable-ScheduledTask -TaskName Broodforge-serve`.

Then hand the running bot to the task, from a normal shell (or tell the coordinator, and an
agent runs it):

```
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\dev-task.ps1 redeploy -Instance dev-live
```

It builds, stops the hand-started bot with `stop`, starts the task and waits for `status`.

Check it took, from a normal (not elevated) shell: `scripts\dev-task.ps1 status -Instance
dev-live` prints `task  Broodforge-dev-live  Running  S4U ...` and the bot's `discord: live
connected`. After your next restart, before or without logging in, the bot answers in Discord;
once logged in, the same `status` shows a `last-run` at boot time. About ten minutes, plus the
reboot.

When done: delete this file and its row in `README.md`. If S4U did not work, tell the
coordinator which line failed, so the run skill and the script's default change.
