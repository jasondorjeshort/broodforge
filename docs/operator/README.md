# Steps only the owner can take

One file per step that an agent cannot or must not do itself, because it needs the owner's own
hands, credentials or elevation: a store or registry account, a signing key, a secret only they
hold, a permission or grant on a host, a program installed with elevation, a setting in a service
console. Not what qualifies: work an agent could do with the tools it has, a decision with no
action attached (that is a `Question:` in the idea or finding that raised it), or a defect.

This is a queue of waiting actions, and nothing builds from it: no agent, skill or loop takes an
entry here. The owner works it when they are at the keyboard, deletes each entry when its step is
done, and updates the doc that owns the step in the same sitting, so that the owning doc and not
this directory stays the record. An entry is never ticked, only deleted; a directory holding only
this README is the resting state.

## Filing an entry

An agent files one whenever a change it is making leaves a step of the kind above waiting, in the
same commit that lands the change, so the tree never carries code whose operator step exists only
in a conversation.

Files are named `YYYY-MM-DD-<slug>.md`, the date being when the entry was filed. No frontmatter.
Skim the listing first for an entry already covering the step; tightly coupled steps are one entry
in order, independent steps separate files. Every entry carries:

- An opening bold line saying what the step does, then why it waits on the owner, and the path of
  the doc that owns the step and its reasoning. Point at that reasoning; do not restate it.
- The exact command to paste, in a code block, with placeholders such as `<token>` where a secret
  goes. Never a real secret, never a value read out of a local file.
- Which account and which shell it runs as, what has to be true first, how to check it took, and a
  rough time estimate.
- What to update when it is done, when that is more than deleting the file.

Verify before filing: an entry says the step is still waiting, so check the host or the service
rather than trusting a status line, and drop a step already done.

## Index

| File | One line |
|---|---|
| `2026-10-14-droplet-users-and-sudoers.md` | Create the `broodforge` and `broodforge-engine` accounts, the instance root with its owners and modes, and the sudoers lines on the droplet. |
| `2026-10-14-droplet-bubblewrap-and-apparmor.md` | Install bubblewrap, socat and, on Ubuntu 24.04, the unconfined-userns AppArmor profile. |
| `2026-10-14-droplet-engine-node-and-broodforge.md` | Install the pinned engine as its native ELF, Node and Broodforge at paths both users read. |
| `2026-10-14-pc-broodforge-account-and-task.md` | Create the PC `broodforge` account, log in once, install Claude Code, register the Task Scheduler task. |
| `2026-10-14-credential-store-key-per-host.md` | Run `credential init` per host and store the key in the password manager. |
| `2026-10-14-spike-login-on-the-pc.md` | Log the spike's config directory into your subscription on the PC through the piped login (C25). |
