<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
# spike/

Throwaway harness for claims S1, S2 and C25 to C39 (`docs/design/07-spike-plan.md`, droplet half
`docs/design/07-spike-droplet.md`). Deleted in the commit that lands `engine run --real`; the
recorded fixtures move to `fixtures/engine/<name>/` after a person reads them.

| File | What it is |
|---|---|
| `common.ts` | Shared names: `$S`, droplet paths, the spike git identity, default engine path, env strip, flag parser. |
| `run.ts` | One engine turn: `$FLAGS`, stdin user line, raw `turn-<n>.jsonl`, `EngineEvent` parser, `modelUsage` delta (session found by the `--resume` id), kills, deadline; `--replay` and `--version-only`. `--cred key` is droplet-only. |
| `setup.ts` | Fixture repositories and commits through `child_process`, settings from the template. |
| `hook.ts` | The PreToolUse hook; mode from `spike-hook.json` beside `BROODFORGE_HOOK_LOG`; `node hook.js mode ...` sets it on the PC. |
| `login.ts` | C25: piped `claude auth login` with `CLAUDE_CONFIG_DIR` set; `--console` retry. |
| `keys.ts` | C36: key paths of a JSON file, values over 40 characters as `<long>`. |
| `settings.template.json` | One `PreToolUse` hook, empty matcher, timeout 10, `node <bf>/hook.js`. |
| `bwrap.sh` | Droplet: the companion's bind block as `sudo -n -u broodforge-engine bwrap`; `--maintenance`, `--check`, `--print`. |
| `engine-entry` | Droplet: the entry wrapper (checks, `exec`) and the maintenance commands, in `sh`. |
| `hookmode.sh` | Droplet: writes `<sess>/home/spike-hook.json` through the sandbox. |
| `strace-cfg.sh` | Droplet C38: paths under `<cfg>` in an strace log, written and read, unexpected ones marked `!`. |
| `share/empty/` | An empty `CLAUDE.md` (must stay empty: no SPDX line), `mcp.json` (`{"mcpServers":{}}`), and `dir/`, which is not in git because it must hold nothing: `setup.js` and `bwrap.sh` create it. |
| `package.json`, `package-lock.json`, `tsconfig.json`, `.gitignore` | `typescript` and `@types/node` only; `tsc` writes `*.js` beside the sources (ignored). |

## Build (Git Bash on the PC)

The owner's `~/.npmrc` has `location=global`: never a bare `npm install`, `npm ci`, `-g` or
`--no-global` (`docs/reference/claude-code-on-windows.md`, "npm on this PC"). Create an empty
file first (`: > <empty.npmrc>`), then:

```
npm_config_userconfig=<empty.npmrc> npm --prefix ~/Documents/src/broodforge/spike install
~/Documents/src/broodforge/spike/node_modules/.bin/tsc -p ~/Documents/src/broodforge/spike/tsconfig.json
```

## PC run order

`B=~/Documents/src/broodforge/spike`, `S=$TEMP/bf-spike` (`BF_SPIKE` overrides),
`RUN="node $B/run.js"`. Every real turn spends the owner's subscription; read both meters from
`~/.claude/usage-state.json` before and after each group.

1. `node $B/setup.js pc` (tree, canon, `$S/cfg`, settings, hook mode `allow`); `$RUN --version-only`.
2. C25, the owner: `docs/operator/2026-10-14-spike-login-on-the-pc.md`. C36: `node $B/keys.js $S/cfg/.claude.json`.
3. S1: `$RUN --out $S/fx/basic --prompt "Read hello.txt and reply with its one word."`, then
   `$RUN --out $S/fx/basic --resume last --prompt "Say the word again."`.
4. C27: `node $B/hook.js mode defer`; `$RUN --mode prompt --out $S/fx/defer --prompt "Create a file named hi.txt containing hi."`;
   `node $B/hook.js mode defer <id>:allow`; `$RUN --mode prompt --out $S/fx/defer --resume last`
   (stdin closed; else `--empty-line`, or `--stdin-json '<line>'`); then a fresh deferred turn answered `deny`.
5. C39: `node $B/hook.js mode exit2|exit1|sleep`, one turn each into `$S/fx/hook-<mode>`; add
   `--permission-prompts none` when a run reports `waits for a prompt`, and keep it after.
6. S2: `$RUN --out $S/fx/interrupted --kill-after 5000 --prompt "Run \`sleep 30\` in Bash, then say done."`,
   then `$RUN --out $S/fx/interrupted --resume last --prompt "Continue."` (second try:
   `--env CLAUDE_CODE_RESUME_INTERRUPTED_TURN=1`).
7. C35 PC half: the plan's commands, the commit by `node $B/setup.js commit $S/clone`.
8. C34 PC half, C28 (`--mode prompt`, hook `defer`, `--model sonnet` for the last attempt),
   C26 (`node $B/setup.js settings --no-hook`; `node $B/setup.js settings --out $S/alt-settings.json`;
   `--settings $S/alt-settings.json`, then add `--setting-sources ""`; restore with `node $B/setup.js settings`).
9. Replay every fixture: `$RUN --replay $S/fx/<name>`; no `unknown` for a kind 03 lists.

The droplet order is the companion's; there `$RUN --spawn bwrap` and `hookmode.sh` replace the PC forms.
