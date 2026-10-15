---
date: 2026-10-15
status: open
area: spike / engine contract (ADR 0020)
---

# C25 holds on the PC: the piped login prints the URL and takes the pasted code on stdin

## Question

Do the engine behaviours ADR 0020 assumes hold against the real engine?
`docs/design/07-spike-plan.md` turns the ADR's claims table into runs. This finding is the ADR 0020
half of the spike's output, one row per claim; it stays `open` while any row is open, and when the
last row closes it is `promoted_to` ADR 0020 with the rows copied into the ADR's table. ADR 0022's
claims are a separate finding.

| Claim | Host | Verdict | When |
|---|---|---|---|
| C25 headless login with `CLAUDE_CONFIG_DIR` | PC | holds, piped form | 2026-10-15 |
| C26 `--settings` with `--setting-sources user` | PC | open | |
| C27 `defer` round trip | PC, then droplet | open | |
| C28 parallel calls in prompt mode | PC | open | |
| C29 per-turn overhead | droplet | open | |
| C39 hook exit codes | PC, exit 2 again on droplet | open | |
| S1 stream shape and `modelUsage` delta | PC | open | |
| S2 Windows tree kill and resume | PC | open | |

C25 on the droplet is a separate run of the same claim and is still open.

## What was tried

- Host: the PC, Windows 11 Pro 10.0.26300, Git Bash (UCRT64), Node v22.23.2.
- Engine: `@anthropic-ai/claude-code` 2.1.296, the global npm install, launched as
  `AppData/Roaming/npm/node_modules/@anthropic-ai/claude-code/bin/claude.exe`.
- Fixtures: `node spike/setup.js pc`, with `BF_SPIKE` set explicitly to the spike root so the
  agent's `TEMP` could not move it. `$S` is `AppData/Local/Temp/bf-spike`. It created `tree` on
  `main` and `canon` on `base` (both at `7295ef5 spike fixture: hello.txt`), wrote
  `cfg/settings.json` (281 bytes) and `spike-hook.json` (17 bytes), and left `cfg` untouched.
  Note the order: the owner ran the login before the fixtures, which is harmless because the `pc`
  step only ensures `cfg` exists and never rewrites it.
- C25, run by the owner because the code is theirs:
  `CLAUDE_CONFIG_DIR="$TEMP/bf-spike/cfg" node spike/login.js`, stdin, stdout and stderr all piped,
  which is the relay's shape.

## Result

C25 holds in its first form. No TTY was needed, and neither retry was used.

- The engine printed the authorize URL on its piped stdout, and `login.js` relayed it to the owner.
- `login: paste the code from the browser here and press Enter:` appeared; the pasted code reached
  the engine's stdin and the engine answered `Login successful.`
- `login: engine exited 0`.
- `login: auth status exit 0`, reporting `Login method: Claude Max account`.
- The directory listing carried `.claude.json` (56693 bytes), `.credentials.json` (524 bytes) and a
  `backups` directory.

So the relay sentence in ADR 0020's Credentials section stands on Windows. The `--console` retry and
the direct terminal login, the ADR's other path, were not needed and remain untested on this host.

The account is not the owner's, which matters for everything after this row. The login signed in as
the PC owner's Claude Max account; the project owner is a different person. C25's verdict does
not depend on whose code was pasted, but every later PC claim spends the subscription of whichever
account `$CFG` holds, and the owner may move the spike to his own account. Switching means deleting
`$CFG` and running C25 again, which re-tests the claim rather than wasting the work.

## Next

- The open rows above, in the plan's order: S1, C27, C39, S2, C28, C26. C29 and the droplet repeats
  wait on the droplet operator entries.
- The owner decides, before any claim that spends: whose subscription the PC claims run on. Weekly
  usage was 68% against a 75% HOLD line when this was written, and the plan's PC group needs
  roughly a dozen engine turns.
- `promoted_to` ADR 0020 when the last row closes.
