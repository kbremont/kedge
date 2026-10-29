import { useSyncExternalStore } from 'react';

import type {
  Dashboard,
  History,
  Lift,
  Settings,
  SetupHistory,
  Workout,
} from '@/api/schema.gen';

// Dev only: the screen data a gallery state puts in place of the sample, so
// each state renders in its real route with the real native chrome. Screens
// that hold form state key it on `version`, so opening a state starts them
// fresh.
export type Overrides = {
  dashboard?: Dashboard;
  workout?: Workout;
  history?: History;
  lift?: Lift;
  settings?: Settings;
  setupHistory?: SetupHistory;
  // Messages the server would answer with, shown as the screen shows them.
  connectError?: string;
  setupError?: string;
  estimateError?: string;
  settingsSaved?: boolean;
  // Setup history: sessions ticked as done, by "round-slug", with reps.
  ticked?: Record<string, string>;
  // How far down the screen opens, to show what sits below the fold.
  scrollY?: number | 'end';
};

let state: { version: number; o: Overrides } = { version: 0, o: {} };
const listeners = new Set<() => void>();

export function setOverrides(o: Overrides) {
  state = { version: state.version + 1, o };
  listeners.forEach((l) => l());
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

export function useOverrides() {
  return useSyncExternalStore(subscribe, () => state);
}
