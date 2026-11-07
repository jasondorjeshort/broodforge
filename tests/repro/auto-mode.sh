#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# The engine's auto permission mode as the third session mode, P39
# (docs/design/08-implementation-plan-auto-mode.md, Debug surface), on a scratch root, from the
# repository root:
#
#   MSYS_NO_PATHCONV=1 bin/broodforge scratch run -- bash tests/repro/auto-mode.sh
#
# Channel 100 is given "mode": "auto" (config check clean; another word refused naming the three).
# hook try passes an ask-cell Write; hook cases runs the shipped cases. s0001 replays the
# hand-written fixtures/engine/auto-refused (shape B): the request's permissionMode and args carry
# auto, the refused Write is decided by engine with no hook-bypassed row; `session mode s0001
# prompt` puts the next process on default; on the dry-run Discord `/session mode auto` and the
# mode menu's pick post the switch and edit the header. s0002 takes the channel's auto and
# replays fixtures/engine/auto-prompt (shape A): the engine's ask waits for an answer with its
# reason in the prompt, and the driver's allow resumes it. s0003, policy mode on auto-refused,
# is the engine-mode-mismatch row. Last, `engine run --real` with a stand-in engine shows the
# spawn's settings file with "onFailure": "block" on the PreToolUse entry alone; the stand-in is
# STUB_ENGINE when set, else the cli package's test binary built with go (go on PATH). The bot's
# output is serve.log under the root, kept on a failure.
set -euo pipefail

bf=${BF:-bin/broodforge}
: "${BROODFORGE_ROOT:?run this under broodforge scratch run}"
root=${BROODFORGE_ROOT//\\//}
log=$root/serve.log
project=$root/config/projects/scratch.json
work=$root/auto-mode
mkdir -p "$work"

fail() {
	echo "auto-mode: $*" >&2
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

# 1. Configuration: channel 100's mode is auto; any other word is refused naming the three.
grep -q '"100": {' "$project" || fail "the scratch project has no channel 100"
sed -i '/"100": {/,/"effort"/ s/"mode": "prompt",/"mode": "auto",/' "$project"
[ "$(grep -c '"mode": "auto"' "$project")" = 1 ] || fail "channel 100 did not get mode auto"
"$bf" config check >/dev/null || fail "config check refused mode auto"
copy=$work/copy
mkdir -p "$copy"
cp -r "$root/config" "$copy/config"
sed -i 's/"mode": "auto"/"mode": "acts-freely"/' "$copy/config/projects/scratch.json"
set +e
refused=$(BROODFORGE_ROOT=$copy "$bf" config check 2>&1)
code=$?
set -e
printf '%s\n' "$refused"
[ "$code" = 1 ] || fail "config check on mode acts-freely exited $code, want 1"
grep -q 'channels.100.mode: "acts-freely", want prompt or policy or auto' <<<"$refused" ||
	fail "config check does not name the accepted modes"

# 2. The hook: an ask-cell Write under the tree is a pass in auto mode; the shipped cases.
tree=$work/tree
mkdir -p "$tree"
pass=$("$bf" hook try --mode auto --reach code --tool Write --input "{\"file_path\":\"$tree/a.go\"}" \
	--tree "$tree" --transcript-path "$work/cfg/projects/x/s.jsonl" --expect pass)
printf '%s\n' "$pass"
grep -q "^pass  Broodforge: inside this channel's reach; the engine's own check decides" <<<"$pass" ||
	fail "hook try printed no pass with its reason"
# The chain cases run their children through Git Bash on a host with drive letters: name the
# usual install where the environment does not, as internal/cli's gitBashEnv does.
gitbash="C:/Program Files/Git/bin/bash.exe"
if [ -z "${CLAUDE_CODE_GIT_BASH_PATH:-}" ] && [ -f "$gitbash" ]; then
	export CLAUDE_CODE_GIT_BASH_PATH=$gitbash
fi
cases=$("$bf" hook cases tests/hook/cases.jsonl) || fail "hook cases failed"
if grep -v -e '^ok  ' -e '^skip  ' <<<"$cases"; then
	fail "hook cases has a line that is neither ok nor skip"
fi
grep -q '^ok  [0-9]*  Write  auto  code  pass  pass  ' <<<"$cases" || fail "hook cases has no auto pass case"

"$bf" serve >"$log" 2>&1 &
serve=$!
trap 'kill "$serve" 2>/dev/null || true' EXIT
"$bf" status --wait 10 --expect ok >/dev/null || fail "the bot did not answer (serve.log: $log)"
"$bf" register 1001 --name "Ada Example" --email ada@example.com >/dev/null

# 3. Shape B: the engine's check refuses a hook-passed Write with no prompt.
id=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode auto --fixture auto-refused)
[ "$id" = s0001 ] || fail "session start printed $id, want s0001"
show=$("$bf" session show "$id")
printf '%s\n' "$show"
grep -q "^session: $id  starting\|^session: $id  [a-z-]*  100  scratch  1001  auto  code  " <<<"$show" ||
	fail "session show has no auto in the mode column"
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "write hello.go" >/dev/null
recv=$root/sessions/$id/fake/received-1.json
[ -f "$recv" ] || fail "no $recv"
grep -q '"permissionMode": "auto",' "$recv" || fail "received-1.json has no permissionMode auto"
tr -d ' \n\r' <"$recv" | grep -q '"args":\["--permission-mode","auto",' || fail "received-1.json's args have no --permission-mode auto"
grep -q '"name": "BROODFORGE_MODE",' -A1 "$recv" && grep -A1 '"name": "BROODFORGE_MODE",' "$recv" | grep -q '"value": "auto"' ||
	fail "received-1.json's hookEnv has no BROODFORGE_MODE auto"
tail=$("$bf" session tail "$id" --all)
printf '%s\n' "$tail"
for want in \
	'  permission-decision  deny \[Write\] hello\.go (engine)$' \
	"  tool-result  \[Write\] hello\.go -> refused by the engine's check: Permission for this action"; do
	grep -q -- "$want" <<<"$tail" || fail "session tail has no line matching: $want"
done
if grep -q '  internal-error  ' <<<"$tail"; then
	fail "an auto session asking for auto got an internal-error"
fi
[ -z "$("$bf" audit --kind hook-bypassed --session "$id")" ] || fail "audit holds a hook-bypassed row for $id"
"$bf" --json audit --kind permission-decision --session "$id" | grep -q '"decidedBy":"engine"' ||
	fail "audit has no permission-decision by engine"

# The switch back: the next process asks for default.
"$bf" session mode "$id" prompt --as 1001 | grep -q '^mode-changed  2  prompt$' || fail "session mode printed no mode-changed"
"$bf" session send "$id" --as 1001 --wait --expect turn-ended "and now?" >/dev/null
grep -q '"permissionMode": "default",' "$root/sessions/$id/fake/received-2.json" || fail "received-2.json has no permissionMode default"
tr -d ' \n\r' <"$root/sessions/$id/fake/received-2.json" | grep -q '"args":\["--permission-mode","default",' ||
	fail "received-2.json's args have no --permission-mode default"

# On the dry-run Discord: a bad value refused naming every mode, the typed switch, then the
# menu's pick; the post and the header edit.
set +e
bad=$("$bf" discord inject command --channel "$id" --user 1001 --roles 10 "/session mode sideways" 2>&1)
code=$?
set -e
[ "$code" = 1 ] && [ "$bad" = "error: bad-args: /session mode takes asks-first, acts-freely or auto" ] ||
	fail "/session mode sideways: exit $code, $bad"
"$bf" discord inject command --channel "$id" --user 1001 --roles 10 "/session mode auto" >/dev/null
outbox_has "$id" '  create  m[0-9]*  [0-9]*  From the next turn: auto (within reach)\. Broodforge still decides' ||
	fail "session outbox has no third-mode post after /session mode auto"
"$bf" discord inject click --custom-id "bf:$id:mode" --value auto --user 1001 --roles 10 >/dev/null
for _ in $(seq 1 40); do
	[ "$("$bf" session outbox "$id" --all | grep -c 'From the next turn: auto (within reach)\.')" -ge 2 ] && break
	sleep 0.25
done
[ "$("$bf" session outbox "$id" --all | grep -c 'From the next turn: auto (within reach)\.')" -ge 2 ] ||
	fail "the menu's pick posted no second switch"
outbox=$("$bf" --json session outbox "$id" --all)
grep -q '"label":"auto (within reach)","value":"auto","default":true' <<<"$outbox" ||
	fail "no header edit with the mode menu's third option marked"
"$bf" session show "$id" | grep -q "^session: $id  [a-z-]*  100  scratch  1001  auto  " || fail "the switch did not set auto"

# 4. Shape A: the engine's check asks; the channel's mode is auto with no --mode given.
id2=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --fixture auto-prompt)
[ "$id2" = s0002 ] || fail "session start printed $id2, want s0002"
"$bf" session send "$id2" --as 1001 --wait --expect waiting-for-answer "run the installer" >/dev/null
"$bf" session show "$id2" | grep -q '^pending: toolu_auto_bash  Bash  curl ' || fail "session show has no pending Bash call"
# The reason is the prompt's last line, so the --json body is read (text mode shows a first line).
outbox_has "$id2" "The engine's check asked: the fixture asks: a download piped into a shell" --json ||
	fail "the prompt has no line with the engine's reason"
"$bf" session answer "$id2" allow --as 1001 --roles 10 --wait --expect turn-ended >/dev/null
tail2=$("$bf" session tail "$id2" --all)
printf '%s\n' "$tail2"
for want in \
	'  permission-decision  deferred \[Bash\] curl .* (engine)$' \
	'  waiting-for-answer  toolu_auto_bash \[Bash\] curl ' \
	'  permission-decision  allow \[Bash\] curl .* (participant)$' \
	'  turn-ended  end_turn$'; do
	grep -q -- "$want" <<<"$tail2" || fail "session tail $id2 has no line matching: $want"
done
grep -q '"permissionMode": "auto",' "$root/sessions/$id2/fake/received-2.json" || fail "the resume did not ask for auto"
grep -q '"toolUseId": "toolu_auto_bash",' "$root/sessions/$id2/fake/received-2.json" || fail "the resume carries no answer for the call"

# 5. A process the engine runs in another mode than asked: engine-mode-mismatch, the turn goes on.
id3=$("$bf" session start --channel 100 --as 1001 --roles 10 --credential cred-scratch --mode policy --fixture auto-refused)
"$bf" session send "$id3" --as 1001 --wait --expect turn-ended "write hello.go" >/dev/null
"$bf" session tail "$id3" --all | grep -q '  internal-error  engine mode: wanted default, got auto$' ||
	fail "session tail $id3 has no engine-mode internal-error"
"$bf" --json audit --kind engine-mode-mismatch --session "$id3" | grep -q '"details":{"got":"auto","turn":"1","wanted":"default"}' ||
	fail "audit has no engine-mode-mismatch row for $id3"
"$bf" session tail "$id3" --all | grep -q ' -> refused by the engine without Broodforge'"'"'s check: ' ||
	fail "outside auto mode the refusal is not the hook bypassed"

# engine run --real takes a registered user's credential: the stand-in never reads the value.
cred=$(printf 'stand-in' | "$bf" credential add --kind api-key --user 1001 --label stand-in)
"$bf" stop >/dev/null

# 6. The spawn's settings file: onFailure block on the PreToolUse entry, not on PostToolUse.
stub=$work/stub
mkdir -p "$stub"
exe=claude
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=claude.exe ;; esac
if [ -n "${STUB_ENGINE:-}" ]; then
	cp "$STUB_ENGINE" "$stub/$exe"
else
	go test -c -o "$stub/$exe" ./internal/cli
fi
# The instance's engine is the stand-in, at the version it reports.
instance=$root/config/instance.json
grep -q '"path": "",' "$instance" && grep -q '"version": "",' "$instance" ||
	fail "the scratch instance has no empty engine.path and engine.version"
sed -i -e "0,/\"path\": \"\",/s||\"path\": \"$stub/$exe\",|" -e '0,/"version": "",/s//"version": "2.1.296",/' "$instance"
mkdir -p "$work/cwd"
fixture=$(pwd)/fixtures/engine/auto-refused/turn-1.jsonl
BFTEST_STANDIN=1 BFTEST_STANDIN_FIXTURE=$fixture \
	"$bf" engine run --real --credential "$cred" --cwd "$work/cwd" --prompt "x" >/dev/null ||
	fail "engine run --real with the stand-in failed"
settings=$root/sessions/engine-run/home/.claude/settings.json
[ -f "$settings" ] || fail "no settings file at $settings"
cat "$settings"
node -e '
const s = JSON.parse(require("fs").readFileSync(process.argv[1], "utf8")).hooks;
if (s.PreToolUse[0].hooks[0].onFailure !== "block") process.exit(1);
if ("onFailure" in s.PostToolUse[0].hooks[0]) process.exit(2);' "$(command -v cygpath >/dev/null && cygpath -m "$settings" || echo "$settings")" ||
	fail "the settings file does not carry onFailure block on PreToolUse alone"
# engine run's standalone session directory is no session of the root's: scratch run's
# cleanup refuses it, so it goes once read.
rm -rf "$root/sessions/engine-run"

echo "auto-mode: ok"
