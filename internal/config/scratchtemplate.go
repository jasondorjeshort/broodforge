// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// The scratch configuration (05-cli-conventions.md, Scratch): dry-run Discord, the fake
// harness reading the repository's fixtures/engine/, the same-account backend, the store key
// under the root, and one project `scratch` with channel 100 (code reach, prompt mode) and
// channel 200 (read-only reach, policy mode), both open to group `trusted` (role 10): the ids
// of setuptemplate.go's DryRunChannels and PlaceholderRole.

// ScratchKeyFile is a scratch root's credentials.keyFile, relative to the root, so a scratch
// run never names the owner's key (BROODFORGE_KEY_FILE is ignored on a scratch root).
const ScratchKeyFile = "key/store.key"

// ScratchProject is the scratch root's one project id.
const ScratchProject = "scratch"

// ScratchOptions are what the template needs from its caller.
type ScratchOptions struct {
	Root          string // the scratch root, absolute
	InstanceName  string // scratch-<12 hex>
	Install       string // paths.install: the directory holding bin/ and share/
	Fixtures      string // engine.fakeFixtures: the repository's fixtures/engine
	DefaultBranch string // the scratch project's default branch
}

// ScratchTemplate is the scratch configuration for o.
func ScratchTemplate(o ScratchOptions) (Instance, Project) {
	fixtures, prompt, policy := filepath.ToSlash(o.Fixtures), ModePrompt, ModePolicy
	inst := Instance{
		SchemaVersion: SchemaVersion,
		InstanceName:  o.InstanceName,
		Credentials:   Credentials{KeyFile: ScratchKeyFile},
		Discord:       Discord{Mode: DiscordDryRun},
		Engine:        Engine{FakeFixtures: &fixtures},
		Spawn:         Spawn{Backend: BackendSameAccount},
		Paths:         Paths{Install: filepath.ToSlash(o.Install)},
		RoleGroups:    map[string][]string{"trusted": {PlaceholderRole}},
		Defaults:      Defaults{IdleTimeoutMinutes: 0, PromptTimeoutMinutes: 0, Mode: ModePrompt, Harness: HarnessFake},
	}
	roles := Roles{Start: []string{"trusted"}, Contribute: []string{"trusted"}, Answer: []string{"trusted"}}
	proj := Project{
		ID:   ScratchProject,
		Name: "Scratch",
		Repo: Repo{Canonical: slashJoin(o.Root, "projects", ScratchProject), Remote: "origin",
			DefaultBranch: o.DefaultBranch},
		Handoff: HandoffPush,
		Channels: map[string]Channel{
			DryRunChannels[0].ID: {Reach: ReachCode, Roles: roles, Mode: &prompt},
			DryRunChannels[1].ID: {Reach: ReachReadOnly, Roles: roles, Mode: &policy},
		},
	}
	return inst, proj
}

// WriteScratch writes the template's config/instance.json and config/projects/scratch.json
// under o.Root.
func WriteScratch(o ScratchOptions) error {
	inst, proj := ScratchTemplate(o)
	if err := writeJSON(InstancePath(o.Root), inst); err != nil {
		return err
	}
	return writeJSON(slashJoin(Dir(o.Root), "projects", ScratchProject+".json"), proj)
}

func writeJSON[T Instance | Project | map[string]json.RawMessage](path string, v T) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// PlainInstanceName reads instanceName from config/instance.json of root and nothing else: no
// last-good fallback and no recovery (InstanceName's), so it never renames or writes anything
// under root. The scratch deletion check reads a candidate root's name with it.
func PlainInstanceName(root string) (string, error) { return readInstanceName(InstancePath(root)) }

// CopyForScratch copies config/ of the instance at src into dst (a backup rehearsal's scratch
// root), every file as it is but instance.json, whose instanceName becomes name and whose
// credentials.keyFile becomes ScratchKeyFile, so the copy is a scratch root that names neither
// the source's control socket nor its key. Comment keys survive; key order does not.
func CopyForScratch(src, dst, name string) error {
	from, to := filepath.Clean(Dir(src)), filepath.Clean(Dir(dst))
	if _, err := os.Stat(slashJoin(from, "instance.json")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &NotFoundError{Path: slashJoin(from, "instance.json"), Err: err}
		}
		return err
	}
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		out := filepath.Join(to, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(out, 0o700)
		case !d.Type().IsRegular():
			return nil // a link or device under config/ is not copied
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(rel) == "instance.json" {
			return rewriteInstance(out, b, name)
		}
		return os.WriteFile(out, b, 0o600)
	})
}

func rewriteInstance(out string, src []byte, name string) error {
	var top map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(src))
	if err := dec.Decode(&top); err != nil || top == nil {
		return &InvalidError{Problems: []Problem{{File: filepath.ToSlash(out), Message: "instance.json is not a JSON object"}}}
	}
	var creds map[string]json.RawMessage
	if raw, ok := top["credentials"]; ok && strings.HasPrefix(strings.TrimSpace(string(raw)), "{") {
		if err := json.Unmarshal(raw, &creds); err != nil {
			return err
		}
	}
	if creds == nil {
		creds = map[string]json.RawMessage{}
	}
	var err error
	if creds["keyFile"], err = json.Marshal(ScratchKeyFile); err != nil {
		return err
	}
	if top["credentials"], err = json.Marshal(creds); err != nil {
		return err
	}
	if top["instanceName"], err = json.Marshal(name); err != nil {
		return err
	}
	return writeJSON(filepath.ToSlash(out), top)
}
