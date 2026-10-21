# Requirements

Date: 2026-10-14. Status: first draft, derived from `00-vision.md` (the initial vision, accepted
2026-10-14) and the owner's statements in the vision conversation of that day. Changes when a proposal, an ADR or
the owner changes the vision; a requirement is never silently dropped, it is struck through with
the ADR that removed it.

How to read this doc:

- Each row is one requirement in one sentence, with an identifier that proposals, ADRs, the UX
  doc and the reviewer cite: `R-nn` is something Broodforge does, `N-nn` is a quality it must have
  or a constraint it must respect.
- `When` is `v1` for the first usable version (the vision's "first usable version" section) or
  `later` for the vision's "explicitly later" list. A tag followed by `(Q n)` is a provisional
  guess that the owner's open question n in `00-vision.md` may change.
- Where the vision was silent, the smallest reasonable reading is written down and marked `(choice)`
  so the owner can reverse it cheaply. The list of those choices is at the end.
- "The owner" is always the person who runs the Broodforge instance, never a session's driver.
  Vocabulary is defined in `02-domain.md`.

## Functional requirements

### Registration and identity

| Id | Requirement | When |
|---|---|---|
| R-01 | A Discord user registers with an in-Discord command that gives a name and an email, and Broodforge stores that pair as the git identity of that Discord account. | v1 |
| R-02 | Running the registration command again replaces the earlier name and email for that account (choice). | v1 |
| R-03 | A Discord user who is not registered cannot start or contribute to a session; Broodforge answers their attempt with how to register (choice). | v1 |
| R-04 | Broodforge reads a user's trust from the Discord roles they hold at the moment of each action and never assigns, removes or proposes a role by itself. | v1 |
| R-05 | An agent may recommend a role upgrade for a user to the owner as a message; the recommendation is never applied by Broodforge. | later |

### Projects and channels

| Id | Requirement | When |
|---|---|---|
| R-06 | A project exists only because the owner configured it on the host, naming its git repository; there is no command in Discord that imports a repository. | v1 |
| R-07 | A project has one or more Discord channels, and a channel belongs to exactly one project (choice). | v1 |
| R-08 | Each channel has its own configuration: its reach, the roles allowed to start sessions, to contribute to them and to answer permission prompts, its default permission mode, its prompt timeout, and its default backend. | v1 |
| R-09 | Project and channel configuration can be read and changed on the host without Discord and without a web site, by a file or a command (choice). | v1 |
| R-10 | A web site shows and edits project and channel configuration. | later |
| R-11 | One Broodforge instance serves one Discord server; serving several servers from one instance is not required (choice). | v1 |

### Sessions and who is in them

| Id | Requirement | When |
|---|---|---|
| R-12 | A user who holds a role the channel allows starts a session from that channel; how the start looks in Discord is settled in the UX doc with the owner. | v1 (Q 3) |
| R-13 | A session has exactly one driver, the person who started it, and any number of contributors: other permitted users who write in the session or are invited into it. | v1 |
| R-14 | Broodforge records which Discord person wrote each message in a session and hands that identity to the agent with the message, so the agent can tell people apart. | v1 |
| R-15 | A session ends when its driver ends it, when its channel's idle timeout passes with no activity, or when the owner ends it from the host (choice). | v1 |
| R-16 | Several sessions can run at once, in different channels or in the same one, each in its own worktree (choice). | v1 |
| R-17 | A session interrupted by a restart or crash of the bot is reported as interrupted in its channel and its worktree is left intact. | v1 |
| R-18 | A session interrupted by a restart can be resumed with its conversation. | later |
| R-19 | A "simple questions" thread lets anyone ask a question and get a read-only answer from a small model with a tighter token limit than a coding session. | later |

### Output in Discord

| Id | Requirement | When |
|---|---|---|
| R-20 | The agent's progress appears in Discord as it happens: edited in place where that reads well, posted as a new message where a notification matters. | v1 |
| R-21 | Broodforge sends at most about one message creation or edit per second per session and coalesces output that arrives faster, dropping nothing. | v1 |
| R-22 | Output longer than one Discord message is split or attached so nothing is silently cut off (choice). | v1 |
| R-23 | When the agent finishes a turn or waits for a person, the session's channel gets a message that notifies, not an edit. | v1 |

### Permission prompts and modes

| Id | Requirement | When |
|---|---|---|
| R-24 | When the agent asks permission for an action, Broodforge posts the request in the session with buttons, and the driver's answer is passed back to the agent. | v1 |
| R-25 | Whether contributors may also answer permission prompts is set per channel; the default is driver only (choice). | v1 |
| R-26 | A session can run in policy mode, where Broodforge answers permission prompts itself, and the channel sets whether sessions start in policy mode or prompt mode. | v1 |
| R-27 | A policy-mode answer never allows more than the channel's reach allows; a request outside the reach is denied, not escalated to a person (choice). | v1 |
| R-53 | A permission prompt nobody answers within the channel's prompt timeout (default 5 minutes) is denied, the turn finishes with its output kept, and the session stays open; the idle timeout of R-15 is separate. | v1 |
| R-28 | The driver can switch a running session between policy mode and prompt mode from Discord. | v1 (Q 5) |
| R-29 | A person can change the model or effort of a session from Discord. | later (Q 5) |

### Git, worktrees and attribution

| Id | Requirement | When |
|---|---|---|
| R-30 | Every session works in its own git worktree of the project's repository, on its own branch, created when the session starts. | v1 |
| R-31 | A subagent started inside a session may work in a worktree of its own, and its work reaches the session's branch by ordinary git operations such as cherry-pick or merge. | v1 |
| R-32 | Every commit made in a session names one registered person as author and carries a co-author line for every other person who contributed to the session and for every AI model that did work in it. | v1 |
| R-33 | The author of a commit is the session's driver at the time of the commit (choice). | v1 (Q 4) |
| R-34 | When a session ends with commits, Broodforge pushes the branch or opens a pull request as the project's configuration says; the default for a new project is a pull request (choice). | v1 (Q 2) |
| R-35 | A session's worktree is kept until its commits are pushed or opened as a pull request, then removed; a worktree with unpushed commits is kept and listed for the owner (choice). | v1 |
| R-36 | Pushes and pull requests use one git credential the owner configures per project. | v1 |
| R-37 | A registered user can store their own GitHub credential so pushes and pull requests act as that person. | later |

### Credentials and backends

| Id | Requirement | When |
|---|---|---|
| R-38 | A backend is a harness plus a credential; the first harness is the real Claude Code program, so that a Claude subscription can pay for a session. | v1 |
| R-39 | A registered user can store one or more credentials with Broodforge, each a Claude subscription login or an API key, and chooses which one their sessions use. | v1 (Q 1) |
| R-40 | A session uses one credential from start to end, chosen when it starts from the driver's credentials. | v1 |
| R-41 | A user can remove a credential they stored, and Broodforge then refuses to start a session with it. | v1 |
| R-42 | A second harness drives Anthropic's API directly with an API key, behind the same interface as the first. | later |
| R-43 | Harnesses for other AI providers exist and a user can switch between logins across them. | later |

### Token tracking

| Id | Requirement | When |
|---|---|---|
| R-44 | Broodforge records the tokens used by every turn, attributed to the session, to the person who requested the turn and to the credential that paid for it. | v1 |
| R-45 | A registered user can see their own token totals and the owner can see everyone's, from Discord and from the debug interface (choice). | v1 |
| R-46 | A token cap per user, per channel or per session stops further turns when reached; the thresholds are the owner's to set because they decide what is spent. | later |

### Later features that need a place in the design

| Id | Requirement | When |
|---|---|---|
| R-47 | A project's tests run on each commit, or on a schedule during and after a session, and their result is reported in the session. | later |
| R-48 | Broodforge works with the Ticket Tool bot, or offers the same open-a-private-channel-on-request experience itself. | later |
| R-49 | A channel can have the reach "host": its sessions may act on the host machine beyond any one repository. | later |

### Debug interface

| Id | Requirement | When |
|---|---|---|
| R-50 | Every feature above can be driven, observed and reproduced from a terminal without a Discord server, through the surface designed in `05-debug-interface.md` [no-check: written after this doc]. | v1 |
| R-51 | An agent can run a whole session through that surface: register a user, start a session as a named user with a named set of roles, send messages, answer a permission prompt, read what would have been posted and end the session. | v1 |
| R-52 | The same surface reports the state of the running instance: sessions, worktrees, pending permission prompts, token totals, recent audit events and configuration. | v1 |

## Non-functional requirements

### Hosting and ownership

| Id | Requirement | When |
|---|---|---|
| N-01 | Broodforge runs from one code base on the owner's Windows 11 PC and on a Linux VM, with no feature available on only one of them. | v1 |
| N-02 | Everything Broodforge needs runs on a host the owner controls: the bot, the checkouts, the database, the credential store and any web site; nothing depends on a service this project would run for others. | v1 |
| N-03 | Nothing in the repository names the owner's Discord server, channels, users, paths or secrets; all of that is configuration outside the repository. | v1 |
| N-04 | Windows-specific behaviour that matters (processes, worktrees, shells, shared state) follows `../hatchery/reference/claude-code-on-windows.md`. | v1 |

### Agent-debuggable, as the `debug-interface` skill demands

| Id | Requirement | When |
|---|---|---|
| N-05 | Every feature has a surface (a CLI entry point, an HTTP endpoint, a REPL command or a scripted harness) designed in `05-debug-interface.md` [no-check: written after this doc] before the feature is built, with its drive commands, how to observe state, how to reproduce a failure and how to run isolated. | v1 |
| N-06 | Every behaviour in the design has at least one drive command that an agent with only a shell can run on a clean checkout, following the doc alone. | v1 |
| N-07 | The surface reads state back as stable text an agent can grep, offers a structured form such as a `--json` flag, and exits 0 on success, 1 on failure with a one-line reason on stderr, 2 on a usage error. | v1 |
| N-08 | A failing case can be captured as one command line or one fixture file, so that a bug report is a command. | v1 |
| N-09 | Every surface can run against scratch state (a temp dir, an in-memory store, an env var) that touches no real data, credential or Discord server. | v1 |
| N-10 | Answering a question about the backend costs few tokens: output is short by default and detail is opt-in, so a Haiku agent can do the debugging. | v1 |
| N-11 | No behaviour lives only in the Discord layer; Discord translates between people and the backend and everything the backend does is reachable without it. | v1 |

### Secrets and reach

| Id | Requirement | When |
|---|---|---|
| N-12 | A credential is never posted in Discord, written to a log, returned by the debug interface or placed in a worktree. | v1 |
| N-13 | Credentials are stored apart from ordinary data, and a backup never contains a credential in the clear (choice). | v1 |
| N-14 | A session reaches only what its channel's reach allows: a read-only channel reads the project and writes nothing, a code channel changes only its own worktree, and the limit is enforced by the host, not only by instructions to the agent. | v1 |
| N-15 | The harness runs with the least the credential and the reach require, and the isolation proposal states, per host, what a session can still see of the credential the harness itself uses. | v1 |
| N-16 | The host Broodforge runs on holds nothing the owner cannot restore from a backup; anything that must not be stolen lives elsewhere. | v1 |
| N-17 | Every security-relevant action is written to an append-only audit log the owner can read: registration and credential changes, session start and end, permission decisions, commits, pushes, pull requests, configuration changes and refused actions. | v1 |
| N-18 | Permission checks use the roles a user holds at the moment of the action, so a role change in Discord takes effect at once (choice). | v1 |

### Backups

| Id | Requirement | When |
|---|---|---|
| N-19 | A nightly backup covers everything needed to restore a host: database, configuration, credential store, audit log and repositories, and the restore is a written, rehearsed procedure. | v1 |
| N-20 | Broodforge keeps no state outside the directories the backup covers, so that a restore is complete. | v1 |

### Token tracking

| Id | Requirement | When |
|---|---|---|
| N-21 | Token usage is recorded for every turn, including one that fails or is interrupted, as fully as the harness reports it, and the record is written before the turn's result is shown. | v1 |
| N-22 | Usage records persist across restarts and are part of the backup. | v1 |

### Open source

| Id | Requirement | When |
|---|---|---|
| N-23 | Broodforge is licensed GPL-3.0-or-later under `../hatchery/decisions/0001-licensing.md`, every source file carries an SPDX header, and a linkable component may be LGPL-3.0-or-later. | v1 |
| N-24 | Every dependency is GPL-compatible, and the proposal that picks a dependency names its license. | v1 |

### Behaviour under load and failure

| Id | Requirement | When |
|---|---|---|
| N-25 | Discord rate limits are respected without losing output; delayed output is coalesced, not dropped. | v1 |
| N-26 | Concurrent sessions do not block one another; a slow or stuck session delays only itself (choice). | v1 |
| N-27 | A restart or crash of the bot never corrupts a worktree, the database or the audit log. | v1 |

## Choices made where the vision was silent

R-02, R-03, R-07, R-09, R-11, R-15, R-16, R-22, R-25, R-27, R-33, R-34 (the default), R-35, R-45,
N-13, N-18, N-26. Each is the smallest reading that lets the next doc be written; the owner can
reverse any of them by saying so, and a proposal may argue against one with its reasons.

## What waits on the owner

The vision's open questions, with the requirements they touch:

1. Subscription terms exposure: R-38, R-39. Answered by ADR 0020.
2. Push or pull request as the default: R-34. Answered 2026-10-14 in `04-ux.md`.
3. Threads, private channels or the project channel for a session: R-12, R-20, R-23. Answered
   2026-10-14 in `04-ux.md`.
4. The author rule when several people drove: R-33. Answered 2026-10-14 in `04-ux.md`.
5. Mode, model and effort switching from Discord: R-28, R-29. Answered 2026-10-14 in `04-ux.md`.

Also the owner's, because they decide what is spent: the cap thresholds of R-46, when it comes.

## Tensions the proposals must resolve

- Claude Code reads the session's credential itself, from an environment variable or a config
  directory, and the agent's own tools run on the same host; N-12 and N-15 therefore depend on the
  isolation and credential-storage proposals, and a proposal that cannot fully hide the credential
  must say so rather than claim it.
- Attribution (R-32) needs the names of every model that worked; whether the harness reports
  subagent models is a fact the harness-adapter proposal must establish.
- Windows hosting (N-01) and enforced reach (N-14) pull against each other. Both hosts are
  first-version targets: the Windows instance serves the owner's own development, the Linux
  droplet the Utopia community. The isolation proposal may propose a weaker first-version reach
  on Windows, not on Linux, and must say what each host enforces.
