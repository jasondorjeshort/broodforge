#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# broodforge setup against a scripted Discord (ADR 0027; 05-debug-interface-setup.md), on roots
# under a scratch root, from the repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/setup.sh
#
# The repository is the scratch root's own remote/scratch.git. The engine is a stand-in on
# PATH printing 2.1.296: on Linux a sh script; on Windows, where the engine must be an .exe,
# STUB_ENGINE names one, else the cli package's test binary is built with go (its --version
# answers when BFTEST_STANDIN is set). BF names another binary than bin/broodforge.
# Discord is the scripted fake BROODFORGE_SETUP_FAKE_DISCORD names (discord.FakeScript): an
# application whose bot joins the server on the poll after the invite, with Manage Roles, which
# the owner removes on the poll after the removal fix; the file records every write to it.
# Each setup's output is printed; every root is under the scratch root, kept on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
scratch=${BROODFORGE_ROOT//\\//}
work=$scratch/setup
repo=$scratch/remote/scratch.git
unset BROODFORGE_KEY_FILE
mkdir -p "$work/stub"

fail() {
	echo "setup.sh: $*" >&2
	exit 1
}

# show <label> <command...>: run it, print its output and exit code; the code is in $code.
show() {
	local label=$1
	shift
	echo "## $label"
	echo "\$ broodforge ${*:2}"
	set +e
	"$@" >"$work/out" 2>"$work/err"
	code=$?
	set -e
	cat "$work/out" "$work/err"
	echo "exit $code"
}

case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN*)
	if [ -n "${STUB_ENGINE:-}" ]; then
		cp "$STUB_ENGINE" "$work/stub/claude.exe"
	else
		go test -c -o "$work/stub/claude.exe" ./internal/cli
	fi
	export BFTEST_STANDIN=1
	export ProgramData
	ProgramData=$(cygpath -w "$work/programdata")
	;;
*)
	printf '#!/bin/sh\necho "2.1.296 (Claude Code)"\n' >"$work/stub/claude"
	chmod +x "$work/stub/claude"
	;;
esac
stub=$work/stub
command -v cygpath >/dev/null && stub=$(cygpath -u "$stub") # C:/... would split PATH at its colon
export PATH="$stub:$PATH"

# The scripted Discord. token is the one bot token it accepts (the bot 5000's id in base64).
token=NTAwMA.abcd.abcdefghijklmnopqrstuvwxyz
app='"application":{"id":"4000","botId":"5000"}'
roles='"ownerId":"6000","roles":[{"id":"3000","name":"@everyone","position":0,"permissions":0},{"id":"5001","name":"Broodforge","position":2,"permissions":2251800082246736}],"botRoles":["5001"]'
script() { # script <file> <json after the token>: write one world
	printf '{"token":"%s",%s}\n' "$token" "$2" >"$work/$1"
}
script discord.json "$app,\"guilds\":[],\"invite\":{\"id\":\"3000\",\"name\":\"Test\",$roles,\"dropManageRolesAfter\":1},\"joinAfter\":1"
script discord-channels.json "$app,\"guilds\":[{\"id\":\"3000\",\"name\":\"Test\",$roles,\"channels\":[{\"id\":\"700\",\"name\":\"Text Channels\",\"category\":true},{\"id\":\"701\",\"name\":\"general\",\"parent\":\"700\"}]}]"
script discord-two.json "$app,\"guilds\":[{\"id\":\"3000\",\"name\":\"Test\",$roles},{\"id\":\"3100\",\"name\":\"Other\",$roles}]"
script discord-kept.json "$app,\"guilds\":[],\"invite\":{\"id\":\"3000\",\"name\":\"Test\",$roles},\"joinAfter\":1"
export BROODFORGE_SETUP_FAKE_DISCORD=$work/discord.json

r1=$work/1/root
full=(setup --yes --root "$r1" --instance t --repo "$repo" --wait 1)

# 1. A fresh run with every flag given and the token on stdin: ok for every step but the
# deferred git credential; the invite and the Manage Roles removal print their fix first.
show "fresh run" "$bf" "${full[@]}" <<<"$token"
[ "$code" = 0 ] || fail "the fresh run exited $code"
for s in token application guild channels role; do
	grep -q "^ok  $s  " "$work/out" || fail "$s is not ok"
done
grep -qx 'fix  guild  Open this link, pick the server, Authorize: https://discord.com/oauth2/authorize?client_id=4000&scope=bot+applications.commands&permissions=2251800082246736' "$work/out" ||
	fail "no invite line with Manage Roles"
grep -qx 'waiting for the bot to join' "$work/err" || fail "no waiting line"
grep -qx "fix  role  remove Manage Roles from the bot's role in Server Settings, Roles; setup is waiting" "$work/out" || fail "no removal line"
grep -qx 'ok  channels  Sessions 9001, #scratch 9002, #scratch-notices 9003' "$work/out" || fail "the channels were not created"
grep -qx 'ok  check  discord check  guild  3000  Test' "$work/out" || fail "discord check is not echoed in check"
grep -q '^skip  git-credential  deferred' "$work/out" || fail "git-credential is not deferred"
grep -q "^ok  engine  .*/stub/claude[^/]*  2\.1\.296$" "$work/out" || fail "the engine is not the stand-in"
bad=$(grep -v -E '^(ok  |fix  (guild|role)  |skip  git-credential  )' "$work/out" || true)
[ -z "$bad" ] || fail "the fresh run printed: $bad"
grep -q "$token" "$work/out" "$work/err" "$r1/config/instance.json" "$r1/credentials/store.json" && fail "the token is in the clear"
"$bf" config check --root "$r1" >/dev/null || fail "config check fails after setup"

# 2. The rerun: only ok and skip, every Discord step skip, config/ unchanged (ADR 0027,
# prediction 1), nothing written to Discord.
cp -r "$r1/config" "$work/config-before"
writes=$(grep -c '"Create\|"Update\|"AddMember' "$work/discord.json")
show "rerun" "$bf" "${full[@]}"
[ "$code" = 0 ] || fail "the rerun exited $code"
bad=$(grep -v -E '^(ok|skip)  ' "$work/out" || true)
[ -z "$bad" ] || fail "the rerun printed: $bad"
for s in token application guild channels role; do
	grep -q "^skip  $s  " "$work/out" || fail "rerun: $s is not skip"
done
git diff --no-index --exit-code "$work/config-before" "$r1/config" || fail "the rerun changed config/"
[ "$(grep -c '"Create\|"Update\|"AddMember' "$work/discord.json")" = "$writes" ] || fail "the rerun wrote to Discord"

# 2a. The round trip (P24): r1 gains a git credential, so its export has one to leave behind;
# config export, then setup --from on a fresh root with its own token: the host-bound and
# secret steps ok, the imported ones skip, every non-secret key equal, no secret in the file,
# nothing written to Discord, the new root with its own key and store. The repository URL
# travels (here a path in the scratch root); paths of r1 do not.
"$bf" setup git-credential --yes --root "$r1" <<<"example-git-token" >/dev/null || fail "setup git-credential on r1 failed"
show "config export" "$bf" config export --root "$r1" --out "$work/e.json"
[ "$code" = 0 ] && grep -q '^written  .*/e\.json$' "$work/out" || fail "config export exited $code"
echo "## the export"
cat "$work/e.json"
[ "$(grep -c cred- "$work/e.json" || true)" = 0 ] || fail "the export holds a credential id"
grep -q -e "$token" -e example-git-token -e "$r1" -e "$work/1/t.key" -e "$(cat "$work/1/t.key")" "$work/e.json" && fail "the export holds a secret or a path of this host"
r20=$work/20/root
show "setup --from the export" "$bf" setup --from "$work/e.json" --yes --root "$r20" --instance t2 --wait 1 <<<"$token"
[ "$code" = 0 ] || fail "setup --from exited $code"
for s in root config key engine token project; do
	grep -q "^ok  $s  " "$work/out" || fail "setup --from: $s is not ok"
done
for s in application guild channels role git-credential defaults; do
	grep -q "^skip  $s  " "$work/out" || fail "setup --from: $s is not skip"
done
grep -q '^ok  config  imported config/instance.json; config/projects/scratch.json from ' "$work/out" || fail "setup --from: no import in the config line"
[ "$(grep -c '"Create\|"Update\|"AddMember' "$work/discord.json")" = "$writes" ] || fail "setup --from wrote to Discord"
"$bf" config check --root "$r20" >/dev/null || fail "config check fails after setup --from"
"$bf" config export --root "$r20" --out "$work/e2.json" >/dev/null
sed 's/"instanceName": "t2"/"instanceName": "t"/' "$work/e2.json" | diff - "$work/e.json" || fail "a non-secret key differs after the round trip"
[ -s "$work/20/t2.key" ] && ! cmp -s "$work/20/t2.key" "$work/1/t.key" || fail "the new root has no key of its own"
for id in $(grep -o 'cred-[a-z0-9]*' "$r1/config/instance.json" "$r1/config/projects/scratch.json" | cut -d: -f2); do
	grep -q "$id" "$r20/credentials/store.json" "$r20/config/instance.json" && fail "$id travelled"
done
cmp -s "$r20/data/config-export.json" "$work/e.json" || fail "no copy of the export in the new root's data/"
show "config import on a configured root" "$bf" config import --root "$r20" "$work/e.json"
[ "$code" = 1 ] && grep -qx "error: config-exists: .*/20/root/config/instance.json exists" "$work/err" || fail "no config-exists"
sed 's/"mode": "dry-run"/"mode": "dry-run", "shard": 1/' "$work/e.json" >"$work/e-bad.json"
show "setup --from an export with an unknown key" "$bf" setup --from "$work/e-bad.json" --yes --root "$work/21/root" --instance t2 --wait 1
[ "$code" = 1 ] && grep -q '^error: mismatch-config: export .*e-bad.json has instance: unknown key discord.shard; fix the file or export it again with this version$' "$work/err" ||
	fail "no mismatch-config for an unknown key"
[ ! -e "$work/21" ] || fail "the refused import created $work/21"

# 3. --check on an empty root: fix for every step, nothing created.
show "--check on an empty root" "$bf" setup --check --root "$work/2/root" --instance t
[ "$code" = 1 ] || fail "--check on an empty root exited $code"
grep -q '^error: check-failed: root config key engine token application guild channels role git-credential project defaults check next$' "$work/err" ||
	fail "--check on an empty root did not name every step"
[ ! -e "$work/2" ] || fail "--check created $work/2"

# 4. --check on a half-written config: the instance file alone.
mkdir -p "$work/3/root/config"
cp "$r1/config/instance.json" "$work/3/root/config/"
show "--check on a half-written config" "$bf" setup --check --root "$work/3/root"
[ "$code" = 1 ] || fail "--check on a half-written config exited $code"
grep -q '^fix  config  no project file' "$work/out" || fail "--check did not find the project file missing"
[ ! -e "$work/3/root/data" ] || fail "--check created data/"

# 5. --yes with no repository.
show "--yes without --repo" "$bf" setup --yes --root "$work/4/root" --instance t --wait 1
[ "$code" = 1 ] && grep -qx 'error: needs-input-repo: pass --repo' "$work/err" || fail "no needs-input-repo"

# 6. No token on stdin: the portal steps as the token's fix, then needs-input-token. Then setup
# <step> on a present key: replaced while the store holds no entry, then key-in-use.
r8=$work/8/root
show "no token on stdin" "$bf" setup --yes --root "$r8" --instance t --repo "$repo" --wait 1
[ "$code" = 1 ] && grep -qx 'error: needs-input-token: pass the bot token on stdin with --yes' "$work/err" || fail "no needs-input-token"
grep -q '^fix  token  no bot token yet: in the Developer Portal (https://discord.com/developers/applications), New Application, then Bot, Reset Token' "$work/out" ||
	fail "no portal fix line"
show "setup key on a present key" "$bf" setup key --yes --root "$r8"
[ "$code" = 0 ] && grep -q '^old  ' "$work/out" && grep -q '^ok  key  [0-9a-f]\{64\}  ' "$work/out" || fail "setup key did not replace the key"
show "setup git-credential, the token on stdin" "$bf" setup git-credential --yes --root "$r8" <<<"example-token"
[ "$code" = 0 ] && grep -q '^ok  git-credential  cred-' "$work/out" || fail "setup git-credential failed"
grep -q example-token "$r8/credentials/store.json" "$r8/config/projects/scratch.json" && fail "the token is in the clear"
show "setup key with a stored credential" "$bf" setup key --yes --root "$r8"
[ "$code" = 1 ] && grep -q '^error: key-in-use: credentials/store.json holds 1 entries' "$work/err" || fail "no key-in-use"

# 7. A key file that is not a key.
mkdir -p "$work/7"
"$bf" setup --yes --root "$work/7/root" --instance t --repo "$repo" --wait 1 <<<"$token" >/dev/null
chmod 600 "$work/7/t.key"
printf 'placeholder\n' >"$work/7/t.key"
show "a key file that is not a key" "$bf" setup --yes --root "$work/7/root"
[ "$code" = 1 ] && grep -q '^error: mismatch-key: config has key file .*, found a file that is not a key (12 bytes); run setup key to replace it$' "$work/err" ||
	fail "no mismatch-key"

# 8. A bot on the root: setup is refused running; setup --check still runs.
"$bf" serve --root "$r1" >"$work/serve.log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --root "$r1" --wait 10 --expect ok >/dev/null || fail "serve did not answer (serve.log: $work/serve.log)"
show "setup with a bot running" "$bf" setup --yes --root "$r1"
[ "$code" = 1 ] && grep -q '^error: running: a bot answers at .*; stop it with broodforge stop in this shell (every open session is interrupted) or use setup --check$' "$work/err" ||
	fail "no running refusal"
show "setup --check with a bot running" "$bf" setup --check --root "$r1"
[ "$code" = 0 ] || fail "setup --check with a bot exited $code"
"$bf" stop --root "$r1" >/dev/null
wait "$serve" || fail "serve exited $?"

# 9. A setup paused at the key holds the address: serve fails start-control, a second setup is
# setup-running, status answers config: setup. Off a terminal without --yes, stdin is the key's
# Enter alone, so the run then ends at the token, needs-input-token.
r5=$work/5/root
rm -f "$work/release"
{
	while [ ! -e "$work/release" ]; do sleep 0.2; done
	echo
} | "$bf" setup --root "$r5" --instance t --repo "$repo" --wait 1 >"$work/paused.out" 2>"$work/paused.err" &
paused=$!
for _ in $(seq 1 150); do
	grep -q '^ok  key  ' "$work/paused.out" 2>/dev/null && break
	sleep 0.2
done
grep -q '^ok  key  ' "$work/paused.out" || fail "the setup did not reach the key"
echo "## paused at the key"
cat "$work/paused.out" "$work/paused.err"
show "serve beside the paused setup" "$bf" serve --root "$r5"
[ "$code" = 1 ] && grep -q '^error: start-control: ' "$work/err" || fail "serve was not refused start-control"
show "a second setup" "$bf" setup --yes --root "$r5"
[ "$code" = 1 ] && grep -q '^error: setup-running: another setup holds .*; finish it or press Ctrl-C there$' "$work/err" ||
	fail "no setup-running"
show "status while setup holds the address" "$bf" status --root "$r5" --expect setup
[ "$code" = 0 ] || fail "status --expect setup exited $code"
touch "$work/release"
set +e
wait "$paused"
code=$?
set -e
echo "## the paused setup, finished"
cat "$work/paused.out" "$work/paused.err"
[ "$code" = 1 ] && grep -qx 'error: needs-input-token: pass the bot token on stdin with --yes' "$work/paused.err" ||
	fail "the paused setup exited $code"

# 10. --json: one object per line, the key in the key object only.
show "--json" "$bf" setup --json --yes --root "$work/6/root" --instance t --repo "$repo" --wait 1 <<<"$token"
[ "$code" = 0 ] || fail "--json exited $code"
key=$(sed -n 's/.*"step":"key".*"key":"\([0-9a-f]\{64\}\)".*/\1/p' "$work/out")
[ -n "$key" ] || fail "no key object"
[ "$(grep -c "$key" "$work/out")" = 1 ] || fail "the key is printed more than once"

# 11. A server with channels and --channels no (the --yes default): channels creates nothing and
# waits for the owner's channels by name.
show "a server with channels" env BROODFORGE_SETUP_FAKE_DISCORD="$work/discord-channels.json" \
	"$bf" setup --yes --root "$work/11/root" --instance t --repo "$repo" --wait 1 <<<"$token"
[ "$code" = 1 ] && grep -q '^error: timeout-channels: still waiting after 1s: create the category Sessions' "$work/err" || fail "no timeout-channels"
grep -q '"CreateGuildChannel' "$work/discord-channels.json" && fail "channels were created under --channels no"

# 12. The bot in two servers and no --guild: needs-input-guild.
show "two servers" env BROODFORGE_SETUP_FAKE_DISCORD="$work/discord-two.json" \
	"$bf" setup --yes --root "$work/12/root" --instance t --repo "$repo" --wait 1 <<<"$token"
[ "$code" = 1 ] && grep -qx 'error: needs-input-guild: pass --guild' "$work/err" || fail "no needs-input-guild"

# 13. The bot keeps Manage Roles: the removal fix, then timeout-role, on this run and the next.
for n in 1 2; do
	show "Manage Roles kept, run $n" env BROODFORGE_SETUP_FAKE_DISCORD="$work/discord-kept.json" \
		"$bf" setup --yes --root "$work/13/root" --instance t --repo "$repo" --wait 1 <<<"$token"
	[ "$code" = 1 ] && grep -q '^error: timeout-role: ' "$work/err" || fail "run $n: no timeout-role"
	grep -qx "fix  role  remove Manage Roles from the bot's role in Server Settings, Roles; setup is waiting" "$work/out" || fail "run $n: no removal fix"
done

# 14. A configured channel id the server lacks: mismatch-channels.
sed -i 's/"noticeChannelId": "9003"/"noticeChannelId": "1234"/' "$r1/config/instance.json"
show "a configured channel gone" "$bf" "${full[@]}"
[ "$code" = 1 ] && grep -qx 'error: mismatch-channels: config has channel 1234 (#scratch-notices), found no such channel in guild 3000; run setup channels to replace it' "$work/err" ||
	fail "no mismatch-channels"

# 15. A live root refuses the scripted Discord.
sed -i 's/"mode": "dry-run"/"mode": "live"/' "$r1/config/instance.json"
show "the scripted Discord on a live root" "$bf" "${full[@]}"
[ "$code" = 1 ] && grep -q '^error: live-instance: ' "$work/err" || fail "no live-instance"

trap - EXIT
echo "setup.sh: ok"
