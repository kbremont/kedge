-- The 5/3/1 Forever 7th week: a training max test after the leaders, and an
-- optional deload after the anchor.

-- The top of a rep range such as 3-5, with target_reps the bottom of it. Zero
-- for a set with a fixed rep count, which is every set written before this.
ALTER TABLE sets ADD COLUMN max_reps INTEGER NOT NULL DEFAULT 0;

-- When the offer to lower a training max after a failed TM test was settled,
-- whether the user lowered it or kept it. Only a failed test asks; a TM test
-- session with this still NULL and fewer than 3 reps at 100% holds the anchor.
ALTER TABLE workouts ADD COLUMN tm_reviewed_at TEXT;

-- Optional cycles the user passed on. A skipped cycle is left out of its
-- block's schedule entirely, the same way cycles before a block's start are.
CREATE TABLE skipped_cycles (
    block_id    INTEGER NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    cycle_index INTEGER NOT NULL,
    skipped_at  TEXT    NOT NULL DEFAULT (datetime('now')),

    PRIMARY KEY (block_id, cycle_index)
);

-- Upgrade the stored default program in place: the old 40/50/60 deload at
-- index 2 becomes the TM test, and the 7th week deload is appended after the
-- anchor. Indices 0, 1 and 3 keep their meaning, so recorded workouts stay
-- valid. The JSON is program.LeaderAnchorBBB().Cycles as encoding/json writes
-- it; a test holds the two together. A program that does not have the old
-- default's shape is someone's own and is left alone.
UPDATE programs
SET config = json_set(config, '$.Cycles', json('[{"Label":"Leader 1","Phase":"leader","Main":"5s_pro","Supp":{"Kind":"bbb","Sets":5,"Reps":10,"Pct":0.5},"Rounds":3,"Optional":false},{"Label":"Leader 2","Phase":"leader","Main":"5s_pro","Supp":{"Kind":"bbb","Sets":5,"Reps":10,"Pct":0.6},"Rounds":3,"Optional":false},{"Label":"7th Week TM Test","Phase":"tm_test","Main":"tm_test","Supp":{"Kind":"none","Sets":0,"Reps":0,"Pct":0},"Rounds":1,"Optional":false},{"Label":"Anchor","Phase":"anchor","Main":"531_pr","Supp":{"Kind":"fsl","Sets":5,"Reps":5,"Pct":0},"Rounds":3,"Optional":false},{"Label":"7th Week Deload","Phase":"deload","Main":"7th_deload","Supp":{"Kind":"none","Sets":0,"Reps":0,"Pct":0},"Rounds":1,"Optional":true}]'))
WHERE json_array_length(config, '$.Cycles') = 4
  AND json_extract(config, '$.Cycles[2].Main') = 'deload'
  AND json_extract(config, '$.Cycles[3].Main') = '531_pr';

-- Put an open block that has reached the old deload back at the start of the
-- 7th week, so it runs the TM test instead. Sessions of the old 40/50/60 week
-- are removed (their sets cascade), but only while the anchor has not begun:
-- once it has, that week is history and stays as it was.
DELETE FROM workouts
WHERE cycle_index = 2
  AND phase = 'deload'
  AND block_id IN (SELECT id FROM blocks WHERE completed_at IS NULL)
  AND NOT EXISTS (
      SELECT 1 FROM workouts later
      WHERE later.block_id = workouts.block_id
        AND later.cycle_index >= 3
  )
  AND EXISTS (
      SELECT 1 FROM programs
      WHERE is_active = 1
        AND json_extract(config, '$.Cycles[2].Phase') = 'tm_test'
  );
