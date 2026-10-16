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
| `0023-language-for-the-long-term.md` | draft; reopens the language bundle of ADR 0020 (not the engine mechanism) under the owner's principle of 2026-10-16; seven options scored on thirteen weighted criteria, Go recommended with Rust closest; library claims unverified, awaiting the critic |
| `0023-critic-review.md` | critic review of 0023 (none blocking, six serious, nine minor); C44 refuted as bundled (`-race` needs cgo and a C compiler on Windows), C40 measured on the PC (Go 9 ms, Node 48 ms); every other claim held or was narrowed; Go upheld over TypeScript, Go against Rust turns on the owner weighting of type strength against per-change cost |

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
