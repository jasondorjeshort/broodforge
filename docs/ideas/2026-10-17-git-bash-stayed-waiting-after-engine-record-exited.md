---
date: 2026-10-17
kind: bug
related: []
---

# Git Bash stayed waiting after `engine record` had exited

The owner's first `engine record` (2026-10-17 08:17, engine 2.1.296, PC, Git Bash from
`git-bash.exe`, the four commands of the P7 walkthrough pasted as one block: record, `ls`, the
`--fake` replay piped to `tail -3`, `echo`) never returned a prompt. Looked at from the coordinator
two minutes later: the fixture directory was complete (`fixture.json`, a six-line `turn-1.jsonl`
ending in the `result`), no `broodforge.exe`, no engine `claude.exe` and no `tail.exe` existed, and
under the interactive bash sat three forked `bash.exe --login -i` plus the ConPTY pair Git Bash
makes for a native program (`conhost.exe --headless` and `cygwin-console-helper.exe`), all from
08:17:22, with no native child. Ctrl-C did not free it; the owner closed the window. A fresh shell
ran the replay at once, and `engine run --real` in step 3 had returned normally through the same
path minutes earlier.

Which command the screen was on is not known (the owner did not see). The evidence fits bash
waiting on a console that an inherited handle kept open after the recorder's process tree was
gone, but nothing holding it was visible in the process list. Reproduce first: the same block, one
command at a time, in Git Bash, and note whether `engine record` alone returns.

Anchors: `internal/spawn/` (`same-account` on Windows: the job object per engine, the handles the
engine's process inherits), `internal/engine/record.go` (what record holds open until exit),
`docs/hatchery/reference/claude-code-on-windows.md` (where the standing rule goes once the cause is known).

## Open

- Whether it reproduces at all, and whether `--fake` piped to `tail` is the trigger rather than the recorder.

Plan: coder, reproduction first. From Git Bash on the PC, run the block's forms, three tries each,
noting which leave no prompt: `bin/broodforge.exe engine run --fake fixtures/engine/basic --prompt x
| tail -3`, the same without `tail`, `engine record` against the owner root, then the whole block
pasted at once. Split: the recorder (`internal/engine/record.go` holds only the turn file), the
pipeline to `tail`, and the kill-on-close job `spawn.JoinKillOnCloseJob` puts the CLI in
(`internal/spawn/process_windows.go`): list what that job holds during a run. Steps needing the
interactive `git-bash.exe` window go to the owner as one `docs/operator/` entry. Reproduced: a
finding with the root cause and the standing rule for
`docs/hatchery/reference/claude-code-on-windows.md` (Shells). Not reproduced: a short open finding
naming what was tried. Either deletes this file. Not a `spawn` change before a cause is known.
