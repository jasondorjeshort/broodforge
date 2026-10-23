// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"reflect"
	"strings"
	"testing"
)

// Fixture values use the scan's placeholders (user, alice, bob). An address that must be a hit
// is pieced together in source, so the repository's own scan of this file does not read one.
var (
	bobMail = "bob" + "@" + "mail.net"
	bob     = Identity{Name: "Bob", Email: bobMail}
)

// The never set: the OS user name and host name beside privacy.never, each a whole word in any
// case, privacy.allow lifting them, a literal under three characters skipped. The literals are
// seeded, never the machine's own.
func TestPrivacyMachineLiterals(t *testing.T) {
	p := Privacy{Machine: []string{"svcacct", "hostbox", "ab"}, Never: []string{"xy", "zyzzyva"}}
	cases := map[string][]string{
		"run by svcacct on HostBox": {ClassName},
		"svcaccts and hostboxes":    nil,
		"ab xy cd":                  nil,
		"zyzzyva":                   {ClassName},
	}
	for text, want := range cases {
		if got := lineClasses(p, text, lineOpts{}); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %q, want %q", text, got, want)
		}
	}
	s := newPrivacyScanner(p)
	s.line("run by svcacct", "message:1", "c", lineOpts{})
	s.line("on hostbox", "a.txt:2", "c", lineOpts{})
	if want := []PrivacyHit{{"c", ClassName, "message:1"}, {"c", ClassName, "a.txt:2"}}; !reflect.DeepEqual(s.hits, want) {
		t.Errorf("hits %+v, want %+v", s.hits, want)
	}
	lifted := Privacy{Machine: []string{"svcacct"}, Allow: []string{"SVCACCT"}}
	if got := lineClasses(lifted, "run by svcacct", lineOpts{}); got != nil {
		t.Errorf("allow did not lift the OS user name: %q", got)
	}
}

func lineClasses(p Privacy, text string, o lineOpts) []string {
	s := newPrivacyScanner(p)
	s.line(text, "f:1", "c", o)
	var out []string
	for _, h := range s.hits {
		out = append(out, h.Class)
	}
	return out
}

// The classes the base's privacy-scan.js defines, as ported: each line's classes in order.
func TestPrivacyClasses(t *testing.T) {
	never := Privacy{Never: []string{"Zyzzyva"}}
	cases := []struct {
		text string
		o    lineOpts
		want []string
	}{
		{"write to " + bobMail, lineOpts{}, []string{ClassEmail}},
		{"alice@example.org and bob@example.invalid", lineOpts{}, nil},
		{"Co-Authored-By: Claude <noreply@anthropic.com>", lineOpts{}, nil},
		{"git clone https://alice@" + "mail.net/r.git", lineOpts{}, nil},
		{`x = "\n` + "bob" + "@" + `mail.net"`, lineOpts{}, []string{ClassEmail}},
		{`C:\Users\alice\notes`, lineOpts{}, []string{ClassHomePath}},
		{"see /home/bob/x", lineOpts{}, []string{ClassHomePath}},
		{"x/home/bob", lineOpts{}, nil},
		{"cd /c/Users/alice/src", lineOpts{}, []string{ClassHomePath}},
		{`\\wsl$\Ubuntu\home\alice`, lineOpts{}, []string{ClassHomePath}},
		{"C--Users-alice-src", lineOpts{}, []string{ClassHomePath}},
		{`C:\Users\Public\x and C:\Users\Default User\y`, lineOpts{}, nil},
		{"/home/<user>/x and C:/Users/%USERNAME%", lineOpts{}, nil},
		{"on alice-desktop", lineOpts{}, []string{ClassHost}},
		{"DESKTOP-ALICE", lineOpts{}, []string{ClassHost}},
		{"the desktop - app", lineOpts{}, nil},
		{"see https://claude" + ".ai/code/x", lineOpts{}, []string{ClassSessionLink}},
		{"https://claude" + ".ai/", lineOpts{}, nil},
		{"Claude-Session: https://claude" + ".ai/code/x", lineOpts{trailer: true}, nil},
		{"Claude-Session: https://claude" + ".ai/code/x", lineOpts{}, []string{ClassSessionLink}},
		{"Claude-Session: https://claude" + ".ai/code/x more", lineOpts{trailer: true}, []string{ClassSessionLink}},
	}
	for _, c := range cases {
		if got := lineClasses(never, c.text, c.o); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q %+v: %q, want %q", c.text, c.o, got, c.want)
		}
	}
	for text, want := range map[string][]string{"zyzzyva said": {ClassName}, "ZYZZYVA": {ClassName}, "zyzzyvas": nil} {
		if got := lineClasses(never, text, lineOpts{}); !reflect.DeepEqual(got, want) {
			t.Errorf("never %q: %q, want %q", text, got, want)
		}
	}
}

// privacy.allow lifts a hit whose key it equals: a home path's user segment, a host's word, an
// address as a whole.
func TestPrivacyAllow(t *testing.T) {
	cases := []struct {
		allow, text string
		lifted      bool
	}{
		{"alice", `C:\Users\alice\notes`, true},
		{"ALICE.", "see /home/alice/x", true},
		{"alice", "on alice-desktop", true},
		{"bob", "write to " + bobMail, false},
		{bobMail, "write to " + bobMail, true},
		{"", `C:\Users\alice\notes`, false},
	}
	for _, c := range cases {
		got := lineClasses(Privacy{Allow: []string{c.allow}}, c.text, lineOpts{})
		if (len(got) == 0) != c.lifted {
			t.Errorf("allow %q on %q: %q, lifted want %v", c.allow, c.text, got, c.lifted)
		}
	}
}

// An identity's own address passes as author and committer and in a Co-Authored-By trailer
// naming it, never in other text; a pair the identities do not hold is a name hit.
func TestPrivacyIdentities(t *testing.T) {
	s := newPrivacyScanner(Privacy{Identities: []Identity{bob}})
	s.person("Bob", bobMail, LocationAuthor, "c", false)
	s.person("Bob", strings.ToUpper(bobMail), LocationCommitter, "c", true)
	trailer := "Co-Authored-By: Bob <" + bobMail + ">"
	if !s.coAuthor(trailer) || s.coAuthor("Co-Authored-By: Robert <"+bobMail+">") {
		t.Error("coAuthor")
	}
	s.commit(logCommit{sha: "c1", an: "Bob", ae: bobMail, cn: "Bob", ce: bobMail,
		message: []string{"one", "", trailer}, lines: []addedLine{{file: "a.txt", n: 2, text: "ask " + bobMail}}})
	s.person("Alice", "alice@example.com", LocationAuthor, "c2", false)
	s.person("GitHub", "noreply@github.com", LocationCommitter, "c2", true)
	want := []PrivacyHit{{"c1", ClassEmail, "a.txt:2"}, {"c2", ClassName, LocationAuthor}}
	if !reflect.DeepEqual(s.hits, want) {
		t.Errorf("hits %+v, want %+v", s.hits, want)
	}
	// The instance's identities stay; the session's follow.
	drv := Identity{Name: "Test Driver", Email: "driver@example.com"}
	got := Privacy{Identities: []Identity{bob}}.WithSession(Attribution{Driver: drv,
		Models: []Contributor{{Identity: Identity{Name: "Claude Opus 5.5", Email: ModelEmail}}}})
	if len(got.Identities) != 3 || got.Identities[0] != bob || got.Identities[1] != drv {
		t.Errorf("WithSession %+v", got.Identities)
	}
	// Outside the trailer block the same line is text.
	if got := lineClasses(Privacy{Identities: []Identity{bob}}, trailer, lineOpts{}); !reflect.DeepEqual(got, []string{ClassEmail}) {
		t.Errorf("trailer line as text: %q", got)
	}
}

func TestTrailerBlock(t *testing.T) {
	cases := []struct {
		msg  []string
		want map[int]bool
	}{
		{[]string{"one", "", "Co-Authored-By: x", "Claude-Session: y"}, map[int]bool{2: true, 3: true}},
		{[]string{"Co-Authored-By: x"}, nil},
		{[]string{"one", "", "not a trailer"}, nil},
		{[]string{"one", "", "  Key: v"}, nil},
	}
	for _, c := range cases {
		if got := trailerBlock(c.msg); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: %v, want %v", c.msg, got, c.want)
		}
	}
}

// The log reader: header lines are read only before the first hunk, so an added line that
// begins "++ " is content; a quoted path is unquoted; a merge counts lines every parent lacks.
func TestParseLog(t *testing.T) {
	log := "\x00C\nabc\nA\na@x\nC\nc@x\nsubject\n\nbody\n\n\x00E\n\n" +
		"diff --git a/a.txt b/a.txt\nnew file mode 100644\n--- /dev/null\n+++ b/a.txt\n@@ -0,0 +1,2 @@\n+one\n+++ two\n" +
		"diff --git \"a/q\\\"x\" \"b/q\\\"x\"\n--- a/q\"x\n+++ b/q\"x\n@@ -3 +3 @@\n-old\n+new\n" +
		"\x00C\ndef\nA\na@x\nC\nc@x\nmerge\n\x00E\n\n" +
		"diff --cc m.txt\n--- a/m.txt\n+++ b/m.txt\n@@@ -1,1 -1,1 +1,3 @@@\n  same\n+ ours\n++both\n"
	got := parseLog(log)
	want := []logCommit{
		{sha: "abc", an: "A", ae: "a@x", cn: "C", ce: "c@x", message: []string{"subject", "", "body"},
			lines: []addedLine{{"a.txt", 1, "one"}, {"a.txt", 2, "++ two"}, {`q"x`, 3, "new"}}},
		{sha: "def", an: "A", ae: "a@x", cn: "C", ce: "c@x", message: []string{"merge"},
			lines: []addedLine{{"m.txt", 3, "both"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseLog:\n got %+v\nwant %+v", got, want)
	}
}

// The refusal's message is the venue's text: n commits, each hit by short sha, class and
// location, at most privacyListed of them.
func TestPrivacyRefusalMessage(t *testing.T) {
	a, b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	e := PrivacyRefusal([]PrivacyHit{{a, ClassEmail, "message:1"}, {a, ClassHomePath, "n.txt:2"}, {b, ClassHost, "author"}}, 4)
	if want := "2 commits carry personal data. aaaaaaa: email at message:1, home-path at n.txt:2; bbbbbbb: host at author"; e.Message != want {
		t.Errorf("message %q, want %q", e.Message, want)
	}
	if e.Code != CodeRefusedPrivacy || PrivacySummary(e.Hits, e.Checked) != "2 of 4 commits carry personal data" {
		t.Errorf("%+v", e)
	}
	var many []PrivacyHit
	for i := range privacyListed + 2 {
		many = append(many, PrivacyHit{a, ClassEmail, "message:" + string(rune('1'+i))})
	}
	if m := PrivacyRefusal(many, 1).Message; !strings.HasSuffix(m, "; and 2 more") {
		t.Errorf("long message %q", m)
	}
}
