// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"strings"
	"testing"
)

// A trailing YAML comment is not part of a value: on a scalar, a comma list, an inline list
// and a block list's items; a # inside quotes or inside a word stays.
func TestFrontmatterTrailingComments(t *testing.T) {
	fm, body, ok := frontmatter(strings.Join([]string{"---",
		"name: checker # the reader",
		"tools: Read, Grep # no Bash",
		"model: \"opus # not a comment\"",
		"effort: high#tag",
		"list: [a, b] # two",
		"items:",
		"  - Read # first",
		"  - 'Grep # quoted'",
		"# a whole-line comment: here",
		"---", "Body."}, "\n"))
	if !ok || body != "Body." {
		t.Fatalf("ok %v body %q", ok, body)
	}
	for key, want := range map[string]string{"name": "checker", "model": "opus # not a comment", "effort": "high#tag",
		"tools": "Read, Grep", "list": "a, b", "items": "Read, Grep # quoted"} {
		if got := fm[key].str(); got != want {
			t.Errorf("%s: got %q, want %q", key, got, want)
		}
	}
	if got := strings.Join(fm["tools"].items(), ","); got != "Read,Grep" {
		t.Errorf("tools items %q", got)
	}
	if _, ok := fm["# a whole-line comment"]; ok || len(fm) != 6 {
		t.Errorf("fields %v", fm)
	}
}
