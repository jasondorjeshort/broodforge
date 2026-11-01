-- SPDX-License-Identifier: GPL-3.0-or-later
--
-- 0005: when the bot first saw each model in a session (ADR 0025 rule 10 as amended
-- 2026-10-31; 03-architecture-git.md, Attribution). One row per session and model, the first
-- sighting kept (an insert that leaves an existing row alone): via `turn` at the turn's start
-- for the turn's own model when set, `pin` at an Agent (or Task) launch for the model the call
-- pins (tool_use_id and agent name the call), `stream` at the first assistant line naming any
-- other. The check owes a model from first_seen_at; the trailers file is rewritten from these
-- rows and the usage table together. model is the id as seen (an alias resolved), so two ids of
-- one display name are two rows and the earlier counts. via is checked in Go (0001).
--
-- Backfill: a session from before this migration has no sightings, so each model with usage
-- gets a `turn` row at the start of its first turn with usage (the usage row's own time where no
-- turn row exists), the moment the check owed it from until now, so its earlier commits keep
-- owing the models that worked on them.

CREATE TABLE model_seen (
    session_id    TEXT NOT NULL REFERENCES session (id),
    model         TEXT NOT NULL,
    first_seen_at TEXT NOT NULL,
    via           TEXT NOT NULL,
    tool_use_id   TEXT,
    agent         TEXT,
    PRIMARY KEY (session_id, model)
);

INSERT INTO model_seen (session_id, model, first_seen_at, via)
SELECT u.session_id, u.model, min(coalesce(t.started_at, u.recorded_at)), 'turn'
FROM usage_record u
LEFT JOIN turn t ON t.session_id = u.session_id AND t.seq = u.turn_seq
WHERE u.input_tokens + u.output_tokens + u.cache_read_tokens + u.cache_write_tokens > 0
GROUP BY u.session_id, u.model;
