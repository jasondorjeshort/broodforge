// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Backend names, the values of the instance config's spawn.backend.
const (
	NameSameAccount    = "same-account"
	NameLinuxSudoBwrap = "linux-sudo-bwrap"
)

// Error is a refusal with a stable code (05-cli-conventions.md); the CLI maps it to its error
// line. Preflight codes are preflight-<check>.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Codes this package refuses with, besides preflight-<check>.
const (
	CodeSpawnFailed = "spawn-failed" // the engine process could not be started
	CodeHarness     = "harness"      // harness-create or harness-remove failed
	CodeBadDelivery = "bad-delivery" // a Spec without its credential or home
	CodeBackendHeld = "preflight-backend"
)

// DefaultInterrupt is spawn.interruptTimeoutSeconds when unset (03-architecture.md).
const DefaultInterrupt = 20 * time.Second

const (
	settingsFile      = "settings.json"
	hookTimeoutSecond = 30 // the hook's timeout in the settings template (03-hook-contract.md)
)

// Backend is SpawnBackend: how an engine process, and the maintenance commands around it, are
// started on this host.
type Backend interface {
	Name() string
	// Start writes the settings template to the engine's settings path for the delivery's
	// kind, then starts the engine with spec's arguments and the explicit environment.
	Start(spec Spec) (*Process, error)
	// HarnessCreate makes a subscription credential's config directory; HarnessRemove deletes it.
	HarnessCreate(dir string) error
	HarnessRemove(dir string) error
	// Preflight runs the backend's checks without a session (05, Spawn preflight and reach).
	Preflight() []Check
	// MCPConfig is the empty MCP map every engine process names (ADR 0022 companion).
	MCPConfig() string
	// WorktreeCreate clones the canonical into the session's tree on the session branch;
	// WorktreeBundle writes the session's bundle of its commits since the base; WorktreeRemove
	// deletes the tree and the bundle (worktree.go; ADR 0022 companion).
	WorktreeCreate(w Worktree) error
	WorktreeBundle(w Worktree) (Bundle, error)
	WorktreeRemove(session string) error
	// HooksDir is the directory the clone's core.hooksPath names; Askpass the GIT_ASKPASS
	// helper the bot's push names. Both are under <bf>/share.
	HooksDir() string
	Askpass() string
}

// Config is what a backend needs from the instance config; spawn does not import config.
type Config struct {
	Backend       string
	EnginePath    string // engine.path: the binary itself, never a PATH lookup or the .cmd shim
	EngineVersion string // engine.version
	GitBashPath   string // engine.gitBashPath; "" lets the engine find Git Bash itself
	Install       string // <bf>: holds share/; "" is the parent of the binary's directory
	Binary        string // the broodforge binary the hook runs; "" is os.Executable
	// InterruptTimeout is how long a Linux SIGINT waits before SIGKILL; 0 is DefaultInterrupt.
	InterruptTimeout time.Duration
}

// New returns the configured backend. linux-sudo-bwrap is held: it is returned, and refuses
// every operation with preflight-backend: held.
func New(c Config) (Backend, error) {
	switch c.Backend {
	case NameSameAccount:
		return NewSameAccount(c)
	case NameLinuxSudoBwrap:
		return LinuxSudoBwrap{}, nil
	}
	return nil, &Error{CodeBackendHeld, "spawn.backend is " + quote(c.Backend) + ", want " + NameSameAccount + " or " + NameLinuxSudoBwrap}
}

// DeliveryKind is how the credential reaches the engine (ADR 0022, Both hosts: delivery).
type DeliveryKind string

const (
	// DeliverConfigDir is a subscription: CLAUDE_CONFIG_DIR names harness/<credential id>.
	DeliverConfigDir DeliveryKind = "config-dir"
	// DeliverAPIKey is an API key: ANTHROPIC_API_KEY on the engine process only.
	DeliverAPIKey DeliveryKind = "api-key"
)

// Delivery is the credential for one process. APIKey is the value, for the process
// environment only; nothing in this package logs or prints it.
type Delivery struct {
	Kind      DeliveryKind
	ConfigDir string
	APIKey    []byte
}

// Var is one environment variable.
type Var struct {
	Name  string
	Value string
}

// Spec is one engine process.
type Spec struct {
	Args     []string // after the binary: ADR 0020's flag list
	Cwd      string   // the session tree
	Home     string   // sessions/<id>/home, created if missing: the hook log's directory
	Delivery Delivery
	Env      []Var  // the hook variables and the four git identity variables
	Stdin    []byte // written to the engine's stdin, which is then closed
}

// HarnessDir is a subscription credential's config directory, <root>/harness/<id>.
func HarnessDir(root, id string) string {
	return filepath.ToSlash(filepath.Join(root, "harness", id))
}

// EmptyShare is <bf>/share/empty, the empty files every engine names or is bound over.
func EmptyShare(install string) string {
	return filepath.ToSlash(filepath.Join(install, "share", "empty"))
}

// SettingsPath is where the engine reads its user settings for a delivery (ADR 0022
// companion, <ecfg>): the config directory for a subscription, <home>/.claude for an API key.
func SettingsPath(d Delivery, home string) (string, error) {
	switch d.Kind {
	case DeliverConfigDir:
		if d.ConfigDir == "" {
			return "", &Error{CodeBadDelivery, "a config-dir delivery names no directory"}
		}
		return filepath.ToSlash(filepath.Join(d.ConfigDir, settingsFile)), nil
	case DeliverAPIKey:
		if home == "" {
			return "", &Error{CodeBadDelivery, "an api-key delivery needs the session home"}
		}
		return filepath.ToSlash(filepath.Join(home, ".claude", settingsFile)), nil
	}
	return "", &Error{CodeBadDelivery, "delivery kind " + quote(string(d.Kind))}
}

// HookCommand is the settings template's hook command: the binary, forward slashes, quoted
// with double quotes when it holds a space (both sh and cmd read that form), then
// `hook pre-tool-use`.
func HookCommand(binary string) string {
	b := filepath.ToSlash(binary)
	if strings.ContainsAny(b, " \t") {
		b = `"` + b + `"`
	}
	return b + " hook pre-tool-use"
}

// Settings is the settings template (03-hook-contract.md, How the engine runs it): one
// PreToolUse command hook with an empty matcher and a thirty-second timeout. It is the same for
// both credential kinds; the kind picks the path. ADR 0022's sandbox block joins it once C33
// holds.
func Settings(hookCommand string) []byte {
	type hook struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	type matcher struct {
		Matcher string `json:"matcher"`
		Hooks   []hook `json:"hooks"`
	}
	v := struct {
		Hooks struct {
			PreToolUse []matcher `json:"PreToolUse"`
		} `json:"hooks"`
	}{}
	v.Hooks.PreToolUse = []matcher{{Matcher: "", Hooks: []hook{{Type: "command", Command: hookCommand, Timeout: hookTimeoutSecond}}}}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err) // a fixed struct of strings and ints always marshals
	}
	return append(b, '\n')
}

// writeSettings writes content to path by temp file and rename (ADR 0022: before every process).
func writeSettings(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".settings-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(content)
	cerr := f.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp, path)
	}
	if werr != nil {
		_ = os.Remove(tmp)
	}
	return werr
}

// strippedPrefixes are inherited names the engine never gets from the bot's environment: its
// own credential and configuration names (ADR 0020: every process gets an explicit credential,
// the owner's ~/.claude is never used), Broodforge's (the key file's path stays out of a
// session's Bash) and the git identity, which comes from the driver alone.
var strippedPrefixes = []string{"CLAUDE", "ANTHROPIC_", "BROODFORGE_", "GIT_AUTHOR_", "GIT_COMMITTER_",
	"GIT_CONFIG_KEY_", "GIT_CONFIG_VALUE_"} // the last two: GIT_CONFIG_COUNT's numbered pairs

// gitLocalNames are git's repository-local variables (`git rev-parse --local-env-vars`, git
// 2.5x): inherited, they would point the session's git at the bot's own repository.
var gitLocalNames = map[string]bool{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_CONFIG": true, "GIT_CONFIG_PARAMETERS": true, "GIT_CONFIG_COUNT": true,
	"GIT_OBJECT_DIRECTORY": true, "GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_IMPLICIT_WORK_TREE": true, "GIT_GRAFT_FILE": true,
	"GIT_INDEX_FILE": true, "GIT_NO_REPLACE_OBJECTS": true, "GIT_REPLACE_REF_BASE": true, "GIT_PREFIX": true,
	"GIT_SHALLOW_FILE": true, "GIT_COMMON_DIR": true,
}

// keptInherited survive the strip (the spike's rule): Git Bash's location for the engine.
var keptInherited = map[string]bool{"CLAUDE_CODE_GIT_BASH_PATH": true}

// sessionNames are names an agent session sets for its own children: a bot or debug command
// started from inside one must not hand them on (the spike's list).
var sessionNames = map[string]bool{
	"AI_AGENT": true, "GIT_EDITOR": true, "TRACEPARENT": true, "GIT_CONFIG_PARAMETERS": true, "TMPDIR": true,
	"COREPACK_ENABLE_AUTO_PIN": true, "WT_SESSION": true, "MSYSTEM": true, "SHELL": true,
}

// Environment is the engine's explicit environment: parent minus the stripped names, then
// set, later entries winning (names compared as the host compares them).
func Environment(parent []string, set []Var) []string {
	var out []string
	for _, kv := range parent {
		name, _, ok := strings.Cut(kv, "=")
		if !ok || name == "" { // Windows' hidden "=C:=C:\..." entries have an empty name
			out = append(out, kv)
			continue
		}
		up := strings.ToUpper(name)
		if sessionNames[up] || gitLocalNames[up] {
			continue
		}
		strip := false
		for _, p := range strippedPrefixes {
			if strings.HasPrefix(up, p) && !keptInherited[up] {
				strip = true
				break
			}
		}
		if !strip {
			out = append(out, kv)
		}
	}
	for _, v := range set {
		out = append(out, v.Name+"="+v.Value)
	}
	return dedupe(out)
}

// dedupe keeps the last entry of each name.
func dedupe(env []string) []string {
	last := map[string]int{}
	for i, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if name != "" {
			last[foldName(name)] = i
		}
	}
	out := env[:0:0]
	for i, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if name == "" || last[foldName(name)] == i {
			out = append(out, kv)
		}
	}
	return out
}

// VarNames is the names of an environment, for records: never values.
func VarNames(env []string) []string {
	names := make([]string, 0, len(env))
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
