// SPDX-License-Identifier: GPL-3.0-or-later

package setup

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jasondorjeshort/broodforge/internal/audit"
	"github.com/jasondorjeshort/broodforge/internal/config"
	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/credentials"
	"github.com/jasondorjeshort/broodforge/internal/git"
	"github.com/jasondorjeshort/broodforge/internal/spawn"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// The owner's texts (decisions/0027-setup-commands.md, The owner's texts), quoted as they stand.
const (
	textKeyPause      = "Copy the key into your password manager, then press Enter."
	textRepo          = "Repository URL of the project (its canonical):"
	textGitCredential = "Token for pushes to it (not shown; Enter to add later with setup git-credential):"
	textNext          = "the bot is running. In Discord: /register, then /credential add kind:subscription; then here: broodforge setup login <credential id>"
)

// Steps is the run in the companion's order; the Discord steps are in discord.go.
func Steps() []Step {
	return []Step{
		{"root", stepRoot},
		{"config", stepConfig},
		{"key", stepKey},
		{"engine", stepEngine},
		{"token", stepToken},
		{"application", stepApplication},
		{"guild", stepGuild},
		{"channels", stepChannels},
		{"role", stepRole},
		{"git-credential", stepGitCredential},
		{"project", stepProject},
		{"defaults", stepDefaults},
		{"check", stepCheck},
		{"next", stepNext},
	}
}

func ok(step, detail string) Line   { return Line{Result: ResultOK, Step: step, Detail: detail} }
func skip(step, detail string) Line { return Line{Result: ResultSkip, Step: step, Detail: detail} }
func fix(step, detail string) Line  { return Line{Result: ResultFix, Step: step, Detail: detail} }

// rootDirs are the directories 03 and ADR 0022 name under the root.
var rootDirs = []string{"config/projects", "data", "credentials", "projects", "sessions", "harness", "settings", "backup"}

// stepRoot makes the root's directories and data/store.sqlite with its migrations.
func stepRoot(_ context.Context, s *State) (Line, error) {
	var missing []string
	for _, d := range rootDirs {
		if st, err := os.Stat(filepath.Join(s.root, d)); err != nil || !st.IsDir() {
			missing = append(missing, d+"/")
		}
	}
	if _, err := os.Stat(store.Path(s.root)); err != nil {
		missing = append(missing, "data/store.sqlite")
	}
	if len(missing) == 0 {
		return skip("root", s.root), nil
	}
	if s.check {
		return fix("root", s.root+" lacks "+strings.Join(missing, " ")+"; run setup"), nil
	}
	for _, d := range rootDirs {
		if err := os.MkdirAll(filepath.Join(s.root, d), 0o700); err != nil {
			return Line{}, err
		}
	}
	if _, err := s.openStore(); err != nil {
		return Line{}, toControl(err)
	}
	return ok("root", s.root), nil
}

// stepConfig writes config/instance.json and config/projects/<id>.json from the templates
// when absent and, in a present file, every key the template has and the file lacks (the rerun
// rule per key); the repository is asked here, since its id names the project file.
// privacy.allow without the service accounts is a disagreement: mismatch-config, and `setup
// config` adds them to the owner's list.
func stepConfig(_ context.Context, s *State) (Line, error) {
	const step = "config"
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	var wrote, fixes []string
	tmpl := config.SetupInstance(config.SetupOptions{InstanceName: s.instance,
		KeyFile: DefaultKeyFile(s.root, s.instance), Install: filepath.ToSlash(s.env.Install)})
	if inst != nil {
		changed := false
		if name := inst.String("instanceName"); inst.Has("instanceName") && name != s.instance {
			if !s.alone || s.check {
				return Line{}, mismatch(step, "instanceName "+name, s.instance)
			}
			yes, err := s.replace(step, name, s.instance)
			if err != nil {
				return Line{}, err
			}
			if !yes {
				return skip(step, "kept instanceName "+name), nil
			}
			if err := inst.Set("", config.RawValue(s.instance), "instanceName"); err != nil {
				return Line{}, err
			}
			changed = true
			wrote = append(wrote, "instanceName")
		}
		added, err := inst.AddAbsent(tmpl)
		if err != nil {
			return Line{}, control.NewError("bad-config", s.instancePath()+": "+err.Error())
		}
		if len(added) > 0 {
			if s.check {
				fixes = append(fixes, "config/instance.json lacks "+strings.Join(added, ", "))
			} else {
				changed = true
				wrote = append(wrote, "config/instance.json "+strings.Join(added, ", "))
				for _, k := range added {
					if k == "defaults" || strings.HasPrefix(k, "defaults.") {
						s.wroteDefaults = true
					}
				}
			}
		}
		if lacking := lacksAccounts(inst); len(lacking) > 0 && !slices.Contains(added, "privacy") && !slices.Contains(added, "privacy.allow") {
			have, _ := inst.Get("privacy", "allow")
			if !s.alone || s.check {
				return Line{}, mismatch(step, "privacy.allow "+string(have), "the service accounts "+strings.Join(lacking, ", ")+" missing from it")
			}
			var list []string
			_ = json.Unmarshal(have, &list)
			union := append(list, lacking...)
			yes, err := s.replace(step, "privacy.allow "+string(have), "privacy.allow "+string(config.RawValue(union)))
			if err != nil {
				return Line{}, err
			}
			if !yes {
				return skip(step, "kept privacy.allow "+string(have)), nil
			}
			if err := inst.Set("", config.RawValue(union), "privacy", "allow"); err != nil {
				return Line{}, control.NewError("bad-config", s.instancePath()+": "+err.Error())
			}
			changed = true
			wrote = append(wrote, "privacy.allow")
		}
		if changed {
			if err := config.WriteDoc(s.instancePath(), inst); err != nil {
				return Line{}, err
			}
		}
	}
	known, err := s.resolveProject()
	if err != nil {
		return Line{}, err
	}
	if inst == nil {
		if s.check {
			fixes = append(fixes, "config/instance.json is missing")
		} else {
			if err := config.WriteDoc(s.instancePath(), tmpl); err != nil {
				return Line{}, err
			}
			s.wroteDefaults = true
			wrote = append(wrote, "config/instance.json")
		}
	}
	pfile := "config/projects/" + s.project + ".json"
	if known {
		name := RepoName(s.repoURL)
		if name == "" {
			name = s.project
		}
		forge, handoff := "", config.HandoffPush
		if strings.Contains(strings.ToLower(s.repoURL), "github.com") {
			forge, handoff = config.ForgeGitHub, config.HandoffPullRequest
		}
		ptmpl := config.SetupProject(config.SetupProjectOptions{ID: s.project, Name: name,
			Canonical: filepath.ToSlash(filepath.Join(s.root, "projects", s.project)), Forge: forge, Handoff: handoff,
			URL: s.repoURL})
		pdoc, _, err := config.ReadDoc(s.projectPath(s.project))
		if err != nil {
			return Line{}, toControl(err)
		}
		switch {
		case pdoc == nil && s.check:
			fixes = append(fixes, pfile+" is missing")
		case pdoc == nil:
			if err := config.WriteDoc(s.projectPath(s.project), ptmpl); err != nil {
				return Line{}, err
			}
			wrote = append(wrote, pfile)
		default:
			added, err := pdoc.AddAbsent(ptmpl)
			if err != nil {
				return Line{}, control.NewError("bad-config", s.projectPath(s.project)+": "+err.Error())
			}
			switch {
			case len(added) == 0:
			case s.check:
				fixes = append(fixes, pfile+" lacks "+strings.Join(added, ", "))
			default:
				if err := config.WriteDoc(s.projectPath(s.project), pdoc); err != nil {
					return Line{}, err
				}
				wrote = append(wrote, pfile+" "+strings.Join(added, ", "))
			}
		}
	} else {
		fixes = append(fixes, "no project file")
	}
	switch {
	case len(fixes) > 0:
		return fix(step, strings.Join(fixes, "; ")+"; run setup --repo <url>"), nil
	case len(wrote) > 0:
		return ok(step, "wrote "+strings.Join(wrote, "; ")), nil
	}
	return skip(step, "config/instance.json, "+pfile), nil
}

// lacksAccounts is the service accounts privacy.allow lacks (any case), when it is present.
func lacksAccounts(inst *config.Doc) []string {
	raw, ok := inst.Get("privacy", "allow")
	if !ok {
		return nil
	}
	var list []string
	if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	var out []string
	for _, a := range config.SetupAllow {
		if !slices.ContainsFunc(list, func(v string) bool { return strings.EqualFold(v, a) }) {
			out = append(out, a)
		}
	}
	return out
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

// stepKey is `credential init`: the one print as the ok line, then the pause; a valid key is
// skip; anything else at the path is mismatch-key; `setup key` replaces a key only while the
// credential store holds no entry (key-in-use otherwise).
func stepKey(_ context.Context, s *State) (Line, error) {
	const step = "key"
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	path, err := s.keyPath(inst)
	if err != nil {
		return Line{}, err
	}
	b, rerr := os.ReadFile(path)
	if rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
		return Line{}, control.NewError("key-unreadable", path+": "+rerr.Error())
	}
	present, valid := rerr == nil, rerr == nil && validKey(b)
	switch {
	case !present && s.check:
		return fix(step, "no key file at "+path+"; run setup"), nil
	case !present:
		// A new key would leave the store's sealed entries unreadable for good.
		if err := keyInUse(s.root); err != nil {
			return Line{}, err
		}
		return s.newKey(path)
	case valid && (!s.alone || s.check):
		return skip(step, "exists"), nil
	case !valid && (!s.alone || s.check):
		return Line{}, mismatch(step, "key file "+path, fmt.Sprintf("a file that is not a key (%d bytes)", len(b)))
	}
	if err := keyInUse(s.root); err != nil {
		return Line{}, err
	}
	yes, err := s.replace(step, path, path)
	if err != nil {
		return Line{}, err
	}
	if !yes {
		return skip(step, "kept "+path), nil
	}
	_ = os.Chmod(path, 0o600) // credential init leaves it read-only
	if err := os.Remove(path); err != nil {
		return Line{}, err
	}
	return s.newKey(path)
}

// keyInUse refuses a new key while credentials/store.json holds entries sealed under the old one.
func keyInUse(root string) error {
	n, err := credentials.Entries(credentials.StorePath(root))
	if err != nil {
		return toControl(err)
	}
	if n > 0 {
		return control.NewError(CodeKeyInUse, fmt.Sprintf(
			"credentials/store.json holds %d entries under the present key; restore it from your password manager or remove them first", n))
	}
	return nil
}

// validKey is a key file as credentials.ReadKey reads it: 64 hex digits, surrounding whitespace
// (a CRLF from an editor) ignored.
func validKey(b []byte) bool {
	t := bytes.TrimSpace(b)
	if len(t) != 2*credentials.KeySize {
		return false
	}
	_, err := hex.DecodeString(string(t))
	return err == nil
}

func (s *State) newKey(path string) (Line, error) {
	key, err := credentials.Init(path)
	if err != nil {
		return Line{}, toControl(err)
	}
	return Line{Result: ResultOK, Step: "key", Detail: "created", Key: key, Path: path,
		After: func() error { return s.prompt.Pause(textKeyPause) }}, nil
}

// stepEngine finds the engine (the pinned place, the configured path, PATH), reads its
// --version as spawn preflight does, and writes engine.path and engine.version when absent.
func stepEngine(_ context.Context, s *State) (Line, error) {
	const step = "engine"
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	cfgPath, cfgVer := inst.String("engine", "path"), inst.String("engine", "version")
	found := s.findEngine(cfgPath)
	if found == "" {
		return Line{Result: ResultFix, Step: step, Detail: "install Claude Code", Wait: true}, nil
	}
	_, ver, err := spawn.EngineVersion(found)
	if err != nil {
		var se *spawn.Error
		msg := err.Error()
		if errors.As(err, &se) {
			msg = se.Message
		}
		return Line{Result: ResultFix, Step: step, Detail: found + " --version: " + msg + "; install Claude Code", Wait: true}, nil
	}
	have := found + "  " + ver
	pathOff := cfgPath != "" && filepath.ToSlash(filepath.Clean(cfgPath)) != found
	verOff := cfgVer != "" && cfgVer != ver
	if pathOff || verOff {
		if !s.alone || s.check {
			if pathOff {
				return Line{}, mismatch(step, cfgPath, found)
			}
			return Line{}, mismatch(step, "version "+cfgVer, ver)
		}
		yes, err := s.replace(step, orDash(cfgPath)+"  "+orDash(cfgVer), have)
		if err != nil {
			return Line{}, err
		}
		if !yes {
			return skip(step, "kept "+cfgPath+"  "+cfgVer), nil
		}
		cfgPath, cfgVer = "", ""
	}
	if cfgPath != "" && cfgVer != "" {
		return skip(step, have), nil
	}
	if s.check {
		return fix(step, "engine.path and engine.version are not set; run setup"), nil
	}
	if err := inst.Set("", config.RawValue(found), "engine", "path"); err != nil {
		return Line{}, toControl(err)
	}
	if err := inst.Set("", config.RawValue(ver), "engine", "version"); err != nil {
		return Line{}, toControl(err)
	}
	if err := config.WriteDoc(s.instancePath(), inst); err != nil {
		return Line{}, err
	}
	return ok(step, have), nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// findEngine is the engine binary: the pinned place setup host fills, else the configured
// path when it is a file, else the engine on PATH; "" when there is none. Slash form.
func (s *State) findEngine(configured string) string {
	if p := pinnedEngine(s.env.LookupEnv); p != "" && isFile(p) {
		return filepath.ToSlash(p)
	}
	if configured != "" && isFile(configured) {
		return filepath.ToSlash(filepath.Clean(configured))
	}
	look := s.env.LookPath
	if look == nil {
		return ""
	}
	p, err := look(engineName)
	if err != nil {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return filepath.ToSlash(p)
}

// noConfig is a step that needs config/instance.json run without it.
func (s *State) noConfig(step string) (Line, error) {
	if s.check {
		return fix(step, "config/instance.json is missing; run setup"), nil
	}
	return Line{}, control.NewError(CodeNoConfig, s.instancePath()+" is missing; run setup")
}

// projectDoc is the settled project's file; with no project or no file, the step's fix under
// --check or its refusal.
func (s *State) projectDoc(step string) (*config.Doc, *Line, error) {
	known, err := s.resolveProject()
	if err != nil {
		return nil, nil, err
	}
	if !known {
		l := fix(step, "no project; run setup --repo <url>")
		return nil, &l, nil
	}
	d, _, err := config.ReadDoc(s.projectPath(s.project))
	if err != nil {
		return nil, nil, toControl(err)
	}
	if d == nil {
		if s.check {
			l := fix(step, "config/projects/"+s.project+".json is missing; run setup")
			return nil, &l, nil
		}
		return nil, nil, control.NewError(CodeNoConfig, s.projectPath(s.project)+" is missing; run setup")
	}
	return d, nil, nil
}

// service is the credential store over the store setup opened (ADR 0027 amends ADR 0022: setup
// writes the store while it holds the control address).
func (s *State) service() (*credentials.Service, error) {
	inst, err := s.instanceDoc()
	if err != nil {
		return nil, toControl(err)
	}
	key, err := s.keyPath(inst)
	if err != nil {
		return nil, err
	}
	st, err := s.openStore()
	if err != nil {
		return nil, toControl(err)
	}
	svc, err := credentials.NewService(st, audit.NewWriter(nil), credentials.Config{Root: s.root, KeyFile: key})
	if err != nil {
		return nil, toControl(err)
	}
	return svc, nil
}

// usable reports whether the credential id is in the store and not removed.
func usable(q store.Querier, id string) bool {
	r, err := credentials.Show(q, id)
	return err == nil && r.Status != credentials.StatusRemoved
}

// stepGitCredential stores the project's git credential, read hidden or deferred with Enter;
// `setup git-credential` reads it from stdin alone (hidden on a terminal), a new id each time,
// the old one removed.
func stepGitCredential(_ context.Context, s *State) (Line, error) {
	const step = "git-credential"
	pdoc, l, err := s.projectDoc(step)
	if err != nil || l != nil {
		return deref(l), err
	}
	cur := pdoc.String("repo", "gitCredentialId")
	deferred := skip(step, "deferred; add it later with setup git-credential")
	if s.check {
		if cur == "" {
			return deferred, nil
		}
		r, err := store.OpenReadOnly(s.root)
		if err != nil {
			return fix(step, "no store to hold "+cur+"; run setup"), nil
		}
		defer r.Close()
		if !usable(r, cur) {
			return Line{}, mismatch(step, cur, "no such credential")
		}
		return skip(step, cur), nil
	}
	svc, err := s.service()
	if err != nil {
		return Line{}, err
	}
	if !s.alone {
		if cur != "" {
			if !usable(s.st, cur) {
				return Line{}, mismatch(step, cur, "no such credential")
			}
			return skip(step, cur), nil
		}
		if s.opt.Yes {
			return deferred, nil // Enter, the shown default
		}
		value, err := s.prompt.Hidden(textGitCredential)
		if err != nil {
			return Line{}, err
		}
		if value == "" {
			return deferred, nil
		}
		rec, err := svc.AddGit("git "+s.project, []byte(value))
		if err != nil {
			return Line{}, toControl(err)
		}
		if err := s.setGitCredential(pdoc, rec.ID); err != nil {
			_, _ = svc.Remove(rec.ID)
			return Line{}, err
		}
		return ok(step, rec.ID), nil
	}
	// `setup git-credential`: the one secret of this command, hidden on a terminal, else stdin.
	var value string
	if s.prompt.tty {
		value, err = s.prompt.Hidden(textGitCredential)
	} else {
		value, err = s.prompt.All()
	}
	if err != nil {
		return Line{}, err
	}
	if value == "" {
		return Line{}, control.NewError("needs-input-"+step, "pass the token on stdin")
	}
	rec, err := svc.AddGit("git "+s.project, []byte(value))
	if err != nil {
		return Line{}, toControl(err)
	}
	if cur != "" {
		yes, err := s.replace(step, cur, rec.ID)
		if err != nil || !yes {
			_, _ = svc.Remove(rec.ID)
			if err != nil {
				return Line{}, err
			}
			return skip(step, "kept "+cur), nil
		}
	}
	if err := s.setGitCredential(pdoc, rec.ID); err != nil {
		_, _ = svc.Remove(rec.ID)
		return Line{}, err
	}
	if cur != "" && usable(s.st, cur) {
		if _, err := svc.Remove(cur); err != nil {
			return Line{}, toControl(err)
		}
		return ok(step, rec.ID+"; "+cur+" removed"), nil
	}
	return ok(step, rec.ID), nil
}

func deref(l *Line) Line {
	if l == nil {
		return Line{}
	}
	return *l
}

func (s *State) setGitCredential(pdoc *config.Doc, id string) error {
	if err := pdoc.Set("", config.RawValue(id), "repo", "gitCredentialId"); err != nil {
		return toControl(err)
	}
	return config.WriteDoc(s.projectPath(s.project), pdoc)
}

// stepProject clones the canonical into projects/<id>/ with the stored git credential through
// GIT_ASKPASS, the push's route; a present clone with the same remote is skip.
func stepProject(ctx context.Context, s *State) (Line, error) {
	const step = "project"
	pdoc, l, err := s.projectDoc(step)
	if err != nil || l != nil {
		return deref(l), err
	}
	canonical := pdoc.String("repo", "canonical")
	if canonical == "" {
		canonical = filepath.ToSlash(filepath.Join(s.root, "projects", s.project))
	}
	remote := pdoc.String("repo", "remote")
	if remote == "" {
		remote = "origin"
	}
	if exists(canonical) {
		have, err := git.OriginURL(ctx, canonical, remote)
		if err != nil {
			return Line{}, mismatch(step, "repo.canonical "+canonical, "no remote "+remote)
		}
		if s.repoURL != "" && have != s.repoURL {
			if !s.alone || s.check {
				return Line{}, mismatch(step, have, s.repoURL)
			}
			yes, err := s.replace(step, have, s.repoURL)
			if err != nil {
				return Line{}, err
			}
			if !yes {
				return skip(step, "kept "+have), nil
			}
			if err := git.SetOriginURL(ctx, canonical, remote, s.repoURL); err != nil {
				return Line{}, toControl(err)
			}
			return ok(step, canonical+"  "+s.repoURL), nil
		}
		if pdoc.String("repo", "defaultBranch") == "" && !s.check {
			return s.defaultBranch(ctx, pdoc, canonical)
		}
		return skip(step, canonical), nil
	}
	if s.check {
		return fix(step, "no clone at "+canonical+"; run setup"), nil
	}
	if s.repoURL == "" {
		url, err := s.prompt.Prompt("repo", "repo", textRepo, "")
		if err != nil {
			return Line{}, err
		}
		s.repoURL = url
	}
	if pdoc.String("repo", config.SetupURLKey) != s.repoURL {
		// Kept in the project file, so the rerun the refused clone's fix asks for finds it.
		if err := pdoc.Set("The repository setup clones the canonical from (a comment key: the bot ignores it).",
			config.RawValue(s.repoURL), "repo", config.SetupURLKey); err != nil {
			return Line{}, toControl(err)
		}
		if err := config.WriteDoc(s.projectPath(s.project), pdoc); err != nil {
			return Line{}, err
		}
	}
	inst, err := s.instanceDoc()
	if err != nil || inst == nil {
		return s.noConfig(step)
	}
	b, err := s.backend(inst)
	if err != nil {
		return Line{}, err
	}
	var token []byte
	if id := pdoc.String("repo", "gitCredentialId"); id != "" {
		svc, err := s.service()
		if err != nil {
			return Line{}, err
		}
		if token, err = svc.Secret(id); err != nil {
			return Line{}, toControl(err)
		}
	}
	err = git.CloneCanonical(ctx, b.Askpass(), s.repoURL, canonical, token)
	clear(token)
	if err != nil {
		msg := err.Error()
		var ge *git.Error
		if errors.As(err, &ge) {
			msg = ge.Message
		}
		msg += "; add the token with setup git-credential and rerun"
		return fix(step, msg), refused(step, msg)
	}
	return s.defaultBranch(ctx, pdoc, canonical)
}

// defaultBranch writes repo.defaultBranch from the clone's HEAD when the file has none.
func (s *State) defaultBranch(ctx context.Context, pdoc *config.Doc, canonical string) (Line, error) {
	branch, err := git.HeadBranch(ctx, canonical)
	if err != nil {
		return Line{}, toControl(err)
	}
	if pdoc.String("repo", "defaultBranch") == "" {
		if err := pdoc.Set("", config.RawValue(branch), "repo", "defaultBranch"); err != nil {
			return Line{}, toControl(err)
		}
		if err := config.WriteDoc(s.projectPath(s.project), pdoc); err != nil {
			return Line{}, err
		}
	}
	return ok("project", canonical+"  "+branch), nil
}

// The defaults step's keys and values: timeouts 0, mode prompt.
var defaultKeys = []struct{ key, value string }{
	{"idleTimeoutMinutes", "0"}, {"promptTimeoutMinutes", "0"}, {"mode", `"prompt"`},
}

// stepDefaults writes the timeouts 0 and mode prompt where absent; the run that wrote the
// instance file wrote them with it.
func stepDefaults(_ context.Context, s *State) (Line, error) {
	const step = "defaults"
	inst, err := s.instanceDoc()
	if err != nil {
		return Line{}, toControl(err)
	}
	if inst == nil {
		return s.noConfig(step)
	}
	const want = "timeouts 0, mode prompt"
	var absent []string
	for _, k := range defaultKeys {
		if !inst.Has("defaults", k.key) {
			absent = append(absent, k.key)
		}
	}
	if len(absent) > 0 {
		if s.check {
			return fix(step, "defaults."+strings.Join(absent, ", defaults.")+" not set; run setup"), nil
		}
		for _, k := range defaultKeys {
			if !inst.Has("defaults", k.key) {
				if err := inst.Set("", []byte(k.value), "defaults", k.key); err != nil {
					return Line{}, toControl(err)
				}
			}
		}
		if err := config.WriteDoc(s.instancePath(), inst); err != nil {
			return Line{}, err
		}
		return ok(step, want), nil
	}
	raw := func(k string) string { v, _ := inst.Get("defaults", k); return strings.TrimSpace(string(v)) }
	have := fmt.Sprintf("timeouts %s and %s, mode %s", raw("idleTimeoutMinutes"), raw("promptTimeoutMinutes"), inst.String("defaults", "mode"))
	same := raw("idleTimeoutMinutes") == "0" && raw("promptTimeoutMinutes") == "0" && inst.String("defaults", "mode") == config.ModePrompt
	if same {
		have = want
	}
	if s.alone && !s.check && !same {
		yes, err := s.replace(step, have, want)
		if err != nil {
			return Line{}, err
		}
		if !yes {
			return skip(step, "kept "+have), nil
		}
		for _, k := range defaultKeys {
			if err := inst.Set("", []byte(k.value), "defaults", k.key); err != nil {
				return Line{}, toControl(err)
			}
		}
		if err := config.WriteDoc(s.instancePath(), inst); err != nil {
			return Line{}, err
		}
		return ok(step, want), nil
	}
	if s.wroteDefaults {
		return ok(step, have), nil
	}
	return skip(step, have), nil
}

// stepCheck runs `config check`, `spawn preflight` and `discord check` in process and echoes
// each line.
func stepCheck(ctx context.Context, s *State) (Line, error) {
	const step = "check"
	var echoes []Echo
	if s.env.Checks != nil {
		echoes = s.env.Checks(ctx, s.root)
	}
	echoes = append(echoes, s.discordEchoes(ctx)...)
	if len(echoes) == 0 {
		return skip(step, "not run"), nil
	}
	failed := 0
	for _, e := range echoes {
		if !e.OK {
			failed++
		}
	}
	if failed == 0 {
		for _, e := range echoes[:len(echoes)-1] {
			if err := s.emit(ok(step, e.Text)); err != nil {
				return Line{}, err
			}
		}
		return ok(step, echoes[len(echoes)-1].Text), nil
	}
	for _, e := range echoes {
		l := ok(step, e.Text)
		if !e.OK {
			l = fix(step, e.Text)
		}
		if err := s.emit(l); err != nil {
			return Line{}, err
		}
	}
	summary := fix(step, fmt.Sprintf("%d of %d lines failed; fix them and rerun", failed, len(echoes)))
	if s.check {
		return summary, nil
	}
	return summary, control.NewError(CodeCheckFailed, step)
}

// stepNext is the one closing line; under --check with a fix printed, what to run instead.
func stepNext(_ context.Context, s *State) (Line, error) {
	if s.check && len(s.fixed) > 0 {
		return fix("next", "run broodforge setup"), nil
	}
	return ok("next", textNext), nil
}
