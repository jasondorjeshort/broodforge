# Domain

Date: 2026-10-14. Status: first draft, derived from `00-vision.md` and `01-requirements.md`.
This doc names the things Broodforge is about so that every later doc, proposal and brief uses one
word for one thing: for each concept, what it is, what identifies it, what it relates to and what
is open. It decides nothing the vision left open; it names what is pending and which doc settles it.

## Vocabulary rules

- The owner is the person who runs a Broodforge instance and configures its projects. Never use
  "owner" for the person who started a session; that person is the driver.
- A session has a driver and contributors; together they are its participants. "User" alone means
  a registered Discord user, whatever they are doing.
- Backend means a harness plus a credential, always both. "Engine" is the vision's older word for
  harness; use harness.
- Channel always means a Discord text channel bound to a project. Where a session's messages go
  (a thread, a private channel or the channel itself) is its venue: a private channel per session, a thread later (04).
- Turn means what Claude Code means by it: one input from a person and everything the agent does
  until it waits for input again.
- Reach is what a channel lets a session touch; permission mode (prompt or policy) is how a session
  answers the agent's requests inside that reach. They are different things.
- A trust tier is a way of talking about a Discord role; Broodforge has no tier of its own.

## Discord-side concepts

Broodforge stores Discord's identifiers and reads Discord's state; it owns none of it. A Discord
server (guild), a role, a text channel, a thread, a user and a message are each identified by the
id Discord assigns. An instance serves one server (R-11). Roles are assigned by the server's
admins in Discord; Broodforge reads which roles a user holds at the moment of an action (N-18) and
never writes one (R-04).

## Instance and host

An instance is one running Broodforge: a bot process, its database, its configuration, its
credential store and its audit log, serving one Discord server from one host. A host is the
machine it runs on, the owner's Windows PC or a Linux VM (N-01). Identity: the host name plus the
configuration directory; one host may in principle run two instances with separate directories.
Relations: an instance has projects; everything below belongs to one instance.

## Project

A git repository the owner configured, with the channels that work on it. Identity: a short name
the owner picks, unique in the instance; also the path of its main checkout on the host. Holds:
the repository location (local path, and the remote it pushes to), the default branch, the
end-of-session action (push the branch or open a pull request), the git credential used for that
(R-36), where its worktrees are created, and project-wide defaults a channel may override.
Relations: one or more channels (R-07); zero or more sessions through those channels; zero or more
worktrees. Open: the default end-of-session action for a new project (vision Q 2, in the UX doc);
whether a project may span several repositories (not required; the vision speaks of one).

## Channel

A Discord text channel bound to one project (R-07); the unit of configuration (R-08). Identity:
the Discord channel id. Holds: its project; its reach (read-only, code; later host, R-49); the
role ids allowed to start a session, to contribute, and to answer permission prompts (R-25);
the default permission mode (R-26); the default harness; the idle timeout (R-15; 0 is none, the default; the driver may change a session's); the prompt timeout (R-53; 0 is none, the default); later a token
limit (R-46). A "questions" channel is a channel whose reach is read-only and whose harness is a
small model (R-19); it is not a different kind of thing. Relations: one project; many sessions.
Open: the venue a session gets when it starts in the channel (vision Q 3, UX doc); whether a
channel without a project is ever needed (nothing in the vision needs one).

## Session

One run of a harness over one worktree, started from a channel by a driver, ending when the
driver ends it, the channel's idle timeout passes, the owner ends it, or the bot dies (R-15,
R-17). Identity: an id the instance assigns, short and unique, used in Discord and by the debug
interface alike; the harness's own session id is stored as a reference, not as identity. Holds:
channel and project; driver; venue (the thread or channel id where its messages go); worktree;
backend (harness and credential, fixed for the session, R-40); permission mode, which the driver
may switch (R-28); state; started and ended times; the outcome of the end-of-session action (a
pushed branch name or a pull-request URL). States: starting, waiting for input, running a turn,
waiting for a permission answer, ending, ended, interrupted. Relations: one worktree; one
backend; participants; turns in order; permission decisions; usage records; audit events; the
commits it made. Model and effort change mid-way from the next turn (answered: 04, Model and
effort, 2026-10-22). Open: changing harness mid-way; outliving its venue (deleted by hand).

## Participant, driver and contributor

A participant is a registered user's presence in one session, with the role driver or contributor
(R-13). Identity: session id plus Discord user id. The driver is the user who started the session;
there is one at a time. A contributor is any other user whose roles allow it and who wrote in the
venue or was invited into it. Each message a participant writes is recorded with their identity
and handed to the agent that way (R-14), so the agent can follow who said what. Relations: one
session; one registration; the turns they requested; the permission decisions they answered.
Open: whether the driver role can be handed to a contributor when the starter leaves, and who is
then the commit author (vision Q 4); whether messages written while a turn is running are queued
for the next turn, merged into one, or injected into the running turn (harness-adapter proposal
and UX doc together).

## Registration and identity

A registration ties one Discord account to one git identity: a name and an email (R-01). Identity:
the Discord user id; one registration per account, replaced by re-registering (R-02). Holds: git
name, git email, the Discord display name at registration time for the owner's convenience, when
it was made, and the user's default credential. The git identity is what commits are authored and
co-authored with (R-32). Relations: zero or more credentials; participations in sessions; usage
records as the requester. Open: whether the email is ever verified (the vision does not ask for
it); what a user may store besides name and email (a GitHub credential later, R-37).

## Credential, harness and backend

A credential lets a harness act as a person against an AI provider: a Claude subscription or an
API key first (R-39), other providers later (R-43). A subscription credential is a config
directory Broodforge owns but never reads (the user logs in themselves, ADR 0020); only an API
key is a secret in the credential store. Identity: a credential id the instance assigns. Holds:
its kind, its provider, the registration that owns it, a label the user chose, when it was added
and last used, its status (usable, removed), and a reference to the config directory or into the
credential store where the API key lives; the database never holds a secret (N-12, N-13). Relations: one registration; the sessions that used it; usage records as the
payer. Open: the store and the mechanism by which the harness receives the secret without the
session seeing it (credential-storage and isolation proposals); whether a credential may be shared
with other users, which the vision does not ask for.

A harness is a program that drives a model with tools: the real Claude Code program first (R-38),
Anthropic's API directly later (R-42), other providers later. Identity: a harness name, a short
string such as the program it wraps. Holds: which credential kinds it accepts and which
capabilities it offers, at least streaming, permission prompts, resume, and usage reporting; the
list of capabilities is the harness-adapter proposal's to fix.

A backend is one harness with one credential; it is what a session actually runs on. Identity:
the pair. It is not stored on its own: a user has a default (the credential, with the harness it
implies), a channel has a default harness, and a session records the pair it used. Model and
effort belong to the session, defaulted by the channel (answered: 04, Model and effort, 2026-10-22).

## Worktree

A git worktree of a project's repository, on its own branch, dedicated to one session (R-30).
Identity: its path on the host; the branch name is unique in the repository too. Holds: project,
session, branch, path, when it was created, and its state: active, pushed, kept with unpushed
commits, removed (R-35). A subagent worktree is a worktree whose parent is the session's worktree,
not a session of its own (R-31). Relations: one session; zero or more child worktrees; the commits
made in it. Open: naming of branches and directories, and what Windows requires of worktree
creation and removal (`../hatchery/reference/claude-code-on-windows.md`, Git worktrees; the
architecture doc).

## Turn

One input from a participant and everything the agent does until it next waits for input.
Identity: session id plus a sequence number within the session. Holds: the requesting participant,
the input text, start and end times, the harness's reference for it, and its result: completed,
stopped by a denied permission, error, or interrupted. Relations: one session; one requester; the
permission decisions raised during it; the usage records it produced; the messages Broodforge
posted or edited for it. Open: whether a permission answer is its own turn (no, in Claude Code's
usage; the harness-adapter proposal confirms for each harness).

## Permission decision

One answer to one permission request the harness raised during a turn (R-24). Identity: session
id plus the harness's request id. Holds: the turn; what was asked (the tool or action and a
summary of its input); who decided (a participant, the policy mode, R-26, the prompt timeout, R-53,
or the engine's own default when the hook was bypassed, `03-hook-contract.md`); the decision
(allow or deny; deferred while a person is asked; dropped, for a parallel call the engine did not
carry to the prompt; an "allow for the rest of this session" form is open); the reason when the policy
decided; the time. A policy-mode decision never allows more than the channel's reach (R-27). Every
permission decision is also an audit event. Relations: one turn; optionally one participant.
Open: which contributors may answer (R-25, per channel); whether "always allow" exists and how
far it reaches (UX doc with the owner, since people see it).

## Reach

What a channel lets its sessions touch: read-only (read the project, write nothing), code (change
only its own worktree), and later host (act on the machine, R-49). Identity: the name. It is a
property of a channel, inherited by its sessions, and enforced on the host (N-14). Open: how each
level is enforced on Windows and on Linux (isolation proposal); whether a level between code and
host is needed, for example running the project's tests (R-47 will ask).

## Agent and subagent

An agent is a model doing work in a session under the harness; a subagent is one the harness
started inside a turn. For Broodforge an agent matters in two ways: as a co-author on commits,
identified by the model name as it appears in the attribution line (R-32), and as a possible
owner of a subagent worktree. Agents have no record of their own; the names come from the
harness's reports and appear on commits and usage records. Open: whether the harness reports
subagent models (requirements, Tensions).

## Commit

A git commit made in a session's worktree. Identity: its hash. Holds: session, worktree, author
(the driver's git identity at the time, R-33), co-authors (other participants' git identities and
the agents that worked), the time, and whether it has been pushed or is in a pull request (R-34).
Broodforge records commits it knows about for attribution and audit; git stays the source of truth
for their content. Relations: one session; one worktree; one author registration; co-author
registrations.

## Usage record

The tokens one turn used, as the harness reports them (R-44, N-21). Identity: the turn, or the
harness's own id when a turn reports usage in several pieces. Holds: session, turn, the requesting
participant, the paying credential, harness and model, input tokens, output tokens, cache tokens
read and written, and the time. Totals are computed, not stored: per session, per user as
requester, per credential as payer, per channel and per project. The two per-user totals differ
when a person drives on another's credential; both are kept. Relations: one turn; one session;
one registration; one credential. Open: a cost estimate in money (needs a price table; later);
caps (R-46, the owner's).

## Audit event

An append-only record of a security-relevant action (N-17). Identity: a sequence number that only
grows. Holds: the time, the kind of event, the actor (a Discord user id, the policy mode, the
owner on the host, or the instance itself), the session where there is one, and details specific
to the kind. Kinds at least: registration changed, credential added or removed, session started,
ended or interrupted, permission decision, commit, push, pull request opened, configuration
changed, action refused. Relations: optionally one session; optionally one registration. Audit
events are never edited or deleted and are part of the backup (N-19).

## Trust tier as a Discord role

The vision's tiers (fully trusted, well trusted, member, open sign-up) are Discord roles that the
server's admins assign. In Broodforge a tier appears only as a role id written into a channel's
configuration under one of its permissions: who may start, who may contribute, who may answer
prompts. There is no tier table, no ranking of roles, and no role that Broodforge creates. Open:
whether configuration names roles directly or through named groups the owner defines once and
reuses across channels; a convenience the configuration format proposal may add, never a new
authority.

## Data model

Each row is one kind of record; "points to" are the relations above. Storage is a major choice
and comes later; this table is what any storage must hold.

| Record | Identity | Holds | Points to |
|---|---|---|---|
| Instance | host plus config dir | Discord server id, directories, policy | projects |
| Project | name | repo path, remote, default branch, end action, git credential, worktree root | channels, worktrees |
| Channel | Discord channel id | reach, allowed role ids per permission, default mode, default harness, idle timeout, prompt timeout | project |
| Registration | Discord user id | git name, git email, display name, registered at, default credential | credentials |
| Credential | credential id | kind, provider, label, status, added at, last used, store reference | registration |
| Harness | name | accepted credential kinds, capabilities | none (configured, not stored) |
| Session | session id | state, venue id, harness, credential id, permission mode, started, ended, end outcome | channel, driver, worktree |
| Participant | session id plus user id | role (driver or contributor), joined at | session, registration |
| Worktree | path | branch, state, created at, parent worktree | project, session |
| Turn | session id plus sequence | requester, input, started, ended, result, harness ref | session, participant |
| Permission decision | session id plus request id | request summary, decider, decision, reason, time | turn, participant |
| Commit | hash | author, co-authors, time, pushed or PR | session, worktree |
| Usage record | turn or harness id | model, input, output, cache read, cache write, time | turn, participant, credential |
| Audit event | sequence number | time, kind, actor, details | session, registration |

Not records: reach and permission mode are named values on a channel and a session; a trust tier
is a role id; a backend is the harness name and credential id a session carries; an agent is a
model name on a commit or a usage record.

## What this doc leaves open, and where it is settled

- Session venue (thread, private channel, the channel itself): `04-ux.md` with the owner.
- End-of-session default, commit author rule, mode and model switching: the same UX doc, from the
  vision's questions 2, 4 and 5.
- Credential store, how the harness gets the secret, what a session can see: the
  credential-storage and isolation proposals in `06-platforms.md`.
- Harness capabilities, subagent reporting, message handling during a running turn: the
  harness-adapter proposal.
- Branch and directory naming for worktrees, configuration format: `03-architecture.md`
  [no-check: not yet written].
- Storage of every record above: the storage proposal.
