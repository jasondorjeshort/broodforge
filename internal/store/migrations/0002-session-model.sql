-- SPDX-License-Identifier: GPL-3.0-or-later
--
-- 0002: the session's model and effort (R-29; 05-debug-interface-session.md). Copied at start
-- like reach (the start's choice, else the channel's), changed by `session model|effort`, and
-- read by every engine process the session spawns. Null is unset: the engine's own.

ALTER TABLE session ADD COLUMN model TEXT;
ALTER TABLE session ADD COLUMN effort TEXT;
