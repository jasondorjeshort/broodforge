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
# its usage, `config reload`, the handoff and `stop`; `session outbox --json` gives the header's menus. The scratch canonical first gets
# fixtures/workflow/'s CLAUDE.md and .claude on main and the project both hook events (ADR 0025),
# so the session starts with the workflow injected: the fake's request carries the file flag and
# --agents, `session show` its workflow:, hooks: and review: lines and --header the header. The
# probe runs the fixture's hooks' runtime, so node must be on PATH. The bot's output is kept as
# serve.log under the root, which scratch run keeps when this exits non-zero.
set -euo pipefail

bf=bin/broodforge
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log

fail() {
	echo "full-session: $*" >&2
	exit 1
}

# The project's workflow on its default branch, and the key listing both hook events.
canon=$root/projects/scratch
cp -R fixtures/workflow/CLAUDE.md fixtures/workflow/.claude "$canon/"
git -C "$canon" add CLAUDE.md .claude
git -C "$canon" -c user.name="Ada Example" -c user.email=ada@example.com -c commit.gpgsign=false \
	-c core.hooksPath=/dev/null commit --quiet -m "The workflow fixture"
commit=$(git -C "$canon" rev-parse HEAD)
sed -i 's/^  "name": "Scratch",$/  "name": "Scratch",\n  "workflow": {"hooks": ["PreToolUse", "PostToolUse"]},/' \
	"$root/config/projects/scratch.json"
grep -q '"workflow"' "$root/config/projects/scratch.json" || fail "the workflow key was not written"

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

# The workflow injected (ADR 0025, P19): the request's flags, the three session show lines,
# and the header the session got, first its precedence sentence.
recv=$root/sessions/$id/fake/received-1.json
grep -q -- "\"--append-system-prompt-file\",\$" "$recv" || fail "received-1.json has no --append-system-prompt-file"
grep -q -- "\"--agents\",\$" "$recv" || fail "received-1.json has no --agents"
if grep -q -- "\"--append-system-prompt\"" "$recv"; then fail "received-1.json has the retired --append-system-prompt"; fi
show=$("$bf" session show "$id")
printf '%s\n' "$show"
for want in "^workflow: $commit 20" '^hooks: git-guard\.js, post-edit\.js$' "^review: none since $commit\$"; do
	grep -q -- "$want" <<<"$show" || fail "session show has no line matching: $want"
done
header=$("$bf" session show --header "$id")
printf '%s\n' "$header"
[ "$(head -n 1 <<<"$header")" = "This session is run by Broodforge. Where the project's instructions below conflict with this header, this header wins; it never says how the project's work is done." ] ||
	fail "session show --header does not open with the precedence sentence"
grep -q -- "^The project's instructions below were taken from main at $commit on " <<<"$header" ||
	fail "session show --header has no taken-from line"

# The dry-run renderer's posts (P11): the venue, the pinned header, the working message, then
# the turn's final message ending in 04's footer. The coalescer paces them, so poll.
outbox_has() {
	for _ in $(seq 1 40); do
		"$bf" --json session outbox "$1" --all | grep -q -- "$2" && return 0
		sleep 0.25
	done
	return 1
}
outbox_has "$id" '\\nIdle · turn 1 · [0-9:]* · [0-9.k]* in / [0-9.k]* out · session [0-9.k]* · Haiku · acts freely"' ||
	fail "session outbox has no final message ending in the footer"
outbox=$("$bf" session outbox "$id" --all)
printf '%s\n' "$outbox"
for want in \
	'  channel  s0001  0  #s0001-ada$' \
	'  create  m[0-9]*  [0-9]*  Session s0001 on Scratch, branch broodforge/' \
	'  create  m[0-9]*  [0-9]*  Working · turn 1 · ' \
	'  delete  m[0-9]*  0  -$'; do
	grep -q -- "$want" <<<"$outbox" || fail "session outbox has no line matching: $want"
done

# The header's menus (P12b; 05-debug-interface-discord.md, The header's menus): the pinned create
# carries model, effort and mode in that order; channel 100 sets no model and the start no effort,
# so only mode, started acts-freely, marks a default; disgo's payload has one row per menu, then
# the End session row.
header=$("$bf" --json session outbox "$id" --all | grep -m 1 '"pinned":true') || fail "session outbox has no pinned header"
menus=${header%%,\"pinned\":*}
menus=${menus#*\"selects\":}
grep -q -- "^\[{\"customId\":\"bf:$id:model\",\"placeholder\":\"Model: the engine.s own\",\"options\":\[{\"label\":\"Opus\",\"value\":\"opus\"}" <<<"$menus" ||
	fail "the header's first menu is not the model's with its placeholder: $menus"
grep -q -- "},{\"customId\":\"bf:$id:effort\",\"placeholder\":\"Effort: the engine.s own\",\"options\":\[{\"label\":\"low\",\"value\":\"low\"}" <<<"$menus" ||
	fail "the header's second menu is not the effort's with its placeholder: $menus"
grep -q -- "},{\"customId\":\"bf:$id:mode\",\"options\":\[{\"label\":\"asks first\",\"value\":\"asks-first\"},{\"label\":\"acts freely (within reach)\",\"value\":\"acts-freely\",\"default\":true}\]}\]$" <<<"$menus" ||
	fail "the header's last menu is not the mode's marking acts-freely: $menus"
[ "$(grep -o '"default":true' <<<"$menus" | wc -l)" = 1 ] || fail "a menu other than mode marks a default: $menus"
rows=$(grep -o '"custom_id":"[^"]*"' <<<"${header#*\"payload\":}" | tr '\n' ' ')
[ "$rows" = "\"custom_id\":\"bf:$id:model\" \"custom_id\":\"bf:$id:effort\" \"custom_id\":\"bf:$id:mode\" \"custom_id\":\"bf:$id:end\" " ] ||
	fail "the header's payload rows are not model, effort, mode, End session: $rows"

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
# The scratch default is 0 (never) since 2026-10-23; any other value is a change the bot applies.
sed -i 's/"idleTimeoutMinutes": 0,/"idleTimeoutMinutes": 30,/' "$root/config/instance.json"
[ "$("$bf" config reload)" = "reloaded  defaults" ] || fail "config reload did not report defaults"
[ "$("$bf" audit --kind configuration-changed --all | wc -l)" = 2 ] || fail "config reload wrote no configuration-changed"

"$bf" worktree handoff "$id" --expect none
"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
trap - EXIT
"$bf" audit --session "$id" --kind session-interrupted --all | grep -q '"reason":"shutdown"' ||
	fail "stop did not mark $id interrupted"
echo "full-session: ok"
