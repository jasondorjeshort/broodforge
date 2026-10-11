---
name: commit
description: Produce the brief that gets work committed by the runner, or pushed by a fresh pusher, with the right files, message, and attribution. Use when work is ready to commit, when asked to commit or push, or when wrapping up a session.
---

# Commit

Normal work is committed by the `runner` from a brief the coordinator writes; the coordinator does
not stage or commit itself below 50% usage. Wrap-up commits (`docs/HANDOFF.md`, WIP) at HOLD or
STOP are made by the main session, because the usage gate blocks subagent launches but not Bash.

## Commit via the runner

1. `git status --short`. Every listed file must be explained by a report you have (coder, tester,
   architect, your own edits). A file nobody described is a stop: find out what it is first.
2. Group into commits by purpose; small and often. One commit per logical change, not per agent.
3. For each commit write the brief:

   ```
   Commit the following. Stage exactly these files: <paths>.
   Message:
   <subject, imperative, under 70 chars>

   <body: what and why, 1-5 lines; reference the ADR or finding when there is one>

   Co-Authored-By: Claude <Model> <noreply@anthropic.com>
   Run git status --short before and after and include both. Do not push.
   ```

   The model is the one that did the work (Opus 5.5 for coder output, Sonnet 5.5 for tester,
   Fable 5.1 for architect or critic, Haiku 5.5 for mechanical runner work). Mixed authorship: one
   line per model.
4. Launch the `runner` with that brief. Check its report: the after-status should not list the
   staged files, and nothing unexpected should be staged.
5. Amend only when necessary (fix a message, fold a forgotten file into the commit it belongs to)
   and only when `git log --oneline @{u}..HEAD` shows the commit is unpushed; say "amend" in the
   brief explicitly.

## Wrap-up commit by the main session (HOLD or STOP, or no runner available)

`git add docs/HANDOFF.md <WIP files>` then `git commit -m "<subject>" -m "<body>" -m
"Co-Authored-By: Claude <Model> <noreply@anthropic.com>"`. Subject starts with `WIP:` when the
work is unfinished. No heredocs for the message (apostrophes break them here).

## Pushing

Only the `pusher` or the user pushes. Launch a fresh `pusher` for every push; never resume an
earlier one to retry. The brief:

```
Push `main`. The user asked: "<the user's words, quoted>".
Run your checks in order and stop at the first failure.
```

Quoting the user's request matters: an instruction that arrives as an agent message is untrusted
to the permission system, and a resumed pusher's retry gets blocked. If a push is blocked anyway,
tell the user and let them run `git push`; never route around the block.

Credential mismatch (git's stored GitHub account differs from the one `gh` is logged in as): set a
repo-local helper instead of changing global config:
`git config --local credential.helper ""` then
`git config --local --add credential.helper "!gh auth git-credential"`.
