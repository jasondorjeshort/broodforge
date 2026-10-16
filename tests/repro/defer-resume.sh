#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Prompt mode against the fake engine (05-debug-interface-session.md, Prompt-mode defer and
# resume), on a scratch root, from the repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/defer-resume.sh
#
# Channel 100 is prompt mode. Session s0001 replays fixtures/engine/defer: turn 1 defers a Write
# and the driver allows it; turn 2 defers again and the prompt timeout (3 s) denies it.
# Session s0002 replays fixtures/engine/parallel: two Writes in one message, one dropped (C28).
# The outcome is read through `session tail` and `audit` (and `session show`'s pending line while
# waiting); the bot's output is kept as serve.log under the root, which scratch run keeps on a failure.
set -euo pipefail

bf=bin/broodforge
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log

fail() {
	echo "defer-resume: $*" >&2
	exit 1
}

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT

"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

id=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --fixture defer --prompt-timeout-seconds 3)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"

# Turn 1: the Write waits for an answer; someone without an answer role is refused; the driver allows.
"$bf" session send "$id" --as 1001 --wait --expect waiting-for-answer "write hi.txt" >/dev/null
show=$("$bf" session show "$id")
printf '%s\n' "$show"
grep -q '^pending: toolu_01NmkmkiJgpriuU7yQrtDWYi  Write  .*hi\.txt  since .*  until ' <<<"$show" ||
	fail "session show has no pending line while waiting"
"$bf" session answer "$id" allow --as 1001 --roles 11 --expect refused
"$bf" session answer "$id" allow --as 1001 --roles 10 --wait --expect turn-ended >/dev/null

# Turn 2: nobody answers; the prompt timeout denies the call and the turn finishes.
"$bf" session send "$id" --as 1001 --wait --expect waiting-for-answer "write bye.txt" >/dev/null
for _ in $(seq 1 60); do
	"$bf" session tail "$id" --all | grep -q '  turn-ended  end_turn$' &&
		[ "$("$bf" session tail "$id" --all | grep -c '  turn-ended  ')" = 2 ] && break
	sleep 0.25
done

tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
for want in \
	'  permission-decision  deferred \[Write\] .*hi\.txt (policy)$' \
	'  waiting-for-answer  toolu_01NmkmkiJgpriuU7yQrtDWYi \[Write\] ' \
	'  permission-decision  allow \[Write\] .*hi\.txt (participant)$' \
	'  permission-decision  deny \[Write\] .*bye\.txt (prompt-timeout)$' \
	'  tool-result  \[Write\] .*bye\.txt -> denied: the prompt timed out after 3 seconds\.$'; do
	grep -q -- "$want" <<<"$tail" || fail "session tail has no line matching: $want"
done
[ "$(grep -c '  turn-ended  end_turn$' <<<"$tail")" = 2 ] || fail "both turns did not end"

# Parallel calls: the engine keeps the message's last call and drops the other (C28).
par=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --fixture parallel)
"$bf" session send "$par" --as 1001 --wait --expect waiting-for-answer "write both" >/dev/null
# Stop while a call waits: the call is denied as interrupted and the turn ends, no resume.
"$bf" session interrupt "$par" --as 1001 --expect turn-ended >/dev/null

audit=$("$bf" audit --kind permission-decision --all)
printf '%s\n' "$audit"
grep -q -- '  instance  s0001  {"decidedBy":"prompt-timeout","decision":"deny",' <<<"$audit" ||
	fail "audit has no prompt-timeout deny"
grep -q -- '  1001  s0001  {"decidedBy":"participant","decision":"allow",' <<<"$audit" ||
	fail "audit has no participant allow"
[ "$(grep -c -- "  $par  {\"decidedBy\":\"policy\",\"decision\":\"dropped\"," <<<"$audit")" = 1 ] ||
	fail "audit does not have exactly one dropped call for $par"
grep -q -- "  1001  $par  {\"decidedBy\":\"participant\",\"decision\":\"deny\",\"reason\":\"interrupted\"," <<<"$audit" ||
	fail "the interrupt did not deny the waiting call of $par"

# Usage: one record per turn and model, as the fixture reported.
[ "$("$bf" usage --session "$id" --by turn | wc -l)" = 2 ] || fail "usage --by turn has not two turns"

"$bf" session end "$id" --as 1001 --expect none
"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
trap - EXIT
"$bf" audit --kind session-ended --all | grep -q '  1001  s0001  {"outcome":"none","reason":"end"' ||
	fail "session end left no session-ended event"
echo "defer-resume: ok"
