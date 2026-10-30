---
name: run
description: Start, drive, observe and stop this project's own app, the Broodforge bot, on the PC development instance from a terminal (no Discord, no GUI). This is the project's launch procedure; use it, not a generic pattern, when asked to run, start or exercise Broodforge, to confirm a change works in the running bot, or to reproduce a bug against the real binary.
---

# Run Broodforge on the development instance

The development instance is one kept root under the owner's own account, instance `dev`, in
dry-run for good (`docs/design/05-debug-interface-dev-instance.md`: what it is, what it is not, the
owner's one step). Everything below is Git Bash; every path is the binary's own
(`bin/broodforge.exe`), nothing goes on `PATH`. Channel `100` is the fake engine and free; channel
`200` and `engine run --real` spend the owner's plan (step 4). The default loop, steps 0 to 3 and
5, starts, drives and stops the bot inside one agent run; a bot left running for the owner is
step 2b, never step 2.

## 0. One agent, one bot

The brief names the dev instance as yours for this run; that assignment is the only guard there is
(`status` carries no PID and no owner token), so do not assume it. Set the root for every command
and drop the service instance's key override:

```
export BROODFORGE_ROOT="$LOCALAPPDATA/Broodforge/dev"; unset BROODFORGE_KEY_FILE
bin/broodforge.exe status
```

Exit 1 with `error: no-config: ...` (no root yet) or `error: not-running: ...` is the normal start.
A bot that answers before you started one is another agent's or the owner's: do not `stop` it;
report `a bot already answers on the dev root` and stop here. Never stop a bot by PID or process
name (`docs/hatchery/reference/claude-code-on-windows.md`); a `stop` is only ever of the bot this
run started.

## 1. Build, then make or refresh the root (main checkout only)

```
CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge
bin/broodforge.exe setup --discord dry-run --yes --root "$BROODFORGE_ROOT" --instance dev --project broodforge --repo "$(pwd -W)" --wait 1
```

The `setup` line runs from the main checkout only: it writes that checkout's paths into the root.
In a worktree, build there and skip the line; a missing root is then a report, not a setup from the
worktree. First time: every step `ok`, the key line included (a dev key; it goes in no password
manager, and it stays out of your report). Later: only `ok` and `skip`. `error: mismatch-engine`
means the owner's Claude Code auto-updated: run `bin/broodforge.exe setup engine --yes`, then the
same `setup` line again. Any other refusal is a finding, not something to work around by editing
the root; `live-instance` means someone promoted the root by mistake (the doc says what then).

## 2. Start the bot in the background and wait for it

One Bash call with `run_in_background: true` (a foreground call's process tree ends with the call),
logging to a file of this run's own:

```
bin/broodforge.exe serve --discord dry-run >"$BROODFORGE_ROOT/serve-$(date +%Y%m%d-%H%M%S).log" 2>&1
```

Then `bin/broodforge.exe status --wait 10 --expect ok`. Exit 1 here: read your log for the `error:
start-<step>` line and report it with the line. Before driving (step 3) and before stopping (step
5), check your own `serve` is still alive: the harness has not reported your background call
exited, and `status` answers. A `status` that answers after your call exited is a foreign bot.

## 2b. Leaving the bot running for the owner (only when the brief says so)

A background call does not outlive the agent, so a bot the owner will use after you (the credential
step) is started detached, the way `docs/hatchery/reference/claude-code-on-windows.md` prescribes,
with its PID in `<root>/serve.pid` and its output in `<root>/serve-<stamp>.log` and `.err`:

```
exe="$(pwd -W)/bin/broodforge.exe"; rw=$(cygpath -w "$BROODFORGE_ROOT"); stamp=$(date +%Y%m%d-%H%M%S)
powershell -NoProfile -Command "\$p = Start-Process -FilePath '$exe' -ArgumentList '--root','$rw','serve','--discord','dry-run' -WindowStyle Hidden -RedirectStandardOutput '$rw\serve-$stamp.log' -RedirectStandardError '$rw\serve-$stamp.err' -PassThru; Set-Content -Path '$rw\serve.pid' -Value \$p.Id"
bin/broodforge.exe status --wait 10 --expect ok
```

Report the PID file and the log names. The matching stop, by whoever the brief says (the owner
after the credential step, or a later run told to): `bin/broodforge.exe stop`; only when the bot
does not answer that within its timeout, `taskkill /PID "$(cat "$BROODFORGE_ROOT/serve.pid")" /F`;
then delete `serve.pid`.

## 3. The round trip on the fake channel

```
bin/broodforge.exe register 1001 --name "Ada Example" --email ada@example.com
printf scratch | bin/broodforge.exe credential add --kind api-key --user 1001 --label fake
bin/broodforge.exe register set-default 1001 <the cred-… just printed>
bin/broodforge.exe discord inject click --custom-id bf:100:start --user 1001 --roles 10
bin/broodforge.exe session list
bin/broodforge.exe discord inject message --channel <id> --user 1001 --roles 10 "Read hello.txt and reply with its one word."
bin/broodforge.exe session tail <id> --all
bin/broodforge.exe session outbox <id>
```

`register` and the credential are idempotent on a rerun (`register` replaces the pair; a second
`credential add` is a second id, harmless). `<id>` is the session `inject click` started, the next
free one (`s0001` on a fresh root); `session list` shows it. What to check: the tail's last line is
`turn-ended  end_turn`; the outbox has the `channel` line for `<id>` and a `create` whose first line
starts `Session <id> on Broodforge`; the final message and the working message's `delete` arrive
once the coalescer is through, so poll the outbox rather than count its lines. Drive the feature
under test the same way: every command of `docs/design/05-debug-interface.md` and its companions
works here.

## 4. The real engine, on the owner's word only

A real turn runs only when the brief quotes the owner's go-ahead for that real turn in so many
words; a brief that merely asks to "test on the real engine" is not that. Then, with the dev
credential `usable` in `credential list` (the owner's step): `bin/broodforge.exe engine run --real
--credential <id> --cwd <tmp tree> --prompt "Read hello.txt and reply with its one word."` streams
a `result`; a session on channel `200` (`--custom-id bf:200:start --user <owner id> --roles 10`)
is the same cost per turn.

## 5. Stop

```
bin/broodforge.exe session end <id>
bin/broodforge.exe stop
bin/broodforge.exe status
```

`stop` prints `stopped  <n>`; the `status` after it prints `error: not-running: ...`, exit 1, which
is the wanted outcome (a bot still answering a few seconds later is a bug to report).

End your sessions first: an open session at `stop` is held as `interrupted` at the next start
(ADR 0026) and someone has to `end` it. Report what ran and the lines that mattered (never the key
line), and, for step 2b, that the bot is left running with its PID file and log names.
