**Run `credential init` on each host and put the key it prints into the owner's password manager.** It waits on the owner because the key is printed once, is the only copy outside the host, and a restore from an empty host needs it. The reasoning is ADR 0022 (Decision, Both hosts) and `docs/design/decisions/0021-backup-and-restore.md` (Restore from an empty host, step 5).

Status: waiting by construction. No key exists and `credential init` is not built yet (`[no-check: not yet built]`), so the command below is the working name the ADR gives; the hosts were not reachable when this was filed and nothing was checked. Do this after the user and account entries for the host are done.

Runs as the bot's account on each host, in a shell on it: `broodforge` on the droplet (`sudo -u broodforge ...`), `broodforge` on the PC (a PowerShell in that account). Once per host and instance, not once in total: each host has its own key.

```
broodforge credential init
```

The command writes the key file the instance config names (default `/etc/broodforge/<instance>.key` at `0400 broodforge` on the droplet; `%LOCALAPPDATA%\Broodforge\<instance>.key` on the PC) and prints the key once. Copy the printed key into the password manager under a name that carries the host and the instance, for example `broodforge-<host>-<instance>-key`, before closing the terminal. Do not paste it into chat, a file in the repository or a ticket.

Check it took:

```
stat -c '%U:%G %a %n' /etc/broodforge/<instance>.key
```

On the PC: `icacls "$env:LOCALAPPDATA\Broodforge\<instance>.key"` and expect only `broodforge`. Then confirm the password manager entry opens and matches the length of the key file (32 bytes). Expect the droplet file at `0400 broodforge:broodforge`. Allow 10 minutes per host.

When done: delete this file once both hosts have a key. If a host is added later, file a new entry for it.
