// SPDX-License-Identifier: GPL-3.0-or-later

package setup_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/git/gittest"
	"github.com/jasondorjeshort/broodforge/internal/setup"
)

// setup --from (08-implementation-plan-setup.md, P24): a set-up root exported, then a fresh root
// set up from the export with --yes against the same scripted Discord: the host-bound and secret
// steps ok, the imported ones skip, every non-secret key equal, no secret carried.

// exportRoot writes root's config export to a file under dir and returns its path.
func exportRoot(t *testing.T, root, dir string) string {
	t.Helper()
	c, err := config.Load(root, config.AnyCredential)
	if err != nil {
		t.Fatal(err)
	}
	b, err := config.Export(c)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "e.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFrom(t *testing.T) {
	first := withFake(newFixture(t), fresh())
	if out, c := first.run(tokA + "\n"); c != "" {
		t.Fatalf("the first setup refused %s:\n%s", c, out)
	}
	// A git credential on the first machine, so the export has one to leave behind.
	if out, c := first.run("a-git-token", func(o *setup.Options) { o.Step = "git-credential" }); c != "" {
		t.Fatalf("setup git-credential refused %s:\n%s", c, out)
	}
	export := exportRoot(t, first.root, first.dir)
	doc, _ := os.ReadFile(export)
	firstKey, _ := os.ReadFile(filepath.Join(filepath.Dir(first.root), first.instance+".key"))
	for _, bad := range []string{"cred-", tokA, "a-git-token", strings.TrimSpace(string(firstKey)), first.root, first.dir} {
		if bytes.Contains(doc, []byte(bad)) {
			t.Fatalf("the export holds %q:\n%s", bad, doc)
		}
	}

	second := newFixture(t)
	second.fake = first.fake // the same application and guild
	calls := len(first.fake.Script().Calls)
	keyFile := filepath.Join(filepath.Dir(second.root), second.instance+".key")
	from := func(o *setup.Options) { o.From = export; o.Repo = "" }
	out, c := second.run(tokA+"\n", from)
	if c != "" {
		t.Fatalf("setup --from refused %s:\n%s", c, out)
	}
	want := []string{"ok root", "ok config", "ok key", "ok engine", "ok token", "skip application", "skip guild",
		"skip channels", "skip role", "skip git-credential", "ok project", "skip defaults"}
	if got := results(out); strings.Join(got[:len(want)], ",") != strings.Join(want, ",") {
		t.Fatalf("lines:\n%s\nwant %v", out, want)
	}
	if l := line(out, "ok  config  "); l != "ok  config  imported config/instance.json; config/projects/demo.json from "+filepath.ToSlash(export) {
		t.Errorf("config line %q", l)
	}
	if n := len(first.fake.Script().Calls); n != calls {
		t.Errorf("setup --from wrote to Discord: %v", first.fake.Script().Calls[calls:])
	}

	a, err := config.Load(first.root, config.AnyCredential)
	if err != nil {
		t.Fatal(err)
	}
	b, err := config.Load(second.root, config.AnyCredential)
	if err != nil {
		t.Fatalf("config check fails on the second root: %v", err)
	}
	if b.Instance.InstanceName != second.instance || b.Instance.Credentials.KeyFile != filepath.ToSlash(keyFile) ||
		b.Projects[0].Repo.Canonical != second.root+"/projects/demo" || b.Projects[0].Repo.GitCredentialID != "" {
		t.Errorf("host-bound keys: %+v %+v", b.Instance, b.Projects[0].Repo)
	}
	// Every non-secret key equal: the instances and projects with the host-bound keys and
	// the instance name set aside.
	ai, bi := a.Instance, b.Instance
	for _, in := range []*config.Instance{&ai, &bi} {
		in.InstanceName, in.Engine.Path, in.Credentials.KeyFile, in.Paths.Install, in.Discord.BotCredentialID = "", "", "", "", ""
	}
	ap, bp := a.Projects[0], b.Projects[0]
	for _, p := range []*config.Project{&ap, &bp} {
		p.Repo.Canonical, p.Repo.GitCredentialID = "", ""
	}
	if !reflect.DeepEqual(ai, bi) || !reflect.DeepEqual(ap, bp) {
		t.Errorf("a non-secret key differs:\n%+v\n%+v\n%+v\n%+v", ai, bi, ap, bp)
	}
	// The second root has its own key and only its own bot token: nothing of the first's store.
	secondKey, err := os.ReadFile(keyFile)
	if err != nil || bytes.Equal(secondKey, firstKey) {
		t.Errorf("the second key: %v, same as the first: %v", err, bytes.Equal(secondKey, firstKey))
	}
	if n, err := credentials.Entries(credentials.StorePath(second.root)); err != nil || n != 1 {
		t.Errorf("the second store holds %d entries, %v", n, err)
	}
	if b.Instance.Discord.BotCredentialID == a.Instance.Discord.BotCredentialID {
		t.Error("the bot credential id travelled")
	}
	// The export's copy in the root's data/.
	if copied, err := os.ReadFile(filepath.Join(second.root, "data", config.ExportName)); err != nil || !bytes.Equal(copied, doc) {
		t.Errorf("no copy of the export in data/: %v", err)
	}

	// The rerun with --from, the export changed by hand: only ok and skip, config/ unchanged,
	// and the config line names the key it kept over the export's.
	before := snapshot(t, second.root)
	changed := bytes.Replace(doc, []byte(`"noticeChannelId": "9003"`), []byte(`"noticeChannelId": "9999"`), 1)
	if err := os.WriteFile(export, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	out, c = second.run("", from)
	if c != "" {
		t.Fatalf("the rerun refused %s:\n%s", c, out)
	}
	for _, r := range results(out) {
		if !strings.HasPrefix(r, "ok ") && !strings.HasPrefix(r, "skip ") {
			t.Errorf("rerun printed %q:\n%s", r, out)
		}
	}
	if l := line(out, "skip  config  "); !strings.HasSuffix(l, "; kept config/instance.json discord.noticeChannelId over the export's (setup <step> replaces a key)") {
		t.Errorf("rerun: config line %q:\n%s", l, out)
	}
	if snapshot(t, second.root) != before {
		t.Error("the rerun changed config/ or the key")
	}
}

// TestFromRefusals: an export this binary's schema rejects is mismatch-config naming the key,
// before anything is written; one from a newer schema version likewise.
func TestFromRefusals(t *testing.T) {
	src := newFixture(t)
	if out, c := src.run(""); c != "" {
		t.Fatalf("setup refused %s:\n%s", c, out)
	}
	export := exportRoot(t, src.root, src.dir)
	doc, _ := os.ReadFile(export)
	for _, tc := range []struct{ name, from, to, want string }{
		{"unknown key", `"guildId": ""`, `"guildId": "", "voiceId": "1"`, "instance: unknown key discord.voiceId"},
		{"newer schema", `"schemaVersion": 1`, `"schemaVersion": 2`, "instance: schemaVersion: 2, newer than this binary's 1"},
		{"host-bound key", `"keyFile": ""`, `"keyFile": "C:/elsewhere/x.key"`,
			`instance: credentials.keyFile: "C:/elsewhere/x.key", want "" (a host-bound key: this host fills it)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Contains(doc, []byte(tc.from)) {
				t.Fatalf("the export lacks %q", tc.from)
			}
			bad := filepath.Join(t.TempDir(), "bad.json")
			if err := os.WriteFile(bad, bytes.Replace(doc, []byte(tc.from), []byte(tc.to), 1), 0o600); err != nil {
				t.Fatal(err)
			}
			f := newFixture(t)
			var out syncBuffer
			err := setup.Run(context.Background(), f.opts(func(o *setup.Options) { o.From = bad }), f.env(strings.NewReader(""), &out))
			var ce *control.Error
			if !errors.As(err, &ce) || ce.Code != "mismatch-config" || !strings.Contains(ce.Message, tc.want) ||
				ce.Message != "export "+filepath.ToSlash(bad)+" has "+tc.want+"; fix the file or export it again with this version" {
				t.Errorf("got %v, want mismatch-config naming %q", err, tc.want)
			}
			if _, err := os.Stat(filepath.Dir(f.root)); err == nil {
				t.Error("the refused import created the root")
			}
		})
	}
}

// TestFromSettlesTheProjectFirst: an export whose project is not the configured root's leaves two
// projects and no --project: needs-input-project, refused before the import writes anything, so
// the root still loads.
func TestFromSettlesTheProjectFirst(t *testing.T) {
	f := newFixture(t)
	if out, c := f.run(""); c != "" {
		t.Fatalf("setup refused %s:\n%s", c, out)
	}
	other := newFixture(t)
	_, other.repo = gittest.Project(t, filepath.Join(other.dir, "src2"), "other")
	if out, c := other.run(""); c != "" {
		t.Fatalf("setup of other refused %s:\n%s", c, out)
	}
	export := exportRoot(t, other.root, other.dir)
	before := snapshot(t, f.root)
	out, c := f.run("", func(o *setup.Options) { o.From = export; o.Repo = "" })
	if c != "needs-input-project" {
		t.Fatalf("code %q:\n%s", c, out)
	}
	if _, err := os.Stat(filepath.Join(f.root, "config", "projects", "other.json")); err == nil {
		t.Error("the refused run wrote config/projects/other.json")
	}
	if snapshot(t, f.root) != before {
		t.Error("the refused run changed config/")
	}
	if _, err := config.Load(f.root, config.AnyCredential); err != nil {
		t.Errorf("the root no longer loads: %v", err)
	}
}

// TestFromTwoProjects: an export with two projects sets both up on the new root, each canonical
// filled and cloned, the second through its kept repository URL.
func TestFromTwoProjects(t *testing.T) {
	src := newFixture(t)
	if out, c := src.run(""); c != "" {
		t.Fatalf("setup refused %s:\n%s", c, out)
	}
	_, otherRepo := gittest.Project(t, filepath.Join(src.dir, "src2"), "other")
	if out, c := src.run("", func(o *setup.Options) { o.Repo = otherRepo }); c != "" {
		t.Fatalf("setup of the second project refused %s:\n%s", c, out)
	}
	export := exportRoot(t, src.root, src.dir)
	dst := newFixture(t)
	from := func(o *setup.Options) { o.From = export; o.Repo = ""; o.Project = "demo" }
	out, c := dst.run("", from)
	if c != "" {
		t.Fatalf("setup --from refused %s:\n%s", c, out)
	}
	cfg, err := config.Load(dst.root, config.AnyCredential)
	if err != nil {
		t.Fatalf("config check fails: %v", err)
	}
	if len(cfg.Projects) != 2 {
		t.Fatalf("projects %+v", cfg.Projects)
	}
	for _, p := range cfg.Projects {
		if p.Repo.Canonical != dst.root+"/projects/"+p.ID || p.Repo.DefaultBranch != "main" {
			t.Errorf("project %s: %+v", p.ID, p.Repo)
		}
		if _, err := os.Stat(filepath.Join(dst.root, "projects", p.ID, "README")); err != nil {
			t.Errorf("project %s is not cloned: %v", p.ID, err)
		}
		if !strings.Contains(out, "ok  project  "+dst.root+"/projects/"+p.ID+"  main\n") {
			t.Errorf("no project line for %s:\n%s", p.ID, out)
		}
	}
	// The rerun: only ok and skip.
	out, c = dst.run("", from)
	if c != "" {
		t.Fatalf("rerun refused %s:\n%s", c, out)
	}
	for _, r := range results(out) {
		if !strings.HasPrefix(r, "ok ") && !strings.HasPrefix(r, "skip ") {
			t.Errorf("rerun printed %q:\n%s", r, out)
		}
	}
}

// TestRepoURLCredentialRefused: a repository URL carrying a credential is refused before any
// file is written.
func TestRepoURLCredentialRefused(t *testing.T) {
	f := newFixture(t)
	var out syncBuffer
	err := setup.Run(context.Background(), f.opts(func(o *setup.Options) { o.Repo = "https://user:ghp_x@example.com/o/demo.git" }),
		f.env(strings.NewReader(""), &out))
	var ce *control.Error
	if !errors.As(err, &ce) || ce.Code != "bad-args" || !strings.Contains(ce.Message, "add the token with setup git-credential") {
		t.Fatalf("got %v:\n%s", err, out.String())
	}
	if ents, _ := os.ReadDir(filepath.Join(f.root, "config", "projects")); len(ents) > 0 {
		t.Errorf("a project file was written: %v", ents)
	}
}
