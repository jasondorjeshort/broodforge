#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# The development instance's loop on a temporary root (05-debug-interface-dev-instance.md,
# Reproduce; the dry-run form of decisions/0027-setup-commands.md), from the repository root:
#
#   bash tests/repro/dev-instance.sh
#
# A dry-run root under the OS temp directory with a generated instance name, this checkout as
# the repository (as the run skill's setup line has it): `setup --discord dry-run`, the round
# trip of the run skill's step 3 on channel 100 (the fake), `stop`; the rerun (only ok and skip,
# config/ unchanged); then a `--discord live` run against the scripted Discord
# (BROODFORGE_SETUP_FAKE_DISCORD) replacing the placeholders 100, 200 and 10 with no mismatch
# and no question, and `--discord dry-run` refused live-instance once the root reads live. The
# root is removed on success and kept, with its path printed, on a failure. No real engine turn:
# the engine is a stand-in printing 2.1.296, as in tests/repro/setup.sh (on Windows STUB_ENGINE
# names one, else the cli package's test binary is built with go), and ProgramData points into
# the temporary directory, so the pinned engine is never the machine's. The binary is this
# checkout's bin/broodforge: the dry-run form finds fixtures/engine beside its bin/.
set -euo pipefail

bf=bin/broodforge
unset BROODFORGE_KEY_FILE BROODFORGE_SETUP_FAKE_DISCORD
tmp=$(mktemp -d)
case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN*) win=1 ;;
*) win= ;;
esac
if [ -n "$win" ]; then
	base=$(cygpath -m "$tmp")
	repo=$(pwd -W)
else
	base=$tmp
	repo=$(pwd)
fi
root=$base/root
instance=dev-$(od -An -tx1 -N6 /dev/urandom | tr -d ' \n')
export BROODFORGE_ROOT=$root
serve=
passed=

fail() {
	echo "dev-instance.sh: $*" >&2
	exit 1
}
# Any exit before the end, a failed assertion or a command set -e stopped on: the bot killed,
# the root kept and named.
leave() {
	[ -n "$serve" ] && kill "$serve" 2>/dev/null
	[ -n "$passed" ] || echo "dev-instance.sh: the root is kept at $root" >&2
	true
}
trap leave EXIT

# show <label> <command...>: run it, print its output (never the key line, nor spawn preflight's
# account line, which names the machine's user) and exit code; the code is in $code, stdout in
# $tmp/out, stderr in $tmp/err.
show() {
	local label=$1
	shift
	echo "## $label"
	set +e
	"$@" >"$tmp/out" 2>"$tmp/err"
	code=$?
	set -e
	grep -v -e '^ok  key  ' -e '^ok  check  spawn preflight  ok  account  ' "$tmp/out" || true
	cat "$tmp/err"
	echo "exit $code"
}

mkdir -p "$tmp/stub"
if [ -n "$win" ]; then
	if [ -n "${STUB_ENGINE:-}" ]; then
		cp "$STUB_ENGINE" "$tmp/stub/claude.exe"
	else
		go test -c -o "$tmp/stub/claude.exe" ./internal/cli
	fi
	export BFTEST_STANDIN=1
	export ProgramData
	ProgramData=$(cygpath -w "$tmp/programdata")
else
	printf '#!/bin/sh\necho "2.1.296 (Claude Code)"\n' >"$tmp/stub/claude"
	chmod +x "$tmp/stub/claude"
fi
export PATH="$tmp/stub:$PATH"

line=(setup --discord dry-run --yes --root "$root" --instance "$instance" --project broodforge --repo "$repo" --wait 1)

# 1. The dry-run form: no Discord call, the placeholders and the fixtures written.
show "setup --discord dry-run" "$bf" "${line[@]}"
[ "$code" = 0 ] || fail "setup --discord dry-run exited $code"
for s in token application guild; do
	grep -qx "skip  $s  dry-run" "$tmp/out" || fail "no skip  $s  dry-run"
done
grep -qx "ok  channels  dry-run 100 fake, 200 claude-code; fixtures $repo/fixtures/engine" "$tmp/out" || fail "no dry-run channels line"
grep -qx 'ok  role  dry-run 10' "$tmp/out" || fail "no dry-run role line"
grep -qx 'ok  next  the dry-run instance is ready: broodforge serve --discord dry-run starts it; a rerun with --discord live adds Discord' "$tmp/out" ||
	fail "no dry-run closing line"
grep -q 'discord check' "$tmp/out" && fail "a discord check line under dry-run"

# 2. The bot, then the round trip on channel 100 (the run skill's step 3).
log=$base/serve-test.log
"$bf" serve --discord dry-run >"$log" 2>&1 &
serve=$!
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (log $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null
cred=$(printf scratch | "$bf" credential add --kind api-key --user 1001 --label fake)
"$bf" register set-default 1001 "$cred" >/dev/null
show "inject click" "$bf" discord inject click --custom-id bf:100:start --user 1001 --roles 10
[ "$code" = 0 ] || fail "inject click exited $code"
id=$("$bf" session list | awk '{print $1; exit}')
[ "$id" = s0001 ] || fail "session list names $id, want s0001"
show "inject message" "$bf" discord inject message --channel "$id" --user 1001 --roles 10 "Read hello.txt and reply with its one word."
[ "$code" = 0 ] || fail "inject message exited $code"
for _ in $(seq 1 80); do
	"$bf" session tail "$id" --all | tail -n 1 | grep -q '  turn-ended  ' && break
	sleep 0.25
done
show "session tail" "$bf" session tail "$id" --all
tail -n 1 "$tmp/out" | grep -q '  turn-ended  end_turn$' || fail "the tail's last line is not turn-ended  end_turn"
show "session outbox" "$bf" session outbox "$id"
grep -q "  channel  $id  0  #$id-ada\$" "$tmp/out" || fail "the outbox has no channel line"
grep -q "  create  m[0-9]*  [0-9]*  Session $id on broodforge, branch broodforge/$instance/$id" "$tmp/out" || fail "the outbox has no header create"
show "session end" "$bf" session end "$id"
[ "$code" = 0 ] || fail "session end exited $code"
show "stop" "$bf" stop
[ "$code" = 0 ] && grep -q '^stopped  ' "$tmp/out" || fail "stop exited $code"
wait "$serve" || fail "serve exited $?"
serve=
show "status after stop" "$bf" status
[ "$code" = 1 ] && grep -q '^error: not-running: ' "$tmp/err" || fail "status after stop exited $code"

# 3. The rerun: only ok and skip, config/ unchanged.
cp -r "$root/config" "$tmp/config-before"
show "rerun" "$bf" "${line[@]}"
[ "$code" = 0 ] || fail "the rerun exited $code"
bad=$(grep -v -E '^(ok|skip)  ' "$tmp/out" || true)
[ -z "$bad" ] || fail "the rerun printed: $bad"
diff -r "$tmp/config-before" "$root/config" || fail "the rerun changed config/"

# 4. A --discord live run against the scripted Discord: the placeholders count as no id,
# replaced without a question, no old or new record, no mismatch.
token=NTAwMA.abcd.abcdefghijklmnopqrstuvwxyz
roles='"ownerId":"6000","roles":[{"id":"3000","name":"@everyone","position":0,"permissions":0},{"id":"5001","name":"Broodforge","position":2,"permissions":2251800082246736}],"botRoles":["5001"]'
printf '{"token":"%s","application":{"id":"4000","botId":"5000"},"guilds":[],"invite":{"id":"3000","name":"Test",%s,"dropManageRolesAfter":1},"joinAfter":1}\n' \
	"$token" "$roles" >"$tmp/discord.json"
show "setup --discord live, the scripted Discord" env BROODFORGE_SETUP_FAKE_DISCORD="$tmp/discord.json" \
	"$bf" setup --discord live --yes --root "$root" --wait 5 <<<"$token"
[ "$code" = 0 ] || fail "the live run exited $code"
grep -q -e mismatch -e '^old  ' -e '^new  ' "$tmp/out" "$tmp/err" && fail "the live run printed a mismatch or a replacement"
grep -qx 'ok  channels  Sessions 9001, #broodforge 9002, #broodforge-notices 9003' "$tmp/out" || fail "the channels were not made"
grep -qx 'ok  role  Trusted  9004  held by 6000; discord.mode stays dry-run (scripted Discord)' "$tmp/out" || fail "the role was not written"
grep -qE '"(100|200)": \{' "$root/config/projects/broodforge.json" && fail "a placeholder channel is kept"
grep -q '"10"' "$root/config/instance.json" && fail "the placeholder role is kept"
grep -q '"9002": {' "$root/config/projects/broodforge.json" || fail "the lobby is not configured"

# 5. On a root that reads live, --discord dry-run is refused before anything is written.
sed -i 's/"mode": "dry-run"/"mode": "live"/' "$root/config/instance.json"
rm -rf "$tmp/config-before"
cp -r "$root/config" "$tmp/config-before"
show "setup --discord dry-run on a live root" "$bf" "${line[@]}"
[ "$code" = 1 ] && grep -qx 'error: live-instance: --discord dry-run is refused on a live instance' "$tmp/err" || fail "no live-instance"
[ ! -s "$tmp/out" ] || fail "the refused run printed step lines"
diff -r "$tmp/config-before" "$root/config" || fail "the refused run changed config/"

passed=1
rm -rf "$tmp"
echo "dev-instance.sh: ok"
