// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jasondorjeshort/broodforge/internal/policy"
)

// The rules are 03-architecture.md's (Configuration) and ADR 0021's. Each check returns one
// message per problem, in a fixed order, so `config check` output is stable.

// projectIDRE is 03's project id rule.
var projectIDRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// discordIDRE is a Discord snowflake as config writes it: decimal digits.
var discordIDRE = regexp.MustCompile(`^[0-9]{1,20}$`)

func oneOf(key, got string, allowed ...string) []string {
	if slices.Contains(allowed, got) {
		return nil
	}
	if got == "" {
		return []string{key + ": missing, want " + strings.Join(allowed, " or ")}
	}
	return []string{key + ": " + quote(got) + ", want " + strings.Join(allowed, " or ")}
}

func positive(key string, v *int) []string {
	if v != nil && *v <= 0 {
		return []string{key + ": " + strconv.Itoa(*v) + ", want a positive number"}
	}
	return nil
}

func required(key, v string) []string {
	if v == "" {
		return []string{key + ": missing"}
	}
	return nil
}

func credential(key, id string, exists CredentialExists) []string {
	if id != "" && !exists(id) {
		return []string{key + ": credential " + id + " is not in the credential store"}
	}
	return nil
}

func validateInstance(in *Instance, exists CredentialExists) []string {
	var out []string
	if in.SchemaVersion != SchemaVersion {
		out = append(out, "schemaVersion: "+strconv.Itoa(in.SchemaVersion)+", want "+strconv.Itoa(SchemaVersion))
	}
	switch {
	case in.InstanceName == "":
		out = append(out, "instanceName: missing")
	case !ValidInstanceName(in.InstanceName):
		out = append(out, "instanceName "+quote(in.InstanceName)+": "+instanceNameRule)
	}
	out = append(out, oneOf("discord.mode", in.Discord.Mode, DiscordLive, DiscordDryRun)...)
	if in.Discord.Mode == DiscordLive {
		out = append(out, required("discord.applicationId", in.Discord.ApplicationID)...)
		out = append(out, required("discord.guildId", in.Discord.GuildID)...)
		out = append(out, required("discord.botCredentialId", in.Discord.BotCredentialID)...)
	}
	out = append(out, credential("discord.botCredentialId", in.Discord.BotCredentialID, exists)...)
	out = append(out, oneOf("spawn.backend", in.Spawn.Backend, BackendSameAccount, BackendLinuxSudoBwrap)...)
	out = append(out, positive("spawn.interruptTimeoutSeconds", in.Spawn.InterruptTimeoutSeconds)...)
	for _, g := range sortedGroupNames(in.RoleGroups) {
		key := "roleGroups." + g
		if discordIDRE.MatchString(g) || g == RoleDriver {
			out = append(out, key+": a group name must not be a role id or "+quote(RoleDriver))
		}
		for _, id := range in.RoleGroups[g] {
			if !discordIDRE.MatchString(id) {
				out = append(out, key+": "+quote(id)+" is not a role id")
			}
		}
	}
	d := &in.Defaults
	out = append(out, neverOrPositive("defaults.idleTimeoutMinutes", &d.IdleTimeoutMinutes)...)
	out = append(out, neverOrPositive("defaults.promptTimeoutMinutes", &d.PromptTimeoutMinutes)...)
	out = append(out, oneOf("defaults.mode", d.Mode, ModePrompt, ModePolicy)...)
	out = append(out, oneOf("defaults.harness", d.Harness, HarnessClaudeCode, HarnessFake)...)
	// The scan skips a never literal under three characters, as the base's does, so config
	// refuses one rather than let it pass silently; an allow entry is a literal, never a pattern
	// or a path (the base's policy.js).
	for i, n := range in.Privacy.Never {
		if utf8.RuneCountInString(strings.TrimSpace(n)) < minLiteral {
			out = append(out, "privacy.never."+strconv.Itoa(i)+": "+quote(n)+", want a literal of 3 or more characters")
		}
	}
	for i, a := range in.Privacy.Allow {
		if strings.ContainsAny(a, "*/") {
			out = append(out, "privacy.allow."+strconv.Itoa(i)+": "+quote(a)+", want a literal without * or /")
		}
	}
	for i, id := range in.Privacy.Identities {
		if _, _, ok := ParseIdentity(id); !ok {
			out = append(out, "privacy.identities."+strconv.Itoa(i)+": "+quote(id)+", want Name <email>")
		}
	}
	return out
}

// minLiteral is the shortest literal the push scan reads; it skips a shorter one.
const minLiteral = 3

var identityRE = regexp.MustCompile(`^([^<>\s](?:[^<>]*[^<>\s])?)\s*<([^<>\s@]+@[^<>\s@]+)>$`)

// ParseIdentity reads a privacy.identities entry, `Name <email>`.
func ParseIdentity(s string) (name, email string, ok bool) {
	m := identityRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// MaxTimeoutMinutes caps a timeout in minutes at a year, so the product with time.Minute
// cannot overflow time.Duration. A larger value is a warning and is capped on load
// (capTimeouts), not an error. It is the same year as session.MaxTimeout, which config
// cannot import.
const MaxTimeoutMinutes = 365 * 24 * 60

// neverOrPositive checks a timeout in minutes where 0 (or absent) is none: a session that never
// ends for inactivity, a permission prompt that waits for its answer.
func neverOrPositive(key string, v *int) []string {
	if v != nil && *v < 0 {
		return []string{key + ": " + strconv.Itoa(*v) + ", want 0 (never) or a positive number"}
	}
	return nil
}

func sortedGroupNames(m map[string][]string) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

// validateProject checks one project file; fileName is its base name, which must be <id>.json.
// groups are the instance's role groups, checked only when the instance file parsed.
func validateProject(p *Project, fileName string, groups map[string][]string, checkGroups bool, exists CredentialExists) []string {
	var out []string
	switch {
	case p.ID == "":
		out = append(out, "id: missing")
	case !projectIDRE.MatchString(p.ID):
		out = append(out, "id "+quote(p.ID)+": want "+projectIDRE.String())
	case fileName != p.ID+".json":
		out = append(out, "id "+quote(p.ID)+": the file must be named "+p.ID+".json")
	}
	out = append(out, required("name", p.Name)...)
	out = append(out, required("repo.canonical", p.Repo.Canonical)...)
	if p.Repo.Forge != "" {
		out = append(out, oneOf("repo.forge", p.Repo.Forge, ForgeGitHub)...)
	}
	out = append(out, credential("repo.gitCredentialId", p.Repo.GitCredentialID, exists)...)
	out = append(out, oneOf("handoff", p.HandoffOrDefault(), HandoffPullRequest, HandoffPush)...)
	// The workflow key (ADR 0025 rule 6): only the two events are ever run from a repository.
	w := p.WorkflowOrDefault()
	out = append(out, oneOf("workflow.inject", w.Inject, InjectRepository, InjectNone)...)
	for i, e := range w.Hooks {
		out = append(out, oneOf("workflow.hooks."+strconv.Itoa(i), e, EventPreToolUse, EventPostToolUse)...)
	}
	for _, id := range channelIDs(p) {
		out = append(out, validateChannel("channels."+id, id, p.Channels[id], groups, checkGroups)...)
	}
	return out
}

func validateChannel(key, id string, ch Channel, groups map[string][]string, checkGroups bool) []string {
	var out []string
	if !discordIDRE.MatchString(id) {
		out = append(out, key+": a channel id is decimal digits")
	}
	out = append(out, oneOf(key+".reach", ch.Reach, ReachReadOnly, ReachCode)...)
	if ch.Mode != nil {
		out = append(out, oneOf(key+".mode", *ch.Mode, ModePrompt, ModePolicy)...)
	}
	if ch.Harness != nil {
		out = append(out, oneOf(key+".harness", *ch.Harness, HarnessClaudeCode, HarnessFake)...)
	}
	// model and effort: what the session's start would accept (policy's resolver, P12a); null
	// is unset, the engine's own.
	if ch.Model != nil {
		if _, ok := policy.ResolveModel(*ch.Model); !ok {
			out = append(out, key+".model: "+quote(*ch.Model)+", want "+strings.Join(policy.ModelAliases(), ", ")+
				" or a model id they name")
		}
	}
	if ch.Effort != nil {
		out = append(out, oneOf(key+".effort", *ch.Effort, policy.Efforts...)...)
	}
	out = append(out, neverOrPositive(key+".idleTimeoutMinutes", ch.IdleTimeoutMinutes)...)
	out = append(out, neverOrPositive(key+".promptTimeoutMinutes", ch.PromptTimeoutMinutes)...)
	lists := []struct {
		name   string
		roles  []string
		driver bool
	}{
		{"start", ch.Roles.Start, false},
		{"contribute", ch.Roles.Contribute, false},
		{"answer", ch.Roles.Answer, true},
	}
	for _, l := range lists {
		for _, r := range l.roles {
			switch {
			case r == RoleDriver && l.driver:
			case r == RoleDriver:
				out = append(out, key+".roles."+l.name+": "+quote(RoleDriver)+" is allowed under answer only")
			case discordIDRE.MatchString(r):
			case !checkGroups:
			default:
				if _, ok := groups[r]; !ok {
					out = append(out, key+".roles."+l.name+": "+quote(r)+" is neither a role id nor a group in roleGroups")
				}
			}
		}
	}
	return out
}
