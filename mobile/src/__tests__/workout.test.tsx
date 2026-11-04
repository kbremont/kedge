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
import { repsText } from '@/lib/format';
import { clearRest } from '@/lib/rest';
import { fakeServer, json, renderApp, type Routes } from '@/test-utils';

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
async function openSession(w: Workout, routes: Routes = {}, queryClient = makeQueryClient()) {
  const server = fakeServer({ [`GET /workouts/${w.id}`]: w, ...routes });
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
  const label = (reps: number, target: boolean) =>
    `Log ${reps} ${reps === 1 ? 'rep' : 'reps'}${target ? ', meets the target' : ''}`;

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
    const buttons = screen.getAllByRole('button', { name: /^Log \d+ reps?\b/ });
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
    for (const b of screen.getAllByRole('button', { name: /^Log \d+ reps?\b/ })) expect(b).toBeDisabled();

    await put.answer(logged(w, idx, pick));
    const next = w.sets[idx + 1];
    expect(await screen.findByLabelText(`${next.weight} lb, ${repsText(next.repTarget)}`)).toBeOnTheScreen();
  });

  describe('the field beside the choices', () => {
    const amrap = workoutAmrap as Workout;
    const prIdx = 5;
    const field = () => screen.getByLabelText('Reps');
    const logTyped = () => screen.getByRole('button', { name: 'Log' });

    test.each([
      { name: 'a PR set', fixture: workoutAmrap, idx: prIdx },
      { name: 'a 3–5 set', fixture: workoutRange, idx: 4 },
    ])('$name logs a count past every choice', async ({ fixture, idx }) => {
      const w = fixture as Workout;
      const server = await openSession(w, { [`PUT /workouts/${w.id}/sets/${idx}`]: logged(w, idx, 15) });

      await fireEvent.changeText(field(), '15');
      await fireEvent.press(logTyped());

      await waitFor(() =>
        expect(server.sent).toContainEqual(
          expect.objectContaining({ method: 'PUT', path: `/workouts/${w.id}/sets/${idx}`, body: { reps: 15 } }),
        ),
      );
      const next = w.sets[idx + 1];
      expect(await screen.findByLabelText(`${next.weight} lb, ${repsText(next.repTarget)}`)).toBeOnTheScreen();
    });

    test('logs a count below the first choice, down to a failed set', async () => {
      const server = await openSession(amrap, { [`PUT /workouts/${amrap.id}/sets/${prIdx}`]: logged(amrap, prIdx, 0) });

      await fireEvent.changeText(field(), '0');
      await fireEvent.press(logTyped());

      await waitFor(() => expect(server.sent).toContainEqual(expect.objectContaining({ body: { reps: 0 } })));
    });

    test.each([
      { name: 'blank', text: '' },
      { name: 'only spaces', text: '  ' },
      { name: 'not a whole number', text: '1.5' },
    ])('won’t log $name', async ({ text }) => {
      await openSession(amrap);
      await fireEvent.changeText(field(), text);
      expect(logTyped()).toBeDisabled();
    });

    test('a count the server refuses says why and stays to be fixed', async () => {
      const refusal = 'Reps must be between 0 and 500.';
      await openSession(amrap, {
        [`PUT /workouts/${amrap.id}/sets/${prIdx}`]: () => json({ error: refusal }, 400),
      });

      await fireEvent.changeText(field(), '501');
      await fireEvent.press(logTyped());

      expect(await screen.findByText(refusal)).toBeOnTheScreen();
      expect(field()).toHaveProp('value', '501');
      expect(field()).toHaveStyle({ borderWidth: 2 });
      expect(logTyped()).toBeEnabled();

      // Being fixed, it's no longer the count that was refused.
      await fireEvent.changeText(field(), '50');
      expect(field()).not.toHaveStyle({ borderWidth: 2 });
    });

    test('a network failure says so without blaming the count', async () => {
      await openSession(amrap, {
        [`PUT /workouts/${amrap.id}/sets/${prIdx}`]: () => Promise.reject(new TypeError('Network request failed')),
      });

      await fireEvent.changeText(field(), '11');
      await fireEvent.press(logTyped());

      expect(await screen.findByText(unreachable)).toBeOnTheScreen();
      expect(field()).toHaveProp('value', '11');
      expect(field()).not.toHaveStyle({ borderWidth: 2 });
    });

    test('in flight, it can’t log the set again', async () => {
      const put = held<Workout>();
      await openSession(amrap, { [`PUT /workouts/${amrap.id}/sets/${prIdx}`]: put.reply });

      await fireEvent.changeText(field(), '11');
      await fireEvent.press(logTyped());

      await waitFor(() => expect(logTyped()).toBeDisabled());
      expect(field()).toHaveProp('editable', false);
      await put.answer(logged(amrap, prIdx, 11));
    });

    test('starts empty again once a set is undone', async () => {
      await openSession(amrap, {
        [`PUT /workouts/${amrap.id}/sets/${prIdx}`]: logged(amrap, prIdx, 11),
        [`DELETE /workouts/${amrap.id}/sets/${prIdx}`]: amrap,
      });

      await fireEvent.changeText(field(), '11');
      await fireEvent.press(logTyped());
      const set = amrap.sets[prIdx];
      await fireEvent.press(await screen.findByRole('button', { name: `Undo ${set.weight} × 11` }));

      await waitFor(() => expect(field()).toHaveProp('value', ''));
    });
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

test('a single is one rep, on the button and to VoiceOver', async () => {
  const single = {
    ...current,
    sets: current.sets.map((x) => (x.idx === current.currentIdx ? { ...x, targetReps: 1, repTarget: '1' } : x)),
  };
  await openSession(single);
  expect(screen.getByRole('button', { name: 'Log 1 rep' })).toBeOnTheScreen();
  expect(screen.getByLabelText('65 lb, 1 rep')).toBeOnTheScreen();
});

test('a session discarded somewhere else says so, and goes back', async () => {
  let discarded = false;
  await resume(current, {
    [`GET /workouts/${id}`]: () => (discarded ? json({ error: 'Not found.' }, 404) : current),
    [`PUT /workouts/${id}/sets/3`]: () => json({ error: 'Not found.' }, 404),
  });
  discarded = true;

  // Logging finds the session gone, and the screen asks again.
  await fireEvent.press(await screen.findByRole('button', { name: 'Log 5 reps' }));
  expect(await screen.findByText(/discarded somewhere else/)).toBeOnTheScreen();
  expect(screen.queryByRole('button', { name: 'Log 5 reps' })).not.toBeOnTheScreen();

  await fireEvent.press(screen.getByRole('button', { name: 'Back to Today' }));
  await waitFor(() => expect(screen).toHavePathname('/'));
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

test('once every set is logged, the last one can still be taken back', async () => {
  // A TM test ends on its PR set, the set most worth taking back.
  const amrap = workoutAmrap as Workout;
  const tmTest = { ...amrap, sets: amrap.sets.slice(0, 6) };
  const server = await openSession(logged(tmTest, 5, 9), { [`DELETE /workouts/${id}/sets/5`]: tmTest });

  // The row reads as it does over a current set, here above Finish.
  expect(screen.getByText('Logged 85 × 9')).toBeOnTheScreen();
  const undo = screen.getByRole('button', { name: 'Undo 85 × 9' });
  expect(undo.parent!.parent).toContainElement(screen.getByRole('button', { name: 'Finish session' }));

  await fireEvent.press(undo);
  expect(await screen.findByRole('button', { name: 'Log 9 reps' })).toBeOnTheScreen();
  expect(server.sent.at(-1)).toMatchObject({ method: 'DELETE', path: `/workouts/${id}/sets/5` });
  expect(screen.queryByRole('button', { name: 'Finish session' })).not.toBeOnTheScreen();
  expect(screen.getByRole('button', { name: 'Undo 75 × 5' })).toBeOnTheScreen();
});

describe('a failure is shown only beside the set it was about', () => {
  const unreached = () => Promise.reject(new TypeError('Network request failed'));

  test('a failed log, once the session has moved on', async () => {
    const queryClient = makeQueryClient();
    await openSession(current, { [`PUT /workouts/${id}/sets/3`]: unreached }, queryClient);
    await fireEvent.press(screen.getByRole('button', { name: 'Log 5 reps' }));
    expect(await screen.findByText(unreachable)).toBeOnTheScreen();

    // The set was logged after all, as a refetch shows.
    await act(async () => queryClient.setQueryData(keys.workout(id), logged(current, 3, 5)));
    expect(await screen.findByLabelText('75 lb, 5 reps')).toBeOnTheScreen();
    expect(screen.queryByText(unreachable)).not.toBeOnTheScreen();
  });

  test('a failed undo, once another set is the last logged', async () => {
    const queryClient = makeQueryClient();
    const after3 = logged(current, 3, 5);
    await openSession(after3, { [`DELETE /workouts/${id}/sets/3`]: unreached }, queryClient);
    await fireEvent.press(screen.getByRole('button', { name: 'Undo 65 × 5' }));
    expect(await screen.findByText(unreachable)).toBeOnTheScreen();

    await act(async () => queryClient.setQueryData(keys.workout(id), logged(after3, 4, 5)));
    expect(await screen.findByRole('button', { name: 'Undo 75 × 5' })).toBeOnTheScreen();
    expect(screen.queryByText(unreachable)).not.toBeOnTheScreen();
  });
});

test('a second tap on Log before the screen redraws sends nothing more', async () => {
  const put = held<Workout>();
  const server = await openSession(current, { [`PUT /workouts/${id}/sets/3`]: put.reply });
  const log = screen.getByRole('button', { name: 'Log 5 reps' });
  await fireEvent.press(log);
  await fireEvent.press(log);

  await put.answer(logged(current, 3, 5));
  expect(await screen.findByLabelText('75 lb, 5 reps')).toBeOnTheScreen();
  expect(server.sent.filter((x) => x.method === 'PUT')).toHaveLength(1);
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
