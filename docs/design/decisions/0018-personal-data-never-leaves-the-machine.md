# 0018. Personal data never leaves the machine: one scan, three callers, roles in committed text

Date: 2026-10-17
Status: accepted

Record: no proposal (not a listed major choice; an independent reviewer reads this record).
Evidence: `../../findings/2026-10-17-personal-data-leak-protocol-gap.md`. Contract, harness group
and briefs: [the sibling](0018-privacy-scan.md). Amends the living key table of ADR 0006; adds rows
to ADR 0013 by this record (its item 4); replaces pusher checks 4 and 5.

## Context

A downstream project pushed a machine user name inside home paths five times (a hook test case, an
idea quoting a refused path, a handoff line). Its pusher compared only the net diff of the push, so
a fix on top made the range look clean, and nothing read what each commit carried. Hatchery has the
same shape: check 5 reads `git diff @{u}..HEAD` for secrets only, check 4 reads a subject and a
trailer, the guard's rules 1 to 19 read the command line and never what a push sends, the brief and
docs rules say nothing about user names, emails or people in committed text, the finding skill
invites pasted output, and the reviewer has no privacy item. Hatchery's own public history carries
a third party's name and email as author and committer on its first 28 commits (the machine's
global git identity at the time, which no step set or checked), two first names in one finding, and
a `Claude-Session:` trailer with a session link in 25 messages, which no check matched.

The downstream fixes (a per-commit pusher check, a guard rule on every push) are sound in shape and
carry two defects a straight copy would bring: both exempt the repository's `git config user.email`,
the very identity that leaked, and the expression misses the project-folder spelling of a path
(`C--Users-<name>-...`, as Claude Code writes under `~/.claude/projects/` and in every scratchpad
path), a bare user name and a host name without a desktop marker.

Ruled out: a check over the net diff; exempting the committer email; trusting the global identity.
Verified 2026-10-17: no file in this tree writes the `Claude-Session:` trailer (grep over `.claude/`
and `docs/`); the guard's harness already builds scratch repositories with a bare `origin` per case;
`fitness.json` covers every `.claude/hooks/*.js`, so a new hook script is inside the zero budget.

## Decision

1. One scan, three callers. `.claude/hooks/privacy-scan.js` (CommonJS, Node built-ins only) is
   both a module and a CLI. The guard's new rule 20 requires it; pusher check 5 runs the CLI;
   `.claude/hooks/git/pre-push` runs it for pushes from a terminal or a script. The pusher's read
   for people, numbers, addresses and identifiers is the one part no script does.
2. What it reads. For every commit in `git rev-list <refs the push sends> --not --remotes` (every
   remote, not only the target: a commit any fetched remote already has is published, which is
   what lets a decant's first push leave the base's history alone): author name and email,
   committer name and email, the whole message, every added line against each parent, every added
   file name; and each annotated tag sent, its message and tagger. Never the net diff.
3. Classes, each with a one-word label the hit carries: `email`; `home` (a home path with a user
   segment, in the drive, root, cygdrive, mnt and project-folder-slug spellings); `host` (a
   desktop marker); `word` (the machine's own user name and host name, read from the OS at run
   time, and every literal in `~/.claude/privacy-never.txt`, as whole words, any case: this is how a
   bare user name, a bare host name and a listed person's name are caught); `identity` (an author
   or committer whose name or email is not in `privacy.identities`); `session` (a `Claude-Session:`
   trailer, or a `claude.ai` URL with a path); `transcript` (a Claude Code transcript line);
   `secret` (check 5's list, moved here). Patterns and exemptions: the sibling, section 2.
4. Policy, two keys under ADR 0006, both additions with preserving defaults: `privacy.allow`
   (literal strings, case-insensitive, default `["user"]`: segments and words that are not a
   person, such as a service account named after the project) and `privacy.identities` (a list
   of `{ "name", "email" }`, default `[]`, which switches the `identity` class off with a notice).
   The committer email is never exempt on its own: only an identity the policy lists passes.
   Decant and adopt ask the owner for the identity, write it into the policy, and set the
   repository's `user.name` and `user.email` from its first entry; the global identity is not read.
5. The guard. Rule 20, tried right after rule 3, on `push` and `send-pack` from every caller, the
   main session and `--dry-run` included; refs and remote as git picks them (the sibling, section
   1); a range it cannot compute denies with its own sentence. Fail mode as the guard's: closed.
6. The pusher. Check 4 becomes: each message has a subject, a `Co-Authored-By:` trailer, and no
   `Claude-Session:` trailer. Check 5 becomes: run the CLI over the outgoing range (exit 0 or
   stop), then the read of every outgoing commit, not the net diff, for a person's name other than
   an identity in the policy, a phone number, a street address, an account or subscription
   identifier, a recorded conversation. A hit the CLI finds that the guard let through is a hook
   bug: stop and report it, never push.
7. The pre-push hook. `.claude/hooks/git/pre-push` (POSIX sh, so Git for Windows runs it) calls
   the CLI with `--pre-push`; a nonzero exit refuses the push. The base's policy sets
   `git.hooksPath` to `.claude/hooks/git` (rule 14 then allows exactly that value); decant, adopt
   and every existing clone run `git config core.hooksPath .claude/hooks/git` once, from the main
   session. `--no-verify` from a terminal still bypasses it; that is the owner's own act.
8. Lifting a refusal. A true hit is fixed by editing to `~`, `<repo>`, `../<dir>`, a role or a
   placeholder and folding the edit into the unpushed commit: `git commit --amend` for the last,
   `git reset --soft <last pushed sha>` and one new commit for several (rules 10 and 12 stay: no
   interactive rebase, no filter). The `identity` class is fixed by setting the repository identity
   and `--amend --reset-author`. A false positive is the owner's: a literal added to
   `privacy.allow` from the main session, named in that commit's message. The never list is the
   owner's file. Nothing in a brief lifts a hit; the pusher has no lift; the guard has no grant.
9. Committed text. `.claude/rules/docs.md` gets a section "Personal data": tracked text (docs,
   messages, tests, fixtures, hook examples) never carries a home path with a user segment in any
   spelling, an email other than a listed identity, the name of a person other than a listed
   identity (roles: the owner, the coordinator, a downstream project's owner), a host name, a
   session link, a recorded conversation or a grant file's content; paths are written `~`,
   `<repo>`, `../<dir>` or `<scratchpad>`. The brief skill keeps absolute paths in briefs (they are
   not committed) and says that whatever the agent writes into a tracked file follows that
   section. The finding skill's paste rule: output goes in as classes, counts and `file:line`,
   scrubbed before pasting; a value that would itself be a hit is never quoted, in a finding or a
   report. The reviewer gets one item: any class above in the diff is a blocking finding.
10. The `Claude-Session:` trailer is dropped. The commit skill's message ends at the
    `Co-Authored-By` lines; the scan denies the trailer wherever it comes from. The 25 messages
    that carry it are history, the owner's rewrite below.
11. The reference. `docs/reference/claude-code-on-windows.md` states that every absolute path,
    scratchpad and project-folder slug on a Windows PC carries the machine user name, and that a
    grant's `project` scope is such a path.

Out of scope, the owner's separate steps (an operator entry each, from the coordinator): the
history rewrite of the 28 root commits and the trailers, which starts at Hatchery, reaches every
project that cloned it and re-records each base line; and whether the Hatchery repository goes
private until that rewrite has landed. This record prevents a recurrence; it does not clean history.

Not covered, by design: pull-request titles and bodies and `gh repo create --push` (no PR flow in
the base; trigger: the first project that adopts one); a `--no-verify` terminal push; names of
people the never list does not hold (the pusher's read and the reviewer's item). Deferred: a
write-time warning from `post-edit.js` using `--text`, trigger: the first hit the scan reports in
this tree after the scan ships (the CLI already accepts `--text <file>` so the hook needs no new
contract).

## Consequences

- Every push from every caller is read commit by commit before it leaves, and the three readers
  agree because they are one script; a difference between the pusher's finding and the guard's is
  a bug, not a judgment call.
- The owner's identity is in the policy file, which is tracked. It is the identity on every
  commit already, so this publishes nothing new; a project whose owner wants no name in the tree
  lists a noreply address and uses it as the repository identity.
- Costs: a hook script and a sh hook to keep; a harness group; one more per-clone config step at
  decant and adopt; the guard runs git on every push (bounded by one timeout per command). The
  `word` class can hit a common word when a user name is one; `privacy.allow` is the answer.
- A hit shows the matched text to the agent that was refused. That agent's report may carry it
  only as the class and location (item 9); the paste rule is what keeps a refusal from becoming
  the next leak.
- Supersede when Claude Code passes hooks a parsed push plan, or when a project needs a PR scan.

Predictions (the trigger scan reads these):
- Prediction, check by 2027-04-17: no commit on any remote of this tree made after this ADR
  carries a hit of any class above; every refusal was fixed by an edit, never by an allow entry
  for a real value.
- Prediction, check when the next sibling update runs: each sibling takes rule 20, check 5 and the
  two keys by the migration line alone, with no edit to the scan.
