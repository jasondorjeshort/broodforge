**Install the Claude Code engine as its native ELF, Node and Broodforge on the droplet at shared paths both users can read, with the engine pinned.** It waits on the owner because it needs root on the droplet. The paths and why both users must read them are `docs/design/decisions/0022-linux-spawn-and-handoff.md` (The engine spawn, Operator prerequisites); the rule that the engine path names the binary itself, never the npm shim, is ADR 0020 (Decision, Mechanism), and the interrupt in the companion relies on the process being named `claude`, which only the ELF is.

Status: waiting by construction. Nothing is installed yet and the droplet was not reachable when this was filed, so no check was made. Broodforge has no release to install yet, so that step is `[no-check: not yet built]`.

Runs as root (or via `sudo`) on the droplet, except the staging step, which runs as your own login user. True first: the accounts and the bubblewrap entries are done (`2026-10-14-droplet-users-and-sudoers.md`, `2026-10-14-droplet-bubblewrap-and-apparmor.md`). The engine must use the system `/usr/bin/bwrap`, never a copy it ships. Pick the paths: `<engine>` (directory of the pinned engine binary), `<node>` (Node install), `<bf>` (Broodforge install), for example under `/opt/broodforge/`. Each must be `0755` root-owned, readable and not writable by both users.

```
sudo install -d -m 0755 /opt/broodforge /opt/broodforge/node /opt/broodforge/engine
sudo tar -xJf node-v<version>-linux-x64.tar.xz --strip-components=1 -C /opt/broodforge/node
```

The engine: Claude Code's native installer, run as your own user into a staging `HOME` so nothing lands in your real profile; it places the versioned ELF under `.local/share/claude/` and links `.local/bin/claude` to it. The version argument and the installer URL are the ones the Claude Code installation documentation gives; check them there rather than trusting this sketch.

```
mkdir -p /tmp/claude-stage
curl -fsSL https://claude.ai/install.sh | HOME=/tmp/claude-stage bash -s <pinned version>
sudo install -m 0755 "$(readlink -f /tmp/claude-stage/.local/bin/claude)" /opt/broodforge/engine/claude
rm -rf /tmp/claude-stage
```

`<pinned version>` is the engine version the project pins; the owner picks it from the one the spike findings were run against. Broodforge itself goes to `<bf>` by the release procedure, which does not exist yet.

Check it took. The first two lines run with a `PATH` that has no `node` at all, which is the point: the ELF must not need one.

```
sudo -u broodforge-engine env -i PATH=/usr/bin:/bin /opt/broodforge/engine/claude --version
sudo -u broodforge env -i PATH=/usr/bin:/bin /opt/broodforge/engine/claude --version
file /opt/broodforge/engine/claude
sudo -u broodforge-engine sh -c '/opt/broodforge/engine/claude --help >/dev/null 2>&1 & sleep 1; cat /proc/$!/comm'
sudo -u broodforge-engine /opt/broodforge/node/bin/node --version
namei -m /opt/broodforge/engine/claude /opt/broodforge/node/bin/node
```

Expect the pinned version from both users, `file` saying `ELF 64-bit`, the fourth line printing `claude` (the process name the interrupt walk looks for), and no directory in either path that is not world-readable and searchable. Allow 30 minutes.

When done: tell the coordinator what you ran and anything that differed (a tester moves the commands as run into `docs/reference/droplet-host-setup.md` and deletes this file in the same commit) `[no-check: the reference is written when this step is done]`, and record the three chosen paths in the instance config so the spawn binds them.
