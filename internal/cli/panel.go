// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/discord"
	"github.com/jasondorjeshort/broodforge/internal/store"
)

// CodeNoPanel is `panel show` for a channel with no panel posted, in this run or one before.
const CodeNoPanel = "no-panel"

var panelColumns = []string{"channel", "messageId", "postedAt"}

// panelPost is `broodforge panel post <channel id>`: posts the project channel's panel (04), or
// refreshes it in place; on a dry-run instance it lands in the outbox.
var panelPost = Command{
	Name:    "panel post",
	Class:   ClassWrite,
	Flags:   []Flag{{Name: "channel", Kind: FlagString, Value: "<channel id>"}},
	Columns: panelColumns,
}

// panelShow is `broodforge panel show <channel id>`: the panel's message id and when it was
// posted, as the bot holds it: loaded from the store at start, saved there by each post.
var panelShow = Command{
	Name:    "panel show",
	Class:   ClassRead,
	Flags:   []Flag{{Name: "channel", Kind: FlagString, Value: "<channel id>"}},
	Columns: panelColumns,
}

func init() {
	commands = append(commands, &panelPost, &panelShow)
	positionals[&panelPost] = []string{"channel"}
	positionals[&panelShow] = []string{"channel"}
}

func panelHandlers(a *discord.Adapter, reg *control.Registry) {
	reg.Handle("panel.post", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
		var r struct{ Channel string }
		if err := json.Unmarshal(call.Args, &r); err != nil {
			return nil, control.NewError(control.CodeBadArgs, err.Error())
		}
		p, err := a.PostPanel(r.Channel)
		if err != nil {
			return nil, sessionError(err)
		}
		return encodeDiscord(p)
	})
	reg.Handle("panel.show", func(_ context.Context, call *control.Call) (json.RawMessage, error) {
		var r struct{ Channel string }
		if err := json.Unmarshal(call.Args, &r); err != nil {
			return nil, control.NewError(control.CodeBadArgs, err.Error())
		}
		p, ok := a.PanelOf(r.Channel)
		if !ok {
			return nil, control.NewError(CodeNoPanel, "no panel posted in "+r.Channel)
		}
		return encodeDiscord(p)
	})
}

// storedPanels is the store's panel records as the adapter takes them at start.
func storedPanels(st *store.Store) ([]discord.Panel, error) {
	rows, err := store.Panels(st)
	if err != nil {
		return nil, err
	}
	out := make([]discord.Panel, 0, len(rows))
	for _, r := range rows {
		out = append(out, discord.Panel{Channel: r.Channel, MessageID: r.MessageID, PostedAt: r.PostedAt, DiscordID: r.DiscordID})
	}
	return out, nil
}

// savePanel records one panel in the store (03, Configuration: per project channel).
func savePanel(st *store.Store) func(discord.Panel) error {
	return func(p discord.Panel) error {
		return store.SavePanel(st, store.PanelRow{Channel: p.Channel, MessageID: p.MessageID, PostedAt: p.PostedAt, DiscordID: p.DiscordID})
	}
}
