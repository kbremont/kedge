-- The weight a set was lifted at, when the lifter changed it at the bar. NULL
-- otherwise, so sets.weight stays what was suggested and undoing a set returns
-- to it. Sets logged before this column went by the suggestion, which is what
-- they were logged at.
ALTER TABLE sets ADD COLUMN actual_weight REAL;
