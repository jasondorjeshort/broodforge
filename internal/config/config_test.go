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
	"github.com/jasondorjeshort/broodforge/internal/control"
)

// tree is tests/config/<name> from this package's directory.
// Paths are given and expected in the slash form config prints on every host.
func tree(name string) string { return "../../tests/config/" + name }

func ptr[T string | int](v T) *T { return &v }

func TestValidTree(t *testing.T) {
	c, err := config.Load(tree("valid"), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Instance.InstanceName != "test-valid" || len(c.Projects) != 1 || c.Projects[0].ID != "scratch" {
		t.Fatalf("instance %q, projects %d", c.Instance.InstanceName, len(c.Projects))
	}
	if got := c.Projects[0].HandoffOrDefault(); got != config.HandoffPush {
		t.Errorf("handoff %q", got)
	}
	if len(c.Files) != 2 || c.Files[0].Name != "instance.json" || c.Files[1].Name != "projects/scratch.json" {
		t.Errorf("files %+v", c.Files)
	}
	want := []config.EffectiveChannel{
		{ID: "100", Project: "scratch", Reach: "code",
			Roles: config.Roles{Start: []string{"trusted"}, Contribute: []string{"members"}, Answer: []string{"driver"}},
			Mode:  "prompt", Harness: "fake", IdleTimeoutMinutes: 60, PromptTimeoutMinutes: 5},
		{ID: "200", Project: "scratch", Reach: "read-only",
			Roles: config.Roles{Start: []string{"10"}, Contribute: []string{"10"}, Answer: []string{"10"}},
			Mode:  "policy", Harness: "fake", IdleTimeoutMinutes: 30, PromptTimeoutMinutes: 2,
			Model: ptr("sonnet"), Effort: ptr("low")},
	}
	if got := c.Channels(); !reflect.DeepEqual(got, want) {
		t.Errorf("Channels:\n got %+v\nwant %+v", got, want)
	}
	b, err := json.Marshal(c.Effective(c.ChannelProject("100"), "100"))
	if err != nil {
		t.Fatal(err)
	}
	const wantJSON = `{"id":"100","project":"scratch","reach":"code","roles":{"start":["trusted"],"contribute":["members"],"answer":["driver"]},"mode":"prompt","harness":"fake","idleTimeoutMinutes":60,"promptTimeoutMinutes":5,"model":null,"effort":null}`
	if string(b) != wantJSON {
		t.Errorf("Effective 100:\n got %s\nwant %s", b, wantJSON)
	}
	if bytes.Contains(c.Files[0].Stripped, []byte(`"//"`)) || bytes.Contains(c.Files[1].Stripped, []byte(`"//"`)) {
		t.Errorf("a comment key survived stripping")
	}
}

func problems(t *testing.T, err error) []string {
	t.Helper()
	var inv *config.InvalidError
	if !errors.As(err, &inv) {
		t.Fatalf("want *InvalidError, got %v", err)
	}
	out := make([]string, len(inv.Problems))
	for i, p := range inv.Problems {
		out[i] = p.String()
	}
	return out
}

func TestDupChannelTree(t *testing.T) {
	_, err := config.Load(tree("dup-channel"), nil)
	want := []string{"../../tests/config/dup-channel/config/projects/beta.json: channel 100 is also in ../../tests/config/dup-channel/config/projects/alpha.json"}
	if got := problems(t, err); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestUnknownKeyTree(t *testing.T) {
	_, err := config.Load(tree("unknown-key"), nil)
	want := []string{"../../tests/config/unknown-key/config/projects/alpha.json: unknown key channels.100.idleTimeout"}
	if got := problems(t, err); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

const minimalInstance = `{"schemaVersion":1,"instanceName":"t","discord":{"mode":"dry-run"},"spawn":{"backend":"same-account"},
"roleGroups":{"trusted":["10"]},"defaults":{"idleTimeoutMinutes":60,"promptTimeoutMinutes":5,"mode":"prompt","harness":"fake"}}`

const minimalProject = `{"id":"p","name":"P","repo":{"canonical":"c"},"channels":{"100":{"reach":"code","roles":{"start":["trusted"]}}}}`

// root writes config/instance.json and config/projects/<name> for each project given and
// returns the root.
func root(t *testing.T, instance string, projects map[string]string) string {
	t.Helper()
	r := t.TempDir()
	if err := os.MkdirAll(filepath.Join(r, "config", "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r, "config", "instance.json"), []byte(instance), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, body := range projects {
		if err := os.WriteFile(filepath.Join(r, "config", "projects", name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestProblems(t *testing.T) {
	inst := func(old, new string) string { return strings.Replace(minimalInstance, old, new, 1) }
	proj := func(old, new string) string { return strings.Replace(minimalProject, old, new, 1) }
	cases := []struct {
		name     string
		instance string
		project  string // as config/projects/p.json; "" for none
		file     string // "i" or "p": which file the problems name
		want     []string
	}{
		{"valid", minimalInstance, minimalProject, "", nil},
		{"no projects", minimalInstance, "", "", nil},
		{"comments everywhere", inst(`"schemaVersion":1`, `"//a":{"x":1},"schemaVersion":1`),
			proj(`"reach":"code"`, `"//":"c","reach":"code"`), "", nil},
		{"comment key as a channel", minimalInstance, proj(`"100":`, `"// old 999": {"bogus": 1}, "100":`), "", nil},
		{"repeated comment keys", inst(`"schemaVersion":1`, `"//":"a","//":"b","schemaVersion":1`), minimalProject, "", nil},
		{"byte order mark", "\xef\xbb\xbf" + minimalInstance, minimalProject, "", nil},
		{"duplicate keys", minimalInstance, proj(`"100":{`, `"100":{"reach":"code"},"100":{"reach":"code",`), "p",
			[]string{"duplicate key channels.100", "duplicate key channels.100.reach"}},
		{"syntax after blank lines", "\n\n{\n  oops\n}", minimalProject, "i",
			[]string{"invalid JSON at line 4, column 3: invalid character 'o' looking for beginning of object key string"}},
		{"type in an array", inst(`"trusted":["10"]`, `"trusted":[10]`), minimalProject, "i",
			[]string{"roleGroups.trusted.0: want a string, got number"}},
		{"syntax", "{\n  \"schemaVersion\": 1,\n  oops\n}", minimalProject, "i",
			[]string{"invalid JSON at line 3, column 3: invalid character 'o' looking for beginning of object key string"}},
		{"not an object", "[1]", minimalProject, "i", []string{"want a JSON object"}},
		{"empty", "  ", minimalProject, "i", []string{"empty file"}},
		{"unknown keys, all of them, exact case", inst(`"schemaVersion":1`, `"schemaVersion":1,"InstanceName":"x","zzz":1`),
			minimalProject, "i", []string{"unknown key InstanceName", "unknown key zzz"}},
		{"wrong type", inst(`"idleTimeoutMinutes":60`, `"idleTimeoutMinutes":"60"`), minimalProject, "i",
			[]string{"defaults.idleTimeoutMinutes: want an integer, got string"}},
		{"schema and name", inst(`"schemaVersion":1,"instanceName":"t"`, `"schemaVersion":2,"instanceName":"a/b"`), minimalProject, "i",
			[]string{"schemaVersion: 2, want 1", `instanceName "a/b": want 1 to 64 of [A-Za-z0-9._-], starting with a letter or digit`}},
		{"enums", inst(`"mode":"dry-run"`, `"mode":"test"`), minimalProject, "i",
			[]string{`discord.mode: "test", want live or dry-run`}},
		{"live needs ids", inst(`"mode":"dry-run"`, `"mode":"live"`), minimalProject, "i",
			[]string{"discord.applicationId: missing", "discord.guildId: missing", "discord.botCredentialId: missing"}},
		{"missing defaults", inst(`,"defaults":{"idleTimeoutMinutes":60,"promptTimeoutMinutes":5,"mode":"prompt","harness":"fake"}`, ``), minimalProject, "i",
			[]string{"defaults.idleTimeoutMinutes: 0, want a positive number", "defaults.promptTimeoutMinutes: 0, want a positive number",
				"defaults.mode: missing, want prompt or policy", "defaults.harness: missing, want claude-code or fake"}},
		{"group names", inst(`"trusted":["10"]`, `"trusted":["x"],"12":["10"]`), minimalProject, "i",
			[]string{"roleGroups.12: a group name must not be a role id or \"driver\"", "roleGroups.trusted: \"x\" is not a role id"}},
		{"file name", minimalInstance, proj(`"id":"p"`, `"id":"q"`), "p", []string{`id "q": the file must be named q.json`}},
		{"project id rule", minimalInstance, proj(`"id":"p"`, `"id":"P"`), "p", []string{`id "P": want ^[a-z][a-z0-9-]{0,31}$`}},
		{"project required", minimalInstance, proj(`"name":"P","repo":{"canonical":"c"}`, `"handoff":"merge"`), "p",
			[]string{"name: missing", "repo.canonical: missing", `handoff: "merge", want pull-request or push`}},
		{"channel", minimalInstance, proj(`"100":{"reach":"code","roles":{"start":["trusted"]}}`,
			`"x1":{"reach":"all","mode":"auto","idleTimeoutMinutes":0,"roles":{"start":["driver","nobody","20"]}}`), "p",
			[]string{"channels.x1: a channel id is decimal digits", `channels.x1.reach: "all", want read-only or code`,
				`channels.x1.mode: "auto", want prompt or policy`, "channels.x1.idleTimeoutMinutes: 0, want a positive number",
				`channels.x1.roles.start: "driver" is allowed under answer only`,
				`channels.x1.roles.start: "nobody" is neither a role id nor a group in roleGroups`}},
		{"channel type", minimalInstance, proj(`"reach":"code"`, `"reach":"code","model":5`), "p",
			[]string{"channels.100.model: want a string, got number"}},
		{"channel model alias and effort", minimalInstance, proj(`"reach":"code"`, `"reach":"code","model":"opus","effort":"high"`), "", nil},
		{"channel model id and null effort", minimalInstance, proj(`"reach":"code"`, `"reach":"code","model":"claude-haiku-5-5-20261001","effort":null`), "", nil},
		{"channel model and effort", minimalInstance, proj(`"reach":"code"`, `"reach":"code","model":"claude-opus-4-1-20250805","effort":"xhigh"`), "p",
			[]string{`channels.100.model: "claude-opus-4-1-20250805", want opus, sonnet, haiku, fable or a model id they name`,
				`channels.100.effort: "xhigh", want low or medium or high or max`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projects := map[string]string{}
			if tc.project != "" {
				projects["p.json"] = tc.project
			}
			r := root(t, tc.instance, projects)
			c, err := config.Load(r, nil)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("Load: %v", err)
				}
				if c.Instance.InstanceName != "t" {
					t.Errorf("instanceName %q", c.Instance.InstanceName)
				}
				return
			}
			path := filepath.ToSlash(r) + "/config/instance.json"
			if tc.file == "p" {
				path = filepath.ToSlash(r) + "/config/projects/p.json"
			}
			want := make([]string, len(tc.want))
			for i, w := range tc.want {
				want[i] = path + ": " + w
			}
			if got := problems(t, err); !reflect.DeepEqual(got, want) {
				t.Errorf("got  %q\nwant %q", got, want)
			}
		})
	}
}

func TestCredentialExistence(t *testing.T) {
	inst := strings.Replace(minimalInstance, `"mode":"dry-run"`, `"mode":"dry-run","botCredentialId":"cred-gone"`, 1)
	proj := strings.Replace(minimalProject, `"canonical":"c"`, `"canonical":"c","gitCredentialId":"cred-ok"`, 1)
	r := root(t, inst, map[string]string{"p.json": proj})
	if _, err := config.Load(r, nil); err != nil {
		t.Fatalf("stub: %v", err)
	}
	_, err := config.Load(r, func(id string) bool { return id == "cred-ok" })
	want := []string{config.InstancePath(r) + ": discord.botCredentialId: credential cred-gone is not in the credential store"}
	if got := problems(t, err); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestNoConfig(t *testing.T) {
	_, err := config.Load(t.TempDir(), nil)
	var nf *config.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want *NotFoundError, got %v", err)
	}
}

func TestInstanceName(t *testing.T) {
	for body, want := range map[string]string{
		`{"instanceName":"pc"}`:                      "",
		`{"//":"x","instanceName":"a.b_c-1","zz":1}`: "",
		`{}`:                     ": no instanceName",
		`{"instanceName":"a/b"}`: `: instanceName "a/b": want 1 to 64 of [A-Za-z0-9._-], starting with a letter or digit`,
		`{"instanceName":5}`:     ": instanceName: want a string",
		`not json`:               ": invalid JSON at line 1, column 2: invalid character 'o' in literal null (expecting 'u')",
	} {
		r := root(t, body, nil)
		name, err := config.InstanceName(r)
		if want == "" {
			if err != nil || name == "" {
				t.Errorf("%s: %q %v", body, name, err)
			}
			continue
		}
		var inv *config.InvalidError
		if !errors.As(err, &inv) || err.Error() != config.InstancePath(r)+want {
			t.Errorf("%s: got %v, want %s", body, err, config.InstancePath(r)+want)
		}
	}
	var nf *config.NotFoundError
	if _, err := config.InstanceName(t.TempDir()); !errors.As(err, &nf) {
		t.Errorf("missing: want *NotFoundError, got %v", err)
	}
}

func TestInstanceNameFallsBackToLastGood(t *testing.T) {
	r := root(t, minimalInstance, nil)
	c, err := config.Load(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteLastGood(r, c); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.InstancePath(r), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if name, err := config.InstanceName(r); err != nil || name != "t" {
		t.Errorf("got %q %v, want t from last-good", name, err)
	}
	if _, err := config.Load(r, nil); err == nil {
		t.Errorf("Load of the bad file succeeded")
	}
}

func TestInstanceNameRuleMatchesControl(t *testing.T) {
	for _, s := range []string{"", "a", "pc", "scratch-0123456789ab", "a.b_c-d", "-a", ".a", "a/b", "a b", "ä",
		strings.Repeat("a", 64), strings.Repeat("a", 65)} {
		if config.ValidInstanceName(s) != control.ValidInstanceName(s) {
			t.Errorf("%q: config %v, control %v", s, config.ValidInstanceName(s), control.ValidInstanceName(s))
		}
	}
}

func TestLastGood(t *testing.T) {
	r := root(t, minimalInstance, map[string]string{"p.json": minimalProject, "notes.txt": "not JSON, and not read"})
	var nf *config.NotFoundError
	if _, err := config.LoadLastGood(r, nil); !errors.As(err, &nf) {
		t.Fatalf("before any write: want *NotFoundError, got %v", err)
	}
	c, err := config.Load(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 { // the second write replaces the first
		if err := config.WriteLastGood(r, c); err != nil {
			t.Fatal(err)
		}
	}
	lg, err := config.LoadLastGood(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(lg.Instance, c.Instance) || !reflect.DeepEqual(lg.Projects, c.Projects) {
		t.Errorf("last-good differs from the load it copied")
	}
	for _, f := range c.Files {
		b, err := os.ReadFile(filepath.Join(config.LastGoodDir(r), filepath.FromSlash(f.Name)))
		if err != nil || !bytes.Equal(b, f.Raw) {
			t.Errorf("%s: not copied byte for byte (%v)", f.Name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(config.LastGoodDir(r), "projects", "notes.txt")); err == nil {
		t.Errorf("a file the load did not read was copied")
	}
	entries, err := os.ReadDir(filepath.Join(r, "data"))
	if err != nil || len(entries) != 1 {
		t.Errorf("data/ holds %d entries, want only config-last-good (%v)", len(entries), err)
	}
	if tm, err := config.LastGoodTime(r); err != nil || tm.IsZero() {
		t.Errorf("LastGoodTime: %v %v", tm, err)
	}
}

// TestLastGoodRecovery: a write cut off between its two renames leaves only
// config-last-good.old-*; the next read renames it back, and the next write clears stale
// .new-* directories.
func TestLastGoodRecovery(t *testing.T) {
	r := root(t, minimalInstance, map[string]string{"p.json": minimalProject})
	c, err := config.Load(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteLastGood(r, c); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(r, "data")
	if err := os.Rename(config.LastGoodDir(r), filepath.Join(data, "config-last-good.old-cut")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(data, "config-last-good.new-cut"), 0o700); err != nil {
		t.Fatal(err)
	}
	lg, err := config.LoadLastGood(r, nil)
	if err != nil || lg.Instance.InstanceName != "t" {
		t.Fatalf("LoadLastGood after a cut write: %v", err)
	}
	if err := config.WriteLastGood(r, c); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(data)
	if err != nil || len(entries) != 1 || entries[0].Name() != "config-last-good" {
		t.Errorf("data/ after the next write: %v %v", entries, err)
	}
}
