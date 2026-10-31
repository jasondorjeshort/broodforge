// SPDX-License-Identifier: GPL-3.0-or-later

package git

import "time"

// SetFastForwardTimeout sets Fetch's fast-forward deadline for a test and returns the restore.
func SetFastForwardTimeout(d time.Duration) (restore func()) {
	old := fastForwardTimeout
	fastForwardTimeout = d
	return func() { fastForwardTimeout = old }
}
