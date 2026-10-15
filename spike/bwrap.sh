#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# The spike's bwrap route on the droplet (docs/design/07-spike-droplet.md, "The bwrap route"),
# built from the companion's bind block (docs/design/decisions/0022-linux-spawn-and-handoff.md,
# "The engine spawn") with <bf> bound in place of Broodforge's install. Runs as broodforge.
#
#   bwrap.sh [--print] --reach code|read-only --cred sub|key [--strace <log>] -- <command...>
#       the session sandbox; <command> runs under engine-entry, cwd <sess>/tree, HOME <sess>/home
#   bwrap.sh [--print] --maintenance <dir> -- <command...>
#       system, Node and <bf> binds, <canon> read-only when it exists, <dir> read-write, HOME=/tmp;
#       a first word `engine-entry` is <bf>/engine-entry
#   bwrap.sh --check
#       namei -m over every bind source as broodforge-engine, then the full session spawn with
#       `engine-entry --check`; a kept name that does not arrive prints the sudoers block, exit 1
#   --print   print the command, one argument per line, instead of running it
#
# Names (override by environment): BF_ROOT (/srv/broodforge), BF_SESSION (spike),
# BF_ENGINE_DIR (/opt/broodforge/engine), BF_NODE_DIR (/opt/broodforge/node).
# The credential and git identity reach the engine through sudo's env_keep, never --setenv.

set -eu

BF=$(cd "$(dirname "$0")" && pwd)
# The empty directory bound over the loaded lists is not in git (it must hold nothing); make it.
mkdir -p "$BF/share/empty/dir"
ROOT=${BF_ROOT:-/srv/broodforge}
SESSION=${BF_SESSION:-spike}
SESS=$ROOT/sessions/$SESSION
CFG=$ROOT/harness/spike
CANON=$ROOT/projects/spike
SETTINGS=$ROOT/settings/spike.json
ENGINE=${BF_ENGINE_DIR:-/opt/broodforge/engine}
NODE=${BF_NODE_DIR:-/opt/broodforge/node}
SUDO="sudo -n -u broodforge-engine /usr/bin/bwrap"

usage() { sed -n '7,17p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 2; }

# Quote one argument for eval.
q() { printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"; }
ARGS=
add() { for a in "$@"; do ARGS="$ARGS $(q "$a")"; done; }

system_binds() {
  add --ro-bind /usr /usr --symlink usr/lib /lib --symlink usr/lib64 /lib64 \
      --symlink usr/bin /bin --symlink usr/sbin /sbin --ro-bind /etc /etc
  # The tmpfs needs an existing mount point inside the read-only /etc.
  if [ -d /etc/broodforge ]; then add --tmpfs /etc/broodforge; fi
  add --ro-bind-try /run/systemd/resolve /run/systemd/resolve
}

tail_opts() {
  add --tmpfs /tmp --proc /proc --dev /dev \
      --unshare-pid --unshare-ipc --unshare-uts --die-with-parent --new-session
}

session_binds() { # $1 reach, $2 cred
  system_binds
  add --ro-bind "$ENGINE" "$ENGINE" --ro-bind "$NODE" "$NODE" --ro-bind "$BF" "$BF" \
      --ro-bind "$CANON" "$CANON" \
      --bind "$SESS/home" "$SESS/home"
  if [ "$1" = read-only ]; then add --ro-bind "$SESS/tree" "$SESS/tree"; else add --bind "$SESS/tree" "$SESS/tree"; fi
  if [ "$2" = sub ]; then
    ECFG=$CFG
    add --bind "$CFG" "$CFG" \
        --bind "$SESS/cfg/projects" "$CFG/projects" \
        --bind "$SESS/cfg/shell-snapshots" "$CFG/shell-snapshots" \
        --bind "$SESS/cfg/todos" "$CFG/todos" \
        --ro-bind "$BF/share/empty/dir" "$CFG/agents" \
        --ro-bind "$BF/share/empty/dir" "$CFG/commands" \
        --ro-bind "$BF/share/empty/dir" "$CFG/skills" \
        --ro-bind "$BF/share/empty/dir" "$CFG/plugins" \
        --ro-bind "$BF/share/empty/CLAUDE.md" "$CFG/CLAUDE.md"
  else
    ECFG=$SESS/home/.claude
  fi
  add --ro-bind "$SETTINGS" "$ECFG/settings.json"
  tail_opts
  add --setenv HOME "$SESS/home" --setenv PATH "$ENGINE:$NODE/bin:/usr/bin:/bin" \
      --setenv DISABLE_AUTOUPDATER 1 --chdir "$SESS/tree"
}

credential_env() { # $1 cred
  if [ "$1" = sub ]; then
    unset ANTHROPIC_API_KEY || true
    CLAUDE_CONFIG_DIR=${CLAUDE_CONFIG_DIR:-$CFG}
    export CLAUDE_CONFIG_DIR
  else
    unset CLAUDE_CONFIG_DIR || true
    [ -n "${ANTHROPIC_API_KEY:-}" ] || { echo "bwrap.sh: --cred key needs ANTHROPIC_API_KEY in this environment" >&2; exit 1; }
  fi
}

finish() { # run or print "$SUDO $ARGS"
  if [ "$PRINT" = 1 ]; then
    eval "set -- $SUDO $ARGS"
    for a in "$@"; do printf '%s\n' "$a"; done
    exit 0
  fi
  eval "exec $SUDO $ARGS"
}

PRINT=0 REACH=code CRED=sub STRACE= MAINT= CHECK=0
while [ $# -gt 0 ]; do
  case $1 in
    --print) PRINT=1; shift ;;
    --reach) REACH=$2; shift 2 ;;
    --cred) CRED=$2; shift 2 ;;
    --strace) STRACE=$2; shift 2 ;;
    --maintenance) MAINT=$2; shift 2 ;;
    --check) CHECK=1; shift ;;
    --) shift; break ;;
    *) usage ;;
  esac
done
case $REACH in code|read-only) ;; *) usage ;; esac
case $CRED in sub|key) ;; *) usage ;; esac

if [ "$CHECK" = 1 ]; then
  echo "bwrap.sh: namei -m over every bind source, as broodforge-engine"
  $SUDO --ro-bind / / --proc /proc --dev /dev -- /usr/bin/namei -m \
    /usr /etc "$ENGINE" "$NODE" "$BF" "$CANON" "$SESS/home" "$SESS/tree" "$CFG" \
    "$SESS/cfg/projects" "$SESS/cfg/shell-snapshots" "$SESS/cfg/todos" \
    "$BF/share/empty/dir" "$BF/share/empty/CLAUDE.md" "$SETTINGS" ||
    { echo "bwrap.sh: a bind source is not traversable by broodforge-engine (above)" >&2; exit 1; }
  # Every kept name gets a placeholder unless set; only names are compared.
  KEPT=$(sh "$BF/engine-entry" --kept)
  for n in $KEPT; do eval "[ -n \"\${$n+x}\" ] || export $n=check"; done
  CLAUDE_CONFIG_DIR=${CLAUDE_CONFIG_DIR:-$CFG}; export CLAUDE_CONFIG_DIR
  session_binds code sub
  add -- /bin/sh "$BF/engine-entry" --check
  SEEN=$(eval "$SUDO $ARGS")
  echo "bwrap.sh: kept names seen inside:"
  printf '%s\n' "$SEEN" | sed 's/^/  /'
  MISSING=
  for n in $KEPT; do printf '%s\n' "$SEEN" | grep -qx "$n" || MISSING="$MISSING $n"; done
  if [ -n "$MISSING" ]; then
    echo "bwrap.sh: missing:$MISSING. Sudoers block to install:" >&2
    echo "Defaults:broodforge env_reset" >&2
    echo "Defaults:broodforge env_keep += \"$(echo $KEPT)\"" >&2
    echo "broodforge ALL=(broodforge-engine) NOPASSWD: /usr/bin/bwrap" >&2
    echo "broodforge ALL=(broodforge-engine) NOPASSWD: /usr/bin/kill" >&2
    exit 1
  fi
  echo "bwrap.sh: check passed"
  exit 0
fi

[ $# -ge 1 ] || usage

if [ -n "$MAINT" ]; then
  system_binds
  add --ro-bind "$NODE" "$NODE" --ro-bind "$BF" "$BF"
  if [ -d "$CANON" ]; then add --ro-bind "$CANON" "$CANON"; fi
  add --bind "$MAINT" "$MAINT"
  tail_opts
  add --setenv HOME /tmp --setenv PATH "$NODE/bin:/usr/bin:/bin" --
  if [ "$1" = engine-entry ]; then shift; add /bin/sh "$BF/engine-entry"; fi
  add "$@"
  finish
fi

credential_env "$CRED"
session_binds "$REACH" "$CRED"
add -- /bin/sh "$BF/engine-entry"
if [ -n "$STRACE" ]; then add --strace "$STRACE"; fi
add -- "$@"
finish
