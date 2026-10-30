# Design

What we build and why. The documents here are authoritative for the build: code follows them, and
a change to one goes through the proposal loop (`.claude/agreement/major-decisions.md`).

Planned documents (added as we go):

| File | Purpose |
|------|---------|
| `00-vision.md` | What the app is, who it is for, what "done" looks like |
| `01-requirements.md` | Functional and non-functional requirements |
| `02-domain.md` | Core concepts, vocabulary, data model |
| `03-architecture.md` | Components and boundaries, the per-turn data flow, the engine adapter, configuration format, git naming and attribution, the process model per host, failure modes |
| `03-hook-contract.md` | Companion to 03: how the engine runs the PreToolUse hook and reads its exit codes, the hook environment, path checks, the policy table per reach and mode, the decision order |
| `04-ux.md` | The Discord flows, venues, buttons and message wording, Ticket Tool as the reference; a proposal with every owner decision marked and collected at the end |
| `04-ux-session-start.md` | Sibling of 04: the header the agent's prompt opens with (`session show --header`) and the not-started flow when a project hook's runtime is missing (ADR 0025) |
| `04-ux-session-controls.md` | Sibling of 04: the driver's switches in the venue (policy mode, model and effort, the idle timeout) and the pinned header's three select menus for model, effort and mode |
| `05-debug-interface.md` | How agents drive, observe, and reproduce every feature from a terminal |
| `05-debug-interface-discord.md` | Companion to 05: the Discord adapter's drive, observe, reproduce and isolation, including the `discord` commands |
| `05-debug-interface-discord-live.md` | Sibling of the Discord companion: `serve --discord live` (the gateway, the sweep, the REST surface), the one stderr line per inbound event and its outcomes, every interaction acknowledged, the held interrupted sessions (ADR 0026) |
| `05-debug-interface-git.md` | Companion to 05: the git handoff's drive, observe, reproduce and isolation, including `worktree check\|bundle\|handoff`, the askpass helper and the commit-msg and pre-push hooks |
| `05-debug-interface-setup.md` | Companion to 05: `broodforge setup`'s drive, observe, reproduce and isolation (the steps' lines, `--check`, `setup <step>`, `--discord dry-run`, the held control address and `status`'s `config: setup`), the surface itself being ADR 0027's companion |
| `05-debug-interface-dev-instance.md` | Companion to 05: the kept development instance on the owner's PC (root, instance `dev`, dry-run, the owner's engine, this repository as its project), what distinguishes it from scratch and from the service instance, the agent's loop (the `run` skill), the owner's one credential step |
| `05-debug-interface-session.md` | Companion to 05: registration and participants, the session commands (`session start\|send\|show\|tail\|list` and the rest), the trace summaries and deciders `session tail` shows, prompt-mode defer and resume |
| `05-cli-conventions.md` | Companion to 05: the one binary and its flags, the control-socket wire format, output and exit-code rules, `--expect`, the scratch root, the shared isolation rules |
| `06-platforms.md` | Target hosts, what each asks of every part, the sharing strategy between hosts, and the major choices bundled into proposal loops |
| `07-spike-plan.md` | The spike that tests claims C25 to C39 of ADRs 0020 and 0022 before application code: per claim the host, the owner step first, the commands, the evidence, the fallback; the throwaway `spike/` code and the fixtures kept; cost; exit criterion |
| `07-spike-droplet.md` | Companion to 07: the droplet half of the spike: packages and layout under the instance root, the bwrap route for setup and hook mode, and the claims that need the two users (C30 to C35, C38, C29, the droplet repeats of C27 and C39) |
| `08-implementation-plan.md` | The ordered coder passes for `cmd/` and `internal/`, from the toolchain to the held Linux backend: per pass what it implements, the files, the proof command, dependencies, size, and whether the owner sees its output; the facts the findings hold that the design docs do not |
| `08-implementation-plan-discord.md` | Companion to 08: the Discord passes P11 to P12d (the dry-run renderer and outbox, the live gateway, model and effort from Discord, the header's select menus, the held sessions, the audited refusals and the inbound log), under 08's conventions |
| `08-implementation-plan-session-workflow.md` | Companion to 08: the passes P15 to P19 that follow ADR 0025 (the session never pushes, the push scan, the workflow copy and probe, the hook child chain, injection in a session), under 08's conventions |
| `08-implementation-plan-setup.md` | Companion to 08: the passes P20 to P26 that build ADR 0027 (the Windows host spike, `setup` host-neutral, the Discord steps, `setup host` on Windows, `config export` and `--from`, `setup login`, the droplet), the machines each verifies on, and which pass lands each command, under 08's conventions |
| `08-implementation-plan-engine-stream.md` | Companion to 08: the passes P28 and P29 from the 2026-10-29 triage (the `rate_limit_event` line as a `rate-limit` event with a per-credential meter; the unknown event shrunk to its type, subtype, length and head), under 08's conventions |
| `08-implementation-plan-canonical.md` | Companion to 08: the passes P30 and P31 from the 2026-10-29 triage (the canonical fetched and fast-forwarded before the clone; the review fact of ADR 0025 rule 7 measured against the sha the bot bundled at the reviewer's launch), under 08's conventions |
| `08-implementation-plan-golden-sessions.md` | Companion to 08: the pass P32 (golden sessions: a recorded session's events replayed through the dry-run renderer by `discord render --golden`, its outbox and transcript diffed against the blessed ones, `--update` to re-bless, one transcript per session, the P11 replay fixtures promoted and one golden captured on the dev root), under 08's conventions |
| `proposals/` | Draft proposals for major choices; each gets a critic pass before becoming an ADR |
| `decisions/` | Architecture decision records (ADRs), one per decision |

Major choices (language, toolkit, API, storage, protocol, build system, core dependencies) follow
the proposal, critic, ADR process in `.claude/agreement/major-decisions.md`. Technical choices are
settled by agents; what humans see and touch is settled with the user.

Decisions so far (this project's own; Hatchery's records are in
[`../hatchery/decisions/`](../hatchery/decisions/README.md)):

- [0020 Harness, language, library, toolchain](decisions/0020-harness-language-library-toolchain.md):
  the Claude Code CLI the owner installs, run as one `-p` stream-json process per turn and
  resumed by session id, behind an adapter interface with a recorded-transcript fake; TypeScript
  on Node 24, discord.js, `better-sqlite3`, npm and `node:test`; policy mode and prompt mode
  through one `PreToolUse` hook (`deny`, `allow`, `defer`); `--setting-sources user` on every
  process; usage as `modelUsage` deltas; subscriptions as a per-user config dir the user logs
  into (no stored tokens, relay off by default), API keys as env. The Agent SDK waits on a
  license exception and a clean Windows interrupt. Five spike claims it must check.
- [0021 Storage](decisions/0021-storage.md): one SQLite file in WAL mode for records, usage and
  the audit log (append-only table, scoped as forensics, with a file sink and a nightly JSONL
  export); configuration in owner-edited files with immutable project ids and a last-good
  fallback at start; scratch as a generated temp instance root kept on failure; reconciliation
  never changes a worktree state from a path check. Companion
  [backup and restore](decisions/0021-backup-and-restore.md): snapshot, backup set, restore from
  an empty host, rehearsal. Hands 0022 the harness-owned-state and worktree-placement constraints.
- [0022 Session isolation and credential storage](decisions/0022-session-isolation-and-credential-storage.md):
  on the droplet two unix users, `sudo -n` into bubblewrap with binds built from the reach, the
  settings file re-bound read-only, SIGINT through a second sudoers line, the subprocess scrub and
  the engine's sandbox as mitigations; on the PC the whole instance under a second standard
  account from Task Scheduler, gaps stated; one AES-GCM store with the key outside the backup set;
  a shared clone per session handed back as a git bundle, the bot never opening a session's
  repository; WSL2 and a per-turn cross-account Windows spawn deferred. A table of spike claims.
  Companion [Linux spawn and handoff](decisions/0022-linux-spawn-and-handoff.md): the bind list,
  sudoers, entry wrapper, signal path, clone and bundle steps.
- [0023 Language for the long term](decisions/0023-language-for-the-long-term.md): under the
  owner's principle of 2026-10-16, Go replaces 0020's TypeScript bundle for the bot, the debug
  CLI, the hook and the tests: a pinned toolchain, disgo behind a narrow `discord` package with
  arikawa as the exit, zombiezen over modernc for SQLite with the audit authorizer, x/sys job
  objects on Windows, the standard library elsewhere, one static binary per host that is also the
  sandbox entry wrapper; `-race` on both hosts; the engine
  stream never refuses an unknown field; 0020's Agent SDK route closed. Amends 0020, 0021 and the
  0022 companion.
- [0024 Decanted from Hatchery](decisions/0024-decanted-from-hatchery.md): the provenance record:
  the `Hatchery base:` line the updates patch from (ADR 0016 of the base), the licence kept, and
  the departure table of base-owned files this project edits, each with its carry-back.
- [0025 Session workflow composition](decisions/0025-session-workflow-composition.md): a Discord
  session composes the bot's contract (a precedence header, the hook, attribution, the push) with
  the repository's own workflow, copied per session from the canonical's default branch and
  injected by `--append-system-prompt-file` and `--agents`; repository `PreToolUse` and
  `PostToolUse` hooks run only as deny-only children of the bot's hook, code reach, opt-in per
  project; a session never pushes (`pre-push` and `pushurl`); every push is scanned; the driver
  process coordinates; the trailer list lives in the `commit-msg` hook alone; the PC keeps a gate
  outside the session. Skills by plugin and the landing ideas deferred with triggers; `auto` mode
  out of scope, its own proposal from its idea file. Companion
  [refusals and checks](decisions/0025-refusals-and-checks.md): the copy procedure, the runtime
  probe, the refusal texts and lifts, the open checks.
- [0026 Interrupted sessions held after a restart](decisions/0026-interrupted-sessions-held-after-a-restart.md):
  every `interrupted` row is loaded into the service's memory at start-up as a held session, so
  `end` works on it from Discord and the CLI as 03 and 04 promise, a refused handoff returning it
  to `interrupted`; every refusal the service returns is audited (the P12b carve-out gone); the
  gateway writes one stderr line per inbound event with its outcome and acknowledges every
  interaction; a refusal on an interrupted session has its own wording (the owner's).
- [0027 Setup in few steps](decisions/0027-setup-in-few-steps.md): one command, `broodforge
  setup`, does every scriptable step from "has the binary" to a bot online (the application
  flags, the invite, the config, the key, the credentials, the clone, the checks) and asks the
  rest with a shown default and a flag per prompt; the bot creates its category, lobby, notice
  channel and the `Trusted` role by name (Manage Roles on the invite, default yes, removed by the
  owner at the end); `setup` holds the control address while it writes; `setup host` is the
  elevated host part, on the PC creating the `broodforge` account with a generated password under
  `%ProgramData%`; `setup login`, `setup service`, `config export` and `setup --from`. Owner's
  answers 2026-10-26; amends 0022; a simple same-account PC default rejected.

The rest of `docs/` (handoff, ideas, reference, findings) is mapped in [`../README.md`](../README.md).
