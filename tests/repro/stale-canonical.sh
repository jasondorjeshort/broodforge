#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# The fetch at session start (ADR 0022 companion, Clone at session start, step 0;
# 05-debug-interface-git.md, The fetch at session start; P30), on a scratch root, from the
# repository root:
#
#   bin/broodforge scratch run -- bash tests/repro/stale-canonical.sh
#
# BF names another binary than bin/broodforge. A throwaway clone of <root>/remote/scratch.git
# pushes a commit to main; a start on channel 100 then branches from it (`session show`'s
# worktree, the canonical's main equal to the remote's, `canonical` `fetched`). An empty commit in
# the canonical gives an `ahead 1` start with no stale line in the venue. With the bare remote
# renamed away a start goes on, `canonical` `stale: ...` and the venue's line; with it back and a
# second commit pushed from the clone the two have diverged and the start is refused
# canonical-diverged, audited with reason canonical. Each serve's output is kept under the root
# (serve.log, serve2.log), which scratch run keeps on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log
log2=$root/serve2.log
canon=$root/projects/scratch
remote=$root/remote/scratch.git
clone=$root/clone

fail() {
	echo "stale-canonical: $*" >&2
	exit 1
}

# commit_in <dir> <message> [args]: a commit as the scratch driver, no hooks, no signing.
commit_in() {
	git -C "$1" -c user.name="Ada Example" -c user.email=ada@example.com -c commit.gpgsign=false \
		-c core.hooksPath=/dev/null commit --quiet -m "$2" "${@:3}"
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

# started_with <session> <pattern>: its session-started row's canonical detail, from its opening quote.
started_with() {
	"$bf" --json audit --kind session-started --session "$1" --all | grep -q -- "\"canonical\":\"$2" ||
		fail "$1's session-started has no canonical matching $2: $("$bf" --json audit --kind session-started --session "$1" --all)"
}

start() {
	"$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture basic
}

# The remote moves on: a commit pushed to main from a throwaway clone.
git clone --quiet "$remote" "$clone"
printf 'pushed\n' >"$clone/pushed.txt"
git -C "$clone" -c core.autocrlf=false add pushed.txt
commit_in "$clone" "Pushed from elsewhere"
git -C "$clone" push --quiet origin main
pushed=$(git -C "$clone" rev-parse HEAD)
[ "$(git -C "$canon" rev-parse main)" != "$pushed" ] || fail "the canonical already has the pushed commit"

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null
"$bf" --json status | grep -q '"projects":\[{"id":"scratch","canonical":{"branch":"main","commit":"","fetched":null,"ahead":0,"error":null}}\]' ||
	fail "status --json before a start: $("$bf" --json status)"

# Behind: the start fast-forwards the canonical and branches from the pushed commit.
id=$(start)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"
wt=$("$bf" session show "$id" | grep '^worktree: ') || fail "session show has no worktree line"
read -r _ tree branch _ <<<"$wt"
[ "$(git -C "$tree" rev-parse "$branch")" = "$pushed" ] || fail "$branch in $tree is not on the pushed commit $pushed"
[ "$(git -C "$canon" rev-parse main)" = "$(git -C "$remote" rev-parse main)" ] || fail "the canonical's main is not the remote's"
git -C "$canon" log --oneline -1 main
started_with "$id" 'fetched"'
"$bf" --json status | grep -q "\"projects\":\[{\"id\":\"scratch\",\"canonical\":{\"branch\":\"main\",\"commit\":\"$pushed\",\"fetched\":\"20[0-9T:.-]*Z\",\"ahead\":0,\"error\":null}}\]" ||
	fail "status --json after the fetch: $("$bf" --json status)"

# Ahead: a commit in the canonical itself, not pushed, is left alone and the start goes on.
commit_in "$canon" "Local, not pushed" --allow-empty
id=$(start)
[ "$id" = s0002 ] || fail "session start printed $id, want s0002"
started_with "$id" 'ahead 1"'
outbox_has "$id" "  create  m[0-9]*  [0-9]*  Session $id on Scratch, branch " || fail "$id's venue has no header"
if "$bf" --json session outbox "$id" --all | grep -q 'could not be fetched'; then fail "$id's venue has a stale line"; fi

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve.log: $log)"

# Stale: the remote gone, the start goes on from the local branch and the venue says so.
mv "$remote" "$remote.away"
"$bf" serve >"$log2" 2>&1 &
serve=$!
"$bf" status --wait 10 --expect ok >/dev/null || fail "the second bot did not answer (serve2.log: $log2)"
id=$(start)
[ "$id" = s0003 ] || fail "session start printed $id, want s0003"
started_with "$id" "stale: [^\"]"
"$bf" --json audit --kind session-started --session "$id" --all
# The header's second line, after its first (04, canonical-stale).
outbox_has "$id" '"body":"Session '"$id"' on Scratch, branch [^\\]*\\nThe project.s main could not be fetched (' --json ||
	fail "$id's header has no stale line after its first: $("$bf" --json session outbox "$id" --all)"
"$bf" --json session outbox "$id" --all | grep -o 'The project.s main could not be fetched[^"]*' | head -n 1
"$bf" --json status | grep -q '"error":"[^"]' || fail "status --json has no fetch error: $("$bf" --json status)"

# Diverged: the remote back with a second commit, the canonical holding its own; refused.
mv "$remote.away" "$remote"
commit_in "$clone" "Pushed again" --allow-empty
git -C "$clone" push --quiet origin main
set +e
refusal=$(start 2>&1)
code=$?
set -e
[ "$code" = 1 ] || fail "the diverged start exited $code: $refusal"
grep -q "^error: canonical-diverged: $canon main is at [0-9a-f]\{40\}, origin at $(git -C "$remote" rev-parse main); neither contains the other$" <<<"$refusal" ||
	fail "the diverged start printed: $refusal"
"$bf" --json audit --kind action-refused --session s0004 --all | grep -q '"action":"start",.*"reason":"canonical"' ||
	fail "audit has no action-refused with reason canonical: $("$bf" --json audit --kind action-refused --all)"
"$bf" session list --state ended --all | grep -q '^s0004  ended  ' || fail "s0004 is not ended"
[ ! -e "$root/sessions/s0004/tree" ] || fail "the diverged start cloned a tree"

"$bf" stop >/dev/null
wait "$serve" || fail "serve exited $? (serve2.log: $log2)"
trap - EXIT
echo "stale-canonical: ok"
