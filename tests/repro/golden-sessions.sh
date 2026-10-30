#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Every golden session against the current build (05-debug-interface-discord.md, Golden
# sessions; P32), from the repository root, with no root and not under scratch run:
#
#   bash tests/repro/golden-sessions.sh
#
# One `discord render --golden <dir> --expect 'edits<=1/s'` per directory under
# tests/discord/golden/: `ok  <name>` when it matches, else `differs  <name>` followed by the
# render's records for each differing file and its error line. Exit 1 if any golden differs.
# The binary is bin/broodforge, or BF when set.
set -uo pipefail

bf=${BF:-bin/broodforge}
status=0
found=0
for dir in tests/discord/golden/*/; do
	dir=${dir%/}
	name=${dir##*/}
	found=1
	if out=$("$bf" discord render --golden "$dir" --expect 'edits<=1/s' 2>&1); then
		echo "ok  $name"
		continue
	fi
	status=1
	echo "differs  $name"
	# The records of a differing file (text columns `<file>  <line>  golden|now  <text>` and
	# `<file>  more  <n>`) and the error line; the posts themselves are left out.
	printf '%s\n' "$out" | grep -E '^(outbox\.jsonl|transcript-[^ ]+\.md)  |^error: ' || printf '%s\n' "$out"
done
if [ "$found" = 0 ]; then
	echo "golden-sessions: no golden under tests/discord/golden/" >&2
	exit 1
fi
exit "$status"
