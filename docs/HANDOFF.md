# Handoff

Coordinator state. Subagents follow their brief, not this file.
Updated: 2026-10-16, at commit 94b1eaa, by main session

## Now

- `main` is two commits ahead of `origin/main` (`762e6c5` is pushed): the prior-art finding and
  the design edits folding its lessons in, reviewed once with all ten items fixed. HOLD 60%/75%.
- Owner is Jason Dorje Short <jdorje@gmail.com>, repo-local git identity. Push only at session
  end on his words; his UX decisions are applied.
- Prior-art verdict: build. One owner question left, in ADR 0020's 2026-10-16 amendment and
  vision question 1: whether the DM login relay counts as "Anthropic's own flow".
- Owner decided 2026-10-16 to move the spike to his own account; the login has not been re-run,
  so `$CFG` still holds the PC owner's Max login. Nothing spends an engine turn until it has.
- Spike otherwise live on the PC: C25 and C36 hold, fixtures in place. Two global npm links
  from the 2026-10-14 restore still need the owner's names.

## Next

1. Owner deletes `$CFG`, runs `spike/login.ts` as himself in a terminal, then `setup.js settings`;
   tester adds the second-login row to the ADR 0020 finding and fixes its account paragraph.
2. PC claims in `07-spike-plan.md` order: S1 (now with the AskUserQuestion observable), C27, C39,
   S2, C35 and C34 PC halves, C28, C26; each `$RUN` against `$CFG` and `$TREE`.
3. Owner answers the login-relay question; architect records it as an ADR 0020 amendment.
4. Owner runs the droplet operator entries and names the two npm link targets.
5. Droplet claims (`07-spike-droplet.md`); tester then writes a droplet host-setup reference.
6. Spike exit met: architect records refutations as ADR amendments; coder starts `src/` (engine
   adapter, fake engine, hook with `BROODFORGE_ANSWER_REASON`, outbound mask, storage, Discord).
7. Pusher pushes `main` on the owner's words, at session end.

## Waiting on user

- The C25 login as himself; the login-relay answer; npm link targets; droplet entries; the push.

## Open findings

- `docs/findings/2026-10-16-prior-art-chat-front-ends-for-coding-agents.md` (relay question open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0020.md` (C25 closed, seven rows open)
- `docs/findings/2026-10-15-spike-claims-for-adr-0022.md` (C36 closed, eight rows open)
