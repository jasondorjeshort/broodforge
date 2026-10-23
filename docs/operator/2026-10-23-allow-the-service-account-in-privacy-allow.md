# Allow the service account in privacy.allow

**Add `broodforge` and `broodforge-engine` to `privacy.allow` in the live `config/instance.json`
before the bot runs under a service account.** It waits on the owner because `privacy.allow` and the
live configuration are the owner's (the agreement's `git.md`; ADR 0025 rule 7). The push scan treats
the OS user name of the account the bot runs as as a never literal, as the base's scan does, and the
project's name is in every Go import path (`/broodforge/`), so under the `broodforge` account every
session commit that names the project would be refused at handoff with a `name` row. Nothing refuses
at load: the scan's `privacy:` row, class and location only, is how a missing entry shows. It
applies to both service-account hosts: the PC account of
`2026-10-14-pc-broodforge-account-and-task.md` and the droplet's of
`2026-10-14-droplet-users-and-sudoers.md`. The step and its reasoning are owned by
`docs/design/05-debug-interface-git.md` (the push scan, in Rules) and
`docs/design/03-architecture.md` (Configuration).

Runs as: the owner, editing `<root>/config/instance.json` of the live instance, then Git Bash in
`<repo>` with the binary built from this change (`CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`).
First: the bot is about to run, or runs, as the service account; a running bot reads the edit at
`config reload` or a restart. Time: about 5 minutes.

1. In `<root>/config/instance.json`, add or extend the `privacy` key, then load it:

```json
"privacy": { "allow": ["broodforge", "broodforge-engine"] }
```

```
bin/broodforge --root <root> config reload
```

2. Prove it on a session whose commits mention the project (any change to a Go file does, through
   its imports), before ending it:

```
bin/broodforge --root <root> worktree check <session id>
```

It took when no `privacy: <sha>  name  <location>` row is printed. Without the entry each commit
naming the project prints one, then `error: handoff-refused-privacy: ...`, exit 1.

When done: delete this file and its row in `docs/operator/README.md`.
