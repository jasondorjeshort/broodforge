#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# C38: every path under the config directory in an `strace -f -e trace=file` log, with the
# syscall, open flags and result, split into written and read (docs/design/07-spike-droplet.md, C38).
#
#   strace-cfg.sh <strace.log> <cfg>
#
# A line marked `!` is outside the expected sets: for written, outside the per-session list
# (projects, shell-snapshots, todos) and the credential's names (.credentials.json, .claude.json);
# for read, outside those, the loaded list (agents, commands, skills, plugins, CLAUDE.md) and
# settings.json. Limitation: a path opened relative to a directory fd (openat(3, "x")) shows no
# <cfg> prefix without strace -y and is not listed.

set -eu
[ $# -eq 2 ] || { echo "usage: strace-cfg.sh <strace.log> <cfg>" >&2; exit 2; }
log=$1
cfg=${2%/}

awk -v cfg="$cfg" '
BEGIN { W = "^(mkdir|mkdirat|unlink|unlinkat|rename|renameat|renameat2|rmdir|creat|link|linkat|symlink|symlinkat|chmod|fchmodat|chown|fchownat|lchown|truncate|utime|utimes|utimensat|setxattr|lsetxattr|removexattr|lremovexattr)$" }
{
  line = $0
  sub(/^[0-9]+ +/, "", line)               # strace -f pid prefix
  if (line ~ /^<\.\.\. /) next              # "<... openat resumed>" carries no path
  sc = line; sub(/\(.*/, "", sc)
  if (line ~ /<unfinished \.\.\.>/) res = "unfinished"
  else { n = split(line, parts, " = "); res = (n > 1 ? parts[n] : "?"); sub(/ \(.*/, "", res) }
  flags = "-"
  if (match(line, /O_[A-Z_|]+/)) flags = substr(line, RSTART, RLENGTH)
  rest = line
  while (match(rest, /"[^"]*"/)) {
    p = substr(rest, RSTART + 1, RLENGTH - 2)
    rest = substr(rest, RSTART + RLENGTH)
    if (p != cfg && index(p, cfg "/") != 1) continue
    rel = substr(p, length(cfg) + 2); if (rel == "") rel = "."
    top = rel; sub(/\/.*/, "", top)
    w = (sc ~ W) || (flags ~ /O_WRONLY|O_RDWR|O_CREAT|O_TRUNC|O_APPEND/)
    if (w) ok = (top ~ /^(projects|shell-snapshots|todos|\.credentials\.json.*|\.claude\.json.*)$/)
    else ok = (top ~ /^(\.|projects|shell-snapshots|todos|\.credentials\.json.*|\.claude\.json.*|agents|commands|skills|plugins|CLAUDE\.md|settings\.json)$/)
    key = (w ? "1written" : "2read") "\t" (ok ? " " : "!") "\t" rel "\t" sc " " flags " = " res
    if (!(key in seen)) { seen[key] = 1; print key }
  }
}' "$log" | sort | awk -F '\t' '
{ head = substr($1, 2); if (head != last) { print head ":"; last = head } print " " $2 " " $3 "  " $4 }
END { if (NR == 0) print "no path under the config directory in the log" }'
