#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# broodforge setup without Discord (ADR 0027; 05-debug-interface-setup.md), on roots under a
# scratch root, from the repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/setup.sh
#
# The repository is the scratch root's own remote/scratch.git. The engine is a stand-in on
# PATH printing 2.1.296: on Linux a sh script; on Windows, where the engine must be an .exe,
# STUB_ENGINE names one, else the cli package's test binary is built with go (its --version
# answers when BFTEST_STANDIN is set). BF names another binary than bin/broodforge.
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

r1=$work/1/root
full=(setup --yes --root "$r1" --instance t --repo "$repo" --wait 1)

# 1. A fresh run with every flag given: ok for every step but the Discord ones and the
# deferred git credential.
show "fresh run" "$bf" "${full[@]}"
[ "$code" = 0 ] || fail "the fresh run exited $code"
for s in token application guild channels role; do
	grep -qx "skip  $s  not built" "$work/out" || fail "$s is not skip  not built"
done
grep -q '^skip  git-credential  deferred' "$work/out" || fail "git-credential is not deferred"
grep -q "^ok  engine  .*/stub/claude[^/]*  2\.1\.296$" "$work/out" || fail "the engine is not the stand-in"
bad=$(grep -v -E '^(ok  |skip  (token|application|guild|channels|role|git-credential)  )' "$work/out" || true)
[ -z "$bad" ] || fail "the fresh run printed: $bad"
"$bf" config check --root "$r1" >/dev/null || fail "config check fails after setup"

# 2. The rerun: only ok and skip, and config/ unchanged (ADR 0027, prediction 1).
cp -r "$r1/config" "$work/config-before"
show "rerun" "$bf" "${full[@]}"
[ "$code" = 0 ] || fail "the rerun exited $code"
bad=$(grep -v -E '^(ok|skip)  ' "$work/out" || true)
[ -z "$bad" ] || fail "the rerun printed: $bad"
git diff --no-index --exit-code "$work/config-before" "$r1/config" || fail "the rerun changed config/"

# 3. --check on an empty root: fix for every step but the Discord ones, nothing created.
show "--check on an empty root" "$bf" setup --check --root "$work/2/root" --instance t
[ "$code" = 1 ] || fail "--check on an empty root exited $code"
grep -q '^error: check-failed: root config key engine git-credential project defaults check next$' "$work/err" ||
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

# 6. setup <step> on a present key: replaced while the store holds no entry, then key-in-use.
show "setup key on a present key" "$bf" setup key --yes --root "$r1"
[ "$code" = 0 ] && grep -q '^old  ' "$work/out" && grep -q '^ok  key  [0-9a-f]\{64\}  ' "$work/out" || fail "setup key did not replace the key"
show "setup git-credential, the token on stdin" "$bf" setup git-credential --yes --root "$r1" <<<"example-token"
[ "$code" = 0 ] && grep -q '^ok  git-credential  cred-' "$work/out" || fail "setup git-credential failed"
grep -q example-token "$r1/credentials/store.json" "$r1/config/projects/scratch.json" && fail "the token is in the clear"
show "setup key with a stored credential" "$bf" setup key --yes --root "$r1"
[ "$code" = 1 ] && grep -q '^error: key-in-use: credentials/store.json holds 1 entries' "$work/err" || fail "no key-in-use"

# 7. A key file that is not a key.
mkdir -p "$work/7"
"$bf" setup --yes --root "$work/7/root" --instance t --repo "$repo" --wait 1 >/dev/null
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
# setup-running, status answers config: setup.
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
wait "$paused" || fail "the paused setup exited $?"
echo "## the paused setup, finished"
cat "$work/paused.out"

# 10. --json: one object per line, the key in the key object only.
show "--json" "$bf" setup --json --yes --root "$work/6/root" --instance t --repo "$repo" --wait 1
[ "$code" = 0 ] || fail "--json exited $code"
key=$(sed -n 's/.*"step":"key".*"key":"\([0-9a-f]\{64\}\)".*/\1/p' "$work/out")
[ -n "$key" ] || fail "no key object"
[ "$(grep -c "$key" "$work/out")" = 1 ] || fail "the key is printed more than once"

trap - EXIT
echo "setup.sh: ok"
