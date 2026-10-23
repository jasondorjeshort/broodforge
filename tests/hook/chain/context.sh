#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# PostToolUse child: prints an additionalContext, which hook post-tool-use passes through.
cat >/dev/null
printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"chain fixture: context from context.sh"}}'
