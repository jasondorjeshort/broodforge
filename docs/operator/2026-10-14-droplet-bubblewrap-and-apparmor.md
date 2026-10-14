**Install `bubblewrap` and `socat` on the droplet and, on Ubuntu 24.04 only, the AppArmor profile that lets `/usr/bin/bwrap` create user namespaces.** It waits on the owner because it needs root on the droplet. The reasoning is ADR 0022 (Decision, Linux engine mechanisms) and `docs/design/decisions/0022-linux-spawn-and-handoff.md` (Operator prerequisites).

Status: waiting by construction. Nothing is installed yet and the droplet was not reachable when this was filed, so no check was made. Run `lsb_release -d` first: Debian 12 needs only the packages, not the profile.

Runs as root (or via `sudo`) on the droplet. True first: the accounts entry (`2026-10-14-droplet-users-and-sudoers.md`) is done, so the sudoers line for `/usr/bin/bwrap` names a real path.

```
sudo apt-get update
sudo apt-get install -y bubblewrap socat
```

Ubuntu 24.04 only. The profile text is the one Claude Code documents for bubblewrap, and the companion records its shape: `flags=(unconfined)` with `userns`, attached to `/usr/bin/bwrap`. Copy the text from the Claude Code sandboxing documentation rather than from memory; the sketch below shows the shape only.

```
sudo tee /etc/apparmor.d/bwrap >/dev/null
```

```
abi <abi/4.0>,
include <tunables/global>

profile bwrap /usr/bin/bwrap flags=(unconfined) {
  userns,
  include if exists <local/bwrap>
}
```

```
sudo apparmor_parser -r /etc/apparmor.d/bwrap
```

Check it took, as the `broodforge-engine` user (the profile is what makes this succeed on 24.04):

```
sudo -u broodforge-engine /usr/bin/bwrap --ro-bind / / --unshare-pid --die-with-parent /usr/bin/true && echo ok
which socat
```

Expect `ok`, and without the profile on 24.04 an `Operation not permitted` error. `spawn preflight` (not yet built, `[no-check: not yet built]`) repeats this on demand. Allow 15 minutes.

When done: delete this file. Spike C33 of ADR 0022 later shows whether the engine's own sandbox also starts under this profile; that result is recorded in the ADR, not here.
