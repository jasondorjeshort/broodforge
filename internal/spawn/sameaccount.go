// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SameAccount is the same-account backend (ADR 0022: Windows, and scratch runs on both
// hosts): the engine runs as the bot's own account through os/exec, with nothing bound and
// nothing hidden. Its zero value does the harness directory operations only.
type SameAccount struct {
	c       Config
	install string
	binary  string
}

// NewSameAccount resolves the install directory and the hook's binary.
func NewSameAccount(c Config) (*SameAccount, error) {
	s := &SameAccount{c: c, install: c.Install, binary: c.Binary}
	if s.binary == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, err
		}
		s.binary = exe
	}
	if s.install == "" {
		s.install = filepath.Dir(filepath.Dir(s.binary))
	}
	if s.c.InterruptTimeout <= 0 {
		s.c.InterruptTimeout = DefaultInterrupt
	}
	return s, nil
}

var _ Backend = (*SameAccount)(nil)

// Name is the configuration's name for the backend.
func (s *SameAccount) Name() string { return NameSameAccount }

// MCPConfig is <bf>/share/empty/mcp.json.
func (s *SameAccount) MCPConfig() string {
	return filepath.ToSlash(filepath.Join(EmptyShare(s.install), "mcp.json"))
}

// Start writes the settings template to the delivery's settings path and starts the engine
// with the explicit environment: the parent's minus the stripped names, then spec's variables,
// then the backend's own, which no spec can override.
func (s *SameAccount) Start(spec Spec) (*Process, error) {
	if s.c.EnginePath == "" {
		return nil, &Error{CodeSpawnFailed, "engine.path is not set"}
	}
	path, err := SettingsPath(spec.Delivery, spec.Home)
	if err != nil {
		return nil, err
	}
	own := []Var{{"CLAUDE_CODE_SUBPROCESS_ENV_SCRUB", "1"}, {"DISABLE_AUTOUPDATER", "1"}}
	switch spec.Delivery.Kind {
	case DeliverConfigDir:
		own = append(own, Var{"CLAUDE_CONFIG_DIR", filepath.ToSlash(spec.Delivery.ConfigDir)})
	case DeliverAPIKey:
		if len(spec.Delivery.APIKey) == 0 {
			return nil, &Error{CodeBadDelivery, "an api-key delivery carries no key"}
		}
		// The same account has the owner's ~/.claude, which the engine would read without a
		// config directory (the spike refused --cred key on the PC for that reason), so an API
		// key also names <home>/.claude, the settings path above.
		own = append(own, Var{"ANTHROPIC_API_KEY", string(spec.Delivery.APIKey)},
			Var{"CLAUDE_CONFIG_DIR", filepath.ToSlash(filepath.Dir(path))})
	}
	if s.c.GitBashPath != "" {
		own = append(own, Var{"CLAUDE_CODE_GIT_BASH_PATH", s.c.GitBashPath})
	}
	if spec.Home != "" {
		if err := os.MkdirAll(spec.Home, 0o750); err != nil {
			return nil, &Error{CodeSpawnFailed, "session home: " + err.Error()}
		}
	}
	if err := writeSettings(path, Settings(HookCommand(s.binary))); err != nil {
		return nil, &Error{CodeSpawnFailed, "settings: " + err.Error()}
	}
	env := Environment(os.Environ(), append(append([]Var{}, spec.Env...), own...))
	return start(s.c.EnginePath, spec.Args, spec.Cwd, env, spec.Stdin, s.c.InterruptTimeout)
}

// HarnessCreate makes <root>/harness/<id> (0750) as a directory operation, the parent created
// if needed; an existing directory is refused, since ids are fresh.
func (s *SameAccount) HarnessCreate(dir string) error {
	if err := checkHarnessDir(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return &Error{CodeHarness, err.Error()}
	}
	if err := os.Mkdir(dir, 0o750); err != nil {
		return &Error{CodeHarness, err.Error()}
	}
	return nil
}

// HarnessRemove deletes <root>/harness/<id> and everything the engine wrote there; a missing
// directory is not an error.
func (s *SameAccount) HarnessRemove(dir string) error {
	if err := checkHarnessDir(dir); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return &Error{CodeHarness, err.Error()}
	}
	return nil
}

// checkHarnessDir refuses anything but an absolute <root>/harness/cred-<id>, so a bad record
// can never point either operation elsewhere.
func checkHarnessDir(dir string) error {
	clean := filepath.Clean(dir)
	if !filepath.IsAbs(clean) || filepath.Base(filepath.Dir(clean)) != "harness" || !strings.HasPrefix(filepath.Base(clean), "cred-") {
		return &Error{CodeHarness, quote(filepath.ToSlash(dir)) + " is not <root>/harness/cred-<id>"}
	}
	return nil
}

// Preflight runs the same-account checks (preflight.go).
func (s *SameAccount) Preflight() []Check { return preflightSameAccount(s) }
