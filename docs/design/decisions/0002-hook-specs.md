# ADR 0002 companion: hook specifications

Behaviour of the scripts under `.claude/hooks/` and of `.claude/statusline.js`, as implemented and
tested on 2026-10-13 (`node .claude/hooks/test-hooks.js`, 325 cases). The code is the reference;
this document describes it exactly enough to re-implement or extend it. What each project may choose
lives in `.claude/hooks/policy.json`, read through `policy.js`; its schema, defaults and fail modes
are ADR 0006. Every script: Node, `'use strict'`, standard library only, SPDX header, reads all of
stdin as JSON, runs under Git Bash and PowerShell on Windows and bash on Linux. Hook contract:
https://code.claude.com/docs/en/hooks; status line: https://code.claude.com/docs/en/statusline.

Facts the design relies on (from those pages): exit 0 with JSON on stdout means "the fields decide";
exit 1 is a non-blocking error; exit 2 blocks, with stderr as the reason; `hookSpecificOutput` must
wrap `additionalContext`; `agent_type` (the agent's name) and `agent_id` are present only when the
hook fires inside a subagent; a matcher such as `Bash|PowerShell` is an exact list of tool names;
`onFailure: "block"` (Claude Code 2.1.295 and later) turns a crash, timeout or missing interpreter
into a block, and without it only exit 2 blocks. The status line refreshes on events (session
start, each assistant message, `/compact`, a rate-limit reset), debounced 300 ms, and also every
`refreshInterval` seconds when that is set; `rate_limits` appears only on claude.ai plans, only
after the first API response, and each window may be absent on its own.

## 1. `git-guard.js` (PreToolUse, matcher `Bash|PowerShell`, `"onFailure": "block"`)

Input: `tool_input.command`, `tool_name`, `agent_type`, `cwd`. Caller: `agent_type` when it is a
non-empty string, otherwise the main session; an empty string counts as absent, because it carries
no identity and must not lock the main session out. A missing or non-string command allows.

Parsing (`shell-parse.js`). `tool_name` `PowerShell` selects PowerShell rules, anything else Bash.
The reader splits the line into simple commands at `;`, `|`, `&`, `&&`, `||`, newlines and
parentheses (and, in PowerShell, braces), and gives each its dequoted words. Bash: single quotes are
literal, `$'...'` takes backslash escapes, double quotes take `\` escapes, backslash escapes outside
quotes, `{` and `}` as words are grouping, `#` at a word start is a comment. PowerShell: single
quotes double a quote to escape it, double quotes and bare words take backtick escapes, `@'...'@`
and `@"..."@` here-strings, `#` and `<# #>` comments; a backslash is literal. Redirections (`>`,
`2>&1`, `*>`, `<`, ...) and their targets are not words. Bash heredocs (`<<`, `<<-`, quoted or not)
are read at the next newline; their bodies, quoted strings and here-strings are *data*. Commands
inside `$(...)`, backticks and `<(...)`, and in an unquoted heredoc's `$(...)`, are listed as
commands, because they run; their value in the enclosing word is unknown. A word is *opaque* when
it holds something the reader cannot see: a substitution, a variable (`$X`, `${F:-...}`, a `$name`
inside double quotes), a Bash brace expansion (`{-f,}`, `{1..3}`), or a PowerShell splat (`@a`).

Data becomes commands only when a runner runs in the line: `sh`, `bash`, `zsh`, `dash`, `ksh`,
`fish`, `pwsh`, `powershell`, `cmd`, `wsl`, `eval`, `iex`, `Invoke-Expression`, `Invoke-Command`,
`Start-Process`, `xargs`, `source`, `.` or `[scriptblock]::Create`, as a command word or after a
wrapper (`sudo`, `env`, `xargs`, `timeout`, `nohup`, `Start-Process`, ...); a runner's name as a
plain argument runs nothing. Then every data string is parsed with both dialects, three levels
deep. So `git commit -m "fix git push -n"`, a heredoc commit message and `echo "git push --force"`
allow, while `bash -c "git reset --hard"` and `echo "git push -f" | sh` are read as the git commands
they run. `Start-Process git -ArgumentList "push --force"` (or `"push","--force"`) is reassembled
into the git command it starts.

Locating git. In each command every word whose base name is `git` or `git.exe` (any case, any
path) starts an invocation. Words after it that start with `-` are global options; `-C`, `-c`,
`--git-dir`, `--work-tree`, `--namespace`, `--config-env` and `--super-prefix` take the next word,
and `--opt=value` is split. The next word is the subcommand; a git word with nothing after it is not
an invocation. Arguments are normalised: a short cluster becomes single flags (`-nm x` is `-n -m`);
an abbreviated long option of four or more characters becomes every guarded option it starts
(`--har` is `--hard`, `--forc` is each `--force*`), because git accepts any unambiguous prefix; and
each subcommand's value-taking options drop their value (`VALUE_OPTS` in the code), so `-m "-n"` and
`-mnote` are a message, not `-n`, and `-m "add $f"` is not opaque. An invocation is opaque when a
remaining argument is, when a `-c`/`--config-env` value is, or when `xargs` runs it (xargs appends
arguments from stdin). A command whose command word is opaque, in a line that names git (a parsed word
that is or contains a git token and is unquoted, or whose unquoted part does; or a quoted word that is git
or a path ending in git, as in `$g = 'git'`, or an assignment whose quoted value starts with one, as in
`$g='C:\Program Files\Git\cmd\git.exe'` or `c='git push -f'`; a quoted argument that merely contains git
or starts with it, such as a regex, `"git push"` for grep, or a `.git` path part, does not), is an
invocation with an unknown subcommand, unless it is a PowerShell assignment (`$h = git ...`) or the
word ends in a literal file name that is not git (`$(dirname "$0")/check.sh`).

An invocation is *first* when only `NAME=value`
words or prefix words (`sudo`, `env`, `time`, `then`, `do`, ...) precede it and it did not come
from data. Its directory is the input `cwd`, moved by earlier `cd`, `pushd`, `Set-Location` or
`sl` commands in the line and then by `-C`; Git Bash `/c/...` resolves as `C:/...` on Windows.

Policy. The guard loads the policy once. A malformed file denies every command that contains a git
invocation, with reason `git-guard: policy file is malformed (<error>): `<text>`. Every git command
is denied until it is fixed; ask the user.` and allows commands without one.

Rules, tried in order, each over every invocation in source order. "Branch patterns" match a whole
name with `*` matching anything, `/` included. "Targets" of a push are the destinations of its
refspecs (`HEAD` and no refspec mean the current branch, read with `git rev-parse --abbrev-ref
HEAD`); `--all`, `--branches`, `--tags`, `--mirror` and `--follow-tags`, a detached HEAD or an
unreadable branch match no pattern list but `*`.

| # | Denies ... | Reason prefix |
|---|---|---|
| 1 | `push` or `send-pack` with `--force`, `--force=...`, `-f` or `+<refspec>`; or `--force-with-lease`/`--force-if-includes` unless `git.push.forceWithLease` lists patterns matching every target of a `push` | `force push` |
| 2 | `push` or `send-pack` with `--mirror` or `--prune`; or `--delete`, `-d` or `:<ref>`, unless every deleted branch matches `git.branchDelete.branches` and, when `requireMergedPr`, GitHub shows a merged PR from it | `remote branch deletion` |
| 3 | `push` or `send-pack` from the main session when `git.push.mainSession` is false; from a subagent not allowed by `git.push.subagents`; from an allowed subagent when `git.push.branches` is not `["*"]` and some target matches none of them | `push not permitted for this caller` |
| 20 | tried right after rule 3: `push` or `send-pack` from any caller, the main session and `--dry-run` included, when what it sends matches pusher check 5a's mechanical grep (an email, a home path that carries a user name, a host-name marker with a word character on the hyphen's far side), run with the repository and config the invocation has (its directory, `--git-dir`, `--work-tree`, `-c`, `--config-env`, and `GIT_DIR`, `GIT_WORK_TREE` and `GIT_CONFIG_COUNT`/`_KEY_n`/`_VALUE_n` prefixed to it or exported earlier in the line: `export`, `declare -x`, `$env:`, `Set-Item env:`). An opaque push is left to rule 19. The remote `<r>` is git's pick: the command line's, `--repo`, `branch.<b>.pushRemote`, `remote.pushDefault`, `branch.<b>.remote`, `origin`; a name that is not a configured remote denies. The refs it sends: each refspec's source (`HEAD` as itself; a glob, such as `refs/tags/*`, as every ref it matches); with no refspec every ref when `remote.<r>.mirror` is true, else `remote.<r>.push`, else `push.default` (`matching`: each local branch the remote has; `nothing`: none; otherwise HEAD); every local branch for `--all` or `--branches`; every tag for `--tags` or `--follow-tags`; nothing for a deletion. Scanned per commit in `git rev-list <refs> --not --remotes=<r>` (all reachable commits for a URL remote or one not fetched yet): its added lines (a merge: the lines it adds against every parent), added file names, message, and author and committer emails; and each annotated tag sent, its message and tagger email. A hit passes when its home or host segment is in `git.push.privacyAllow` (case-insensitive; default `["user"]`), or its whole home segment is a Windows system profile (`Public`, `Default`, `Default User`, `All Users`, any case) or exactly `...`, or its email is the repository's `user.email`, `noreply@anthropic.com`, `*@noreply.anthropic.com`, at an RFC 2606 reserved domain, has the user part `git`, `x-access-token` or `oauth2`, or is URL userinfo after `http://`, `https://`, `ssh://` or `git://`. A ref, range or repository the guard cannot compute denies with its own sentence | `personal data in the outgoing commits` |
| 4 | `commit`, `merge`, `pull`, `cherry-pick`, `revert`, `am` or `rebase` (not with `--abort` or `--quit`) from a subagent not allowed by `git.commit.subagents`, or on a current branch matching `git.commit.protectedBranches` | `commit not permitted for this caller` |
| 5 | `reset` with `--hard` or `--merge` | `destructive reset` |
| 6 | `clean` without `-n` or `--dry-run` | `git clean deletes untracked files` |
| 7 | `checkout` whose pathspec (words after `--`, else the last argument) covers the whole tree: `.`, `./.`, `*`, `:/`, `:(top)`, or a path that `..` walks back to the start (`src/..`, `..`) | `whole-tree checkout discards edits` |
| 8 | `restore` with one of those pathspecs (after `--`, else any non-dash argument), unless `--staged`/`-S` without `--worktree`/`-W` | `whole-tree restore discards edits` |
| 9 | `branch` with `-D`, or a delete flag and a force flag, unless the branches pass the rule-2 test | `forced branch delete` |
| 10 | `rebase` with `-i`, `--interactive` or `--interactive=...` | `interactive rebase is unsupported here` |
| 11 | `stash` whose first positional is `drop` or `clear` | `stash drop/clear is unrecoverable` |
| 12 | `filter-branch`, `filter-repo`, `replace`, `update-ref`, or `reflog expire`/`delete` | `history rewrite` |
| 13 | `--no-verify` on `commit`, `push`, `merge`, `pull`, `rebase`, `am`, `cherry-pick` or `revert`; `commit -n` (`push -n` is `--dry-run` and allowed) | `hooks bypass` |
| 14 | any `GIT_CONFIG`, `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_SYSTEM` or `GIT_CONFIG_PARAMETERS` assignment in the line (prefix, `export` or `$env:`); `core.hooksPath`, `include.*` or `includeIf.*.path` set by `-c`, `--config-env` or `GIT_CONFIG_KEY_n`; `git config` writing `include.path`/`includeIf.*.path`, or unsetting `core.hooksPath` or setting it to anything but `git.hooksPath` (null: any value). A read allows. | `hooks path override` |
| 15 | `alias.*` set by `-c`, `--config-env` or `GIT_CONFIG_KEY_n`, or `git config alias.<x> <value>` | `git alias definition` |
| 16 | `checkout` with `-f`/`--force`; `switch` with `-f`, `--force` or `--discard-changes` | `forced checkout discards edits` |
| 17 | `worktree remove` with `-f` or `--force` | `forced worktree removal` |
| 18 | `checkout -B <b>`, `switch -C`/`--force-create <b>`, or `branch -f <b>` (not deleting, moving or copying) when `<b>` is the current branch or matches `git.commit.protectedBranches` | `forced branch reset` |
| 19 | an invocation with an unknown subcommand; or an opaque invocation of `push`, `send-pack`, `reset`, `checkout`, `switch`, `restore`, `clean`, `branch`, `worktree`, `stash`, `config`, `reflog` or a rule-4 subcommand. Other subcommands (`log`, `diff`, `add`, ...) stay allowed. | `unexpanded argument` |

The merged-PR check runs `gh pr list --head <branch> --state merged --json number --limit 1` in the
invocation's directory, once per branch; any failure counts as not merged. Test seam:
`GIT_GUARD_MERGED`, comma-separated branch names, replaces the call.

Rule 20 followed the 2026-10-17 catch: pusher check 5a stopped a push that carried the PC user name
in two doc paths, and the owner asked that the check never be skipped; until then a main-session
push ran no checks at all. Decision: the mechanical half lives in the guard, so every push runs it;
the pusher keeps the human read (names of people, numbers, addresses). The first version scanned
the net diff against the upstream and the tip tree, so a commit that added a name and a later one
that removed it both went out allowed, which is how this repository's history came to carry the
PC user name that same day; an independent review found it. The rule scans per commit for that
reason, against what the remote lacks rather than the upstream, and its advice is to amend or
squash, since a later commit on top leaves the name in history. The expression is check 5a's,
written in JS (PCRE's `(?-i:...)` has no JS form) and fed by read-only git (`log -z`, `log -p --cc
-U0`, `for-each-ref`, `rev-parse --verify`, `config`), each with a 20-second timeout. A segment
starting with `<`, `$`, `%` or `~` is a placeholder, never a name, and one starting with a
character no user name starts with (a quote, a bracket or brace, `: ; , | = + * ? >`) is code or
prose, such as a regular expression's `[a-z]:/Users/([^/]+)`. Reserved-domain emails and the
host marker's word-character edge keep the base's own harness and the pusher's prose clean.

Deny output, exit 0. The advice is the rule's own sentence, or, for rules 1 to 4, 9, 14 and 18, a
sentence built from the policy (which agents may act, which branches); rule 19's says the guard
cannot see what the expansion becomes; rule 20's gives the hit count and the first three hits
(`<sha> file:line`, `<sha> message`, `<sha> file name <path>`, `<sha> author`, `<sha> committer`
or `tag <t> message`, each with the matched text), then ``The repo is public: write `~`, a role or a
placeholder instead. The fix is to amend or squash the unpushed commit, never a later commit on
top, and never a change to this rule.``:

```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny",
 "permissionDecisionReason":"git-guard: <reason prefix>: `<command text>`. <advice>[ <note>]"}}
```

The command text is the simple command's source, cut at 200 characters. The note is appended when
the invocation is not first: `git-guard reads the command line, not what runs: it counted this git
word as a command because it is unquoted or sits in text handed to a shell. If nothing here runs
git, tell the user it is a false positive.` Decision: the note asks for a report rather than a
rewording, so a deny message never teaches the reader how to get past the guard.

Failure: any exception, unparseable stdin (empty included) or a missing module writes `git-guard:
internal error (<message>); command not run. Fix the hook or ask the user to run it.` to stderr and
exits 2. Decision: exit 2 blocks on every Claude Code version; `onFailure: "block"` stays as the
second layer for a crash before the handler is installed or a missing `node`.

Not covered, by design: aliases already in the user's git config; scripts that run git; another
interpreter running git (`node -e "...execSync('git push -f')"`, `python -c "os.system(...)"`),
whose code is text to the guard; `gh pr merge --delete-branch`; and `branch -f` onto a branch that
is neither current nor protected (the reflog keeps the old commit).

Test groups: G01 to G25 and GX01 to GX25 are the original table (G24 now allows: quoted text is
data); GN covers rules 12 to 17 and lease-as-force; GQ the parser in both shells; GR the shapes an
independent review reproduced as bypasses on 2026-10-13 (abbreviated options, opaque arguments,
PowerShell launchers, commit-making subcommands, config by environment and include, forced branch
resets, `send-pack`, whole-tree spellings), with the false positives they must not cause; GF
malformed and absent policy files; GV rule 20 on scratch repositories with a bare `origin` (one commit
pushed, the rest outgoing), with sample data built from pieces so the suite's own lines never match the scan; GP one command set under the Hatchery policy, no policy, and
the two examples, in scratch repositories on `main` and on `task/T1`, which shows the absent file
behaving as the
Hatchery one. LIVE01 to LIVE04 check that the repository's own `policy.json` and the examples
load and that the harness's Hatchery fixture equals the loader's defaults. Group F is the reference
fitness check of ADR 0013 (the dependency budget): F1 runs it on this repository and F2 to F6 run it
on scratch trees (a forbidden require, a skip entry present and absent, a vendored file with and
without its licence line). It counts a `require('<m>')` literal wherever it appears in a scanned
file, comments and strings included, by design.

Apart from F1, every case runs in a scratch tree: the Hatchery policy is a fixture inside the
harness, written there, and the docs it checks (a 41-line handoff included) are synthetic. F1
reads `.claude/hooks/fitness.json` and the files it names, and the harness needs
`.claude/statusline.js` beside the hooks. A project that copies `.claude/hooks/` and sets its own
policy and fitness config runs it unchanged.

Debug interface: `node .claude/hooks/test-hooks.js` runs every case for all hooks, printing a
`PASS`, `FAIL` or `SKIP` line per case plus a summary (a skip is not a failure); exit 1 on any
failure, 2 on a usage error. Flags: `--only <id-prefix>` (`--only deps` runs the reference check
alone, F1),
`--verbose`, `--keep`, `--input <file> <hook>` (reproduce one input from a file; `statusline` is
accepted as a hook name), and with `usage-gate` or `statusline` `--home <dir>` (HOME and
USERPROFILE set to it, `CLAUDE_CONFIG_DIR` removed) and `--config-dir <dir>` (`CLAUDE_CONFIG_DIR`
set to it).
`node .claude/hooks/policy.js [file]` prints the effective policy or the error;
`node .claude/hooks/policy.js --cap-check <listfile>` is pusher check 7's per-file cap line (one
`FAIL <path>: N lines, cap C` per file over its cap, a `FAIL policy:` line for a policy that does
not load, exit 1 on any `FAIL`; stdout always ends with `cap-check: <N> files, <M> over cap`, and a run without that line is a FAIL). Group CAP runs it on a scratch tree and the pusher's line through
the guard.

## 2. `post-edit.js` (PostToolUse, matcher `Write|Edit`, no `onFailure`)

Input: `tool_input.file_path` (non-empty string, else nothing happens), `cwd`. Decision: the project
root is the input `cwd`, falling back to `process.cwd()`; `CLAUDE_PROJECT_DIR` is not used, because
`cwd` is what a relative `file_path` resolves against and the harness drives a scratch tree through
it. "Inside" means the path relative to the root does not start with `..`. The docs checks run only
for files inside, under `docs/`, ending in `.md`; the root `CLAUDE.md` has its own cap check. The
formatter runs for any file whose lower-cased extension has one. Never exits non-zero; any
exception is swallowed. On Windows the `cwd` must be a Windows path (`C:/...`); a Git Bash
`/c/...` path resolves to a missing file and the hook stays silent. A malformed policy means the
base defaults here, since the hook only advises.

`CLAUDE.md` at the root over 150 lines: `post-edit: CLAUDE.md is N lines; cap is 150
(.claude/rules/docs.md). Move facts to docs/reference/ and path-scoped rules to .claude/rules/.`
An `@` line of the root `CLAUDE.md` whose target is absent: the import message below.

Checks on `docs/**/*.md` (`<path>` from the root with forward slashes; line count as `wc -l`):

| Check | Message (one line each, joined with newlines) |
|---|---|
| over 300 lines, except under `docs/design/proposals/`, paths matching `edit.capExempt`, and the two handoff paths of the next row (`docCap` in `policy.js`, shared with pusher check 7) | `post-edit: <path> is N lines; cap is 300 (.claude/rules/docs.md). Split by topic; the pusher will refuse to push.` |
| `docs/HANDOFF.md` or `docs/handoff/*.md` over 40 lines, `edit.capExempt` notwithstanding | `post-edit: <path> is N lines; cap is 40. Move detail to docs/ideas/ or a finding.` |
| when the edited file is either, `docs/HANDOFF.md` plus every `docs/handoff/*.md` on disk (`.md` only) over 120 lines in total | `post-edit: the handoff loads N lines in total (docs/HANDOFF.md plus docs/handoff/*.md); cap is 120. Close a stream or trim.` |
| a `docs/handoff/*.md` with no line `@handoff/<name>` in `docs/HANDOFF.md` (this replaces the README row for that directory, which has no README) | `post-edit: <path> is not imported by docs/HANDOFF.md. Add the line @handoff/<name> or delete the file.` |
| a line starting `@<target>`, outside fenced blocks, in `CLAUDE.md`, `docs/HANDOFF.md` or `docs/handoff/*.md`, whose target does not exist when resolved from the file's directory | `post-edit: <path> imports @<target>, which resolves to <resolved> and does not exist (an import resolves from the importing file's directory).` |
| `docs/findings/*.md` (not README) missing frontmatter `date:`, `status:` or `area:` | `post-edit: <path> lacks finding frontmatter (date, status, area). See the finding skill.` |
| `docs/reference/*.md` (not README) missing `verified:` | `post-edit: <path> lacks verified: frontmatter. See .claude/rules/docs.md.` |
| a file at least one directory below `docs/` (not `docs/handoff/*.md`, see the import row above), basename not `README.md` or `HANDOFF.md`, named in no `README.md` from its own folder up to `docs/<top>/README.md` | `post-edit: <path> is not listed in <folder>/README.md. Add one line.` |
| a backtick-quoted token holding `/` whose first segment is a top-level directory of the root (not `.git`), that does not exist | `post-edit: <path> names paths that do not exist: <list>. Fix them, or mark the paragraph [no-check: <reason>] (.claude/rules/docs.md).` |

Path anchors skip files matching `edit.anchorExempt` (by default the closed records under
`docs/design/proposals/`, `docs/design/decisions/` and `docs/findings/`, which name paths as they
were), fenced code blocks (indented ones too), any paragraph (blank-line separated) or list item
containing `[no-check:` (a list item scopes it even without blank lines between items), and tokens
with `*`, `<`, `>`, `{`, `}`, `$`, `NNNN` or `YYYY`; a `#anchor` or `:line` suffix is dropped
before the lookup. Each missing path is listed once.

Frontmatter means the file starts with `---`, closes with `---` within 20 lines, and each named key
starts a line between them. Lines split on LF or CRLF.

Formatters: `edit.formatters` in the policy, keyed by lower-case extension with the dot, value an
argv array; the file's absolute path is appended. Test seam: a JSON object in
`POST_EDIT_FORMATTERS_JSON` is merged over them. Run as `spawnSync(argv[0], [...argv.slice(1), abs],
{ timeout: 20000 })`, no shell, so on Windows an npm `.cmd` shim cannot be the command. Failure is a
spawn error or a non-zero status: `post-edit: formatter <argv[0]> failed on <path>: <detail>`, the
first 5 non-empty stderr lines, else the spawn error, else `killed by <signal> (timeout 20 s?)`.

Output with at least one message, exit 0:
`{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"<messages>"}}`; none:
exit 0, no stdout. Harness cases P01 to P04 and PX01 to PX14 cover the checks above, the
formatter and the `CLAUDE.md` cap; PA the path anchors, closed records and list-item scope; PC cap
exemptions and formatters from a policy file, a malformed policy, and the task-branches example;
PH the handoff paths (ADR 0014): the stream and total caps, the map import, and the `@` line
resolution, each case in its own scratch root (`ph-<id>`) so P02 keeps its tree.

## 3. `usage-gate.js` and the status line

Moved on 2026-10-16 to `0002-hook-specs-usage.md`, still section 3: the per-session cache, the
account label, the status line's text and flags, the gate's thresholds, staleness, grants and pool
advice, and the harness cases U, UO, UG, UX, UP and S. Other records that cite "section 3" of the
hook specs mean that file.
