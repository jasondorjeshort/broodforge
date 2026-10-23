// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/jasondorjeshort/broodforge/internal/spawn"
)

// The runtime probe (ADR 0025 rule 6; the companion's The runtime probe, verbatim): per entry,
// the first word of the command, read as splitWords reads it:
//   - a runtime of the list: `<word> --version` where the children run; not found or non-zero fails;
//   - a word that names a file: the file must exist (a relative word against the copy); a .js also
//     needs node, a .py python, a .ps1 pwsh, each probed as above;
//   - anything else: the word must resolve on the children's PATH.
// Where: this process's environment (the instance account's on the PC; on the droplet `spawn
// preflight` runs it inside the sandbox, a later pass).

// runtimes are the words probed with --version.
var runtimes = []string{"node", "python", "python3", "bash", "sh", "pwsh", "deno", "bun"}

// suffixRuntime is the runtime a script file needs, "" for none beyond the file.
var suffixRuntime = map[string]string{".js": "node", ".py": "python", ".ps1": "pwsh", ".sh": "", ".exe": ""}

// versionTimeout bounds one `<word> --version`.
const versionTimeout = 10 * time.Second

// Probed is one entry that passed: Runtime is the word the probe ran or found.
type Probed struct {
	Hook
	Runtime string `json:"runtime"`
}

// ProbeFailure is the first entry whose runtime is missing: Word is what it needs.
type ProbeFailure struct {
	Hook Hook
	Word string
	Why  string
}

func (f *ProbeFailure) Error() string {
	return CodeProbeFailed + ": " + f.Hook.Command + " needs " + f.Word
}

// NotStarted is the companion's not-started text for f in project id's venue (wording: owner to
// confirm).
func (f *ProbeFailure) NotStarted(project string) string {
	return fmt.Sprintf("Not started: the project's hook %s needs %s, which is not installed where sessions run. "+
		"The owner installs it, or removes %s from workflow.hooks in config/projects/%s.json.",
		f.Hook.Command, f.Word, f.Hook.Event, project)
}

// Probe probes each hook of the copy in dir in order and returns the entries that passed; the
// first failure stops it and is returned as a *ProbeFailure.
func Probe(ctx context.Context, dir string, hooks []Hook) ([]Probed, error) {
	versions := map[string]string{} // word: "" when it ran, else why not
	version := func(w string) string {
		if why, ok := versions[w]; ok {
			return why
		}
		why := runVersion(ctx, w)
		versions[w] = why
		return why
	}
	var out []Probed
	for _, h := range hooks {
		ws := splitWords(h.Command, dir)
		if len(ws) == 0 {
			return out, &ProbeFailure{h, `""`, "no command"}
		}
		w := ws[0]
		runtime := w.Raw
		switch {
		case slices.Contains(runtimes, w.Expanded):
			if why := version(w.Expanded); why != "" {
				return out, &ProbeFailure{h, w.Raw, why}
			}
		case namesFile(w.Expanded):
			p := filepath.FromSlash(w.Expanded)
			if !filepath.IsAbs(p) {
				p = filepath.Join(dir, p)
			}
			if fi, err := os.Stat(p); err != nil || fi.IsDir() {
				return out, &ProbeFailure{h, w.Raw, "no such file"}
			}
			if need := suffixRuntime[suffix(w.Expanded)]; need != "" {
				if why := version(need); why != "" {
					return out, &ProbeFailure{h, need, why}
				}
				runtime = need
			}
		default:
			if _, err := exec.LookPath(w.Expanded); err != nil {
				return out, &ProbeFailure{h, w.Raw, err.Error()}
			}
		}
		out = append(out, Probed{h, runtime})
	}
	return out, nil
}

// runVersion runs `<word> --version` with this process's environment minus the names spawn
// strips; "" when it ran and exited 0, else why not.
func runVersion(ctx context.Context, w string) string {
	p, err := exec.LookPath(w)
	if err != nil {
		return err.Error()
	}
	ctx, cancel := context.WithTimeout(ctx, versionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p, "--version")
	cmd.Env = spawn.Environment(os.Environ(), nil)
	if err := cmd.Run(); err != nil {
		return w + " --version: " + err.Error()
	}
	return ""
}
