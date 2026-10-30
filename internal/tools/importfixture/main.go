// SPDX-License-Identifier: GPL-3.0-or-later

// Command importfixture copies one recorded fixture directory into fixtures/engine/<name>/,
// rewriting every PC home and spike-root path to a placeholder (docs/design/08-implementation-plan.md, P1).
//
//	go run ./internal/tools/importfixture [--force] [--user <pc user>] <src> <dst>
//	go run ./internal/tools/importfixture --events <file> [--user <pc user>]
//
// <src> is a spike fixture, C:/Users/<pc user>/AppData/Local/Temp/bf-spike/fx/<name>, or a
// directory engine record wrote. The PC user is --user, else read from the source path's
// C:/Users/<user> prefix. Only turn-<n>.jsonl and fixture.json are written: fixture.json is built
// from the spike's meta-<n>.json when every turn has one, else it is the source's own fixture.json,
// rewritten like the turns. The rewrite is textual: every byte outside a rewritten path token or
// project slug is kept, so the fake replays the real stream.
//
// --events rewrites one captured `session tail --json --all` file in place (a golden session's
// events.jsonl, 05-debug-interface-discord.md, Golden sessions): the same path and slug rewrite,
// the PC user being --user, else the one the file's first PC home path names (a file with none
// gets no home rewrite), and every whole-token occurrence of the captured session id, read from
// the session-started line, rewritten to s0001.
// Exit codes: 0 done, 1 failure with one line on stderr, 2 usage.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// claims names the spike claim each fixture records (the findings for ADRs 0020 and 0022).
var claims = map[string]string{
	"basic":       "S1",
	"defer":       "C27",
	"interrupted": "S2",
	"parallel":    "C28",
	"hook-exit2":  "C39",
	"hook-exit1":  "C39",
	"hook-sleep":  "C39",
	"scrub":       "C34",
	"settings-a":  "C26",
	"settings-b":  "C26",
}

const (
	homePlaceholder  = "/home/user"
	spikePlaceholder = "/home/user/bf-spike"
)

// spikeUnderHome is the spike root's path below the PC home, one segment per entry.
var spikeUnderHome = []string{"AppData", "Local", "Temp", "bf-spike"}

var srcPattern = regexp.MustCompile(`(?i)^[a-z]:/Users/([^/]+)/AppData/Local/Temp/bf-spike/fx/([^/]+)$`)

// homePattern reads the PC user from any source under the PC home.
var homePattern = regexp.MustCompile(`(?i)^[a-z]:/Users/([^/]+)/`)

var turnPattern = regexp.MustCompile(`^turn-([0-9]+)\.jsonl$`)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("importfixture", flag.ContinueOnError)
	force := fs.Bool("force", false, "write into a non-empty <dst>, replacing its turn files and fixture.json")
	user := fs.String("user", "", "the PC user whose home the source's paths name (default: read from <src>)")
	events := fs.String("events", "", "rewrite this captured events file in place instead of importing a fixture directory")
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: importfixture [--force] [--user <pc user>] <src> <dst>")
		fmt.Fprintln(os.Stderr, "       importfixture --events <file> [--user <pc user>]")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *events != "" {
		if fs.NArg() != 0 || *force {
			fs.Usage()
			return 2
		}
		if err := rewriteEvents(*events, *user); err != nil {
			fmt.Fprintln(os.Stderr, "error: "+err.Error())
			return 1
		}
		return 0
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return 2
	}
	if err := importDir(fs.Arg(0), fs.Arg(1), *user, *force); err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		return 1
	}
	return 0
}

// fixture is fixture.json, read by the fake engine (P3).
type fixture struct {
	Claim         string `json:"claim"`
	EngineVersion string `json:"engineVersion"`
	Recorded      string `json:"recorded"`
	Turns         []turn `json:"turns"`
}

type turn struct {
	N               int    `json:"n"`
	Exit            *int   `json:"exit"`
	EngineSessionID string `json:"engineSessionId"`
}

// meta is the subset of the spike's meta-<n>.json that fixture.json takes.
type meta struct {
	EngineVersion   string `json:"engineVersion"`
	StartedAt       string `json:"startedAt"`
	EngineSessionID string `json:"engineSessionId"`
	Exit            struct {
		Code *int `json:"code"`
	} `json:"exit"`
}

func importDir(src, dst, user string, force bool) error {
	abs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	if user == "" {
		user, err = userFromPath(filepath.ToSlash(abs))
		if err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	var ns []int
	for _, e := range entries {
		if t := turnPattern.FindStringSubmatch(e.Name()); t != nil && e.Type().IsRegular() {
			n, _ := strconv.Atoi(t[1])
			ns = append(ns, n)
		}
	}
	if len(ns) == 0 {
		return fmt.Errorf("no turn-<n>.jsonl in %s", src)
	}
	sort.Ints(ns)

	if err := checkDst(abs, dst); err != nil {
		return err
	}

	// Everything is rewritten and scanned in memory first, so a refusal leaves dst untouched.
	r := newRewriter(user)
	outs := map[string][]byte{}
	for _, n := range ns {
		raw, err := os.ReadFile(filepath.Join(src, fmt.Sprintf("turn-%d.jsonl", n)))
		if err != nil {
			return err
		}
		outs[fmt.Sprintf("turn-%d.jsonl", n)] = r.rewrite(raw)
	}
	metas := 0
	for _, n := range ns {
		if _, err := os.Stat(filepath.Join(src, fmt.Sprintf("meta-%d.json", n))); err == nil {
			metas++
		}
	}
	switch metas {
	case len(ns):
		if _, err := os.Stat(filepath.Join(src, "fixture.json")); err == nil {
			return fmt.Errorf("%s has both meta-<n>.json and fixture.json; remove the one that is not the record", src)
		}
		out, err := fixtureFromMetas(src, filepath.Base(abs), ns)
		if err != nil {
			return err
		}
		outs["fixture.json"] = out
	case 0:
		out, err := recordedFixture(src, ns)
		if err != nil {
			return err
		}
		outs["fixture.json"] = r.rewrite(out)
	default:
		return fmt.Errorf("%s has meta-<n>.json for %d of %d turns", src, metas, len(ns))
	}
	if err := scanLeak(user, outs); err != nil {
		return err
	}

	if err := prepareDst(dst, force); err != nil {
		return err
	}
	for _, n := range ns {
		f := fmt.Sprintf("turn-%d.jsonl", n)
		if err := os.WriteFile(filepath.Join(dst, f), outs[f], 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dst, "fixture.json"), outs["fixture.json"], 0o644)
}

// userFromPath reads the PC user from a spike fixture path or any path under C:/Users/<user>.
func userFromPath(abs string) (string, error) {
	if m := srcPattern.FindStringSubmatch(abs); m != nil {
		return m[1], nil
	}
	if m := homePattern.FindStringSubmatch(abs); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("source %s is not under <drive>:/Users/<user>; pass --user", abs)
}

// fixtureFromMetas builds fixture.json from the spike's meta-<n>.json files; the claim comes
// from the spike fixture's name.
func fixtureFromMetas(src, name string, ns []int) ([]byte, error) {
	claim, ok := claims[name]
	if !ok {
		return nil, fmt.Errorf("no claim known for fixture %q", name)
	}
	fx := fixture{Claim: claim}
	for _, n := range ns {
		mb, err := os.ReadFile(filepath.Join(src, fmt.Sprintf("meta-%d.json", n)))
		if err != nil {
			return nil, err
		}
		var md meta
		if err := json.Unmarshal(mb, &md); err != nil {
			return nil, fmt.Errorf("meta-%d.json: %w", n, err)
		}
		if fx.EngineVersion == "" {
			fx.EngineVersion = md.EngineVersion
			if len(md.StartedAt) >= len("2006-01-02") {
				fx.Recorded = md.StartedAt[:len("2006-01-02")]
			}
		} else if md.EngineVersion != fx.EngineVersion {
			return nil, fmt.Errorf("meta-%d.json: engine %s, turn %d had %s", n, md.EngineVersion, ns[0], fx.EngineVersion)
		}
		fx.Turns = append(fx.Turns, turn{N: n, Exit: md.Exit.Code, EngineSessionID: md.EngineSessionID})
	}
	out, err := json.MarshalIndent(fx, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// recordedFixture returns the source's own fixture.json, as engine record wrote it, after
// checking it names a claim and an engine version and lists exactly the turn files found.
func recordedFixture(src string, ns []int) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(src, "fixture.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s has neither meta-<n>.json nor fixture.json", src)
	}
	if err != nil {
		return nil, err
	}
	var fx fixture
	if err := json.Unmarshal(b, &fx); err != nil {
		return nil, fmt.Errorf("fixture.json: %w", err)
	}
	if fx.Claim == "" || fx.EngineVersion == "" {
		return nil, fmt.Errorf("fixture.json: claim or engineVersion missing")
	}
	if len(fx.Turns) != len(ns) {
		return nil, fmt.Errorf("fixture.json lists %d turns, %s has %d turn files", len(fx.Turns), src, len(ns))
	}
	for k, t := range fx.Turns {
		if t.N != ns[k] {
			return nil, fmt.Errorf("fixture.json turn %d does not match turn-%d.jsonl", t.N, ns[k])
		}
	}
	return b, nil
}

// checkDst refuses a destination that is the source or lies under the spike root (the bf-spike
// ancestor of src; src itself when it is not under one), where --force would delete the only
// copy of the recordings.
func checkDst(absSrc, dst string) error {
	d, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	d = strings.ToLower(filepath.ToSlash(d))
	s := strings.ToLower(filepath.ToSlash(absSrc))
	root := s
	if i := strings.Index(s, "/bf-spike/"); i >= 0 {
		root = s[:i+len("/bf-spike")]
	}
	if d == s || d == root || strings.HasPrefix(d, root+"/") {
		return fmt.Errorf("destination %s is the source or under the spike root %s", dst, root)
	}
	return nil
}

// leakPatterns are what no rewritten output may hold whoever the user is: a Users path token in
// any separator form the rewrite knows (/, \, and the 2- and 4-backslash JSON escapes), and the
// engine's home slug, either of which names some account's home.
var leakPatterns = []struct {
	what string
	re   *regexp.Regexp
}{
	{"a Users path", regexp.MustCompile(`(?i)(^|[^a-z0-9_])users(/|\\{1,4})[^/\\\s"'` + "`" + `]`)},
	{"a C--Users- slug", regexp.MustCompile(`(?i)[a-z]--users-`)},
}

// scanLeak refuses output that still contains the PC user name in any form, case-insensitively,
// or any user's home path or slug: the rewrite covers known path forms of one user only.
func scanLeak(user string, outs map[string][]byte) error {
	needle := []byte(strings.ToLower(user)) // none when the user is unknown (an events file with no home path)
	names := make([]string, 0, len(outs))
	for name := range outs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b := outs[name]
		low := bytes.ToLower(b)
		if k := bytes.Index(low, needle); len(needle) > 0 && k >= 0 {
			line := bytes.Count(low[:k], []byte("\n")) + 1
			return fmt.Errorf("the PC user name survives the rewrite in %s line %d; extend the rewrite", name, line)
		}
		for _, p := range leakPatterns {
			if loc := p.re.FindIndex(b); loc != nil {
				line := bytes.Count(b[:loc[0]], []byte("\n")) + 1
				return fmt.Errorf("%s survives the rewrite in %s line %d; check --user or extend the rewrite", p.what, name, line)
			}
		}
	}
	return nil
}

// prepareDst creates dst, refusing a non-empty one unless force, which removes only the files
// this tool writes so no stale turn survives.
func prepareDst(dst string, force bool) error {
	entries, err := os.ReadDir(dst)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(dst, 0o755)
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("destination %s is not empty (pass --force to replace it)", dst)
	}
	for _, e := range entries {
		if turnPattern.MatchString(e.Name()) || e.Name() == "fixture.json" {
			if err := os.Remove(filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// rewriter replaces PC paths and the engine's project slugs for one PC user.
type rewriter struct {
	user  string
	slugs [][2][]byte // longest first
}

func newRewriter(user string) *rewriter {
	home := "C--Users-" + user
	return &rewriter{user: user, slugs: [][2][]byte{
		{[]byte(home + "-" + strings.Join(spikeUnderHome, "-")), []byte(slug(spikePlaceholder))},
		{[]byte(home), []byte(slug(homePlaceholder))},
	}}
}

// slug is the engine's project directory name for a path: every separator becomes "-".
func slug(p string) string { return strings.ReplaceAll(p, "/", "-") }

func (r *rewriter) rewrite(in []byte) []byte {
	return r.rewriteSlugs(r.rewritePaths(in))
}

// rewritePaths replaces every path token that starts with the PC home in any of its forms:
// C:/Users/<user>, C:\Users\<user>, the JSON-escaped forms with 2 or 4 backslashes, and the MSYS
// form /c/Users/<user> (forward slashes only, not after a word byte). For the drive forms the
// separator form is fixed by the first separator after the drive; inside the token that form or
// "/" separates, and a backslash run of any other length (a JSON escape such as \n) ends it.
func (r *rewriter) rewritePaths(in []byte) []byte {
	var out bytes.Buffer
	i := 0
	for i < len(in) {
		if repl, end, ok := r.matchPath(in, i); ok {
			out.WriteString(repl)
			i = end
			continue
		}
		out.WriteByte(in[i])
		i++
	}
	return out.Bytes()
}

// matchPath reports whether a home path token starts at i, its replacement, and where it ends.
// Besides the drive forms it knows the MSYS form "/c/Users/<user>", which has "/" separators only.
func (r *rewriter) matchPath(in []byte, i int) (string, int, bool) {
	if i > 0 && isWordByte(in[i-1]) {
		return "", 0, false
	}
	if i+3 < len(in) && in[i] == '/' && isLetter(in[i+1]) && in[i+2] == '/' {
		return r.matchSegments(in, i+2, 0)
	}
	if i+2 >= len(in) || !isLetter(in[i]) || in[i+1] != ':' {
		return "", 0, false
	}
	j := i + 2
	sepLen := 0 // 0: "/"; n: a run of n backslashes
	if in[j] == '\\' {
		for j+sepLen < len(in) && in[j+sepLen] == '\\' {
			sepLen++
		}
	} else if in[j] != '/' {
		return "", 0, false
	}
	return r.matchSegments(in, j, sepLen)
}

// matchSegments reads the path segments from the first separator at j and, when they name the
// PC user's home, returns the replacement and the token's end.
func (r *rewriter) matchSegments(in []byte, j, sepLen int) (string, int, bool) {
	// Split the token into segments; trailing is a separator with no segment after it.
	var segs []string
	var trailing bool
	for {
		n := sepAt(in, j, sepLen)
		if n == 0 {
			break
		}
		j += n
		k := j
		for k < len(in) && isSegmentByte(in[k]) {
			k++
		}
		if k == j {
			trailing = true
			break
		}
		segs = append(segs, string(in[j:k]))
		j = k
	}
	if len(segs) < 2 || !strings.EqualFold(segs[0], "Users") || !strings.EqualFold(segs[1], r.user) {
		return "", 0, false
	}
	root, rest := homePlaceholder, segs[2:]
	if hasPrefixFold(rest, spikeUnderHome) {
		root, rest = spikePlaceholder, rest[len(spikeUnderHome):]
	}
	repl := root
	for _, s := range rest {
		repl += "/" + s
	}
	if trailing {
		repl += "/"
	}
	return repl, j, true
}

// sepAt returns the length of a separator at j in the token's form, or 0.
func sepAt(in []byte, j, sepLen int) int {
	if j >= len(in) {
		return 0
	}
	if in[j] == '/' {
		return 1
	}
	if sepLen == 0 || in[j] != '\\' {
		return 0
	}
	n := 0
	for j+n < len(in) && in[j+n] == '\\' {
		n++
	}
	if n != sepLen {
		return 0
	}
	return n
}

func hasPrefixFold(segs, prefix []string) bool {
	if len(segs) < len(prefix) {
		return false
	}
	for k, p := range prefix {
		if !strings.EqualFold(segs[k], p) {
			return false
		}
	}
	return true
}

// rewriteSlugs replaces the engine's project slugs, spike root first. A slug must not be
// preceded or followed by a letter or digit, so a longer user name is left alone.
func (r *rewriter) rewriteSlugs(in []byte) []byte {
	for _, s := range r.slugs {
		var out bytes.Buffer
		i := 0
		for {
			k := bytes.Index(in[i:], s[0])
			if k < 0 {
				out.Write(in[i:])
				break
			}
			at, end := i+k, i+k+len(s[0])
			if (at > 0 && isAlnum(in[at-1])) || (end < len(in) && isAlnum(in[end])) {
				out.Write(in[i : at+1])
				i = at + 1
				continue
			}
			out.Write(in[i:at])
			out.Write(s[1])
			i = end
		}
		in = out.Bytes()
	}
	return in
}

func isLetter(b byte) bool { return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') }

func isAlnum(b byte) bool { return isLetter(b) || (b >= '0' && b <= '9') }

func isWordByte(b byte) bool { return isAlnum(b) || b == '_' }

// isSegmentByte is true for a byte that may sit inside one path segment. Quotes, backticks,
// whitespace, separators and the punctuation that closes a path in prose or a command line end it.
func isSegmentByte(b byte) bool {
	if b < 0x20 || b == 0x7f {
		return false
	}
	switch b {
	case ' ', '"', '\'', '`', '/', '\\', ',', ';', ':', '*', '?', '<', '>', '|', '(', ')', '[', ']', '{', '}':
		return false
	}
	return true
}

// goldenSession is the id every captured events file is rewritten to.
const goldenSession = "s0001"

// startedSummary reads the session id from a session-started summary: `Session <id> on ...`.
var startedSummary = regexp.MustCompile(`^Session (\S+) on `)

// homeInText finds a PC home path in any form the rewrite knows: a drive with / or a run of
// backslashes, or the MSYS /<drive>/Users/; the user is the segment after Users.
var homeInText = regexp.MustCompile(`(?i)(?:^|[^a-z0-9_])(?:[a-z]:(?:/|\\{1,4})|/[a-z]/)Users(?:/|\\{1,4})([^/\\\s"'` + "`" + `,;:*?<>|()\[\]{}]+)`)

// rewriteEvents rewrites one captured events file in place: the PC paths and slugs, then every
// whole-token occurrence of the captured session id to s0001. Everything is rewritten and
// scanned in memory first, so a refusal leaves the file untouched.
func rewriteEvents(path, user string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	id, err := capturedSession(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if user == "" {
		if m := homeInText.FindSubmatch(raw); m != nil {
			user = string(m[1])
		}
	}
	out := raw
	if user != "" {
		out = newRewriter(user).rewrite(out)
	}
	if id != goldenSession {
		out = rewriteToken(out, []byte(id), []byte(goldenSession))
	}
	if err := scanLeak(user, map[string][]byte{filepath.Base(path): out}); err != nil {
		return err
	}
	// A temporary file beside it, then a rename: a failure mid-write leaves the capture whole.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// capturedSession is the session id of the file's session-started line: its `session` key when
// it has one (a replay file's), else the id its summary names (`session tail --json` has no key).
func capturedSession(raw []byte) (string, error) {
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		var ev struct {
			Type    string `json:"type"`
			Session string `json:"session"`
			Summary string `json:"summary"`
		}
		if json.Unmarshal(line, &ev) != nil || ev.Type != "session-started" {
			continue
		}
		if ev.Session != "" {
			return ev.Session, nil
		}
		if m := startedSummary.FindStringSubmatch(ev.Summary); m != nil {
			return m[1], nil
		}
		return "", errors.New("the session-started line names no session id")
	}
	return "", errors.New("no session-started line")
}

// rewriteToken replaces every occurrence of from that is not preceded or followed by a letter or
// digit (so s0002 in `bf:s0002:end`, `#s0002-ada` and `broodforge/scratch/s0002`, never in s00021).
func rewriteToken(in, from, to []byte) []byte {
	var out bytes.Buffer
	i := 0
	for {
		k := bytes.Index(in[i:], from)
		if k < 0 {
			out.Write(in[i:])
			return out.Bytes()
		}
		at, end := i+k, i+k+len(from)
		if (at > 0 && isAlnum(in[at-1])) || (end < len(in) && isAlnum(in[end])) {
			out.Write(in[i : at+1])
			i = at + 1
			continue
		}
		out.Write(in[i:at])
		out.Write(to)
		i = end
	}
}
