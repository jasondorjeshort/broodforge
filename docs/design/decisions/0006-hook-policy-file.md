# 0006. Hook policy lives in one file beside the hooks

Date: 2026-10-13
Status: accepted

Record: proposal `../proposals/0005-rules-that-bit-in-dystopo.md` (option B, and its two extra
items: the PowerShell matcher and the base harness as a contract); Dystopo's ADRs 0020 and 0021
(its alignment with the base and its `git-guard` exceptions); Dystopia's `CLAUDE.md`, section
Git & Commits. Behaviour of each script: the companion `0002-hook-specs.md`. ADR 0007 is the docs
half of the same sitting.

## Context

Three projects run these hooks. Hatchery is the base; Dystopo was its source and has since moved
ahead; Dystopia commits straight to `master` and pushes through one designated agent. Each
differs in who may commit, who may push, which branches may be deleted, where the usage lines sit
and which docs are exempt from the 300-line cap. Dystopo held those differences as edits inside
the scripts (`PROTECTED`, `isTaskRef`, `HOOKS_PATH`, `CAP_EXEMPT`, its own gate at 75% and its own
cache file name). That made every `update-from-hatchery` patch a conflict on the project's own
choices, and the base fell behind because improving it meant re-applying each project's edits.

Other faults found while merging, all fixed by this change: the guard split command text on spaces,
so a commit message containing `git` or ` -n` could trip a rule and every deny carried a note
about quoted text; the base matched the Bash tool only, so git run through PowerShell went
unchecked; the guard relied on `onFailure: "block"`, which older Claude Code lacks, so a crash there
was non-blocking; the status line wrote nulls over good numbers whenever `rate_limits` was absent;
the gate's thresholds were duplicated in the status line.

## Decision

### The file

`.claude/hooks/policy.json`, read by `.claude/hooks/policy.js` for `git-guard.js`, `usage-gate.js`,
`post-edit.js` and `.claude/statusline.js`. The scripts hold mechanism; the file holds choices. The
base ships one reproducing its own behaviour; `.claude/hooks/policy.examples/` holds `dystopo.json`
and `dystopia.json`, each with a `$comment` saying what it encodes.

| Key | Type | Base value | Meaning |
|---|---|---|---|
| `git.commit.subagents` | `"any"`, `"none"` or agent names | `["runner"]` | Subagents that may commit. The main session always may. |
| `git.commit.protectedBranches` | branch patterns | `[]` | No subagent commits while the current branch matches. |
| `git.push.subagents` | as above | `["pusher"]` | Subagents that may push. |
| `git.push.branches` | branch patterns | `["*"]` | Every branch a subagent's push updates must match. |
| `git.push.mainSession` | boolean | `true` | Whether the main session pushes. |
| `git.push.forceWithLease` | branch patterns | `[]` | `--force-with-lease` is allowed, for anyone, when every target matches; otherwise it is a force push. |
| `git.branchDelete.branches` | branch patterns | `[]` | `branch -D` and `push --delete` are allowed only for these. |
| `git.branchDelete.requireMergedPr` | boolean | `false` | Also require a merged pull request on GitHub (`gh`, fails closed). |
| `git.hooksPath` | path or `null` | `null` | The one value `core.hooksPath` may be set to; `null` allows none. |
| `usage.hold` / `usage.deny` | percent | `50` / `80` | No new subagents at `hold`; stop at `deny`. `hold` may not exceed `deny`. |
| `usage.fiveHour` / `usage.sevenDay` | `{ "hold", "deny" }`, each percent or `null` | both `null` | Per-meter thresholds; `null` inherits `usage.hold` / `usage.deny`. Each meter is judged by its own pair and the worse verdict wins (ReGenesis, its ADR 0004). |
| `usage.staleSeconds` | seconds | `7200` | Older usage data is stale. |
| `usage.whenStale` | `"allow"` or `"deny"` | `"allow"` | What the gate does with stale or missing data. |
| `usage.holdAdvice` / `usage.denyAdvice` | text | the base's wording | The sentence after the threshold in each deny. |
| `edit.capExempt` | path patterns | `[]` | Docs exempt from the 300-line cap. |
| `edit.anchorExempt` | path patterns | proposals, decisions, findings | Docs whose backtick-quoted paths are not checked: closed records that name paths as they were. |
| `edit.formatters` | extension to argv | `{}` | Formatter per file extension, e.g. `".rs": ["rustfmt"]`. |

Branch and path patterns match the whole name; `*` matches anything, `/` included. A missing key
takes the base value, so a project writes only what differs. Keys starting with `$` are ignored at
any level. An unknown key or a value of the wrong type makes the file malformed, so a typo such as
`requireMergedPR` is an error rather than a silent default. `node .claude/hooks/policy.js` prints
the effective policy or the error.

The file is the project's. `update-from-hatchery` keeps the project's copy on any conflict;
`adopt-hatchery` sets it at adoption. Caps that are conventions of the base itself (300, 40 and
150 lines) stay constants in `post-edit.js`; only exemptions are policy.

### Fail modes

| Script | File absent | File malformed | Script crashes |
|---|---|---|---|
| `git-guard` | base values | denies every command containing a git invocation, naming the error; other commands pass | exit 2 with the reason on stderr; `onFailure: "block"` stays as a second layer |
| `usage-gate` | base values | base values, error on stderr | exit 1: non-blocking, so the launch goes ahead |
| `post-edit` | base values | base values | swallowed; no output |
| status line | base values | base values | flags omitted; the line still prints |

Why the split. A wrong git command can be irreversible (a force push, a deleted branch) or
invisible (a skipped git hook), so the guard fails closed, and a policy it cannot read is a policy
it cannot honour. An unwanted subagent costs quota, which the status line shows and the user can
stop; a broken gate that blocked every launch would stop all delegation for a fault in a budget
check. So the gate fails open and has no `onFailure`. The same reasoning sets `whenStale` to
`allow` in the base; a project that would rather stop than guess sets `deny`, and then delegation
waits until a session's status line has written fresh data (the hook never runs `claude -p`).

### The HOLD override

`~/.claude/usage-gate-override.json`, `{"until": <epoch seconds>, "reason": "<who asked and
why>"}`, lifts HOLD (never the deny line) until `until`, and the gate prints the reason each time it
is used. It is written only on the user's explicit instruction. The file is machine-wide, like the
usage cache beside it: one override lifts HOLD for every project on the machine, not only the one
where it was asked for, so the reason should name the scope the user gave. The deny text no longer
explains how to write it; it points here, because a deny message is read by the agent that was
refused.

### Test seams

`HOOK_POLICY_FILE` names another policy file, `GIT_GUARD_MERGED` replaces the `gh` call, and
`POST_EDIT_FORMATTERS_JSON` adds formatters. They are read from Claude Code's environment, which a
command run through a tool cannot change; `test-hooks.js` sets them per case.

### Module type

The scripts are CommonJS, and `.claude/package.json` says so (`"type": "commonjs"`). Without it, a
project whose root `package.json` declares `"type": "module"` makes Node load them as ES modules,
`require` throws, and the guard's exit 2 then blocks every shell call (seen in Dystopia on
2026-10-13, the day the hooks were first installed there).

### Deferred

- Per-rule advice text for git rules 5 to 17 stays in the script, worded for any project. Revive
  when a project needs different wording, as an `advice` map keyed by rule number.
- The pusher agent's check 7 keeps its own list of cap exemptions in `.claude/agents/pusher.md`.
  Revive when the first project exempts a file: the check should read `edit.capExempt`.

## Consequences

- An update from the base patches scripts without touching any project's choices, and a project
  changes its rules by editing one JSON file that the harness can run against.
- The three projects can be compared by diffing two small files. Dystopo's ADR 0021 exceptions
  E1 to E5, A1 and A2 are `dystopo.json`, and the GP group of the harness runs them.
- In `dystopia.json` only `pusher` pushes among subagents: Dystopia's push cards start a fresh
  agent of that type, and every other subagent commits but never pushes.
- Dystopo's in-flight relaunch override (`DYSTOPO_GATE_OVERRIDE=1`) is not carried over; its
  example sets `deny` to 100 so the override file can lift its 75% line instead.
- The guard now reads quoted text as data. Text handed to a shell (`sh -c`, `eval`, `| bash`,
  `powershell -Command`) is still read as commands. What the reader cannot see (a variable, a
  substitution, a brace expansion, xargs input) is denied on any subcommand that can discard work
  or bypass hooks, so `git checkout "$BRANCH"` needs the name written out; `git log $(...)` still
  runs. Still unguarded: aliases already in the user's git config, scripts, and other interpreters
  running git (`node -e`, `python -c`).
- A malformed policy stops all git work until it is fixed. That is deliberate and loud, and
  `node .claude/hooks/policy.js` says what is wrong.
- More moving parts: four scripts share a loader and the guard has a shell reader of a few hundred
  lines. The harness (325 cases) is what keeps them honest; a change to either needs its cases.
  It reads no file of the repository it runs in (its Hatchery policy is a fixture inside it, its
  docs a scratch tree), so it passes unchanged in every project that copies it; the project's own
  `policy.json` is checked only for loading.
- Supersede when Claude Code passes hooks a parsed argv for shell tools (the reader could go), or
  when proposal 0003 settles a branch model that wants policy keys this file lacks.
