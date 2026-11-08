#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# A subagent's prompt in prompt mode against the fake engine (ADR 0029 as amended 2026-11-07;
# docs/design/03-hook-contract-prompt-tool.md, Debug surface), on a scratch root, from the
# repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/subagent-prompt.sh
#
# BF names another binary than bin/broodforge. Channel 100 (prompt mode) gets
# subagentPromptTimeoutMinutes 1 before serve. s0001 replays fixtures/engine/prompt-ask-allow: the
# subagent's Bash waits in place (`waiting-for-answer`, `session show`'s pending line with `agent
# probe`, the prompt naming the agent after the turn number) and the driver's allow completes the
# turn. s0002 replays prompt-ask-two, answered deny then allow by --tool-use-id. s0003 replays
# prompt-ask-background: the request open at the first result is dropped by the instance and the
# turn ends once. s0004 replays prompt-ask-interrupt: Stop denies the request and the turn ends
# interrupted. s0005 replays prompt-ask-no-tool: rule 8's engine-prompt-tool-missing row. Then
# config check on tests/config's three subagent-prompt trees. The bot's output is serve.log under
# the root, kept on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log
project=$root/config/projects/scratch.json

fail() {
	echo "subagent-prompt: $*" >&2
	exit 1
}

# outbox_has <session> <pattern>: the coalescer paces posts, so wait for one.
outbox_has() {
	for _ in $(seq 1 40); do
		"$bf" session outbox "$1" --all | grep -q -- "$2" && return 0
		sleep 0.25
	done
	return 1
}

# tail_has <session> <pattern>: the turn runs on after the command returns, so wait for a line.
tail_has() {
	for _ in $(seq 1 40); do
		"$bf" session tail "$1" --all | grep -q -- "$2" && return 0
		sleep 0.25
	done
	return 1
}

start() {
	"$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --fixture "$1"
}

# Channel 100's bound, the one channel of the two: its block runs from its id to its effort key.
grep -q '"subagentPromptTimeoutMinutes": null' "$project" || fail "the scratch project has no subagentPromptTimeoutMinutes key"
sed -i '/"100": {/,/"effort"/ s/"subagentPromptTimeoutMinutes": null/"subagentPromptTimeoutMinutes": 1/' "$project"
"$bf" config check >/dev/null || fail "config check refused subagentPromptTimeoutMinutes 1"
eff=$("$bf" config show --effective --channel 100)
printf '%s\n' "$eff"
grep -q '^100  scratch  code  prompt  fake  0  0  1  ' <<<"$eff" || fail "config show --effective does not print channel 100's 1"

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

# 1. prompt-ask-allow: the live request, its pending line and prompt, the driver's allow.
id=$(start prompt-ask-allow)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"
bash1=toolu_01GgR6UWwxWCU6FVfmQSDfqe
w=$("$bf" session send "$id" --as 1001 --wait --expect waiting-for-answer "Have probe run echo hi.")
printf '%s\n' "$w"
grep -q "^waiting-for-answer  1  .*$bash1 \[Bash\] echo hi\$" <<<"$w" || fail "send --wait did not print the subagent's waiting-for-answer"
show=$("$bf" session show "$id")
printf '%s\n' "$show"
grep -q "^pending: $bash1  Bash  echo hi  since .*  until .*  agent probe\$" <<<"$show" ||
	fail "session show has no pending line with agent probe"
outbox_has "$id" '  create  m[0-9]*  [0-9]*  <@1001> Permission needed · turn 1 · probe$' || fail "the prompt does not name the agent after the turn number"
done1=$("$bf" session answer "$id" allow --as 1001 --roles 10 --wait --expect turn-ended)
printf '%s\n' "$done1"
[ "$done1" = "turn-ended  1  completed" ] || fail "session answer --wait printed $done1"
tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
for want in \
	"  permission-decision    deferred \[Bash\] echo hi (policy)\$" \
	"  waiting-for-answer    $bash1 \[Bash\] echo hi\$" \
	'  permission-decision    allow \[Bash\] echo hi (participant)$'; do
	grep -q -- "$want" <<<"$tail" || fail "s0001's tail has no line matching: $want"
done

# 2. prompt-ask-two: deny the first by its id, then allow the second by its id.
id=$(start prompt-ask-two)
alpha=toolu_018uaKNYzw3pU4MMDYJq1Mhh beta=toolu_0142ACPeh2HeMXsQwZ2wcynU
"$bf" session send "$id" --as 1001 --wait --expect waiting-for-answer "Have probe echo alpha and beta." >/dev/null
"$bf" session answer "$id" deny --tool-use-id "$beta" --as 1001 --roles 10 --expect not-pending >/dev/null
"$bf" session answer "$id" deny --tool-use-id "$alpha" --as 1001 --roles 10 --wait --expect waiting-for-answer >/dev/null
"$bf" session answer "$id" allow --tool-use-id "$beta" --as 1001 --roles 10 --wait --expect turn-ended >/dev/null
tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
grep -q '  permission-decision    deny \[Bash\] echo alpha (participant)$' <<<"$tail" || fail "s0002's alpha is not denied by the participant"
grep -q '  permission-decision    allow \[Bash\] echo beta (participant)$' <<<"$tail" || fail "s0002's beta is not allowed by the participant"

# 3. prompt-ask-background: the request open at the first result is dropped; one turn-ended.
id=$(start prompt-ask-background)
"$bf" session send "$id" --as 1001 --wait --expect waiting-for-answer "Have probe echo hi in the background." >/dev/null
tail_has "$id" '  turn-ended  end_turn$' || fail "s0003's turn did not end"
tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
grep -q '  permission-decision    dropped \[Bash\] echo hi (instance)$' <<<"$tail" || fail "s0003's request was not dropped by the instance"
grep -q '  tool-result    \[Bash\] echo hi -> dropped: the turn ended before anyone answered$' <<<"$tail" ||
	fail "s0003 has no dropped trace line"
[ "$(grep -c '  turn-ended  ' <<<"$tail")" = 1 ] || fail "s0003 does not have exactly one turn-ended"
grep -q 'allow \[Bash\] echo hi' <<<"$tail" && fail "s0003 acted on the allow after the first result"

# 4. prompt-ask-interrupt: Stop denies the open request and the turn ends interrupted.
id=$(start prompt-ask-interrupt)
"$bf" session send "$id" --as 1001 --wait --expect waiting-for-answer "Have probe echo hi." >/dev/null
stopped=$("$bf" session interrupt "$id" --as 1001 --roles 10 --expect turn-ended)
printf '%s\n' "$stopped"
[ "$stopped" = "turn-ended  1  interrupted" ] || fail "session interrupt printed $stopped"
tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
grep -q '  permission-decision    deny \[Bash\] echo hi (participant)$' <<<"$tail" || fail "s0004's request is not denied by the participant"
grep -q '"interrupt": true' "$root/sessions/$id/fake/received-1.json" || fail "s0004's deny did not carry interrupt true"

# 5. prompt-ask-no-tool: a hook ask with no request on the stream is rule 8's detector.
id=$(start prompt-ask-no-tool)
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Have probe echo hi." >/dev/null
missing=$("$bf" audit --kind engine-prompt-tool-missing --all)
printf '%s\n' "$missing"
[ "$(grep -c "  $id  " <<<"$missing")" = 1 ] || fail "audit has not exactly one engine-prompt-tool-missing row for $id"
tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
grep -q '  internal-error  engine prompt tool: wanted stdio, got none$' <<<"$tail" || fail "s0005 has no internal-error"
grep -q '  tool-result    \[Bash\] echo hi -> no prompt reached anyone: ' <<<"$tail" || fail "s0005 has no trace line naming the lift"

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
trap - EXIT

# The key's refusals and cap, on tests/config's trees.
for tree in zero negative; do
	if out=$("$bf" config check --root "tests/config/subagent-prompt-$tree" 2>&1); then
		fail "config check accepted subagent-prompt-$tree"
	fi
	printf '%s\n' "$out"
	grep -q 'subagentPromptTimeoutMinutes: -\?[0-9]*, want a positive number of minutes (0 is not never here)$' <<<"$out" ||
		fail "config check on subagent-prompt-$tree does not name the key"
done
out=$("$bf" config check --root tests/config/subagent-prompt-capped) || fail "config check refused subagent-prompt-capped"
printf '%s\n' "$out"
grep -q 'subagentPromptTimeoutMinutes 2000 is more than a day; capped at 1440$' <<<"$out" || fail "config check did not warn of the cap"
echo "subagent-prompt: ok"
