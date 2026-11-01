#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# The end's WIP commit of a dirty tree and the end without a handoff (04-ux-session-end.md;
# 05-debug-interface-git.md, session end --discard), on a scratch root, from the repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/end-wip-discard.sh
#
# BF names another binary than bin/broodforge. Session s0001 runs one policy-mode turn against
# fixtures/engine/basic, leaves an edited file uncommitted, and is ended: the line is pushed with 1
# commit, the remote branch holds the one WIP commit by the driver with the model's trailer, the
# summary carries the owner's WIP line and the audit's session-ended its wip. Session s0002 commits
# one file and leaves another uncommitted; the header carries End session and End and discard side
# by side; End and discard posts the confirm with its Discard button, which ends the session: two
# commits on refs/aborted/s0002 in the canonical, nothing pushed, the tree gone, session-discarded
# in the audit, the discard summary in the venue and the lobby. Session s0003, clean, is discarded
# from the CLI with no ref. The bot's output is serve.log under the root, kept on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log
canon=$root/projects/scratch
remote=$root/remote/scratch.git

fail() {
	echo "end-wip-discard: $*" >&2
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

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

start() {
	"$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic
}

# 1. A dirty tree at the end: one WIP commit, handed off as the project's push.
id=$(start)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Read hello.txt and reply with its one word." >/dev/null
tree=$root/sessions/$id/tree
branch=$(git -C "$tree" symbolic-ref --short HEAD)
printf 'an idea, never committed\n' >"$tree/idea.md"
ended=$("$bf" session end "$id" --as 1001 --roles 10)
printf '%s\n' "$ended"
grep -q "^pushed  origin  refs/sessions/$id:refs/heads/$branch  1  " <<<"$ended" || fail "session end printed: $ended"
[ ! -e "$tree" ] || fail "the tree of $id is kept"
wip=$(git -C "$remote" rev-parse "refs/heads/$branch")
[ "$(git -C "$remote" log -1 --format=%s "$wip")" = "WIP: uncommitted changes committed at end" ] || fail "the pushed commit is not the WIP commit"
[ "$(git -C "$remote" log -1 --format='%an <%ae>|%cn <%ce>' "$wip")" = "Ada Example <ada@example.com>|Ada Example <ada@example.com>" ] ||
	fail "the WIP commit is not the driver's"
git -C "$remote" log -1 --format=%B "$wip" | grep -q '^Co-Authored-By: Claude .* <noreply@anthropic.com>$' || fail "the WIP commit has no model trailer"
[ "$(git -C "$remote" show --name-only --format= "$wip")" = idea.md ] || fail "the WIP commit does not hold idea.md"
"$bf" --json audit --session "$id" --kind session-ended --all | grep -q "\"wip\":\"$wip\"" || fail "session-ended has no wip"
outbox_has "$id" '  create  m[0-9]*  [0-9]*  Ending: checking and handing off 1 commit\.\.\.$' || fail "no Ending line with 1 commit"
outbox_has "$id" "1 commit · pushed to $branch (WIP: uncommitted changes committed at end)" --json || fail "the summary has no WIP line"

# 2. End and discard from the header: the confirm, then the end without a handoff.
id=$(start)
[ "$id" = s0002 ] || fail "session start printed $id, want s0002"
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Read hello.txt and reply with its one word." >/dev/null
tree=$root/sessions/$id/tree
branch=$(git -C "$tree" symbolic-ref --short HEAD)
printf 'kept\n' >"$tree/kept.txt"
git -C "$tree" -c core.autocrlf=false add kept.txt
git -C "$tree" -c user.name="Ada Example" -c user.email=ada@example.com -c commit.gpgsign=false \
	-c core.hooksPath=/dev/null commit --quiet -m "Add kept.txt" -m "Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>"
printf 'dirty\n' >"$tree/dirty.txt"

header=$("$bf" --json session outbox "$id" --all | grep -m 1 '"pinned":true') || fail "no pinned header"
grep -q '"buttons":\[{"label":"End session","customId":"bf:'"$id"':end","style":"danger"},{"label":"End and discard","customId":"bf:'"$id"':discard","style":"danger"}\]' <<<"$header" ||
	fail "the header has not both end buttons: $header"
rows=$(grep -o '"custom_id":"bf:'"$id"':\(end\|discard\)"' <<<"${header#*\"payload\":}" | tr '\n' ' ')
[ "$rows" = "\"custom_id\":\"bf:$id:end\" \"custom_id\":\"bf:$id:discard\" " ] || fail "the payload's end row: $rows"

# Role 11 is in no list of channel 100 (05-cli-conventions.md, Scratch): refused before any confirm.
set +e
refusal=$("$bf" discord inject click --custom-id "bf:$id:discard" --user 1001 --roles 11 2>&1)
code=$?
set -e
[ "$code" = 1 ] && grep -q '^error: refused: ' <<<"$refusal" || fail "End and discard without a role: exit $code, $refusal"
asked=$("$bf" discord inject click --custom-id "bf:$id:discard" --user 1001 --roles 10)
[ "$asked" = "none  $id discard confirm posted" ] || fail "End and discard printed: $asked"
outbox_has "$id" '"first":"Discard session '"$id"'? Its 2 commits stay recoverable.",.*"buttons":\[{"label":"Discard","customId":"bf:'"$id"':discard-confirm","style":"danger"}\]' --json ||
	fail "no confirm with its Discard button"
confirm=$("$bf" session outbox "$id" --all | grep '  create  m[0-9]*  [0-9]*  Discard session ' | awk '{print $4}')
done=$("$bf" discord inject click --custom-id "bf:$id:discard-confirm" --user 1001 --roles 10)
[ "$done" = "none  $id discarded" ] || fail "Discard printed: $done"

[ ! -e "$tree" ] || fail "the tree of $id is kept"
[ -z "$(git -C "$remote" for-each-ref "refs/heads/$branch")" ] || fail "$branch was pushed"
[ -z "$(git -C "$canon" for-each-ref "refs/sessions/$id")" ] || fail "refs/sessions/$id is kept"
aborted=$(git -C "$canon" log --format=%s "main..refs/aborted/$id")
printf '%s\n' "$aborted"
[ "$aborted" = "$(printf 'WIP: uncommitted changes committed at end\nAdd kept.txt')" ] || fail "refs/aborted/$id holds: $aborted"
discarded=$("$bf" --json audit --session "$id" --kind session-discarded --all)
printf '%s\n' "$discarded"
grep -q "\"actor\":\"1001\",\"session\":\"$id\",.*\"commits\":\"2\",\"ref\":\"refs/aborted/$id\"" <<<"$discarded" ||
	fail "no session-discarded row by 1001 with 2 commits and the ref"
"$bf" --json audit --session "$id" --kind session-ended --all | grep -q '"outcome":"discarded"' || fail "session-ended is not discarded"
for channel in "$id" 100; do
	outbox_has "$id" '"op":"create","session":"'"$id"'","channel":"'"$channel"'","messageId":"m[0-9]*","length":[0-9]*,"first":"Session '"$id"' ended by Ada Example' --json ||
		fail "no summary in channel $channel"
done
outbox_has "$id" '2 commits · discarded; recoverable on request\\nKept as refs/aborted/'"$id"' in the project' --json ||
	fail "the summary has no discard lines"
outbox_has "$id" '"op":"attach","session":"'"$id"'","channel":"100",' --json || fail "the lobby's summary has no transcript"
# The standing confirm is edited once the session ended: its text kept, its button gone.
edit=$("$bf" --json session outbox "$id" --all | grep '"op":"edit","session":"'"$id"'","channel":"'"$id"'","messageId":"'"$confirm"'",') ||
	fail "the confirm $confirm was not edited after the end"
grep -q '"first":"Discard session '"$id"'? Its 2 commits stay recoverable."' <<<"$edit" && ! grep -q 'discard-confirm' <<<"$edit" ||
	fail "the confirm kept its button after the end: $edit"
"$bf" session outbox "$id" --all

# 3. The CLI's --discard on a clean session with no commits: no ref.
id=$(start)
[ "$id" = s0003 ] || fail "session start printed $id, want s0003"
line=$("$bf" session end "$id" --as 1001 --discard)
[ "$line" = "discarded  origin  -  0  -  -" ] || fail "session end --discard printed: $line"
[ -z "$(git -C "$canon" for-each-ref "refs/aborted/$id")" ] || fail "refs/aborted/$id was written for no commits"
"$bf" session end "$id" --as 1001 --discard --expect session-closed

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
trap - EXIT
echo "end-wip-discard: ok"
