---
date: 2026-10-14
status: resolved
area: environment
promoted_to: docs/reference/claude-code-on-windows.md
---

# A global `location=global` in `.npmrc` makes a plain `npm install` destructive

## Question

How does a project install its dependencies on the owner's Windows PC without touching the global
npm folder?

## What was tried

Observed 2026-10-14. `~/.npmrc` contains `location=global`.

1. A coder ran a plain `npm install` in `spike/` (a new package with `typescript` and
   `@types/node` as dependencies). npm linked the package globally as
   `AppData\Roaming\npm\node_modules\broodforge-spike` and created no `spike/node_modules`.
2. `npm install --location=project` and `npm_config_location=project npm install` both failed
   with "global requires add, rm, or update option".
3. `npm install --no-global` treated the global prefix `~/AppData/Roaming/npm` as the
   project. It wrote a `package.json` and `package-lock.json` there listing only
   `broodforge-spike`, and removed 296 packages ("added 1 package, and removed 296 packages").
   Log: `~/AppData/Local/npm-cache/_logs/2026-10-15T01_36_50_698Z-debug-0.log`.

## Result

The `.npmrc` setting is the cause: with `location=global`, a plain `npm install` is a global
install, and `--no-global` makes npm adopt the global prefix as the project and prune everything
not in the new `package.json`. Neither `--location=project` nor the environment variable
overrides it.

Removed (versions from the npm cache index, `_cacache/index-v5`; the tarballs are still cached):
`@anthropic-ai/claude-code` 2.1.296, `npm` 11.12.1, `discord.js` 14.15.3, `docx` 9.8.1,
`dotenv` 16.4.5 or 16.5.0, `node-cron` 4.1.0, `puppeteer` 24.10.0, `types-registry`, and two
links, `bot-folder` and `project`, to local folders whose paths are not in the log.

Effects:

- `claude --version` failed ("No such file or directory" for `claude-code\bin\claude.exe`).
- The running session survived because its binary was already loaded.
- One `claude.exe` survived, locked, under `node_modules\@anthropic-ai\.claude-code-KFqzmJqh\...`.
- `npm` fell back to the Node-bundled 10.9.8.

Repair, in progress by the runner: delete the stray `package.json`, `package-lock.json`,
`node_modules/.package-lock.json` and the `broodforge-spike` link; then
`npm install -g --prefer-offline` of the known packages at the versions above. The two local
links need the owner.

## Tests

Run 2026-10-14 in a throwaway package (`is-odd` 3.0.1) in the scratchpad, with the global folder
and `npm ls -g --depth=0` compared before and after.

- (a) `npm_config_userconfig=<empty file> npm --prefix <dir> install`: passed. Global folder and
  global listing identical before and after; `<dir>/node_modules` held `is-odd` and `is-number`.
- (b) A project `.npmrc` with `location=project` or `global=false`: not run, (a) passed first.
- (c) Node type stripping with no local install: not run, (a) passed first.

## Winner

Point npm at an empty user config for the call, so the owner's `location=global` is not read:

```
npm_config_userconfig=<path to an empty file> npm --prefix <dir> install
```

An agent never runs a bare `npm install`, `npm ci` or `npm install --no-global` on this PC.
