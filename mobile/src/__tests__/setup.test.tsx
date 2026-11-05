import { router, Stack } from 'expo-router';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import { Keyboard } from 'react-native';

import dashboardNext from '@/api/fixtures/dashboard-next.json';
import dashboardSetup from '@/api/fixtures/dashboard-setup.json';
import setupHistoryAnchor from '@/api/fixtures/setup-history-anchor.json';
import setupHistoryLeader from '@/api/fixtures/setup-history-leader.json';
import setupHistoryTMTest from '@/api/fixtures/setup-history-tm-test.json';
import setupPosition from '@/api/fixtures/setup-position.json';
import setupLocked from '@/api/fixtures/setup-locked.json';
import setupFixture from '@/api/fixtures/setup.json';
import { keys, makeQueryClient } from '@/api/queries';
import type { Dashboard, Setup, SetupHistory, SetupPosition } from '@/api/schema.gen';
import TodayScreen from '@/app/(tabs)/(today)/index';
import SetupScreen from '@/app/setup';
import SetupHistoryScreen from '@/app/setup/history';
import SetupPositionScreen from '@/app/setup/position';
import { fakeServer, json, renderApp, type Routes } from '@/test-utils';

// First-run setup against a stand-in server, over Today as a first sign-in
// opens it: the training maxes, where in the programme the lifter is, and
// the sessions of that cycle already done.

const setup = setupFixture as Setup;
const position = setupPosition as SetupPosition;
const leader = setupHistoryLeader as SetupHistory;
const anchor = setupHistoryAnchor as SetupHistory;
const tmTest = setupHistoryTMTest as SetupHistory;
const needsSetup = dashboardSetup as Dashboard;
const next = dashboardNext as Dashboard;

const routes = {
  _layout: () => <Stack />,
  index: TodayScreen,
  'setup/index': SetupScreen,
  'setup/position': SetupPositionScreen,
  'setup/history': SetupHistoryScreen,
};

// Today as the server has it, for a refetch to read.
let today: Dashboard;

type Server = ReturnType<typeof fakeServer>;

// Opens Today before setup, as it is after a first sign-in.
async function openToday(more: Routes = {}) {
  today = needsSetup;
  const server = fakeServer({
    'GET /dashboard': () => today,
    'GET /setup': setup,
    'GET /setup/position': position,
    ...more,
  });
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.dashboard(), needsSetup);
  const view = await renderApp(routes, { server, queryClient });
  return { server, queryClient, view };
}

// Opens the training maxes from Today's Set up button.
async function openSetup(more: Routes = {}) {
  const opened = await openToday(more);
  await fireEvent.press(screen.getByRole('button', { name: 'Set up' }));
  await screen.findByLabelText('Squat training max, lb');
  return opened;
}

// Fills in the training maxes, by lift name.
async function enter(tms: Record<string, string>) {
  for (const [name, value] of Object.entries(tms)) {
    await fireEvent.changeText(screen.getByLabelText(`${name} training max, lb`), value);
  }
}

// Picks `h`'s cycle on the position screen, over Today, and goes on to its
// checklist.
async function openChecklist(h: SetupHistory, more: Routes = {}) {
  const opened = await openToday({ [`GET /setup/history/${h.cycleIndex}`]: h, ...more });
  await act(() => router.push('/setup/position'));
  await fireEvent.press(await screen.findByRole('radio', { name: new RegExp(`^${h.cycleLabel}`) }));
  await fireEvent.press(screen.getByRole('button', { name: 'Continue' }));
  await screen.findByRole('checkbox', { name: /^Overhead Press, round 1/ });
  return opened;
}

// The checkbox for one session on the checklist.
const session = (name: string, round: number) =>
  screen.getByRole('checkbox', { name: new RegExp(`^${name}, round ${round},`) });

// What `server` was sent as `route`, such as 'PUT /setup'.
function sentTo(server: Server, route: string) {
  return server.sent.filter((x) => `${x.method} ${x.path}` === route);
}

// The screens on the stack, bottom first.
const stack = (view: Awaited<ReturnType<typeof renderApp>>) =>
  view.getRouterState()?.routes[0].state?.routes.map((r) => r.name) ?? [];

// Every request but a read.
const writes = (server: Server) => server.sent.filter((x) => x.method !== 'GET');

// Holds a request in flight until the test answers it.
function held<T extends object>() {
  let answer: (reply: T) => void = () => {};
  const reply = () => new Promise<T>((resolve) => (answer = resolve));
  return { reply, answer: (r: T) => act(async () => answer(r)) };
}

afterEach(() => jest.restoreAllMocks());

describe('training maxes', () => {
  test('Continue sends every training max as a number, then asks where you are', async () => {
    const { server } = await openSetup({ 'PUT /setup': position });
    expect(screen.getByLabelText('Squat training max, lb')).toHaveDisplayValue('');
    await enter({ 'Overhead Press': '115', Deadlift: '365', 'Bench Press': '205', Squat: '302.5' });
    const dismiss = jest.spyOn(Keyboard, 'dismiss');
    await fireEvent.press(screen.getByRole('button', { name: 'Continue' }));
    // So the keyboard isn't still leaving over the next screen.
    expect(dismiss).toHaveBeenCalled();

    await waitFor(() => expect(screen).toHavePathname('/setup/position'));
    expect(sentTo(server, 'PUT /setup')).toHaveLength(1);
    expect(sentTo(server, 'PUT /setup')[0].body).toEqual({
      tms: { press: 115, deadlift: 365, bench: 205, squat: 302.5 },
      unit: 'lb',
    });
    expect(await screen.findByRole('radio', { name: /^Anchor/ })).toBeOnTheScreen();
  });

  test('choosing kilograms clears what was typed, labels the fields in kg, and sends the unit', async () => {
    const { server } = await openSetup({ 'PUT /setup': position });
    await enter({ 'Overhead Press': '115' });
    await fireEvent(screen.getByTestId('Picker'), 'selectionChange', 'kg');

    expect(screen.queryByLabelText('Squat training max, lb')).not.toBeOnTheScreen();
    expect(screen.getByLabelText('Overhead Press training max, kg')).toHaveDisplayValue('');
    const kg = { 'Overhead Press': '52,5', Deadlift: '170', 'Bench Press': '87.5', Squat: '142.5' };
    for (const [name, value] of Object.entries(kg)) {
      await fireEvent.changeText(screen.getByLabelText(`${name} training max, kg`), value);
    }
    await fireEvent.press(screen.getByRole('button', { name: 'Continue' }));

    await waitFor(() => expect(screen).toHavePathname('/setup/position'));
    // A decimal comma, as a phone in much of the world types it, is a point.
    expect(sentTo(server, 'PUT /setup')[0].body).toEqual({
      tms: { press: 52.5, deadlift: 170, bench: 87.5, squat: 142.5 },
      unit: 'kg',
    });
  });

  test.each([
    ['once training is logged', setupLocked as Setup],
    ['on a server from before kilograms', { lifts: setup.lifts, unit: 'lb' } as Setup],
  ])('%s there is no unit to choose, and none is sent', async (_, answer) => {
    const { server } = await openSetup({ 'GET /setup': answer, 'PUT /setup': position });
    expect(screen.queryByTestId('Picker')).not.toBeOnTheScreen();
    await enter({ 'Overhead Press': '115', Deadlift: '365', 'Bench Press': '205', Squat: '302.5' });
    await fireEvent.press(screen.getByRole('button', { name: 'Continue' }));
    await waitFor(() => expect(sentTo(server, 'PUT /setup')).toHaveLength(1));
    expect(sentTo(server, 'PUT /setup')[0].body).not.toHaveProperty('unit');
  });

  test('running setup again starts from the training maxes saved', async () => {
    const saved = setup.lifts.map((l, i) => ({ ...l, value: [115, 365, 205, 302.5][i] }));
    await openSetup({ 'GET /setup': { ...setup, lifts: saved } });
    expect(screen.getByLabelText('Overhead Press training max, lb')).toHaveDisplayValue('115');
    expect(screen.getByLabelText('Squat training max, lb')).toHaveDisplayValue('302.5');
  });

  test('a refusal shows the server’s message beside the field it is about, and keeps what was typed', async () => {
    const message = 'Enter a positive training max for Squat.';
    const { server } = await openSetup({ 'PUT /setup': json({ error: message }, 400) });
    await enter({ 'Overhead Press': '115', Deadlift: '365', 'Bench Press': '205' });
    await fireEvent.press(screen.getByRole('button', { name: 'Continue' }));

    expect(await screen.findByText(message)).toBeOnTheScreen();
    // The server is the one that says no: an empty field goes as 0.
    expect(sentTo(server, 'PUT /setup')[0].body).toEqual({
      tms: { press: 115, deadlift: 365, bench: 205, squat: 0 },
      unit: 'lb',
    });
    expect(screen).toHavePathname('/setup');
    expect(screen.getByLabelText('Squat training max, lb')).toHaveStyle({ borderWidth: 2 });
    expect(screen.getByLabelText('Deadlift training max, lb')).not.toHaveStyle({ borderWidth: 2 });
    expect(screen.getByLabelText('Overhead Press training max, lb')).toHaveDisplayValue('115');
    expect(screen.getByLabelText('Deadlift training max, lb')).toHaveDisplayValue('365');
    expect(screen.getByLabelText('Bench Press training max, lb')).toHaveDisplayValue('205');
    expect(screen.getByRole('button', { name: 'Continue' })).toBeEnabled();

    // Until the lifter changes what they typed.
    await enter({ Squat: '305' });
    expect(screen.queryByText(message)).not.toBeOnTheScreen();
    expect(screen.getByLabelText('Squat training max, lb')).not.toHaveStyle({ borderWidth: 2 });
  });

  test('a second tap before the button redraws saves once, and goes on once', async () => {
    const put = held<SetupPosition>();
    const { server } = await openSetup({ 'PUT /setup': put.reply });
    await enter({ 'Overhead Press': '115', Deadlift: '365', 'Bench Press': '205', Squat: '305' });
    const go = screen.getByRole('button', { name: 'Continue' });
    await fireEvent.press(go);
    await fireEvent.press(go);
    await waitFor(() => expect(screen.getByRole('button', { name: 'Continue' })).toBeDisabled());

    await put.answer(position);
    await waitFor(() => expect(screen).toHavePathname('/setup/position'));
    expect(sentTo(server, 'PUT /setup')).toHaveLength(1);
  });
});

describe('where you are', () => {
  test('starting fresh sends nothing, and Today asks again for its first session', async () => {
    const { server, queryClient, view } = await openSetup({ 'PUT /setup': position });
    await enter({ 'Overhead Press': '115', Deadlift: '365', 'Bench Press': '205', Squat: '305' });
    await fireEvent.press(screen.getByRole('button', { name: 'Continue' }));
    await screen.findByRole('radio', { name: /^Starting fresh/ });
    await waitFor(() => expect(queryClient.isFetching()).toBe(0));
    const sentBefore = server.sent.length;

    // Only a fresh read of Today can show the session.
    today = next;
    await fireEvent.press(screen.getByRole('button', { name: 'Start training' }));
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    // Setup was over Today, and is gone.
    expect(stack(view)).toEqual(['index']);
    expect(writes(server)).toHaveLength(1); // PUT /setup, before it
    expect(server.sent.slice(sentBefore).map((x) => `${x.method} ${x.path}`)).toEqual(['GET /dashboard']);
  });

  test.each([
    { name: 'a leader', h: leader },
    { name: 'the TM test', h: tmTest },
    { name: 'the anchor', h: anchor },
  ])('picking $name opens its checklist', async ({ h }) => {
    const { server } = await openChecklist(h);
    expect(screen).toHavePathname('/setup/history');
    expect(screen).toHaveSearchParams({ cycle: String(h.cycleIndex) });
    expect(sentTo(server, `GET /setup/history/${h.cycleIndex}`)).toHaveLength(1);
    const top = h.rounds[0].days[0].topWeight;
    expect(session('Overhead Press', 1)).toHaveProp('accessibilityLabel', `Overhead Press, round 1, top set ${top} lb`);
    // A cycle of one round has no round headings.
    if (h.rounds.length > 1) expect(screen.getByText('Round 2')).toBeOnTheScreen();
    else expect(screen.queryByText('Round 1')).not.toBeOnTheScreen();
  });
});

test('a checklist is cached by its cycle’s number, as the API has it', async () => {
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.setupHistory(3), anchor);
  const server = fakeServer({ 'GET /setup/history/3': held<SetupHistory>().reply });
  await renderApp(routes, { initialUrl: '/setup/history?cycle=3', server, queryClient });
  expect(session('Overhead Press', 1)).toBeOnTheScreen();
});

describe('sessions already done', () => {
  test('Save sends the ticked sessions, then lands on the Today it answered with', async () => {
    const { server, view } = await openChecklist(leader, { 'PUT /setup/history/0': next });
    await fireEvent.press(session('Deadlift', 1));
    await fireEvent.press(session('Overhead Press', 1));
    await fireEvent.press(session('Bench Press', 2));
    await fireEvent.press(session('Bench Press', 2));
    // A leader doesn't ask for reps.
    expect(screen.queryByLabelText(/reps$/)).not.toBeOnTheScreen();

    const dismiss = jest.spyOn(Keyboard, 'dismiss');
    await fireEvent.press(screen.getByRole('button', { name: 'Save and start' }));
    expect(dismiss).toHaveBeenCalled();
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(stack(view)).toEqual(['index']);
    // In the checklist's order, as the server reads it.
    expect(sentTo(server, 'PUT /setup/history/0')[0].body).toEqual({
      done: [
        { round: 1, slug: 'press' },
        { round: 1, slug: 'deadlift' },
      ],
    });
    // Today is the answer: the server still has it before setup.
    expect(sentTo(server, 'GET /dashboard')).toHaveLength(1);
  });

  test.each([
    { name: 'the anchor’s PR set', h: anchor, label: 'PR set reps' },
    { name: 'the TM test’s 100% set', h: tmTest, label: '100% set reps' },
  ])('reps go with $name, only where they were given', async ({ h, label }) => {
    const route = `PUT /setup/history/${h.cycleIndex}`;
    const { server } = await openChecklist(h, { [route]: next });
    await fireEvent.press(session('Overhead Press', 1));
    await fireEvent.press(session('Squat', 1));
    const reps = screen.getByLabelText(`Overhead Press, round 1, ${label}`);
    // Left blank, the set records its target.
    expect(reps).toHaveProp('placeholder', String(h.rounds[0].days[0].prTarget));
    await fireEvent.changeText(reps, '8');

    await fireEvent.press(screen.getByRole('button', { name: 'Save and start' }));
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(sentTo(server, route)[0].body).toEqual({
      done: [
        { round: 1, slug: 'press', prReps: 8 },
        { round: 1, slug: 'squat' },
      ],
    });
  });

  test('a refusal shows the server’s message, and keeps the checklist as it was', async () => {
    const message = 'The request body is not valid JSON for this route.';
    await openChecklist(anchor, { 'PUT /setup/history/3': json({ error: message }, 400) });
    await fireEvent.press(session('Deadlift', 2));
    await fireEvent.changeText(screen.getByLabelText('Deadlift, round 2, PR set reps'), '6');
    await fireEvent.press(screen.getByRole('button', { name: 'Save and start' }));

    expect(await screen.findByText(message)).toBeOnTheScreen();
    expect(screen).toHavePathname('/setup/history');
    expect(session('Deadlift', 2)).toBeChecked();
    expect(screen.getByLabelText('Deadlift, round 2, PR set reps')).toHaveDisplayValue('6');
    expect(screen.getByRole('button', { name: 'Save and start' })).toBeEnabled();

    // Until the lifter changes what they entered.
    await fireEvent.press(session('Squat', 2));
    expect(screen.queryByText(message)).not.toBeOnTheScreen();
  });

  test('while it saves nothing can change, and a second tap saves once', async () => {
    const put = held<Dashboard>();
    const { server } = await openChecklist(anchor, { 'PUT /setup/history/3': put.reply });
    await fireEvent.press(session('Squat', 1));
    const save = screen.getByRole('button', { name: 'Save and start' });
    await fireEvent.press(save);
    await fireEvent.press(save);

    await waitFor(() => expect(screen.getByRole('button', { name: 'Save and start' })).toBeDisabled());
    expect(session('Squat', 1)).toBeDisabled();
    expect(session('Bench Press', 1)).toBeDisabled();
    expect(screen.getByLabelText('Squat, round 1, PR set reps')).toHaveProp('editable', false);

    await put.answer(next);
    expect(await screen.findByRole('button', { name: 'Start session' })).toBeOnTheScreen();
    expect(sentTo(server, 'PUT /setup/history/3')).toHaveLength(1);
  });
});

// Each screen opened straight, before its read has come back.
describe.each([
  {
    name: 'Training maxes',
    url: '/setup',
    read: 'GET /setup',
    answer: setup,
    find: () => screen.findByLabelText('Squat training max, lb'),
  },
  {
    name: 'Where are you',
    url: '/setup/position',
    read: 'GET /setup/position',
    answer: position,
    find: () => screen.findByRole('radio', { name: /^Anchor/ }),
  },
  {
    name: 'The checklist',
    url: '/setup/history?cycle=3',
    read: 'GET /setup/history/3',
    answer: anchor,
    find: () => screen.findByRole('checkbox', { name: /^Overhead Press, round 1/ }),
  },
])('$name, loading', ({ url, read, answer, find }) => {
  test('waits a moment for a spinner, then shows the screen', async () => {
    const load = held<object>();
    await renderApp(routes, { initialUrl: url, server: fakeServer({ [read]: load.reply }) });

    await act(() => jest.advanceTimersByTime(399));
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
    await act(() => jest.advanceTimersByTime(1));
    expect(screen.getByLabelText('Loading')).toBeOnTheScreen();

    await load.answer(answer);
    expect(await find()).toBeOnTheScreen();
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
  });

  test('a failed first load says why, and Try again asks again', async () => {
    const message = 'Something went wrong. Check the server log.';
    let up = false;
    const server = fakeServer({ [read]: () => (up ? answer : json({ error: message }, 500)) });
    await renderApp(routes, { initialUrl: url, server });

    expect(await screen.findByText(message)).toBeOnTheScreen();
    up = true;
    await fireEvent.press(screen.getByRole('button', { name: 'Try again' }));
    expect(await find()).toBeOnTheScreen();
    expect(screen.queryByText(message)).not.toBeOnTheScreen();
    expect(sentTo(server, read)).toHaveLength(2);
  });
});
