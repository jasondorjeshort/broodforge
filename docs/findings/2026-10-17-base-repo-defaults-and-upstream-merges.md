---
date: 2026-10-17
status: resolved
area: process / base-to-project updates
promoted_to: docs/design/decisions/0016-base-owned-and-project-owned-files.md
---

# Bases update cleanly when the project never edits a file the base also edits

## Question

Every update from Hatchery conflicts on `.claude/hooks/policy.json` and usually on `CLAUDE.md` (the
owner). Both mix base-owned and project-owned content: the base changed them in 5 and 17 commits
(`git log --oneline -- <path> | wc -l` at 48a43e4); a sibling changed its policy file in 3 (the
brief; not checked). How do base and template repositories ship defaults that live in git, and how
do downstreams that merge an upstream continuously keep it sustainable? The owner: "these configs
should be in a default/ directory or something". Evidence only, no design. "States" is the source,
"Inference" is mine, "Observed" is local.

## What was tried

2026-10-17: WebSearch, WebFetch; `curl` of raw sources (cruft `_commands/update.py`, copier `_main.py`,
systemd `man/systemd.unit.xml`, dpkg-source(1), brave-core, Electron, OpenWrt, ungoogled-chromium);
the GitLab API for one MR; `grep -a` of the installed Claude Code 2.1.296 binary; git 2.56.0.windows.2
in the scratchpad with blobs and trees only (`git-guard` denies this agent `commit`, so no `git merge`).
Failed: public-inbox and lore.kernel.org (502, 403), so two git-list mails found by search went unread;
freedesktop.org (403; the XML source was read). Observed: `node .claude/hooks/policy.js` with and
without `HOOK_POLICY_FILE` at a missing file differ only in `usage.pool`.

### 1. Layered configuration: defaults the downstream never edits, plus an overlay

- Debian Policy 10.7.3 (debian.org/doc/debian-policy/ch-files.html): "local changes must be preserved
  during a package upgrade"; scripts "must not alter a `conffile` of _any_ package". Debian Handbook
  5.2.3 (debian-handbook.info/browse/stable/sect.package-meta-information.html): admin untouched, dpkg
  "automatically installs the new version"; package file unchanged, "it does nothing"; both, it "asks".
  dpkg(1) (man7.org): `--force-confold`/`confnew`/`confdef`, `.dpkg-dist`. Hertzog, 2010-09-21
  (raphaelhertzog.com): "a checksum of the last installed version". Chapter 10 offers no `.d` override.
- FreedomBox RFC (salsa.debian.org/freedombox-team/plinth/issues/1483, undated): add files beside the
  originals so they "are not seen as modifications ... by dpkg"; cost: after structural changes the
  "upgrade will succeed but wrong configuration will cause the service startup to fail".
- systemd.unit(5) (github.com/systemd/systemd/blob/main/man/systemd.unit.xml): `foo.service.d/*.conf`
  changes a unit "without having to modify unit files"; `/etc` over `/run` over `/usr/lib`. A full copy:
  vendor "improvements ... are not automatically incorporated"; a drop-in: "some future updates by the
  vendor might be incompatible with the local changes".
- Docker Compose (docs.docker.com/compose/how-tos/multiple-compose-files/merge/): `compose.yaml` plus an
  optional `compose.override.yaml`; scalars replaced, `ports` concatenated, `environment` by name.
- git-config (git-scm.com/docs/git-config, 2.56.0): `include.path` content is "inserted immediately";
  "last value found taking precedence". Inference: a committed config is read only once each clone
  adds the include. `.local` files are a dotfiles habit, not a git feature (not in git-config).
- Django (docs.djangoproject.com/en/stable/topics/settings/): a settings file "doesn't have to define any
  settings if it doesn't need to"; defaults in `global_settings.py`; `diffsettings` prints the delta.
  Rails (guides.rubyonrails.org/configuring.html): `config.load_defaults` loads a version's defaults
  "and all versions prior"; `app:update` writes `new_framework_defaults_X_Y.rb` to flip them singly.
- Renovate presets (docs.renovatebot.com/config-presets/): `extends`, tag-pinned (`github>org/repo#v2.0.0`),
  "make the change in one location rather than ... copy/paste it to every repository".
- Claude Code, fetched 2026-10-17 (code.claude.com/docs/en/memory, settings, plugins-reference):
  project file "`./CLAUDE.md` or `./.claude/CLAUDE.md`"; "All discovered files are concatenated into
  context rather than overriding each other"; `CLAUDE.local.md` loads last, "add to `.gitignore`";
  unscoped rules load "with the same priority as `.claude/CLAUDE.md`"; on a user and project rule
  conflict "Claude may follow either one"; imports four hops, relative to the importing file; outside
  imports and symlinked rules need approval; a committed symlink is "a plain text file" on Windows
  unless `core.symlinks`. Both project files present: not stated. Observed in the 2.1.296 binary: per
  directory the loader pushes `CLAUDE.md`, `.claude/CLAUDE.md`, `.claude/rules`, `CLAUDE.local.md`,
  so both load (read in code, not run). Settings: "Lists merge instead of overriding"; one committed
  project layer. Plugins carry skills, agents, hooks; a plugin's `CLAUDE.md` "isn't loaded as context".

### 2. Git-level tools

- gitattributes (git-scm.com/docs/gitattributes): built-in drivers `text`, `binary`, `union`, no `ours`;
  a driver is defined "in the `.git/config` file, not in the `gitattributes` file". Pro Git 8.2
  (git-scm.com/book/en/v2/Customizing-Git-Git-Attributes): `database.xml merge=ours` plus
  `git config --global merge.ours.driver true`. git-read-tree, 3-Way Merge: "stage 1 and stage 2 are the
  same and stage 3 is different; take stage 3". Inference: drivers run only where both sides changed.
- Experiment, `git apply --3way` of a base edit `b`->`UPSTREAM` onto a project edit `b`->`PROJECT`: no
  attribute, conflict, exit 1; `merge=ours` with no driver configured, the same conflict, silently;
  with `merge.ours.driver true`, "Applied patch ... cleanly", project kept; project file unedited, the
  base line lands; `merge=union`, both lines, exit 0 (Inference: broken JSON on a changed key);
  `--exclude=policy.json`, skipped. Observed: `git diff --stat b9e6b28 HEAD -- .claude CLAUDE.md
  ':(exclude).claude/hooks/policy.json'` lists 38 files, 39 without. Not tested: `git merge`; which
  `.gitattributes` governs the merge that first brings it in.
- git-apply docs: `--exclude`/`--include`, first match wins; `--3way` needs the blobs locally.
  git-rerere (git-scm.com/docs/git-rerere): for "long lived topic branches" that "resolve the same
  conflicts over and over"; resolutions pruned after 60 days; cache location and sharing not stated.
  `union` can bring a deleted line back, exit 0 (2026-10-16 layout finding, MediaWiki T200392).

### 3. Template-update tools and files the project changed

- cruft (cruft.github.io/cruft/; `update.py`): the template's diff goes through `git apply -3`, falling
  back to `--reject` and `.rej` files; `skip` globs leave files out; `cruft check` exits 1 when behind.
- copier (copier.readthedocs.io/en/stable/updating/, /configuring/; `_main.py`): renders the old version
  with recorded answers, diffs it against the project for the project's edits, renders the new version,
  reapplies the edits with `git apply --reject`, then `git merge-file`; conflicts `inline` or `rej`;
  `_skip_if_exists` files the project edited go to `git apply --exclude`; deleted files stay deleted;
  "Never update `.copier-answers.yml` manually". gitlab-com/gl-infra/common-template-copier MR 30,
  2022-06-30, "for smoother updates": `.gitignore`, `.tool-versions`, `go.mod`, `go.sum`, `main.go`,
  `README.md` made skip-if-exists; its `renovate.json` extends shared presets.
- No update path: cookiecutter (cruft exists for it, pypi.org/project/cruft); GitHub templates ("starts
  with a single commit", "unrelated histories", docs.github.com); `nix flake init` ("It will not
  overwrite existing files", nix.dev 2.28); degit. Rails `app:update` and Yeoman (yeoman.io/authoring/
  file-system.html) prompt per overwritten file. Django `startproject`, Yeoman's options: not checked.

### 4. Long-lived downstreams that keep merging

- Debian 3.0 (quilt), dpkg-source(1): upstream tarball untouched, `debian/` apart, patches from
  `debian/patches/series` "expected to apply without any fuzz". gbp pq (honk.sigxcpu.org, gbp.patches):
  forward-port by rebase, upstreamed patches "detected automatically", cost "the lack of history".
  git-debrebase(1) (manpages.debian.org): "Rebases the delta queue onto a new upstream version".
- Brave (github.com/brave/brave-core/blob/master/docs/patching_and_chromium_src.md,
  best-practices/patches.md): aim "easy for Chromium rebases"; own code, then `chromium_src` overrides
  picked at compile time, subclassing, patches last; one-line patches with a `#define` or `include` so
  lists grow "without modifying the patch"; "electron went wrong ... copied entire files ... do NOT do this".
- Electron (github.com/electron/electron/blob/main/docs/development/patches.md): "Every patch in Electron
  is a maintenance burden"; a `.patches` order file "reduces conflicts related to patch ordering".
  ungoogled-chromium (docs/developing.md): quilt refresh per Chromium rebase.
- OpenWrt (target/linux/generic/PATCHES.md): `backport-*`, `pending-*`, `hack-*`, then per target;
  every patch "potentially upstreamable".
- Android: Corbet, 2018-11-15 (lwn.net/Articles/771974/): "millions of lines" out of tree "makes it hard
  to merge in new stable updates"; the common kernel down to "about 30 patches". source.android.com/docs/
  core/architecture/kernel/kernel-code: LKML first; "product-specific code must be implemented in modules".
- GitLab, Marin Jankovski, 2019-02-21 (about.gitlab.com/blog/merging-ce-and-ee-codebases/): CE into EE
  at release, then "once a day" (2016), then automated "every three hours" (2017); still "days of
  engineering time" on "busy work"; the repositories were merged. docs.gitlab.com/development/ee_features/:
  a prepended EE module "makes conflicts less likely ... because only one line is added to the CE class".
- LLVM, 2021-05-07 (lists.llvm.org/pipermail/llvm-dev/2021-May/150463.html): Sony's "automated merge
  system to keep ourselves continually updated to upstream HEAD" is "much much better"; "I do *not*
  recommend ... our changes intermixed with upstream changes". Rocha, 2024-09-22 (joaquimrocha.com/
  how-to-fork): "The longer the period between rebases, the higher the chances ..."; no data.

## Result

| Practice | Who | Evidence it cuts conflicts | Cost | Fit for Hatchery's two files, both modes |
|---|---|---|---|---|
| Defaults in code; project file holds only differences | Django, Rails `load_defaults`, Renovate presets, `policy.js` | By construction the base never edits the project file | Defaults move under the project unseen; needs a printer (`diffsettings`, `node .claude/hooks/policy.js`) | `policy.json`: in place already (absent file = defaults, missing key = base value); the base file is defaults plus `usage.pool`. Mode-independent. |
| Vendor file plus a separate overlay | dpkg conffiles and fragments, systemd drop-ins, Compose override, Claude Code local files | dpkg asks only when both sides changed; FreedomBox adds files to stop prompts; systemd documents the trade | Overlay goes stale silently on structural change; merge rules differ per tool | `CLAUDE.md`: the loader already reads `./CLAUDE.md`, `./.claude/CLAUDE.md` and unscoped rules, concatenated, no override; both modes. `policy.json`: `policy.js` reads one file. |
| Per-path rule: `merge=ours`, `skip`, `_skip_if_exists`, `git apply --exclude` | Pro Git, cruft, copier, the GitLab copier template | Experiment: clean apply, project kept; MR 30 "for smoother updates" | Driver lives in each clone's config; unset, it falls back to a conflict; base edits to that file never arrive | `policy.json`: patch mode is one flag on the skill's command; merge mode needs the attribute plus per-clone config. `CLAUDE.md`: no, the project wants the base's lines. |
| Patch stack refreshed per upstream release | Debian quilt and gbp, Electron, ungoogled-chromium, OpenWrt | Conflicts confined to small, ordered patches | Tooling, a refresh per update, lost history | Neither: a project's edits there are content, not a patch series. |
| Downstream code in its own files, one-line hook in upstream's | Brave `chromium_src`, GitLab `ee/` prepend, Android vendor modules | GitLab: "only one line is added"; ACK down to about 30 patches | Upstream must offer the hook; copied whole files go stale (Brave) | `CLAUDE.md`: an `@` line or `.claude/CLAUDE.md` is such a hook. `policy.json`: see the overlay row. |
| Frequent automated merges | GitLab 2017, Sony LLVM | "much much better" (Sony) | GitLab still spent days, then merged repos | Shrinks each conflict; a file both sides edit still conflicts. |
| `git rerere` | Long-lived topic branches | Replays an identical conflict | Local only; new hunks need new work | Low (Inference): each base edit is new text. |
| Template copied once, no update | GitHub templates, degit, `nix flake init`, cookiecutter | None: there is no second merge | Projects drift from the base | Neither. |

Inference: in every source that reports fewer conflicts, upstream and downstream stopped writing the
same file; frequent merges and rerere shrink a conflict but do not remove it. Not verified: `git merge`
with `merge=ours` (read-tree docs and the apply test only); whether the binary's loader order matches
what `/context` shows in a session; the sibling's three policy commits; rerere's cache location.

## Next

- If the coordinator wants a change: a proposal by the architect; this finding is its evidence.
- A runner-driven scratch test of `git merge` with `merge=ours`, including a `.gitattributes` added
  by the same merge; `/context` in a session with both `./CLAUDE.md` and `./.claude/CLAUDE.md`.
