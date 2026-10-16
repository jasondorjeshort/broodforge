#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# One whole session against the fake engine (05-debug-interface-session.md, Session lifecycle
# without Discord; R-51), on a scratch root, from the repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/full-session.sh
#
# It drives the bot through the CLI and reads the outcome through `session tail` and `audit`
# only: a refused start, one policy-mode turn against fixtures/engine/basic, the reach report,
# its usage, `config reload`, the handoff and `stop`. The bot's output is kept as serve.log
# under the root, which scratch run keeps when this exits non-zero.
set -euo pipefail

bf=bin/broodforge
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log

fail() {
	echo "full-session: $*" >&2
	exit 1
}

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT

"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

# Roles: 11 is in no list of channel 100; 10 is group trusted (05-cli-conventions.md, Scratch).
"$bf" session start --channel 100 --as 1001 --roles 11 --credential cred-scratch --expect refused
id=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"

"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Read hello.txt and reply with its one word." >/dev/null

tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
for want in \
	'  session-started  Session s0001 on Scratch, branch broodforge/' \
	'  tool-call  \[Read\] .*hello\.txt$' \
	'  permission-decision  allow \[Read\] .*hello\.txt (policy)$' \
	'  text  hello$' \
	'  turn-ended  end_turn$'; do
	grep -q -- "$want" <<<"$tail" || fail "session tail has no line matching: $want"
done

audit=$("$bf" audit --all)
printf '%s\n' "$audit"
for want in \
	'  action-refused  -  1001  -  {"action":"start",.*"reason":"roles"}' \
	'  session-started  -  1001  s0001  ' \
	'  permission-decision  -  policy  s0001  {"decidedBy":"policy","decision":"allow",'; do
	grep -q -- "$want" <<<"$audit" || fail "audit has no line matching: $want"
done

"$bf" session reach-report "$id" --expect code >/dev/null
[ "$("$bf" usage --session "$id" --by turn | wc -l)" = 1 ] || fail "usage --by turn has not one turn"
# A reload with nothing changed writes nothing; one of a key the bot applies writes the event.
[ "$("$bf" config reload)" = "reloaded  -" ] || fail "an unchanged reload did not print reloaded  -"
sed -i 's/"idleTimeoutMinutes": 60/"idleTimeoutMinutes": 30/' "$root/config/instance.json"
[ "$("$bf" config reload)" = "reloaded  defaults" ] || fail "config reload did not report defaults"
[ "$("$bf" audit --kind configuration-changed --all | wc -l)" = 2 ] || fail "config reload wrote no configuration-changed"

"$bf" worktree handoff "$id" --expect none
"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
trap - EXIT
"$bf" audit --session "$id" --kind session-interrupted --all | grep -q '"reason":"shutdown"' ||
	fail "stop did not mark $id interrupted"
echo "full-session: ok"
