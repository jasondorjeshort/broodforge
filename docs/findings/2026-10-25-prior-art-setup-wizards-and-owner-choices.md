---
date: 2026-10-25
status: resolved (ADR 0027)
area: setup
promoted_to: docs/design/decisions/0027-setup-in-few-steps.md
---

# Good setup tools prompt for what is irreversible, take a flag for everything, default the rest

## Question

Broodforge's setup is about 43 hand steps on Windows and 45 on a Linux droplet (accounts, a
scheduled task or systemd unit, the credential store, a Discord application, a server with channels
and a role, JSON config). Proposal 0027 weighs a `setup` command, a bot that creates its own
channels, and a single-account default on the PC. Several of its points are owner calls (isolation
mode, which server, what the bot may create). The coordinator asked: in well-regarded self-hosted
tools, is such a choice put to the owner interactively, taken from a flag, or defaulted in a config
file with the setting documented, and when does each win? Also asked per tool: rerunnability,
secret entry, Discord channel and role creation with the invite's permissions, and the single next
action printed. Ruled out beforehand: a separate configuration app.

## What was tried

About twenty-five web searches and forty fetches on 2026-10-25; no installer was run. "Verified"
means a primary page or the tool's own source was fetched; "secondary" means search snippets,
mirrors or forums only. Primary sources, by tool:

- `gh auth login`: cli.github.com/manual/gh_auth_login; source
  raw.githubusercontent.com/cli/cli/trunk/pkg/cmd/auth/login/login.go and `.../auth/shared/login_flow.go`.
- Tailscale: tailscale.com/kb/1241/tailscale-up, /kb/1080/cli, /kb/1085/auth-keys, /kb/1031/install-linux.
- Home Assistant: home-assistant.io/getting-started/onboarding/, /docs/configuration/basic/.
- Gitea: docs.gitea.com/installation/install-from-binary, /installation/install-with-docker,
  /administration/command-line.
- Nextcloud: docs.nextcloud.com/server/latest/admin_manual/installation/command_line_installation.html
  and `.../installation/automatic_configuration.html`.
- Mastodon: docs.joinmastodon.org/admin/install/; raw.githubusercontent.com/mastodon/mastodon/main/lib/tasks/mastodon.rake.
- Red-DiscordBot: docs.discord.red/en/latest/install_guides/debian-12.html, /bot_application_guide.html,
  /cog_guides/core.html; source raw.githubusercontent.com/Cog-Creators/Red-DiscordBot/V3/develop/redbot/
  `setup.py`, `__main__.py`, `core/_cli.py`.
- Modmail (Discord): docs.modmail.dev/usage-guide/configuration, docs.modmail.dev/installation.md;
  source raw.githubusercontent.com/modmail-dev/Modmail/master/cogs/modmail.py.
- Discord Tickets (Discord): discordtickets.app/self-hosting/installation/docker/, /configuration/.
- Synapse: element-hq.github.io/synapse/latest/setup/installation.html;
  raw.githubusercontent.com/element-hq/synapse/develop/docker/README.md.
- Others: docs.docker.com/reference/cli/docker/init/; developers.cloudflare.com/workers/wrangler/commands/general/
  and `.../commands/workers/`; guides.rubyonrails.org/security.html; github.com/dani-garcia/vaultwarden/wiki/Enabling-admin-page;
  docs.immich.app/install/post-install/ and /install/config-file/; caddyserver.com/docs/automatic-https.
- Discord platform: docs.discord.com/developers/topics/permissions and /topics/oauth2.

Not verified: whether `gh`'s pasted-token prompt is masked (in a prompter file not fetched);
whether Wrangler's secret prompt hides input (the page shows `***`, an inference); what
`occ maintenance:install` does on an installed instance; Discord Tickets' invite permissions
(no page found); whether Discord lets the authorizing user untick requested permissions (the
OAuth2 page does not say); Gitea's first-user-becomes-admin is secondary (forum posts and a mirror
of its installer text); Wrangler `init` / C3 and Rails beyond credentials were not looked at.

## Result

### Per tool

| Tool | How owner choices are gathered | Rerun and skip | Secrets | Next action printed |
|---|---|---|---|---|
| `gh auth login` | Interactive prompts (auth method, git protocol, SSH key) when a TTY; every prompt has a flag (`--hostname`, `--git-protocol`, `--web`, `--skip-ssh-key`, `--scopes`); default host github.com (manual) | Prints "You were already logged in to this account" (login_flow.go) | Browser device flow by default; `--with-token` reads stdin; `GH_TOKEN` for headless; stored in the OS credential store, plain-text fallback announced (manual) | "Logged in as %s" (login_flow.go) |
| Tailscale `up` | Flags only; login is a printed URL to visit (install-linux) or `--auth-key` | "Flags are not persisted between runs; you must specify all flags each time"; since 1.8 the CLI warns and gives a copyable full command; `--reset`; `tailscale set` "only updates the settings you explicitly set" (1241, 1080) | `--auth-key file:<path>` reads the key from a file (1241); one-off keys revoked after use (1085) | The login URL |
| Home Assistant onboarding | Web page: owner account required; location found by "Detect", which fills time zone, units and currency; "You can always change this information later"; analytics "turned off by default" (onboarding) | Restore from backup is the second door; a value set in `configuration.yaml` is greyed out in the UI (basic) | Owner password typed in the browser; "no way to recover owner credentials" | "Finish" opens the dashboard |
| Gitea | Web install page, prefilled from `app.ini` and `GITEA__section__KEY` env vars (docker); or skip it: `INSTALL_LOCK = true` with secrets from `gitea generate secret` (binary) | Locked after install; admin by `gitea admin user create --admin --random-password` (command-line); first registered user becomes admin (secondary); CVE-2026-96404, a reopened installer let a known admin name in without a password (secondary, app.opencve.io) | `__FILE` suffix reads a value from a file (docker); `app.ini` made `0640` after install (binary) | Run as a service or `gitea web -c <ini>` (binary) |
| Nextcloud | `occ maintenance:install` flags (`--database`, `--admin-user`, `--data-dir`...), "takes the place of the graphical Installation Wizard"; or `config/autoconfig.php` prefills the web form, which shows only what is missing or invalid | autoconfig deleted after success, kept on failure ("remove it or rotate any exposed secrets") | "This command prompts for the database and administrator passwords" when omitted; autoconfig "commonly contains plaintext ... passwords" | Not stated |
| Mastodon `mastodon:setup` | One interactive wizard: domain, single-user mode, Docker, DB, Redis, storage, SMTP, update checks, then "Save configuration?", "Prepare the database now?", "Create an admin user straight away?"; app secrets generated unasked | Does not check for an existing `.env.production`, only warns if `LOCAL_DOMAIN` is set; tests DB, Redis and mail with "Try again?" then "Continue anyway?" | DB and SMTP passwords `echo false`; the admin password is generated and printed once | "All done! You can now power on the Mastodon server", or a list of the errors met along the way |
| Red-DiscordBot (Discord) | `redbot-setup` prompts name, data path (computed default, Enter accepts), backend (JSON "recommended default"); `--no-prompt` needs `--instance-name`; first `redbot <name>` asks token and prefix; `--token`, `RED_TOKEN`, `--prefix`, `--owner` skip them (setup.py, `__main__.py`) | Existing instance: non-interactive refuses without `--overwrite-existing-instance`; interactive warns and asks a second time, default No | Token read by plain `input("> ")`, echoed, length-checked only (`_cli.py`); guide: "Do not share your token" | "run `redbot <name>`" plus the quickstart link (setup.py); with `--no-prompt` and no token: "Token and prefix must be set in order to login.", exit code CONFIGURATION_ERROR |
| Modmail (Discord) | `.env` file (`TOKEN`, `GUILD_ID`, `OWNERS`, `CONNECTION_URI`), then owner-only `?setup` in Discord; later changes by `config set` | `?setup` refuses when a main category exists; "run `?setup` again" if it breaks; a missing fallback category is recreated (configuration) | Token only in `.env` | "Successfully set up server. Consider setting permission levels..." with the commands to do it |
| Discord Tickets (Discord) | `docker-compose.yml` edited by hand; `ENCRYPTION_KEY` generated on the docs page; then a web settings portal; `settings` in the console prints its link | Not stated | Token and OAuth secret pasted into the compose file | "Don't invite the bot yet": publish commands, then the configuration guide |
| Synapse | `--server-name` and `--report-stats=[yes\|no]` given at generate time; it "cannot be changed later"; Docker generate refuses to run without both env vars | Synapse will not start without a config file | Signing key and `registration_shared_secret` generated; users by `register_new_matrix_user`, which prompts | The docs' run step |
| `docker init` | Interactive only; no skip flags (`--version` is the only option) | If any target file exists it offers to overwrite all; "You can't recover overwritten files" | None | `docker compose up --build`, and `README.Docker.md` |
| Wrangler | `wrangler login` opens a browser, `--browser=false` prints the link | n/a | `secret put` prompts ("Enter a secret value") or reads a pipe; tokens plain-text TOML unless `--use-keyring` | The URL |
| Rails credentials | `credentials:edit` creates `config/master.key` if missing | Same command edits later | Key in a file "not committed" or `RAILS_MASTER_KEY` | n/a |
| Vaultwarden | Env vars; admin page saves `config.json`, whose "values ... take precedence over the corresponding environment variable" | Env changes then silently ignored, the cause of "plain text ADMIN_TOKEN" warnings | `vaultwarden hash` asks the password "twice" and prints a PHC string | n/a |
| Immich, Caddy | Immich: first web user is admin; `IMMICH_CONFIG_FILE` "will disable the ability to edit other properties from the Immich web UI". Caddy: "By default, Caddy serves all sites over HTTPS", no setup step | n/a | n/a | Immich: a seven-step checklist |

### Discord-specific

- Creating structure is a separate owner-only command run after the bot joins, not part of the
  host install. Modmail `?setup` makes a `Modmail` category hidden from `@everyone`, readable by
  the bot and the configured levels, plus `#bot-logs`, then stores both ids in its config
  (modmail.py). Discord Tickets makes no channels at install; its portal does the rest.
- Invite permissions: Modmail's link carries a fixed set; "You must leave on all permissions excepted for Administrator
  (optional)", Administrator recommended during setup and restricted after (installation.md). Red's
  link is the owner's to configure (`[p]inviteset perms`), the bot "will create its own role with
  the desired permissions when it joins", and `[p]invite` is DMed to stay private (core.html).
- Platform facts: Administrator "bypasses channel permission overwrites"; "A bot can edit roles of a
  lower position than its highest role, but it can only grant permissions it has" (permissions).
  The authorize URL's `guild_id` with `disable_guild_select=true` pins the server (oauth2), so the
  server choice can be made in Discord's own dialog.

### Conclusions

#### The answer to the coordinator's question

Every tool that is well regarded for setup does all three, split by the kind of choice. Prompt
when the owner is present and the choice is consequential, hard to undo and has no safe default:
Synapse's server name, Red's instance and data path, Mastodon's domain, Home Assistant's owner. The
prompt shows a computed default that Enter accepts (redbot-setup, Home Assistant's Detect) and
checks the answer on the spot (Mastodon). Flags win for repeatability and headless hosts, and the
good tools give every prompt one (`gh`, redbot-setup, `occ`). With no TTY they never guess an
irreversible choice: they fail and name what is missing (redbot-setup without `--instance-name`,
Red without a token, Synapse without `SYNAPSE_REPORT_STATS`, a privacy choice with no default).
Config defaults win for anything with a safe value that can change later (Home Assistant
analytics off, Caddy HTTPS on, Modmail's `config set`, Immich's storage template). The settings
file is the record of every choice whichever way it was made, and a value set in a file locks the
interactive surface (Home Assistant greys the field, Immich disables UI edits, Nextcloud's form
asks only for what autoconfig lacks). Vaultwarden shows the reverse, a UI-saved file silently
overriding the environment, and it is the one people trip on. Discord choices are made in
Discord's own dialogs. The server is picked in the authorize screen, which can be pinned with
`guild_id`. What the bot may do is the invite's permission integer. What it creates is an
owner-only command run after it joins, which stores the ids it made.

#### Worth copying

1. A flag for every prompt; non-interactive mode refuses an irreversible choice and names the
   flag (redbot-setup `--no-prompt`, Synapse generate, `gh --with-token`).
2. Check each step as it is answered, offer retry or continue, end with the errors listed and one
   next command (Mastodon `mastodon:setup`; redbot-setup's "run `redbot <name>`"; `docker init`).
3. Bot-made Discord structure through an owner-only, rerun-safe command that refuses when it
   exists, recreates what went missing and writes the ids into config (Modmail `?setup`).
4. A value in the config file wins and the interactive surface shows it as fixed (Home Assistant,
   Immich, Nextcloud autoconfig).
5. The setup surface closes after use: a lock (Gitea `INSTALL_LOCK`), a deleted seed file
   (Nextcloud autoconfig), a generated secret printed once (Mastodon's admin password).
6. A separate "change one setting" verb next to first setup (`tailscale set`, `redbot --edit`).

#### Worth avoiding

- Echoing a bot token while it is typed (Red's `interactive_config` uses `input()`).
- A second store that silently beats the first (Vaultwarden `config.json` over env vars).
- Overwrite-all with no backup on rerun (`docker init`); not detecting an existing config
  (Mastodon checks only `LOCAL_DOMAIN`).
- Making a rerun restate every earlier choice (`tailscale up`, mitigated by its copyable command).
- Asking for Administrator "for setup" (Modmail, the Tickets docs): it bypasses every channel
  overwrite, the mechanism a bot that makes private channels relies on.
- A web installer that can be reopened over live data (Gitea, CVE-2026-96404, secondary).

## Next

Resolved 2026-10-26: proposal 0027 (Constraints) and ADR 0027 (Context, Decision 2) class each
choice as prompt, flag or config default as the conclusions say, with the invite integer and the
owner-only `channels` and `role` steps for Discord. The unverified rows were not leaned on.
