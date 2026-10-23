#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# PostToolUse child: exit 2 with a stderr line, which hook post-tool-use passes on as context.
cat >/dev/null
echo "chain fixture: stderr from post-exit2.sh" >&2
exit 2
