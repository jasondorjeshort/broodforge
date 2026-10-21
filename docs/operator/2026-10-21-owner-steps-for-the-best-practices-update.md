**One step left of the best-practices update: the Go code-intelligence plugin.** The identity in `privacy.identities` and the permissions allowlist in `.claude/settings.json` are in (commits of 2026-10-21). The reasoning is `docs/findings/2026-10-21-claude-code-best-practices-for-a-build-phase-go-project.md`.

Runs as the owner, in the repository root, in the shell `claude` starts from. About 10 minutes.

Install `gopls` on the `PATH` of the shell `claude` starts from (`go.md`'s "never `go install`" is about the broodforge binary):

```
go install golang.org/x/tools/gopls@latest
```

Inside Claude Code in this repository run `/plugin install gopls-lsp@claude-plugins-official` at project scope, or `claude plugin install gopls-lsp@claude-plugins-official --scope project`. It writes the `enabledPlugins` entry into `.claude/settings.json`; commit that entry. Check: the `Found N new diagnostic issues` line after the first Go edit, or `/plugin` then Errors for `Executable not found in $PATH: "gopls"`. It gives compiler and type diagnostics after every edit and a read-only `LSP` tool (definition, references, hover, call hierarchy); terminal sessions only. Unverified: whether subagents (the coder) get the `LSP` tool; run one coder pass after installing and record the answer in the handoff. Source: https://code.claude.com/docs/en/plugins/code-intelligence and https://code.claude.com/docs/en/plugins/install.

When done: delete this file. Update nothing else; the finding is the record.
