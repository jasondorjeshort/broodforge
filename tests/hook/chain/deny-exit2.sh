#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# PreToolUse child: exit 2 when the call names deny-exit2; the chain takes the first stderr line.
input=$(cat)
case $input in
*deny-exit2*)
	echo "the chain fixture refuses deny-exit2" >&2
	echo "a second line the chain drops" >&2
	exit 2
	;;
esac
exit 0
