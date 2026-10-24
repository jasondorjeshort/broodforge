#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# A restart, then `end` on the interrupted session (ADR 0026; 05-debug-interface-discord-live.md,
# Inbound log and held sessions), on a scratch root, from the repository root:
#
#   MSYS_NO_PATHCONV=1 bin/broodforge scratch run -- bash tests/repro/restart-end.sh
#
# (MSYS_NO_PATHCONV=1 under Git Bash, which would rewrite a "/session ..." argument into a
# path.) BF names another binary than bin/broodforge, for a build kept out of bin/.
# Session s0001 runs one policy-mode turn against fixtures/engine/basic, gets one commit in its
# tree with the Haiku co-author trailer, and is interrupted by `stop`. The second `serve` holds
# it: its stderr says `held  s0001`, its log opens with the restart, the venue gets the notice
# with the tree's commit count, a message is refused with the interrupted text, and End session
# hands it off and ends it with the summary, the lobby's copy and the transcript. Each serve's
# output is kept under the root (serve.log, serve2.log), which scratch run keeps on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log
log2=$root/serve2.log

fail() {
	echo "restart-end: $*" >&2
	exit 1
}

# outbox_has <session> <pattern> [global flags]: the post is in the outbox once the coalescer
# lets it out, so wait for it.
outbox_has() {
	for _ in $(seq 1 40); do
		"$bf" "${@:3}" session outbox "$1" --all | grep -q -- "$2" && return 0
		sleep 0.25
	done
	return 1
}

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
grep -q '^held  -$' "$log" || fail "the first serve did not print held  - (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

id=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Read hello.txt and reply with its one word." >/dev/null

# One commit in the tree, as the agent would make it: the driver the author, the model credited.
tree=$root/sessions/$id/tree
printf 'hi\n' >"$tree/hi.txt"
git -C "$tree" -c core.autocrlf=false add hi.txt
git -C "$tree" -c user.name="Ada Example" -c user.email=ada@example.com -c commit.gpgsign=false \
	-c core.hooksPath=/dev/null commit --quiet -m "Add hi.txt" \
	-m "Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>"

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
"$bf" session list --state interrupted | grep -q "^$id  interrupted  " || fail "$id is not interrupted after the stop"

# The restart: the session is held.
"$bf" serve >"$log2" 2>&1 &
serve=$!
"$bf" status --wait 10 --expect ok >/dev/null || fail "the second bot did not answer (serve2.log: $log2)"
grep -q "^held  $id$" "$log2" || fail "the second serve did not print held  $id (serve2.log: $log2)"

tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
head -n 1 <<<"$tail" | grep -q '  session-interrupted  restart$' || fail "session tail does not open with the restart"

outbox_has "$id" '  create  m[0-9]*  [0-9]*  Broodforge restarted; this session was interrupted. Its tree is kept with 1 commit\. ' ||
	fail "session outbox has no notice with 1 commit"

transcript=$("$bf" session transcript "$id")
printf '%s\n' "$transcript"
head -n 1 <<<"$transcript" | grep -qx "# Session $id" || fail "the transcript does not open with its heading"
grep -q 'Broodforge restarted; this session was interrupted\.' <<<"$transcript" || fail "the transcript has no restart line"

# Nothing but end acts on it.
set +e
refusal=$("$bf" discord inject message --channel "$id" --user 1001 --roles 10 "hi" 2>&1)
code=$?
set -e
[ "$code" = 1 ] || fail "a message on the held session exited $code"
[ "$refusal" = "error: session-closed: This session was interrupted by a restart. /session end hands off its work." ] ||
	fail "a message on the held session: $refusal"

# End session from the header: the handoff runs, the session ends.
ended=$("$bf" discord inject click --custom-id "bf:$id:end" --user 1001 --roles 10)
[ "$ended" = "none  $id ended" ] || fail "End session printed: $ended"
audit=$("$bf" audit --kind session-ended --all)
printf '%s\n' "$audit"
grep -- "  1001  $id  " <<<"$audit" | grep -q '"reason":"end"' || fail "audit has no session-ended with the reason end"
grep -- "  1001  $id  " <<<"$audit" | grep -q '"state":"interrupted"' || fail "audit's session-ended does not say interrupted"

outbox_has "$id" '"op":"create","session":"'"$id"'","channel":"'"$id"'","messageId":"m[0-9]*","length":[0-9]*,"first":"Session '"$id"' ended by Ada Example' --json ||
	fail "the venue has no summary"
outbox_has "$id" '"op":"create","session":"'"$id"'","channel":"100","messageId":"m[0-9]*","length":[0-9]*,"first":"Session '"$id"' ended by Ada Example' --json ||
	fail "the lobby has no summary"
outbox_has "$id" '"op":"attach","session":"'"$id"'","channel":"100",' --json || fail "the lobby's summary has no transcript"
"$bf" session outbox "$id" --all

# Ended now: end again is session-closed.
"$bf" session end "$id" --as 1001 --expect session-closed

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve2.log: $log2)"
trap - EXIT
echo "restart-end: ok"
