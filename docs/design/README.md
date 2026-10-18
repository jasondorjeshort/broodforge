# Design

Design-first: nothing gets built until it is written down here and agreed.

Planned documents (added as we go):

| File | Purpose |
|------|---------|
| `00-vision.md` | What the app is, who it is for, what "done" looks like |
| `01-requirements.md` | Functional and non-functional requirements |
| `02-domain.md` | Core concepts, vocabulary, data model |
| `03-architecture.md` | Components, boundaries, tech choices and why, the promised surfaces with their compatibility promise, and the fitness checks |
| `04-ux.md` | Flows, screens, interaction model |
| `05-debug-interface.md` | How agents drive, observe, and reproduce every feature from a terminal |
| `06-platforms.md` | Target platforms and the language, toolkit, and API chosen for each part, with the sharing strategy between them |
| `proposals/` | Draft proposals for major choices; each gets a critic pass before becoming an ADR |
| `decisions/` | Architecture decision records (ADRs), one per decision |

Major choices (language, toolkit, API, storage, protocol, build system, core dependencies) follow
the proposal, critic, ADR process in `CLAUDE.md`. Technical choices are settled by agents; what
humans see and touch is settled with the user.

Decisions so far (this tree's own, in `decisions/`; Hatchery's records, the practice this tree inherits, are indexed in [`../hatchery/decisions/README.md`](../hatchery/decisions/README.md)):

The rest of `docs/` (handoff, ideas, reference, findings) is mapped in [`../README.md`](../README.md).
