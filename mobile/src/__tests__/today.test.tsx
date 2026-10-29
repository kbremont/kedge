import { Stack } from 'expo-router';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import { Alert } from 'react-native';

import { unreachable } from '@/api/client';
import dashboardBlockReview from '@/api/fixtures/dashboard-block-review.json';
import dashboardInProgress from '@/api/fixtures/dashboard-in-progress.json';
import dashboardNext from '@/api/fixtures/dashboard-next.json';
import dashboardSkipDeload from '@/api/fixtures/dashboard-skip-deload.json';
import dashboardTMTestFailed from '@/api/fixtures/dashboard-tm-test-failed.json';
import dashboardTMTestPassed from '@/api/fixtures/dashboard-tm-test-passed.json';
import workoutCurrent from '@/api/fixtures/workout-current.json';
import { keys, makeQueryClient } from '@/api/queries';
import type { Dashboard, Workout } from '@/api/schema.gen';
import TodayScreen from '@/app/(tabs)/(today)/index';
import WorkoutScreen from '@/app/workout/[id]';
import { fakeServer, json, renderApp, type Routes } from '@/test-utils';

// Today against a stand-in server: what the server's pages_test pins for
// the web dashboard (the TM test cards, the block review, skip), plus
// starting a session and the screen's first load.

const next = dashboardNext as Dashboard;
const failed = dashboardTMTestFailed as Dashboard;
const passed = dashboardTMTestPassed as Dashboard;
const review = dashboardBlockReview as Dashboard;
const skipDeload = dashboardSkipDeload as Dashboard;
const current = workoutCurrent as Workout;

const routes = { _layout: () => <Stack />, index: TodayScreen, 'workout/[id]': WorkoutScreen };

// Today as the server has it: a change answers with it, and a refetch reads it.
let today: Dashboard;
const becomes = (d: Dashboard) => () => (today = d);

// Opens Today on `d`, seeded as it is after signing in.
async function openToday(d: Dashboard, more: Routes = {}) {
  today = d;
  const server = fakeServer({ 'GET /dashboard': () => today, ...more });
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.dashboard(), d);
  await renderApp(routes, { server, queryClient });
  return { server, queryClient };
}

// What `server` was sent as `route`, such as 'POST /tm-review'.
function sentTo(server: ReturnType<typeof fakeServer>, route: string) {
  return server.sent.filter((x) => `${x.method} ${x.path}` === route);
}

// Holds a request in flight until the test answers it.
function held<T extends object>() {
  let answer: (reply: T) => void = () => {};
  const reply = () => new Promise<T>((resolve) => (answer = resolve));
  return { reply, answer: (r: T) => act(async () => answer(r)) };
}

// Confirms the next native alert with its destructive button.
function confirmAlerts() {
  return jest
    .spyOn(Alert, 'alert')
    .mockImplementation((_title, _message, buttons) => buttons?.find((b) => b.style === 'destructive')?.onPress?.());
}

afterEach(() => jest.restoreAllMocks());

describe('starting a session', () => {
  test.each([
    { name: 'Start', d: next, label: 'Start session' },
    { name: 'Resume', d: dashboardInProgress as Dashboard, label: 'Resume session' },
  ])('$name asks the server for the session and opens it', async ({ d, label }) => {
    const { server } = await openToday(d, { 'POST /workouts': current, [`GET /workouts/${current.id}`]: current });

    await fireEvent.press(screen.getByRole('button', { name: label }));
    await waitFor(() => expect(screen).toHavePathname(`/workout/${current.id}`));
    expect(sentTo(server, 'POST /workouts')).toHaveLength(1);
    expect(await screen.findByLabelText('65 lb, 5 reps')).toBeOnTheScreen();
  });

  test('a second tap before the button redraws asks for one session, and opens it', async () => {
    const post = held<Workout>();
    const { server } = await openToday(next, {
      'POST /workouts': post.reply,
      [`GET /workouts/${current.id}`]: current,
    });
    const start = screen.getByRole('button', { name: 'Start session' });
    await fireEvent.press(start);
    await fireEvent.press(start);

    await post.answer(current);
    await waitFor(() => expect(screen).toHavePathname(`/workout/${current.id}`));
    expect(sentTo(server, 'POST /workouts')).toHaveLength(1);
  });

  test('a refusal says why, and Today refetches to show the hold until it is settled', async () => {
    const { server } = await openToday(next, {
      'POST /workouts': () => {
        today = failed;
        return json({ error: 'Training is waiting on a training max decision.' }, 409);
      },
      'POST /tm-tests/4': becomes(next),
    });

    await fireEvent.press(screen.getByRole('button', { name: 'Start session' }));
    expect(await screen.findByText('Training is waiting on a training max decision.')).toBeOnTheScreen();
    expect(await screen.findByText('Settle your training maxes above to start.')).toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: 'Start session' })).not.toBeOnTheScreen();
    expect(screen).toHavePathname('/');

    await fireEvent.press(screen.getByRole('button', { name: 'Keep 250 lb' }));
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(screen.queryByText('Training is waiting on a training max decision.')).not.toBeOnTheScreen();
    expect(sentTo(server, 'POST /tm-tests/4')).toHaveLength(1);
  });
});

describe('TM test cards', () => {
  test('a pass is news, with nothing to fill in', async () => {
    await openToday(passed);
    expect(screen.getByText('Overhead Press TM test')).toBeOnTheScreen();
    expect(screen.getByText(passed.testResults[0].headline)).toBeOnTheScreen();
    expect(screen.queryByLabelText(/^New .* training max/)).not.toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: /^(Lower|Keep)/ })).not.toBeOnTheScreen();
    expect(screen.getByRole('button', { name: 'Start session' })).toBeOnTheScreen();
  });

  test('a failure is prefilled with the proposal, and holds the next session', async () => {
    await openToday(failed);
    expect(screen.getByLabelText('New Squat training max, lb')).toHaveDisplayValue('225');
    expect(screen.getByRole('button', { name: 'Lower to 225 lb' })).toBeEnabled();
    expect(screen.getByText('Settle your training maxes above to start.')).toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: 'Start session' })).not.toBeOnTheScreen();
  });

  test('Lower sends the edited training max, and the answer replaces the card', async () => {
    const decision = held<Dashboard>();
    const { server } = await openToday(failed, { 'POST /tm-tests/4': decision.reply });

    await fireEvent.changeText(screen.getByLabelText('New Squat training max, lb'), '230');
    await fireEvent.press(screen.getByRole('button', { name: 'Lower to 230 lb' }));
    await waitFor(() => expect(sentTo(server, 'POST /tm-tests/4')).toHaveLength(1));
    expect(sentTo(server, 'POST /tm-tests/4')[0].body).toEqual({ action: 'lower', tm: 230 });
    // Neither choice can be sent twice while the first is on its way.
    await waitFor(() => expect(screen.getByRole('button', { name: 'Lower to 230 lb' })).toBeDisabled());
    expect(screen.getByRole('button', { name: 'Keep 250 lb' })).toBeDisabled();

    today = next;
    await decision.answer(next);
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(screen.queryByText('Squat TM test')).not.toBeOnTheScreen();
  });

  test('Keep sends only the choice', async () => {
    const { server } = await openToday(failed, { 'POST /tm-tests/4': becomes(next) });
    await fireEvent.press(screen.getByRole('button', { name: 'Keep 250 lb' }));
    await waitFor(() => expect(sentTo(server, 'POST /tm-tests/4')).toHaveLength(1));
    expect(sentTo(server, 'POST /tm-tests/4')[0].body).toEqual({ action: 'keep' });
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
  });

  test('a decision that fails says why beside the buttons, and the card stays', async () => {
    const refused = json({ error: 'Something went wrong. Check the server log.' }, 500);
    await openToday(failed, { 'POST /tm-tests/4': refused });
    await fireEvent.press(screen.getByRole('button', { name: 'Lower to 225 lb' }));

    const message = await screen.findByText('Something went wrong. Check the server log.');
    const lower = screen.getByRole('button', { name: 'Lower to 225 lb' });
    // The Notice sits in the TM test card, with the choices it is about.
    expect(message.parent!.parent).toContainElement(lower);
    expect(lower).toBeEnabled();
    expect(screen.getByText('Settle your training maxes above to start.')).toBeOnTheScreen();
  });
});

describe('block review', () => {
  test('Accept sends the key and every training max as edited', async () => {
    const { server } = await openToday(review, { 'POST /tm-review': becomes(next) });
    expect(screen.getByText('Settle your training maxes above to start.')).toBeOnTheScreen();

    await fireEvent.changeText(screen.getByLabelText('New Squat training max, lb'), '255');
    await fireEvent.press(screen.getByRole('button', { name: 'Accept training maxes' }));
    await waitFor(() => expect(sentTo(server, 'POST /tm-review')).toHaveLength(1));
    expect(sentTo(server, 'POST /tm-review')[0].body).toEqual({
      key: 'b2:c0',
      tms: { press: 95, deadlift: 270, bench: 210, squat: 255 },
    });
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(screen.queryByText(review.review!.title)).not.toBeOnTheScreen();
  });

  test('a new review starts from its own proposals', async () => {
    const { queryClient } = await openToday(review);
    await fireEvent.changeText(screen.getByLabelText('New Squat training max, lb'), '255');

    const rows = review.review!.rows.map((r) => (r.slug === 'squat' ? { ...r, proposed: 265 } : r));
    await act(async () =>
      queryClient.setQueryData(keys.dashboard(), { ...review, review: { ...review.review!, key: 'b3:c0', rows } }),
    );
    expect(await screen.findByDisplayValue('265')).toBeOnTheScreen();
    expect(screen.queryByDisplayValue('255')).not.toBeOnTheScreen();
  });
});

describe('skip', () => {
  test('is offered only when the cycle can be skipped', async () => {
    await openToday(next);
    expect(screen.getByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: 'Skip it and start the next block' })).not.toBeOnTheScreen();
  });

  test('asks first, then skips the cycle', async () => {
    const alert = confirmAlerts();
    const { server } = await openToday(skipDeload, { 'POST /skip-cycle': becomes(next) });

    expect(screen.getByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    await fireEvent.press(screen.getByRole('button', { name: 'Skip it and start the next block' }));
    expect(alert).toHaveBeenCalledWith('Skip the 7th Week Deload?', 'The next block starts now.', expect.any(Array));
    await waitFor(() => expect(sentTo(server, 'POST /skip-cycle')).toHaveLength(1));
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: 'Skip it and start the next block' })).not.toBeOnTheScreen(),
    );
  });
});

describe('first load', () => {
  test('a spinner appears only after a moment, then the screen', async () => {
    const load = held<Dashboard>();
    await renderApp(routes, { server: fakeServer({ 'GET /dashboard': load.reply }) });

    await act(() => jest.advanceTimersByTime(399));
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
    await act(() => jest.advanceTimersByTime(1));
    expect(screen.getByLabelText('Loading')).toBeOnTheScreen();

    await load.answer(next);
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
  });

  test('a failed first load says why, and Try again asks again', async () => {
    let up = false;
    const retry = held<Dashboard>();
    const server = fakeServer({
      'GET /dashboard': () => (up ? retry.reply() : Promise.reject(new TypeError('Network request failed'))),
    });
    await renderApp(routes, { server });

    // A server out of reach is asked once more before the screen gives up.
    expect(await screen.findByText(unreachable, {}, { timeout: 5_000 })).toBeOnTheScreen();
    expect(sentTo(server, 'GET /dashboard')).toHaveLength(2);

    up = true;
    await fireEvent.press(screen.getByRole('button', { name: 'Try again' }));
    // The message stays while the screen asks again.
    expect(await screen.findByRole('button', { name: 'Trying again…' })).toBeDisabled();
    expect(screen.getByText(unreachable)).toBeOnTheScreen();
    expect(sentTo(server, 'GET /dashboard')).toHaveLength(3);

    await retry.answer(next);
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(screen.queryByText(unreachable)).not.toBeOnTheScreen();
  });

  test('a refetch that fails leaves the screen as it was', async () => {
    const { queryClient } = await openToday(next, {
      'GET /dashboard': json({ error: 'Something went wrong. Check the server log.' }, 500),
    });
    await waitFor(() => expect(queryClient.getQueryState(keys.dashboard())?.status).toBe('error'));
    expect(screen.getByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(screen.queryByText('Something went wrong. Check the server log.')).not.toBeOnTheScreen();
  });
});
