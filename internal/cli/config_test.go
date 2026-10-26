// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasondorjeshort/broodforge/internal/cli/clitest"
)

// The P4 Proof (08-implementation-plan.md), on the built binary against tests/config/.

// Paths are given and expected in the slash form the config commands print on every host.
func configTree(name string) string { return "../../tests/config/" + name }

func TestProofConfigCheck(t *testing.T) {
	want(t, clitest.Run(t, nil, "config", "check", "--root", configTree("valid")), 0, "ok  1  2\n", "")
	want(t, clitest.Run(t, nil, "--json", "config", "check", "--root", configTree("valid")), 0,
		`{"result":"ok","projects":1,"channels":2,"warnings":[]}`+"\n", "")

	want(t, clitest.Run(t, nil, "config", "check", "--root", configTree("dup-channel")), 1, "",
		"error: bad-config: ../../tests/config/dup-channel/config/projects/beta.json: channel 100 is also in "+
			"../../tests/config/dup-channel/config/projects/alpha.json\n")

	want(t, clitest.Run(t, nil, "config", "check", "--root", configTree("unknown-key")), 1, "",
		"error: bad-config: ../../tests/config/unknown-key/config/projects/alpha.json: unknown key channels.100.idleTimeout\n")

	want(t, clitest.Run(t, nil, "config", "check", "--root", configTree("dup-key")), 1, "",
		"error: bad-config: ../../tests/config/dup-key/config/projects/alpha.json: duplicate key channels.100\n")

	r := clitest.Run(t, nil, "config", "check", "--root", t.TempDir())
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: no-config: ") {
		t.Errorf("no config: exit %d stderr %q", r.Code, r.Stderr)
	}
}

func TestProofConfigShowEffective(t *testing.T) {
	want(t, clitest.Run(t, nil, "--root", configTree("valid"), "config", "show", "--effective", "--channel", "100", "--json"), 0,
		`{"id":"100","project":"scratch","reach":"code","roles":{"start":["trusted"],"contribute":["members"],"answer":["driver"]},`+
			`"mode":"prompt","harness":"fake","idleTimeoutMinutes":60,"promptTimeoutMinutes":5,"model":null,"effort":null}`+"\n", "")
	want(t, clitest.Run(t, nil, "--root", configTree("valid"), "config", "show", "--effective"), 0,
		`100  scratch  code  prompt  fake  60  5  -  -  {"start":["trusted"],"contribute":["members"],"answer":["driver"]}`+"\n"+
			`200  scratch  read-only  policy  fake  30  2  sonnet  low  {"start":["10"],"contribute":["10"],"answer":["10"]}`+"\n", "")
	want(t, clitest.Run(t, nil, "--root", configTree("valid"), "config", "show", "--effective", "--limit", "1", "--project", "scratch"), 0,
		`100  scratch  code  prompt  fake  60  5  -  -  {"start":["trusted"],"contribute":["members"],"answer":["driver"]}`+"\n", "")
	want(t, clitest.Run(t, nil, "--root", configTree("valid"), "config", "show", "--channel", "999"), 1, "",
		"error: unknown-channel: no channel 999 in ../../tests/config/valid/config\n")
	want(t, clitest.Run(t, nil, "--root", configTree("valid"), "config", "show", "--project", "nope"), 1, "",
		"error: unknown-project: no project nope in ../../tests/config/valid/config\n")
}

func TestConfigShowFiles(t *testing.T) {
	r := clitest.Run(t, nil, "--root", configTree("valid"), "--json", "config", "show", "--channel", "200")
	if r.Code != 0 || strings.Count(r.Stdout, "\n") != 1 || !strings.Contains(r.Stdout, `"canonical":"<root>/projects/scratch"`) ||
		strings.Contains(r.Stdout, `"//"`) {
		t.Errorf("show --channel 200: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
	r = clitest.Run(t, nil, "--root", configTree("valid"), "config", "show")
	if r.Code != 0 || strings.Count(r.Stdout, "\n") != 2 ||
		!strings.HasPrefix(r.Stdout, "../../tests/config/valid/config/instance.json  {") || !strings.Contains(r.Stdout, `"instanceName":"test-valid"`) {
		t.Errorf("show: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
}

// TestConfigLastGoodAndManyProblems: every problem is its own stderr line, and --last-good
// reads data/config-last-good/ (written here by hand: the bot's write is P10's).
func TestConfigLastGoodAndManyProblems(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	lg := filepath.Join(root, "data", "config-last-good")
	for _, d := range []string{cfg, lg} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	good, err := os.ReadFile(filepath.Join(configTree("valid"), "config", "instance.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lg, "instance.json"), good, 0o600); err != nil {
		t.Fatal(err)
	}
	bad := `{"schemaVersion":2,"instanceName":"x","typo":1,"other":2}`
	if err := os.WriteFile(filepath.Join(cfg, "instance.json"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	ip := filepath.ToSlash(root) + "/config/instance.json"
	want(t, clitest.Run(t, nil, "--root", root, "config", "check"), 1, "",
		"error: bad-config: "+ip+": unknown key other\nerror: bad-config: "+ip+": unknown key typo\n")
	r := clitest.Run(t, nil, "--root", root, "--json", "config", "check")
	if r.Code != 1 || strings.Count(r.Stderr, `{"code":"bad-config","message":`) != 2 {
		t.Errorf("--json: exit %d stderr %q", r.Code, r.Stderr)
	}
	r = clitest.Run(t, nil, "--root", root, "config", "show", "--last-good")
	if r.Code != 0 || !strings.Contains(r.Stdout, `"instanceName":"test-valid"`) {
		t.Errorf("--last-good: exit %d stdout %q stderr %q", r.Code, r.Stdout, r.Stderr)
	}
}

// TestConfigExportImport is P24's surface on the built binary: config export to its default
// path and to --out, config import into an empty root, config-exists on a configured one, and
// bad-config naming the key of an export this binary's schema rejects.
func TestConfigExportImport(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	src := dir + "/src"
	if err := os.CopyFS(src, os.DirFS(configTree("valid"))); err != nil {
		t.Fatal(err)
	}
	want(t, clitest.Run(t, nil, "config", "export", "--root", src), 0, "written  "+src+"/data/config-export.json\n", "")
	want(t, clitest.Run(t, nil, "--json", "config", "export", "--root", src, "--out", dir+"/e.json"), 0,
		`{"path":"`+dir+`/e.json","result":"written"}`+"\n", "")
	a, _ := os.ReadFile(src + "/data/config-export.json")
	b, _ := os.ReadFile(dir + "/e.json")
	if len(a) == 0 || string(a) != string(b) {
		t.Errorf("the two exports differ:\n%s\n%s", a, b)
	}
	want(t, clitest.Run(t, nil, "config", "import", "--root", dir+"/dst", dir+"/e.json"), 0,
		"written  "+dir+"/dst/config/instance.json\nwritten  "+dir+"/dst/config/projects/scratch.json\n", "")
	want(t, clitest.Run(t, nil, "config", "import", "--root", dir+"/dst", dir+"/e.json"), 1, "",
		"error: config-exists: "+dir+"/dst/config/instance.json exists\n")
	if err := os.WriteFile(dir+"/bad.json", []byte(strings.Replace(string(b), `"mode": "dry-run"`, `"mode": "dry-run", "shard": 1`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	want(t, clitest.Run(t, nil, "config", "import", "--root", dir+"/bad", dir+"/bad.json"), 1, "",
		"error: bad-config: "+dir+"/bad.json: instance: unknown key discord.shard\n")
	if _, err := os.Stat(dir + "/bad"); err == nil {
		t.Error("a refused import wrote the root")
	}
	// A value config check refuses is refused before anything is written.
	if err := os.WriteFile(dir+"/evil.json", []byte(strings.Replace(string(b), `"instanceName": "test-valid"`, `"instanceName": "../../evil"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	r := clitest.Run(t, nil, "config", "import", "--root", dir+"/evil", dir+"/evil.json")
	if r.Code != 1 || !strings.HasPrefix(r.Stderr, "error: bad-config: "+dir+"/evil.json: instance: instanceName \"../../evil\": ") {
		t.Errorf("evil instanceName: exit %d stderr %q", r.Code, r.Stderr)
	}
	if _, err := os.Stat(dir + "/evil"); err == nil {
		t.Error("a refused import wrote the root")
	}
}
