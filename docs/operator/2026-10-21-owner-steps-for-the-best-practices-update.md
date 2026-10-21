**Three steps to finish the best-practices update: a commit identity in the push scan, a permissions allowlist, and the Go code-intelligence plugin.** They wait on the owner because the harness's auto-mode classifier refuses an agent that writes an identity (personal data) or edits its own permissions (provenance). The reasoning is `docs/findings/2026-10-21-claude-code-best-practices-for-a-build-phase-go-project.md`.

Runs as the owner, in the repository root, in the shell `claude` starts from. About 15 minutes.

**a. Identity.** Add the owner's commit identity to `privacy.identities` in `.claude/hooks/policy.json`, as `{ "name": "<owner name>", "email": "<owner email>" }`, the same entry Hatchery's `policy.json` carries (ADR 0018: a listed identity passes the push scan; with the list empty the scan refuses every commit the owner authored). Check it took:

```
node .claude/hooks/policy.js
```

The printed policy lists the identity.

**b. Permissions allowlist.** Add to `permissions.allow` in `.claude/settings.json` (a `mixed` file under the ownership map, so this is the project's to edit):

```json
"Bash(go build *)",
"Bash(CGO_ENABLED=0 go build *)",
"Bash(go vet *)",
"Bash(go test *)",
"Bash(CGO_ENABLED=1 go test *)",
"Bash(go tool staticcheck *)",
"Bash(go tool exhaustive *)",
"Bash(go env *)",
"Bash(go version)",
"Bash(gofmt -l *)",
"Bash(gofmt -d *)",
"Bash(node .claude/hooks/test-hooks.js*)",
"Bash(node .claude/hooks/policy.js*)",
"Bash(node .claude/hooks/queue-list.js*)",
"Bash(gh pr view *)",
"Bash(gh pr list *)",
"Bash(gh pr checks *)",
"Bash(gh run view *)",
"Bash(gh run list *)",
"Bash(gh release list *)"
```

Deliberately not allowed: `bin/broodforge` (can touch a live instance), `go run`, `go install`, `go get`, `go mod`, `gofmt -w` (the post-edit hook does it), `gh api`, `gh pr create` and `gh pr merge`. Check: `node .claude/hooks/test-hooks.js` still passes, and a coder pass runs `go vet ./...` without a prompt.

**c. Go code-intelligence plugin.** Install `gopls` on the `PATH` of the shell `claude` starts from (`go.md`'s "never `go install`" is about the broodforge binary):

```
go install golang.org/x/tools/gopls@latest
```

Inside Claude Code in this repository run `/plugin install gopls-lsp@claude-plugins-official` at project scope, or `claude plugin install gopls-lsp@claude-plugins-official --scope project`. It writes the `enabledPlugins` entry into `.claude/settings.json`; commit that entry. Check: the `Found N new diagnostic issues` line after the first Go edit, or `/plugin` then Errors for `Executable not found in $PATH: "gopls"`. It gives compiler and type diagnostics after every edit and a read-only `LSP` tool (definition, references, hover, call hierarchy); terminal sessions only. Unverified: whether subagents (the coder) get the `LSP` tool; run one coder pass after installing and record the answer in the handoff. Source: https://code.claude.com/docs/en/plugins/code-intelligence and https://code.claude.com/docs/en/plugins/install.

When done: delete this file. Update nothing else; the finding is the record.
