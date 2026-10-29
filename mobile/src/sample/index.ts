import type {
  Dashboard,
  History,
  Info,
  Lift,
  Settings,
  Setup,
  SetupHistory,
  SetupPosition,
  Workout,
} from '@/api/schema.gen';
import setupHistoryAnchor from '@/api/fixtures/setup-history-anchor.json';
import setupHistoryLeader from '@/api/fixtures/setup-history-leader.json';
import setupHistoryTMTest from '@/api/fixtures/setup-history-tm-test.json';
import setupPosition from '@/api/fixtures/setup-position.json';
import setup from '@/api/fixtures/setup.json';

import dashboard from './dashboard.json';
import history from './history.json';
import bench from './lift-bench.json';
import deadlift from './lift-deadlift.json';
import press from './lift-press.json';
import squat from './lift-squat.json';
import settings from './settings.json';
import workoutDone from './workout-done.json';
import workout from './workout-mid.json';

// Synthetic screens until the API is wired up: the server's -seed-demo block
// read through the API, with one bench session started and four sets logged,
// and BBB overrides for deadlift and squat saved in settings. No real
// lifter's numbers. Setup reads the server's golden responses, since the
// demo block is already past it.
export const sample = {
  dashboard: dashboard as Dashboard,
  workout: workout as Workout,
  // Deadlift, Anchor round 2: the last session finished, opened from history.
  workoutDone: workoutDone as Workout,
  lifts: { press, deadlift, bench, squat } as Record<string, Lift>,
  history: history as History,
  settings: settings as Settings,
  setup: setup as Setup,
  setupPosition: setupPosition as SetupPosition,
  setupHistory: {
    0: setupHistoryLeader,
    // 5's PRO prescribes the same main sets in both leaders.
    1: { ...setupHistoryLeader, cycleIndex: 1, cycleLabel: 'Leader 2' },
    2: setupHistoryTMTest,
    3: setupHistoryAnchor,
  } as Record<number, SetupHistory>,
  info: { apiVersion: 1, version: 'dev' } as Info,
  server: 'http://raspberrypi.tail1e2f.ts.net:3000',
  // The main set before the current one was logged this many seconds ago.
  restStartedAgo: 46,
};
