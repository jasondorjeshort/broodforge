# Findings

Results of investigations: a root cause, a benchmark, an approach that did not work and why.
Findings are evidence, not truth: read the `status` before trusting one. Files are named
`YYYY-MM-DD-<slug>.md`, carry frontmatter (`date`, `status: open | resolved | superseded`, `area`,
optional `supersedes`, `promoted_to`), and have the sections Question, What was tried, Result, Next.
Written with the `finding` skill. When a finding is resolved, its durable part is promoted to
`reference/` or `design/` and `promoted_to` records where.

Open items are marked in the Status column; `HANDOFF.md` links the open ones.

| File | Status | Area | One line |
|---|---|---|---|
| `2026-10-14-npmrc-location-global-makes-a-plain-npm-install-destructive.md` | resolved | environment | `location=global` in the user `.npmrc` turns `npm install` global; `--no-global` pruned 296 global packages. |
| `2026-10-15-spike-claims-for-adr-0020.md` | open | spike / engine contract | C25 holds on the PC: the piped login prints the URL and takes the code on stdin; seven rows still open. |
| `2026-10-15-spike-claims-for-adr-0022.md` | open | spike / host mechanisms | C36 holds: `.claude.json` holds account metadata and caches, not a credential; eight rows still open. |
| `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` | open | design / prior art | Nothing shipped covers the pillars together; build, with four changes (outbound masking, terms record, `-p` question tools, prompt timeout) and ten confirmations. |
