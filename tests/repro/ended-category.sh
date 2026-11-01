#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# An ended session's venue moves to the category Ended sessions and its lock against sends is attempted
# (04-ux-session-end.md, Ending a session), on a scratch root, from the repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/ended-category.sh
#
# BF names another binary than bin/broodforge. The scratch project's venue.category is set to 900
# and its venue.endedCategory to 910 before serve. Session s0001 runs one policy-mode turn against fixtures/engine/basic and is
# ended: after the lobby's summary and transcript the outbox has `move  s0001  0  910` (payload
# {"parent_id":"910"}) then `lock  s0001  0  send: the bot only`, the move and the lock the live
# gateway sends as REST calls. The key is emptied and the config reloaded; s0002 ends with neither
# line and one warning in serve.log naming the setup step. With venue.category emptied too (the
# dry-run form) s0003 ends with neither line and no warning. The bot's output is serve.log under the
# root, kept on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log
project=$root/config/projects/scratch.json

fail() {
	echo "ended-category: $*" >&2
	exit 1
}

# outbox_has <session> <pattern> [global flags]: the coalescer paces posts, so wait for one.
outbox_has() {
	for _ in $(seq 1 40); do
		"$bf" "${@:3}" session outbox "$1" --all | grep -q -- "$2" && return 0
		sleep 0.25
	done
	return 1
}

grep -q '"category": ""' "$project" && grep -q '"endedCategory": ""' "$project" ||
	fail "the scratch project has no empty venue.category and venue.endedCategory"
sed -i -e 's/"category": ""/"category": "900"/' -e 's/"endedCategory": ""/"endedCategory": "910"/' "$project"
"$bf" config check >/dev/null || fail "config check refused the categories"

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

start() {
	"$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic
}

# 1. The end moves the venue under 910, then locks it, after the lobby's transcript.
id=$(start)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Read hello.txt and reply with its one word." >/dev/null
"$bf" session end "$id" --as 1001 --roles 10 >/dev/null
outbox_has "$id" "  lock  $id  0  send: the bot only\$" || fail "no lock line"
out=$("$bf" session outbox "$id" --all)
printf '%s\n' "$out" | tail -n 4
ops=$(awk '{print $3}' <<<"$out" | tail -n 3 | tr '\n' ' ')
[ "$ops" = "attach move lock " ] || fail "the last three posts are $ops, want attach move lock"
grep -q "  move  $id  0  910\$" <<<"$out" || fail "no move line to 910"
json=$("$bf" --json session outbox "$id" --all)
grep -q '"op":"move","session":"'"$id"'","channel":"'"$id"'","messageId":"'"$id"'","length":0,"first":"910","payload":{"parent_id":"910"}}' <<<"$json" ||
	fail "the move's payload: $(grep '"op":"move"' <<<"$json")"
grep -q '"op":"lock","session":"'"$id"'","channel":"'"$id"'","messageId":"'"$id"'","length":0,"first":"send: the bot only"}' <<<"$json" ||
	fail "the lock: $(grep '"op":"lock"' <<<"$json")"
! grep -q 'venue.endedCategory' "$log" || fail "a warning with the key set: $(grep 'venue.endedCategory' "$log")"

# 2. No key: no move, no lock, one warning naming the step.
sed -i 's/"endedCategory": "910"/"endedCategory": ""/' "$project"
"$bf" config reload >/dev/null
id=$(start)
[ "$id" = s0002 ] || fail "session start printed $id, want s0002"
"$bf" session end "$id" --as 1001 --roles 10 >/dev/null
outbox_has "$id" '"op":"attach","session":"'"$id"'","channel":"100",' --json || fail "no transcript for $id"
! "$bf" session outbox "$id" --all | grep -q '  \(move\|lock\)  ' || fail "$id moved without a category"
warn=$(grep "moving the venue of $id" "$log") || fail "no warning for $id in serve.log"
printf '%s\n' "$warn"
[ "$warn" = "warning: discord: moving the venue of $id: project scratch has no venue.endedCategory; the channel stays under venue.category (broodforge setup channels makes the category)" ] ||
	fail "the warning: $warn"

# 3. Neither category: no move, no lock, no warning.
sed -i 's/"category": "900"/"category": ""/' "$project"
"$bf" config reload >/dev/null
id=$(start)
[ "$id" = s0003 ] || fail "session start printed $id, want s0003"
"$bf" session end "$id" --as 1001 --roles 10 >/dev/null
outbox_has "$id" '"op":"attach","session":"'"$id"'","channel":"100",' --json || fail "no transcript for $id"
! "$bf" session outbox "$id" --all | grep -q '  \(move\|lock\)  ' || fail "$id moved without a category"
! grep -q "moving the venue of $id" "$log" || fail "a warning with no category: $(grep "moving the venue of $id" "$log")"

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
trap - EXIT
echo "ended-category: ok"
