// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TrailerKey is the co-author trailer (R-32); ModelEmail every model's address in it.
const (
	TrailerKey = "Co-Authored-By"
	ModelEmail = "noreply@anthropic.com"
)

// Identity is a git name and email.
type Identity struct {
	Name  string
	Email string
}

// String is `Name <email>`, the form of an author and of a trailer's value.
func (i Identity) String() string { return i.Name + " <" + i.Email + ">" }

// Trailer is the identity's co-author line.
func (i Identity) Trailer() string { return TrailerKey + ": " + i.String() }

// Contributor is a co-author from a moment on: a participant from when they first wrote, a
// model from its first usage row.
type Contributor struct {
	Identity Identity
	Since    time.Time
}

// Attribution is what the check needs from the store (03-architecture.md, Attribution): the
// driver, who authors and commits everything; the other participants in join order; and the
// models with usage in the session, by display name, in order of first use.
type Attribution struct {
	Driver       Identity
	Participants []Contributor
	Models       []Contributor
}

// Required is the trailer lines a commit made at t must carry, in order: every other
// participant who had written by t, then every model with usage by t. A repeated line counts once.
func (a Attribution) Required(t time.Time) []string {
	var out []string
	seen := map[string]bool{a.Driver.Trailer(): true}
	for _, list := range [][]Contributor{a.Participants, a.Models} {
		for _, c := range list {
			line := c.Identity.Trailer()
			if !seen[line] && !c.Since.After(t) {
				seen[line] = true
				out = append(out, line)
			}
		}
	}
	return out
}

// CommitCheck is one commit's result: no problems is ok.
type CommitCheck struct {
	Hash     string
	Problems []string // author:<got>, committer:<got>, missing:<trailer line>
}

// OK reports whether the commit passed.
func (c CommitCheck) OK() bool { return len(c.Problems) == 0 }

// Status is `ok` or the problems joined by "; ".
func (c CommitCheck) Status() string {
	if c.OK() {
		return "ok"
	}
	return strings.Join(c.Problems, "; ")
}

const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
	logFormat = "%H" + fieldSep + "%an" + fieldSep + "%ae" + fieldSep + "%cn" + fieldSep + "%ce" + fieldSep + "%ct" +
		fieldSep + "%(trailers:only,unfold)" + recordSep
)

// Check is the attribution check (ADR 0022 companion, Handoff step 3) on every commit of
// <base>..refs/sessions/<id> in the canonical, oldest first: author and committer equal the
// driver, and the trailers Required at the commit's committer time are present (extra
// trailers are fine; the key is matched without regard to case, the value exactly).
func Check(ctx context.Context, s Session, base string, a Attribution) ([]CommitCheck, error) {
	out, err := run(ctx, CodeRefusedAttribution, s.Canonical, nil, "log", "--reverse", "--topo-order",
		"--format="+logFormat, base+".."+SessionRef(s.ID), "--")
	if err != nil {
		return nil, err
	}
	var checks []CommitCheck
	for _, rec := range strings.Split(out, recordSep) {
		rec = strings.TrimLeft(rec, "\r\n")
		if rec == "" {
			continue
		}
		f := strings.SplitN(rec, fieldSep, 7)
		if len(f) != 7 {
			return nil, &Error{Code: CodeRefusedAttribution, Message: "git log: unexpected output"}
		}
		secs, err := strconv.ParseInt(f[5], 10, 64)
		if err != nil {
			return nil, &Error{Code: CodeRefusedAttribution, Message: "git log: committer time " + strconv.Quote(f[5])}
		}
		c := CommitCheck{Hash: f[0]}
		if got := (Identity{f[1], f[2]}); got != a.Driver {
			c.Problems = append(c.Problems, "author:"+got.String())
		}
		if got := (Identity{f[3], f[4]}); got != a.Driver {
			c.Problems = append(c.Problems, "committer:"+got.String())
		}
		have := coAuthors(f[6])
		for _, line := range a.Required(time.Unix(secs, 0)) {
			if !have[strings.TrimPrefix(line, TrailerKey+": ")] {
				c.Problems = append(c.Problems, "missing:"+line)
			}
		}
		checks = append(checks, c)
	}
	return checks, nil
}

// coAuthors is the set of co-author values in a commit's trailer block.
func coAuthors(trailers string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(trailers, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(k), TrailerKey) {
			out[strings.TrimSpace(v)] = true
		}
	}
	return out
}

// WriteTrailers writes the lines the commit-msg hook appends to <sess>/home/trailers, one per
// line, by temp file and rename. The session (P10) rewrites it as participants and models join.
func WriteTrailers(sessionDir string, lines []string) error {
	path := filepath.FromSlash(TrailersPath(sessionDir))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".trailers-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
	_, werr := f.WriteString(b.String())
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp, path)
	}
	if werr != nil {
		_ = os.Remove(tmp)
	}
	return werr
}
