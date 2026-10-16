---
date: 2026-10-16
kind: bug
---
# The post-edit path check warns on elided `.../` paths that exist

`.claude/hooks/post-edit.js`, the path-anchor check (`.claude/rules/docs.md`, "A path written in a
doc names a file that exists"). A sibling's docs write deep source paths with an elision,
`shared/.../routing/`, and the hook warns on nine of them; expanding every one of the eighteen
with `find` showed they all exist (scan of 2026-10-16). The project's own handoff already lists a
carry-back for this.

Fix: the check accepts a path segment `...` as "any directories" and resolves it with a glob
walk, warning only when nothing matches; or the rule names `edit.anchorExempt` in the policy file
for such docs. The first keeps the check honest and is a few lines; the second hides the docs from
the check entirely. The harness gains a case for an elided path that exists and one that does not.
