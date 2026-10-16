# 0021 companion: snapshot, backup set, restore and rehearsal

Companion to `0021-storage.md`; the ADR's Decision section points here. Edit this file when the
layout, 0022's key handling or the per-host service steps change, in the same commit.

## Snapshot (`backup snapshot`)

1. `VACUUM INTO 'backup/store-<UTC timestamp>.sqlite.tmp'`, from any process; never the online
   backup API from a second process (it restarts whenever the bot writes).
2. Export `audit_event` in `seq` order to `backup/audit.jsonl.tmp`, one JSON object per line
   with every column.
3. `PRAGMA integrity_check` on the `.tmp` snapshot; on anything but `ok`, delete it, exit 1 with
   the result on stderr.
4. Rename both `.tmp` files to their final names; write the snapshot file name to
   `backup/latest`; delete snapshots older than the two newest.
5. When run by the bot, `PRAGMA wal_checkpoint(TRUNCATE)` afterwards. A second process does not
   checkpoint.

A snapshot interrupted mid-write leaves only a `.tmp` file, which the next run deletes; `latest`
always names a complete, checked snapshot.

## Verify (`backup verify [path]`)

Opens the named snapshot, default `latest`, read-only; runs `PRAGMA integrity_check`; prints the
migration id and one line per table with its row count; exit 0 on `ok`, 1 otherwise. Proves the
file is readable, not that a restore works.

## Rehearse (`backup rehearse [path]`)

Restores into a scratch instance root and deletes it on success (kept on failure, path on stderr,
like every scratch run):

1. Create the scratch root; copy `config/` and the named snapshot (default `latest`) into it as
   `data/store.sqlite`.
2. Run migrations on the copy; refuse per the schema rule if the snapshot is newer than the
   binary.
3. Run reconciliation in report-only mode (nothing written; every missing worktree listed, since
   the scratch root has none).
4. Open the data layer, read one record of each kind in the data model, print the counts per
   table and the migration id.
5. Exit 0; the nightly job treats any other exit as a failed backup and reports it.

The rehearsal does not exercise the credential store or the key outside the backup set; those
steps are 0022's and are listed below as inputs the rehearsal cannot prove.

## The nightly job

`backup snapshot`, then `backup rehearse`, then the host's copy of the instance root plus every
checkout and worktree path the store records, to the off-host destination the owner chose. The
restore source is always the snapshot; the live `data/store.sqlite` and its `-wal` are copied
along but are never what a restore opens first.

## Backup set

| In the set | Where |
|---|---|
| Configuration | `config/` |
| Store snapshots, `latest`, audit export | `backup/` |
| Credential store, as 0022 encrypts it | `credentials/` |
| Canonical checkouts | `projects/<project id>/` |
| Session directories: shared clone, home, per-session config, bundle | `sessions/<session id>/`, the paths the store records |
| Harness config directories, minus the excluded files | `harness/<credential id>/` |
| Live store (last resort only) | `data/` |

Outside the set, by design: the credential-store key and anything else 0022 keeps off the host;
the runtime, Broodforge itself and the harness; the service definition; `settings/`, which the bot
regenerates before every engine process; the files 0022 lists as excluded: `.credentials.json` and
`.claude.json` in every harness directory and every session home, by name (spike C36 kept it:
`0022-session-isolation-and-credential-storage.md`, Amendments from the spike). The owner's own `~/.claude` on the PC is
never in the set.

## Restore from an empty host

Inputs: the backup tree above; the credential-store key from wherever 0022 keeps it; the host
setup reference for that host: `../../reference/droplet-host-setup.md` for the droplet
[no-check: written from the droplet operator entries once the owner has run them], its PC
counterpart for the PC [no-check: likewise, from the PC entry]. Until a reference exists the
operator entries under `../../operator/` are the only copy of those commands.

1. Build the host per its reference: users, groups, sudoers, packages and the AppArmor profile
   on the droplet, the account on the PC; the runtime, Broodforge, git and the harness. Log the
   harness in per 0022 if its delivery mechanism needs a login on the host.
2. Create the instance root at the path the instance config names, with the owners and modes of
   the 0022 companion's table (the reference's commands create them); copy `config/`,
   `credentials/` and `backup/` into it. Do not copy `data/`.
3. Copy the snapshot `backup/latest` names to `data/store.sqlite`. No `-wal` or `-shm` file.
4. Copy `projects/`, `harness/` and `sessions/` into the root and restore the owners of the
   companion's table (`chown -R`; on the droplet `projects/` is the bot's with the engine group,
   `harness/` and `sessions/` the engine user's). If the root's path differs from the backed-up
   host (a new user, a new drive letter), rewrite the one line in each
   `sessions/<id>/tree/.git/objects/info/alternates` to the new `projects/<project
   id>/.git/objects`, because `git worktree repair` does not touch alternates; then run
   `git worktree repair` in each session tree for its subagent worktrees, as the engine user on
   the droplet (`sudo -u broodforge-engine git -C <tree> worktree repair`; the tree is its and
   git refuses another owner). On Windows, give paths with the `\\?\` prefix where they exceed
   the length limit (`../../reference/claude-code-on-windows.md`, Git worktrees).
5. Supply the credential-store key per 0022's procedure.
6. Run `config check`; fix the errors it lists.
7. Start the bot once from a terminal: migrations run; reconciliation reports every missing
   worktree and every unrecorded directory; `status` must not show `last-good`.
8. Install and start the service: the Task Scheduler task on Windows, the systemd unit on Linux,
   both per the host's reference.
9. Run `backup snapshot` and `backup rehearse` once so the first nightly job has a baseline.

Last resort: if no snapshot is usable, copy the live `data/store.sqlite` and `-wal` from the
backup into `data/`, run `backup verify data/store.sqlite`, and continue from step 4 only on
`ok`; otherwise restore the older generation.

## Windows and Linux differences

Only the paths, the service step and the long-path prefix differ; the commands are the same on
both hosts, which is how the restore is tested: `backup rehearse` runs in the test suite on
both.
