// SPDX-License-Identifier: GPL-3.0-or-later

package config

// EffectiveChannel is one channel as the bot sees it: 03's channel object with every key it
// omits taken from the instance's defaults, plus its id and project. Role lists are as written
// (group names are not expanded); a missing list is empty.
type EffectiveChannel struct {
	ID                   string  `json:"id"`
	Project              string  `json:"project"`
	Reach                string  `json:"reach"`
	Roles                Roles   `json:"roles"`
	Mode                 string  `json:"mode"`
	Harness              string  `json:"harness"`
	IdleTimeoutMinutes   int     `json:"idleTimeoutMinutes"`
	PromptTimeoutMinutes int     `json:"promptTimeoutMinutes"`
	Model                *string `json:"model"`
	Effort               *string `json:"effort"`
}

// Effective merges defaults into one channel.
func (c *Config) Effective(project *Project, id string) EffectiveChannel {
	ch := project.Channels[id]
	d := c.Instance.Defaults
	e := EffectiveChannel{
		ID:                   id,
		Project:              project.ID,
		Reach:                ch.Reach,
		Roles:                Roles{Start: nonNil(ch.Roles.Start), Contribute: nonNil(ch.Roles.Contribute), Answer: nonNil(ch.Roles.Answer)},
		Mode:                 d.Mode,
		Harness:              d.Harness,
		IdleTimeoutMinutes:   d.IdleTimeoutMinutes,
		PromptTimeoutMinutes: d.PromptTimeoutMinutes,
		Model:                ch.Model,
		Effort:               ch.Effort,
	}
	if ch.Mode != nil {
		e.Mode = *ch.Mode
	}
	if ch.Harness != nil {
		e.Harness = *ch.Harness
	}
	if ch.IdleTimeoutMinutes != nil {
		e.IdleTimeoutMinutes = *ch.IdleTimeoutMinutes
	}
	if ch.PromptTimeoutMinutes != nil {
		e.PromptTimeoutMinutes = *ch.PromptTimeoutMinutes
	}
	return e
}

// Channels returns every channel merged, by project id and then channel id.
func (c *Config) Channels() []EffectiveChannel {
	var out []EffectiveChannel
	for i := range c.Projects {
		out = append(out, c.ProjectChannels(&c.Projects[i])...)
	}
	return out
}

// Project returns the project with the id, or nil.
func (c *Config) Project(id string) *Project {
	for i := range c.Projects {
		if c.Projects[i].ID == id {
			return &c.Projects[i]
		}
	}
	return nil
}

// ChannelProject returns the project whose file holds the channel id, or nil. A valid
// configuration has at most one (R-07).
func (c *Config) ChannelProject(id string) *Project {
	for i := range c.Projects {
		if _, ok := c.Projects[i].Channels[id]; ok {
			return &c.Projects[i]
		}
	}
	return nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ProjectChannels returns the project's channels merged, by channel id.
func (c *Config) ProjectChannels(p *Project) []EffectiveChannel {
	var out []EffectiveChannel
	for _, id := range channelIDs(p) {
		out = append(out, c.Effective(p, id))
	}
	return out
}
