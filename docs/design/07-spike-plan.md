# Spike plan: claims C25 to C39

Date: 2026-10-14. Status: plan, written from the claims tables of ADR 0020 and ADR 0022 (with
`decisions/0022-linux-spawn-and-handoff.md`), `03-architecture.md`, `03-hook-contract.md`,
`05-debug-interface.md` and the operator queue. The spike runs before application code: its
throwaway runner precedes the adapter of 03 and the fake harness of 05, the fixtures it records
are the fake's first input, and `engine run --real` is the production form of that runner, not
what the spike runs. Its output is the findings (`finding` skill), one per ADR and the ADR 0020 one split by topic into login, stream and process, and hook flow, each `open` with
a row per claim for as long as any row is open, `promoted_to` its ADR when it resolves, the rows
copied by the coordinator into the ADR tables; this plan does not change when a claim falls, the
ADR's own fallback or an amendment does. S1 and S2 are the unnumbered items of ADR 0020's spike
paragraph, included because every parser and the Windows interrupt depend on them. The droplet
half of the plan is `07-spike-droplet.md`; the names below apply there too.

## Order, hosts, prerequisites

Ranked by what hurts most if refuted; the PC group runs first because it needs only the owner's
own Claude Code login, opened by C25 because every PC run uses the directory it creates. ADR
0020's claims are stated for both hosts, so C27, C39's exit-2 run and a resume after an
interrupt repeat on the droplet. "Turns" counts real-engine processes the claim needs on that
host; each spends the owner's subscription.

| Rank | Claim | Host | Owner must have done first | Turns |
|---|---|---|---|---|
| 1 | S1 stream shape and `modelUsage` delta | PC | C25 (the login) | 2 |
| 2 | C27 defer round trip | PC, then droplet | C25 on each | 4 per host |
| 3 | C39 hook exit codes | PC; exit 2 again on the droplet | C25 on each | 3 to 5, then 1 |
| 4 | S2 Windows tree kill and resume | PC | C25 | 2 |
| 4 | C32 SIGINT yields a `result`, then a resume | droplet | the droplet entries, C25 there | 2 |
| 5 | C30 binds per reach | droplet | the droplet entries | 0 |
| 6 | C31 settings file read-only mid-turn | droplet | the droplet entries, C25 there; a key for the key half | 1 per kind |
| 7 | C35 shared clone and bundle | PC half today; droplet | the droplet entries | 0 |
| 8 | C38 config-directory lists | droplet | the droplet entries, C25 there | 3 |
| 9 | C34 subprocess scrub | PC half today; droplet | the droplet entries; an API key for the key half | 1 per host |
| 10 | C28 parallel calls in prompt mode | PC | C25 | up to 4 |
| 11 | C26 `--settings` with `--setting-sources` | PC | C25 | 2 |
| 12 | C33 engine sandbox under the profile | droplet | the droplet entries, C25 there | 1 to 2 |
| 13 | C25 headless login, C36 `.claude.json` | PC today; droplet | the owner pastes the code | 0 |
| 14 | C37 instance from a scheduled task | PC | `2026-10-14-pc-broodforge-account-and-task.md`, C25 as that account | 3 |
| 15 | C29 droplet per-turn overhead | droplet | the droplet entries, C25 there | 3 |

The droplet entries are the files in `docs/operator/` whose slug starts `droplet-` (the
Broodforge install step is not needed: `<bf>` for the spike is the `spike/` checkout, readable by
both users) [no-check: the spike directory is created by the coder]; the spike adds `strace` to
the packages, the companion says how. C25 on the droplet is the relay shape of the login; each
host login is one more device on the owner's subscription.

## Shared setup

Names. `$S` is the spike scratch root: `$TEMP/bf-spike` on the PC (Git Bash; `$LOCALAPPDATA/Temp`
when `TEMP` is unset), `<root>/sessions/spike` on the droplet. `<bf>` is the compiled `spike/`
checkout on both hosts. `$CFG` is the spike config directory C25 logs into: `$S/cfg` on the PC,
`<root>/harness/spike` on the droplet, created by `harness-create`; the owner's `~/.claude` is
never read. `$TREE` is `$S/tree` on the PC, a throwaway git repository with one committed
`hello.txt`, and the session clone on the droplet; `$S/canon` is a second repository with the
branch `base` (C35). `$RUN` is `node <bf>/run.js`. `$FLAGS` is ADR 0020's command line: `-p
--output-format stream-json --verbose --input-format stream-json --permission-mode default
--setting-sources user --strict-mcp-config --mcp-config <bf>/share/empty/mcp.json`, plus
`--model haiku` (the alias; the full id if refused) on every turn that does not test a model
behaviour. `$CFG/settings.json` is written from `<bf>/settings.template.json`: one `PreToolUse`
hook, empty matcher, `timeout` 10, command `node <bf>/hook.js`, forward slashes on both hosts.

Fixture commits. The `git-guard` hook denies `git commit` from every agent but the runner in any
repository, and a commit handed to a shell inside the command line counts (`.claude/rules/docs.md`,
last rule). The guard protects project commits; the spike's commits are scratch fixtures, so every
one of them (the `hello.txt` commit in `$TREE`, the `base` commit in `$S/canon`, C35's commit in
the clone on either host) is made by `node <bf>/setup.js <step>` through `child_process`, never
from the Bash tool and never inside `sh -c`; on the droplet `setup.js` runs its git through
`bwrap.sh` (companion). Reviews and hook tests meet the same friction;
`docs/ideas/2026-10-14-git-guard-could-scope-commits-by-repository-location.md` records the base
change that would remove it.

The hook. `hook.js` chooses its behaviour from the file `spike-hook.json` in the directory of
`BROODFORGE_HOOK_LOG` (`{"mode":"allow|deny|defer|exit1|exit2|sleep","answer":"<tool_use_id>:
allow|deny"}`), not from a variable, so the droplet sudoers keeps only the names the companion
lists; `deny` returns the reason `Denied by Broodforge: spike deny`, `exit2` prints `hook: spike
exit 2` on stderr. It logs every call (`tool_name`, `tool_use_id`, the names of every
`BROODFORGE_*`, `CLAUDE_*` and `ANTHROPIC_*` variable it sees, never a value) to
`BROODFORGE_HOOK_LOG`, which is `$S/hook.log` on the PC and `<sess>/home/hook.log` on the droplet.
Every engine process also gets `BROODFORGE_SESSION=spike`, `BROODFORGE_MODE`, `BROODFORGE_REACH`,
`BROODFORGE_TREE`, `BROODFORGE_CANONICAL`, `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`,
`DISABLE_AUTOUPDATER=1` and the four `GIT_AUTHOR_*` and `GIT_COMMITTER_*` names set to a spike
identity.

Every run. `$RUN` records `claude --version` first; that version is the pin `engine.version`
starts from. Every run has a deadline (`--deadline`, default 120 s): at it `$RUN` kills the engine
tree (S2's kill on the PC, the companion's Interrupt step 4 on the droplet) and records the
outcome `deadline` with the last stdout line. One such outcome is foreseen: engine 2.1.296
defaults `--permission-prompts` to `host`, so a call no hook answers in `-p` with stdin closed may
emit a `control_request` line and wait for an answer that never comes. A run that ends so is
recorded as `waits for a prompt`; the remedy ADR 0020's amendment names is `--permission-prompts
none`: the turn is repeated with it, and the flag joins `$FLAGS` for every later run. Each claim
below: statement, run, the evidence that settles it, what a refutation changes; commands are Git
Bash on the PC and `sh` on the droplet.

## Runs on the PC today

**C25.** The login command, without a TTY and with `CLAUDE_CONFIG_DIR` set, prints the URL and
accepts the pasted code on stdin. Run: `CLAUDE_CONFIG_DIR=$CFG node <bf>/login.js`, which spawns
`claude auth login` with piped stdio (the relay's shape), prints what the engine printed, reads
the code from its own prompt and writes it to the engine's stdin; if `auth login` is not a
subcommand, the same with `claude /login`. The owner runs it, since the code is theirs; the coder
files the operator entry with the exact command when `login.js` exists [no-check: created by the
coder]. Evidence: the URL on the engine's stdout; exit 0 after the paste; `ls -la $CFG`;
`CLAUDE_CONFIG_DIR=$CFG claude auth status` (if it exists) saying logged in. Refuted (needs a
TTY): the retry is `login.js --console` on the PC, which inherits the owner's console for stdout
and stderr and pipes only stdin (Git Bash has no `socat`), and `socat - EXEC:'claude auth
login',pty,setsid,ctty` on the droplet; if that fails too, ADR 0020 Credentials loses the relay
sentence and the user runs the login in a terminal with the directory set, the ADR's other path.

**C36.** What `.claude.json` holds after a login. Run, after C25: `node <bf>/keys.js
$CFG/.claude.json` [no-check: created by the coder], which prints the key list and every value
longer than 40 characters as `<long>`. Evidence: whether any value is a token or refresh secret or
only account metadata. Either way changes only ADR 0022's backup exclusion: a secret keeps it,
metadata lifts it.

**S1.** The stream has the shapes 03's `EngineEvent` type reads and `modelUsage` is cumulative
across resumes. Run: hook `allow`; `$RUN --cfg $CFG --cwd $TREE --out $S/fx/basic --prompt
"Read hello.txt and reply with its one word."`; then `$RUN ... --resume <session id from
init> --prompt "Say the word again."` into the same fixture directory as `turn-2.jsonl`.
Evidence: `system/init` carries `session_id`, `model`, `tools`, `capabilities`; an `assistant`
line carries `message.id`, `message.model`, `message.usage` and a `tool_use` block; a `user`
line carries `tool_result`; `result` carries `stop_reason`, `num_turns`, `permission_denials`,
`usage` and `modelUsage` keyed by model id, and turn 2's `modelUsage` totals exceed turn 1's;
the parser prints `unknown` for none of the kinds 03 lists. Also observe, in a third turn, a
prompt that tells the agent to call AskUserQuestion: what the stream carries under `-p` (an
error `tool_result`, a `permission_denials` entry, or no call at all), recorded in the ADR 0020
finding's S1 row; a usable call withdraws that ADR consequence. Refuted: 03's `EngineEvent` type
and usage-delta paragraph change to what was seen; not cumulative means the delta is the value
itself (ADR 0020, Usage and attribution).

**C27.** The `defer` round trip across a wait, four turns. Run: hook `defer`,
`BROODFORGE_MODE=prompt`; `$RUN --cfg $CFG --cwd $TREE --out $S/fx/defer --prompt "Create a
file named hi.txt containing hi."`; expect `result.stop_reason` `tool_deferred` with the
pending `tool_use_id`; then hook `answer:<that id>:allow`, `$RUN ... --resume <id>` with stdin
closed at once (if the engine waits or errors, once more with one empty user line; which input a
resume takes is part of the result); then a fresh deferred turn and its resume answered `deny`.
Evidence: exit code of the deferred process; `hook.log` showing `PreToolUse` fired on resume
with the same `tool_use_id`; `hi.txt` exists after `allow`; after `deny` the `tool_result` is an
error, its text quoted in the finding so that the `Denied by Broodforge:` prefix 03 classifies
by is checked as the model sees it (verbatim, or wrapped as C39 describes), and the model
continues to a `result`. `updatedInput` is not tested (03's hook never sends it): the testable
form is allow and deny honoured. Refuted: ADR 0020 says stop and report; the proposal's
spike-gated fallbacks are a `--permission-prompt-tool` server or the SDK callback under a
section 7 exception, both owner-visible.

**C39.** Exit 2 blocks with stderr as the result; exit 1 and a timeout do not; what the
engine's default then does in `-p`. Run three turns, hook `exit2`, `exit1`, `sleep` (sleeps 30
against `timeout` 10), each with the prompt "Read hello.txt, then write the word to note.txt,
then run `echo ok` in Bash." Evidence per turn. For `exit2`, each `tool_result` has `is_error`
and its text is one of two outcomes: equal to the hook's stderr line `hook: spike exit 2`
(blocks, verbatim), or that line wrapped by the engine as `PreToolUse:<tool> hook error: hook:
spike exit 2` (blocks, text wrapped; the reviewer observed the wrapper on 2.1.296). Wrapped is
not a refutation: 03-hook-contract's classification strips the engine's `PreToolUse:<tool> hook
error: ` prefix before its prefix test, which is the amendment 03 gets if observed (a
contains-match is not used, because the rest of the text is the model's). For `exit1` and
`sleep`, the Read succeeds and the Write and Bash results show the engine's own `-p` default: a
deny with its text (`result.permission_denials` lists which), an allow, or `waits for a prompt`,
in which case the turn repeats with `--permission-prompts none` and what that produces is the
recorded default (up to five turns); the gap between the `tool_use` and its result under `sleep`
is about the `timeout` value. Refuted (exit 2 does not block): 03-hook-contract's Failure
paragraph cannot fail closed; a `deny` decision printed on every caught error replaces it, and
03's `hook:` prefix classification moves to the deny reason. If the default allows a Write when
no hook answers, the timeout is the hook's hole: 03's timeout row becomes an audit event from the
`hook.log` gap, the `timeout` is raised, amended into ADR 0020 under C39.

**S2.** Windows: terminate the process tree by PID, then resume. Run: hook `allow`; `$RUN
--out $S/fx/interrupted --kill-after 5000 --prompt "Run \`sleep 30\` in Bash, then say
done."`; at the deadline the spike first records the engine's tree (`Get-CimInstance
Win32_Process` walked by `ParentProcessId` from the engine pid; pids and names printed), then
runs `taskkill /T /F /PID <engine pid>`; then `$RUN --resume <id> --prompt "Continue."`, with
`CLAUDE_CODE_RESUME_INTERRUPTED_TURN=1` as the second attempt if the first refuses. Evidence: no
`result` in turn 1, the `assistant` lines seen with their `message.id` and `usage`; two seconds
after the kill none of the recorded pids exists (`Get-Process -Id` on each), not the absence of
every `sleep.exe` or `bash.exe` on the machine; turn 2 reaches a `result`. Refuted (no resume
after a kill): ADR 0020 Interrupt, Windows line; an interrupt then ends the engine session and
the next turn starts a new one, 04 says so.

**C28.** A parallel-tool-call turn in prompt mode. Run: hook `defer`, `BROODFORGE_MODE=prompt`,
prompt "Create a.txt containing a and b.txt containing b. Make both tool calls in the same
message." Up to three attempts with Haiku, then one with Sonnet if it never emits two
`tool_use` blocks in one `assistant` message (up to four turns). Evidence: that message; the
`tool_result` text each call gets; whether `result.permission_denials` names both
`tool_use_id`s; whether the turn ends or continues; `waits for a prompt` means the engine's own
deny did not fire for the second call, and the attempt repeats with `--permission-prompts none`.
Refuted (a hang to the deadline, or a crash): 03's prompt-mode paragraph and ADR 0020's "fall
through to the engine's deny" give way to a hook-side rule (deny every call after the first of
one message, which needs the message id in the hook input); without that id the system prompt
append is the only mitigation and 04 says so.

**C26.** `--settings <file>` is honoured with `--setting-sources user`; whether an empty
`--setting-sources` still runs hooks given by `--settings`. Run: the hook registered in
`$S/alt-settings.json` only (`$CFG/settings.json` holds no hooks), one Read turn with
`--settings $S/alt-settings.json --setting-sources user`, then the same with
`--setting-sources ""`. Evidence: `hook.log` lines, or none, per run. Either result changes
nothing in the Decision: ADR 0022 bound the hook's file over the engine's own settings path;
C26 decides only the critic's inline alternative (`--settings` with the JSON inline). It no
longer gates `--bare` for API-key turns: `--bare` skips settings hooks, so that deferral is
withdrawn (ADR 0020, Amendments).

**C34, PC half.** Under the scrub, Bash sees no credential name and the hook still receives
every `BROODFORGE_*` variable. Run: hook `allow`, one turn "Run `env | sort | grep -E
'^(ANTHROPIC|CLAUDE|BROODFORGE|GIT_)'` in Bash and show the output." Evidence: the Bash output
lacks `CLAUDE_CONFIG_DIR`, keeps `BROODFORGE_*` and `GIT_*`; `hook.log` lists every
`BROODFORGE_*` name and says whether `CLAUDE_CONFIG_DIR` reached the hook, since hooks are
subprocesses too. The `ANTHROPIC_API_KEY` half cannot run on a subscription (setting the name
switches the engine to it): it runs on the droplet, same prompt, once the owner funds a key.
Refuted (the hook loses `CLAUDE_CONFIG_DIR`): 03-hook-contract's credential-path deny derives
the directory from `transcript_path` on stdin instead; (Bash keeps the name): ADR 0022's
mitigation line drops to "the binds hide the file, the name is visible", N-15 restated.

**C35, PC half.** The `-c safe.directory` clone form works, a commit under the four variables
carries the driver, the bundle fetches into `refs/sessions/<id>`. Run, no engine: `node
<bf>/setup.js canon` makes `$S/canon` (`git init -b base`, one commit: the branch is named,
never the host's `init.defaultBranch`); `git -c safe.directory=$S/canon clone --shared --branch
base $S/canon $S/clone`; `git -C $S/clone checkout -b spike`; `node <bf>/setup.js commit
$S/clone` (a change, then `git commit -am t` under the four variables); `git -C $S/clone bundle
create $S/out.bundle base..spike`; `git -C $S/canon bundle verify $S/out.bundle && git -C
$S/canon fetch $S/out.bundle spike:refs/sessions/spike`; `git -C $S/canon log -1 --format='%an
%ae %cn %ce' refs/sessions/spike`. Evidence: the four fields equal the spike identity. The
dubious-ownership half needs two users: droplet.

## Waits on the PC account

**C37.** The instance runs from a scheduled task with no interactive session; `taskkill /T`
ends the tree; the owner's profile is unreadable from Bash. Before: the operator entry; `<bf>`
copied to a path that account can read (`C:\Users\Public\bf-spike`, since the owner's profile is
closed to it) with Node on that account's `PATH` (a machine-wide install is; otherwise the task
action names `node.exe` in full); the owner writes a canary file `C:\Users\<owner>\bf-canary.txt`
in their own profile (the probe never touches the owner's real `~/.claude`), then logs in once as
`broodforge` into that account's `$CFG` (C25 again, a third device). Run, as the owner in an
elevated shell: `Register-ScheduledTask -TaskName Broodforge-spike` with the entry's settings
and an action of `node <copy>/run.js --out C:\Users\broodforge\spike-out --prompt "Reply with
ok."`; `Start-ScheduledTask`; a second action with `--kill-after 5000` and the `sleep 30`
prompt; a third with "Run `cat C:/Users/<owner>/bf-canary.txt` in Bash."; unregister the task
after. Evidence: a `result` in the first output file; `Get-Process -Id <pid>` empty after the
second's kill; `Permission denied` in the third's Bash output. Refuted (no turn from the task):
ADR 0022's Windows paragraph has no fallback; the options (auto-logon, or the owner's own
account) change how the owner runs the bot and go to the owner.

## The code: `spike/`

A throwaway directory at the repository root [no-check: created by the coder], never under `internal/`:
TypeScript compiled with `tsc` on Node 24 per ADR 0020 (the PC has 22 today; the finding
records which ran), no dependency but `typescript` and `@types/node`, SPDX headers as
everywhere. Files: `run.ts` (spawn with `$FLAGS`, the stdin user line, raw stdout to
`turn-<n>.jsonl` under `--out`, the parser to `EngineEvent` as JSONL, the `modelUsage` delta,
`--resume`, `--cfg`, `--cwd`, `--settings`, `--kill-after` with the per-host kill, `--deadline`,
`--spawn same|bwrap`, `--strace`, `time` per run; the miniature of 05's `engine run --real` and
`engine record`); `setup.ts` (the fixture commits above, `--droplet` through `bwrap.sh`);
`hook.ts`; `login.ts` (C25); `keys.ts` (C36); `settings.template.json`; `bwrap.sh`,
`hookmode.sh`, `engine-entry` and `strace-cfg.sh` for the droplet; `share/empty/` with `dir/`,
`CLAUDE.md` and `mcp.json`, as the companion names them. Fixtures recorded: `basic` (S1, two
turns), `defer` (C27), `interrupted` (S2), `parallel-deny` (C28), `hook-exit2` (C39). Kept
afterwards: the fixtures and the findings (one per ADR, the 0020 one split by topic). The
fixtures move to `fixtures/engine/<name>/` [no-check: created with the adapter] as 05's fake
expects, each read before the commit by an agent that did not record it, with nested JSON decoded.
Before the read the import rewrite (`go run ./internal/tools/importfixture`, P1; generalised when
`engine record` lands) replaces the recording host's home with `/home/user`, and the read confirms
no real host path, name, email, hostname or repository content survives. What is account-level or
personal and cannot be classified by a script is reported to the owner in a few lines; the owner
decides on that report, never by reading the files (the recorder redacts nothing). The rest of `spike/` is deleted in the commit that lands
`engine run --real` and `engine record`. The spike stays TypeScript as the record; its parser is ported to Go in the plan's engine pass (ADR 0023).

## Cost

Every real turn draws on the owner's subscription, the same 5-hour and weekly meters the
`usage-gate` hook reads for this project's agents, so the spike competes with the design work.
A subagent cannot run `/usage`: the coder reads both meters from `~/.claude/usage-state.json`
before and after each group and notes them in the finding; the hook refuses new launches from
the HOLD line of `.claude/hooks/policy.json`, and a spike group in flight finishes like any
in-flight agent. Real turns, Haiku, one-line prompts: PC (S1 2, C27 4, C39 3 to 5, S2 2, C28 up
to 4 with one Sonnet attempt at most, C26 2, C34 1: about 18, up to 20); droplet (C27 4, C39 1,
C32 2, C31 1 per kind, C38 3, C33 1 to 2, C29 3, C34 1 with a key: about 18); C37 (3). C25,
C36, C30, C35 spend nothing. Owner costs in their own time: a device login per host account
(PC, droplet, the PC `broodforge` account); the C33 profile toggle on Ubuntu; an API key only if
they choose to fund the key halves of C34 and C31, at whatever rate they set.

## Exit criterion

The spike is done when both findings carry a row per claim (S1, S2, C25 to C39), each `open`,
`holds` or `refuted` with the observed output quoted; no stop claim (C27, C30, C31, C35, C38's
planted files, C39's exit 2) is refuted, or the ADR amendment or proposal answering it is
recorded before any code lands under `internal/`; every recorded fixture replays through the spike's
own parser with no `unknown` for the kinds 03 lists; and the engine version run is the pin on
both hosts. Rows that may stay `open` at exit, each with a Next line in its finding: C38's
refresh (it cannot be forced), the key halves of C31 and C34 (until the owner funds a key), and
C33's negative half on Debian (no profile to unload). The coordinator may let host-neutral code
start on the PC findings alone and hold the `linux-sudo-bwrap` backend until the droplet rows
land; C37 gates only the Task Scheduler install.
