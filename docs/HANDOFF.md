# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-14, at commit e75f048, by main session (Hatchery coordinator)

## Now

- The base is ready for its first decant. Four commits since bc76896, each reviewed once, unpushed:
  the status line prints the working directory above the meters (takes effect at restart); ADR 0008
  (Hatchery owns ADR numbers 0001 to 0019, decant keeps every ADR and the Windows reference, a
  `policy.json` step); the Windows reference with a link-safe worktree removal and the finding that
  robocopy purges through junctions; routing by need (settled edits go to `tester` or `coder`, the
  architect decides) and a reviewer that tests destructive claims in isolation.
- Owner answers of 2026-10-14 are recorded in ADR 0008 and on the verify-agent idea (`Waits:` on
  proposal 0003).

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

## Waiting on user

- Push of the four commits.
- Deletion of the merged `task/T94` branch on GitHub; the guard denies branch deletion for every
  session here, so it is a plain terminal or the GitHub UI.

## Open findings

- `docs/findings/2026-10-11-bash-heredoc-apostrophes.md`
