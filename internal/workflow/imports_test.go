// SPDX-License-Identifier: GPL-3.0-or-later

package workflow

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The base's imports.js cases (test-hooks.js, group IM), ported: the resolver agrees with the
// parser the base's hooks run.
func TestParseImportsAsImportsJS(t *testing.T) {
	cases := []struct {
		id, text string
		want     []string
	}{
		{"IM01 an @ at line start", "@a.md\n", []string{"a.md"}},
		{"IM02 an @ mid-line after white space", "See @README for more.\n", []string{"README"}},
		{"IM03 an @ in a code span is skipped", "Write `@x.md` or ``a ` @y.md`` here.\n", nil},
		{"IM04 an @ in a fenced block is skipped", "```\n@x.md\n```\n~~~~\n@y.md\n~~~~\n  ```sh\n  @z.md\n  ```\n@after.md\n", []string{"after.md"}},
		{"IM05 trailing . , ; : ) are stripped", "(see @a.md). @b.md, @c.md; @d.md:\n", []string{"a.md", "b.md", "c.md", "d.md"}},
		{"IM06 an @ inside a word is not an import", "mail me" + "@" + "example.com or x@y\n", nil},
		{"IM07 an @ right after a code span", "`x`@a.md\n", nil},
		{"IM10 a ``` line whose info string holds a backtick is not a fence", "``` `x`\n@a.md\n", []string{"a.md"}},
		{"IM11 a code span crossing a line break; a blank line ends it", "Write `foo\n@bar.md` here.\n\n@baz.md\n", []string{"baz.md"}},
	}
	for _, c := range cases {
		var got []string
		for _, it := range parseImports(c.text) {
			got = append(got, it.token)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.id, got, c.want)
		}
	}
}

func put(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveImportsAsImportsJS(t *testing.T) {
	root := t.TempDir()
	put(t, root, "docs/HANDOFF.md", "# H\n@handoff/x.md\n")
	put(t, root, "docs/handoff/x.md", "# X\n@../HANDOFF.md\n")
	for k := 0; k <= 5; k++ {
		put(t, root, "chain/c"+string(rune('0'+k))+".md", "# c\n@c"+string(rune('1'+k))+".md\n")
	}
	// IM08: each import from its own file; a cycle is read once.
	got := resolveImports(root, "docs/HANDOFF.md")
	want := []importRef{
		{file: "docs/HANDOFF.md", line: 2, token: "handoff/x.md", target: "docs/handoff/x.md", exists: true},
		{file: "docs/handoff/x.md", line: 2, token: "../HANDOFF.md", target: "docs/HANDOFF.md", exists: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("IM08: got %+v", got)
	}
	// IM09: four hops; the imports of a hop-4 file are not followed.
	var targets []string
	for _, r := range resolveImports(root, "chain/c0.md") {
		targets = append(targets, r.target)
	}
	if strings.Join(targets, " ") != "chain/c1.md chain/c2.md chain/c3.md chain/c4.md" {
		t.Errorf("IM09: got %q", targets)
	}
	// IM12: a directory target is not a file and is not followed.
	put(t, root, "dirs/a.md", "@../chain\n")
	if r := resolveImports(root, "dirs/a.md"); len(r) != 1 || r[0].target != "chain" || r[0].exists || !r[0].isDir {
		t.Errorf("IM12: got %+v", r)
	}
	// IM13: targets outside the root are marked outside and never stat'ed.
	put(t, root, "out/a.md", "@~/.claude/mine.md @/etc/hosts @C:/x.md @../../up.md\n")
	var outside []string
	for _, r := range resolveImports(root, "out/a.md") {
		if !r.outside || r.exists {
			t.Errorf("IM13: %+v", r)
		}
		outside = append(outside, r.target)
	}
	if strings.Join(outside, " ") != "~/.claude/mine.md /etc/hosts C:/x.md ../up.md" {
		t.Errorf("IM13: got %q", outside)
	}
}

// A backslash import never leaves the copy: Windows would read `.claude\..\..\x` as a climb, so
// a token holding a backslash is outside on every host, and any target filepath.IsLocal refuses
// is outside too; instructions.md then carries a note, never the file's text.
func TestImportsStayInTheCopy(t *testing.T) {
	for _, tok := range []string{`.claude\..\..\outside.md`, `..\outside.md`, `a\b.md`} {
		if _, outside := targetOf("CLAUDE.md", tok); !outside {
			t.Errorf("%s: not outside", tok)
		}
	}
	for _, tok := range []string{"../outside.md", ".claude/../../outside.md", "/abs.md", "C:/x.md", "~/x.md"} {
		if _, outside := targetOf("CLAUDE.md", tok); !outside {
			t.Errorf("%s: not outside", tok)
		}
	}
	for _, tok := range []string{"x:stream.md", "CON", ".claude/NUL.md"} {
		target, outside := targetOf("CLAUDE.md", tok)
		if outside != !filepath.IsLocal(filepath.FromSlash(target)) {
			t.Errorf("%s: outside %v, IsLocal %v", tok, outside, filepath.IsLocal(filepath.FromSlash(target)))
		}
	}
	if target, outside := targetOf(".claude/a/b.md", "../c.md"); outside || target != ".claude/c.md" {
		t.Errorf("a local climb within the copy: %s %v", target, outside)
	}

	parent := t.TempDir()
	put(t, parent, "outside.md", "SECRET-OUTSIDE\n")
	c := &Copy{Dir: filepath.Join(parent, "copy")}
	put(t, c.Dir, "CLAUDE.md", "# T\n@"+`.claude\..\..\outside.md`+"\n@../outside.md\n")
	got := c.instructions()
	if strings.Contains(got, "SECRET-OUTSIDE") || strings.Count(got, "is outside the copy") != 2 {
		t.Errorf("instructions.md:\n%s", got)
	}
}

// extract refuses every name that is not a local path under CLAUDE.md or .claude, and writes
// nothing outside the copy whatever the name.
func TestExtractRefusesNamesOutsideTheCopy(t *testing.T) {
	names := []string{"../evil", ".claude/../../evil", "/abs", "C:/evil", `.claude/a\..\..\..\evil`,
		".claude/f:stream", ".claude/CON", "CON", "README.md"}
	for _, name := range names {
		parent := t.TempDir()
		c := &Copy{Dir: filepath.Join(parent, "copy")}
		if err := os.MkdirAll(c.Dir, 0o755); err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		tw := tar.NewWriter(&b)
		if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: 4}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("evil")); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		err := c.extract(b.Bytes())
		// A stream or a device name is a legal file name where the OS allows it (IsLocal
		// accepts it there); it is then written inside the copy and nowhere else.
		legal := strings.HasPrefix(name, ".claude/") && !strings.Contains(name, `\`) && !strings.Contains(name, "..") &&
			filepath.IsLocal(filepath.FromSlash(name))
		var we *Error
		if legal {
			if err != nil {
				t.Errorf("%q: legal on this host, refused: %v", name, err)
			}
		} else if !errors.As(err, &we) || we.Code != CodeCanonical {
			t.Errorf("%q: extracted, err %v", name, err)
		}
		entries, _ := os.ReadDir(parent)
		inner, _ := os.ReadDir(c.Dir)
		if len(entries) != 1 || (len(inner) != 0) != legal {
			t.Errorf("%q: wrote %v beside the copy, %v in it", name, entries, inner)
		}
	}
}
