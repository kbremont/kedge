-- Estimated maxes from before the app, taken from a notebook: either an
-- estimated max (weight x 1) or a PR set such as 135 x 8, converted the same
-- way a logged AMRAP set is. They are kept apart from workouts on purpose. A
-- workout is a session that was planned, performed and finished here; a past
-- estimate is only a number with a date, so it never becomes a session or
-- shows up in history, and it is never evidence for a TM test verdict, the
-- anchor's PR sets or the block-start review. It does count toward a lift's
-- best estimated max and appears on its chart.
CREATE TABLE past_estimates (
    id          INTEGER PRIMARY KEY,
    lift_slug   TEXT    NOT NULL REFERENCES lifts(slug),
    recorded_on TEXT    NOT NULL,                 -- YYYY-MM-DD, the date in the notebook
    weight      REAL    NOT NULL CHECK (weight > 0),
    reps        INTEGER NOT NULL CHECK (reps > 0),
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_past_estimates_lift ON past_estimates (lift_slug, recorded_on, id);
