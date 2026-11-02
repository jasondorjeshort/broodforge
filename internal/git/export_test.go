// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"time"
)

// FollowTracking is followTracking, the land's step 5 write of the tracking ref.
func FollowTracking(ctx context.Context, s Session, sha string) error {
	return followTracking(ctx, s, sha)
}

// SetFastForwardTimeout sets Fetch's fast-forward deadline for a test and returns the restore.
func SetFastForwardTimeout(d time.Duration) (restore func()) {
	old := fastForwardTimeout
	fastForwardTimeout = d
	return func() { fastForwardTimeout = old }
}
