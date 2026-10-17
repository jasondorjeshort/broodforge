// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"

	"github.com/jasondorjeshort/broodforge/internal/control"
	"github.com/jasondorjeshort/broodforge/internal/discord"
)

// CodeNoPanel is `panel show` for a channel whose panel this bot has not posted.
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
// posted. Held by the bot in memory until the store records it.
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
			return nil, control.NewError(CodeNoPanel, "no panel posted in "+r.Channel+" since the bot started")
		}
		return encodeDiscord(p)
	})
}
