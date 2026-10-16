-- SPDX-License-Identifier: GPL-3.0-or-later
--
-- 0001: one table per record of docs/design/02-domain.md (Data model) that the store holds,
-- plus ADR 0021's additive columns and the two audit triggers. Instance, project, channel and
-- harness are configuration (ADR 0021), so rows carry their ids and no table holds them.
-- Times are UTC RFC 3339 text to the second (`2026-10-16T12:00:00Z`), so text order is time
-- order. Named values (state, kind, mode, reach, result, decision, phase) are checked in Go,
-- not by CHECK constraints, so a new value is not a table rebuild. The migrations table itself
-- is created by the runner (migrate.go), not here.

CREATE TABLE registration (
    user_id               TEXT PRIMARY KEY,
    git_name              TEXT NOT NULL,
    git_email             TEXT NOT NULL,
    display_name          TEXT NOT NULL,
    registered_at         TEXT NOT NULL,
    default_credential_id TEXT REFERENCES credential (id)
);

CREATE TABLE credential (
    id           TEXT PRIMARY KEY,
    kind         TEXT NOT NULL,
    provider     TEXT NOT NULL,
    user_id      TEXT REFERENCES registration (user_id), -- null: the instance's own (bot, git)
    label        TEXT NOT NULL,
    status       TEXT NOT NULL,
    added_at     TEXT NOT NULL,
    last_used_at TEXT,
    store_ref    TEXT NOT NULL -- a config directory or an id in the credential store, never a secret
);
CREATE INDEX credential_user ON credential (user_id);

CREATE TABLE session (
    id                TEXT PRIMARY KEY,
    channel_id        TEXT NOT NULL,
    project_id        TEXT NOT NULL,
    driver_id         TEXT NOT NULL REFERENCES registration (user_id),
    state             TEXT NOT NULL,
    venue_id          TEXT,
    harness           TEXT NOT NULL,
    credential_id     TEXT NOT NULL REFERENCES credential (id),
    mode              TEXT NOT NULL,
    reach             TEXT NOT NULL, -- copied at start (ADR 0021)
    engine_session_id TEXT,
    started_at        TEXT NOT NULL,
    ended_at          TEXT,
    end_outcome       TEXT
);
CREATE INDEX session_channel ON session (channel_id);
CREATE INDEX session_driver ON session (driver_id);
CREATE INDEX session_state ON session (state);

CREATE TABLE participant (
    session_id TEXT NOT NULL REFERENCES session (id),
    user_id    TEXT NOT NULL REFERENCES registration (user_id),
    role       TEXT NOT NULL,
    joined_at  TEXT NOT NULL,
    PRIMARY KEY (session_id, user_id)
);
CREATE INDEX participant_user ON participant (user_id);

CREATE TABLE worktree (
    path          TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    session_id    TEXT REFERENCES session (id),
    branch        TEXT NOT NULL,
    state         TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    parent_path   TEXT REFERENCES worktree (path),
    missing_since TEXT -- set and cleared by reconciliation (ADR 0021)
);
CREATE INDEX worktree_session ON worktree (session_id);
CREATE INDEX worktree_parent ON worktree (parent_path);

CREATE TABLE turn (
    session_id   TEXT NOT NULL REFERENCES session (id),
    seq          INTEGER NOT NULL,
    requester_id TEXT REFERENCES registration (user_id),
    input        TEXT NOT NULL,
    started_at   TEXT NOT NULL,
    ended_at     TEXT,
    result       TEXT,
    harness_ref  TEXT,
    PRIMARY KEY (session_id, seq)
);
CREATE INDEX turn_requester ON turn (requester_id);

CREATE TABLE permission_decision (
    session_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    turn_seq   INTEGER NOT NULL,
    tool       TEXT NOT NULL,
    summary    TEXT NOT NULL,
    decided_by TEXT NOT NULL,
    user_id    TEXT REFERENCES registration (user_id),
    decision   TEXT NOT NULL,
    reason     TEXT,
    decided_at TEXT NOT NULL,
    PRIMARY KEY (session_id, request_id),
    FOREIGN KEY (session_id, turn_seq) REFERENCES turn (session_id, seq)
);
CREATE INDEX permission_decision_turn ON permission_decision (session_id, turn_seq);

CREATE TABLE git_commit ( -- the Commit record; COMMIT is a keyword
    hash          TEXT PRIMARY KEY,
    session_id    TEXT NOT NULL REFERENCES session (id),
    worktree_path TEXT NOT NULL REFERENCES worktree (path),
    author_id     TEXT NOT NULL REFERENCES registration (user_id),
    co_authors    TEXT NOT NULL DEFAULT '[]', -- JSON array of user ids and model names
    committed_at  TEXT NOT NULL,
    published     TEXT -- null, a pushed branch name or a pull-request URL
);
CREATE INDEX git_commit_session ON git_commit (session_id);

CREATE TABLE usage_record (
    id                 INTEGER PRIMARY KEY,
    session_id         TEXT NOT NULL,
    turn_seq           INTEGER NOT NULL,
    harness_ref        TEXT UNIQUE, -- the harness's own id when a turn reports in pieces
    requester_id       TEXT REFERENCES registration (user_id),
    credential_id      TEXT NOT NULL REFERENCES credential (id),
    channel_id         TEXT NOT NULL, -- copied at write time (ADR 0021)
    project_id         TEXT NOT NULL, -- copied at write time (ADR 0021)
    harness            TEXT NOT NULL,
    model              TEXT NOT NULL,
    input_tokens       INTEGER NOT NULL DEFAULT 0,
    output_tokens      INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens  INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    interrupted        INTEGER NOT NULL DEFAULT 0, -- 1: what the harness reported before the turn was cut (N-21)
    raw                TEXT NOT NULL DEFAULT '{}', -- the turn's modelUsage and usage as reported
    recorded_at        TEXT NOT NULL,
    FOREIGN KEY (session_id, turn_seq) REFERENCES turn (session_id, seq)
);
CREATE INDEX usage_record_turn ON usage_record (session_id, turn_seq);
CREATE INDEX usage_record_requester ON usage_record (requester_id);
CREATE INDEX usage_record_credential ON usage_record (credential_id);
CREATE INDEX usage_record_channel ON usage_record (channel_id);
CREATE INDEX usage_record_project ON usage_record (project_id);
CREATE INDEX usage_record_time ON usage_record (recorded_at);

-- The audit log (ADR 0021): append-only, no foreign keys, so an event about an unregistered
-- user or a refused session is never itself refused.
CREATE TABLE audit_event (
    seq             INTEGER PRIMARY KEY AUTOINCREMENT,
    time            TEXT NOT NULL,
    kind            TEXT NOT NULL,
    phase           TEXT, -- attempted, done, failed, or null
    actor           TEXT NOT NULL,
    session_id      TEXT,
    registration_id TEXT,
    details         TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX audit_event_kind ON audit_event (kind);
CREATE INDEX audit_event_session ON audit_event (session_id);

CREATE TRIGGER audit_event_no_update BEFORE UPDATE ON audit_event
BEGIN
    SELECT RAISE(ABORT, 'audit_event is append-only (ADR 0021); change this trigger only through a numbered migration');
END;

CREATE TRIGGER audit_event_no_delete BEFORE DELETE ON audit_event
BEGIN
    SELECT RAISE(ABORT, 'audit_event is append-only (ADR 0021); change this trigger only through a numbered migration');
END;

-- REPLACE INTO and INSERT OR REPLACE delete the old row without firing the DELETE trigger
-- (recursive_triggers is off), so an insert onto an existing seq is refused here.
CREATE TRIGGER audit_event_no_replace BEFORE INSERT ON audit_event
WHEN EXISTS (SELECT 1 FROM audit_event WHERE seq = NEW.seq)
BEGIN
    SELECT RAISE(ABORT, 'audit_event is append-only (ADR 0021); change this trigger only through a numbered migration');
END;
