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
