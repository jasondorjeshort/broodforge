# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-21, at commit a022bd8, by main session (a session rooted in `../`, no hooks loaded)

## Now

- The history rewrite is done on `origin`; everything through the allowlist commit is pushed.
- Base updated to Hatchery `dc896c3`, ADR 0016 migration done (ADR 0024: template `CLAUDE.md`,
  `docs/hatchery/`, two departure rows, five carry-back ideas); Opus review's fixes are in.
- Best-practices pass for a build-phase Go project (finding of 2026-10-21): `gofmt -w` on every Go
  edit (proven), `git.hooksPath` set and `core.hooksPath` configured in this clone, `go.md` aligned.

## Next

1. Owner: start Claude Code inside this repo from a new shell (`gopls` on PATH) so the hooks load;
   the first coder pass records in the handoff whether subagents get the `LSP` tool.
2. Carry-backs to Hatchery (`carry-back` skill), the `Plan:` ideas of 2026-10-21: the quoted-git
   guard fix, the main-session push read, the adr amendments section, the brief escape classes.
3. Tester: resolve the net-diff finding (rewrite done); `session end` runs the closing turn from
   the `closing` fixture per the P10 remainder idea's 2026-10-17 line; then its other `Plan:` items.
4. Architect, then coder: the earlier privacy layers (a PreToolUse scan of Write and Edit content,
   a commit-time scan), now against the base's ADR 0018 pieces.
5. Triage the ideas of 2026-10-17 and the two Questions of 2026-10-21. P12 needs the bot token and
   a test guild. Architect: the surfaces ADR. Owner: the droplet and PC account entries; P14, `spike/`.

## Waiting on user

- Which name and email go on this project's commits? (`privacy.identities`, as in Hatchery's policy.)
- The two `Question:` ideas of 2026-10-21 (coder Stop hook now or on trigger; Windows facts carry-back).
- Wording: the hook no-answer phrase; first vs full names in Discord; "acts freely" short or long.

## Open findings

- `docs/findings/2026-10-17-net-diff-privacy-scan-let-named-commits-go-public.md` (rewrite done; resolve)
- `docs/findings/2026-10-15-spike-login-claims-for-adr-0020.md`,
  `docs/findings/2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`,
  `docs/findings/2026-10-15-spike-hook-flow-claims-for-adr-0020.md` (PC rows hold, droplet open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (PC rows hold, droplet, C37, key half)
- `docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay decided; L1-L4 mine)
