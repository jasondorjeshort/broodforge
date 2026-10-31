---
date: 2026-10-31
kind: feature
---

# One session reaches several repositories, and one bot serves several servers

The owner wants, after the first milestone: a session that works in several repositories at
once; several such groups under one bot; and one bot serving several Discord servers, each
server possibly with its own account settings (which subscription pays, which credentials). The
rule today is one of each: a project is one repository (`docs/design/02-domain.md`, Project; the
open line there asks exactly whether a project may span several), a session gets one worktree of
one project (R-30) and reaches no further (N-14), a channel binds to one project (R-07), and one
instance serves one server (R-11, a v1 choice). The "host" reach (R-49, later) is the only escape,
and it is all-or-one.

Anchors: `internal/config/types.go` (`Project` holds one `Repo`; `Instance` holds one guild),
`internal/config/load.go` (every `config/projects/*.json` already loads, so many projects per
instance works today), `internal/git/create.go` (one worktree per session), `internal/setup`
(`setup --repo` writes one project file per run; `resolveProject` asks which project when more
than one file exists). The cheap half, many single-repository projects, is configured by running
`setup --repo <url>` once per repository.

Shape to argue over, not decided: a project lists repositories, each with its own remote,
default branch and credential; a session gets a worktree of each under one session directory, the
reach being the set; the handoff pushes or opens a pull request per repository that changed. A
server is then a second level above projects (an instance holds servers, a server holds projects
and the account settings R-40's credential choice reads from), which reverses R-11 and touches the
usage meters (ADR 0011 pools are per account). This is an incompatible change to promised
surfaces (ADR 0012): the project file schema, `session show`, the audit's session rows.

## Open

- Whether "several repositories" is one worktree each or a parent directory the session sees as
  one tree; the hook's path checks (N-14) assume one root.
- Which per-server settings exist beyond the credential: harness, engine version, policy.
- Whether a server is an instance with a shared binary and separate roots (no schema change,
  the service runs one process per server) or one process serving several guilds.

Waits: the owner's deferral, 2026-10-31: not before the service-instance milestone on the PC is
reached (`docs/design/08-implementation-plan.md`, Milestones, third row). Then the proposal loop
(`proposal` skill): the architect writes it, the critic reads it, an ADR supersedes the R-11
choice and amends the Project and Session sections of `02-domain.md`.
