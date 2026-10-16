# .claude/rules/ — path-scoped conventions

A rule file here is loaded only when Claude reads, writes, or edits a file matching its `paths:`
frontmatter (a YAML list of globs; see https://code.claude.com/docs/en/memory). That makes rules the
right place for conventions that matter in one part of the tree and would be noise everywhere else.
A rule file without `paths:` loads on every session like `CLAUDE.md`; do not do that here.

This tree ships two rules: `docs.md`, for `docs/**` (also the example of the format), and
`hooks.md`, for `.claude/hooks/**`.

A decanted project adds one rule per language or area once code exists, for example:

```
---
paths:
  - "src/**/*.rs"
  - "crates/**/*.rs"
---
# Rust conventions
- Format with `cargo fmt`; lint with `cargo clippy -D warnings`.
- Error type per crate; no `unwrap` outside tests.
```

Keep each rule short and concrete. Procedures belong in skills; decisions belong in ADRs; a rule is
what to do and not do while editing matching files.
