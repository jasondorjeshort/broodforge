---
date: 2026-10-15
status: open
area: spike / host mechanisms (ADR 0022)
---

# C36 holds: after a login `.claude.json` carries account metadata and cached client state, no credential

## Question

Do the host mechanisms ADR 0022 assumes hold on the real hosts? `docs/design/07-spike-plan.md`
turns the ADR claims into runs. This finding is the ADR 0022 half of the spike output, one row per
claim; it stays `open` while any row is open, and when the last row closes it is `promoted_to`
ADR 0022 with the rows copied into the ADR table. ADR 0020 claims are a separate finding.

Rows are taken from the host and prerequisite table in `07-spike-plan.md`.

| Claim | Host | Verdict | When |
|---|---|---|---|
| C36 what `.claude.json` holds after a login | PC | confirmed, metadata only | 2026-10-15 |
| C30 binds per reach | droplet | open | |
| C31 settings file read-only mid-turn | droplet | open | |
| C32 SIGINT yields a `result`, then a resume | droplet | open | |
| C33 engine sandbox under the profile | droplet | open | |
| C34 subprocess scrub | PC half, then droplet | open | |
| C35 shared clone and bundle | PC half, then droplet | open | |
| C37 instance from a scheduled task | PC | open | |
| C38 config-directory lists | droplet | open | |

## What was tried

- Host, engine and fixtures as in the ADR 0020 finding of the same date: the PC, Git Bash,
  Node v22.23.2, engine `@anthropic-ai/claude-code` 2.1.296, `$S` at `AppData/Local/Temp/bf-spike`.
- C36, run straight after C25 and spending no engine turn:
  `node spike/keys.js $CFG/.claude.json`, which prints the key path of every entry and masks each
  value longer than 40 characters as `<long>`.

## Result

C36 is confirmed on the metadata side. The file is 56693 bytes and the listing is 1284 lines. No key
name contains token, refresh, secret, api, key or credential, and no access or refresh token appears
by name. The credential lives in the separate `.credentials.json` (524 bytes), which was not opened.

What the key names cover:

- `oauthAccount.*`, about thirty entries: account, organization and workspace identifiers, the
  account email, display and full name, organization role and type, rate-limit tier, billing type,
  and account and subscription creation timestamps.
- Install and migration state: `firstStartTime`, `firstStartVersion`, `migrationVersion`,
  `claudeCodeFirstTokenDate`, and several completed-migration booleans.
- Caches: `clientDataCacheSlots.*`, `additionalModelOptionsCache`, `cachedExperimentFeatures`,
  `cachedExperimentData.*`, and about 1193 lines of `cachedGrowthBookFeatures.*` feature flags.

Three values are masked and their names do not settle what they hold: `machineID`, `userID` and
`oauthAccount.organizationName`. The first two read as identifiers, but a name alone cannot prove a
value is not a secret, so that part is ambiguous rather than closed.

Values are deliberately not copied into this finding. This repository is open source, and the
listing carries account identifiers and an email address. The raw output stayed in the session
tool-results file and was not committed.

## Next

- On the plan's rule a secret keeps ADR 0022's backup exclusion for `.claude.json` and metadata
  lifts it, so the rule points at lifting. The architect should weigh two things the rule does not
  cover before amending: the file carries account identifiers and an email, and about 93% of it is a
  feature-flag cache that churns, so a backup would copy personal data and noise rather than a
  secret. That is an ADR decision, not a spike verdict.
- Whether `machineID` and `userID` are identifiers or secrets, if it changes the answer above.
- The open rows, which all wait on the droplet operator entries or the PC account entry.
