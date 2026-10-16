// SPDX-License-Identifier: GPL-3.0-or-later

package config

// The file format is 03-architecture.md's (Configuration). A key beginning with "//" is
// ignored at every level; every other key must be one of the json tags below, matched exactly.
// Pointer fields are the ones whose absence (or null) means something: a channel key that
// inherits from defaults, an optional path.

// Instance is config/instance.json.
type Instance struct {
	SchemaVersion int                 `json:"schemaVersion"`
	InstanceName  string              `json:"instanceName"`
	Credentials   Credentials         `json:"credentials"`
	Discord       Discord             `json:"discord"`
	Engine        Engine              `json:"engine"`
	Spawn         Spawn               `json:"spawn"`
	Paths         Paths               `json:"paths"`
	Audit         Audit               `json:"audit"`
	Relay         Relay               `json:"relay"`
	RoleGroups    map[string][]string `json:"roleGroups"`
	Defaults      Defaults            `json:"defaults"`
}

// Credentials names the credential store key (ADR 0022).
type Credentials struct {
	KeyFile string `json:"keyFile"`
}

// Discord is the bot's Discord identity and mode.
type Discord struct {
	Mode            string `json:"mode"` // live or dry-run
	ApplicationID   string `json:"applicationId"`
	GuildID         string `json:"guildId"`
	BotCredentialID string `json:"botCredentialId"`
	NoticeChannelID string `json:"noticeChannelId"`
}

// Engine names the harness binary and the fake's fixtures.
type Engine struct {
	Path         string  `json:"path"`
	Version      string  `json:"version"`
	GitBashPath  string  `json:"gitBashPath"`
	FakeFixtures *string `json:"fakeFixtures"`
}

// Spawn picks the spawn backend.
type Spawn struct {
	Backend                 string `json:"backend"` // same-account or linux-sudo-bwrap
	InterruptTimeoutSeconds *int   `json:"interruptTimeoutSeconds"`
}

// Paths holds the install directory.
type Paths struct {
	Install string `json:"install"`
}

// Audit holds the optional file sink (ADR 0021).
type Audit struct {
	FileSink *string `json:"fileSink"`
}

// Relay is the DM login relay, off by default (ADR 0020).
type Relay struct {
	LoginByDirectMessage bool `json:"loginByDirectMessage"`
}

// Defaults are what a channel inherits for a key it omits.
type Defaults struct {
	IdleTimeoutMinutes   int    `json:"idleTimeoutMinutes"`
	PromptTimeoutMinutes int    `json:"promptTimeoutMinutes"`
	Mode                 string `json:"mode"`    // prompt or policy
	Harness              string `json:"harness"` // claude-code or fake
}

// Project is config/projects/<id>.json.
type Project struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Repo     Repo               `json:"repo"`
	Handoff  string             `json:"handoff"` // pull-request (the default) or push
	Venue    Venue              `json:"venue"`
	Channels map[string]Channel `json:"channels"`
}

// HandoffOrDefault is the project's handoff, pull-request when the file omits it.
func (p *Project) HandoffOrDefault() string {
	if p.Handoff == "" {
		return HandoffPullRequest
	}
	return p.Handoff
}

// Repo is the project's canonical checkout and forge.
type Repo struct {
	Canonical       string `json:"canonical"`
	Remote          string `json:"remote"`
	DefaultBranch   string `json:"defaultBranch"`
	Forge           string `json:"forge"`
	GitCredentialID string `json:"gitCredentialId"`
}

// Venue is where sessions are created.
type Venue struct {
	Category       string  `json:"category"`
	SummaryChannel *string `json:"summaryChannel"`
}

// Channel is one entry of a project's channels, keyed by Discord channel id.
type Channel struct {
	Reach                string  `json:"reach"` // read-only or code
	Roles                Roles   `json:"roles"`
	Mode                 *string `json:"mode"`
	Harness              *string `json:"harness"`
	IdleTimeoutMinutes   *int    `json:"idleTimeoutMinutes"`
	PromptTimeoutMinutes *int    `json:"promptTimeoutMinutes"`
	Model                *string `json:"model"`
	Effort               *string `json:"effort"`
}

// Roles are the role lists of a channel: group names from roleGroups or raw role ids, and
// "driver" under Answer meaning the driver only (R-25).
type Roles struct {
	Start      []string `json:"start"`
	Contribute []string `json:"contribute"`
	Answer     []string `json:"answer"`
}

// The enumerated values of 03.
const (
	SchemaVersion = 1

	DiscordLive   = "live"
	DiscordDryRun = "dry-run"

	BackendSameAccount    = "same-account"
	BackendLinuxSudoBwrap = "linux-sudo-bwrap"

	ModePrompt = "prompt"
	ModePolicy = "policy"

	HarnessClaudeCode = "claude-code"
	HarnessFake       = "fake"

	ReachReadOnly = "read-only"
	ReachCode     = "code"

	HandoffPullRequest = "pull-request"
	HandoffPush        = "push"

	ForgeGitHub = "github"

	RoleDriver = "driver"
)
