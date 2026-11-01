---
date: 2026-10-30
kind: chore
related: [2026-10-21-carry-back-main-session-push-runs-check-5-read.md]  # that one is the read at push; this moves the mechanical scan earlier
---

# Carry back: the privacy scan runs at commit time, not only at push

The push scan refused a fixture placeholder (`C:/Users/<user>` written with a real name) that a
runner had committed hours earlier, and the fix was a fixup rebase of an unpushed commit. A commit-time
run of `privacy-scan.js --text` over the staged files catches the string when it is committed, while
the fix is still an edit. Base-owned, so it goes back to Hatchery as a carry-back.

Anchor: `.claude/hooks/privacy-scan.js` and `.claude/hooks/git/` (a `pre-commit` beside `pre-push`),
or the runner's commit step in `.claude/skills/commit/SKILL.md`.

2026-10-31: the first build (c98fb2a, a hook calling `--text` once per staged file) was reverted in bf96a70:
it scanned whole files and would refuse 21 committed files (the identity email in `policy.json`, emails in
`04-ux.md`, home-path false positives), `git diff --name-only` quotes unusual names so `git show :name`
failed silently, `--diff-filter=ACM` skipped renames, and one node process per file cost about 0.45 s each.

Plan: coder, then the carry-back Send (`.claude/skills/carry-back/SKILL.md`). Hooks are not copied:
`policy.json` `git.hooksPath` is `.claude/hooks/git` and each clone runs `git config core.hooksPath
.claude/hooks/git` once (the header of `.claude/hooks/git/pre-push`), so a `pre-commit` file there
reaches every clone with no install step; it must be executable in git. The base's `pre-push` stays as is.
- `privacy-scan.js` gains `--staged`: it reads `git diff --cached -z --no-renames -U0` once, scans added
  lines only, and applies the same policy-line and allow passes the push scan applies (so the identity email
  in `policy.json` and allowed names pass). It prints hits as file, line and class only, never the string,
  and exits 0 (clean), 1 (hits) or 2 (cannot read the diff).
- Add `.claude/hooks/git/pre-commit` (POSIX sh, SPDX header, a few lines, like `pre-push`): one call,
  `node "$top/.claude/hooks/privacy-scan.js" --staged`, its output to stderr, its exit status the hook's.
  [no-check: the pre-commit file is what this pass creates]
- `node .claude/hooks/test-hooks.js` gains five cases in a scratch repository with the hooks copied in:
  PV40 (two cases: a refused string added is refused, a clean commit goes) and the reviewer's three:
  PV48a (a file whose name needs quoting, such as one with a space or non-ASCII byte, holding a refused
  line is refused), PV48b (a rename of a file with the refused line added in the same commit is
  refused), PV48c (a clean line added to a file whose committed line already hits goes through).
- Add a departure row to `docs/design/decisions/0024-decanted-from-hatchery.md` (base-owned
  `.claude/hooks/git/` and `privacy-scan.js`; Carry-back cell `pending Send; <this idea>` until the
  branch exists), then the Send, then the cell reads `carry-back/<slug> at <sha7>`.
- Verify: `test-hooks.js` passes; with the hook on, the scratch commit is refused; run once over this
  repository's index with a clean tree, which exits 0.
