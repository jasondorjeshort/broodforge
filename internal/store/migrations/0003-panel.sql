-- SPDX-License-Identifier: GPL-3.0-or-later
--
-- 0003: the panel message of each project channel (03-architecture.md, Configuration: recorded
-- per project channel in the store when `panel post <channel id>` runs), so a restarted bot
-- edits the panel it posted instead of posting a second. message_id is the adapter's id, as the
-- outbox shows it; discord_message_id is Discord's, once the live gateway created the message.

CREATE TABLE panel (
    channel_id         TEXT PRIMARY KEY,
    message_id         TEXT NOT NULL,
    posted_at          TEXT NOT NULL,
    discord_message_id TEXT
);
