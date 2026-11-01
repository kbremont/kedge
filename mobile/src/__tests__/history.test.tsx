import { Stack } from 'expo-router';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import { Dimensions } from 'react-native';

import { unreachable } from '@/api/client';
import historyFixture from '@/api/fixtures/history.json';
import historyEmpty from '@/api/fixtures/history-empty.json';
import liftEmpty from '@/api/fixtures/lift-empty.json';
import workoutCompleted from '@/api/fixtures/workout-completed.json';
import { keys, makeQueryClient } from '@/api/queries';
import type { History, Lift, Workout } from '@/api/schema.gen';
import HistoryScreen from '@/app/(tabs)/(history)/history';
import LiftScreen from '@/app/lift/[slug]/index';
import WorkoutScreen from '@/app/workout/[id]';
import { fakeServer, json, renderApp, type Routes } from '@/test-utils';

// History against a stand-in server: the career read from GET /history, its
// empty state, the ways out to a session and a lift, and its first load.

const history = historyFixture as History;
const completed = workoutCompleted as Workout;
// A lift with no past estimates: their swipeable rows need Reanimated, which
// lift.test.tsx stands in for.
const bench = { ...(liftEmpty as Lift), slug: 'bench', name: 'Bench Press' };

const routes = {
  _layout: () => <Stack />,
  history: HistoryScreen,
  'workout/[id]': WorkoutScreen,
  'lift/[slug]/index': LiftScreen,
};

// The text size the screen lays out for. Jest's stand-in for the device
// reports 2x, where History stacks each round into a list.
function atTextSize(fontScale: number) {
  Dimensions.set({
    window: { ...Dimensions.get('window'), fontScale },
    screen: { ...Dimensions.get('screen'), fontScale },
  });
}

// Opens History on `h`, seeded as if it had been read before.
async function openHistory(h: History, more: Routes = {}) {
  const server = fakeServer({ 'GET /history': h, ...more });
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.history(), h);
  await renderApp(routes, { initialUrl: '/history', server, queryClient });
  return { server, queryClient };
}

// Holds a request in flight until the test answers it.
function held<T extends object>() {
  let answer: (reply: T) => void = () => {};
  const reply = () => new Promise<T>((resolve) => (answer = resolve));
  return { reply, answer: (r: T) => act(async () => answer(r)) };
}

beforeEach(() => atTextSize(1));
afterAll(() => atTextSize(2));

describe('the career', () => {
  test('reads one line per round, newest first, each session in its lift’s column', async () => {
    await openHistory(history);
    for (const l of history.lifts) {
      expect(screen.getByRole('button', { name: `${l.name}: chart and past estimates` })).toBeOnTheScreen();
    }
    const cells = screen.getAllByRole('button', { name: /^(Overhead Press|Deadlift|Bench Press|Squat), / });
    // Read across each line in the heads' order; the anchor has no squat yet.
    expect(cells.map((c) => c.props.accessibilityLabel)).toEqual([
      'Overhead Press, Anchor · Round 1, Jan 15, 85 × 5',
      'Deadlift, Anchor · Round 1, Jan 17, 255 × 5',
      'Bench Press, Anchor · Round 1, Jan 19, 170 × 5',
      'Overhead Press, 7th Week TM Test, Jan 7, 100 × 6',
      'Deadlift, 7th Week TM Test, Jan 9, 300 × 4',
      'Bench Press, 7th Week TM Test, Jan 11, 200 × 5',
      'Squat, 7th Week TM Test, Jan 13, 250 × 4',
    ]);
    expect(screen.getByLabelText('Anchor · Round 1')).toBeOnTheScreen();
    expect(screen.getByLabelText('7th Week TM Test')).toBeOnTheScreen();
    expect(screen.queryByText(/^No sessions yet/)).not.toBeOnTheScreen();
  });

  test('at accessibility text sizes each round is a list of its sessions', async () => {
    atTextSize(2);
    await openHistory(history);
    expect(screen.getByText('Anchor · Round 1')).toBeOnTheScreen();
    expect(screen.getByText('7th Week TM Test')).toBeOnTheScreen();
    expect(screen.getByText('100 × 6')).toBeOnTheScreen();
    expect(screen.getByRole('button', { name: 'Squat: chart and past estimates' })).toBeOnTheScreen();
  });

  test('with no sessions yet it says so, under the lifts’ heads', async () => {
    await openHistory(historyEmpty as History);
    expect(screen.getByText('No sessions yet. Each round you finish adds a line here.')).toBeOnTheScreen();
    expect(screen.getByRole('button', { name: 'Squat: chart and past estimates' })).toBeOnTheScreen();
    expect(screen.queryAllByRole('button', { name: /^(Overhead Press|Deadlift|Bench Press|Squat), / })).toHaveLength(0);
  });

  test('a session opens as it was lifted, with History as the way back', async () => {
    const { server } = await openHistory(history, { [`GET /workouts/${completed.id}`]: completed });
    await fireEvent.press(screen.getByRole('button', { name: 'Overhead Press, 7th Week TM Test, Jan 7, 100 × 6' }));

    await waitFor(() => expect(screen).toHavePathname(`/workout/${completed.id}`));
    expect(screen).toHaveSearchParams({ id: String(completed.id), back: 'History' });
    expect(await screen.findByText('85 × 9')).toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: /^Log/ })).not.toBeOnTheScreen();
    expect(server.sent).toContainEqual(expect.objectContaining({ method: 'GET', path: `/workouts/${completed.id}` }));
  });

  test('a lift’s head opens its page, with History as the way back', async () => {
    await openHistory(history, { 'GET /lifts/bench': bench });
    await fireEvent.press(screen.getByRole('button', { name: 'Bench Press: chart and past estimates' }));

    await waitFor(() => expect(screen).toHavePathname('/lift/bench'));
    expect(screen).toHaveSearchParams({ slug: 'bench', back: 'History' });
    expect(await screen.findByText('None yet.')).toBeOnTheScreen();
  });
});

describe('first load', () => {
  test('a spinner appears only after a moment, then the career', async () => {
    const load = held<History>();
    await renderApp(routes, { initialUrl: '/history', server: fakeServer({ 'GET /history': load.reply }) });

    await act(() => jest.advanceTimersByTime(399));
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
    await act(() => jest.advanceTimersByTime(1));
    expect(screen.getByLabelText('Loading')).toBeOnTheScreen();

    await load.answer(history);
    expect(await screen.findByLabelText('Anchor · Round 1')).toBeOnTheScreen();
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
  });

  test('a failed first load says why, and Try again asks again', async () => {
    let up = false;
    const server = fakeServer({
      'GET /history': () => (up ? history : Promise.reject(new TypeError('Network request failed'))),
    });
    await renderApp(routes, { initialUrl: '/history', server });

    expect(await screen.findByText(unreachable, {}, { timeout: 5_000 })).toBeOnTheScreen();
    up = true;
    await fireEvent.press(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByLabelText('Anchor · Round 1')).toBeOnTheScreen();
    expect(screen.queryByText(unreachable)).not.toBeOnTheScreen();
  });

  test('a refetch that fails leaves the career as it was', async () => {
    const { queryClient } = await openHistory(history, {
      'GET /history': json({ error: 'Something went wrong. Check the server log.' }, 500),
    });
    await waitFor(() => expect(queryClient.getQueryState(keys.history())?.status).toBe('error'));
    expect(screen.getByLabelText('Anchor · Round 1')).toBeOnTheScreen();
    expect(screen.queryByText('Something went wrong. Check the server log.')).not.toBeOnTheScreen();
  });
});
