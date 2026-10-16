# Vision

Date: 2026-10-14. Status: the initial vision, accepted by the owner on 2026-10-14; expected to change as proposals land. Source: the owner's four statements of
2026-10-14 and the coordinator's fact check of the same day. Where the owner stated an intent
rather than a pick, this doc states the intent and lists the pick under "Choices still to make"
or "Open questions for the owner"; nothing below is decided that the owner did not say.

## What Broodforge is

Broodforge is a self-hosted Discord bot that puts Claude Code behind a Discord channel. A person
with the right Discord role types in a project's channel, a coding session starts in its own git
worktree on the host, Claude Code works the repository, and its output streams back into Discord
as it happens. Commits are attributed to the Discord people who drove the session and to the AI
agents that did the work. The point is that "vibe programmers", people who want to build and
have no terminal and no AI experience, can code a project through the chat tool they already use,
while the owner keeps the code, the host, the credentials and the backups under their own control.

Broodforge drives the real Claude Code program rather than re-implementing it on Anthropic's raw
API, because that is the only way a user's Claude subscription (Pro, Max, Team) can be the thing
that pays. An API-key-only backend on the raw API may exist later as a second engine; it is not
the primary one.

## Who it is for

- The owner, Jason Dorje Short, first. The first repository Broodforge hosts is Broodforge itself:
  the owner develops the bot through the bot, from a Discord server, the way this project is
  being designed today from a terminal.
- Utopia next: a 30-year-old text-based game, heavily a work in progress being brought back to
  usability. Its community is the model for every trust tier Broodforge must handle:
  - about 5 fully trusted players;
  - 10 to 50 well trusted players;
  - about 1000 players who are mostly responsible, with a few trolls;
  - open sign-up beyond that.
  Most of these people have no AI experience. Trust tiers are Discord roles, assigned by humans;
  Broodforge reads them and never changes them. An agent may one day recommend a role upgrade
  for a player who seems trustworthy; recommendations only, never automatic.
- The Discord bot Ticket Tool (tickettool.xyz) is the interaction this community already knows
  and likes: a private channel per ticket, opened on request, closed when done. It is the UX
  reference. Integrating with it and replacing it are both on the table; it is external and paid,
  so it could go away.

## Principles the owner has stated

- Self-hosted, always. A dev server holds the checkouts and runs the bot and any web site.
  Production is a cloud VM; it is not a service where anyone imports their own repository.
- Windows hosting is required. Coding and debugging happen on the owner's Windows PC, a permanent
  host, so the bot must run there as well as on the Linux VM.
- Configured projects only. The owner calls this an artificial limit, since checkouts and PRs are
  easy, and keeps it anyway: a project exists because the owner set it up.
- Discord roles are the permission source. Who may start a session, in which channel, with what
  reach, follows from roles the server admins manage in Discord, not from Broodforge's own user
  database. Anyone with permission is either in the channel or invited into the session.
- One worktree per session, for sure. Subagents may have their own worktrees with cherry-picking,
  the normal git workflow for AI. Git is the primary backend for coding.
- Secrets stay secret. Anyone with a high enough role could trick the AI into doing damage, so
  the defence is layered: a session reaches only what its channel is for (a questions channel
  reads code only; a code channel touches only its code), important things live on other servers
  so there is nothing to steal, and the bot never shows a credential to a session or a channel.
- Backups are the backstop. Nightly backups of the host are the recovery path when the layers
  above fail. Repositories can contain scripts that later run on the server, so the cloud box is
  the biggest hole by a wide margin and the backups exist for it.
- Everything is agent-debuggable. The bot must be driven, observed and reproduced from a terminal
  by our own Haiku agents, without clicking Discord, and cheaply: the debugging interface answers
  questions about the backend in few tokens. Discord itself is a thin layer on top.
- Token use is tracked for every session and user from the start; capping is a possibility.
- Open source under GPL-3.0-or-later (`LICENSE`, ADR 0001). The owner knows an open-source project
  cannot stop others misusing it and accepts that.
- Long-term design over time to first release (2026-10-16). Architecture choices are made for
  scalability and maintainability; reaching a first release sooner is secondary. Growth that
  counts: sessions and users per instance (hundreds of concurrent sessions, the thousand-player
  tier, on one host); codebase longevity (years of agent-written changes staying readable and
  safe: type strength, a small dependency tree, a stable toolchain, cheap refactors); and engines
  and providers (a second and third engine without touching the core). Many instances or servers
  per deployment is not a growth axis. Agents know every language, so language and library
  choices are judged on mechanical properties, not on a programmer's familiarity.

## The first usable version

Done, for the first version, means: the owner opens a Discord server, registers, starts a session
in the Broodforge channel, asks for a change to Broodforge, watches it happen, and ends with a
commit on Broodforge's own repository that names the owner as author and the agents that worked
as co-authors. From then on the owner develops Broodforge through Broodforge, as this session
works from the terminal today. Concretely the first version includes:

- registration by an in-Discord command, roughly `.register Jason Dorje Short <jdorje@gmail.com>`,
  which ties a Discord account to a git identity;
- one Discord channel per project (several per project allowed), with the session started from
  the channel and its permissions taken from Discord roles;
- streamed output: Claude Code's progress appears as it happens, edited in place where that
  reads well and posted as a new message where a notification matters, about one message per
  second at most;
- permission prompts as Discord buttons, with a policy mode that answers them for the session most
  or all of the time;
- a worktree per session, commits attributed to the Discord person and to every AI agent that
  contributed, and a push or pull request at the end;
- a user's own Claude subscription as the credential: a per-user config directory the user logs
  into themselves, which Broodforge never reads or stores a token from (ADR 0020), selected per
  user, with API keys as the fallback;
- token tracking per session and per user;
- hosting on both targets from the first version, from one code base, ideally within days of
  each other: an instance on the owner's Windows PC serves the Broodforge channel, and an
  instance on a Linux droplet is where Utopia will be developed;
- the agent-facing debug interface, designed before the bot is built, so the whole thing can be
  exercised without Discord.

Several people may be in one session; the bot keeps track of who said what so the agent can
follow who is who.

## Explicitly later

- The "simple questions" thread: anyone asks, a read-only Haiku answers, with its own tighter
  token limit. The easiest feature, and not in the first wave.
- A side web site for configuration, especially per-channel settings, which is where a web form
  helps most. Per-channel configuration itself is in from the start; the web view is not.
- Other AI providers and switching between logins across them. Multiple backends even for Claude
  are fine from the design; only the Claude Code engine ships first.
- Utopia itself: the droplet host is first-version, but configuring the Utopia project on it and
  running its tests on commit, or regularly during and after commits, come later.
- Ticket Tool integration or replacement.
- A channel that controls the cloud machine itself, beyond any one repository.
- Token caps and role-upgrade recommendations.
- GitHub credentials stored per user, so that pushes and pull requests act as the person.

## Choices still to make

Each of these is a major choice under the working agreement: a proposal, a critic pass and an ADR
before any code (`CLAUDE.md`, the `proposal` and `adr` skills; ADRs for this project start at
0020). The owner has familiar tools but asked for the design to be made without preconceptions.

- Language and runtime for the bot and its backend. Answered by ADR 0023 (Go), which reopened
  ADR 0020's language choice on 2026-10-16 under the long-term-design principle above.
- Discord library, including how it does streaming edits, buttons and private channels.
- Harness adapter shape: how Broodforge drives Claude Code (the Agent SDK, the CLI as a
  subprocess, or both behind one interface), and how a second engine would plug in.
- Storage: the backend database for users, projects, sessions and token accounts.
- Isolation on Windows and on Linux: what a session can reach beyond its worktree, and how that
  is enforced on each host.
- Credential storage: how per-user subscription tokens, config directories and any API keys are
  kept, selected and kept out of sessions.

## Open questions for the owner

1. Answered by ADR 0020 (per-user config directory, no token read or stored, API keys as
   fallback). Terms exposure of subscription tokens, as asked. For: the owner wants users' own subscriptions to pay,
   it is self-hosted and open source, and it drives the real Claude Code program, which is what
   subscriptions are for. Against: Anthropic's SDK terms say third-party developers may not offer
   claude.ai login or rate limits in their products without approval, and it is not clear which
   side of that line a self-hosted bring-your-own-subscription bot falls on; terms also change,
   and a plan to meter headless use separately has been reported. Options: proceed and accept
   the risk, ask Anthropic first, or ship API keys first and subscriptions once answered. The
   owner decides; the design supports either credential kind. The 2026-10-16 amendment to ADR
   0020 records the primary-source answer on both the terms and the metering of headless use. Open for the owner: whether the droplet's DM login relay counts as sign-in "through Anthropic's own flow".
2. Pull request or direct push as the default at the end of a session. The owner said either,
   probably configurable; the default for a new project still needs a pick. Answered 2026-10-14
   in `04-ux.md`.
3. Channels or threads for sessions. Threads have the right lifecycle but hide badly; a private
   channel per session, Ticket Tool style, may be better. This shapes the first flow the owner
   sees, so it is settled with the owner in the UX doc. Answered 2026-10-14 in `04-ux.md`.
4. The primary-author rule. Commits carry one primary author and further attributions; who is
   primary when several people drove the session, or when the owner started it and someone else
   finished. The owner noted nobody cares that much; a simple default needs choosing anyway.
   Answered 2026-10-14 in `04-ux.md`.
5. Switching mode, model or effort from Discord. The owner called this the harder question: it
   does not map well to chat and invites people to switch repeatedly and lose context. Whether
   the first version exposes any of it, and how, goes to the UX doc with the owner. Answered
   2026-10-14 in `04-ux.md`.

## What comes next

The requirements doc (`01-requirements.md`) is written from this one once the owner confirms it,
then the domain doc, the platforms doc listing the choices above, and one proposal per choice.
