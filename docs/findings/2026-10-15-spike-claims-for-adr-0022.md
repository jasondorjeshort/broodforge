---
date: 2026-10-15
status: open
area: spike / host mechanisms (ADR 0022)
---

# C36, C35 and the C34 PC half hold: `.claude.json` carries no credential, the shared clone and bundle carry the spike identity, and the scrub hides `CLAUDE_CONFIG_DIR` from Bash and from the hook alike

## Question

Do the host mechanisms ADR 0022 assumes hold on the real hosts? `docs/design/07-spike-plan.md`
turns the ADR claims into runs. This finding is the ADR 0022 half of the spike output, one row per
claim; it stays `open` while any row is open, and when the last row closes it is `promoted_to`
ADR 0022 with the rows copied into the ADR table. ADR 0020 claims are a separate finding.

Rows are taken from the host and prerequisite table in `07-spike-plan.md`.

| Claim | Host | Verdict | When |
|---|---|---|---|
| C36 what `.claude.json` holds after a login | PC; droplet | holds on the PC, metadata only | 2026-10-15 |
| C30 binds per reach | droplet | open | |
| C31 settings file read-only mid-turn | droplet | open | |
| C32 SIGINT yields a `result`, then a resume | droplet | open | |
| C33 engine sandbox under the profile | droplet | open | |
| C34 subprocess scrub | PC half, then droplet | holds on the PC for Bash; the hook loses `CLAUDE_CONFIG_DIR` too (the plan's `transcript_path` fallback); key half open | 2026-10-16 |
| C35 shared clone and bundle | PC half, then droplet | holds on the PC; the four identity fields match | 2026-10-16 |
| C37 instance from a scheduled task | PC | open | |
| C38 config-directory lists | droplet | open | |

C36, C34 and C35 on the droplet are separate runs of the same claims and are still open.

## What was tried

- Host, engine and fixtures as in the hook-flow finding of the same date (Shared setup): the PC, Git Bash,
  Node v22.23.2, engine `@anthropic-ai/claude-code` 2.1.296, `$S` at `AppData/Local/Temp/bf-spike`.
- C36, run straight after C25 and spending no engine turn:
  `node spike/keys.js $CFG/.claude.json`, which prints the key path of every entry and masks each
  value longer than 40 characters as `<long>`.
- C35 PC half, 2026-10-16, no engine turn. `$S/canon` already existed from `setup.js pc` (branch
  `base`, `7295ef5 spike fixture: hello.txt`) and was reused. Then the plan's commands verbatim from
  Git Bash: `git -c safe.directory=$S/canon clone --shared --branch base $S/canon $S/clone`; `git -C
  $S/clone checkout -b spike`; `node spike/setup.js commit $S/clone` (the commit through
  `child_process`, under the four `GIT_*` names, `commit.gpgsign=false` and an empty `core.hooksPath`);
  `git -C $S/clone bundle create $S/out.bundle base..spike`; `git -C $S/canon bundle verify
  $S/out.bundle && git -C $S/canon fetch $S/out.bundle spike:refs/sessions/spike`; `git -C $S/canon
  log -1 --format='%an %ae %cn %ce' refs/sessions/spike`.
- C34 PC half, 2026-10-16, engine 2.1.296 as in the hook-flow finding, `$FLAGS`, `--model haiku`,
  hook `allow`, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` on the engine process as every run sets it, one
  turn into `$S/fx/scrub` with the plan's prompt; then `$RUN --replay $S/fx/scrub`. Meters (shared with
  the C28 turns of the stream and process finding and the C26 turns of the hook-flow finding): before 5-hour 25%, weekly 7%; after 27%, 7%.

## Result

C36 holds on the metadata side. The file is 56693 bytes and the listing is 1284 lines. No key
name contains token, refresh, secret, api, key or credential, and no access or refresh token appears
by name. The credential lives in the separate `.credentials.json` (524 bytes), which was not opened.
The 56693-byte file was deleted when the directory was recreated on 2026-10-16: the verdict was taken on the earlier account's file, and the key structure was not re-listed for the project owner's file.

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

C35 holds on the PC half. The clone form worked (`Cloning into '.../clone'... done.`), the commit
is `10ffddf t` with `%h %an <%ae> / %cn <%ce>` printed by `setup.js` as `Broodforge Spike
<spike@broodforge.invalid> / Broodforge Spike <spike@broodforge.invalid>`, `bundle verify` said
`out.bundle is okay`, listed `refs/heads/spike` and the prerequisite `7295ef5`, the fetch printed `*
[new branch] spike -> refs/sessions/spike`, and the deciding line, verbatim:
`Broodforge Spike spike@broodforge.invalid Broodforge Spike spike@broodforge.invalid`. The four
fields equal the spike identity. The dubious-ownership half needs two users and stays with the droplet.

C34 holds on the PC for Bash, and the hook branch the plan calls a refutation is what happens.
Session `5305d0f8-21fa-4951-9520-6dc9beb9a7c0`, exit 0 in 8.1 s, one `Bash` call
`toolu_01CV5vXrBPqnQZDCgaAV2fqH`, `hook.log` `decided` `allow`.

- The Bash output (`tool_result`, `is_error` false) lists, in this order, every `BROODFORGE_*` name
  (`BROODFORGE_CANONICAL`, `BROODFORGE_HOOK_LOG`, `BROODFORGE_MODE`, `BROODFORGE_REACH`,
  `BROODFORGE_SESSION`, `BROODFORGE_TREE`), the four `GIT_AUTHOR_*` and `GIT_COMMITTER_*` names with
  the spike values, plus `GIT_EDITOR=true`, and the engine's own names: `CLAUDECODE`,
  `CLAUDE_CODE_CHILD_SESSION`, `CLAUDE_CODE_ENTRYPOINT=sdk-cli`, `CLAUDE_CODE_EXECPATH`,
  `CLAUDE_CODE_MESSAGING_SOCKET` (a named pipe), `CLAUDE_CODE_SESSION_ATTENDED`,
  `CLAUDE_CODE_SESSION_ID`, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, `CLAUDE_EFFORT`, `CLAUDE_PID`. It
  lacks `CLAUDE_CONFIG_DIR` and `DISABLE_AUTOUPDATER`, both of which `meta-1.json` records on the
  engine's environment, and no `ANTHROPIC_*` name was set or seen. So the scrub hides the
  credential directory's name from Bash while the engine runs against `$CFG`.
- The hook's `hook.log` line lists the same six `BROODFORGE_*` names and the engine's
  `CLAUDE_CODE_*`, `CLAUDE_EFFORT`, `CLAUDE_PID` and `CLAUDE_PROJECT_DIR` (which Bash did not get), and
  no `CLAUDE_CONFIG_DIR`: the scrub applies to hooks as to Bash. This is the plan's "hook loses
  `CLAUDE_CONFIG_DIR`" branch, confirmed under C34's own prompt after the S1 note: the fallback the
  plan names is 03-hook-contract's credential-path deny deriving the directory from
  `transcript_path` on stdin. The spike hook does not log its stdin keys, so that field's presence is
  unobserved here; the design edit is the architect's.

## Next

- On the plan's rule a secret keeps ADR 0022's backup exclusion for `.claude.json` and metadata
  lifts it, so the rule points at lifting. The architect should weigh two things the rule does not
  cover before amending: the file carries account identifiers and an email, and about 93% of it is a
  feature-flag cache that churns, so a backup would copy personal data and noise rather than a
  secret. That is an ADR decision, not a spike verdict.
- Whether `machineID` and `userID` are identifiers or secrets, if it changes the answer above.
- For ADR 0022 and 03-hook-contract: the hook runs without `CLAUDE_CONFIG_DIR` under the scrub, so
  the credential-path deny takes the directory from `transcript_path` (to be confirmed present in
  the hook input) or from a `BROODFORGE_*` name the harness sets itself.
- The open rows: C37 waits on the PC account entry; the rest, and the droplet halves of C34, C35 and
  C36 and C34's key half, wait on the droplet operator entries and a funded key. Fixture `scrub`
  replays clean (`rate_limit_event` the only unknown kind) and carries the PC username in paths.
