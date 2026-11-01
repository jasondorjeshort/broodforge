// SPDX-License-Identifier: GPL-3.0-or-later

package git_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/git"
)

func TestRefreshWaitsOnTheBundleLock(t *testing.T) {
	f := newFixture(t)
	unlock, err := git.LockBundle(context.Background(), f.sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	var ge *git.Error
	if _, err := git.Refresh(ctx, f.backend, f.sess); !errors.As(err, &ge) || !strings.Contains(ge.Message, "bundle lock") {
		t.Errorf("Refresh under a held bundle lock: %v, want the bundle lock's refusal", err)
	}
}
