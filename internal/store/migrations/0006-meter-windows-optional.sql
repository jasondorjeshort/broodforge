-- SPDX-License-Identifier: GPL-3.0-or-later
--
-- 0006: a meter window may be absent (ADR 0021 Amendments 2026-11-07). Some accounts have no
-- five-hour limit, and engine 2.1.319 then sends `unifiedWindows` with `seven_day` alone; either
-- window may be the missing one. A window the event did not carry is NULL in both its columns,
-- never zeros, which would read as a limit that does not exist; a present window has both, zero
-- where a field was absent. SQLite cannot drop NOT NULL in place, so the table is rebuilt.
--
-- Backfill: under 0004 an absent window was stored as utilization 0 resetting at 0. No real
-- window resets at the epoch, so a pair of zeros becomes NULL; every other row is copied as is.

CREATE TABLE credential_meter_new (
    credential_id         TEXT PRIMARY KEY REFERENCES credential (id),
    seen_at               TEXT NOT NULL,
    session_id            TEXT NOT NULL,
    status                TEXT NOT NULL,
    rate_limit_type       TEXT NOT NULL,
    five_hour_utilization REAL,
    five_hour_resets_at   INTEGER,
    seven_day_utilization REAL,
    seven_day_resets_at   INTEGER,
    overage_status        TEXT NOT NULL,
    is_using_overage      INTEGER NOT NULL,
    CHECK ((five_hour_utilization IS NULL) = (five_hour_resets_at IS NULL)),
    CHECK ((seven_day_utilization IS NULL) = (seven_day_resets_at IS NULL))
);

INSERT INTO credential_meter_new (credential_id, seen_at, session_id, status, rate_limit_type,
    five_hour_utilization, five_hour_resets_at, seven_day_utilization, seven_day_resets_at,
    overage_status, is_using_overage)
SELECT credential_id, seen_at, session_id, status, rate_limit_type,
    CASE WHEN five_hour_utilization = 0 AND five_hour_resets_at = 0 THEN NULL ELSE five_hour_utilization END,
    CASE WHEN five_hour_utilization = 0 AND five_hour_resets_at = 0 THEN NULL ELSE five_hour_resets_at END,
    CASE WHEN seven_day_utilization = 0 AND seven_day_resets_at = 0 THEN NULL ELSE seven_day_utilization END,
    CASE WHEN seven_day_utilization = 0 AND seven_day_resets_at = 0 THEN NULL ELSE seven_day_resets_at END,
    overage_status, is_using_overage
FROM credential_meter;

DROP TABLE credential_meter;

ALTER TABLE credential_meter_new RENAME TO credential_meter;
