// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// A config export (ADR 0027, Decision 5; decisions/0027-setup-commands.md, `config export`):
// config/ as one JSON document, so a second machine is set up from one file plus its own
// secrets. The document is an object with three members:
//
//	"//"       a comment saying what the file is
//	"instance" config/instance.json as written, its comments kept
//	"projects" an object of project id to that project's file as written, comments kept
//
// The host-bound keys are emptied (HostBoundInstance, HostBoundProject): paths of this host and
// the credential ids of its store. The repository URL setup keeps in the project file
// (SetupURLKey) travels without its user and password. Nothing else is changed, so the Discord
// ids travel. No secret is in config/, and the store is never read. It is not a backup: the
// store and its secrets stay ADR 0021's.

// ExportName is the default export file under the root's data/.
const ExportName = "config-export.json"

// HostBoundInstance and HostBoundProject are the keys an export empties: the companion's list.
// Each is written "" but the nullable ones (nullBound), written null.
var (
	HostBoundInstance = [][]string{
		{"engine", "path"}, {"engine", "gitBashPath"}, {"engine", "fakeFixtures"}, {"credentials", "keyFile"},
		{"paths", "install"}, {"audit", "fileSink"}, {"discord", "botCredentialId"},
	}
	HostBoundProject = [][]string{{"repo", "canonical"}, {"repo", "gitCredentialId"}}
)

// nullBound are the host-bound keys whose empty value is null: a path that is optional.
var nullBound = map[string]bool{"engine.fakeFixtures": true, "audit.fileSink": true}

func emptyValue(path []string) json.RawMessage {
	if nullBound[strings.Join(path, ".")] {
		return json.RawMessage("null")
	}
	return RawValue("")
}

// isEmptyValue reports whether v is "" or null.
func isEmptyValue(v json.RawMessage) bool {
	t := string(bytes.TrimSpace(v))
	return t == `""` || t == "null"
}

const exportComment = "A Broodforge config export (broodforge config export): config/ with the host-bound keys " +
	"emptied and no secret. On a new machine: broodforge setup --from <this file>."

// Export is the export document of a loaded configuration: c's files as read, comments kept,
// the host-bound keys emptied, the repository URL's credential removed.
func Export(c *Config) ([]byte, error) {
	out := &Doc{}
	out.Comment(exportComment)
	projects := &Doc{}
	for _, f := range c.Files {
		d, err := ParseDoc(f.Raw)
		if err != nil {
			return nil, &InvalidError{Problems: []Problem{{File: f.Path, Message: err.Error()}}}
		}
		if f.Name == "instance.json" {
			if err := emptyKeys(d, HostBoundInstance); err != nil {
				return nil, err
			}
			b, err := d.compact()
			if err != nil {
				return nil, err
			}
			out.members = append(out.members, member{"instance", b})
			continue
		}
		id, ok := strings.CutPrefix(f.Name, "projects/")
		id, ok2 := strings.CutSuffix(id, ".json")
		if !ok || !ok2 {
			continue
		}
		if err := emptyKeys(d, HostBoundProject); err != nil {
			return nil, err
		}
		if u := d.String("repo", SetupURLKey); RepoURLCredential(u) {
			if err := d.Set("", RawValue(StripRepoCredential(u)), "repo", SetupURLKey); err != nil {
				return nil, err
			}
		}
		b, err := d.compact()
		if err != nil {
			return nil, err
		}
		projects.members = append(projects.members, member{id, b})
	}
	pb, err := projects.compact()
	if err != nil {
		return nil, err
	}
	out.members = append(out.members, member{"projects", pb})
	return out.Bytes()
}

// emptyKeys sets every present key of paths to its empty value.
func emptyKeys(d *Doc, paths [][]string) error {
	for _, p := range paths {
		if _, ok := d.Get(p...); !ok {
			continue
		}
		if err := d.Set("", emptyValue(p), p...); err != nil {
			return err
		}
	}
	return nil
}

// RepoURLCredential reports whether a repository URL carries a credential: a password in any
// URL, or any user in an http or https URL (a token passed as the user name). An ssh user
// (ssh://git@host/...) and the scp form (git@host:o/r.git) carry none.
func RepoURLCredential(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User == nil {
		return false
	}
	if _, ok := u.User.Password(); ok {
		return true
	}
	s := strings.ToLower(u.Scheme)
	return s == "http" || s == "https"
}

// StripRepoCredential is the URL without its user and password when RepoURLCredential holds.
func StripRepoCredential(raw string) string {
	if !RepoURLCredential(raw) {
		return raw
	}
	u, _ := url.Parse(strings.TrimSpace(raw))
	u.User = nil
	return u.String()
}

// ExportDoc is a parsed export: the instance file and each project file, by id, sorted.
type ExportDoc struct {
	Instance *Doc
	IDs      []string
	Projects map[string]*Doc
}

// ParseExport reads and checks an export document against this binary's schema and values: the
// three members and no other; the instance and every project as config would decode them (an
// unknown key, a duplicate, a value of the wrong type); the instance's schemaVersion this
// binary's; the values config check refuses that need no host-bound key (instanceName, the
// modes, the ids, the role lists, a project's id and file name); every host-bound key empty; no
// credential in a repository URL. Every problem is one Problem of an *InvalidError, File naming
// the part ("instance", "projects/<id>", "export"), Message naming the key.
func ParseExport(raw []byte) (*ExportDoc, error) {
	top, err := ParseDoc(raw)
	if err != nil {
		return nil, &InvalidError{Problems: []Problem{{File: "export", Message: err.Error()}}}
	}
	var probs []Problem
	add := func(file string, msgs ...string) {
		for _, m := range msgs {
			probs = append(probs, Problem{File: file, Message: m})
		}
	}
	e := &ExportDoc{Projects: map[string]*Doc{}}
	seen := map[string]bool{}
	var projectParts []member
	var inst *Instance
	for _, m := range top.members {
		switch {
		case m.key == CommentPrefix:
			continue
		case seen[m.key]:
			add("export", "duplicate key "+m.key)
			continue
		}
		seen[m.key] = true
		switch m.key {
		case "instance":
			d, err := ParseDoc(m.value)
			if err != nil {
				add("instance", "want a JSON object")
				continue
			}
			e.Instance = d
			v, _, msgs := decodeFile[Instance](m.value)
			add("instance", msgs...)
			add("instance", schemaProblem(m.value)...)
			add("instance", hostBoundProblems(d, HostBoundInstance)...)
			if len(msgs) > 0 {
				continue
			}
			inst = &v
			// The value rules, with the host-bound keys this host fills standing in.
			if v.Discord.BotCredentialID == "" {
				v.Discord.BotCredentialID = "cred-export"
			}
			for _, msg := range validateInstance(&v, AnyCredential) {
				if !strings.HasPrefix(msg, "schemaVersion") {
					add("instance", msg)
				}
			}
		case "projects":
			pd, err := ParseDoc(m.value)
			if err != nil {
				add("projects", "want a JSON object of project id to project")
				continue
			}
			for _, pm := range pd.members {
				if pm.key != CommentPrefix {
					projectParts = append(projectParts, pm)
				}
			}
		default:
			add("export", "unknown key "+m.key)
		}
	}
	if !seen["instance"] {
		add("export", "instance: missing")
	}
	for _, pm := range projectParts {
		file := "projects/" + pm.key
		if !projectIDRE.MatchString(pm.key) {
			add(file, "id "+quote(pm.key)+": want "+projectIDRE.String())
			continue
		}
		if _, dup := e.Projects[pm.key]; dup {
			add("projects", "duplicate key "+pm.key)
			continue
		}
		d, err := ParseDoc(pm.value)
		if err != nil {
			add(file, "want a JSON object")
			continue
		}
		e.Projects[pm.key] = d
		e.IDs = append(e.IDs, pm.key)
		p, _, msgs := decodeFile[Project](pm.value)
		add(file, msgs...)
		add(file, hostBoundProblems(d, HostBoundProject)...)
		if u := d.String("repo", SetupURLKey); RepoURLCredential(u) {
			add(file, "repo."+SetupURLKey+": the URL carries a credential; remove it and add the token with setup git-credential")
		}
		if len(msgs) > 0 {
			continue
		}
		if p.Repo.Canonical == "" {
			p.Repo.Canonical = "<root>/projects/" + pm.key
		}
		var groups map[string][]string
		if inst != nil {
			groups = inst.RoleGroups
		}
		add(file, validateProject(&p, pm.key+".json", groups, inst != nil, AnyCredential)...)
	}
	if len(probs) > 0 {
		return nil, &InvalidError{Problems: probs}
	}
	sort.Strings(e.IDs)
	return e, nil
}

// schemaProblem is the instance's schemaVersion against this binary's.
func schemaProblem(raw json.RawMessage) []string {
	var v struct {
		SchemaVersion *int `json:"schemaVersion"`
	}
	_ = json.Unmarshal(raw, &v)
	switch {
	case v.SchemaVersion == nil:
		return []string{"schemaVersion: missing, want " + strconv.Itoa(SchemaVersion)}
	case *v.SchemaVersion > SchemaVersion:
		return []string{"schemaVersion: " + strconv.Itoa(*v.SchemaVersion) + ", newer than this binary's " + strconv.Itoa(SchemaVersion)}
	case *v.SchemaVersion != SchemaVersion:
		return []string{"schemaVersion: " + strconv.Itoa(*v.SchemaVersion) + ", want " + strconv.Itoa(SchemaVersion)}
	}
	return nil
}

// hostBoundProblems is each host-bound key d holds with a value: this host fills them.
func hostBoundProblems(d *Doc, paths [][]string) []string {
	var out []string
	for _, p := range paths {
		if v, ok := d.Get(p...); ok && !isEmptyValue(v) {
			out = append(out, strings.Join(p, ".")+": "+string(bytes.TrimSpace(v))+", want "+string(emptyValue(p))+
				" (a host-bound key: this host fills it)")
		}
	}
	return out
}

// ImportOptions are what an import takes besides the document.
type ImportOptions struct {
	// InstanceName replaces the export's instanceName when not empty: the name this machine's
	// instance runs under.
	InstanceName string
	// Instance and Project are this host's templates (setup's): the host-bound keys the export
	// left empty are filled from them before a file is written (FillEmpty). nil leaves them empty.
	Instance *Doc
	Project  func(id string) *Doc
}

// Written is one file an import touched: its path relative to the root, the keys it added when
// the file was present (none: the file was written whole), and the keys the file already held
// with another value than the export's, which the import left as they were.
type Written struct {
	File    string
	Keys    []string
	Ignored []string
}

func (w Written) String() string {
	if len(w.Keys) == 0 {
		return w.File
	}
	return w.File + " " + strings.Join(w.Keys, ", ")
}

// Import writes an export into the config/ of root, absent only: a missing file is written
// whole, a present one gains the keys it lacks (Doc.AddAbsent) and nothing else. It returns
// each file it wrote, or found holding other values than the export's (Written.Ignored).
func Import(root string, e *ExportDoc, o ImportOptions) ([]Written, error) {
	var out []Written
	write := func(path, rel string, d *Doc, skip [][]string) error {
		cur, _, err := ReadDoc(path)
		if err != nil {
			return err
		}
		if cur == nil {
			out = append(out, Written{File: rel})
			return WriteDoc(path, d)
		}
		ignored := differs(cur, d, "", skipSet(skip))
		added, err := cur.AddAbsent(d)
		if err != nil {
			return &InvalidError{Problems: []Problem{{File: path, Message: err.Error()}}}
		}
		if len(added) == 0 && len(ignored) == 0 {
			return nil
		}
		out = append(out, Written{File: rel, Keys: added, Ignored: ignored})
		if len(added) == 0 {
			return nil
		}
		return WriteDoc(path, cur)
	}
	inst := &Doc{members: slices.Clone(e.Instance.members)}
	if o.InstanceName != "" {
		if err := inst.Set("", RawValue(o.InstanceName), "instanceName"); err != nil {
			return nil, err
		}
	}
	if o.Instance != nil {
		if _, err := inst.FillEmpty(o.Instance, HostBoundInstance); err != nil {
			return nil, err
		}
	}
	if err := write(InstancePath(root), "config/instance.json", inst, HostBoundInstance); err != nil {
		return out, err
	}
	for _, id := range e.IDs {
		p := &Doc{members: slices.Clone(e.Projects[id].members)}
		if o.Project != nil {
			if _, err := p.FillEmpty(o.Project(id), HostBoundProject); err != nil {
				return nil, err
			}
		}
		if err := write(slashJoin(Dir(root), "projects", id+".json"), "config/projects/"+id+".json", p, HostBoundProject); err != nil {
			return out, err
		}
	}
	return out, nil
}

func skipSet(paths [][]string) map[string]bool {
	m := map[string]bool{}
	for _, p := range paths {
		m[strings.Join(p, ".")] = true
	}
	return m
}

// differs is the dotted path of every key both d and cur hold with different values, objects
// compared key by key, the comment keys (`//url` among them) and the paths in skip left out.
func differs(cur, d *Doc, prefix string, skip map[string]bool) []string {
	var out []string
	for _, m := range d.members {
		if strings.HasPrefix(m.key, CommentPrefix) {
			continue
		}
		path := join(prefix, m.key)
		cv, ok := cur.get(m.key)
		if !ok || skip[path] {
			continue
		}
		if isObject(cv) && isObject(m.value) {
			cs, err1 := ParseDoc(cv)
			ds, err2 := ParseDoc(m.value)
			if err1 == nil && err2 == nil {
				out = append(out, differs(cs, ds, path, skip)...)
				continue
			}
		}
		a, err1 := stripComments(cv)
		b, err2 := stripComments(m.value)
		if err1 != nil || err2 != nil || !bytes.Equal(a, b) {
			out = append(out, path)
		}
	}
	return out
}

// ReadExport reads and parses the export file at path; an *InvalidError's problems name the
// part of the export, not the file.
func ReadExport(path string) (*ExportDoc, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseExport(b)
}

// FillEmpty sets each key of paths that d holds as "" or null to tmpl's value there, when tmpl
// holds a string other than "": a host-bound key an import left empty, filled from this host's
// template as an absent key is. It returns the dotted paths it filled.
func (d *Doc) FillEmpty(tmpl *Doc, paths [][]string) ([]string, error) {
	var filled []string
	for _, p := range paths {
		v, ok := d.Get(p...)
		if !ok || !isEmptyValue(v) {
			continue
		}
		if tmpl.String(p...) == "" {
			continue
		}
		tv, _ := tmpl.Get(p...)
		if err := d.Set("", tv, p...); err != nil {
			return nil, err
		}
		filled = append(filled, strings.Join(p, "."))
	}
	return filled, nil
}
