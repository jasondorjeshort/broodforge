#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
exec "$(dirname "$0")/../bin/broodforge" git-askpass "$@"
