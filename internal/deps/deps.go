// SPDX-License-Identifier: GPL-3.0-or-later

// Package deps holds every ADR 0023 library in go.mod and go.sum until the package that owns it
// imports it, so no later pass fetches a module. Delete an import here in the pass that adds the
// real one; delete the package when it is empty. Nothing imports this package.
package deps

import (
	// discord's library (ADR 0023).
	_ "github.com/disgoorg/disgo"
	// store's library (ADR 0023).
	_ "zombiezen.com/go/sqlite"
	// spawn's and control's module; cpu is held because it builds on every host, whereas
	// x/sys/windows builds only on Windows. The module, not the package, is what is pinned.
	_ "golang.org/x/sys/cpu"
)
