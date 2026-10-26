// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

	// --from: the export, read before the root is resolved, and what its import wrote.
	from     *config.ExportDoc
	imported []string
	ignored  []string // "<file> <key>": present keys holding another value than the export's
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
		def := DefaultInstance
		if s.from != nil {
			// --from: the export's instance name is the shown default.
			if n := s.from.Instance.String("instanceName"); config.ValidInstanceName(n) {
				def = n
			}
		}
		s.instance = def
		if !s.check {
			name, err := s.prompt.Prompt("root", "instance", "Instance name", def)
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

// readExport reads --from's export and checks it against this binary's schema: a key the
// schema rejects, or a newer schemaVersion, is mismatch-config naming it.
func (s *State) readExport() error {
	path, err := filepath.Abs(s.opt.From)
	if err != nil {
		return control.NewError("bad-args", "--from: "+err.Error())
	}
	e, err := config.ReadExport(path)
	var inv *config.InvalidError
	switch {
	case errors.As(err, &inv):
		var found []string
		for _, p := range inv.Problems {
			found = append(found, p.String())
		}
		// Wording pending the owner's review (the coordinator's interim text).
		return control.NewError("mismatch-config", "export "+filepath.ToSlash(path)+" has "+strings.Join(found, ", ")+
			"; fix the file or export it again with this version")
	case err != nil:
		return control.NewError("bad-args", "--from: "+control.OneLine(err.Error()))
	}
	s.from = e
	return nil
}

// importExport is `config import` inside the run, once the address is bound: the export's copy
// at <root>/data/config-export.json (unless --from names it), then its files written, absent
// keys only. The config step's line reports what it wrote.
func (s *State) importExport() error {
	// The project is settled over the files the import will leave, before it writes any: a
	// refusal here (needs-input-project) leaves the root as it was.
	if _, err := s.resolveProject(); err != nil {
		return err
	}
	src, err := filepath.Abs(s.opt.From)
	if err != nil {
		return err
	}
	dst := filepath.Join(s.root, "data", config.ExportName)
	if filepath.Clean(src) != filepath.Clean(dst) { // a copy onto itself would only rewrite it
		b, err := os.ReadFile(src)
		if err != nil {
			return control.NewError("bad-args", "--from: "+control.OneLine(err.Error()))
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o600); err != nil {
			return err
		}
	}
	// The host-bound keys are filled from this host's templates as the files are written, so no
	// file is left with an empty repo.canonical whatever the run does next.
	wrote, err := config.Import(s.root, s.from, config.ImportOptions{InstanceName: s.instance,
		Instance: s.instanceTemplate(), Project: s.projectTemplate})
	if err != nil {
		return toControl(err)
	}
	for _, w := range wrote {
		if len(w.Keys) > 0 || len(w.Ignored) == 0 {
			s.imported = append(s.imported, w.String())
		}
		for _, k := range w.Ignored {
			s.ignored = append(s.ignored, w.File+" "+k)
		}
	}
	return nil
}

// instanceTemplate is the instance file setup writes on this host.
func (s *State) instanceTemplate() *config.Doc {
	return config.SetupInstance(config.SetupOptions{InstanceName: s.instance,
		KeyFile: DefaultKeyFile(s.root, s.instance), Install: filepath.ToSlash(s.env.Install)})
}

// projectTemplate is the host-bound part of a project file on this host: its canonical under
// the root.
func (s *State) projectTemplate(id string) *config.Doc {
	return config.SetupProject(config.SetupProjectOptions{ID: id, Canonical: s.canonicalDefault(id)})
}

func (s *State) canonicalDefault(id string) string {
	return filepath.ToSlash(filepath.Join(s.root, "projects", id))
}

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

// projects is the ids of config/projects/*.json, sorted; with --from (not --check), the
// export's projects too, so the project is settled over the files the import leaves.
func (s *State) projects() []string {
	var ids []string
	if ents, err := os.ReadDir(filepath.Join(config.Dir(s.root), "projects")); err == nil {
		for _, e := range ents {
			if id, ok := strings.CutSuffix(e.Name(), ".json"); ok && !e.IsDir() {
				ids = append(ids, id)
			}
		}
	}
	if s.from != nil && !s.check {
		for _, id := range s.from.IDs {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
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
	if err := checkRepoURL(s.opt.Repo); err != nil {
		return false, err
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
		if err := checkRepoURL(url); err != nil {
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
	} else if s.from != nil && s.from.Projects[s.project] != nil {
		s.repoURL = s.from.Projects[s.project].String("repo", config.SetupURLKey)
	}
}

// checkRepoURL refuses a repository URL carrying a credential: it would be written to the
// project file and travel in an export.
func checkRepoURL(u string) error {
	if config.RepoURLCredential(u) {
		return control.NewError("bad-args", "the repository URL carries a credential; pass it without the user and "+
			"password, and add the token with setup git-credential")
	}
	return nil
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
