// SPDX-License-Identifier: GPL-3.0-or-later

package store

// PanelRow is one project channel's panel message (migration 0003): the adapter's message id,
// when it was last posted, and Discord's id for it ("" until the live gateway created it).
type PanelRow struct {
	Channel   string
	MessageID string
	PostedAt  string
	DiscordID string
}

// Panels reads every panel row, by channel id.
func Panels(q Querier) ([]PanelRow, error) {
	var out []PanelRow
	err := q.Query("SELECT channel_id, message_id, posted_at, coalesce(discord_message_id, '') FROM panel ORDER BY channel_id", nil,
		func(r *Row) error {
			out = append(out, PanelRow{Channel: r.Text(0), MessageID: r.Text(1), PostedAt: r.Text(2), DiscordID: r.Text(3)})
			return nil
		})
	return out, err
}

// SavePanel records a channel's panel, replacing the row it had.
func SavePanel(s *Store, p PanelRow) error {
	return s.Write(func(tx *Tx) error {
		_, err := tx.Exec(`INSERT INTO panel (channel_id, message_id, posted_at, discord_message_id) VALUES (?, ?, ?, ?)
ON CONFLICT (channel_id) DO UPDATE SET message_id = excluded.message_id, posted_at = excluded.posted_at,
    discord_message_id = excluded.discord_message_id`,
			Text(p.Channel), Text(p.MessageID), Text(p.PostedAt), OptText(p.DiscordID))
		return err
	})
}
