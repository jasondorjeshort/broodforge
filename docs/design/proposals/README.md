# Proposals

Drafts for major choices. One file per proposal: `NNNN-short-title.md`, numbered to match the
ADR it will become. A proposal is written by the architect, challenged by the critic, then
revised and promoted to `../decisions/NNNN-short-title.md`. Keep the proposal file; it is the
record of the alternatives and the objections.

| File | Status |
|---|---|
| `0020-harness-language-library-toolchain.md` | promoted to ADR 0020 (C22 settled by the owner on 2026-10-14) |
| `0020-critic-review.md` | critic review of 0020 (two blocking, five serious, seven minor); its verification table is authoritative over the proposal body |
| `0021-storage.md` | promoted to ADR 0021 |
| `0021-critic-review.md` | critic review of 0021 (six serious, nine minor, none blocking); every objection answered in the proposal |
| `0022-session-isolation-and-credential-storage.md` | promoted to ADR 0022; every objection answered, both blockers resolved by change (settings file re-bound read-only; shared clone and bundle handoff), W0 added and chosen for Windows, WSL2 rejected for the first version |
| `0022-critic-review.md` | critic review of 0022 (two blocking, eight serious, six minor); its verification table quotes the sources and is authoritative over the proposal body; V6 and P2 refuted as written |
| `0023-language-for-the-long-term.md` | promoted to ADR 0023 (accepted); reopened the language bundle of ADR 0020 (not the engine mechanism) under the owner's principle of 2026-10-16; seven options scored on thirteen weighted criteria, Go recommended with Rust closest; library claims verified by the critic (`0023-critic-review.md`) |
| `0023-critic-review.md` | critic review of 0023 (none blocking, six serious, nine minor); C44 refuted as bundled (`-race` needs cgo and a C compiler on Windows), C40 measured on the PC (Go 9 ms, Node 48 ms); every other claim held or was narrowed; Go upheld over TypeScript, Go against Rust turns on the owner weighting of type strength against per-change cost |
| `0025-session-workflow-composition.md` | promoted to ADR 0025 (the owner answered all four calls on 2026-10-23 as recommended): how a Discord session composes the bot's contract with the repository's own workflow (instructions, agents, rules, skills, hooks) so a session is held to the project's rules without the session or the repository editing the bot's part; four options and a composition rule common to all; revised against the critic (the child chain replaces merged hooks, a per-host floor, a push-time scan); every objection answered |
| `0025-critic-review.md` | critic review of 0025 (two blocking, eight serious, four minor); its verification table is authoritative over the proposal body: V1 to V15 probed on the pinned engine, V7 refuted as written, P2 (a merged hook's `updatedInput` applied after the bot's hook judged the original) the first blocker, P1 (`Agent`, not `Task`) the second |
| `0027-setup-in-few-steps.md` | promoted to ADR 0027 (the owner answered the calls on 2026-10-26: C, the Manage Roles prompt default yes, every text as quoted): from "has the binary" to a live bot in few hand steps; four options (docs only; `setup` in the binary; plus bot-made channels; plus a simple same-account mode, rejected by the owner 2026-10-25), one row per hand action per host (status quo 49 PC, 53 droplet; C 18 and 20), `setup host` creating the PC account with a generated password, `setup login`, `setup <step>` as the change verb, the refusal table, `config export` and `setup --from`; every objection answered (11 accepted, 8 with change) |
| `0027-critic-review.md` | critic review of 0027 (two blocking, eight serious, nine minor); its verification table is authoritative over the proposal body: C4 holds (the Message Content step goes), W1 holds for registration, W3 refuted (a password-logon task gets a temporary profile), W4 unverified and miscited, W5 holds; C recommended over D |
| `0028-landing-a-session-on-the-default-branch.md` | promoted to ADR 0028 (revised 2026-10-31 against the critic: 14 objections, 9 accepted, 5 accepted with change, none rejected; the development instance's remote settled as GitHub by the coordinator on the owner's "Decide for me", the other owner items recorded as waiting): a session's work reaches the remote before and at its end with a review the bot can verify: one land procedure (`/session land`, `session land <id>`, the push word from inside the session through the hook's `defer`, a done land returned by `allow` with `updatedInput`), `"handoff": "main"` fast-forward only with a fallback and a per-channel `land` role, and the review gate; four evidence options (P31's fact pinned at a `defer` of the reviewer call, recommended; a trailer; an audit record; a bot-run review deferred with a trigger), three push-word options, three gate options; amends ADR 0025 rules 4, 7 and 8, the 0022 companion's Handoff and ADR 0024's departure table |
| `0028-critic-review.md` | critic review of 0028 (one blocking, eight serious, five minor); its verification table is authoritative over the proposal body: V1 held as worded and proved the wrong thing (the hook runs in the sandbox and cannot pin), P1 (`receive.denyCurrentBranch`) and P2 (the `allow` plus `updatedInput` resume) taken into the body |
| `0028-amendment-critic-review.md` | critic review of ADR 0028's amendment of 2026-11-02 (none blocking, three serious, seven minor): the mechanism upheld (every end refusal but `land-canonical-diverged` falls back; the ender judged; the idle end roleless); the fallback line carries an unreachable lift, the unattended consequence is unstated, the End button's effect depends on the ender; position on `worktree handoff`: keep the role unchecked and align the host's `session end` |

Template:

```
# NNNN. Title

Date: YYYY-MM-DD
Status: draft | under review | promoted to ADR NNNN | withdrawn

## Question
## Constraints that matter here
## Options
### A. ...
### B. ...
## Recommendation
## What would change this recommendation
## Critic's objections and responses
```
