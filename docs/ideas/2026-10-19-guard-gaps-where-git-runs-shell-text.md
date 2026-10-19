---
date: 2026-10-19
kind: bug
related: [2026-10-17-guard-false-positives-in-audits.md]
---

# The guard does not read shell text that git itself runs, or several wrappers

`.claude/hooks/git-guard.js`. Found 2026-10-19 by the coder and the reviewer of the `rebase --exec`
fix; all predate it, and each is allowed today:

- Git subcommands that run shell text: `git submodule foreach "<cmd>"`, `git bisect run <cmd>`,
  `git difftool -x <cmd>`, `git for-each-repo --config=<key> <git args>` (runs the args in every
  repository the key lists).
- Config keys that run commands: `git -c core.fsmonitor=<cmd> status`, and `core.pager`,
  `core.sshCommand`, `diff.external`, `credential.helper` the same way.
- Wrappers not in `WRAPPERS`: `| stdbuf sh`, `| busybox sh`.
- A substitution inside `sh -c` data is not opaque: `bash -c "git reset $(echo --hard)"` passes.
- A runner fed from git output: `git log -1 --format=%B | sh` passes, since the line holds no text
  for the guard to read (the same with a commit message written in an earlier call).
- Git run from a script the guard never parses: a reviewer's `node` experiment made commits and
  pushes through `spawnSync` that the same steps in Bash were denied. The guard reads Bash and
  PowerShell lines only; whether to say so in the hook specs' "Not covered" list or to deny
  interpreters that run git is a choice.
- Data and exec commands are judged in the line's last directory; `git -C <dir>` is not followed.

Fix shape: read each command-running argument the way `rebase --exec` is now read, deny the
command-running `-c` keys outright for every caller, add the wrappers, mark substitutions in `sh -c`
data opaque. Rule numbers and spec text: `docs/hatchery/decisions/0002-hook-specs.md`.
