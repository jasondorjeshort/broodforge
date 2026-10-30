# Debug interface: the development instance on the PC

This is the development-instance part of `05-debug-interface.md` and shares its Conventions. It
fixes how an agent gets a kept Broodforge instance on the owner's PC that it starts, drives,
observes and stops from a terminal, so a pass after P22 is debugged through the debug interface
and not through Discord. The `setup` flag it rests on, `--discord dry-run`, is ADR 0027's
companion's (`decisions/0027-setup-commands.md`, The dry-run form) and `05-debug-interface-setup.md`'s;
the agent's loop is the project skill `.claude/skills/run/SKILL.md`; the pass is P27 of
`08-implementation-plan.md`. The owner's decisions this answers (2026-10-30): a runnable instance on
the PC is an early milestone and need not depend on the Discord interface; the root, the instance
name and the key handling below are approved; ADR 0022's Windows gap is accepted for development
on this instance (Isolation).

## The instance

Surface. One kept root under the owner's own account, `%LOCALAPPDATA%\Broodforge\dev` (Git Bash:
`$LOCALAPPDATA/Broodforge/dev`), instance `dev`, so its pipe is `\\.\pipe\broodforge-dev` and its
key the default beside the root, `%LOCALAPPDATA%\Broodforge\dev.key`; the backend `same-account`;
the engine the owner's own native `claude.exe` on `PATH` (the dry-run form's `engine` step looks
there first, so P23's pinned copy under `%ProgramData%` never captures the dev root);
`discord.mode` `dry-run`, for good: this root is never promoted to live (below); one project, this
repository, id `broodforge` by flag, cloned from the main checkout (`--repo` in the `C:/...` form
`pwd -W` prints), handoff `push`, so a session's branch `broodforge/dev/s<n>` lands in that checkout
as a local branch and nothing leaves the PC. The root belongs to the main checkout, not to a
worktree: `setup` is run from the main checkout only, since `--repo` and `engine.fakeFixtures` are
written as that checkout's absolute paths; an agent in a worktree uses the root as it is (its own
`bin/broodforge.exe` is fine) and never runs `setup` from the worktree. It is distinct from the
scratch root (throwaway, fake engine, regenerated per run) and from P23's service instance
(`%ProgramData%\Broodforge\main`, the `broodforge` account, the task, elevation): different root,
different `instanceName`, different pipe, no account, no task, no elevation, and nothing under
`~/.claude`, which the engine never reads once `CLAUDE_CONFIG_DIR` names a harness directory under
the root. Made and remade by one command, by an agent, never by hand: `bin/broodforge.exe setup
--discord dry-run --yes --root "$LOCALAPPDATA/Broodforge/dev" --instance dev --project broodforge
--repo "$(pwd -W)" --wait 1`; the dry-run form it writes (the placeholder channels `100`, the fake,
and `200`, the real engine, both open to the placeholder role `10` as `trusted`, and
`engine.fakeFixtures` naming the checkout's `fixtures/engine/`) is the companion's. A rerun prints
only `ok` and `skip`; after the owner's engine auto-updated it is `mismatch-engine`, lifted by
`setup engine --yes` on the root, which the skill runs on that refusal and then reruns the line. A
plain `setup` on this root (no `--discord dry-run`) would promote it and replace the placeholders,
after which the skill's line is refused `live-instance`: that is a mistake, not a path; the fix is
to delete the root and remake it.

The live form on the PC under the owner's account is a second root, `%LOCALAPPDATA%\Broodforge\dev-live`,
instance `dev-live` (pipe `\\.\pipe\broodforge-dev-live`), made by the same line with `--discord
live` and its own `--instance`, `--root` and application: P22's owner step runs there, and the
companion's placeholder rule (a `--discord live` run replaces a dry-run root's placeholders) serves a
dry-run root the owner chooses to promote, never `dev`.

Drive. Build (`CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge`), `export
BROODFORGE_ROOT="$LOCALAPPDATA/Broodforge/dev"` (and `unset BROODFORGE_KEY_FILE`: the override is
the service instance's, never the dev root's), the `setup` line above, `serve --discord dry-run`
in the background with its output in a per-run `<root>/serve-<stamp>.log`, `status --wait 10
--expect ok`, then any Drive command of 05 and its companions: `register 1001 --name "Ada Example"
--email ada@example.com`, an `api-key` credential whose value is `scratch` (the fake's), `discord
inject click --custom-id bf:100:start --user 1001 --roles 10` and `discord inject message
--channel <session id> --user 1001 --roles 10 "Read hello.txt and reply with its one word."` for the
round trip, `session end <id>`, `stop`. Channel `200` is the real engine: a session there, or
`engine run --real --credential <dev credential> --cwd <tmp tree> --prompt "..."`, spends the
owner's plan and runs on the owner's word only, as every pass's "real turn" does: the brief quotes
the owner's go-ahead for that real turn in so many words, or it does not run.

Observe. `status`; `session tail <id> --all`; `session outbox <id>`; `audit --limit 20`; the
run's `serve-<stamp>.log` for the start-up lines and warnings; `credential list` for the dev
credential's `usable`; `register list` for the owner's registration beside `1001`.

Reproduce. The whole loop on a temporary root, no owner, no plan spent: `bash
tests/repro/dev-instance.sh` makes a dry-run root under the OS temp
directory with a generated instance name, runs the round trip on channel `100`, reruns `setup`
(only `ok` and `skip`), turns the root live against the scripted Discord
(`BROODFORGE_SETUP_FAKE_DISCORD`) and shows the placeholders replaced, then removes the root. On
the dev root itself a bug is one Drive command; a start refusal is its `error: start-<step>` line
in the run's log.

Isolation. The dev root is the owner's account's: a session there runs as the owner, its reach
the hook's alone, and its Bash can read the dev login under `<root>/harness/<id>/`. That is ADR
0022's stated Windows gap; it applies to the dev instance, and the owner accepted it for
development on this instance on 2026-10-30 (a real turn still runs on the owner's word only, as
above). The dev credential is a login the owner can revoke. `--discord dry-run` never calls
Discord; nothing is pushed off the PC; the instance's `privacy.allow` is the template's. One agent
at a time: `serve` refuses a second bot (`start-control`), but nothing tells two agents apart on one
bot (`status` carries no PID and no owner token), so the brief's assignment of the instance to one
agent for its run is the sole guard; the skill adds what an agent can check itself (its own
background `serve` still alive, its own per-run log) and stops only the bot its run started; an
agent whose `status` answers before it started anything leaves that bot alone and reports.

## The owner's one step: the dev credential

Once, with the dev bot running (the agent leaves it running by the skill's detached step and says
so, or the owner starts it with the Drive line in a second Git Bash window), in Git Bash from the
main checkout, `BROODFORGE_ROOT` exported as above:

1. `bin/broodforge.exe register <your Discord user id> --name "<git name>" --email <git email>`
   (the id Discord's "Copy User ID" gives, so the registration carries into the live form later).
2. `bin/broodforge.exe credential add --kind subscription --user <id> --label dev` prints `cred-<x>`.
3. `bin/broodforge.exe credential login cred-<x>` prints one line, `CLAUDE_CONFIG_DIR=<root>/harness/cred-<x>
   <engine> auth login`; paste it, sign in in the browser with the subscription (the one extra login:
   the owner's own `~/.claude` login is not reused), paste the code back if the callback fails.
4. `bin/broodforge.exe credential login cred-<x> --done`, then `bin/broodforge.exe register
   set-default <id> cred-<x>`, then `bin/broodforge.exe stop`, so the next agent's first `status`
   finds no foreign bot.

That is the content of the operator entry P27 files (`docs/operator/README.md`'s shape: the
command block, the shell, what must be true first, how to check: `credential list` shows `usable`).
The dev store key printed by `setup`'s `key` step guards no secret the harness directory does not
already hold, so it goes in no password manager: a lost dev root is remade.
