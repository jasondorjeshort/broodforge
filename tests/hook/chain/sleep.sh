#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# PreToolUse child: sleeps past the ten-second budget when the call names sleep-past.
input=$(cat)
case $input in
*sleep-past*) sleep 12 ;;
esac
exit 0
