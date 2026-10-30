# 0027. Setup in few steps: `broodforge setup`, bot-made channels and role, the PC account created by `setup host`

Date: 2026-10-26
Status: accepted

Record: proposal `../proposals/0027-setup-in-few-steps.md` (options A to D, the step tables in its
Appendix, Owner's answers and Review fixes of 2026-10-26) and `../proposals/0027-critic-review.md`
(its verification table is authoritative over the proposal body). The owner answered the calls on
2026-10-26: option C; the Manage Roles prompt defaults to yes; every name, prompt and refusal text
accepted as quoted. Companion: `0027-setup-commands.md` (the flags, the steps in order, the rerun
rule per key, the refusal table, the PC shell rule, the host commands, `config export`, `--from`).

## Context

The owner set the bot up once from the eight operator files and must redo it on the main PC. Counted
one row per hand action (a page of portal clicks, one command with its elevation prompt, one paste
or typed answer, one copy into a password manager, one link opened, one browser sign-in), the status
quo is 49 steps on the PC and 53 on the droplet. The floor no code removes: the Developer Portal
(create the application, reset the token), a GitHub token, the password manager copies, the browser
sign-in of the first subscription credential. What applied: both hosts (`../06-platforms.md`), no
new dependency; ADR 0021 (owner-edited config with immutable project ids, never written by the
bot); ADR 0022 (the store's one writer, the key printed once, the owner's `~/.claude` never
touched, the PC instance under the local standard account `broodforge`); `../05-cli-conventions.md`
(existing names and the wire format unchanged, every error one stderr line, `(file)` and listener
commands named). The two prior-art findings of 2026-10-25 (setup wizards and owner choices;
isolating agent sessions) gave the shape: a prompt with a shown default for a consequential choice,
a flag for every prompt, non-interactive runs refuse an irreversible choice and name the flag,
reversible values default in config; and no runner or agent sandbox in the field runs jobs as the
Windows owner, which is why a same-account PC default (option D) was rejected by the owner on
2026-10-25 ("we don't want to have claude credentials in the session history"). Verified by the
critic: the application and guild endpoints (C1, C3 to C8, the new role's position undocumented),
the task-scheduler claims (W1, W2), the credentials path (W5), the vault invariant (S1, a race
closed below). Refuted: a password-logon task does not create the account's profile (W3); replaced
by W3a. Still unverified, each with the pass of `../08-implementation-plan-setup.md` that settles
it: W3a (`Start-Process -Credential -LoadUserProfile` from an elevated interactive shell creates
the profile), W4 (the native engine runs from `%ProgramData%`) and W6 (the account cmdlets and
`Register-ScheduledTask` run from an elevated shell with the script and password on stdin, and a
password change does not end the account's processes, which `setup login` and the relaunch below
rely on), the P20 spike; C6's position (where a new role lands), the P22 owner step in the test
guild; L1 (`visudo -cf` on the generated sudoers file), P26 on the VM. C2 (the offline token check)
is not spiked: C1 is the route the code takes.

## Decision

Option C. One command in the binary, `broodforge setup`, does every scriptable step and asks for the
rest; `setup host` does the elevated host part on each host, the PC account included; the bot
creates what it needs in the guild. Simple mode (`spawn.backend: same-account`) stays the
development and scratch backend only.

1. **`broodforge setup [<step>] [flags]`**, its flags, output, `--json` fields, exit codes and its
   steps in order (`root`, `config`, `key`, `engine`, `token`, `application`, `guild`, `channels`,
   `role`, `git-credential`, `project`, `defaults`, `check`, `next`) as the companion specifies:
   one `ok`, `skip` or `fix` line per step, a `fix` that needs a hand polled to `--wait` (default
   900 s); exit 0, 1 (the `fix` line on stdout, one stderr `error: <code>: ...` line), 2 usage;
   `--check` reports and changes, binds and polls nothing; `--yes` takes every shown default and
   refuses `needs-input-<step>` only where there is none; `setup <step>` runs one step and may
   replace its keys after `old`, `new` (a secret as its id or path) and a yes, the only overwrite
   of a present key; one secret per command (the bot token on `setup`'s stdin, the git credential
   hidden or on `setup git-credential`'s stdin alone), the key printed once. The repository is
   asked in `config`, whose project id names the project file, the channels and the clone; the git
   credential precedes the clone, so a private canonical clones.
2. **Choices.** Prompts with a flag: root and instance name (`--root`, `--instance`; defaults
   `%ProgramData%\Broodforge\<instance>` or `/srv/broodforge/<instance>`, `main`); the guild when
   several (`--guild <id>`); create channels in a guild that has some (`--channels yes|no`, default
   no; under `--yes` the default is taken and `channels` waits for the owner's channels by name);
   repository URL and the project id when the derived one is invalid (`--repo`, `--project`);
   create the role (`--create-role yes|no`, default yes). Names `setup` looks up or creates, never
   stored (03's config holds ids): role `Trusted` (`--role`), category `Sessions`, lobby `#<project
   id>`, notices `#<project id>-notices`. Config values with a default and no prompt: timeouts 0,
   mode `prompt`.
3. **Guild objects.** `channels` checks the configured ids first, then finds by name, then creates
   what is missing (Manage Channels is in the invite's set), writes the ids, never deletes or
   renames what it finds; in a guild that has channels it creates nothing without a yes. `role`
   checks `roleGroups.trusted`'s id, then finds by name; absent and the answer yes, the invite
   carried Manage Roles, setup creates the role, checks it sits below the bot's highest role (`fix`
   otherwise), gives it to the guild's `owner_id`; absent and the answer no, or the bot already in
   the guild without Manage Roles, the `fix` that waits for the owner to make and assign it. On
   every run, `--check` included, the bot's permissions are read once the role is settled: Manage
   Roles held prints the removal `fix` and polls until it is gone, so an interrupted wait is caught
   by the next run. A configured id whose object is gone is `mismatch-<step>`, lifted by `setup <step>`.
4. **Single writer.** `setup` (not `--check`) binds the control address (`data/control.sock`; the
   pipe `\\.\pipe\broodforge-<instanceName>`) for its lifetime, as a listener beside `serve`, so
   `serve` fails `start-control` as for any listener and the vault's one-writer invariant holds;
   its `status` reply is `config: setup`. A rerun never overwrites a present config key, a valid
   key file, a stored credential (a new value is a new id, the old one `removed`) or a channel or
   role it found; the rule per key with its two exceptions (a stored token Discord refuses is
   replaced in the run; `setup key` replaces a key only while the store holds no entry, else
   `key-in-use`) is the companion's, as is the refusal table (`running`, `setup-running`,
   `needs-input-<step>`, `mismatch-<step>`, `refused-<step>`, `timeout-<step>`, `needs-elevation`,
   `key-in-use`, `check-failed`, `login-failed`, `config-exists`), each with its message, who
   lifts it and from which shell.
5. **The host commands** (companion). `setup host [--root] [--instance] [--engine-version <v>]
   [--from <export>] [--print]`, elevated or root, idempotent with the bot stopped (`running`
   otherwise), runs the operator files as code and ends by running `setup` as the bot's account
   (`--from`'s file carried into the root first) and starting the service; `--print` writes what
   it would run and change, the password withheld. On the PC it creates the account `broodforge`
   with a 32-character password it generates and stores nowhere (Task Scheduler holds it; a rerun
   rotates it); `%ProgramData%\Broodforge\` with inheritance off and inheritable entries for
   SYSTEM, Administrators and `broodforge` only, which the root and the key inherit (`credential
   init` creates the key there; `setup host` creates no key file); `bin\` and `engine\` read-only
   to `broodforge`; the task `Broodforge` at startup with `PT0S` read back; and runs the inner
   `setup` as `broodforge` through `Start-Process -Credential -LoadUserProfile -Wait` (W3a). On the
   PC the owner's door is an elevated PowerShell: the pipe keeps its default security descriptor
   (Administrators admitted, Everyone read only; no DACL change, since the pipe authenticates
   nothing) and the root's ACL admits no other account, so reads and `stop` run in that shell and
   `setup` or `setup <step>` relaunch themselves as `broodforge` by `setup host`'s route. `setup
   login <credential id>` runs the engine's login under the bot's account and records `--done`;
   `setup service` re-registers the unit or the tasks; `config export [--out <file>]` writes
   `config/` as one JSON document with the host-bound keys emptied and no secret; `setup --from
   <file>` asks only the host-bound and secret questions. One application per instance: a second
   instance from an export retires the first.
6. **The owner's texts** (2026-10-26), quoted verbatim in the companion's "The owner's texts" and
   its refusal table, which the coder copies from. Two changed by the review fixes of the same day,
   for the owner to re-approve: the key line "key  <hex>  <path>" is now the step line "ok  key
   <hex>  <path>", and the `running` text gained "in this shell" after "broodforge stop".
7. **Amendments.** ADR 0022 is amended, not superseded (its Amendments from ADR 0027): the store is
   written by the bot, or by `setup` while it holds the control address; the Windows root and key
   default move from the account's profile to `%ProgramData%\Broodforge\` with the ACL above; the
   PC account is created by `setup host` with a generated password. ADR 0021's Amendments gain the
   entry: "owner-edited files" are created by `setup`, a command the owner runs, writing only
   absent keys, never the bot. `../03-architecture.md`'s config example and its Debug CLI row
   follow. The PC account operator file is retired when `setup host` lands (P23).
8. **Out of scope and deferred.** A session reading its own `harness/<id>/.credentials.json`
   through Bash on the PC (ADR 0022, Windows; its trigger stands). The login relay: until it
   exists, `setup login` and the browser step stay. A release pipeline removes the build and copy
   rows. W3a failing adds one PC step (sign out and in once as `broodforge`); W4 failing keeps the
   installer fallback; C5 or C6 failing returns channels or the role to the client with setup
   prompting for ids. Implementation order: `../08-implementation-plan-setup.md`, P20 to P26.

## Consequences

Counts from the Appendix, same rule for every option: PC 18, droplet 20 (the role by hand 19 and
21; the git credential deferred 15 and 17); a second PC from an export, 9. Cost: the `setup` group
and its tests against the fakes, elevated code on both hosts that no test runs (`--print` is its
proof), the W3a, W4 and W6 spikes, config written once by a command (0021's second writer at the
file level, absent keys only), and on the PC a password rotation and a relaunch as `broodforge`
for every write to the live root (the stdin forms serve tests, scratch roots and Linux). Manage
Roles is held for the minutes between the invite and the removal; a compromised session in that
window could create roles. Exit: delete the `channels` and `role` steps (B's four id prompts
return) or the `setup` package (A's checklist returns). Superseded by: a Discord endpoint that
creates an application or resets a token; the relay landing; the engine's sandbox or MXC on
Windows (ADR 0022's trigger).

- Prediction, check when P21 lands: a rerun of `setup` on a configured host with the bot stopped
  prints only `ok` and `skip` lines and changes no file, no credential, no channel and no role.
  Outcome, 2026-10-30: holds, on the evidence in the tree at `4496e2b`, the scripted Discord
  standing in for the guild. `tests/repro/setup.sh` step 2 (the proof of P21 and P22, under `scratch
  run`) reruns the full `--yes` line on a configured root and asserts every result line is `ok` or
  `skip`, the five Discord steps `skip`, `config/` byte-equal to its copy from before the rerun, and
  no `Create`, `Update` or `AddMember` written to the scripted Discord; `TestPlaceholders` in
  `internal/setup/setup_test.go` asserts only `ok` and `skip` for a live rerun of a root turned live
  from dry-run; the P27 coder's run on the real dev root printed only `ok` and `skip` on the second
  `setup --discord dry-run` line. The credential and key halves rest on `skip  token` and `skip
  key  exists` (a skip writes nothing, the companion's rerun rule); no run diffed
  `credentials/store.json` itself. The owner's guild is P22's owner step, not checked here.
- Prediction, check by 2027-04-26: no later pass adds an owner-only step a command could have
  done; the PC count stays 18 or falls.
- Prediction, check when P23 lands: W3a holds (the profile exists after the inner `setup`, no
  sign-out) and the task runs the bot at boot with `PT0S` read back.

## Amendments

- 2026-10-30: the PC development instance (the owner, 2026-10-30: getting Broodforge runnable on
  the PC to debug through the debug interface is an early milestone and need not depend on the
  Discord interface; `../05-debug-interface-dev-instance.md`; P27 of `../08-implementation-plan.md`).
  Decision 2's flag list ("Prompts with a flag: ...") gains `--discord live|dry-run`, default `live`,
  a flag with no prompt: under `dry-run` the five Discord steps call nothing and write the
  companion's dry-run form (placeholder channels `100`, the fake, and `200`, the real engine;
  placeholder role `10`; `engine.fakeFixtures`), `discord.mode` stays `dry-run`, and a later `live`
  run on such a root replaces the placeholders without asking. That replacement is the one
  exception to Decision 4's "A rerun never overwrites a present config key": the value is setup's
  own placeholder, never the owner's. The instance this serves is the Decision's "development and
  scratch backend only" case of `same-account`: a root under the owner's account, no `setup host`,
  no service, the owner's own engine, kept dry-run for good (the live form under the owner's
  account is a second root and instance); the Windows gap of ADR 0022 applies to it, and the owner
  accepted it for development on that instance on 2026-10-30, a real turn still running on the
  owner's word only. Counts in Consequences are unchanged (the dev instance is not an owner instance).
