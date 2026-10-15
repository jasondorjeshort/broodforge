#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# Sets the spike hook's mode on the droplet (docs/design/07-spike-droplet.md, "Hook mode and log").
# <sess>/home is engine-owned 0750, so broodforge writes <sess>/home/spike-hook.json through the
# sandbox, the only route that can. On the PC use `node hook.js mode ...` instead.
#
#   hookmode.sh allow|deny|defer|exit1|exit2|sleep [<tool_use_id>:allow|deny]

set -eu
BF=$(cd "$(dirname "$0")" && pwd)
SESS=${BF_ROOT:-/srv/broodforge}/sessions/${BF_SESSION:-spike}

usage() { echo "usage: hookmode.sh allow|deny|defer|exit1|exit2|sleep [<tool_use_id>:allow|deny]" >&2; exit 2; }
[ $# -ge 1 ] && [ $# -le 2 ] || usage
case $1 in allow|deny|defer|exit1|exit2|sleep) ;; *) usage ;; esac
json="{\"mode\":\"$1\"}"
if [ $# -eq 2 ]; then
  printf '%s\n' "$2" | grep -Eqx '[A-Za-z0-9_-]+:(allow|deny)' || usage
  json="{\"mode\":\"$1\",\"answer\":\"$2\"}"
fi
sh "$BF/bwrap.sh" --reach code -- sh -c 'printf "%s\n" "$1" > "$2"' sh "$json" "$SESS/home/spike-hook.json"
echo "hookmode.sh: $SESS/home/spike-hook.json: $json"
