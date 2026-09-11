-- name: GetSettings :one
SELECT * FROM settings WHERE id = 1;

-- name: UpdateSettings :exec
UPDATE settings
SET unit = ?, rounding = ?, bar_weight = ?, plates = ?, warmups = ?,
    rest_warmup_seconds = ?, rest_main_seconds = ?, rest_supp_seconds = ?,
    updated_at = datetime('now')
WHERE id = 1;

-- name: ListLifts :many
SELECT * FROM lifts ORDER BY day_order;

-- name: GetLift :one
SELECT * FROM lifts WHERE slug = ?;

-- name: SetLiftBBB :exec
UPDATE lifts SET bbb_mode = ?, bbb_value = ? WHERE slug = ?;

-- name: GetCurrentTrainingMax :one
SELECT * FROM training_maxes
WHERE lift_slug = ?
ORDER BY effective_from DESC, id DESC
LIMIT 1;

-- ListCurrentTrainingMaxes returns the training max in force for each lift,
-- in training-day order. The subquery picks the most recent entry per lift
-- from the append-only log.
--
-- name: ListCurrentTrainingMaxes :many
SELECT l.slug, l.name, l.is_upper, l.day_order, tm.value, tm.effective_from
FROM lifts l
JOIN training_maxes tm ON tm.lift_slug = l.slug
WHERE tm.id = (
    SELECT t2.id FROM training_maxes t2
    WHERE t2.lift_slug = l.slug
    ORDER BY t2.effective_from DESC, t2.id DESC
    LIMIT 1
)
ORDER BY l.day_order;

-- name: SetTrainingMax :one
INSERT INTO training_maxes (lift_slug, value, note)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListTrainingMaxHistory :many
SELECT * FROM training_maxes
WHERE lift_slug = ?
ORDER BY effective_from DESC, id DESC;

-- name: GetActiveProgram :one
SELECT * FROM programs WHERE is_active = 1 LIMIT 1;

-- name: InsertProgram :one
INSERT INTO programs (name, config, is_active) VALUES (?, ?, ?) RETURNING *;

-- name: DeactivateAllPrograms :exec
UPDATE programs SET is_active = 0;

-- name: ActivateProgram :exec
UPDATE programs SET is_active = 1 WHERE id = ?;

-- name: GetCurrentBlock :one
SELECT * FROM blocks
WHERE completed_at IS NULL
ORDER BY number DESC, id DESC
LIMIT 1;

-- name: InsertBlock :one
INSERT INTO blocks (program_id, number) VALUES (?, ?) RETURNING *;

-- name: CompleteBlock :exec
UPDATE blocks SET completed_at = datetime('now') WHERE id = ?;

-- name: CountBlocks :one
SELECT COUNT(*) FROM blocks;

-- name: ListWorkoutsInBlock :many
SELECT * FROM workouts
WHERE block_id = ?
ORDER BY cycle_index, round_number, id;

-- name: GetWorkout :one
SELECT * FROM workouts WHERE id = ?;

-- name: GetWorkoutAt :one
SELECT * FROM workouts
WHERE block_id = ? AND cycle_index = ? AND round_number = ? AND lift_slug = ?;

-- name: InsertWorkout :one
INSERT INTO workouts (block_id, lift_slug, cycle_index, cycle_label, phase, round_number, training_max)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CompleteWorkout :exec
UPDATE workouts SET completed_at = datetime('now') WHERE id = ?;

-- name: SetWorkoutNotes :exec
UPDATE workouts SET notes = ? WHERE id = ?;

-- name: DeleteWorkout :exec
DELETE FROM workouts WHERE id = ?;

-- name: ListRecentWorkouts :many
SELECT w.*, l.name AS lift_name
FROM workouts w
JOIN lifts l ON l.slug = w.lift_slug
WHERE w.completed_at IS NOT NULL
ORDER BY w.completed_at DESC, w.id DESC
LIMIT ?;

-- name: InsertSet :exec
INSERT INTO sets (workout_id, idx, kind, weight, pct, target_reps, is_amrap)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListSets :many
SELECT * FROM sets WHERE workout_id = ? ORDER BY idx;

-- name: LogSet :exec
UPDATE sets
SET actual_reps = ?, completed_at = datetime('now')
WHERE workout_id = ? AND idx = ?;

-- name: ClearSet :exec
UPDATE sets
SET actual_reps = NULL, completed_at = NULL
WHERE workout_id = ? AND idx = ?;

-- name: CountRemainingSets :one
SELECT COUNT(*) FROM sets WHERE workout_id = ? AND completed_at IS NULL;

-- ListAMRAPSets feeds the progress chart and PR list. Leader cycles run 5's
-- PRO, so this only yields data from anchor rounds.
--
-- name: ListAMRAPSets :many
SELECT s.weight, s.actual_reps, w.completed_at, w.training_max
FROM sets s
JOIN workouts w ON w.id = s.workout_id
WHERE w.lift_slug = ?
  AND s.is_amrap = 1
  AND s.actual_reps IS NOT NULL
  AND w.completed_at IS NOT NULL
ORDER BY w.completed_at, w.id;

-- name: DismissCycle :exec
UPDATE settings SET dismissed_cycle = ? WHERE id = 1;

-- name: SetBlockStartCycle :exec
UPDATE blocks SET start_cycle_index = ? WHERE id = ?;

-- InsertCompletedSet writes a set that was performed before the app knew about
-- it, during first-run onboarding.
--
-- name: InsertCompletedSet :exec
INSERT INTO sets (workout_id, idx, kind, weight, pct, target_reps, is_amrap,
                  actual_reps, completed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'));
