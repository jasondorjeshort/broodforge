# 0018, sibling: the privacy scan contract, harness group, briefs and migration

Companion to [ADR 0018](0018-personal-data-never-leaves-the-machine.md). Sections 1 to 5 are the
contract a coder builds from; section 6 the harness group; 7 and 8 the two briefs; 9 the
migration per sibling. No value that would itself be a hit appears here: every example is a
placeholder with `<name>`.

## 1. What the scan reads

Inputs: a repository directory, a list of local refs, a remote name or none, the policy. The refs
and remote come from the caller: rule 20 and `--pre-push` compute them, the CLI takes them.

Refs a push sends (ported from the downstream guard's `pushSources`, unchanged in meaning): each
refspec's source (`HEAD` as itself; a glob such as `refs/tags/*` as every ref it matches, a bare
glob under `refs/heads/`; a deletion sends nothing); with no refspec, every ref when
`remote.<r>.mirror` is true, else `remote.<r>.push`, else `push.default` (`matching`: each local
branch the remote has; `nothing`: none; `current`, `simple`, `upstream` or unset: `HEAD`); every
local branch for `--all` or `--branches`; every tag for `--tags` or `--follow-tags`. The remote as
git picks it: the command line's, `--repo`, `branch.<b>.pushRemote`, `remote.pushDefault`,
`branch.<b>.remote`, `origin`; a URL or path is "no remote"; a name that is not a configured remote
is an error. The repository and config an invocation acts on (`-C`, `--git-dir`, `--work-tree`,
`-c`, `--config-env`, `GIT_DIR`, `GIT_WORK_TREE`, `GIT_CONFIG_COUNT`/`_KEY_n`/`_VALUE_n`, prefixed
or exported earlier in the line) are honoured; a value the shell reader cannot see is an error.

Range: `git rev-list <refs> --not --remotes` (every remote; `--not --remotes` is dropped when the
repository has no remote at all). Per commit, read in this order: author name and email, committer
name and email, the message (`%B`), added lines (`git show -p -U0 --cc --no-renames --format=`
per commit: the lines a merge adds against every parent), added file names (`--diff-filter=A`).
Per annotated tag among the refs and not on any remote: the tag message and the tagger name and
email. Each git call has a timeout of 20 s and a 512 MiB buffer; a failure is an error, never a
pass.

## 2. Classes, patterns, exemptions

Matching is per line of text. A hit is `{ class, where, match }`; `where` is `<sha7> message`,
`<sha7> author`, `<sha7> committer`, `<sha7> <path>:<n>` (line number in the new file),
`<sha7> file <path>`, `tag <name> message` or `tag <name> tagger`; with `--text`, `<file>:<n>`.

| Class | Matches | Passes when |
|---|---|---|
| `email` | `[\w.+-]+@[\w-]+(\.[\w-]+)+`, not preceded by `[\w.+-]` | the address, lower-cased, is in `privacy.identities`; is `noreply@anthropic.com` or ends `@noreply.anthropic.com`; its domain is RFC 2606 (`example.com/net/org`, `.test`, `.example`, `.invalid`, `.localhost`); its user part is `git`, `x-access-token` or `oauth2`; or it directly follows `http://`, `https://`, `ssh://` or `git://` (URL userinfo) |
| `home` | the segment after `<drive>:[\\/]+Users[\\/]+`, any case; after `/home/`, `/Users/`, `/cygdrive/<d>/Users/` or `/mnt/<d>/Users/` where the path starts (line start, whitespace, a quote, a backtick, `=`, `(`, `+`, `:`, `[` or `/` before it); and the project-folder slug: the segment after `<drive>--Users-`, or after `-home-` or `-Users-` at a word start, up to the next `-`, whitespace or quote | the segment starts with `<`, `$`, `%` or `~` (a placeholder); is a Windows system profile (`Public`, `Default`, `Default User`, `All Users`, any case) or exactly `...`; or, lower-cased with trailing dots dropped, is in `privacy.allow` |
| `host` | `<word>-Desktop` or `DESKTOP-<word>`, any case, a word character on the far side of the hyphen | the host word, without the marker, is in `privacy.allow` |
| `word` | each literal in the never set as a whole word (`\b`, any case): `os.userInfo().username`, `os.hostname()`, and every non-empty, non-`#` line of `~/.claude/privacy-never.txt` (`~` is `HOME`, else `USERPROFILE`), trimmed; the file absent is an empty list | the literal is in `privacy.allow`; a literal shorter than 3 characters is skipped with a notice |
| `identity` | an author, committer or tagger whose `name` or `email` (email lower-cased, name exact after trim) is not one entry's pair in `privacy.identities` | the list is empty: class off, one notice on stderr |
| `session` | a trailer line `^Claude-Session:`; a URL whose host is `claude.ai` or ends `.claude.ai` and whose path has at least one segment | never |
| `transcript` | a line that parses as a JSON object with both `sessionId` and `parentUuid` keys (the Claude Code transcript shape; the coder verifies the key names against one real transcript line and records the check in the commit message) | never |
| `secret` | `-----BEGIN [A-Z ]+-----`; `AKIA[A-Z0-9]{16}`; `sk-ant-[A-Za-z0-9_-]{8,}`, `ghp_[A-Za-z0-9]{20,}`, `xox[bp]-[A-Za-z0-9-]{8,}`; an added file named `.env` or ending `.pem` or `.key`; `(password|secret|token)\w*\s*[:=]\s*["'][^"']{4,}["']`, any case | never: the full key shapes do not match a bare prefix in prose (the pusher's and this file's lists) or the harness's pieced samples, so no file is exempt |

The scan never reads `git config user.email` or `user.name` as an exemption. The `email` class
runs on names and emails of author, committer and tagger too, so a third-party address there is
reported under both `email` and `identity`.

## 3. The CLI (promised)

```
node .claude/hooks/privacy-scan.js [--remote <name> | --no-remote] [--json] [<ref> ...]
node .claude/hooks/privacy-scan.js --pre-push <remote-name> <remote-url>     # stdin: git's lines
node .claude/hooks/privacy-scan.js --text <file> [--json]
```

Defaults: refs `HEAD`; remote as git picks it (section 1). `--pre-push` reads git's
`<local ref> <local sha> <remote ref> <remote sha>` lines and takes every local ref whose local sha
is not all zeros; the remote name is `<remote-name>`, "no remote" when it is a URL. `--text` scans
a file's lines as added lines under `<file>:<n>` (classes `email`, `home`, `host`, `word`,
`session`, `transcript`, `secret`). Policy through `HOOK_POLICY_FILE` as the hooks do; a policy that
does not load is an error.

Output, stdout: one line per hit, `HIT <class> <where>: <match>`; last line
`privacy-scan: <n> hits in <c> commits` or `privacy-scan: clean, <c> commits` (`<c> lines` with
`--text`). Notices on stderr, prefixed `privacy-scan:`. Exit 0 clean, 1 hits, 2 cannot compute,
with `privacy-scan: cannot <reason>` on stderr and no last line. `--json` prints one object:
`{ "commits": <c>, "hits": [{ "class", "where", "match", "commit": "<sha40>|null" }], "error":
null | "<reason>" }`, same exit codes.

## 4. Rule 20 and the pre-push hook

Rule 20, row for the hook specs table: tried right after rule 3: `push` or `send-pack` from any
caller, the main session and `--dry-run` included, when `privacy-scan` reports a hit or an error
for what it sends, with the repository and config of the invocation; an opaque push is left to rule
19. Reason prefix `personal data in the outgoing commits`. Advice on hits: `<n> hits, the first
three: <where> (<class>: <match>); ... Fix: write ~, <repo>, a role or a placeholder and fold the
edit into the unpushed commit (amend, or reset --soft and one new commit); a false positive is the
owner's privacy.allow entry, from the main session. Nothing in a brief lifts this. ADR 0018.` On
an error: `The guard could not work out what this push sends (<reason>); no push leaves without the
scan. Fix the repository state (a commit on the branch, a configured remote) or ask the user.` The
scan result is cached per (repository, refs, remote) within one guard run.

`.claude/hooks/git/pre-push`: POSIX sh, executable bit set in git (`git update-index --chmod=+x`),
LF line endings (`.gitattributes` line `*.sh text eol=lf` and the same for the hook path):

```
#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
top=$(git rev-parse --show-toplevel) || exit 2
exec node "$top/.claude/hooks/privacy-scan.js" --pre-push "$1" "$2"
```

Node absent: the hook fails and the push is refused, as decant step 1 already requires Node.
Policy: `git.hooksPath` in Hatchery's own `policy.json` becomes `.claude/hooks/git`; the base
default stays `null`. Per clone, once: `git config core.hooksPath .claude/hooks/git` (decant step
after the policy step; adopt step after its policy step; existing clones by the coordinator).

## 5. Surfaces added (rows for ADR 0013's table, by this record)

| Surface | Carried by | Readers | Record |
|---|---|---|---|
| `privacy.allow`: literal strings, case-insensitive, default `["user"]`; never patterns (a `*` or `/` in an entry is malformed) | `policy.json` | `privacy-scan.js`, rule 20, pusher check 5 | this sibling, section 2 |
| `privacy.identities`: `[{ "name": <non-empty string>, "email": <string with one @> }]`, default `[]`; the first entry is the repository identity decant and adopt set | `policy.json` | the scan; decant and adopt | ADR 0018 item 4 |
| The never list: `~/.claude/privacy-never.txt`, one literal per line, `#` comments, absent is empty | the owner's file | the scan | section 2, class `word` |
| The scan CLI: flags, `HIT` lines, the last line, exit codes, the `--json` object | `.claude/hooks/privacy-scan.js` | pusher check 5, `pre-push`, a project reproducing a refusal | section 3 |
| The pre-push hook path `.claude/hooks/git/pre-push` and the `core.hooksPath` value `.claude/hooks/git` | the hook file; each clone's config | git; rule 14 through `git.hooksPath` | section 4 |

Not promised: the wording of notices and advice; hit ordering; harness case ids.

## 6. Harness group `PV` (in `test-hooks.js`, scratch repositories only)

Sample data is built from pieces so no line of the harness matches the scan that runs on the push
carrying it. Each case: a scratch repository with a bare `origin`, one pushed base commit, then
outgoing commits; `HOOK_POLICY_FILE` a scratch policy with `privacy.identities` holding the scratch
identity and `privacy.allow` as the case needs; `HOME` pointed at a scratch dir (the never file).
Cases: PV01 a home path in an added line, three slash spellings, each denied; PV02 the slug
spelling (`<drive>--Users-<seg>-`), denied; PV03 a placeholder segment (`<name>`, `$USER`, `~`,
`Public`), allowed; PV04 an email at a reserved domain, allowed, and one at another domain,
denied; PV05 the repository `user.email` set to an address not in `privacy.identities`: the commit
is denied under `identity` and `email` (the ruled-out exemption); PV06 `privacy.identities` empty:
PV05's repository passes with the notice; PV07 a hit in the second of three outgoing commits,
removed by the third: denied (the net-diff defect); PV08 a hit in a commit already on a second
remote (`hatchery`): allowed; PV09 a `Claude-Session:` trailer in a message: denied; PV10 a
`claude.ai` URL with a path in an added line: denied, the bare host allowed; PV11 the never file
holding a word that an added line carries as a whole word: denied, as part of a longer word:
allowed; PV12 the OS user name of the running process as a bare word in a message: denied, and in
`privacy.allow`: allowed; PV13 a `DESKTOP-<word>` marker: denied; PV14 an annotated tag whose
message carries an email: denied on `git push origin v1`; PV15 a `.env` file added: denied under
`secret`; PV16 a transcript-shaped JSON line: denied; PV17 `git push --dry-run`: denied like a
push; PV18 a push from the main session (no `agent_type`): denied; PV19 a remote that is not
configured: denied with the error sentence; PV20 the CLI with `--json` on PV07's repository: exit
1, `commits` 3, one hit with its `commit` sha; PV21 `--text` on a scratch file with two classes:
exit 1, two `HIT` lines with `<file>:<n>`; PV22 the `pre-push` script on PV01's repository with
git's stdin lines: exit 1; on a clean one: exit 0; PV23 `privacy.allow` holding a glob, and
`privacy.identities` missing `email`: both malformed policies (the guard denies every git command).
Group `F` runs unchanged: F1 covers the new script's requires.

## 7. Coder brief (Opus)

Goal: the scan, rule 20, the pre-push hook, the keys, group `PV`; no skill or agent edits (the
tester's). Read: ADR 0018, this file, `.claude/hooks/git-guard.js` (rules, `RULES` order,
`shell-parse.js` output), `.claude/hooks/policy.js` (`DEFAULTS`, `SCHEMA`, `TYPES`),
`.claude/hooks/test-hooks.js` (the GV cases' scratch repositories, `scratchPolicy`), ADR 0006.
Port, read-only with `git -C ../broodforge show HEAD:<path>`, never `cd`: that guard's
`pushSources`, `pushRemote`, `gitContext`, the exported-environment tracking its rule 20 needs,
and its `PRIVACY` expression, then apply section 2's differences (no `user.email` read; the slug
spelling; `word`, `identity`, `session`, `transcript`, `secret`). Files: new
`.claude/hooks/privacy-scan.js`, new `.claude/hooks/git/pre-push`, `.gitattributes`; edit
`git-guard.js`, `shell-parse.js` if the environment tracking needs it, `policy.js`, `policy.json`
(the two keys with Hatchery's values and `git.hooksPath`), `policy.examples/*.json` (a `$comment`
line only where the example should show the keys), `test-hooks.js`, the hook specs
`0002-hook-specs.md` (row 20; a section for `pre-push`), the ADR 0006 key table (two rows, living).
Interface: sections 2 to 5 exactly. Dependencies: none (group `F`). Verify: `node
.claude/hooks/test-hooks.js` all pass with `PV` in the groups line; `node
.claude/hooks/privacy-scan.js` on this tree exits 0 at HEAD; the OS user name of this machine as a
scratch added line is denied (PV12); `node .claude/hooks/policy.js` prints the two keys. Report:
cases added, the transcript key check, any place the contract was silent and what was chosen.
Use the Write tool for files with apostrophes. Attribution: Claude Opus 5.5.

## 8. Tester brief (Sonnet)

Goal: the protocol text of ADR 0018 items 4, 6, 7, 9, 10, 11 and the record updates; no hook or
script edits (the coder's). Read: ADR 0018, this file, each file below. Files: `.claude/agents/pusher.md`
(checks 4 and 5 as item 6; check 6's range to the same `--not --remotes` list); `.claude/agents/reviewer.md`
(one rule line: any class of ADR 0018 in the diff is blocking); `.claude/skills/brief/SKILL.md`
(one rule line, item 9); `.claude/skills/finding/SKILL.md` and `template.md` (the paste rule);
`.claude/skills/commit/SKILL.md` (no `Claude-Session:`; the fix path of item 8 under Pushing);
`.claude/skills/decant/SKILL.md` and `.claude/skills/adopt-hatchery/SKILL.md` (the identity question
in the owner's words: "Which name and email go on this project's commits?", the policy keys, the
repo-local config, the `core.hooksPath` line, in the verify step `git config core.hooksPath`
prints `.claude/hooks/git`); `.claude/rules/docs.md` (section "Personal data", item 9);
`.claude/agreement/git.md` (one line: every push is scanned commit by commit, ADR 0018);
`docs/reference/claude-code-on-windows.md` (item 11, `verified:` bumped); ADR 0013's header (a
`Superseded in part by 0018` line naming the rows of section 5); `docs/design/README.md` (the
0013 entry). Verify: the pusher's check 7 lines and `node .claude/hooks/policy.js --claude-check`
print no `FAIL`; `grep -rn "Claude-Session" .claude` empty; no edited file names a sibling project.
Report: files changed, lines added, any sentence the ADR left open. Attribution: Claude Sonnet 5.5.

## 9. Migration, one line per sibling

- The downstream project with its own rule 20 and check 5a: at its next update the base's rule,
  check and scan replace them; it moves `git.push.privacyAllow` to `privacy.allow`, sets
  `privacy.identities` and `git.hooksPath`, runs the `core.hooksPath` line; its finding and
  operator entry stay its own.
- The build-phase sibling on master: takes the update with ADRs 0015 to 0018, sets the two keys
  and the hooks path, runs the config line; its branches carry the 28 commits, so the owner's
  rewrite reaches it.
- The two adopted projects: the same three steps at their next update, after the patch-update
  ideas in the handoff are fixed.
