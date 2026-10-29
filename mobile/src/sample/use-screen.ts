import type { Dashboard, History, Lift, Settings, SetupHistory, Workout } from '@/api/schema.gen';
import { useOverrides } from '@/gallery/store';

import { sample } from '.';

// Each screen's data, as Phase 4's query hooks will return it. Until then it
// is the sample, or the fixture a gallery state put in its place.

export function useDashboard(): Dashboard {
  return useOverrides().o.dashboard ?? sample.dashboard;
}

export function useWorkout(id: string): Workout {
  const { o } = useOverrides();
  if (o.workout) return o.workout;
  return Number(id) === sample.workout.id ? sample.workout : sample.workoutDone;
}

export function useHistory(): History {
  return useOverrides().o.history ?? sample.history;
}

export function useLift(slug: string): Lift {
  const { o } = useOverrides();
  if (o.lift?.slug === slug) return o.lift;
  return sample.lifts[slug] ?? sample.lifts.bench;
}

export function useSettings(): Settings {
  return useOverrides().o.settings ?? sample.settings;
}

export function useSetupHistory(cycle: number): SetupHistory {
  return useOverrides().o.setupHistory ?? sample.setupHistory[cycle] ?? sample.setupHistory[0];
}
