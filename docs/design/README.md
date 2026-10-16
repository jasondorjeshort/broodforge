# Design

Design-first: nothing gets built until it is written down here and agreed.

Planned documents (added as we go):

| File | Purpose |
|------|---------|
| `00-vision.md` | What the app is, who it is for, what "done" looks like |
| `01-requirements.md` | Functional and non-functional requirements |
| `02-domain.md` | Core concepts, vocabulary, data model |
| `03-architecture.md` | Components and boundaries, the per-turn data flow, the engine adapter, configuration format, git naming and attribution, the process model per host, failure modes |
| `03-hook-contract.md` | Companion to 03: how the engine runs the PreToolUse hook and reads its exit codes, the hook environment, path checks, the policy table per reach and mode, the decision order |
| `04-ux.md` | The Discord flows, venues, buttons and message wording, Ticket Tool as the reference; a proposal with every owner decision marked and collected at the end |
| `05-debug-interface.md` | How agents drive, observe, and reproduce every feature from a terminal |
| `05-cli-conventions.md` | Companion to 05: the one binary and its flags, the control-socket wire format, output and exit-code rules, `--expect`, the scratch root, the shared isolation rules |
| `06-platforms.md` | Target hosts, what each asks of every part, the sharing strategy between hosts, and the major choices bundled into proposal loops |
| `07-spike-plan.md` | The spike that tests claims C25 to C39 of ADRs 0020 and 0022 before application code: per claim the host, the owner step first, the commands, the evidence, the fallback; the throwaway `spike/` code and the fixtures kept; cost; exit criterion |
| `07-spike-droplet.md` | Companion to 07: the droplet half of the spike: packages and layout under the instance root, the bwrap route for setup and hook mode, and the claims that need the two users (C30 to C35, C38, C29, the droplet repeats of C27 and C39) |
| `08-implementation-plan.md` | The ordered coder passes for `src/`, from the toolchain to the held Linux backend: per pass what it implements, the files, the proof command, dependencies, size, and whether the owner sees its output; the facts the findings hold that the design docs do not |
| `proposals/` | Draft proposals for major choices; each gets a critic pass before becoming an ADR |
| `decisions/` | Architecture decision records (ADRs), one per decision |

Major choices (language, toolkit, API, storage, protocol, build system, core dependencies) follow
the proposal, critic, ADR process in `CLAUDE.md`. Technical choices are settled by agents; what
humans see and touch is settled with the user.

Decisions so far:

- [0001 Licensing](decisions/0001-licensing.md): GPL-3.0-or-later by default, LGPL for linkable components.
- [0002 Repository layout](decisions/0002-repository-layout.md): where knowledge, state, procedures,
  rules, and hooks live; skills, permissions, and continuity by `@docs/HANDOFF.md` import. Companion
  [hook specifications](decisions/0002-hook-specs.md) for the hooks and the status line.
- [0004 Adopting the Hatchery practice](decisions/0004-adopt-hatchery.md): an existing project takes
  the practice as files from `hatchery/main`, records the base commit in its own ADR, and updates by
  three-way patch; merging stays the path for decanted projects. (0003 and 0005 are proposals in
  Hatchery not yet promoted; the numbers stay theirs.)
- [0006 Hook policy file](decisions/0006-hook-policy-file.md): each project's choices for the hooks
  (who commits and pushes, deletable branches, usage lines, cap exemptions) live in
  `.claude/hooks/policy.json`, not in the scripts; the guard fails closed, the usage gate open.
- [0007 Queues, triage and one review](decisions/0007-queues-triage-and-one-review.md): `docs/ideas/`
  and `docs/operator/` as delete-when-done queues with triage blocks; a `reviewer` agent and one
  independent review per change; amends ADR 0002 where it names `backlog.md`.
- [0008 ADR numbers, and what a decant keeps](decisions/0008-adr-numbers-and-what-decant-keeps.md):
  Hatchery uses `0001` to `0019` and a decanted project starts at `0020`; a decant keeps every
  Hatchery ADR and the Windows reference, deletes proposals, findings, ideas and operator entries;
  resolves the deferral in ADR 0004 and amends ADR 0002 where it says only 0001 and 0002 are kept.
- [0009 ADR numbers when both tests pass](decisions/0009-adr-numbers-when-both-tests-pass.md): a
  tree that is both joined to the base and holds a `Hatchery base: <sha>` line numbers as adopted; the
  line wins. Amends 0008, Numbers.
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
  sandbox entry wrapper; `-race` on Linux always and on the PC after an operator step; the engine
  stream never refuses an unknown field; 0020's Agent SDK route closed. Amends 0020, 0021 and the
  0022 companion.

The rest of `docs/` (handoff, ideas, reference, findings) is mapped in [`../README.md`](../README.md).
