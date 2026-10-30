-- SPDX-License-Identifier: GPL-3.0-or-later
--
-- 0004: the plan meter of each credential (ADR 0021 Amendments 2026-10-30): the latest
-- `rate-limit` event of any turn paid by it, one row per credential, replaced in place by the
-- bot's writer when the event arrives in a turn's stream. A reading, not an action: no audit
-- event records it. The values are the event's as read, zero where a field was absent;
-- utilization is a fraction, resets_at unix seconds, seen_at the store's time text.

CREATE TABLE credential_meter (
    credential_id         TEXT PRIMARY KEY REFERENCES credential (id),
    seen_at               TEXT NOT NULL,
    session_id            TEXT NOT NULL,
    status                TEXT NOT NULL,
    rate_limit_type       TEXT NOT NULL,
    five_hour_utilization REAL NOT NULL,
    five_hour_resets_at   INTEGER NOT NULL,
    seven_day_utilization REAL NOT NULL,
    seven_day_resets_at   INTEGER NOT NULL,
    overage_status        TEXT NOT NULL,
    is_using_overage      INTEGER NOT NULL
);
