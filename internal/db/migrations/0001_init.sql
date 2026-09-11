-- Schema for a single-user 5/3/1 tracker.
--
-- Two rules shape this design:
--
--   1. Training maxes are an append-only log. The user may change a training
--      max at any time, and doing so must never rewrite what they already
--      lifted. Nothing in here ever UPDATEs training_maxes.
--
--   2. Sets store absolute weights, snapshotted when the workout is started.
--      A percentage would be re-derived against whatever training max happens
--      to be current, which would quietly falsify history.

CREATE TABLE settings (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    unit         TEXT    NOT NULL DEFAULT 'lb',
    rounding     REAL    NOT NULL DEFAULT 5,
    bar_weight   REAL    NOT NULL DEFAULT 45,
    plates       TEXT    NOT NULL DEFAULT '[45,35,25,10,5,2.5]', -- JSON, per side
    warmups      INTEGER NOT NULL DEFAULT 1,
    rest_seconds INTEGER NOT NULL DEFAULT 90,
    -- Identifies the cycle rollover whose training-max prompt has been
    -- dismissed, so the offer does not reappear every time the dashboard loads.
    dismissed_cycle TEXT NOT NULL DEFAULT '',
    updated_at   TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE lifts (
    slug      TEXT    PRIMARY KEY,
    name      TEXT    NOT NULL,
    -- Upper-body lifts take the smaller training max increment.
    is_upper  INTEGER NOT NULL,
    -- Position in the training week, 1 through 4.
    day_order INTEGER NOT NULL UNIQUE
);

CREATE TABLE training_maxes (
    id             INTEGER PRIMARY KEY,
    lift_slug      TEXT    NOT NULL REFERENCES lifts(slug),
    value          REAL    NOT NULL CHECK (value > 0),
    effective_from TEXT    NOT NULL DEFAULT (datetime('now')),
    note           TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_training_maxes_lookup
    ON training_maxes (lift_slug, effective_from DESC, id DESC);

CREATE TABLE programs (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL,
    -- JSON serialisation of program.Config. Templates are data, so adding one
    -- is an INSERT rather than a code change.
    config     TEXT    NOT NULL,
    is_active  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE blocks (
    id           INTEGER PRIMARY KEY,
    program_id   INTEGER NOT NULL REFERENCES programs(id),
    number       INTEGER NOT NULL,
    started_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT
);

-- A workout row is created when the session is started, not in advance. That
-- is what lets a training max edit today affect tomorrow's session while
-- leaving yesterday's alone.
CREATE TABLE workouts (
    id           INTEGER PRIMARY KEY,
    block_id     INTEGER NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    lift_slug    TEXT    NOT NULL REFERENCES lifts(slug),
    cycle_index  INTEGER NOT NULL,
    cycle_label  TEXT    NOT NULL, -- snapshot, so renaming a cycle keeps history readable
    phase        TEXT    NOT NULL,
    week         INTEGER NOT NULL,
    training_max REAL    NOT NULL, -- the value in force when this was generated
    started_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT,
    notes        TEXT    NOT NULL DEFAULT '',

    UNIQUE (block_id, cycle_index, week, lift_slug)
);

CREATE INDEX idx_workouts_block ON workouts (block_id, cycle_index, week);
CREATE INDEX idx_workouts_lift  ON workouts (lift_slug, completed_at DESC);

CREATE TABLE sets (
    workout_id   INTEGER NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    idx          INTEGER NOT NULL,
    kind         TEXT    NOT NULL CHECK (kind IN ('warmup', 'main', 'supplemental')),
    weight       REAL    NOT NULL,
    pct          REAL    NOT NULL,
    target_reps  INTEGER NOT NULL,
    is_amrap     INTEGER NOT NULL DEFAULT 0,
    -- NULL until the set is logged.
    actual_reps  INTEGER,
    completed_at TEXT,

    PRIMARY KEY (workout_id, idx)
) WITHOUT ROWID;

INSERT INTO settings (id) VALUES (1);

INSERT INTO lifts (slug, name, is_upper, day_order) VALUES
    ('press',    'Overhead Press', 1, 1),
    ('deadlift', 'Deadlift',       0, 2),
    ('bench',    'Bench Press',    1, 3),
    ('squat',    'Squat',          0, 4);
