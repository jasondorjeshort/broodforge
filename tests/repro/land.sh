#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Landing a session on the default branch from the host (ADR 0028 Decision 1;
# 05-debug-interface-git.md, Landing; P34), on a scratch root, from the repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/land.sh
#
# BF names another binary than bin/broodforge. The scratch project becomes `handoff` main with
# `handoffFallback` push and channel 100's roles.land role 10. Session s0001 runs on a copy of
# fixtures/engine/reviewer-defer with its two turns played twice (the reviewer's defer and resume,
# the ids renamed the second time). A commit in the tree is refused land-unreviewed; after the
# reviewer turn (the pin) role 11 is refused land-not-allowed and role 10 lands it: the remote's
# main is the tip, `audit --kind land` has the host as the actor and 1001 as claimed. A commit
# pushed to the remote's main from a throwaway clone, a second tree commit and reviewer turn:
# land-not-fast-forward; `session end --as 1001 --roles 10` then falls back to push, the refusal
# before its line. Session s0002, one commit and no review, ended with no roles behind it: the
# first refusal in gate order is land-not-allowed and it falls back too (ADR 0028 Amendments
# 2026-11-02). config check on a copy of the root whose channel lost roles.land prints the
# warning. The bot's output is serve.log under the root, which scratch run keeps on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log
canon=$root/projects/scratch
remote=$root/remote/scratch.git
project=$root/config/projects/scratch.json
clone=$root/clone
fixture=$root/reviewer-twice

fail() {
	echo "land: $*" >&2
	exit 1
}

# commit_tree <file> <message> [<session>]: a commit of <file> in the session's tree (s0001 by
# default) as the driver, no hooks, with the trailer lines the session's home holds (the
# commit-msg hook's, which the bot rewrites).
commit_tree() {
	local tree=$root/sessions/${3:-s0001}/tree trailers=$root/sessions/${3:-s0001}/home/trailers
	local args=(-m "$2")
	printf '%s\n' "$1" >"$tree/$1"
	git -C "$tree" -c core.autocrlf=false add "$1"
	if [ -s "$trailers" ]; then args+=(-m "$(tr -d '\r' <"$trailers")"); fi
	git -C "$tree" -c user.name="Ada Example" -c user.email=ada@example.com -c commit.gpgsign=false \
		-c core.hooksPath=/dev/null commit --quiet "${args[@]}"
	git -C "$tree" rev-parse HEAD
}

# The project lands on main, falls back to push, and channel 100 lets role 10 land.
grep -q '^  "handoff": "push",$' "$project" || fail "the scratch project has no handoff push line"
sed -i 's/^  "handoff": "push",$/  "handoff": "main",\n  "handoffFallback": "push",/' "$project"
sed -i '0,/^      "roles": {$/s//      "roles": {\n        "land": ["10"],/' "$project"
checked=$("$bf" config check) || fail "config check refused the land configuration"
printf '%s\n' "$checked"
! grep -q '^warning: scratch 100: ' <<<"$checked" || fail "a warning for channel 100, which has roles.land"

# reviewer-defer's two turns played twice: turns 3 and 4 are 1 and 2 with the ids renamed.
mkdir -p "$fixture"
for n in 1 2; do
	cp "fixtures/engine/reviewer-defer/turn-$n.jsonl" "$fixture/turn-$n.jsonl"
	sed 's/toolu_rev_/toolu_rew_/g; s/msg_rev_/msg_rew_/g' "fixtures/engine/reviewer-defer/turn-$n.jsonl" >"$fixture/turn-$((n + 2)).jsonl"
done
sid=4e71e7e0-0000-4000-8000-000000000003
printf '{"claim":"hand-written","turns":[%s]}\n' \
	"$(for n in 1 2 3 4; do printf '{"n":%d,"exit":0,"engineSessionId":"%s"}' "$n" "$sid"; [ "$n" = 4 ] || printf ','; done)" \
	>"$fixture/fixture.json"

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

id=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture "$fixture")
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"
branch=$(git -C "$root/sessions/$id/tree" symbolic-ref --short HEAD)

# 1. A commit nobody reviewed: refused, nothing pushed.
before=$(git -C "$remote" rev-parse main)
one=$(commit_tree one.txt "Add one.txt")
"$bf" session land "$id" --expect land-unreviewed
[ "$(git -C "$remote" rev-parse main)" = "$before" ] || fail "an unreviewed land moved the remote's main"

# 2. The reviewer turn pins the tip; a role outside roles.land is refused, role 10 lands it.
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Review the last commit." >/dev/null
"$bf" session show "$id" | grep -q "^review: toolu_rev_agent at $one\$" || fail "no review at $one: $("$bf" session show "$id" | grep '^review:')"
"$bf" session land "$id" --as 1001 --roles 11 --expect land-not-allowed
landed=$("$bf" session land "$id" --as 1001 --roles 10)
printf '%s\n' "$landed"
[ "$landed" = "landed  origin  refs/sessions/$id:refs/heads/main  1  -  -" ] || fail "session land printed: $landed"
[ "$(git -C "$remote" rev-parse main)" = "$one" ] || fail "the remote's main is not the tip $one"
[ "$(git -C "$canon" rev-parse main)" = "$one" ] || fail "the canonical's main did not follow the land"
git -C "$remote" log --oneline -1 main
rows=$("$bf" --json audit --kind land --session "$id" --all)
printf '%s\n' "$rows"
grep -q "\"phase\":\"done\",\"actor\":\"host\",\"session\":\"$id\",.*\"claimed\":\"1001\",.*\"outcome\":\"landed\",.*\"sha\":\"$one\"" <<<"$rows" ||
	fail "no land done row by host with claimed 1001"
"$bf" --json audit --kind action-refused --session "$id" --all | grep -q '"reason":"land-not-allowed"' ||
	fail "the refused land has no action-refused row"
[ -d "$root/sessions/$id/tree" ] || fail "the land removed the tree"

# 3. main moves elsewhere; a second commit, reviewed, is not a fast-forward.
git clone --quiet "$remote" "$clone"
printf 'elsewhere\n' >"$clone/elsewhere.txt"
git -C "$clone" -c core.autocrlf=false add elsewhere.txt
git -C "$clone" -c user.name="Ada Example" -c user.email=ada@example.com -c commit.gpgsign=false \
	-c core.hooksPath=/dev/null commit --quiet -m "Pushed from elsewhere"
git -C "$clone" push --quiet origin main
moved=$(git -C "$clone" rev-parse HEAD)
two=$(commit_tree two.txt "Add two.txt")
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "Review the last commit." >/dev/null
"$bf" session show "$id" | grep -q "^review: toolu_rew_agent at $two\$" || fail "no review at $two: $("$bf" session show "$id" | grep '^review:')"
set +e
refusal=$("$bf" session land "$id" 2>&1)
code=$?
set -e
printf '%s\n' "$refusal"
[ "$code" = 1 ] && [ "$refusal" = "error: land-not-fast-forward: Not landed: main moved by 1 commits since this session branched. Fetch origin, rebase this tree onto origin/main (the driver or the runner), review again, then push again." ] ||
	fail "the moved land exited $code: $refusal"
[ "$(git -C "$remote" rev-parse main)" = "$moved" ] || fail "the refused land moved the remote's main"

# 4. The end by a member with the land role falls back to push: the refusal first, then the
# push's line.
ended=$("$bf" session end "$id" --as 1001 --roles 10)
printf '%s\n' "$ended"
[ "$(head -n 1 <<<"$ended")" = "fallback: land-not-fast-forward: Not landed on main: main moved by 1 commits since this session branched. Fetch origin, rebase this tree onto origin/main (the driver or the runner), review again, then push again. Handed off as $branch instead." ] ||
	fail "session end has no fallback line first"
[ "$(tail -n 1 <<<"$ended")" = "pushed  origin  refs/sessions/$id:refs/heads/$branch  1  -  -" ] || fail "session end printed: $ended"
[ "$(git -C "$remote" rev-parse "refs/heads/$branch")" = "$two" ] || fail "$branch on the remote is not $two"
[ "$(git -C "$remote" rev-parse main)" = "$moved" ] || fail "the fallback moved the remote's main"
"$bf" --json audit --kind session-ended --session "$id" --all | grep -q '"outcome":"pushed"' || fail "session-ended is not pushed"

# 5. An end with no roles behind it (the host's, no --as, no --roles) on an unreviewed tip: the
# first refusal in gate order is land-not-allowed, and it falls back as well.
id=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture "$fixture")
[ "$id" = s0002 ] || fail "session start printed $id, want s0002"
branch=$(git -C "$root/sessions/$id/tree" symbolic-ref --short HEAD)
three=$(commit_tree three.txt "Add three.txt" "$id")
ended=$("$bf" session end "$id")
printf '%s\n' "$ended"
[ "$(head -n 1 <<<"$ended")" = "fallback: land-not-allowed: Not landed on main: landing on main needs a land role in this channel. Handed off as $branch instead." ] ||
	fail "the no-roles end has no land-not-allowed fallback line first"
[ "$(tail -n 1 <<<"$ended")" = "pushed  origin  refs/sessions/$id:refs/heads/$branch  1  -  -" ] || fail "session end printed: $ended"
[ "$(git -C "$remote" rev-parse "refs/heads/$branch")" = "$three" ] || fail "$branch on the remote is not $three"
[ "$(git -C "$remote" rev-parse main)" = "$moved" ] || fail "the no-roles end moved the remote's main"

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"
trap - EXIT

# 6. config check on a copy of the root whose channel 100 lost roles.land: the warning.
copy=$root/copy
mkdir -p "$copy"
cp -r "$root/config" "$copy/config"
sed -i '/^        "land": \["10"\],$/d' "$copy/config/projects/scratch.json"
warned=$(BROODFORGE_ROOT=$copy "$bf" config check)
printf '%s\n' "$warned"
grep -qx 'warning: scratch 100: handoff main with no roles.land; every session end there will fall back to push.' <<<"$warned" ||
	fail "config check on the copy has no roles.land warning for channel 100"
echo "land: ok"
