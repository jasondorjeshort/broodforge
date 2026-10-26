# Critic review of 0027: setup in few steps

Date: 2026-10-25
Reviewed: `0027-setup-in-few-steps.md` (status under review), against ADRs 0020 to 0022 and 0025,
`06-platforms.md`, `05-cli-conventions.md`, `03-hook-contract.md`, `05-debug-interface-discord-live.md`,
`08-implementation-plan.md`, every `docs/operator/` entry but the history rewrite, `internal/discord/`
and `internal/credentials/`, and `../../findings/2026-10-25-prior-art-setup-wizards-and-owner-choices.md`.
Counts: 2 blocking, 8 serious, 9 minor. The verification table is authoritative over the proposal
body where they differ.

## Verification

| Claim | Verdict | Evidence |
|---|---|---|
| C1 | holds | Discord OAuth2 topic: `GET /oauth2/applications/@me` "returns the bot's application object"; the application object has `id`, `flags`, `bot` (partial user), `owner`. disgo v0.19.6 `rest/oauth2.go` `GetBotApplicationInfo(opts...)`, no bearer token, used by `internal/discord/check.go:68`. |
| C2 | not checked | Offline folklore; C1 is the authority as the proposal says. Harmless. |
| C3 | holds, with a client wrinkle | Discord user resource: `GET /users/@me/guilds`, default and maximum 200, `before`, `after`, `limit`; the page requires the `guilds` scope for OAuth2 and does not state bot support in words. disgo defines the endpoint bot-auth (`rest_endpoints.go`) but exposes it only as `OAuth2.GetCurrentUserGuilds(bearerToken, ...)`, whose doc says an empty token uses the bot token (`withBearerToken`). The coder passes `""`. |
| C4 | holds | Discord application resource, Edit Current Application: `PATCH /applications/@me`, JSON param `flags`, "Only limited intent flags (`GATEWAY_PRESENCE_LIMITED`, `GATEWAY_GUILD_MEMBERS_LIMITED`, and `GATEWAY_MESSAGE_CONTENT_LIMITED`) can be updated via the API." Flags table: `1<<18` "required for bots in 100 or more servers", `1<<19` "required for bots in under 100 servers ... found on the Bot page". disgo v0.19.6: `UpdateCurrentApplication(discord.ApplicationUpdate)` on `PATCH /applications/@me`, `ApplicationUpdate.Flags *ApplicationFlags`. `gateway.go:278` and `check.go:105` accept either flag. Note: send the existing flags OR `1<<19`, never `1<<19` alone, and skip when `1<<18` is already set. The portal step goes on both hosts. |
| C5 | holds | Guild resource, Create Guild Channel: `POST /guilds/{guild.id}/channels`, "Requires the `MANAGE_CHANNELS` permission", `type` and `parent_id` params, "If setting permission overwrites, only permissions your bot has in the guild can be allowed/denied." `gateway.go:527` already calls it for venues. |
| C6 | holds except one part | `GET /guilds/{id}/roles` lists; Create Guild Role and Add Guild Member Role "Requires the `MANAGE_ROLES` permission"; permissions topic: "A bot can grant roles to other users that are of a lower position than its own highest role" and "it can only grant permissions it has"; `owner_id` is in the guild object. Not in the docs: where a new role lands (Create Guild Role has no `position` param; the default is undocumented). Unverified. |
| C7 | holds | Bits from the permissions topic: View 1<<10, Send 1<<11, Embed 1<<14, Attach 1<<15, Add Reactions 1<<6, Read History 1<<16, Manage Messages 1<<13, Pin Messages 1<<51, Manage Channels 1<<4; sum 2251799813811280, the operator file's integer. Plus Manage Roles 1<<28: 2252068249246736. |
| C8 | holds | Neither the application resource nor the OAuth2 topic has an endpoint that creates an application or resets a token; the portal floor is two steps (three with the token copy, objection 10). |
| W1 | holds for registration; the time limit and two side effects open | Microsoft, Security Contexts for Tasks: a non-administrator registering a task "to run under the security context of your account" with "the S4U or interactive logon type" needs no password; an administrator needs one only for `TASK_LOGON_PASSWORD` or another user's S4U; a low-privilege process registers `TASK_RUNLEVEL_LUA`. `-ExecutionTimeLimit ([TimeSpan]::Zero)` producing `PT0S` (no limit) has mixed evidence (works on 2016 and later per forum reports; a DSC issue saw `PT72H` kept): `setup service` must read the registered XML back and `fix` otherwise. Unstated in the proposal: an interactive-logon task runs only after the owner signs in (the status quo task runs at boot), and a console binary started by one shows a console window at every logon. |
| W2 | holds | Same page: `TASK_LOGON_PASSWORD` tasks "will only launch if the specified user has the Logon as Batch privilege"; the Log on as a batch job policy page: Task Scheduler "automatically grants this right" when a task is scheduled with a user name and password. `New-LocalUser` needs elevation. |
| W3 | refuted | Microsoft, Event ID 1511 when you start a task created in Task Scheduler: a "run whether user is logged on or not" task for "a user who does not have its user profile directory created yet" logs the user on "by using the temporary user profile"; KB2968540: `%USERPROFILE%` may resolve to the Default profile folder when the profile is not loaded. The task does not create the profile. A replacement exists: `setup host` runs `Start-Process -Credential` once from the elevated interactive shell (`CreateProcessWithLogonW` with profile; ADR 0022 verified it fails only from a non-interactive service), which creates the profile without a sign-out. New claim, spike it. |
| W4 | unverified; the citation is wrong | `docs/hatchery/reference/claude-code-on-windows.md` says nothing about the native installer or the executable. Claude Code setup docs: the native Windows install is a signed `claude.exe` ("signed by Anthropic, PBC"), the npm package "installs the same native binary", Git for Windows optional (the Bash tool needs it), auto-update in the background unless `DISABLE_AUTOUPDATER` (the spawn sets it, 0022). A copy to `%ProgramData%` is plausible; nothing documents running from a foreign path. P13 spike. |
| W5 | holds | Claude Code authentication docs: "On Windows, credentials are stored in `%USERPROFILE%\.claude\.credentials.json` and inherit the access controls of your user profile directory." The scrub is environment-only (0022 C34); 03 Paths denies `.credentials.json` by name to the Read class but Bash is `ask`, policy mode `allow`, and "the host is the boundary". In simple mode the host is the owner. The file is also writable: `~/.claude/settings.json` and `~/.claude/CLAUDE.md` of the owner's own interactive engine. |
| L1 | not checked | Out of this review's reach (the droplet is held); `visudo -cf` and `apparmor_parser -r` are the operator files' own commands. |
| S1 | holds, with a race | `internal/credentials/vault.go`: `OpenVault(path, keyFile)` is a plain constructor; `write` is temp file, sync, rename; the comment "the bot is the only writer, so at open any [temp file] is stale" is the invariant setup must keep (objection 9). |

## Objections

1. Blocking. D breaks a constraint the proposal says it keeps. `06-platforms.md`, 0022 Already
   constrained: "the owner's own `~/.claude` on the PC must not be touched by Broodforge's sessions";
   0022 Context: "the owner's `~/.claude` is never touched"; the proposal's own Constraints paragraph
   repeats it. In D a session's Bash reads and writes `%USERPROFILE%\.claude\.credentials.json`,
   `settings.json` and `CLAUDE.md` (W5). The proposal names one 0022 Decision sentence as superseded
   and nothing else. Resolves: list every line D supersedes (06 Already constrained, 0022 Context, the
   Constraints paragraph, 03 Paths "on the PC a session's Bash reads the same files", which gains "and
   the owner's"), and put the owner's call 1 as a requirement change, not a default; or drop D.
2. Blocking. The load-bearing claim is half right, and the wrong half decides milestone 2. "PC sessions
   are already one account with the bot, so a second coder session gains nothing from the service
   account" is true of session-to-session isolation (0022 Windows). It is false of what the service
   account is for: isolating every session from the owner. More coders means more unattended policy-mode
   sessions (03: Bash is `allow` there), more drivers from "a few trusted people" (06), more cloned
   repositories carrying instructions; each is a chance, and in D the prize is the owner's identity, with
   persistence: a hook written into the owner's `~/.claude/settings.json` runs in their next interactive
   session (0022 verified the engine reloads hooks from watched files). "The engine already runs with
   exactly this access in the owner's own sessions" is not exact: those have a person at every Bash
   prompt. Resolves: strike "gains nothing"; state the trade as "a Discord session can read and persist
   into everything the owner's account can, unattended"; the service account's value rises with the
   coder count, the opposite of what the proposal implies.
3. Serious. D buys three or four hand steps, once. By the proposal's numbers C is 18, 16 if W3 and W4
   hold; W3 is refuted and W4 unverified, so C is 17 or 18 against D's 14. The owner's complaint is 40.
   Resolves: recommend C; keep `same-account` as what 0022 already grants it (development and scratch),
   not the installed default; if the owner still picks D after objection 2, that is their call with the
   trade stated.
4. Serious. D puts the owner's account name back into every session path (ADR 0018). The root default
   `%LOCALAPPDATA%\Broodforge\<instance>` makes `BROODFORGE_TREE`, `cwd`, `CLAUDE_PROJECT_DIR`, the
   engine's project slug, Go test output and every stack trace carry it; the 2026-10-17 history rewrite
   was for exactly this, and the service account is what made paths carry `broodforge`. The push scan
   refuses commits, but transcripts, the outbox, audit details and pasted test output are not scanned.
   Resolves: in every PC mode default the root to a path with no user segment (`C:\Broodforge\<instance>`
   or `%ProgramData%\Broodforge\<instance>`; verify a standard user can create it) with the key under it.
5. Serious. D's exit cost line is boilerplate. "Back to C by one config key and `setup host`; nothing
   else differs" hides a program larger than `setup`: move the store, canonicals, session trees, harness
   directories and key into another profile, re-ACL, re-register the task, re-run `credential login`
   where a config directory does not survive, re-point `credentials.keyFile`. The honest exit is the one
   the proposal already has for the second PC: `config export`, a fresh `setup` under the account, 0021
   restore for the store. Resolves: say that, and drop "setup host does the move".
6. Serious. W3 refuted (table); C's sign-in step survives as written. Resolves: replace W3 with the
   `Start-Process -Credential` claim from an elevated interactive `setup host` and spike it in P13; the
   password prompt stays (W2), the sign-out goes.
7. Serious. Manage Roles is asked for once and held forever. A PC session reads the bot token in both
   modes (0022 Windows), so a compromised session can create roles carrying every permission the bot has
   and give them to anyone; the invite's current set has no such lever. The one C6 part that would make
   the create safe (a new role landing below the bot's) is undocumented. Resolves: default to the owner
   making the role (two steps, call 4), `--create-role` opt-in whose last `fix` line tells the owner to
   remove Manage Roles from the bot's role in Server Settings and whose check re-reads the bot's
   permissions and reports it gone.
8. Serious. Rerun on a half-configured or live instance is underspecified, and the refusal has no
   documented lift. (a) `running` is lifted only by `broodforge stop`, which interrupts every open session
   (R-17); the message must say so, and `--check` reads only, so it must not refuse. (b) "A present key is
   never changed" meets `channels` (found id differs from the configured key), `token` (a rotated token:
   the id resolves, the value is dead, `discord check` fails later with no path back through setup),
   `engine` (a moved path). Resolves: a disagreement prints `fix  <step>  config has X, found Y`, changes
   nothing, exits 1; `token` re-reads when `--token` is given or the stored value fails C1; a
   "change one setting" verb per the prior-art finding (its point 6) rather than flags on setup.
9. Serious. S1's single-writer invariant is judged by "the socket answers" at start, then the store is
   written minutes later while the owner clicks through the portal; D's logon task, a reboot or
   `Start-ScheduledTask` starts `serve` in that window and `serve` does not check for a setup. Resolves:
   `setup` binds the control socket or pipe name for its lifetime, so `serve` fails `start-control` and
   a second `setup` fails `running`; the vault comment stays true.
10. Serious. The counts hide steps by bundling. The binary ("a step in every count") is absent from D's
    14-step PC list; step 2 is two actions by the model's own rule (a page of clicks, a copy into the
    password manager); step 14 is a browser OAuth round trip (run, sign in, authorize, paste a code when
    the callback fails: the engine docs name that prompt); step 8 presumes a GitHub token the owner mints
    in GitHub settings, counted nowhere in any option; the second-PC 8 builds nothing. D is 17 to 19 by
    the stated model. The ranking does not change; the quoted number will. Resolves: recount every option
    with the model applied the same way; the GitHub token and the login round trip in all of them.
11. Minor. C3 through disgo takes the OAuth2 route with an empty bearer token (table); name it in the
    pass so the coder does not add a raw request.
12. Minor. C4's write replaces `flags`: OR the existing value with `1<<19`, skip when `1<<18` is set, and
    `fix` when the PATCH is refused (a verified application cannot set the limited flag).
13. Minor. W1 side effects: `setup service` reads back the task XML and requires `PT0S` (`fix` otherwise);
    starts the binary hidden (`conhost --headless` or a hidden-window launcher) or the owner gets a console
    window at every logon; D's risk line says the bot is down from a reboot until the owner signs in,
    where the status quo task runs at boot. These are D-only costs the count does not show.
14. Minor. W4 cites a file that does not say it; the claim moves to P13 as a spike with the engine docs
    as the citation (table).
15. Minor. `config export` carries `privacy.identities` (the owner's commit name and email) and the
    guild and channel ids: not secret, but personal; the default `--out` must land outside any repository
    and the docs rule applies to the file. The second-PC case assumes the first instance is retired: two
    live instances on one application and guild answer the same lobby and register the same commands
    (06: one application per instance). One sentence says so.
16. Minor. Drivability gaps: the `guild` poll has no bound (`--yes` refuses `needs-input: guild`, a
    `--wait <seconds>` for tests); two secrets and one stdin (the token on stdin is defined, the git
    credential is not: define a second line or defer it to `credential add`); the one-time key print
    under `--json` must still be the one print (05 Credentials).
17. Minor. "The id from the repository name" must satisfy `^[a-z][a-z0-9-]{0,31}$` (03 Configuration);
    state the normalisation or ask.
18. Minor. The prior-art finding has channel creation as an owner-only verb run after the bot joins,
    rerun-safe, refusing when the structure exists (Modmail), and warns against folding it into the host
    install. Keep it inside `setup` and also as `setup channels` alone, so the droplet owner (someone
    else's server) runs that step by itself and a rerun does not pass the token step again.
19. Minor. The Question's ten-year sentence is a wish, not a prediction; A's "Moving parts: none" is
    fine, D's "a second backend default on one host" is not the moving part (the backend exists): the
    moving part is the owner's interactive session as the bot's home. Say that.

## Pre-mortem

Ten years on, simple mode is the only PC mode anyone ever ran: the Windows `setup host` was never built
because no step needed it, C37 never closed, and the droplet carried the hardened path alone. A
milestone-2 contributor's repository shipped a `CLAUDE.md` asking sessions to "sync the developer
environment"; a policy-mode session on the PC, unattended at night under the logon task, copied
`~/.claude/.credentials.json`, the SSH keys and the browser profile out through git, and wrote a hook
into the owner's `~/.claude/settings.json`, which the owner's own next interactive session ran. The
handoff scan refused the commit; the exfiltration was a `curl`. Two lines should have caught it: "the
argument against is that one prompt-injected turn can read the owner's credentials", written and then
outweighed by four hand steps, and "Exit cost: back to C by one config key ... nothing else differs",
which made the upgrade look cheap enough to postpone forever.

## Recommendation

Take C, not D: `setup` in the binary, bot-made channels, the service account kept on the PC as ADR
0022 decided, with W3 replaced by the `Start-Process -Credential` claim and W4 spiked in P13, the
role created by the owner unless they opt in to Manage Roles with the removal step printed, the root
defaulted to a path without a user segment on both PC modes, `setup` holding the control socket while
it runs, and every count redone with the step model applied uniformly (C lands near 17, D near 17 to
19 once bundling is undone, so D's margin is smaller than its page suggests). Simple mode stays what
0022 already makes it, the development and scratch backend, one config key away for an owner who
chooses it after reading objection 2 in the owner's question. Confidence: high that objections 1 and 2
stand as written (both are in the cited texts and the engine's own docs); high that C4 holds and the
portal step goes; medium-high that C is the right default rather than D, since the owner has said simple
mode "might be fine for milestone 1" and the choice is theirs once the trade is stated correctly.
