**Log the spike's own Claude Code config directory into your subscription on the PC, through the spike's piped login (claim C25).** It waits on you because the login code is yours: an agent must not complete a login on your account. The claim, its evidence and what a refutation changes are `docs/design/07-spike-plan.md` (C25); every later PC spike run uses the directory this creates, and the owner's own `~/.claude` is never read. This login adds one device to your subscription.

Status: done 2026-10-15, in the piped form, on the Claude Max account belonging to the PC owner. `$TEMP/bf-spike/cfg` now holds `.credentials.json` and `.claude.json`, and the result is recorded as C25 in `docs/findings/2026-10-15-spike-claims-for-adr-0020.md`. Not deleted yet: if the spike moves to a different Claude account, delete `$CFG` and run this entry again.

Runs in Git Bash as you, on the PC. True first: the spike is built (`spike/README.md`, Build). If `$TEMP/bf-spike/cfg` does not exist yet, create the fixtures first (no engine, spends nothing):

```
node ~/Documents/src/broodforge/spike/setup.js pc
```

The login, exactly as C25 runs it (stdin, stdout and stderr piped, the relay's shape):

```
CLAUDE_CONFIG_DIR="$TEMP/bf-spike/cfg" node ~/Documents/src/broodforge/spike/login.js
```

What you see: a `login:` line naming the command, then the engine's own output with a URL. Open the URL, sign in with your Claude subscription account, authorize, copy the code the page shows. When `login: paste the code from the browser here and press Enter:` appears, paste it there and press Enter. Expect `login: engine exited 0`, then `login: auth status: ...` saying you are logged in, then a listing of the directory (names and sizes only) that includes `.credentials.json`.

If the engine complains about a terminal (raw mode, not a TTY), prints no URL within about 15 seconds, or does not accept the code: press Ctrl+C and retry with the console form, where the engine writes straight to this window and only the code goes through the pipe. `--console` here is the spike's flag and is never passed to the engine (the engine's own `auth login --console` would mean Anthropic Console API billing, which is not wanted):

```
CLAUDE_CONFIG_DIR="$TEMP/bf-spike/cfg" node ~/Documents/src/broodforge/spike/login.js --console
```

If that fails too, C25 is refuted on the PC; so the PC runs can go on, log in directly in the terminal with the directory set (the ADR's other path), and say so when reporting:

```
CLAUDE_CONFIG_DIR="$TEMP/bf-spike/cfg" ~/AppData/Roaming/npm/node_modules/@anthropic-ai/claude-code/bin/claude.exe auth login
```

Check it took:

```
CLAUDE_CONFIG_DIR="$TEMP/bf-spike/cfg" ~/AppData/Roaming/npm/node_modules/@anthropic-ai/claude-code/bin/claude.exe auth status --text
ls -la "$TEMP/bf-spike/cfg"
```

Expect logged in with your subscription, and `.credentials.json` and `.claude.json` in the listing. Allow 5 minutes.

When done: tell the coordinator which form worked (piped, `--console`, or the direct terminal login) and paste the `login:` lines and the engine's messages, leaving out the code and the URL's query string; the coordinator records C25 in the spike finding. Delete this file.
