// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

// LinuxSudoBwrap is the droplet backend of ADR 0022 and its companion (sudo into bubblewrap,
// the bind list, engine-entry). It is held until the droplet rows of the spike findings land
// (08-implementation-plan.md, P14): every operation refuses with preflight-backend: held.
type LinuxSudoBwrap struct{}

var _ Backend = LinuxSudoBwrap{}

func held() *Error { return &Error{CodeBackendHeld, "held"} }

// Name is the configuration's name for the backend.
func (LinuxSudoBwrap) Name() string { return NameLinuxSudoBwrap }

// Start refuses: held.
func (LinuxSudoBwrap) Start(Spec) (*Process, error) { return nil, held() }

// HarnessCreate refuses: held.
func (LinuxSudoBwrap) HarnessCreate(string) error { return held() }

// HarnessRemove refuses: held.
func (LinuxSudoBwrap) HarnessRemove(string) error { return held() }

// MCPConfig is unknown until the backend is built.
func (LinuxSudoBwrap) MCPConfig() string { return "" }

// Preflight is one failing check: held.
func (LinuxSudoBwrap) Preflight() []Check {
	return []Check{{Name: "backend", Detail: "held"}}
}

// WorktreeCreate refuses: held.
func (LinuxSudoBwrap) WorktreeCreate(Worktree) error { return held() }

// WorktreeBundle refuses: held.
func (LinuxSudoBwrap) WorktreeBundle(Worktree) (Bundle, error) { return Bundle{}, held() }

// WorktreeRemove refuses: held.
func (LinuxSudoBwrap) WorktreeRemove(string) error { return held() }

// HooksDir is unknown until the backend is built.
func (LinuxSudoBwrap) HooksDir() string { return "" }

// Askpass is unknown until the backend is built.
func (LinuxSudoBwrap) Askpass() string { return "" }
