// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Config is one loaded, valid configuration: the instance file and every project file of one
// directory (config/ or data/config-last-good/).
type Config struct {
	// Dir is the directory the files were read from.
	Dir      string
	Instance Instance
	// Projects are sorted by id.
	Projects []Project
	// Warnings are problems that do not refuse the load (ADR 0021: a store row naming a
	// project id config no longer defines; the store check arrives with P5).
	Warnings []Problem
	// Files are the files read, instance first, then the projects by id.
	Files []File
}

// File is one configuration file as read.
type File struct {
	// Name is the path relative to the config directory, slash-separated
	// ("instance.json", "projects/alpha.json").
	Name string
	// Path is the file on disk.
	Path string
	// Raw is the bytes as read, comments and all; last-good keeps these.
	Raw []byte
	// Stripped is the parsed form: comments removed, compact, keys sorted.
	Stripped json.RawMessage
}

// Problem is one validation error or warning, printed as "<file>: <message>".
type Problem struct {
	File    string
	Message string
}

func (p Problem) String() string { return p.File + ": " + p.Message }

// NotFoundError is a missing instance file: the CLI's no-config.
type NotFoundError struct {
	Path string
	Err  error
}

func (e *NotFoundError) Error() string { return e.Err.Error() }
func (e *NotFoundError) Unwrap() error { return e.Err }

// InvalidError is a configuration that does not load, one Problem per line: the CLI's
// bad-config.
type InvalidError struct {
	Problems []Problem
}

func (e *InvalidError) Error() string {
	lines := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		lines[i] = p.String()
	}
	return strings.Join(lines, "\n")
}

// CredentialExists reports whether the credential store holds an id (ADR 0021: a credential
// id the store does not hold is an error naming the id). Nil means AnyCredential.
type CredentialExists func(id string) bool

// AnyCredential is the stub until the credential store exists (P6): every id exists.
func AnyCredential(string) bool { return true }

// Dir is the configuration directory of the instance at root.
func Dir(root string) string { return slashJoin(root, "config") }

// InstancePath is config/instance.json of the instance at root.
func InstancePath(root string) string { return slashJoin(Dir(root), "instance.json") }

// slashJoin is filepath.Join in the slash form every path config prints takes, on every host
// (Windows opens it as well).
func slashJoin(elem ...string) string { return filepath.ToSlash(filepath.Join(elem...)) }

// Load reads and validates config/ of the instance at root. A missing instance file is a
// *NotFoundError; any problem is an *InvalidError listing every problem found.
func Load(root string, exists CredentialExists) (*Config, error) {
	return LoadDir(Dir(root), exists)
}

// LoadDir reads and validates the instance file and the project files of one directory.
func LoadDir(dir string, exists CredentialExists) (*Config, error) {
	if exists == nil {
		exists = AnyCredential
	}
	dir = filepath.ToSlash(dir)
	c := &Config{Dir: dir}
	var problems []Problem
	add := func(path string, msgs ...string) {
		for _, m := range msgs {
			problems = append(problems, Problem{File: path, Message: m})
		}
	}

	ipath := slashJoin(dir, "instance.json")
	src, err := os.ReadFile(ipath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, &NotFoundError{Path: ipath, Err: err}
		}
		return nil, &InvalidError{Problems: []Problem{{File: ipath, Message: err.Error()}}}
	}
	inst, stripped, msgs := decodeFile[Instance](src)
	instanceOK := len(msgs) == 0
	add(ipath, msgs...)
	if instanceOK {
		c.Instance = inst
		c.Files = append(c.Files, File{Name: "instance.json", Path: ipath, Raw: src, Stripped: stripped})
	}

	pdir := slashJoin(dir, "projects")
	entries, err := os.ReadDir(pdir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		add(pdir, err.Error())
	}
	type loaded struct {
		p Project
		f File
	}
	var lps []loaded
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		ppath := slashJoin(pdir, e.Name())
		src, err := os.ReadFile(ppath)
		if err != nil {
			add(ppath, err.Error())
			continue
		}
		p, stripped, msgs := decodeFile[Project](src)
		if len(msgs) > 0 {
			add(ppath, msgs...)
			continue
		}
		lps = append(lps, loaded{p, File{Name: "projects/" + e.Name(), Path: ppath, Raw: src, Stripped: stripped}})
	}
	sort.SliceStable(lps, func(i, j int) bool { return lps[i].p.ID < lps[j].p.ID })
	projectFiles := make([]File, len(lps))
	for i, lp := range lps {
		c.Projects = append(c.Projects, lp.p)
		projectFiles[i] = lp.f
	}

	if instanceOK {
		add(ipath, validateInstance(&c.Instance, exists)...)
	}
	for i := range c.Projects {
		f := projectFiles[i]
		add(f.Path, validateProject(&c.Projects[i], filepath.Base(f.Path), c.Instance.RoleGroups, instanceOK, exists)...)
	}
	problems = append(problems, duplicateChannels(c.Projects, projectFiles)...)
	// One block per file: the instance file first, then the project files by path.
	sort.SliceStable(problems, func(i, j int) bool {
		a, b := problems[i].File, problems[j].File
		if (a == ipath) != (b == ipath) {
			return a == ipath
		}
		return a < b
	})
	if len(problems) > 0 {
		return nil, &InvalidError{Problems: problems}
	}
	c.Files = append(c.Files, projectFiles...)
	c.capTimeouts(ipath, projectFiles)
	return c, nil
}

// capTimeouts caps every timeout above MaxTimeoutMinutes at it and warns, one warning per
// value: "<key> <value> is more than a year; capped at 525600".
func (c *Config) capTimeouts(ipath string, projectFiles []File) {
	cap1 := func(file, key string, v *int) {
		if v != nil && *v > MaxTimeoutMinutes {
			c.Warnings = append(c.Warnings, Problem{File: file, Message: key + " " + strconv.Itoa(*v) +
				" is more than a year; capped at " + strconv.Itoa(MaxTimeoutMinutes)})
			*v = MaxTimeoutMinutes
		}
	}
	d := &c.Instance.Defaults
	cap1(ipath, "defaults.idleTimeoutMinutes", &d.IdleTimeoutMinutes)
	cap1(ipath, "defaults.promptTimeoutMinutes", &d.PromptTimeoutMinutes)
	for i := range c.Projects {
		for _, id := range channelIDs(&c.Projects[i]) {
			ch := c.Projects[i].Channels[id]
			cap1(projectFiles[i].Path, "channels."+id+".idleTimeoutMinutes", ch.IdleTimeoutMinutes)
			cap1(projectFiles[i].Path, "channels."+id+".promptTimeoutMinutes", ch.PromptTimeoutMinutes)
		}
	}
}

// duplicateChannels is R-07: a channel id in two project files is an error, reported on each
// later file and naming the first. files[i] is the file of projects[i].
func duplicateChannels(projects []Project, files []File) []Problem {
	first := map[string]string{} // channel id to the path of the first file holding it
	var out []Problem
	for i := range projects {
		path := files[i].Path
		for _, id := range channelIDs(&projects[i]) {
			if prev, ok := first[id]; ok {
				out = append(out, Problem{File: path, Message: "channel " + id + " is also in " + prev})
				continue
			}
			first[id] = path
		}
	}
	return out
}

func channelIDs(p *Project) []string {
	ids := make([]string, 0, len(p.Channels))
	for id := range p.Channels {
		ids = append(ids, id)
	}
	sortChannelIDs(ids)
	return ids
}

// sortChannelIDs orders Discord ids numerically (shorter first, then lexically).
func sortChannelIDs(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		if len(ids[i]) != len(ids[j]) {
			return len(ids[i]) < len(ids[j])
		}
		return ids[i] < ids[j]
	})
}

// instanceNameRE is the name rule control.ValidInstanceName also applies (the pipe name):
// 1 to 64 of letters, digits, dot, underscore and hyphen, starting with a letter or digit.
var instanceNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// ValidInstanceName reports whether name follows the instanceName rule.
func ValidInstanceName(name string) bool { return instanceNameRE.MatchString(name) }

const instanceNameRule = "want 1 to 64 of [A-Za-z0-9._-], starting with a letter or digit"

// InstanceName reads the one key the control address needs, for a socket client. It reads only
// that key and validates only its name, so a client still finds a bot that started on its
// last-good copy after a bad edit (ADR 0021): when config/instance.json gives no usable name,
// data/config-last-good/instance.json is tried, and the first file's error is returned if that
// fails too. Errors are *NotFoundError (no-config) or *InvalidError (bad-config) naming the file.
func InstanceName(root string) (string, error) {
	name, err := readInstanceName(InstancePath(root))
	if err == nil {
		return name, nil
	}
	recoverLastGood(root)
	if lg, lerr := readInstanceName(slashJoin(LastGoodDir(root), "instance.json")); lerr == nil {
		return lg, nil
	}
	return "", err
}

func readInstanceName(path string) (string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", &NotFoundError{Path: path, Err: err}
		}
		return "", &InvalidError{Problems: []Problem{{File: path, Message: err.Error()}}}
	}
	src = trimBOM(src)
	bad := func(msg string) (string, error) {
		return "", &InvalidError{Problems: []Problem{{File: path, Message: msg}}}
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(src, &top); err != nil {
		return bad(syntaxMessage(src, err))
	}
	raw, ok := top["instanceName"]
	if !ok {
		return bad("no instanceName")
	}
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return bad("instanceName: want a string")
	}
	if !ValidInstanceName(name) {
		return bad("instanceName " + quote(name) + ": " + instanceNameRule)
	}
	return name, nil
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
