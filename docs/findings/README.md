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
| `2026-10-16-go-evalsymlinks-does-not-follow-windows-junctions.md` | resolved | go / windows paths | `filepath.EvalSymlinks` leaves a directory junction unresolved while `os.Readlink` reads it; the hook walks links itself and denies one it cannot read. |
| `2026-10-16-go-named-pipes-on-windows-two-stdlib-pitfalls.md` | resolved | go / windows named pipes | `os.NewFile` on a pipe races under `-race` (shared offset); a stack `Overlapped` completes into a stale copy (`ERROR_IO_INCOMPLETE`); a stack handle array for `WaitForMultipleObjects` hands the kernel freed memory (`ERROR_INVALID_HANDLE`); the pipe uses heap ops over `x/sys/windows`. |
| `2026-10-16-sqlite-facts-seen-while-building-the-store.md` | resolved | go / sqlite | A read-only open of a WAL store leaves `-wal`/`-shm`; `VACUUM INTO` works read-only and writes a `delete`-mode file; pragmas must run one statement each, not in `ExecuteScript`; a truncated store opens and fails only at its first statement. |
| `2026-10-16-bullguard-sentry-quarantined-the-binary-and-its-process-tree.md` | resolved | environment / windows antivirus | BullGuard Sentry quarantined the fresh binary, Git's tools its tree ran and their outputs as entry 681; each rebuild is judged again until the owner adds exclusions. |
| `2026-10-15-spike-login-claims-for-adr-0020.md` | open | spike / login | C25 holds twice on the PC: the piped login with `CLAUDE_CONFIG_DIR` needs no TTY; the droplet repeat is still open. |
| `2026-10-15-spike-stream-and-process-claims-for-adr-0020.md` | open | spike / stream and process | S1, S2 and C28 hold on the PC: stream shapes and `modelUsage` delta, the tree kill and resume; a prompt-mode parallel pair is deferred twice with no engine deny and the first call dropped; C29 open on the droplet. |
| `2026-10-15-spike-hook-flow-claims-for-adr-0020.md` | open | spike / hook flow | C27, C39 and C26 hold on the PC: the defer round trip (deny reason wrapped), exit 2 blocks with the hook command inserted, exit 1 and the timeout fall through, `--settings` runs the hook even with an empty `--setting-sources`; droplet repeats open. |
| `2026-10-15-spike-claims-for-adr-0022.md` | open | spike / host mechanisms | C36, C35 and the C34 PC half hold: `.claude.json` holds no credential, the shared clone and bundle carry the spike identity, the scrub hides `CLAUDE_CONFIG_DIR` from Bash and from the hook (the `transcript_path` fallback); droplet rows, C37 and the key half still open. |
| `2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` | open | design / prior art | Nothing shipped covers the pillars together; build, with four changes (outbound masking, terms record, `-p` question tools, prompt timeout) and nine confirmations. |
| `2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` | open | git / privacy check | Check 5a and rule 20 scanned the net diff, so a fix commit on top let five commits with the PC user name reach the public branch; 28 Hatchery root commits carry the machine's git identity instead of the owner's; rewrite planned. |
| `2026-10-21-claude-code-best-practices-for-a-build-phase-go-project.md` | resolved | process / claude-code | The best-practices page re-read for a project with Go code: a formatter on edit, a hooks path, `go.md` lines taken; identity and allowlist are owner steps; a lint test and a coder Stop hook filed as ideas. |
