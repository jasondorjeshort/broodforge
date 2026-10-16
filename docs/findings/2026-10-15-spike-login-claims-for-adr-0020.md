---
date: 2026-10-15
status: open
area: spike / engine contract, login (ADR 0020)
supersedes: 2026-10-15-spike-claims-for-adr-0020.md
---

# C25 holds on the PC: the piped login works with `CLAUDE_CONFIG_DIR`, twice; the droplet repeat is open

## Question

Does the headless login ADR 0020 assumes hold against the real engine?
`docs/design/07-spike-plan.md` turns the ADR's claims table into runs. This finding is the login
slice of the ADR 0020 half of the spike's output, split by topic from one finding: the stream and
process rows are in `2026-10-15-spike-stream-and-process-claims-for-adr-0020.md`, the hook-flow
rows in `2026-10-15-spike-hook-flow-claims-for-adr-0020.md`, which also holds the shared spike
setup (host, engine, fixtures, `$S`, `$CFG`). It stays `open` while the droplet row is open, and
when it closes it is `promoted_to` ADR 0020 with the row copied into the ADR's table. ADR 0022's
claims are a separate finding.

| Claim | Host | Verdict | When |
|---|---|---|---|
| C25 headless login with `CLAUDE_CONFIG_DIR` | PC | holds, piped form | 2026-10-15, again 2026-10-16 |

C25 on the droplet is a separate run of the same claim and is still open.

## What was tried

- Host, engine and fixtures: as in the hook-flow finding (the PC, engine 2.1.296, `$S` the spike
  root, `$CFG` its `cfg` directory).
- C25, run by the owner because the code is theirs:
  `CLAUDE_CONFIG_DIR="$TEMP/bf-spike/cfg" node spike/login.js`, stdin, stdout and stderr all piped,
  which is the relay's shape. Run twice: 2026-10-15, and 2026-10-16 after the owner deleted `$S/cfg`.
  Note the order: the owner ran the login before the fixtures, which is harmless because the `pc` step (hook-flow finding, Shared setup)
  only ensures `cfg` exists and never rewrites it.

## Result

C25 holds in its first form. No TTY was needed, and neither retry was used.

- The engine printed the authorize URL on its piped stdout, and `login.js` relayed it to the owner.
- `login: paste the code from the browser here and press Enter:` appeared; the pasted code reached
  the engine's stdin and the engine answered `Login successful.`
- `login: engine exited 0`.
- `login: auth status exit 0`, reporting `Login method: Claude Max account`.
- The directory listing carried `.claude.json` (56693 bytes), `.credentials.json` (524 bytes) and a
  `backups` directory.
- 2026-10-16, the owner's second run as himself, same piped form, no `--console`: `Login
  successful.`, `engine exited 0`, `auth status exit 0` reporting his Claude Max account,
  `.claude.json` 58397 bytes, `.credentials.json` 524 bytes, `backups`; then `setup.js settings`
  rewrote `cfg/settings.json`. C25 held a second time.

So the relay sentence in ADR 0020's Credentials section stands on Windows. The `--console` retry and
the direct terminal login, the ADR's other path, were not needed and remain untested on this host.

Whose account `$CFG` holds. The 2026-10-15 login signed in as the PC owner's Claude Max account,
not the project owner's. On 2026-10-16 the owner deleted `$CFG` and ran C25 again as himself, so
from then on the directory holds the project owner's login and every PC engine turn spends his plan.
Two observations about that account, identifiers compared and not printed: its `oauthAccount`
account, organization and email entries equal those in this PC's `~/.claude.json`, the login the
coordinator session runs under, so the `usage-state.json` meters in the other two findings are the
spike's account after all; and the engine's own stream reports the plan meters of whatever account
it runs as, see S1 in the stream and process finding.
The finding writes the PC owner's username as `<pc user>`; the fixtures do not. They carry it in every
path-bearing field of `turn-*.jsonl` (`file_path`, `cwd`, `command`, `permission_denials`, Bash stdout and
`tool_result` text) and in every `meta-*.json`, with no email and no credential. Two independent agents
read the fixtures and they were committed 2026-10-16 (`50e0c40`), the import rewriting the path (rule in
`9accf2d`).

## Next

- The PC row is closed. The open row: the droplet repeat of C25 waits on the droplet operator
  entries.
- `$CFG` holds the project owner's login as of 2026-10-16; every PC turn spends his plan, and the
  stream's `rate_limit_event` shows the meters after each turn.
- `promoted_to` ADR 0020 when the last row closes.
