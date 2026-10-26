// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// DefaultInstance is the instance name with no flag and no configured one.
const DefaultInstance = "main"

// State is one run: the flags, the resolved root and instance, the store setup opened, the
// project the config step settled, and the steps that printed fix under --check.
type State struct {
	opt    Options
	env    Env
	prompt *prompter
	check  bool // --check: report, change nothing
	alone  bool // `setup <step>`: the one way a present key is replaced

	root       string // absolute, slash form
	instance   string
	configured string // config/instance.json's instanceName when valid, else ""
	st         *store.Store

	project       string // the project id, once known
	repoURL       string // the repository URL, when given or asked
	wroteDefaults bool   // the config step wrote the defaults keys in this run
	fixed         []string

	// The Discord steps: the role question's answer once asked or given (nil: not yet), the
	// steps that wrote a config key in this run (a passing line after a poll is then ok).
	createRole  *bool
	channelsYes *bool // the channels question's answer once asked
	wrote       map[string]bool
	liveNote    string // the role line's note once discord.mode was switched
}

// DefaultRoot is the root with no --root: %ProgramData%\Broodforge\<instance> where the
// ProgramData variable is set, else /srv/broodforge/<instance>.
func DefaultRoot(lookup func(string) (string, bool), instance string) string {
	if pd, ok := lookup("ProgramData"); ok && pd != "" {
		return filepath.ToSlash(filepath.Join(pd, "Broodforge", instance))
	}
	return "/srv/broodforge/" + instance
}

// DefaultKeyFile is credentials.keyFile for a new instance file: on the droplet's default root
// /etc/broodforge/<instance>.key, elsewhere <instance>.key beside the root, outside the backup
// set (ADR 0022 and its ADR 0027 amendment).
func DefaultKeyFile(root, instance string) string {
	if root == "/srv/broodforge/"+instance {
		return "/etc/broodforge/" + instance + ".key"
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(root)), instance+".key"))
}

// resolve settles the instance name and the root before anything is written: the flags, the
// root's own instance file, then the defaults, asked for when no flag gives them (--yes takes
// the default; --check never asks).
func (s *State) resolve() error {
	lookup := s.env.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
		s.env.LookupEnv = lookup
	}
	s.instance = s.opt.Instance
	if s.opt.Root != "" {
		abs, err := filepath.Abs(s.opt.Root)
		if err != nil {
			return err
		}
		s.root = filepath.ToSlash(abs)
		// The root's own name: the default for --instance, and an address to probe besides.
		if d, _, err := config.ReadDoc(s.instancePath()); err == nil && d != nil && d.Has("instanceName") {
			s.configured = d.String("instanceName")
			if !config.ValidInstanceName(s.configured) {
				if s.instance == "" {
					return mismatch("config", "instanceName "+quote(s.configured), "no name that can name the control pipe; pass --instance")
				}
				s.configured = ""
			} else if s.instance == "" {
				s.instance = s.configured
			}
		}
	}
	if s.instance == "" {
		s.instance = DefaultInstance
		if !s.check {
			name, err := s.prompt.Prompt("root", "instance", "Instance name", DefaultInstance)
			if err != nil {
				return err
			}
			s.instance = name
		}
	}
	if !config.ValidInstanceName(s.instance) {
		return control.NewError("bad-args", "instance name "+quote(s.instance)+": want 1 to 64 of [A-Za-z0-9._-], starting with a letter or digit")
	}
	if s.root == "" {
		def := DefaultRoot(lookup, s.instance)
		s.root = def
		if !s.check {
			r, err := s.prompt.Prompt("root", "root", "Instance root", def)
			if err != nil {
				return err
			}
			abs, err := filepath.Abs(filepath.FromSlash(r))
			if err != nil {
				return err
			}
			s.root = filepath.ToSlash(abs)
		}
	}
	return nil
}

func quote(s string) string { return `"` + s + `"` }

// instancePath and projectPath are the config files.
func (s *State) instancePath() string { return config.InstancePath(s.root) }

func (s *State) projectPath(id string) string {
	return filepath.ToSlash(filepath.Join(config.Dir(s.root), "projects", id+".json"))
}

// instanceDoc is config/instance.json; nil when it is absent.
func (s *State) instanceDoc() (*config.Doc, error) {
	d, _, err := config.ReadDoc(s.instancePath())
	return d, err
}

// keyPath is the store key's file as the bot will read it: BROODFORGE_KEY_FILE, else the
// instance file's credentials.keyFile, else the default beside the root.
func (s *State) keyPath(inst *config.Doc) (string, error) {
	configured := DefaultKeyFile(s.root, s.instance)
	if inst != nil && inst.String("credentials", "keyFile") != "" {
		configured = inst.String("credentials", "keyFile")
	}
	p, err := credentials.KeyPath(s.root, s.instance, configured, s.env.LookupEnv)
	if err != nil {
		return "", toControl(err)
	}
	return p, nil
}

// projectIDRE is 03's project id rule.
var projectIDRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

var notIDChars = regexp.MustCompile(`[^a-z0-9-]+`)
var hyphens = regexp.MustCompile(`-{2,}`)

// RepoName is the repository's name in its URL or path: the last segment, `.git` removed.
func RepoName(url string) string {
	u := strings.TrimRight(strings.TrimSpace(url), `/\`)
	if i := strings.LastIndexAny(u, `/\:`); i >= 0 {
		u = u[i+1:]
	}
	return strings.TrimSuffix(u, ".git")
}

// DeriveProjectID is the companion's id rule: the repository's name lowered, runs outside
// [a-z0-9-] made -, collapsed, cut to 32. It is valid when it starts with a letter (03).
func DeriveProjectID(url string) (string, bool) {
	id := strings.ToLower(RepoName(url))
	id = notIDChars.ReplaceAllString(id, "-")
	id = hyphens.ReplaceAllString(id, "-")
	if len(id) > 32 {
		id = id[:32]
	}
	return id, projectIDRE.MatchString(id)
}

// projects is the ids of config/projects/*.json, sorted.
func (s *State) projects() []string {
	ents, err := os.ReadDir(filepath.Join(config.Dir(s.root), "projects"))
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range ents {
		if id, ok := strings.CutSuffix(e.Name(), ".json"); ok && !e.IsDir() {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// resolveProject settles the project id and, when one is given or asked, the repository URL:
// --project, else the id derived from --repo or the asked URL; with no --repo, a root with one
// project file names it. Under --check nothing is asked: ok is false when it stays unknown.
func (s *State) resolveProject() (bool, error) {
	if s.project != "" {
		return true, nil
	}
	if s.opt.Project != "" && !projectIDRE.MatchString(s.opt.Project) {
		return false, control.NewError("bad-args", "--project "+quote(s.opt.Project)+": want "+projectIDRE.String())
	}
	s.repoURL = s.opt.Repo
	if s.repoURL == "" {
		have := s.projects()
		switch {
		case s.opt.Project != "":
			s.project = s.opt.Project
			s.storedURL()
			return true, nil
		case len(have) == 1:
			s.project = have[0]
			s.storedURL()
			return true, nil
		case len(have) > 1:
			if s.check {
				return false, nil
			}
			return false, needsInput("project", "project")
		case s.check:
			return false, nil
		}
		url, err := s.prompt.Prompt("repo", "repo", textRepo, "")
		if err != nil {
			return false, err
		}
		s.repoURL = url
	}
	if s.opt.Project != "" {
		s.project = s.opt.Project
		return true, nil
	}
	id, ok := DeriveProjectID(s.repoURL)
	for !ok {
		if s.check {
			return false, nil
		}
		asked, err := s.prompt.Prompt("project", "project", "Project id (a letter, then letters, digits and -, at most 32):", "")
		if err != nil {
			return false, err
		}
		id, ok = asked, projectIDRE.MatchString(asked)
	}
	s.project = id
	return true, nil
}

// storedURL takes the repository URL the settled project's file keeps (config.SetupURLKey),
// when no flag gave one.
func (s *State) storedURL() {
	if d, _, err := config.ReadDoc(s.projectPath(s.project)); err == nil && d != nil {
		s.repoURL = d.String("repo", config.SetupURLKey)
	}
}

// backend is the spawn backend the instance file names, for the askpass helper of the clone.
func (s *State) backend(inst *config.Doc) (spawn.Backend, error) {
	name := inst.String("spawn", "backend")
	if name == "" {
		name = config.BackendSameAccount
	}
	b, err := spawn.New(spawn.Config{Backend: name, Install: inst.String("paths", "install")})
	if err != nil {
		return nil, toControl(err)
	}
	return b, nil
}

// asControl is errors.As for a *control.Error.
func asControl(err error, ce **control.Error) bool { return errors.As(err, ce) }

// toControl maps a refusal of the packages setup calls to a control error with its code.
func toControl(err error) error {
	if err == nil {
		return nil
	}
	var ce *control.Error
	if errors.As(err, &ce) {
		return ce
	}
	if e, ok := credentials.IsRefusal(err); ok {
		return control.NewError(e.Code, e.Message)
	}
	var ge *git.Error
	if errors.As(err, &ge) {
		return control.NewError(ge.Code, ge.Message)
	}
	var se *spawn.Error
	if errors.As(err, &se) {
		return control.NewError(se.Code, se.Message)
	}
	var inv *config.InvalidError
	if errors.As(err, &inv) {
		return control.NewError("bad-config", inv.Error())
	}
	return err
}
