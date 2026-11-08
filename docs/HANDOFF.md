# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-11-07 by main session (the s000g loss explained; ADR 0029; P40 complete; windows optional)

## Now

- Account `jdorje`. Pushed at `546b48a` (P40d); unpushed: `7546e3d` (P40e's live half: the ADR's prediction held on
  dev s0008) and P40e's recording half (`--answer`; `defer`/`parallel` on 2.1.319; review in flight). The s000g loss is explained (the engine ends a subagent at a hook
  `defer`; `2026-10-23` finding, Third look) and designed (ADR 0029, spike held: `2026-11-07-spike-...-adr-0029.md`);
  the build is P40 (`08-implementation-plan-subagent-prompt.md`). `dev-live` runs the old binary: policy mode there till redeployed.
- Engine 2.1.319; some accounts have no five-hour window: parser, store (schema 6) and CLI take either shape.
- `dev-live`: s000g's `b15362c` sits on `broodforge/dev-live/s000g`, unlanded, pushed by `[End session]` with no
  venue message. `dev`: dry-run, schema 6 at its next start. Hatchery `../hatchery`: `bcc78fe` is on its `origin/main`.

## Next

1. `tester`: resolve the 2026-10-23 and 2026-11-07 spike findings (`finding` skill, promote to ADR 0029); the
   companion's "unbuilt" clauses. `coder` (P40 gaps): a killed `Agent` call's own record after a Stop; the fake
   holding two requests open; the `2026-11-07` ideas (an idle bound for a `result`-less process; the tail's agent column).
2. `runner`/`tester` carry-back Sends (`carry-back` skill): the usage-gate pool advice (`2026-11-07` idea), the
   post-edit hook trap (`2026-11-07` idea), the commit-skill departure, the heredoc-reason idea, the `--staged` scan.
3. `architect`: `auto` under the scrub (opt out on `auto` processes, undoing C34 for Bash, or `permissions.allow`); the
   `engine` label on a forced-default refusal; P35 to P37; after P40, `land` through the tool and a second turn per process.
4. `coder`: the transcript renderer; the P11 remainder; the serve snapshot; the session-start link button. Triage:
   the 2026-11-01 residue, 2026-11-02, 2026-11-07 (two ideas).
5. W6: the owner re-runs `spike/setup-host/` (README); then P23, P25. Land `b15362c` and the five carry-back branches.

## Waiting on user

- ADR 0029 placeholders (the rest approved 2026-11-07): rule 7's header sentence ("Run subagents in the foreground: a
  background subagent's permission request is dropped when your turn ends."), rule 8's summary and trace, the dropped trace.
- The critic on ADR 0028's amendment (serious 2 and 3): the idle end never lands; `[End session]` lands or pushes
  unannounced (seen live, s000g); one reading of no `--roles`. Auto mode: shape B or the relay. Ended sessions: drop
  the lock or restore Manage Roles. End wording on sight; `Question:` blocks; older items: `git show 3db52ab:docs/HANDOFF.md`.

## Open findings

`node .claude/hooks/queue-list.js findings --status open`. Live: `2026-11-07-spike-...-adr-0029.md` (P40 builds on it);
`2026-10-23-subagents-...` (resolves with P40); the 2026-10-31 auto spike (row 3); `2026-10-26-spike-windows-...` (Next 5); the 2026-10-24 session findings; the 2026-10-17 privacy scan finding (owner).
