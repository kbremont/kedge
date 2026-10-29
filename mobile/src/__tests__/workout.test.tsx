import { Stack } from 'expo-router';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import { Alert } from 'react-native';

import { unreachable } from '@/api/client';
import dashboardInProgress from '@/api/fixtures/dashboard-in-progress.json';
import dashboardNext from '@/api/fixtures/dashboard-next.json';
import workoutAmrap from '@/api/fixtures/workout-amrap.json';
import workoutCompleted from '@/api/fixtures/workout-completed.json';
import workoutCurrent from '@/api/fixtures/workout-current.json';
import workoutFinish from '@/api/fixtures/workout-finish.json';
import workoutRange from '@/api/fixtures/workout-range.json';
import { keys, makeQueryClient } from '@/api/queries';
import type { Dashboard, Workout } from '@/api/schema.gen';
import TodayScreen from '@/app/(tabs)/(today)/index';
import WorkoutScreen from '@/app/workout/[id]';
import { clearRest } from '@/lib/rest';
import { fakeServer, renderApp, type Routes } from '@/test-utils';

// The session screen against a stand-in server: what the server's
// workout_test pins for the web page (the rest timer's place, the rep
// picker), plus logging, undo, finish and discard through the API.

const current = workoutCurrent as Workout;
const id = current.id;

// The session once set `idx` is logged with `reps`, as the server answers.
function logged(w: Workout, idx: number, reps: number): Workout {
  const sets = w.sets.map((x) => (x.idx === idx ? { ...x, done: true, actualReps: reps } : x));
  return { ...w, sets, currentIdx: sets.find((x) => !x.done)?.idx ?? -1 };
}

const layout = () => <Stack />;

// Opens `w` straight in the session screen, from a seeded cache.
async function openSession(w: Workout, routes: Routes = {}) {
  const server = fakeServer({ [`GET /workouts/${w.id}`]: w, ...routes });
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.workout(w.id), w);
  await renderApp(
    { _layout: layout, 'workout/[id]': WorkoutScreen },
    { initialUrl: `/workout/${w.id}`, server, queryClient },
  );
  return server;
}

// Opens `w` the way the lifter does: Resume on Today.
async function resume(w: Workout, routes: Routes = {}) {
  const dashboard = dashboardInProgress as Dashboard;
  const server = fakeServer({
    'GET /dashboard': dashboard,
    'POST /workouts': w,
    [`GET /workouts/${w.id}`]: w,
    ...routes,
  });
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.dashboard(), dashboard);
  await renderApp({ _layout: layout, index: TodayScreen, 'workout/[id]': WorkoutScreen }, { server, queryClient });
  await fireEvent.press(screen.getByRole('button', { name: 'Resume session' }));
  await waitFor(() => expect(screen).toHavePathname(`/workout/${w.id}`));
  return server;
}

// Holds a request in flight until the test answers it.
function held<T extends object>() {
  let answer: (reply: T) => void = () => {};
  const reply = () => new Promise<T>((resolve) => (answer = resolve));
  return { reply, answer: (r: T) => act(async () => answer(r)) };
}

beforeEach(() => clearRest(id));

describe('rest timer', () => {
  test('a session opened fresh shows no rest', async () => {
    await openSession(current);
    expect(screen.getByLabelText('65 lb, 5 reps')).toBeOnTheScreen();
    expect(screen.queryByLabelText(/^Rest/)).not.toBeOnTheScreen();
  });

  test('a logged set starts the rest the server sent, right above the next set, and it counts down', async () => {
    // Not the fixture's 90 s, so the bar can only have it from the answer.
    const after = { ...logged(current, 3, 5), restSeconds: 120 };
    const server = await openSession(current, { [`PUT /workouts/${id}/sets/3`]: after });

    await fireEvent.press(screen.getByRole('button', { name: 'Log 5 reps' }));
    const bar = await screen.findByLabelText('Rest, 2:00 left');
    expect(server.sent).toContainEqual(
      expect.objectContaining({ method: 'PUT', path: `/workouts/${id}/sets/3`, body: { reps: 5 } }),
    );
    // The bar opens the new current set's block, over the set it precedes.
    const block = bar.parent!;
    expect(block.children[0]).toBe(bar);
    expect(block).toContainElement(screen.getByLabelText('75 lb, 5 reps'));

    await act(() => jest.advanceTimersByTime(30_000));
    expect(screen.getByLabelText('Rest, 1:30 left')).toBeOnTheScreen();
    await act(() => jest.advanceTimersByTime(90_000));
    expect(screen.queryByLabelText(/^Rest/)).not.toBeOnTheScreen();
  });

  test('once every set is logged there is no rest, and Finish shows instead', async () => {
    const finish = workoutFinish as Workout;
    const last = {
      ...finish,
      currentIdx: 10,
      sets: finish.sets.map((x) => (x.idx === 10 ? { ...x, done: false, actualReps: 0 } : x)),
    };
    await openSession(last, { [`PUT /workouts/${id}/sets/10`]: finish });

    await fireEvent.press(screen.getByRole('button', { name: 'Log 10 reps' }));
    expect(await screen.findByRole('button', { name: 'Finish session' })).toBeOnTheScreen();
    expect(screen.queryByLabelText(/^Rest/)).not.toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: 'Discard this session' })).not.toBeOnTheScreen();
  });
});

describe('rep picker', () => {
  const label = (reps: number, target: boolean) => `Log ${reps} reps${target ? ', meets the target' : ''}`;

  test.each([
    // The deload's 80% × 3–5 starts at a single rep, so a short set can be logged.
    { name: 'a 3–5 set', fixture: workoutRange, idx: 4, from: 1, targets: [3, 4, 5], pick: 4 },
    // An anchor PR set starts at its target, as the server offers it.
    { name: 'a PR set', fixture: workoutAmrap, idx: 5, from: 5, targets: [5], pick: 9 },
  ])('$name offers the server’s choices and logs the one picked', async ({ fixture, idx, from, targets, pick }) => {
    const w = fixture as Workout;
    const put = held<Workout>();
    const server = await openSession(w, { [`PUT /workouts/${w.id}/sets/${idx}`]: put.reply });

    // Exactly the choices, in order: the set asks for its reps rather than
    // offering one Log button.
    const choices = Array.from({ length: 10 }, (_, i) => from + i);
    const buttons = screen.getAllByRole('button', { name: /^Log \d+ reps/ });
    expect(buttons.map((b) => b.props.accessibilityLabel)).toEqual(
      choices.map((r) => label(r, targets.includes(r))),
    );

    await fireEvent.press(screen.getByRole('button', { name: label(pick, targets.includes(pick)) }));
    await waitFor(() =>
      expect(server.sent).toContainEqual(
        expect.objectContaining({ method: 'PUT', path: `/workouts/${w.id}/sets/${idx}`, body: { reps: pick } }),
      ),
    );
    // In flight, no second tap can log this set again or the one after.
    await waitFor(() => expect(buttons[0]).toBeDisabled());
    for (const b of screen.getAllByRole('button', { name: /^Log \d+ reps/ })) expect(b).toBeDisabled();

    await put.answer(logged(w, idx, pick));
    const next = w.sets[idx + 1];
    expect(await screen.findByLabelText(`${next.weight} lb, ${next.repTarget} reps`)).toBeOnTheScreen();
  });
});

test('a failed log says why beside the Log button and leaves the set to log', async () => {
  await openSession(current, {
    [`PUT /workouts/${id}/sets/3`]: () => Promise.reject(new TypeError('Network request failed')),
  });

  await fireEvent.press(screen.getByRole('button', { name: 'Log 5 reps' }));
  const message = await screen.findByText(unreachable);
  const log = screen.getByRole('button', { name: 'Log 5 reps' });
  // The Notice sits in the current set's panel, with the button it is about.
  expect(message.parent!.parent).toContainElement(log);
  expect(log).toBeEnabled();
  expect(screen.getByLabelText('65 lb, 5 reps')).toBeOnTheScreen();
  expect(screen.queryByLabelText(/^Rest/)).not.toBeOnTheScreen();
});

test('Undo takes back the set just logged, and its rest', async () => {
  const server = await openSession(current, {
    [`PUT /workouts/${id}/sets/3`]: logged(current, 3, 5),
    [`DELETE /workouts/${id}/sets/3`]: current,
  });
  await fireEvent.press(screen.getByRole('button', { name: 'Log 5 reps' }));
  await screen.findByLabelText('Rest, 1:30 left');

  await fireEvent.press(screen.getByRole('button', { name: 'Undo 65 × 5' }));
  await waitFor(() => expect(screen.queryByLabelText(/^Rest/)).not.toBeOnTheScreen());
  expect(server.sent.at(-1)).toMatchObject({ method: 'DELETE', path: `/workouts/${id}/sets/3` });
  expect(screen.getByLabelText('65 lb, 5 reps')).toBeOnTheScreen();
});

test('Finish ends the session and goes back to Today', async () => {
  const server = await resume(workoutFinish as Workout, { [`POST /workouts/${id}/finish`]: dashboardNext });

  await fireEvent.press(await screen.findByRole('button', { name: 'Finish session' }));
  await waitFor(() => expect(screen).toHavePathname('/'));
  expect(server.sent).toContainEqual(expect.objectContaining({ method: 'POST', path: `/workouts/${id}/finish` }));
  // Today shows what the server answered with.
  expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
});

test('Discard asks first, then removes the session and goes back', async () => {
  const alert = jest
    .spyOn(Alert, 'alert')
    .mockImplementation((_title, _message, buttons) => buttons?.find((b) => b.style === 'destructive')?.onPress?.());
  const server = await resume(current, { [`DELETE /workouts/${id}`]: dashboardNext });

  await fireEvent.press(await screen.findByRole('button', { name: 'Discard this session' }));
  expect(alert).toHaveBeenCalledWith('Discard this session?', expect.any(String), expect.any(Array));
  await waitFor(() => expect(screen).toHavePathname('/'));
  const deleted = server.sent.findIndex((x) => x.method === 'DELETE' && x.path === `/workouts/${id}`);
  expect(deleted).toBeGreaterThan(-1);
  // The session is gone, so it isn't asked for again.
  expect(server.sent.slice(deleted + 1).some((x) => x.path === `/workouts/${id}`)).toBe(false);
  alert.mockRestore();
});

test('a session opened from history reads as lifted, with the PR set as its bottom line', async () => {
  await openSession(workoutCompleted as Workout);
  expect(screen.getByText('85 × 9')).toBeOnTheScreen();
  expect(screen.queryByRole('button', { name: /^Log/ })).not.toBeOnTheScreen();
  expect(screen.queryByRole('button', { name: 'Discard this session' })).not.toBeOnTheScreen();
  expect(screen.queryByRole('button', { name: 'Finish session' })).not.toBeOnTheScreen();
});
