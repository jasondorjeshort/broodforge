// SPDX-License-Identifier: GPL-3.0-or-later

package spawn

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Check is one preflight check: OK, or a failure whose refusal is preflight-<Name>.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

// Code is the refusal code of a failed check.
func (c Check) Code() string { return "preflight-" + c.Name }

// versionTimeout bounds the engine's --version run.
const versionTimeout = 30 * time.Second

var versionRE = regexp.MustCompile(`^\s*v?(\d+(?:\.\d+)+)`)

// EngineVersion runs `<path> --version`, the one engine invocation preflight makes, with the
// explicit environment and an empty temporary config directory, so no ~/.claude is read, and
// returns the output's first line and its leading dotted number ("2.1.296 (Claude Code)" is
// 2.1.296; 03-architecture.md, Configuration).
func EngineVersion(path string) (raw, version string, err error) {
	cfg, err := os.MkdirTemp("", "broodforge-version-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), versionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Dir = cfg
	cmd.Env = Environment(os.Environ(), []Var{{"CLAUDE_CONFIG_DIR", filepath.ToSlash(cfg)}, {"DISABLE_AUTOUPDATER", "1"}})
	cmd.SysProcAttr = sysProcAttr()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	raw, _, _ = strings.Cut(strings.TrimSpace(string(out)), "\n")
	raw = strings.TrimSpace(raw)
	if err != nil {
		msg := err.Error()
		if s := strings.TrimSpace(stderr.String()); s != "" {
			msg += ": " + s
		}
		return raw, "", &Error{"preflight-engine-version", msg}
	}
	m := versionRE.FindStringSubmatch(raw)
	if m == nil {
		return raw, "", &Error{"preflight-engine-version", "no version in " + quote(raw)}
	}
	return raw, m[1], nil
}

func preflightSameAccount(s *SameAccount) []Check {
	checks := []Check{{Name: "backend", OK: true, Detail: NameSameAccount}}
	path := checkEnginePath(s.c.EnginePath)
	checks = append(checks, path)
	if path.OK {
		checks = append(checks, checkEngineVersion(s.c.EnginePath, s.c.EngineVersion))
	}
	if runtime.GOOS == "windows" {
		checks = append(checks, checkGitBash(s.c.GitBashPath))
	}
	checks = append(checks, checkShare(s.install), checkAccount())
	return checks
}

func checkEnginePath(p string) Check {
	c := Check{Name: "engine-path"}
	switch {
	case p == "":
		c.Detail = "engine.path is not set"
	case !filepath.IsAbs(p):
		c.Detail = "engine.path " + quote(p) + " is not absolute"
	case runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(p), ".exe"):
		// The npm .cmd shim and the extensionless sh wrapper are not the engine (ADR 0020).
		c.Detail = "engine.path " + quote(p) + " is not the engine's .exe"
	default:
		st, err := os.Stat(p)
		switch {
		case err != nil:
			c.Detail = "engine.path: " + err.Error()
		case !st.Mode().IsRegular():
			c.Detail = "engine.path " + quote(p) + " is not a file"
		default:
			c.OK, c.Detail = true, filepath.ToSlash(p)
		}
	}
	return c
}

func checkEngineVersion(path, want string) Check {
	c := Check{Name: "engine-version"}
	raw, got, err := EngineVersion(path)
	switch {
	case err != nil:
		c.Detail = strings.TrimPrefix(err.Error(), c.Code()+": ")
	case want == "":
		c.Detail = "engine.version is not set; the binary says " + got
	case got != want:
		c.Detail = "engine.version is " + want + ", the binary says " + got + " (" + quote(raw) + ")"
	default:
		c.OK, c.Detail = true, got
	}
	return c
}

func checkGitBash(p string) Check {
	c := Check{Name: "git-bash"}
	if p == "" {
		c.OK, c.Detail = true, "unset: the engine finds Git Bash itself"
		return c
	}
	if st, err := os.Stat(p); err != nil {
		c.Detail = "engine.gitBashPath: " + err.Error()
	} else if !st.Mode().IsRegular() {
		c.Detail = "engine.gitBashPath " + quote(p) + " is not a file"
	} else {
		c.OK, c.Detail = true, filepath.ToSlash(p)
	}
	return c
}

// checkShare checks <bf>/share/empty: mcp.json an object, CLAUDE.md empty, dir/ empty. Git
// cannot carry an empty directory, so dir/ is created here when missing.
func checkShare(install string) Check {
	c := Check{Name: "share"}
	share := EmptyShare(install)
	b, err := os.ReadFile(filepath.Join(share, "mcp.json"))
	if err != nil {
		c.Detail = err.Error()
		return c
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		c.Detail = share + "/mcp.json: " + err.Error()
		return c
	}
	if st, err := os.Stat(filepath.Join(share, "CLAUDE.md")); err != nil {
		c.Detail = err.Error()
		return c
	} else if st.Size() != 0 {
		c.Detail = share + "/CLAUDE.md is not empty"
		return c
	}
	dir := filepath.Join(share, "dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.Detail = err.Error()
		return c
	}
	if ents, err := os.ReadDir(dir); err != nil {
		c.Detail = err.Error()
		return c
	} else if len(ents) > 0 {
		c.Detail = share + "/dir is not empty"
		return c
	}
	c.OK, c.Detail = true, share
	return c
}

// checkAccount reports the account engines run as (P13 reads it for the instance account).
func checkAccount() Check {
	u, err := user.Current()
	if err != nil {
		return Check{Name: "account", Detail: err.Error()}
	}
	return Check{Name: "account", OK: true, Detail: u.Username}
}
