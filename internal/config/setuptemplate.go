// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// The setup configuration (ADR 0027; decisions/0027-setup-commands.md): `broodforge setup`
// writes config/instance.json and config/projects/<id>.json from these templates, each only
// when the file is absent, every key with a `//` comment before it; its later steps add an
// absent key to a present file through Doc and never rewrite a present one (ADR 0021's
// Amendments: absent keys only). The sibling of scratchtemplate.go.

// SetupAllow is the service accounts the push scan must let pass (05-debug-interface-git.md):
// the OS user names are never literals, and the project's name is in every import path.
var SetupAllow = []string{"broodforge", "broodforge-engine"}

// DryRunChannel is one channel of setup's dry-run form: its id and harness.
type DryRunChannel struct {
	ID      string
	Harness string
}

// The development placeholders (decisions/0027-setup-commands.md, The dry-run form): the
// channels `setup --discord dry-run` writes, 100 for the fake and 200 for the real engine, and
// the trusted role 10. They are the scratch template's ids too, and never Discord's (a snowflake
// has 17 digits or more), which is how a `--discord live` run tells them from the owner's.
var DryRunChannels = []DryRunChannel{{ID: "100", Harness: HarnessFake}, {ID: "200", Harness: HarnessClaudeCode}}

// PlaceholderRole is the dry-run form's trusted role id.
const PlaceholderRole = "10"

// PlaceholderChannel reports whether id is one of DryRunChannels.
func PlaceholderChannel(id string) bool {
	for _, c := range DryRunChannels {
		if c.ID == id {
			return true
		}
	}
	return false
}

// SetupOptions are what the instance template needs from setup.
type SetupOptions struct {
	InstanceName string
	KeyFile      string // credentials.keyFile, slash form
	Install      string // paths.install, slash form; "" lets the binary's parent directory stand
}

// SetupProjectOptions are what the project template needs from setup.
type SetupProjectOptions struct {
	ID        string
	Name      string
	Canonical string // repo.canonical: <root>/projects/<id>, slash form
	Forge     string // "github" or ""
	Handoff   string // pull-request or push
	URL       string // the repository URL; "" leaves SetupURLKey out
}

// SetupURLKey holds, in a project file's repo object, the repository URL setup clones the
// canonical from, so a rerun after a refused clone finds it. A comment key: config ignores it.
const SetupURLKey = "//url"

// SetupInstance is the instance template: every key of 03's example, Discord in dry-run with
// its ids empty until the Discord steps fill them, the engine empty until setup's engine step
// fills it, timeouts 0 and mode prompt, and privacy.allow holding the service accounts.
func SetupInstance(o SetupOptions) *Doc {
	d := &Doc{}
	d.Comment("Broodforge instance configuration (03-architecture.md, Configuration), written by broodforge setup, " +
		"which adds only absent keys. Edit by hand, then run broodforge config check.")
	d.set("The file format version.", RawValue(SchemaVersion), "schemaVersion")
	d.set("Names the control pipe and the session branches; set once.", RawValue(o.InstanceName), "instanceName")
	d.set("The credential store key (ADR 0022), outside the backup set. setup prints it once, for your password manager.",
		RawValue(o.KeyFile), "credentials", "keyFile")
	d.set("live or dry-run. dry-run renders into the outbox and opens no gateway.", RawValue(DiscordDryRun), "discord", "mode")
	d.set("The Discord application behind the bot token; setup's Discord steps write it.", RawValue(""), "discord", "applicationId")
	d.set("The one guild the bot serves.", RawValue(""), "discord", "guildId")
	d.set("The bot-token credential in the store, by id; never the token.", RawValue(""), "discord", "botCredentialId")
	d.set("Where the start-up report and notices go.", RawValue(""), "discord", "noticeChannelId")
	d.set("The engine binary itself (the .exe on Windows); setup's engine step writes it.", RawValue(""), "engine", "path")
	d.set("Must equal the leading number of the binary's --version.", RawValue(""), "engine", "version")
	d.set("Git Bash for the engine on Windows; empty lets the engine find it.", RawValue(""), "engine", "gitBashPath")
	d.set("The fake harness's fixture directory; null outside tests.", json.RawMessage("null"), "engine", "fakeFixtures")
	d.set("same-account, or linux-sudo-bwrap on the droplet.", RawValue(BackendSameAccount), "spawn", "backend")
	d.set("How long an interrupted engine has before it is killed.", RawValue(20), "spawn", "interruptTimeoutSeconds")
	d.set("The directory holding bin/ and share/; empty is the binary's parent directory.", RawValue(o.Install), "paths", "install")
	d.set("A JSON-lines copy of the audit log, or null.", json.RawMessage("null"), "audit", "fileSink")
	d.set("The Discord direct-message login relay (not built); keep false.", RawValue(false), "relay", "loginByDirectMessage")
	d.set("Group names to Discord role ids; setup's role step fills trusted.", RawValue([]string{}), "roleGroups", "trusted")
	d.set("Minutes without a message before a session ends; 0 is never.", RawValue(0), "defaults", "idleTimeoutMinutes")
	d.set("Minutes a permission prompt waits for its answer; 0 is until answered.", RawValue(0), "defaults", "promptTimeoutMinutes")
	d.set("prompt asks in Discord before a tool runs; policy decides from the reach table.", RawValue(ModePrompt), "defaults", "mode")
	d.set("claude-code, or fake for tests.", RawValue(HarnessClaudeCode), "defaults", "harness")
	d.set("Literals the push scan refuses wherever they stand.", RawValue([]string{}), "privacy", "never")
	d.set("Literals that lift a hit they equal; the service accounts must be here.", RawValue(SetupAllow), "privacy", "allow")
	d.set("Name <email> pairs that may author and co-author commits.", RawValue([]string{}), "privacy", "identities")
	return d
}

// SetupProject is the project template: the canonical under the root, no channels until the
// Discord steps make them, no git credential until setup's git-credential step adds one.
func SetupProject(o SetupProjectOptions) *Doc {
	d := &Doc{}
	d.Comment("A Broodforge project (03-architecture.md, Configuration), written by broodforge setup, " +
		"which adds only absent keys. The id is set once; a rename is a new file.")
	d.set("The project id: the file's name, the lobby's name, the canonical's directory.", RawValue(o.ID), "id")
	d.set("The name sessions show.", RawValue(o.Name), "name")
	d.set("The canonical checkout setup cloned.", RawValue(o.Canonical), "repo", "canonical")
	d.set("The remote pushes go to.", RawValue("origin"), "repo", "remote")
	if o.URL != "" {
		d.set("The repository setup clones the canonical from (a comment key: the bot ignores it).", RawValue(o.URL), "repo", SetupURLKey)
	}
	d.set("The branch sessions start from; setup's project step writes the clone's.", RawValue(""), "repo", "defaultBranch")
	d.set("github opens pull requests; empty pushes only.", RawValue(o.Forge), "repo", "forge")
	d.set("The git credential in the store, by id; setup's git-credential step adds it.", RawValue(""), "repo", "gitCredentialId")
	d.set("pull-request or push: what a session's end does with its branch.", RawValue(o.Handoff), "handoff")
	d.set("repository: a session reads the project's own CLAUDE.md and .claude; none: it does not.",
		RawValue(InjectRepository), "workflow", "inject")
	d.set("The repository hook events the bot's hook runs as children: PreToolUse, PostToolUse.", RawValue([]string{}), "workflow", "hooks")
	d.set("The category sessions are created under; setup's channels step writes it.", RawValue(""), "venue", "category")
	d.set("A channel that also receives each summary, or null.", json.RawMessage("null"), "venue", "summaryChannel")
	d.set("Discord channel ids to their reach, roles and mode; setup's channels step adds the lobby.",
		json.RawMessage("{}"), "channels")
	return d
}

// Doc is a JSON object as setup reads and writes an owner's file: its members in file order,
// the repeatable `//` comment keys kept, so a key setup adds lands after the owner's, with its
// comment, and nothing the owner wrote is reordered or dropped. The layout is not kept: Bytes
// writes the file indented by two spaces.
type Doc struct{ members []member }

type member struct {
	key   string
	value json.RawMessage
}

// ParseDoc reads one JSON object.
func ParseDoc(raw []byte) (*Doc, error) {
	dec := json.NewDecoder(bytes.NewReader(trimBOM(raw)))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, errors.New("not a JSON object")
	}
	d := &Doc{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := kt.(string)
		if !ok {
			return nil, errors.New("not a JSON object")
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		d.members = append(d.members, member{key, v})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("data after the object")
	}
	return d, nil
}

// ReadDoc reads the object in the file at path; a missing file is (nil, false, nil).
func ReadDoc(path string) (*Doc, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	d, err := ParseDoc(b)
	if err != nil {
		return nil, true, &InvalidError{Problems: []Problem{{File: filepath.ToSlash(path), Message: err.Error()}}}
	}
	return d, true, nil
}

// WriteDoc writes d to path, its directory made when missing.
func WriteDoc(path string, d *Doc) error {
	b, err := d.Bytes()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func (d *Doc) get(key string) (json.RawMessage, bool) {
	for _, m := range d.members {
		if m.key == key {
			return m.value, true
		}
	}
	return nil, false
}

// Get is the value at the dotted path, false when a step of it is absent or not an object.
func (d *Doc) Get(path ...string) (json.RawMessage, bool) {
	cur := d
	for i, k := range path {
		v, ok := cur.get(k)
		if !ok {
			return nil, false
		}
		if i == len(path)-1 {
			return v, true
		}
		sub, err := ParseDoc(v)
		if err != nil {
			return nil, false
		}
		cur = sub
	}
	return nil, false
}

// String is the string at path; "" when it is absent, null or not a string.
func (d *Doc) String(path ...string) string {
	v, ok := d.Get(path...)
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(v, &s) != nil {
		return ""
	}
	return s
}

// Has reports whether the key at path is present and not null.
func (d *Doc) Has(path ...string) bool {
	v, ok := d.Get(path...)
	return ok && string(bytes.TrimSpace(v)) != "null"
}

// Comment appends a `//` member.
func (d *Doc) Comment(text string) {
	d.members = append(d.members, member{CommentPrefix, RawValue(text)})
}

// Set puts value at path: a present key's value is replaced where it stands; an absent key is
// appended to its object, after a `//` member holding comment when comment is not empty, and
// an absent object on the way is created. A step on the way that is not an object is an error.
func (d *Doc) Set(comment string, value json.RawMessage, path ...string) error {
	if len(path) == 0 {
		return errors.New("an empty path")
	}
	key := path[0]
	if len(path) == 1 {
		for i := range d.members {
			if d.members[i].key == key {
				d.members[i].value = value
				return nil
			}
		}
		if comment != "" {
			d.Comment(comment)
		}
		d.members = append(d.members, member{key, value})
		return nil
	}
	sub := &Doc{}
	cur, ok := d.get(key)
	if ok && string(bytes.TrimSpace(cur)) != "null" {
		var err error
		if sub, err = ParseDoc(cur); err != nil {
			return errors.New(key + " is not an object")
		}
	}
	if err := sub.Set(comment, value, path[1:]...); err != nil {
		return err
	}
	b, err := sub.compact()
	if err != nil {
		return err
	}
	for i := range d.members {
		if d.members[i].key == key {
			d.members[i].value = b
			return nil
		}
	}
	d.members = append(d.members, member{key, b})
	return nil
}

// AddAbsent adds to d every member of tmpl that d lacks, at every level where both hold an
// object, each after the `//` member that precedes it in tmpl, and returns the dotted paths it
// added: the rerun rule's "a config key: absent, written" for a file that is present. A present
// key is never changed, whatever its value.
func (d *Doc) AddAbsent(tmpl *Doc) ([]string, error) {
	var added []string
	var comment *member
	for i := range tmpl.members {
		m := tmpl.members[i]
		if m.key == CommentPrefix {
			comment = &tmpl.members[i]
			continue
		}
		cur, ok := d.get(m.key)
		switch {
		case !ok:
			if comment != nil {
				d.members = append(d.members, *comment)
			}
			d.members = append(d.members, m)
			added = append(added, m.key)
		case isObject(cur) && isObject(m.value):
			sub, err := ParseDoc(cur)
			if err != nil {
				return nil, err
			}
			st, err := ParseDoc(m.value)
			if err != nil {
				return nil, err
			}
			more, err := sub.AddAbsent(st)
			if err != nil {
				return nil, err
			}
			if len(more) > 0 {
				b, err := sub.compact()
				if err != nil {
					return nil, err
				}
				for j := range d.members {
					if d.members[j].key == m.key {
						d.members[j].value = b
					}
				}
				for _, p := range more {
					added = append(added, m.key+"."+p)
				}
			}
		}
		comment = nil
	}
	return added, nil
}

func isObject(v json.RawMessage) bool {
	t := bytes.TrimSpace(v)
	return len(t) > 0 && t[0] == '{'
}

// set is Set for the templates, whose paths are always objects.
func (d *Doc) set(comment string, value json.RawMessage, path ...string) {
	if err := d.Set(comment, value, path...); err != nil {
		panic("config: template path " + path[0] + ": " + err.Error())
	}
}

func (d *Doc) compact() (json.RawMessage, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range d.members {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(RawValue(m.key))
		b.WriteByte(':')
		if err := json.Compact(&b, m.value); err != nil {
			return nil, err
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Bytes is the object indented by two spaces, with a final newline.
func (d *Doc) Bytes() ([]byte, error) {
	c, err := d.compact()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, c, "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// RawValue is v as JSON without HTML escaping, so a path holding < or > reads as written.
func RawValue[T string | int | bool | []string](v T) json.RawMessage {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic("config: encode " + err.Error())
	}
	return bytes.TrimSpace(b.Bytes())
}
