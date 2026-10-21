---
date: 2026-10-21
status: resolved
area: process / claude-code
---

# Claude Code best practices differ for a build-phase Go project in nine rows, and four of them are edits

## Question

The best-practices page (https://code.claude.com/docs/en/best-practices) was reviewed for the base
while it had no code: `../hatchery/docs/findings/2026-10-18-claude-code-best-practices-review.md`
`[no-check: sibling checkout]`. Broodforge has Go code, tests, a build and a binary. Which of that
review's verdicts change, and what edits follow?

## What was tried

The sibling finding was taken row by row; only rows whose answer differs for a project with code
are kept. Facts were checked in the trees (the post-edit hook, `ownership.json`, the agents and
skills, `.claude/rules/go.md`, `.claude/hooks/policy.json`, `docs/`). Timings were measured on the
PC with a warm cache and no `-race`: build plus vet is seconds, `go test ./...` is seconds.
`gofmt` is at `C:\Program Files\Go\bin`; `staticcheck` and `exhaustive` are `go tool` entries in
`go.mod`, not on `PATH`; gcc for `-race` is `C:\msys64\ucrt64\bin\gcc.exe`, on the owner's user
`PATH` only for processes started after 2026-10-16. `ownership.json` lists `.claude/settings.json`
as `mixed` and `.claude/hooks/policy.json` as `baseExcluded`, so the allowlist and the formatter
are the project's to edit; agents, skills and hook scripts are base-owned (a change there is a
carry-back or a departure).

## Result

Rows that differ from the sibling finding's verdict:

| Practice | Broodforge has | Gap | Verdict |
|---|---|---|---|
| Deterministic gate | `go.md` check line, coder brief step 6, pusher check 8, `tests/repro/*.sh`, `bin/` as the debug interface | No edit-time check: a badly formatted file or a syntax error is found at the pusher | take: `gofmt` on edit |
| Stop hook | The sibling parked its own | The main session is a delegation-only coordinator, so a settings-level Stop hook would run Go checks after turns that edit no Go; code is written in coder subagents, where only `SubagentStop` or the agent's own frontmatter fires | skip at settings level; idea filed for the coder |
| Test commands in `CLAUDE.md` | `go.md` carries them, but loads only when a `.go` file is touched | A coordinator, pusher or runner that never opens Go does not see the commands | owner or coordinator edit of `CLAUDE.md`, not in this commit |
| Permissions allowlist | Read-only git and probes | Go build, vet, test, tools, `gofmt`, hook scripts and read-only `gh` run many times per wave; each is a prompt wherever auto mode is not on | owner step |
| Hooks: must happen every time | The `edit.formatters` seam in `post-edit.js`, empty | It exists for this and nothing used it | take |
| Hooks: delete or convert a rule Claude follows | `go.md` states rules a check could enforce | Prose the coder must remember and the reviewer must grep for; `go vet` and `staticcheck` enforce none of them | idea filed (lint test) |
| Code-intelligence plugin | none | A typed language exists now | operator entry |
| Skills: `disable-model-invocation` | none set | Checked for `commit`, `handoff`, `decant`: none should carry it (`handoff` must be model-loadable, `usage-gating.md` orders it at HOLD and STOP) | skip, same as the sibling |
| Context: compaction line | none | No report of state lost to compaction in ideas, findings, handoff or agents; coder passes start from a fresh brief | skip until a pass loses its file list after a compact; the home would be `coder.md`, a carry-back |
| Fan out across files | waves of briefed agents | A wave of briefed coders already is the fan-out | skip; revisit when a mechanical change touches every package |
| Trust, then verify | reviewer, pusher, `-race`, repro scripts | A process started before gcc reached `PATH` runs `-race` with `CGO_ENABLED=0` | covered by the `PATH` prefix already in `go.md` |

Rows with the same answer as the sibling's: plan mode, specific prompts, MCP, CLI tools, subagent
investigation, checkpoints, `claude -p`, adversarial review, auto mode. Agent `tools:` lists were
read against their jobs and none is wrong for Go.

Ranked edits and where each went:

1. Formatter: `".go": ["gofmt", "-w"]` in `policy.json` `edit.formatters`. Taken in this commit.
   The hook runs `spawnSync(argv[0], [...rest, absPath])` with no shell, so it suits a file-level
   tool and cannot run package-level `go vet` or `go build`. A parse error exits non-zero and comes
   back to the agent as `additionalContext`, a free syntax check. Cost: the agent's next Edit may
   fail once on a rewritten file and be re-read.
2. Hooks path: `git.hooksPath` in `policy.json`. Taken in this commit.
3. `go.md`: the build line gains `CGO_ENABLED=0` to match the plan's standing rule, and one line
   points the coder who edits `internal/control`, `internal/spawn` or `internal/hook` at the two
   reference files (named pipes, junctions, gcc `PATH`, antivirus). Taken in this commit.
4. Commit identity in `privacy.identities`: owner step. The harness's auto-mode classifier refuses
   an agent that writes an identity (personal data); the operator entry carries it.
5. Permissions allowlist: owner step. The classifier refuses an agent that edits its own
   permissions (provenance); the operator entry carries the list and what is left out on purpose.
6. Lint test for the mechanical `go.md` rules: idea filed
   (`docs/ideas/2026-10-21-go-rules-become-a-lint-test.md`).
7. Coder `Stop` hook running build and vet: idea filed, a question for the owner, with the trigger
   (`docs/ideas/2026-10-21-coder-stop-hook-for-build-and-vet.md`).
8. Code-intelligence plugin: operator entry
   (`docs/operator/2026-10-21-owner-steps-for-the-best-practices-update.md`, step c).
9. Skipped with reason: the compaction line (no evidence of loss), `disable-model-invocation`
   (the skills must be model-loadable), a settings-level Stop hook (fires after coordinator turns
   that edit no Go), fan-out (the wave already is one).

Code-intelligence facts (https://code.claude.com/docs/en/plugins/code-intelligence and
https://code.claude.com/docs/en/plugins/install): the plugin is `gopls-lsp` from the
`claude-plugins-official` marketplace and needs the `gopls` binary on the `PATH` of the shell
`claude` starts from. Installed at project scope it writes an `enabledPlugins` entry into
`.claude/settings.json`, which is committed. It gives compiler and type diagnostics after every
edit (the `Found N new diagnostic issues` line) and a read-only `LSP` tool (definition,
references, hover, call hierarchy); terminal sessions only. A missing binary shows under `/plugin`
Errors as `Executable not found in $PATH: "gopls"`. Unverified: whether subagents (the coder) get
the `LSP` tool; one coder pass after installing answers it. `go.md`'s "never `go install`" is
about the broodforge binary, not about `gopls`.
