---
date: 2026-10-31
kind: bug
related: [2026-10-16-p11-remainder-for-the-live-gateway.md]  # that one lists live-gateway texts still to write; this one is texts that exist and mislead
---

# The first live run's texts name commands that do not exist or the wrong ones

Seen on the owner's first `dev-live` setup, 2026-10-31 (milestone 2, the live form under the
owner's account). Every step worked; four texts sent the owner the wrong way, and each cost a
question to the coordinator.

1. `setup`'s closing line, `internal/setup/steps.go` `textNext`: "the bot is running. ... then here:
   broodforge setup login <credential id>". On a root with no service (the development form, P23 not
   run) nothing has started the bot, and `setup login` is P25, not built. The owner read "running",
   found nothing in Discord, and had to be told `serve`. The dry-run form already has its own closing
   line (`textNextDryRun`); the live form without a service needs one too, naming `serve` and
   `credential login <id>`, or the line should depend on whether a service exists.
2. `/credential add kind:subscription`'s reply, `internal/discord/text.go` (the text of `04-ux.md`,
   Credentials): "The owner finishes the login on the host (`broodforge credential login <id>` prints
   the command), sends you the sign-in link, and you send back the code". Written for a member who is
   not the owner, and for a host where `broodforge` is on `PATH`; on `dev-live` the owner is the
   member and the binary is `bin/broodforge.exe` in a checkout. The owner asked what "broodforge" was.
3. The start refusal when the member has a credential but no default, `internal/discord/adapter.go`
   (`couldNotStartText`, marked `// not 04`): "you have no default credential; add one with
   /credential add". The owner had just added one. The fix it should name is `/credential default
   id:<id>` (which exists: `gateway.go` registers `default`); and a credential still `needs-login`
   should get `04-ux.md`'s "your default credential <id> needs a login" line, which is in 04 but was
   not what printed. Whether the member's first credential should become the default on its own is
   the owner's call (humans see it).
4. The git-credential prompt, `steps.go` `textGitCredential`: "Token for pushes to it (not shown;
   Enter to add later with setup git-credential):". "It" is the repository URL asked one line
   earlier; the owner did not connect them and asked what the token was for.

5. (2026-10-31, later the same day.) The `canonical-diverged` refusal, `04-ux.md` Failure flows (marked
   pending the owner's review): "the host's copy of main and the remote have diverged; the owner
   reconciles it". The owner, hitting it after a rebase of the checkout's `main` rewrote the hashes
   the bot's canonical had fetched: "a singularly unhelpful refusal". The design already names the
   remedy (`05-debug-interface-git.md`, The fetch at session start: `git -C <canonical> branch -f
   <branch> <remote>/<branch>`, or `reset --keep` while checked out); the venue line should carry it,
   or the host should offer a one-word command for it (`canonical reset <project>`), and the line
   should say why: the remote's history was rewritten under the host's copy, not "a pull is needed".

Reproduce: `setup` with `--discord live` on a fresh root under the owner's account, no service; then in
Discord `/register`, `/credential add kind:subscription`, Start. The texts are the owner's to word
(`major-decisions.md`: what humans see), so this is a `Question:` for the texts once the owner has
proposed them, and a `Plan:` for item 3's wrong refusal, which is a code bug against 04.

## Open

- Item 1: one line keyed on whether a service exists, or a third form for "live, no service"?
- Item 3: does `/credential add` set the default when the member has none?
