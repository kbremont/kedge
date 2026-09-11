-- Per-lift Boring But Big load. bbb_mode is one of:
--   'program' -- use the percentage the active program prescribes (the default)
--   'percent' -- bbb_value is a fraction of this lift's training max
--   'weight'  -- bbb_value is an absolute load in the configured unit
-- The mode is validated in Go rather than with a CHECK constraint, to keep
-- ALTER TABLE ... ADD COLUMN within what SQLite allows.
ALTER TABLE lifts ADD COLUMN bbb_mode  TEXT NOT NULL DEFAULT 'program';
ALTER TABLE lifts ADD COLUMN bbb_value REAL NOT NULL DEFAULT 0;

-- One rest interval per kind of set, replacing the single global one.
ALTER TABLE settings ADD COLUMN rest_warmup_seconds INTEGER NOT NULL DEFAULT 90;
ALTER TABLE settings ADD COLUMN rest_main_seconds   INTEGER NOT NULL DEFAULT 90;
ALTER TABLE settings ADD COLUMN rest_supp_seconds   INTEGER NOT NULL DEFAULT 90;

-- Carry the existing preference forward unchanged, so upgrading changes nothing
-- until the user actually differentiates the three.
UPDATE settings
SET rest_warmup_seconds = rest_seconds,
    rest_main_seconds   = rest_seconds,
    rest_supp_seconds   = rest_seconds;

ALTER TABLE settings DROP COLUMN rest_seconds;
