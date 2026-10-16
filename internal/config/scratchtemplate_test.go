// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScratchTemplateLoads(t *testing.T) {
	root := filepath.ToSlash(t.TempDir())
	o := ScratchOptions{Root: root, InstanceName: "scratch-0123456789ab", Install: "/opt/bf",
		Fixtures: "/opt/bf/fixtures/engine", DefaultBranch: "main"}
	if err := WriteScratch(o); err != nil {
		t.Fatal(err)
	}
	c, err := Load(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	in := c.Instance
	if in.InstanceName != o.InstanceName || in.Discord.Mode != DiscordDryRun || in.Defaults.Harness != HarnessFake ||
		in.Spawn.Backend != BackendSameAccount || in.Credentials.KeyFile != ScratchKeyFile ||
		in.Engine.FakeFixtures == nil || *in.Engine.FakeFixtures != o.Fixtures || in.Paths.Install != o.Install {
		t.Errorf("instance %+v", in)
	}
	if len(c.Projects) != 1 || c.Projects[0].ID != ScratchProject {
		t.Fatalf("projects %+v", c.Projects)
	}
	p := c.Projects[0]
	if p.Repo.Canonical != root+"/projects/scratch" || p.Repo.DefaultBranch != "main" || p.HandoffOrDefault() != HandoffPush {
		t.Errorf("repo %+v handoff %s", p.Repo, p.HandoffOrDefault())
	}
	for id, want := range map[string][2]string{"100": {ReachCode, ModePrompt}, "200": {ReachReadOnly, ModePolicy}} {
		ch, ok := p.Channels[id]
		if !ok || ch.Reach != want[0] || ch.Mode == nil || *ch.Mode != want[1] || strings.Join(ch.Roles.Start, ",") != "trusted" {
			t.Errorf("channel %s: %+v", id, ch)
		}
	}
}

func TestCopyForScratch(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "config", "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	inst := `{"//": "kept", "instanceName": "home", "credentials": {"keyFile": "/secret/home.key"}, "schemaVersion": 1}`
	proj := "{\"id\": \"alpha\"}\n"
	if err := os.WriteFile(filepath.Join(src, "config", "instance.json"), []byte(inst), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "config", "projects", "alpha.json"), []byte(proj), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CopyForScratch(src, dst, "scratch-0123456789ab"); err != nil {
		t.Fatal(err)
	}
	name, err := InstanceName(dst)
	if err != nil || name != "scratch-0123456789ab" {
		t.Errorf("instanceName %q, %v", name, err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "config", "instance.json"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if strings.Contains(got, "/secret/home.key") || !strings.Contains(got, `"keyFile": "`+ScratchKeyFile+`"`) || !strings.Contains(got, `"//": "kept"`) {
		t.Errorf("instance.json copy:\n%s", got)
	}
	b, err = os.ReadFile(filepath.Join(dst, "config", "projects", "alpha.json"))
	if err != nil || string(b) != proj {
		t.Errorf("project copy %q, %v", b, err)
	}

	if err := CopyForScratch(t.TempDir(), dst, "scratch-0123456789ab"); err == nil {
		t.Error("a source without config/instance.json copied")
	} else if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("got %T %v, want *NotFoundError", err, err)
	}
}
