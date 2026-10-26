// SPDX-License-Identifier: GPL-3.0-or-later

package config_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/config"
)

// The P24 proofs of the config package (08-implementation-plan-setup.md): an export holds every
// key of config/ but the host-bound ones, which it empties, and no secret; an import writes it
// into an empty root, absent only; an export this binary's schema rejects names the key.

// setupRoot writes a set-up root's config/ from the setup templates, every host-bound key filled.
func setupRoot(t *testing.T) string {
	t.Helper()
	root := filepath.ToSlash(t.TempDir())
	inst := config.SetupInstance(config.SetupOptions{InstanceName: "first", KeyFile: root + "/../first.key", Install: root + "/install"})
	for _, kv := range []struct {
		v    string
		path []string
	}{
		{"cred-bot1", []string{"discord", "botCredentialId"}}, {"4000", []string{"discord", "applicationId"}},
		{"3000", []string{"discord", "guildId"}}, {"9003", []string{"discord", "noticeChannelId"}},
		{root + "/engine/claude.exe", []string{"engine", "path"}}, {"2.1.296", []string{"engine", "version"}},
		{root + "/git/bin/bash.exe", []string{"engine", "gitBashPath"}},
		{root + "/fixtures/engine", []string{"engine", "fakeFixtures"}}, {root + "/audit.jsonl", []string{"audit", "fileSink"}},
	} {
		if err := inst.Set("", config.RawValue(kv.v), kv.path...); err != nil {
			t.Fatal(err)
		}
	}
	if err := inst.Set("", config.RawValue([]string{"9004"}), "roleGroups", "trusted"); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteDoc(config.InstancePath(root), inst); err != nil {
		t.Fatal(err)
	}
	p := config.SetupProject(config.SetupProjectOptions{ID: "demo", Name: "demo", Canonical: root + "/projects/demo",
		Handoff: config.HandoffPush, URL: "https://user:ghp_secret1@example.com/demo.git"})
	for _, kv := range [][2]string{{"cred-git1", "gitCredentialId"}, {"main", "defaultBranch"}} {
		if err := p.Set("", config.RawValue(kv[0]), "repo", kv[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Set("", config.RawValue("9001"), "venue", "category"); err != nil {
		t.Fatal(err)
	}
	if err := p.Set("The lobby.", json.RawMessage(`{"reach":"code","roles":{"start":["trusted"]}}`), "channels", "9002"); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteDoc(filepath.Join(root, "config", "projects", "demo.json"), p); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(root, config.AnyCredential); err != nil {
		t.Fatalf("the set-up root does not load: %v", err)
	}
	return root
}

func export(t *testing.T, root string) []byte {
	t.Helper()
	c, err := config.Load(root, config.AnyCredential)
	if err != nil {
		t.Fatal(err)
	}
	b, err := config.Export(c)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// hostNeutral is a loaded configuration with the host-bound keys and the instance name zeroed.
func hostNeutral(c *config.Config) (config.Instance, []config.Project) {
	in := c.Instance
	in.InstanceName, in.Engine.Path, in.Engine.GitBashPath, in.Credentials.KeyFile, in.Paths.Install = "", "", "", "", ""
	in.Discord.BotCredentialID, in.Engine.FakeFixtures, in.Audit.FileSink = "", nil, nil
	ps := append([]config.Project(nil), c.Projects...)
	for i := range ps {
		ps[i].Repo.Canonical, ps[i].Repo.GitCredentialID = "", ""
	}
	return in, ps
}

func TestExportRoundTrip(t *testing.T) {
	src := setupRoot(t)
	doc := export(t, src)
	e, err := config.ParseExport(doc)
	if err != nil {
		t.Fatalf("the export does not parse: %v\n%s", err, doc)
	}
	dst := filepath.ToSlash(t.TempDir())
	wrote, err := config.Import(dst, e, config.ImportOptions{InstanceName: "second"})
	if err != nil {
		t.Fatal(err)
	}
	if len(wrote) != 2 || wrote[0].String() != "config/instance.json" || wrote[1].String() != "config/projects/demo.json" {
		t.Errorf("wrote %v", wrote)
	}
	// The host-bound keys are empty until this host fills them; repo.canonical is required.
	if _, err := config.Load(dst, config.AnyCredential); err == nil || !strings.Contains(err.Error(), "repo.canonical: missing") {
		t.Errorf("the import loads before repo.canonical is filled: %v", err)
	}
	pd, _, err := config.ReadDoc(filepath.Join(dst, "config", "projects", "demo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if pd.String("repo", config.SetupURLKey) != "https://example.com/demo.git" {
		t.Error("the repository URL did not travel")
	}
	tmpl := config.SetupProject(config.SetupProjectOptions{ID: "demo", Canonical: dst + "/projects/demo"})
	if filled, err := pd.FillEmpty(tmpl, config.HostBoundProject); err != nil || strings.Join(filled, ",") != "repo.canonical" {
		t.Fatalf("filled %v, %v", filled, err)
	}
	if err := config.WriteDoc(filepath.Join(dst, "config", "projects", "demo.json"), pd); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(dst, config.AnyCredential)
	if err != nil {
		t.Fatalf("config check fails after the host-bound keys are filled: %v", err)
	}
	want, _ := config.Load(src, config.AnyCredential)
	wi, wp := hostNeutral(want)
	gi, gp := hostNeutral(got)
	if !reflect.DeepEqual(wi, gi) || !reflect.DeepEqual(wp, gp) {
		t.Errorf("a non-secret key differs:\n%+v\n%+v\n%+v\n%+v", wi, gi, wp, gp)
	}
	if got.Instance.InstanceName != "second" || got.Instance.Discord.GuildID != "3000" || got.Projects[0].Venue.Category != "9001" {
		t.Errorf("imported %+v", got.Instance)
	}
	// Comments travel.
	if b, _ := os.ReadFile(config.InstancePath(dst)); !bytes.Contains(b, []byte(`"//"`)) {
		t.Error("the instance's comments did not travel")
	}

	// A second import into the same root adds nothing.
	if wrote, err := config.Import(dst, e, config.ImportOptions{InstanceName: "second"}); err != nil || len(wrote) != 0 {
		t.Errorf("a second import wrote %v, %v", wrote, err)
	}
}

func TestExportHoldsNoSecret(t *testing.T) {
	src := setupRoot(t)
	doc := export(t, src)
	for _, bad := range []string{"cred-", src, "first.key", "claude.exe", "bash.exe", "/install", "ghp_secret1", "user:",
		"audit.jsonl", "fixtures/engine"} {
		if bytes.Contains(doc, []byte(bad)) {
			t.Errorf("the export holds %q:\n%s", bad, doc)
		}
	}
	e, err := config.ParseExport(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range config.HostBoundInstance {
		want := `""`
		if p[0] == "audit" || p[1] == "fakeFixtures" {
			want = "null"
		}
		if v, ok := e.Instance.Get(p...); !ok || string(v) != want {
			t.Errorf("%v is %s, want %s", p, v, want)
		}
	}
	for _, p := range config.HostBoundProject {
		if v, ok := e.Projects["demo"].Get(p...); !ok || string(v) != `""` {
			t.Errorf("%v is %s", p, v)
		}
	}
	if u := e.Projects["demo"].String("repo", config.SetupURLKey); u != "https://example.com/demo.git" {
		t.Errorf("the repository URL travelled as %q", u)
	}
	for _, tc := range []struct {
		url  string
		cred bool
	}{
		{"https://user:pw@example.com/o/r.git", true}, {"https://ghp_x@example.com/o/r.git", true},
		{"ssh://git:pw@example.com/o/r.git", true}, {"ssh://git@example.com/o/r.git", false},
		{"git@example.com:o/r.git", false}, {"C:/repos/r.git", false}, {"https://example.com/o/r.git", false},
	} {
		if got := config.RepoURLCredential(tc.url); got != tc.cred {
			t.Errorf("RepoURLCredential(%q) = %v", tc.url, got)
		}
	}
}

// TestImportKeepsPresentValues: an import into a configured root adds absent keys only and
// reports a present key holding another value as ignored.
func TestImportKeepsPresentValues(t *testing.T) {
	src := setupRoot(t)
	e, err := config.ParseExport(export(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Instance.Set("", config.RawValue("3100"), "discord", "guildId"); err != nil {
		t.Fatal(err)
	}
	wrote, err := config.Import(src, e, config.ImportOptions{InstanceName: "first"})
	if err != nil {
		t.Fatal(err)
	}
	if len(wrote) != 1 || wrote[0].File != "config/instance.json" || len(wrote[0].Keys) != 0 ||
		strings.Join(wrote[0].Ignored, ",") != "discord.guildId" {
		t.Errorf("wrote %+v", wrote)
	}
	c, _ := config.Load(src, config.AnyCredential)
	if c.Instance.Discord.GuildID != "3000" {
		t.Errorf("the import replaced guildId: %s", c.Instance.Discord.GuildID)
	}
}

// TestParseExportRefusals: each problem names the part and the key.
func TestParseExportRefusals(t *testing.T) {
	doc := string(export(t, setupRoot(t)))
	for _, tc := range []struct{ name, from, to, want string }{
		{"unknown key", `"guildId": "3000"`, `"guildId": "3000", "voiceId": "1"`, "instance: unknown key discord.voiceId"},
		{"newer schema", `"schemaVersion": 1`, `"schemaVersion": 2`, "instance: schemaVersion: 2, newer than this binary's 1"},
		{"older schema", `"schemaVersion": 1`, `"schemaVersion": 0`, "instance: schemaVersion: 0, want 1"},
		{"wrong type", `"applicationId": "4000"`, `"applicationId": 4000`, "instance: discord.applicationId: want a string, got number"},
		{"project unknown key", `"handoff": "push"`, `"handoff": "push", "lobby": 1`, "projects/demo: unknown key lobby"},
		{"project id", `"demo": {`, `"../demo": {`, `projects/../demo: id "../demo": want`},
		{"project id differs", `"id": "demo"`, `"id": "other"`, `projects/demo: id "other": the file must be named other.json`},
		{"instance name", `"instanceName": "first"`, `"instanceName": "../../evil"`, `instance: instanceName "../../evil": `},
		{"mode", `"mode": "dry-run"`, `"mode": "liv"`, `instance: discord.mode: "liv", want live or dry-run`},
		{"host-bound key", `"keyFile": ""`, `"keyFile": "C:/elsewhere/x.key"`,
			`instance: credentials.keyFile: "C:/elsewhere/x.key", want "" (a host-bound key: this host fills it)`},
		{"host-bound null key", `"fileSink": null`, `"fileSink": "/var/log/a.jsonl"`, `instance: audit.fileSink: "/var/log/a.jsonl", want null`},
		{"credential in the URL", `"//url": "https://example.com/demo.git"`, `"//url": "https://u:p@example.com/demo.git"`,
			"projects/demo: repo.//url: the URL carries a credential"},
		{"unknown role group", `"start": [`, `"start": ["nobody",`, `projects/demo: channels.9002.roles.start: "nobody" is neither`},
		{"top-level key", `"projects": {`, `"store": {}, "projects": {`, "export: unknown key store"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(doc, tc.from) {
				t.Fatalf("the export lacks %q", tc.from)
			}
			_, err := config.ParseExport([]byte(strings.Replace(doc, tc.from, tc.to, 1)))
			var inv *config.InvalidError
			if !errors.As(err, &inv) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("got %v, want %q", err, tc.want)
			}
		})
	}
}
