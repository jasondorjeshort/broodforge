// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Events are the hook events a repository may have run, in hooks.json's order (ADR 0025 rule 6).
var Events = []string{"PreToolUse", "PostToolUse"}

// The copied files the renders read, slash paths under the copy.
const (
	claudeFile   = "CLAUDE.md"
	rulesDir     = ".claude/rules"
	agentsDir    = ".claude/agents"
	settingsFile = ".claude/settings.json"
)

// Hook is one hooks.json entry: a command hook of a listed event in .claude/settings.json.
// Timeout is the hook's own, in seconds, null when it sets none.
type Hook struct {
	Event   string `json:"event"`
	Matcher string `json:"matcher"`
	Command string `json:"command"`
	Name    string `json:"name"`
	Timeout *int   `json:"timeout"`
}

// Agent is one agents.json value, the engine's --agents shape keyed by the bare name.
type Agent struct {
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	Model       string   `json:"model,omitempty"`
	Effort      string   `json:"effort,omitempty"`
	Tools       []string `json:"tools,omitempty"`
}

func (c *Copy) render(events []string) error {
	instr := ""
	if !c.None() {
		instr = c.instructions()
	}
	if err := c.write(InstructionsFile, []byte(instr)); err != nil {
		return err
	}
	if c.None() {
		return nil
	}
	agents := c.agents()
	if len(agents) > 0 {
		b, err := encodeJSON(agents)
		if err != nil {
			return err
		}
		if err := c.write(AgentsFile, b); err != nil {
			return err
		}
	}
	hooks, err := c.hooks(events)
	if err != nil {
		return err
	}
	c.Hooks = hooks
	if len(hooks) > 0 {
		b, err := encodeJSON(hooks)
		if err != nil {
			return err
		}
		if err := c.write(HooksFile, b); err != nil {
			return err
		}
	}
	return nil
}

func (c *Copy) write(name string, b []byte) error {
	if err := os.WriteFile(filepath.Join(c.Dir, name), b, 0o644); err != nil {
		return &Error{CodeOut, err.Error()}
	}
	c.Files = append(c.Files, c.Dir+"/"+name)
	return nil
}

func (c *Copy) path(rel string) string { return filepath.Join(c.Dir, filepath.FromSlash(rel)) }

// instructions is instructions.md without the header: CLAUDE.md and every file its @ imports
// reach, each once in the order the resolver reaches them, an import not loaded as a `note:`
// line after its file; then every .claude/rules/*.md in name order. Each file opens with a
// `Contents of <path>:` line, as the engine's own memory loader labels them.
func (c *Copy) instructions() string {
	var b strings.Builder
	block := func(rel, text string) {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Contents of " + rel + ":\n\n" + strings.TrimRight(text, "\n") + "\n")
	}
	if text, ok := readText(c.path(claudeFile)); ok {
		refs := resolveImports(c.Dir, claudeFile)
		order := []string{claudeFile}
		for _, r := range refs {
			if r.exists && !slices.Contains(order, r.target) {
				order = append(order, r.target)
			}
		}
		for i, rel := range order {
			if i > 0 {
				text, _ = readText(c.path(rel))
			}
			block(rel, text)
			for _, r := range refs {
				if r.file == rel && !r.exists {
					b.WriteString("\nnote: @" + r.token + " in " + r.file + ":" + strconv.Itoa(r.line) + " was not loaded: " + r.why() + "\n")
				}
			}
		}
	}
	for _, rel := range c.listMD(rulesDir) {
		if text, ok := readText(c.path(rel)); ok {
			block(rel, text)
		}
	}
	return b.String()
}

func (r importRef) why() string {
	switch {
	case r.outside:
		return r.target + " is outside the copy"
	case r.isDir:
		return r.target + " is a directory"
	}
	return r.target + " is not in the copy"
}

// listMD is the regular *.md files directly in dir (a slash path under the copy), in name order.
func (c *Copy) listMD(dir string) []string {
	entries, err := os.ReadDir(c.path(dir))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, dir+"/"+e.Name())
		}
	}
	slices.Sort(out)
	return out
}

// agents is agents.json: each .claude/agents/*.md in name order, its frontmatter's name,
// description, model, effort and tools, its body as the prompt. A file without frontmatter, a
// name or a description, or whose name an earlier file took, is a Skip.
func (c *Copy) agents() map[string]Agent {
	out := map[string]Agent{}
	from := map[string]string{}
	for _, rel := range c.listMD(agentsDir) {
		text, ok := readText(c.path(rel))
		if !ok {
			continue
		}
		fm, body, ok := frontmatter(text)
		name, desc := fm["name"].str(), fm["description"].str()
		switch {
		case !ok:
			c.Skips = append(c.Skips, Skip{rel, "no frontmatter"})
			continue
		case name == "" || desc == "":
			c.Skips = append(c.Skips, Skip{rel, "the frontmatter needs a name and a description"})
			continue
		case from[name] != "":
			c.Skips = append(c.Skips, Skip{rel, "the name " + name + " is " + from[name] + "'s"})
			continue
		}
		from[name] = rel
		a := Agent{Description: desc, Prompt: body, Model: fm["model"].str(), Effort: fm["effort"].str()}
		if t, ok := fm["tools"]; ok {
			a.Tools = t.items()
		}
		out[name] = a
		c.Agents = append(c.Agents, name)
	}
	return out
}

// settings is the part of .claude/settings.json the copy reads.
type settings struct {
	Hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Timeout *float64 `json:"timeout"`
		} `json:"hooks"`
	} `json:"hooks"`
}

// hooks is hooks.json: every command hook of a listed event, the events in Events' order and
// each event's entries in the file's order, named by hookName with a repeat suffixed -2, -3.
// No event listed reads nothing; with one listed, a settings file that does not parse, and a
// hook of a listed event that is not a command or has none, are errors (ADR 0025 rule 6).
func (c *Copy) hooks(events []string) ([]Hook, error) {
	if len(events) == 0 {
		return nil, nil
	}
	raw, err := os.ReadFile(c.path(settingsFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &Error{CodeOut, err.Error()}
	}
	var s settings
	if err := json.Unmarshal(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}), &s); err != nil {
		return nil, &Error{CodeSettings, settingsFile + ": " + err.Error()}
	}
	var out []Hook
	seen := map[string]int{}
	for _, ev := range Events {
		if !slices.Contains(events, ev) {
			continue
		}
		for gi, g := range s.Hooks[ev] {
			for hi, h := range g.Hooks {
				// A listed hook that cannot run is a configuration error, never a skip (rule 6).
				where := settingsFile + ": hooks." + ev + "." + strconv.Itoa(gi) + ".hooks." + strconv.Itoa(hi) +
					" (" + ev + ", matcher " + strconv.Quote(g.Matcher) + ")"
				if h.Type != "command" {
					return nil, &Error{CodeSettings, where + " is a " + strconv.Quote(h.Type) + " hook; only a command hook runs as a child"}
				}
				if strings.TrimSpace(h.Command) == "" {
					return nil, &Error{CodeSettings, where + " has no command"}
				}
				e := Hook{Event: ev, Matcher: g.Matcher, Command: h.Command, Name: hookName(h.Command, c.Dir)}
				if h.Timeout != nil {
					t := int(math.Ceil(*h.Timeout))
					e.Timeout = &t
				}
				seen[e.Name]++
				if n := seen[e.Name]; n > 1 {
					e.Name += "-" + strconv.Itoa(n)
				}
				out = append(out, e)
			}
		}
	}
	return out, nil
}

// encodeJSON is the rendered files' encoding: indented, HTML characters as they are.
func encodeJSON[T map[string]Agent | []Hook](v T) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ReadHooks is a copy's hooks.json, nil when it has none.
func ReadHooks(dir string) ([]Hook, error) {
	b, err := os.ReadFile(filepath.Join(dir, HooksFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &Error{CodeOut, err.Error()}
	}
	var hooks []Hook
	if err := json.Unmarshal(b, &hooks); err != nil {
		return nil, &Error{CodeOut, HooksFile + ": " + err.Error()}
	}
	return hooks, nil
}
