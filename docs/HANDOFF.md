# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-11-07 by main session (the s000g loss explained; ADR 0029; P40a to P40c built; windows optional)

## Now

- Account `jdorje`. Pushed at `96e0d14` (P40b); unpushed on `main`: P40c (the adapter: the stdio flag, stdin held
  open, `Answer`, the kill at `result`, the fake's wait, `--requests`; reviewed). The s000g loss is explained (the engine ends a subagent at a hook
  `defer`; `2026-10-23` finding, Third look) and designed (ADR 0029, spike held: `2026-11-07-spike-...-adr-0029.md`);
  the build is P40, five briefs in `08-implementation-plan-subagent-prompt.md`. Delegate in policy mode till P40d lands.
- Engine 2.1.319; some accounts have no five-hour window: parser, store (schema 6) and CLI take either shape.
- `dev-live`: s000g's `b15362c` sits on `broodforge/dev-live/s000g`, unlanded, pushed by `[End session]` with no
  venue message. `dev`: dry-run, schema 6 at its next start. Hatchery `../hatchery`: `bcc78fe` is on its `origin/main`.

## Next

1. `coder` P40d (`session`: `TurnRequest.Mode`, the key and `config check`, the pending map, the prompt, the bound,
   the first-result rule, Stop, the deciders, the detector; `serve.go` wires `ExitTimeout`; mid-pass commit); then
   P40e (re-recorded `defer`/`parallel` fixtures and one live prompt-mode session: the owner's word).
2. `runner`/`tester` carry-back Sends (`carry-back` skill): the usage-gate pool advice (`2026-11-07` idea), the
   post-edit hook trap (`2026-11-07` idea), the commit-skill departure, the heredoc-reason idea, the `--staged` scan.
3. `architect`: `auto` under the scrub (opt out on `auto` processes, undoing C34 for Bash, or `permissions.allow`); the
   `engine` label on a forced-default refusal; P35 to P37; after P40, `land` through the tool and a second turn per process.
4. `coder`: the transcript renderer; the P11 remainder; the serve snapshot; the session-start link button. Triage:
   the 2026-11-01 residue, 2026-11-02, 2026-11-07 (two ideas).
5. W6: the owner re-runs `spike/setup-host/` (README); then P23, P25. Land `b15362c` and the five carry-back branches.

## Waiting on user

- ADR 0029 rule 8's `internal-error` summary and trace line (placeholders stand until changed; the rest of the ADR's
  wording and the 60-minute default were approved 2026-11-07).
- The critic on ADR 0028's amendment (serious 2 and 3): the idle end never lands; `[End session]` lands or pushes
  unannounced (seen live, s000g); one reading of no `--roles`. Auto mode: shape B or the relay. Ended sessions: drop
  the lock or restore Manage Roles. End wording on sight; `Question:` blocks; older items: `git show 3db52ab:docs/HANDOFF.md`.

## Open findings

`node .claude/hooks/queue-list.js findings --status open`. Live: `2026-11-07-spike-...-adr-0029.md` (P40 builds on it);
`2026-10-23-subagents-...` (resolves with P40); the 2026-10-31 auto spike (row 3); `2026-10-26-spike-windows-...` (Next 5); the 2026-10-24 session findings; the 2026-10-17 privacy scan finding (owner).
