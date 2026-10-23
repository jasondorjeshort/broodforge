// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The push scan (ADR 0025 rule 7): every commit of <base>..refs/sessions/<id> is read before the
// handoff pushes it, and a commit carrying personal data refuses the handoff. The classes, their
// patterns and pass rules, the trailer-block exemption and the lift are the base's
// .claude/hooks/privacy-scan.js (ADR 0018, contract section 2), ported, never run. Go's regexp has
// no lookaround, so each lookbehind or lookahead of the base's patterns is a check on the bytes
// beside the match (findAll). What it reads: author, committer, the message, and the lines each
// commit adds against every parent; not file names, tags, transcripts or key shapes. A hit is
// reported by class and location, never the value.

// The scan's classes: the base's home, email, identity and word (one class here: a name the
// identities do not hold, or a literal of the never set: the OS user name, the host name and
// privacy.never), host and session.
const (
	ClassHomePath    = "home-path"
	ClassEmail       = "email"
	ClassName        = "name"
	ClassHost        = "host"
	ClassSessionLink = "session-link"
)

// The locations of a hit that is not in an added line (`<path>:<line>`) or the message
// (`message:<line>`).
const (
	LocationAuthor    = "author"
	LocationCommitter = "committer"
)

// Privacy is what the scan reads as data: the never and allow lists, and the identities whose
// name and address may stand as author, committer and Co-Authored-By. For a session these are
// the instance's privacy section plus the driver, the participants and the models (ADR 0025 rule 7,
// WithSession).
type Privacy struct {
	// Machine is the OS user name and the host name, which the name class refuses before
	// Never as the base's scanSetup does; the caller reads them (config.MachineLiterals).
	Machine    []string
	Never      []string
	Allow      []string
	Identities []Identity
}

// WithSession is p with the session's identities added, the driver, the other participants and
// the models, whose names and addresses every session commit carries (attribution.go).
func (p Privacy) WithSession(a Attribution) Privacy {
	ids := append([]Identity{}, p.Identities...)
	ids = append(ids, a.Driver)
	for _, list := range [][]Contributor{a.Participants, a.Models} {
		for _, c := range list {
			ids = append(ids, c.Identity)
		}
	}
	p.Identities = ids
	return p
}

// WithCopy is p extended by the privacy section of the policy file in a session's workflow copy,
// <copy>/.claude/hooks/policy.json (ADR 0025 rule 7; the 0025 companion's push-scan row): its
// never and allow literals and its identities, each the base's {name, email} object or a
// `Name <email>` string, read as data. No copy, no file or no section is p unchanged; a file
// that does not parse refuses the scan as a failed read does, the fetch's code.
func (p Privacy) WithCopy(copyDir string) (Privacy, error) {
	b, err := os.ReadFile(filepath.Join(copyDir, filepath.FromSlash(policyPath)))
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	fail := func(why string) (Privacy, error) {
		return p, &Error{Code: CodeRefusedFetch, Message: "the workflow copy's " + policyPath + ": " + why}
	}
	if err != nil {
		return fail(err.Error())
	}
	var file struct {
		Privacy struct {
			Never      []string          `json:"never"`
			Allow      []string          `json:"allow"`
			Identities []json.RawMessage `json:"identities"`
		} `json:"privacy"`
	}
	if err := json.Unmarshal(bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf}), &file); err != nil {
		return fail(err.Error())
	}
	sec := file.Privacy
	p.Never = append(append([]string{}, p.Never...), sec.Never...)
	p.Allow = append(append([]string{}, p.Allow...), sec.Allow...)
	ids := append([]Identity{}, p.Identities...)
	for i, raw := range sec.Identities {
		var obj struct{ Name, Email string }
		var s string
		id := Identity{}
		switch {
		case json.Unmarshal(raw, &obj) == nil:
			id = Identity{Name: obj.Name, Email: obj.Email}
		case json.Unmarshal(raw, &s) == nil:
			if m := identityRE.FindStringSubmatch(s); m != nil {
				id = Identity{Name: m[1], Email: m[2]}
			}
		}
		if strings.TrimSpace(id.Name) == "" || !strings.Contains(id.Email, "@") {
			return fail("privacy.identities." + strconv.Itoa(i) + " is neither {name, email} nor Name <email>")
		}
		ids = append(ids, id)
	}
	p.Identities = ids
	return p, nil
}

// PrivacyHit is one hit: the commit, the class, and where (never the value).
type PrivacyHit struct {
	Commit   string
	Class    string
	Location string
}

// privacyListed bounds the hits the refusal's message names; the CLI lists all of them.
const privacyListed = 10

// policyPath is the base's policy file: a listed identity's address passes in its added lines,
// where the identities are declared.
const policyPath = ".claude/hooks/policy.json"

// ScanPrivacy reads every commit of <base>..refs/sessions/<id> in the canonical, oldest first,
// and returns its hits in commit order, each (commit, class, location) once.
func ScanPrivacy(ctx context.Context, s Session, base string, p Privacy) ([]PrivacyHit, error) {
	// A read that fails is not a hit: the fetch's code and git's message, never the venue's
	// personal-data text; the handoff stops all the same.
	out, err := run(ctx, CodeRefusedFetch, s.Canonical, nil, "-c", "core.quotePath=false", "log", "--reverse", "-p", "--cc",
		"-U0", "--no-color", "--no-show-signature", "--no-use-mailmap", "--no-ext-diff", "--no-textconv", "--no-renames",
		"--no-relative", "--src-prefix=a/", "--dst-prefix=b/", "--format="+logOpen+"%n%H%n%an%n%ae%n%cn%n%ce%n%B%n"+logClose,
		base+".."+SessionRef(s.ID), "--")
	if err != nil {
		return nil, err
	}
	sc := newPrivacyScanner(p)
	for _, c := range parseLog(out) {
		sc.commit(c)
	}
	return sc.hits, nil
}

// PrivacyCommits is how many commits the hits fall in.
func PrivacyCommits(hits []PrivacyHit) int {
	seen := map[string]bool{}
	for _, h := range hits {
		seen[h.Commit] = true
	}
	return len(seen)
}

// PrivacySummary is the CLI's refusal line: `<n> of <m> commits carry personal data`.
func PrivacySummary(hits []PrivacyHit, checked int) string {
	return strconv.Itoa(PrivacyCommits(hits)) + " of " + strconv.Itoa(checked) + " commits carry personal data"
}

// PrivacyRefusal is handoff-refused-privacy for the hits among checked commits. Its message is
// what the venue's handoff-refused line wraps (the ADR 0025 companion's push-scan row):
// `<n> commits carry personal data. <sha7>: <class> at <location>, ...; <sha7>: ...`, the
// first privacyListed hits named and the rest counted.
func PrivacyRefusal(hits []PrivacyHit, checked int) *Error {
	var b strings.Builder
	b.WriteString(strconv.Itoa(PrivacyCommits(hits)) + " commits carry personal data.")
	prev := ""
	for i, h := range hits {
		if i == privacyListed {
			b.WriteString("; and " + strconv.Itoa(len(hits)-i) + " more")
			break
		}
		switch {
		case i == 0:
			b.WriteString(" " + shortSHA(h.Commit) + ": ")
		case h.Commit != prev:
			b.WriteString("; " + shortSHA(h.Commit) + ": ")
		default:
			b.WriteString(", ")
		}
		b.WriteString(h.Class + " at " + h.Location)
		prev = h.Commit
	}
	return &Error{Code: CodeRefusedPrivacy, Message: b.String(), Hits: hits, Checked: checked}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// ---------- the classes ----------

const bq = "`"

var (
	emailRE             = regexp.MustCompile(`[\w.+-]+@[\w-]+(?:\.[\w-]+)+`)
	placeholderDomainRE = regexp.MustCompile(`^example\.(?:com|net|org)$|\.invalid$`)
	urlSchemeRE         = regexp.MustCompile(`(?i)(?:https?|ssh|git)://$`)
	homeDriveRE         = regexp.MustCompile(`[A-Za-z]:[\\/]+[Uu][Ss][Ee][Rr][Ss][\\/]+([A-Za-z0-9._-]+)`)
	homeRootRE          = regexp.MustCompile(`/(?:home|Users|(?:cygdrive/|mnt/)?[A-Za-z]/Users)/([A-Za-z0-9._-]+)`)
	homeWSLRE           = regexp.MustCompile(`(?i)[\\/]{2}wsl(?:\$|\.localhost)[\\/]+[^\\/\s"'` + bq + `]+[\\/]+home[\\/]+([A-Za-z0-9._-]+)`)
	homeSlugRE          = regexp.MustCompile(`(?:[A-Za-z]--[Uu][Ss][Ee][Rr][Ss]|-home|-Users)-([A-Za-z0-9._]+)-`)
	profileSlugRE       = regexp.MustCompile(`(?i)^(?:public|default-user|default|all-users|\.\.\.)-`)
	hostRE              = regexp.MustCompile(`(?i)-desktop|desktop-`)
	sessionKeyRE        = regexp.MustCompile(`(?i)^Claude-Session:`)
	sessionTrailerRE    = regexp.MustCompile(`(?i)^Claude-Session:[ \t]*https://claude\.ai/[^\s"'` + bq + `<>]+[ \t]*$`)
	trailerLineRE       = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9-]*:[ \t]|[ \t]+\S)`)
	urlRE               = regexp.MustCompile(`(?i)\bhttps?://(?:[^\s/@"'` + bq + `<>]*@)?([A-Za-z0-9.-]+)(?::\d+)?(/[^\s"'` + bq + `<>()\[\]{}|\\^]*)?`)
	identityRE          = regexp.MustCompile(`^\s*(.*?)\s*<([^<>]*)>\s*$`)
)

// The characters that may stand before a root spelling and before a slug.
const (
	rootBefore = `"'` + bq + `(=:[+/\,>`
	slugBefore = `"'` + bq + `(=:/\,>`
)

// profiles are the Windows system profiles a home path's segment may name.
var profiles = []string{"public", "default user", "default", "all users", "..."}

// urlUsers are the user parts of an address that is a URL's userinfo, not a person's.
var urlUsers = map[string]bool{"git": true, "x-access-token": true, "oauth2": true}

// findAll is re's matches in text, leftmost first and without overlap, keeping those ok accepts:
// the base's lookarounds. A refused match is tried again one character on, as a lookaround
// failing at that position would be. re must carry no anchor, since it runs on a suffix.
func findAll(re *regexp.Regexp, text string, ok func(start, end int) bool) [][]int {
	var out [][]int
	for at := 0; at <= len(text); {
		m := re.FindStringSubmatchIndex(text[at:])
		if m == nil {
			break
		}
		for i := range m {
			if m[i] >= 0 {
				m[i] += at
			}
		}
		if !ok(m[0], m[1]) {
			_, w := utf8.DecodeRuneInString(text[m[0]:])
			at = m[0] + max(w, 1)
			continue
		}
		out = append(out, m)
		at = max(m[1], m[0]+1)
	}
	return out
}

func isWordByte(b byte) bool {
	return b == '_' || ('0' <= b && b <= '9') || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}

// before reports whether position i of text is a line start or follows white space or one of set.
func before(text string, i int, set string) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(text[:i])
	return unicode.IsSpace(r) || r == 0xfeff || strings.ContainsRune(set, r)
}

func liftKey(s string) string { return strings.TrimRight(strings.ToLower(s), ".") }

func isProfile(rest string) bool {
	for _, p := range profiles {
		if len(rest) >= len(p) && strings.EqualFold(rest[:len(p)], p) {
			if len(rest) == len(p) {
				return true
			}
			if c := rest[len(p)]; !isWordByte(c) && c != '.' {
				return true
			}
		}
	}
	return false
}

type privacyScanner struct {
	allow map[string]bool
	ids   []Identity // names trimmed, emails trimmed and lower-cased
	words []*regexp.Regexp
	hits  []PrivacyHit
	seen  map[PrivacyHit]bool
}

func newPrivacyScanner(p Privacy) *privacyScanner {
	s := &privacyScanner{allow: map[string]bool{}, seen: map[PrivacyHit]bool{}}
	for _, a := range p.Allow {
		if k := liftKey(strings.TrimSpace(a)); k != "" {
			s.allow[k] = true
		}
	}
	for _, i := range p.Identities {
		s.ids = append(s.ids, Identity{Name: strings.TrimSpace(i.Name), Email: strings.ToLower(strings.TrimSpace(i.Email))})
	}
	// The never set as the base's scanSetup builds it: the OS user name, the host name, then
	// privacy.never; each literal once, any case; one shorter than minLiteral is skipped.
	said := map[string]bool{}
	for _, n := range append(append([]string{}, p.Machine...), p.Never...) {
		n = strings.TrimSpace(n)
		low := strings.ToLower(n)
		if n == "" || said[low] {
			continue
		}
		said[low] = true
		if utf8.RuneCountInString(n) < minLiteral {
			continue
		}
		s.words = append(s.words, regexp.MustCompile(`(?i)`+regexp.QuoteMeta(n)))
	}
	return s
}

// minLiteral is the shortest literal the name class reads, as in the base: a shorter one would
// match inside ordinary words. config check refuses a shorter privacy.never entry.
const minLiteral = 3

// add records a hit unless privacy.allow lifts its key (the match; for a home path the user
// segment, for a host the word without the marker) or it was recorded already.
func (s *privacyScanner) add(h PrivacyHit, key string) {
	if s.allow[liftKey(key)] || s.seen[h] {
		return
	}
	s.seen[h] = true
	s.hits = append(s.hits, h)
}

func (s *privacyScanner) listed(email string) bool {
	for _, i := range s.ids {
		if i.Email == email {
			return true
		}
	}
	return false
}

func (s *privacyScanner) identity(name, email string) bool {
	n, e := strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	for _, i := range s.ids {
		if i.Name == n && i.Email == e {
			return true
		}
	}
	return false
}

// emailOK is the email class's pass rule. own: an author or committer field, a Co-Authored-By
// line naming an identity, or an added line of the policy file, where an identity's own address
// passes; anywhere else only its GitHub noreply address does.
func (s *privacyScanner) emailOK(addr, before string, own bool) bool {
	e := strings.ToLower(addr)
	at := strings.IndexByte(e, '@')
	if s.listed(e) && (own || strings.HasSuffix(e, "@users.noreply.github.com")) {
		return true
	}
	return e == ModelEmail || strings.HasSuffix(e, "@noreply.anthropic.com") || e == "noreply@github.com" ||
		placeholderDomainRE.MatchString(e[at+1:]) || urlUsers[e[:at]] || urlSchemeRE.MatchString(before)
}

type address struct{ addr, before string }

// addresses is the addresses in a line. A \n, \r or \t escape in source text is not part of the
// user part after it.
func addresses(text string) []address {
	var out []address
	for _, m := range findAll(emailRE, text, func(st, _ int) bool {
		return st == 0 || !(isWordByte(text[st-1]) || strings.IndexByte(".+-", text[st-1]) >= 0)
	}) {
		addr, at := text[m[0]:m[1]], m[0]
		if at > 0 && text[at-1] == '\\' && strings.IndexByte("nrt", addr[0]) >= 0 {
			addr, at = addr[1:], at+1
		}
		if !strings.HasPrefix(addr, "@") {
			out = append(out, address{addr, text[:at]})
		}
	}
	return out
}

type lineOpts struct {
	trailer bool // a line of the message's trailer block
	own     bool // an identity's own address passes here
}

// line runs the classes over one line of text.
func (s *privacyScanner) line(text, loc, commit string, o lineOpts) {
	add := func(class, key string) { s.add(PrivacyHit{Commit: commit, Class: class, Location: loc}, key) }
	for _, a := range addresses(text) {
		if !s.emailOK(a.addr, a.before, o.own) {
			add(ClassEmail, a.addr)
		}
	}
	ends := map[int]bool{}
	for _, re := range []*regexp.Regexp{homeDriveRE, homeRootRE, homeWSLRE} {
		ms := re.FindAllStringSubmatchIndex(text, -1)
		if re == homeRootRE {
			ms = findAll(re, text, func(st, _ int) bool { return before(text, st, rootBefore) })
		}
		for _, m := range ms {
			if ends[m[1]] {
				continue // a drive spelling read again from its root part
			}
			ends[m[1]] = true
			if isProfile(text[m[2]:]) {
				continue
			}
			add(ClassHomePath, text[m[2]:m[3]])
		}
	}
	for _, m := range findAll(homeSlugRE, text, func(st, _ int) bool { return before(text, st, slugBefore) }) {
		if profileSlugRE.MatchString(text[m[2]:]) {
			continue
		}
		add(ClassHomePath, text[m[2]:m[3]])
	}
	starts := map[int]bool{}
	for _, m := range findAll(hostRE, text, func(st, end int) bool {
		if text[st] == '-' {
			return st > 0 && isWordByte(text[st-1])
		}
		return end < len(text) && isWordByte(text[end])
	}) {
		a, b := m[0], m[1]
		for a > 0 && (isWordByte(text[a-1]) || text[a-1] == '-') {
			a--
		}
		for b < len(text) && (isWordByte(text[b]) || text[b] == '-') {
			b++
		}
		if starts[a] {
			continue
		}
		starts[a] = true
		add(ClassHost, strings.Trim(text[a:m[0]]+text[m[1]:b], "-"))
	}
	for _, re := range s.words {
		for _, m := range findAll(re, text, func(st, end int) bool {
			return (st == 0 || !isWordByte(text[st-1])) && (end == len(text) || !isWordByte(text[end]))
		}) {
			add(ClassName, text[m[0]:m[1]])
		}
	}
	sessionOK := o.trailer && sessionTrailerRE.MatchString(text)
	if sessionOK {
		return
	}
	if sessionKeyRE.MatchString(text) {
		add(ClassSessionLink, strings.TrimSpace(text))
	}
	for _, m := range urlRE.FindAllStringSubmatchIndex(text, -1) {
		host := strings.TrimSuffix(strings.ToLower(text[m[2]:m[3]]), ".")
		path := ""
		if m[4] >= 0 {
			path = text[m[4]:m[5]]
		}
		if i := strings.IndexAny(path, "?#"); i >= 0 {
			path = path[:i]
		}
		if (host == "claude.ai" || strings.HasSuffix(host, ".claude.ai")) && strings.Trim(path, "/") != "" {
			add(ClassSessionLink, text[m[0]:m[1]])
		}
	}
}

// person is an author or committer: the email class on the name and the address (an identity's
// own address passes here), and the name class when the pair is not an identity. GitHub's
// web-flow committer passes as committer.
func (s *privacyScanner) person(name, email, loc, commit string, committer bool) {
	for _, t := range []string{name, email} {
		for _, a := range addresses(t) {
			if !s.emailOK(a.addr, a.before, true) {
				s.add(PrivacyHit{Commit: commit, Class: ClassEmail, Location: loc}, a.addr)
			}
		}
	}
	n, e := strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	if committer && n == "GitHub" && e == "noreply@github.com" {
		return
	}
	if !s.identity(name, email) {
		s.add(PrivacyHit{Commit: commit, Class: ClassName, Location: loc}, name+" <"+email+">")
	}
}

// coAuthor reports whether a trailer line is a Co-Authored-By naming an identity.
func (s *privacyScanner) coAuthor(line string) bool {
	k, v, ok := strings.Cut(line, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(k), TrailerKey) {
		return false
	}
	m := identityRE.FindStringSubmatch(v)
	return m != nil && s.identity(m[1], m[2])
}

func (s *privacyScanner) commit(c logCommit) {
	s.person(c.an, c.ae, LocationAuthor, c.sha, false)
	s.person(c.cn, c.ce, LocationCommitter, c.sha, true)
	trailers := trailerBlock(c.message)
	for i, l := range c.message {
		t := trailers[i]
		s.line(l, "message:"+strconv.Itoa(i+1), c.sha, lineOpts{trailer: t, own: t && s.coAuthor(l)})
	}
	for _, a := range c.lines {
		s.line(a.text, a.file+":"+strconv.Itoa(a.n), c.sha, lineOpts{own: a.file == policyPath})
	}
}

// trailerBlock is the indices of a message's trailer block: its last paragraph when that is not
// the first (the subject's) and every line of it is a trailer line.
func trailerBlock(msg []string) map[int]bool {
	start := len(msg)
	for start > 0 && strings.TrimSpace(msg[start-1]) != "" {
		start--
	}
	if start == len(msg) || start == 0 {
		return nil
	}
	first := 0
	for first < len(msg) && strings.TrimSpace(msg[first]) == "" {
		first++
	}
	if start <= first {
		return nil
	}
	for _, l := range msg[start:] {
		if !trailerLineRE.MatchString(l) {
			return nil
		}
	}
	if r, _ := utf8.DecodeRuneInString(msg[start]); unicode.IsSpace(r) {
		return nil
	}
	out := map[int]bool{}
	for i := start; i < len(msg); i++ {
		out[i] = true
	}
	return out
}

// ---------- the log ----------

// Each commit opens with a NUL-C line (no message holds a NUL) and its message closes with a
// NUL-E line; the patch follows.
const (
	logOpen  = "%x00C"
	logClose = "%x00E"
	openLine = "\x00C"
	endLine  = "\x00E"
)

type logCommit struct {
	sha, an, ae, cn, ce string
	message             []string
	lines               []addedLine
}

type addedLine struct {
	file string
	n    int // the line number in the new file
	text string
}

var hunkRE = regexp.MustCompile(`^(@@+) (?:-\S+ )+\+(\d+)(?:,\d+)? @@+`)

// parseLog reads the one `git log -p --cc -U0` over the range. Patch header lines are read
// only between a diff line and its first hunk, so content that begins "++ " is content. In a
// merge's combined diff a line counts when every parent column is "+": the lines the merge adds
// against every parent.
func parseLog(out string) []logCommit {
	var commits []logCommit
	ls := strings.Split(out, "\n")
	cur := -1
	file, haveFile, header := "", false, false
	cols, n := 1, 0
	get := func(i int) string {
		if i < len(ls) {
			return strings.TrimSuffix(ls[i], "\r")
		}
		return ""
	}
	for k := 0; k < len(ls); k++ {
		l := get(k)
		if l == openLine {
			c := logCommit{sha: strings.TrimSpace(get(k + 1)), an: get(k + 2), ae: get(k + 3), cn: get(k + 4), ce: get(k + 5)}
			e := k + 6
			for e < len(ls) && get(e) != endLine {
				e++
			}
			for i := k + 6; i < e; i++ {
				c.message = append(c.message, get(i))
			}
			for len(c.message) > 0 && c.message[len(c.message)-1] == "" {
				c.message = c.message[:len(c.message)-1]
			}
			commits = append(commits, c)
			cur = len(commits) - 1
			haveFile, header = false, false
			k = e
			continue
		}
		if cur < 0 {
			continue
		}
		if strings.HasPrefix(l, "diff --git ") || strings.HasPrefix(l, "diff --cc ") || strings.HasPrefix(l, "diff --combined ") {
			file, haveFile, header = diffPath(l), true, true
			continue
		}
		if h := hunkRE.FindStringSubmatch(l); h != nil {
			header = false
			cols = len(h[1]) - 1
			n, _ = strconv.Atoi(h[2])
			continue
		}
		if !haveFile {
			continue
		}
		if header {
			if strings.HasPrefix(l, "+++ ") && l != "+++ /dev/null" {
				file = strings.TrimPrefix(unquotePath(strings.TrimSuffix(l[4:], "\t")), "b/")
			}
			continue
		}
		if len(l) < cols {
			continue
		}
		pre := l[:cols]
		if strings.Contains(pre, "-") || strings.HasPrefix(pre, `\`) {
			continue
		}
		if strings.Trim(pre, "+") == "" {
			commits[cur].lines = append(commits[cur].lines, addedLine{file: file, n: n, text: l[cols:]})
		}
		n++
	}
	return commits
}

// unquotePath is a path as git C-quotes it in a diff header when it holds a quote, a backslash
// or a control character.
func unquotePath(p string) string {
	if len(p) < 2 || p[0] != '"' {
		return p
	}
	inner := p[1 : len(p)-1]
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) {
			i++
			switch inner[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(inner[i])
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

var quotedDiffRE = regexp.MustCompile(`^"a/((?:[^"\\]|\\.)*)" "b/((?:[^"\\]|\\.)*)"$`)

// diffPath is the path of a `diff --git a/P b/P` (no renames, so both sides are P) or
// `diff --cc P` line.
func diffPath(l string) string {
	rest, ok := strings.CutPrefix(l, "diff --git ")
	if !ok {
		rest = strings.TrimPrefix(strings.TrimPrefix(l, "diff --cc "), "diff --combined ")
		return unquotePath(rest)
	}
	if q := quotedDiffRE.FindStringSubmatch(rest); q != nil {
		return unquotePath(`"` + q[2] + `"`)
	}
	if len(rest) < 5 {
		return rest
	}
	return rest[len(rest)-(len(rest)-5)/2:]
}
