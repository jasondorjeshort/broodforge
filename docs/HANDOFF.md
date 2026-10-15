# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-15, at commit 2b83bde, by main session (Hatchery coordinator)

## Now

- The base is ready for its first decant. Three commits since 6866d8a (the README decant prompt, the
  scoped override, this handoff), each reviewed once, unpushed. 2b83bde scopes the HOLD override: a grant is one file in
  `~/.claude/usage-gate-overrides/` naming a session, a project, or the user; session is the
  default when the owner says only OVERRIDE (ADR 0006, The HOLD override). The owner's grant for
  the 2026-10-15 session is a session grant there with a 12-hour backstop; the legacy file is gone.
- Owner answers of 2026-10-14 are recorded in ADR 0008 and on the verify-agent idea (`Waits:` on
  proposal 0003). On 2026-10-15 the owner set this machine's global git identity to Jason Dorje
  Short; it is a machine fact, not a repo one.

## Next

1. Pusher pushes `main` on the owner's words; then the first decant with the README prompt.
2. Architect answers the critic reviews of proposals 0003 and 0005 (`docs/design/proposals/README.md`).
   0005 objection 1a is answered by the reference's removal procedure and the robocopy finding; the
   rest decides the `wave` skill, the verify-agent `Waits:`, and an ADR 0007 amendment.
3. Critic review of ADR 0006 as a whole, with the escape-hatch question in `.claude/skills/proposal/brief-critic.md`.
4. Triage `docs/ideas/` (`triage` skill): the files with no block.
5. Sibling work in their own sessions: Regenesis review and push of its update; the first two-way
   sync with the build-phase sibling (`docs/ideas/2026-10-13-first-sync-with-the-build-phase-sibling.md`).
6. The open heredoc finding's Next steps are still undone (reproduce on Linux; check the changelog).
   New evidence 2026-10-15: the coder saw a quoted heredoc collapse a doubled backslash to one in a
   Node snippet.

## Waiting on user

- Push of the three commits.
- Deletion of the merged `task/T94` branch on GitHub; the guard denies branch deletion for every
  session here, so it is a plain terminal or the GitHub UI.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
