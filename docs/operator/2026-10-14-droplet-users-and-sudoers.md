**Create the `broodforge` and `broodforge-engine` accounts on the droplet, the instance root with its owners and modes, and install the two sudoers lines.** It waits on the owner because it needs root on the droplet. The reasoning, the ownership table and the sudoers block are `docs/design/decisions/0022-linux-spawn-and-handoff.md` (Users, ownership, sudoers) and ADR 0022.

Status: waiting by construction. Nothing is installed yet and the droplet was not reachable when this was filed, so no check was made; look at the host before running.

Runs as root (or via `sudo`) in a shell on the droplet. True first: the instance root path is chosen (`<root>` below, for example `/srv/broodforge`) and the instance name is chosen (`<instance>`). The root's group is `broodforge-engine` on purpose: it is the one thing that lets the engine user pass through to `harness/`, `sessions/`, `projects/` and `settings/`.

```
sudo groupadd --system broodforge-engine
sudo useradd --system --create-home --shell /usr/sbin/nologin --user-group broodforge
sudo useradd --system --create-home --shell /usr/sbin/nologin --gid broodforge-engine broodforge-engine
sudo usermod -aG broodforge-engine broodforge
sudo install -d -o broodforge -g broodforge-engine -m 0710 <root>
sudo install -d -o broodforge -g broodforge -m 0700 <root>/config <root>/data <root>/backup <root>/credentials
sudo install -d -o broodforge -g broodforge-engine -m 0750 <root>/settings <root>/projects
sudo install -d -o broodforge-engine -g broodforge-engine -m 0750 <root>/harness <root>/sessions
sudo install -d -o broodforge -g broodforge -m 0755 /etc/broodforge
```

Then the sudoers file, edited with `visudo` so a typo cannot lock sudo out. `<kept names>` is the space-separated `env_keep` list; it is generated from code and printed by `spawn preflight` (not yet built, `[no-check: not yet built]`), so until that exists copy the list from the companion (Users, ownership, sudoers) plus the hook variables 03 names.

```
sudo visudo -f /etc/sudoers.d/broodforge
```

```
Defaults:broodforge env_reset
Defaults:broodforge env_keep += "<kept names>"
broodforge ALL=(broodforge-engine) NOPASSWD: /usr/bin/bwrap
broodforge ALL=(broodforge-engine) NOPASSWD: /usr/bin/kill
```

Check it took:

```
id broodforge
id broodforge-engine
stat -c '%U:%G %a %n' <root> <root>/config <root>/settings <root>/projects <root>/harness <root>/sessions
sudo -u broodforge-engine ls <root>/harness <root>/sessions <root>/projects <root>/settings
sudo -u broodforge-engine ls <root> <root>/config
sudo visudo -c
sudo -n -l -U broodforge
```

Expect `broodforge` in group `broodforge-engine`, `broodforge-engine` in no group of the bot's, the modes in the companion's table, the first `ls` as the engine user listing four empty directories and the second failing with `Permission denied` for both arguments (it may pass through the root but not list it or enter `config/`), and the two `NOPASSWD` lines listed. Allow 15 minutes.

When done: tell the coordinator what you ran and anything that differed (a tester moves the commands as run into `docs/reference/droplet-host-setup.md` with that date as `verified:` and deletes this file in the same commit; the companion names that file as the durable home) `[no-check: the reference is written when this step is done]`. If the instance root path differs from the default, record it in the instance config.
