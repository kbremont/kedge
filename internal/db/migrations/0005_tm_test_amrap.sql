-- The TM test's 100% set became an AMRAP after the first TM test weeks had
-- already been logged. Those sessions stored it as a 3-5 range, so they were
-- left off the estimated max chart and best, and the TM test result and the
-- block-start review could not find them. Mark them as the AMRAP they now are.
-- Weights and reps stay exactly as logged, and the range's other use, the 7th
-- week deload's 80% set, is not a TM test and is left alone.
UPDATE sets
SET is_amrap = 1
WHERE kind = 'main'
  AND max_reps > 0
  AND is_amrap = 0
  AND workout_id IN (SELECT id FROM workouts WHERE phase = 'tm_test');
