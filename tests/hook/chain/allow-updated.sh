#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# PreToolUse child: allow with an updatedInput and a context when the call names rewrite-me;
# the chain takes none of it and logs the field names.
input=$(cat)
case $input in
*rewrite-me*)
	printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","updatedInput":{"command":"echo rewritten"},"additionalContext":"dropped"}}'
	;;
esac
exit 0
