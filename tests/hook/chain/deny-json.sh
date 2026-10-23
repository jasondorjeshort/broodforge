#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# PreToolUse child: a JSON deny when the call names deny-json.
input=$(cat)
case $input in
*deny-json*)
	printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"the chain fixture refuses deny-json"}}'
	;;
esac
exit 0
