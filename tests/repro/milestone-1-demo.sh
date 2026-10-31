#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# A narrated walk through milestone 1 (08-implementation-plan.md, Milestones, first row), to
# watch, not to assert: each command is printed as input, then its real output. From the
# repository root:
#
#   bash tests/repro/milestone-1-demo.sh [--no-build]
#
# Its own temporary root and instance name; the kept dev root is never touched. Channel 100
# only (the fake engine), so nothing spends the plan. Exit 0 and a PASS line, or the failing
# step, its command, its output and the serve log path, with the root kept. Stop and clean-up
# follow dev-instance.sh, which is the asserting form of the same loop. The printed output
# holds absolute temp and repository paths: do not paste it into a tracked doc (ADR 0018).
set -euo pipefail

bf=bin/broodforge
unset BROODFORGE_KEY_FILE BROODFORGE_SETUP_FAKE_DISCORD BFD_EXPECT BFD_IN BFD_PRE
nobuild=
case "${1:-}" in
"") ;;
--no-build) nobuild=1 ;;
*)
	echo "usage: bash tests/repro/milestone-1-demo.sh [--no-build]" >&2
	exit 2
	;;
esac
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
instance=demo-$(od -An -tx1 -N6 /dev/urandom | tr -d ' \n')
export BROODFORGE_ROOT=$root
log=$base/serve.log
serve=
passed=
step=0
total=6
code=0

# Any exit before the end: say what failed first, then stop the bot this script started through
# its own stop command (never by PID or name), waiting a bounded time with signal 0 only. The
# root is kept and named.
leave() {
	if [ -z "$passed" ]; then
		echo
		echo "FAIL at step $step of $total."
		[ -n "$serve" ] && echo "Serve log: $log"
		echo "The root is kept at $root (instance $instance)."
	fi
	if [ -n "$serve" ]; then
		"$bf" stop >/dev/null 2>&1 || true
		for _ in $(seq 1 40); do
			kill -0 "$serve" 2>/dev/null || break
			sleep 0.25
		done
		if kill -0 "$serve" 2>/dev/null; then
			echo "the bot did not stop; BROODFORGE_ROOT=$root $bf stop"
		fi
	fi
	true
}
trap leave EXIT

hdr() { # hdr <title> <what it proves>
	step=$((step + 1))
	printf '\n== Step %d of %d: %s ==\n%s\n\n' "$step" "$total" "$1" "$2"
}

# run <command...>: print "$ command", run it, print its output (the key line and the account
# line masked), and leave the exit code in $code and the raw output in $tmp/out. A command
# that exits non-zero fails the step unless BFD_EXPECT names that code. BFD_IN is a file for
# its stdin, BFD_PRE a prefix shown before the command.
run() {
	local a s=
	for a in "$@"; do
		case "$a" in
		*[[:space:]]*) s+="'$a' " ;;
		*) s+="$a " ;;
		esac
	done
	echo "\$ ${BFD_PRE:-}${s% }"
	set +e
	"$@" <"${BFD_IN:-/dev/null}" >"$tmp/out" 2>&1
	code=$?
	set -e
	sed -e 's/^\(ok  key  \).*/\1[key hidden]/' \
		-e 's/^\(ok  check  spawn preflight  ok  account  \).*/\1[hidden]/' "$tmp/out"
	if [ "$code" != "${BFD_EXPECT:-0}" ]; then
		echo "(exit $code, expected ${BFD_EXPECT:-0})"
		exit 1
	fi
}

if [ -n "$nobuild" ]; then
	hdr "build" "Skipped (--no-build): the existing $bf is used."
else
	hdr "build" "The Go binary builds from this checkout."
	run env CGO_ENABLED=0 go build -o bin/ ./cmd/broodforge
	echo "(built)"
fi

# Scaffolding, as in dev-instance.sh: a stand-in engine on PATH so the machine's own Claude Code
# is never consulted.
mkdir -p "$tmp/stub"
if [ -n "$win" ]; then
	echo "(building a stand-in engine so your own Claude Code is never used)"
	go test -c -o "$tmp/stub/claude.exe" ./internal/cli
	export BFTEST_STANDIN=1
	export ProgramData
	ProgramData=$(cygpath -w "$tmp/programdata")
else
	printf '#!/bin/sh\necho "2.1.296 (Claude Code)"\n' >"$tmp/stub/claude"
	chmod +x "$tmp/stub/claude"
fi
export PATH="$tmp/stub:$PATH"

hdr "set up a throwaway instance" "Setup writes a dry-run root (no Discord, fake engine on channel 100) in a temp directory."
run "$bf" setup --discord dry-run --yes --root "$root" --instance "$instance" --project broodforge --repo "$repo" --wait 1

hdr "start the bot" "The bot serves and answers a status query."
echo "\$ $bf serve --discord dry-run >$log 2>&1 &"
"$bf" serve --discord dry-run >"$log" 2>&1 &
serve=$!
run "$bf" status --wait 10 --expect ok

hdr "register a user and start a session" "A Discord button click on channel 100 opens a session."
run "$bf" register 1001 --name "Ada Example" --email ada@example.com
printf scratch >"$tmp/scratch"
BFD_IN="$tmp/scratch" BFD_PRE='printf scratch | ' run "$bf" credential add --kind api-key --user 1001 --label fake
cred=$(tr -d '\r\n' <"$tmp/out")
run "$bf" register set-default 1001 "$cred"
run "$bf" discord inject click --custom-id bf:100:start --user 1001 --roles 10
run "$bf" session list
id=$(awk '{print $1; exit}' "$tmp/out")

hdr "send a message and read the reply" "The session runs one turn on the fake engine; the tail shows the reply and the outbox the posts Discord would get."
run "$bf" discord inject message --channel "$id" --user 1001 --roles 10 "Read hello.txt and reply with its one word."
for _ in $(seq 1 80); do
	"$bf" session tail "$id" --all | tail -n 1 | grep -q '  turn-ended  ' && break
	sleep 0.25
done
run "$bf" session tail "$id" --all
tail -n 1 "$tmp/out" | grep -q '  turn-ended  end_turn$' || { echo "(the turn did not end)"; exit 1; }
grep -q '  text  hello$' "$tmp/out" || { echo "(the tail has no reply line: text  hello)"; exit 1; }
# The working message's delete arrives once the coalescer is through: wait for it, then show the outbox.
for _ in $(seq 1 40); do
	"$bf" session outbox "$id" | grep -q '  delete  ' && break
	sleep 0.25
done
run "$bf" session outbox "$id"

hdr "end the session and stop the bot" "Everything shuts down cleanly. The final status exits 1 (not-running): that is the expected result."
run "$bf" session end "$id"
run "$bf" stop
for _ in $(seq 1 40); do
	kill -0 "$serve" 2>/dev/null || break
	sleep 0.25
done
if kill -0 "$serve" 2>/dev/null; then echo "(the bot is still running after stop)"; exit 1; fi
serve=
BFD_EXPECT=1 run "$bf" status

passed=1
cat <<EOF

== Summary ==
The bot started on a throwaway root, a session ran one turn on the fake engine,
the engine's reply (hello) is in the session tail and its posts are in the outbox, and the bot stopped.
Milestone 1, dry-run half: PASS
EOF
rm -rf "$tmp"
