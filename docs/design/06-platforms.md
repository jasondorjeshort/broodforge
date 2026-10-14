# Platforms

Date: 2026-10-14. Status: first draft, written from `00-vision.md` (hosting corrected the same
day: both hosts are first-version targets), `01-requirements.md` and `02-domain.md`. This doc
names the hosts, the parts of Broodforge, what each host demands of each part, and the major
choices still open, bundled into proposal loops. It decides nothing a proposal must decide; where
it had to assume something to be written at all, the assumption is listed at the end.

The timeline is days, not months. Every proposal below is asked to prefer the option that gives a
working first version soonest and can be replaced behind an interface later, and to say what the
faster option costs.

## The two hosts

| Host | What it serves | What it must run | Facts that bear on the design |
|---|---|---|---|
| The owner's Windows 11 PC | The Broodforge project's own channel; the owner and a few trusted people drive | One Broodforge instance; git; Claude Code; the owner's own interactive Claude Code sessions at the same time | Behind NAT, no inbound port; may sleep or reboot, so sessions get interrupted (R-17); shares `~/.claude` with the owner's own sessions; long paths, junctions, PowerShell and Git Bash quirks in `../reference/claude-code-on-windows.md` (N-04); no root-style user separation without an admin console |
| A Linux droplet | Utopia's channels; the full trust range of the vision, trolls included | One Broodforge instance; git; Claude Code; later Utopia's own test runs (R-47) | Public IP, always on; headless, so no desktop keychain; root available, so users, namespaces and systemd units are usable; the box the vision calls the biggest hole (N-16), so reach enforcement (N-14) matters most here |

Both instances connect outward to Discord's gateway; nothing listens on a public port in the first
version. A later web site (R-10) is a droplet concern and a separate decision.

Each instance is one Discord bot application with its own bot token, configured with only the
channels it serves and ignoring every other channel; two instances may serve the same Discord
server (R-11 is one server per instance, not one instance per server). This lets both hosts run
against one server from day one.

## The parts of Broodforge and what each host asks of them

| Part | What it does | Host constraints that bear on it |
|---|---|---|
| Discord transport | Gateway connection, commands, streamed edits, buttons, venue creation; translates between people and the backend (N-11) | None by host; rate limits and message size are Discord's (R-21, R-22, N-25) |
| Local debug transport | The terminal-facing surface of R-50 to R-52 and N-05 to N-11; same backend as Discord | Must work in Git Bash and PowerShell on Windows and in a plain shell on Linux; no dependence on a Discord server (N-09) |
| Backend adapter | Starts and drives a harness per session, relays turns, permission requests, usage and subagent reports | Process spawning, interruption and cleanup differ by host (no POSIX signals on Windows); the shell the harness gets differs; the harness's own install and login differ |
| Session and worktree manager | Session state machine, participants, worktree creation and removal, end-of-session push or PR | Worktree removal on Windows needs the `\\?\` procedure; paths with backslashes; concurrent sessions must not share a cwd (N-26) |
| Store | Every record in the domain data model | One file set under the backup directories (N-19, N-20); crash-safe (N-27); scratch mode (N-09) |
| Audit and usage | Append-only audit log (N-17); usage record per turn, written before the result is shown (N-21) | Same storage constraints; the audit log must survive a crash mid-write |
| Config | Instance, project and channel configuration, editable on the host (R-09), outside the repository (N-03) | Paths and service management differ (Task Scheduler or a logon session on Windows, systemd on Linux) |
| Credential store | Holds API keys apart from the store, and owns each subscription's config directory without reading it (N-12, N-13; ADR 0020) | Windows has DPAPI and Credential Manager; a headless droplet has neither a desktop keychain nor a user to unlock one |
| Isolation | Enforces a channel's reach on the host (N-14, N-15) | The two hosts offer different mechanisms; this is the part that differs most |

## Sharing strategy

One code base, one language, one process per instance (N-01). Host differences live behind named
interfaces, each with a Windows and a Linux implementation chosen at start-up, and nowhere else:
process spawning and interruption, worktree removal, the credential store backend, the isolation
backend, and service installation. Everything else (transports, session logic, store, audit,
usage, config) is host-neutral code tested on both hosts by the same debug-interface commands.
A feature that works on one host only is a bug against N-01, with one allowed exception, stated
in the isolation ADR: the reach a host can enforce.

The rule for adding a host difference: it goes behind one of the interfaces above, or a new one
recorded in `03-architecture.md` [no-check: not yet written]; never an `if windows` in session
logic.

## Major choices and their proposals

Three proposal loops, numbered 0020 to 0022 (ADR 0008: this project's numbers start at 0020;
a proposal and its ADR share a number). Each is one proposal file with sections per sub-choice,
one critic pass, one ADR. Fewer loops than choices because the sub-choices below are coupled:
deciding one alone would have to assume the others.

### 0020: harness mechanism, language and runtime, Discord library, build toolchain

Question: how Broodforge drives Claude Code, and in which language, with which Discord library
and which build and run commands per host. Coupled because the harness mechanism restricts the
language (the Agent SDK ships for some languages only; a CLI subprocess admits any), the language
picks the Discord library ecosystem, and the toolchain follows the language.

Requirements: R-14, R-20 to R-24, R-26, R-28, R-32, R-38 to R-40, R-44, N-01, N-04, N-21,
N-23, N-24, N-25, N-26, N-27.

Options the proposal must at least consider:

- Harness mechanism: the Claude Agent SDK; the Claude Code CLI as a subprocess with its streaming
  JSON input and output; both behind one adapter interface. For each: how permission requests
  reach Broodforge and answers go back, how a turn is interrupted on each host, whether usage
  is reported per turn and per model, whether subagent models are reported (R-32), whether the
  credential can vary per turn (`CLAUDE_CODE_OAUTH_TOKEN`) or only per session
  (`CLAUDE_CONFIG_DIR`), how resume works (R-18, later), and whether mode, model and effort can
  be changed mid-session (facts for vision Q 5; the decision is the UX doc's).
- Language and runtime: the languages the SDK supports as first candidates; one compiled language
  as the comparison, so that the cost of the subprocess route is stated and not assumed.
- Discord library: for the chosen language, the mature library and one alternative; each judged
  on gateway reconnection, message editing and rate-limit handling, buttons and components,
  thread and private-channel creation, role lookup at the moment of an action (N-18), and
  license (N-24).
- Build toolchain: the language's standard one; the proposal names the build, test and run
  commands on each host and how the harness is installed on each.

Already constrained: the primary harness is the real Claude Code program (vision; a raw-API loop
is a later second engine, R-42); one code base on both hosts (N-01); GPL-compatible dependencies
with licenses named (N-24); the adapter interface itself, and the harness capability list the
domain doc leaves open, are written in `03-architecture.md` [no-check: not yet written] from the
facts this proposal establishes.

Carries open: vision Q 1 (both credential kinds must work, R-39), Q 3 (the library must support
threads and private channels alike), Q 5 (facts only).

### 0021: storage

Question: where the store, the audit log and the usage records live, and where configuration
lives relative to them.

Requirements: R-01, R-08, R-09, R-44, R-45, N-02, N-03, N-09, N-17, N-19, N-20, N-22, N-26, N-27.

Options the proposal must at least consider:

- An embedded single-file database (SQLite or equivalent) for store, usage and audit together.
- Plain files: one JSON or similar file per record kind, with an append-only text audit log.
- A database server (PostgreSQL or equivalent) on each host.
- For the audit log on its own: a table in the store versus an append-only file; which survives
  a crash mid-write and which the owner reads without a tool.
- For configuration: files the owner edits (R-09) with the store holding only runtime state,
  versus configuration in the store with a command to change it. The format of the files, if
  files, is `03-architecture.md`'s [no-check: not yet written].
- Scratch mode (N-09): in-memory, temp directory, or both, and how the debug interface selects it.

Already constrained: everything the domain doc's data model lists must be held; the secret of a
credential is never in the store (N-12, N-13; the credential store is 0022's); all state under
the backup directories (N-20); no feature available on one host only (N-01). Per-record identity
and relations are the domain doc's; the proposal does not redesign them.

Carries open: nothing of the owner's. Token caps (R-46) are later; the proposal says only that
totals per user, credential, channel and project are cheap to compute.

### 0022: session isolation and credential storage

Question: how a channel's reach is enforced on each host, how a credential is stored on each
host, how the harness receives it, and what a session's own tools can still see of it. Coupled
because the harness holds the credential while the agent's tools run under it: whether the
session can read the credential is decided by the isolation mechanism, not the store.

Requirements: R-27, R-30, R-31, R-35, R-39 to R-41, N-01, N-04, N-12 to N-16, N-19, N-27.

Options the proposal must at least consider:

- Isolation on Linux: a dedicated unix user per instance, per project or per session with
  filesystem permissions; systemd-run or a sandbox tool (bubblewrap or equivalent) with read-only
  and bind paths per reach; a container per session; Claude Code's own sandbox setting where it
  applies; the harness's permission rules (allowed and disallowed tools, permission modes) as the
  instruction layer under any of these.
- Isolation on Windows: a second local user account the harness runs as, with ACLs on the
  worktree root; a job object for process cleanup; WSL2 as the runtime on the PC, which would make
  the two hosts one platform at the price of reinterpreting "Windows hosting", so a recommendation
  for it goes to the owner; the harness's permission rules alone as the first-version Windows
  reach, stated as weaker (requirements, Tensions).
- Credential storage on Linux: a file encrypted with a key held outside the backup set; a secret
  service run headless; plain file with permissions, named as the weakest. On Windows: DPAPI or
  Credential Manager; the same encrypted file for parity.
- Delivery to the harness: `CLAUDE_CODE_OAUTH_TOKEN` in the harness environment per turn versus
  a `CLAUDE_CONFIG_DIR` per credential holding Claude Code's own login state; for each, what a
  Bash tool inside the session can read, on each host, and what the harness's own sandbox hides.
- Worktree placement: under the project's checkout, under a per-instance root, or under a
  per-session user's home; the idea file `../ideas/2026-10-14-sessions-run-remote-controlled-on-the-droplet.md`
  asks the same proposal whether a per-session clone from a canonical checkout is ever preferred
  to a worktree.

Already constrained: reach is read-only or code in the first version, host later (R-49); the
limit is enforced by the host, not only by instructions (N-14); the proposal must say per host
what the session can see of the credential rather than claim it is hidden (N-15); the owner's own
`~/.claude` on the PC must not be touched by Broodforge's sessions; a weaker Windows reach is
acceptable for the first version, a weaker Linux reach is not.

Carries open: vision Q 1 (if the owner ships API keys first, the subscription delivery path is
still designed, not built); Q 2 and R-36 (the git credential for push or PR is stored the same
way as a Claude credential).

## Settled without a proposal loop

These are choices, but not major ones in the working agreement's sense, or they belong to a doc
written with its own skill:

- The debug transport (CLI, local HTTP, REPL): `05-debug-interface.md` [no-check: not yet
  written] with the `debug-interface` skill, after 0020 fixes the language.
- Configuration file format, worktree branch and directory naming, the adapter interface and
  harness capability list: `03-architecture.md` [no-check: not yet written].
- Session venue, end-of-session default, author rule, switching UX: `04-ux.md` [no-check: not
  yet written] with the owner (vision Q 2 to Q 5).
- Service installation per host (Task Scheduler or logon on Windows, a systemd unit on Linux): an
  operator step in `docs/operator/` when the first instance is installed.

## Order

1. 0020 first; everything else is written in its language and against its harness facts.
2. 0021 in parallel with 0020 if usage allows: it depends on the language only to the extent that
   every candidate has a mature binding for every option, which the proposal states.
3. 0022 after 0020: the delivery options depend on which harness mechanism was chosen and on what
   it reports about its own sandbox.
4. Then `03-architecture.md`, `04-ux.md`, `05-debug-interface.md` [no-check: not yet written].

## Choices made where the docs were silent

- One Discord bot application per instance; an instance ignores channels it is not configured
  for; two instances may serve one server.
- The droplet runs a systemd-based Linux (Ubuntu LTS or Debian assumed); no proposal may depend on
  a distribution-specific feature beyond systemd and a package manager.
- No public port in the first version; both instances reach Discord outward only.
- Host differences are confined to the named interfaces in the sharing strategy.
- Three proposal loops rather than six, numbered 0020 to 0022 in the order written here.
