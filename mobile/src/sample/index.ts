import type { Dashboard, Lift, Workout } from '@/api/schema.gen';

import dashboard from './dashboard.json';
import bench from './lift-bench.json';
import deadlift from './lift-deadlift.json';
import press from './lift-press.json';
import squat from './lift-squat.json';
import workout from './workout-mid.json';

// Synthetic screens until the API is wired up: the server's -seed-demo block
// read through the API, with one bench session started and four sets logged.
// No real lifter's numbers.
export const sample = {
  dashboard: dashboard as Dashboard,
  workout: workout as Workout,
  lifts: { press, deadlift, bench, squat } as Record<string, Lift>,
  // The main set before the current one was logged this many seconds ago.
  restStartedAgo: 46,
};
