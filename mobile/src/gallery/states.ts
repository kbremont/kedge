import { router, type Href } from 'expo-router';

import type { Dashboard, History, Lift, Settings, Workout } from '@/api/schema.gen';
import blockReview from '@/api/fixtures/dashboard-block-review.json';
import fresh from '@/api/fixtures/dashboard-fresh.json';
import inProgress from '@/api/fixtures/dashboard-in-progress.json';
import next from '@/api/fixtures/dashboard-next.json';
import skipDeload from '@/api/fixtures/dashboard-skip-deload.json';
import tmTestFailed from '@/api/fixtures/dashboard-tm-test-failed.json';
import tmTestPassed from '@/api/fixtures/dashboard-tm-test-passed.json';
import inputError from '@/api/fixtures/error-input.json';
import historyEmpty from '@/api/fixtures/history-empty.json';
import liftEmpty from '@/api/fixtures/lift-empty.json';
import liftSingle from '@/api/fixtures/lift-single.json';
import workoutAmrap from '@/api/fixtures/workout-amrap.json';
import workoutFinish from '@/api/fixtures/workout-finish.json';
import workoutPlatesShort from '@/api/fixtures/workout-plates-short.json';
import workoutRange from '@/api/fixtures/workout-range.json';
import { sample } from '@/sample';

import { setOverrides, type Overrides } from './store';

// Dev only: every screen and state the design covers, by id, for the gallery
// and the screenshot script. Each opens in its real route, so the captures
// carry the native chrome. Data comes from the sample (the demo block) or the
// server's golden responses.
export type GalleryState = {
  id: string;
  screen: string;
  label: string;
  // Routes to open in order: the first replaces the stack, the rest push.
  paths: Href[];
  data?: Overrides;
};

const d = (x: unknown) => x as Dashboard;
const w = (x: unknown) => x as Workout;

export const galleryStates: GalleryState[] = [
  { id: 'connect', screen: 'Connect', label: 'Default', paths: ['/connect'] },
  {
    id: 'connect-error',
    screen: 'Connect',
    label: 'Error',
    paths: ['/connect'],
    data: { connectError: 'Incorrect password.' },
  },

  { id: 'today', screen: 'Today', label: 'Next up', paths: ['/'] },
  { id: 'today-next', screen: 'Today', label: 'Next up, early in the block', paths: ['/'], data: { dashboard: d(next) } },
  { id: 'today-in-progress', screen: 'Today', label: 'In progress', paths: ['/'], data: { dashboard: d(inProgress) } },
  {
    id: 'today-tm-test-passed',
    screen: 'Today',
    label: 'TM test passed',
    paths: ['/'],
    data: { dashboard: d(tmTestPassed) },
  },
  {
    id: 'today-tm-test-failed',
    screen: 'Today',
    label: 'TM test failed, and blocked',
    paths: ['/'],
    data: { dashboard: d(tmTestFailed) },
  },
  { id: 'today-block-review', screen: 'Today', label: 'Block review', paths: ['/'], data: { dashboard: d(blockReview) } },
  {
    id: 'today-block-review-end',
    screen: 'Today',
    label: 'Block review, accept and blocked',
    paths: ['/'],
    data: { dashboard: d(blockReview), scrollY: 420 },
  },
  { id: 'today-skip-deload', screen: 'Today', label: 'Skip deload', paths: ['/'], data: { dashboard: d(skipDeload) } },
  { id: 'today-empty', screen: 'Today', label: 'Empty', paths: ['/'], data: { dashboard: d(fresh) } },

  { id: 'workout', screen: 'Session', label: 'Rest timer', paths: ['/', `/workout/${sample.workout.id}`] },
  {
    id: 'workout-amrap',
    screen: 'Session',
    label: 'AMRAP picker',
    paths: ['/', `/workout/${sample.workout.id}?logged=1`],
  },
  {
    id: 'workout-amrap-press',
    screen: 'Session',
    label: 'AMRAP picker, press',
    paths: ['/', '/workout/1'],
    data: { workout: w(workoutAmrap) },
  },
  {
    id: 'workout-range',
    screen: 'Session',
    label: '3–5 range picker',
    paths: ['/', '/workout/1'],
    data: { workout: w(workoutRange) },
  },
  {
    id: 'workout-plates-short',
    screen: 'Session',
    label: 'Plates short',
    paths: ['/', '/workout/1?logged=1'],
    data: { workout: w(workoutPlatesShort) },
  },
  {
    id: 'workout-finish',
    screen: 'Session',
    label: 'Finish',
    paths: ['/', '/workout/1'],
    data: { workout: w(workoutFinish) },
  },
  {
    id: 'workout-discard',
    screen: 'Session',
    label: 'Discard',
    paths: ['/', `/workout/${sample.workout.id}?discard=1`],
  },
  {
    id: 'workout-completed',
    screen: 'Session',
    label: 'Completed, from history',
    paths: ['/history', `/workout/${sample.workoutDone.id}?back=History`],
  },

  { id: 'history', screen: 'History', label: 'Career', paths: ['/history'] },
  {
    id: 'history-scrolled',
    screen: 'History',
    label: 'Career, scrolled',
    paths: ['/history'],
    data: { scrollY: 'end' },
  },
  {
    id: 'history-empty',
    screen: 'History',
    label: 'Empty',
    paths: ['/history'],
    data: { history: historyEmpty as History },
  },

  { id: 'lift', screen: 'Lift', label: 'Line', paths: ['/', '/lift/bench'] },
  { id: 'lift-end', screen: 'Lift', label: 'Line, estimates and TM log', paths: ['/', '/lift/bench'], data: { scrollY: 'end' } },
  { id: 'lift-deadlift', screen: 'Lift', label: 'Line, deadlift', paths: ['/', '/lift/deadlift'] },
  {
    id: 'lift-single',
    screen: 'Lift',
    label: 'Single point',
    paths: ['/', '/lift/press'],
    data: { lift: liftSingle as Lift },
  },
  { id: 'lift-empty', screen: 'Lift', label: 'Empty', paths: ['/', '/lift/press'], data: { lift: liftEmpty as Lift } },
  { id: 'lift-add', screen: 'Lift', label: 'Add sheet', paths: ['/', '/lift/bench', '/lift/bench/add-estimate'] },
  {
    id: 'lift-error',
    screen: 'Lift',
    label: 'Error',
    paths: ['/', '/lift/bench', '/lift/bench/add-estimate'],
    data: { estimateError: inputError.error },
  },

  { id: 'settings', screen: 'Settings', label: 'Default', paths: ['/settings'] },
  {
    id: 'settings-rest',
    screen: 'Settings',
    label: 'Loading and rest',
    paths: ['/settings'],
    data: { scrollY: 560 },
  },
  {
    id: 'settings-supplemental',
    screen: 'Settings',
    label: 'Supplemental',
    paths: ['/settings'],
    data: { scrollY: 1180 },
  },
  { id: 'settings-end', screen: 'Settings', label: 'Data and server', paths: ['/settings'], data: { scrollY: 'end' } },
  {
    id: 'settings-saved',
    screen: 'Settings',
    label: 'Saved',
    paths: ['/settings'],
    data: { settingsSaved: true, settings: sample.settings as Settings, scrollY: 1180 },
  },

  { id: 'setup', screen: 'Setup', label: 'Training maxes', paths: ['/setup'] },
  {
    id: 'setup-error',
    screen: 'Setup',
    label: 'Training maxes, error',
    paths: ['/setup'],
    data: { setupError: 'Enter a positive training max for Squat.' },
  },
  { id: 'setup-position', screen: 'Setup', label: 'Position', paths: ['/setup', '/setup/position'] },
  {
    id: 'setup-leader',
    screen: 'Setup',
    label: 'Leader history',
    paths: ['/setup', '/setup/position', '/setup/history?cycle=0'],
    data: { ticked: { '1-press': '', '1-deadlift': '', '1-bench': '', '1-squat': '', '2-press': '' } },
  },
  {
    id: 'setup-anchor',
    screen: 'Setup',
    label: 'Anchor history',
    paths: ['/setup', '/setup/position', '/setup/history?cycle=3'],
    data: { ticked: { '1-press': '7', '1-deadlift': '9', '1-bench': '6', '1-squat': '8', '2-press': '' } },
  },
];

export function openGalleryState(id: string) {
  const st = galleryStates.find((x) => x.id === id);
  if (!st) return;
  setOverrides(st.data ?? {});
  if (router.canDismiss()) router.dismissAll();
  const [first, ...rest] = st.paths;
  router.navigate(first);
  for (const p of rest) router.push(p);
}
