import { router } from 'expo-router';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import { Alert, Text } from 'react-native';

import { unreachable } from '@/api/client';
import dashboardNext from '@/api/fixtures/dashboard-next.json';
import dashboardSetup from '@/api/fixtures/dashboard-setup.json';
import info from '@/api/fixtures/info.json';
import { keys, makeQueryClient, useDashboard } from '@/api/queries';
import type { Dashboard } from '@/api/schema.gen';
import TabLayout from '@/app/(tabs)/_layout';
import HistoryLayout from '@/app/(tabs)/(history)/_layout';
import SettingsLayout from '@/app/(tabs)/(settings)/_layout';
import SettingsScreen from '@/app/(tabs)/(settings)/settings';
import TodayLayout from '@/app/(tabs)/(today)/_layout';
import ConnectScreen from '@/app/connect';
import { RootStack } from '@/components/root-stack';
import { getConnection, sessionEnded } from '@/lib/connection';
import { fakeServer, json, renderApp, testUrl, type Routes } from '@/test-utils';

// Connect and the sign-in gate, rendered through the app's own root stack.
// Today here is a stand-in that only reads the dashboard, so a test can
// answer its read with a 401.

function Today() {
  const { data } = useDashboard();
  return <Text>{data ? 'Today' : 'Loading Today'}</Text>;
}

const stub = (name: string) =>
  function Stub() {
    return <Text>{name}</Text>;
  };

const app = {
  _layout: { default: RootStack, unstable_settings: { anchor: '(tabs)' } },
  '(tabs)/_layout': TabLayout,
  '(tabs)/(today)/_layout': TodayLayout,
  '(tabs)/(today)/index': Today,
  '(tabs)/(history)/_layout': HistoryLayout,
  '(tabs)/(history)/history': stub('History'),
  '(tabs)/(settings)/_layout': SettingsLayout,
  '(tabs)/(settings)/settings': SettingsScreen,
  connect: ConnectScreen,
  'setup/index': stub('Setup'),
  'setup/position': stub('Where are you?'),
  'setup/history': stub('Setup history'),
  'workout/[id]': stub('Session'),
  'lift/[slug]/index': stub('Lift'),
  'lift/[slug]/add-estimate': stub('Add estimate'),
};

const signIn = { 'POST /login': { token: 'fresh', expiresAt: '2099-01-01T00:00:00Z' } };

// Opens the app signed out, which is Connect, before a server that answers
// from `answers`.
async function openConnect(answers: Routes) {
  const server = fakeServer(answers);
  const view = await renderApp(app, { server, signedIn: false, initialUrl: '/' });
  return { server, view };
}

// How the Notice holding `text` reads: an error is announced as an alert
// beside a warning triangle, and news is neither.
async function notice(text: string | RegExp) {
  const box = (await screen.findByText(text)).parent;
  const symbol = box?.children[0];
  return {
    alert: box?.props.accessibilityRole === 'alert',
    icon: typeof symbol === 'object' ? symbol?.props.name : undefined,
  };
}
const error = { alert: true, icon: 'exclamationmark.triangle.fill' };
const news = { alert: false, icon: 'key.fill' };

// A server older than this build, and one newer, with what each says.
const outdated: [number, RegExp][] = [
  [0, /^Your server is too old for this version of Kedge\. Update the server/],
  [2, /^This version of Kedge is too old for your server\. Update Kedge/],
];

// Types the address and password and taps Connect.
async function connect(url: string, password = 'dev') {
  await fireEvent.changeText(screen.getByLabelText('Server address'), url);
  await fireEvent.changeText(screen.getByLabelText('Password'), password);
  await fireEvent.press(screen.getByRole('button', { name: 'Connect' }));
}

describe('Connect', () => {
  test('signs in, saves the connection and opens onto Today', async () => {
    const { server } = await openConnect({ 'GET /info': info, ...signIn, 'GET /dashboard': dashboardNext });
    await connect('raspberrypi:3000');

    expect(await screen.findByText('Today')).toBeOnTheScreen();
    expect(screen).toHavePathname('/');
    expect(server.sent.slice(0, 3).map((s) => `${s.method} ${s.path}`)).toEqual([
      'GET /info',
      'POST /login',
      'GET /dashboard',
    ]);
    expect(server.sent[1].body).toEqual({ password: 'dev' });
    expect(getConnection()).toMatchObject({ signedIn: true, url: 'http://raspberrypi:3000', token: 'fresh' });
  });

  test('goes on to setup, over Today, when the server has no training maxes yet', async () => {
    const { view } = await openConnect({ 'GET /info': info, ...signIn, 'GET /dashboard': dashboardSetup });
    await connect('raspberrypi:3000');

    expect(await screen.findByText('Setup')).toBeOnTheScreen();
    expect(screen).toHavePathname('/setup');
    const stack = view.getRouterState()?.routes[0].state?.routes ?? [];
    expect(stack.map((r) => r.name)).toEqual(['(tabs)', 'setup/index']);

    // So finishing setup comes back to Today.
    await act(() => router.dismissTo('/'));
    expect(await screen.findByText('Today')).toBeOnTheScreen();
    expect(router.canGoBack()).toBe(false);
  });

  test("shows the server's refusal as it is, by the password", async () => {
    await openConnect({ 'GET /info': info, 'POST /login': json({ error: 'Incorrect password.' }, 401) });
    await connect('raspberrypi:3000', 'wrong');

    expect(await notice('Incorrect password.')).toEqual(error);
    expect(screen.getByLabelText('Password')).toHaveStyle({ borderWidth: 2 });
    expect(screen.getByLabelText('Server address')).not.toHaveStyle({ borderWidth: 2 });
    expect(getConnection().signedIn).toBe(false);

    // Typing again clears it.
    await fireEvent.changeText(screen.getByLabelText('Password'), 'dev');
    expect(screen.queryByText('Incorrect password.')).toBeNull();
  });

  test.each([
    ['a web page', new Response('<!doctype html><title>Sign in</title>', { headers: { 'Content-Type': 'text/html' } })],
    ['a 404', new Response('404 page not found', { status: 404, headers: { 'Content-Type': 'text/plain' } })],
  ])('says no Kedge server answered when %s does', async (_, page) => {
    const { server } = await openConnect({ 'GET /info': page });
    await connect('raspberrypi:3000');

    expect(await notice('No Kedge server answered at raspberrypi:3000. Check the address.')).toEqual(error);
    expect(screen.getByLabelText('Server address')).toHaveStyle({ borderWidth: 2 });
    expect(server.sent).toHaveLength(1);
  });

  test('says when the server can’t be reached', async () => {
    await openConnect({
      'GET /info': () => {
        throw new TypeError('Network request failed');
      },
    });
    await connect('raspberrypi:3000');

    expect(await notice(unreachable)).toEqual(error);
    expect(screen.getByLabelText('Server address')).not.toHaveStyle({ borderWidth: 2 });
  });

  test.each(outdated)('stops at a server on API version %i and says what to update', async (apiVersion, message) => {
    const { server } = await openConnect({ 'GET /info': { apiVersion, version: 'other' } });
    await connect('raspberrypi:3000');

    expect(await notice(message)).toEqual(error);
    expect(server.sent.map((s) => s.path)).toEqual(['/info']);
  });

  test('refuses plain http:// to a server on the internet before sending anything', async () => {
    const { server } = await openConnect({});
    await connect('http://lift.lyle.sh');

    expect(
      await notice('Use https:// for lift.lyle.sh. Kedge sends plain http:// only to addresses on your own network.'),
    ).toEqual(error);
    expect(screen.getByLabelText('Server address')).toHaveStyle({ borderWidth: 2 });
    expect(server.sent).toHaveLength(0);
  });

  test('says it’s connecting, and a second tap sends nothing more', async () => {
    let answer: (r: Response) => void = () => {};
    const { server } = await openConnect({ 'GET /info': () => new Promise<Response>((resolve) => (answer = resolve)) });
    await connect('raspberrypi:3000');
    await fireEvent.press(screen.getByRole('button', { name: /^Connect/ }));
    await fireEvent(screen.getByLabelText('Password'), 'submitEditing');

    expect(await screen.findByRole('button', { name: 'Connecting…' })).toBeDisabled();
    expect(server.sent).toHaveLength(1);

    await act(async () => answer(json({ error: 'Not found.' }, 404)));
    expect(await screen.findByRole('button', { name: 'Connect' })).toBeEnabled();
  });
});

describe('the sign-in gate', () => {
  test('signed out, only Connect is open', async () => {
    await openConnect({});
    await waitFor(() => expect(screen).toHavePathname('/connect'));
    expect(screen.queryByText('Today')).toBeNull();

    await act(() => router.push('/workout/3'));
    await waitFor(() => expect(screen).toHavePathname('/connect'));
    expect(screen.queryByText('Session')).toBeNull();
  });

  test('signed in, Connect is closed', async () => {
    const server = fakeServer({ 'GET /info': info, 'GET /dashboard': dashboardNext });
    await renderApp(app, { server, initialUrl: '/connect' });

    expect(await screen.findByText('Today')).toBeOnTheScreen();
    expect(screen).toHavePathname('/');
  });

  test('a 401 closes it onto Connect, with the address kept and the reason given as news', async () => {
    const server = fakeServer({ 'GET /info': info, 'GET /dashboard': json({ error: 'Sign in to continue.' }, 401) });
    const queryClient = makeQueryClient();
    // Today is on show when its refetch is refused.
    queryClient.setQueryData(keys.dashboard(), dashboardNext as Dashboard);
    await renderApp(app, { server, queryClient, initialUrl: '/' });

    expect(await notice(sessionEnded)).toEqual(news);
    expect(screen).toHavePathname('/connect');
    expect(screen.getByLabelText('Server address')).toHaveDisplayValue(testUrl);
    expect(router.canGoBack()).toBe(false);
    // Nothing of the old session is left to show through.
    expect(queryClient.getQueryData(keys.dashboard())).toBeUndefined();
  });

  test.each(outdated)('signs out at launch from a server on API version %i, saying what to update', async (v, text) => {
    const server = fakeServer({ 'GET /info': { apiVersion: v, version: 'other' }, 'GET /dashboard': dashboardNext });
    await renderApp(app, { server, initialUrl: '/' });

    expect(await notice(text)).toEqual(error);
    expect(screen).toHavePathname('/connect');
    expect(screen.getByLabelText('Server address')).toHaveDisplayValue(testUrl);
    expect(getConnection().signedIn).toBe(false);
  });

  test('a server that can’t be reached for its version blocks nothing', async () => {
    const server = fakeServer({
      'GET /info': () => {
        throw new TypeError('Network request failed');
      },
      'GET /dashboard': dashboardNext,
    });
    const queryClient = makeQueryClient();
    await renderApp(app, { server, queryClient, initialUrl: '/' });

    await waitFor(() => expect(queryClient.getQueryState(keys.info())?.status).toBe('error'), { timeout: 3000 });
    expect(screen.getByText('Today')).toBeOnTheScreen();
    expect(getConnection().signedIn).toBe(true);
  });

  test('Settings shows the server, and signing out there opens Connect with the address kept', async () => {
    const server = fakeServer({ 'GET /info': info, 'GET /dashboard': dashboardNext });
    await renderApp(app, { server, initialUrl: '/settings' });

    expect(screen.getByText(testUrl)).toBeOnTheScreen();
    expect(await screen.findByText('test · API 1')).toBeOnTheScreen();

    const alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    await fireEvent.press(screen.getByRole('button', { name: 'Sign out' }));
    const buttons = alert.mock.calls[0][2] ?? [];
    expect(buttons.map((b) => b.text)).toEqual(['Cancel', 'Sign out']);
    expect(getConnection().signedIn).toBe(true);

    await act(async () => buttons[1].onPress?.());
    await waitFor(() => expect(screen).toHavePathname('/connect'));
    expect(screen.getByLabelText('Server address')).toHaveDisplayValue(testUrl);
    // Signing out on purpose needs no explaining.
    expect(screen.queryByText(sessionEnded)).toBeNull();
    alert.mockRestore();
  });

  test('Settings shows a dash for the version until the server gives it, and if it never does', async () => {
    const queryClient = makeQueryClient();
    const server = fakeServer({ 'GET /dashboard': dashboardNext });
    await renderApp(app, { server, queryClient, initialUrl: '/settings' });
    expect(screen.getByLabelText('Unknown')).toHaveTextContent('—');

    await waitFor(() => expect(queryClient.getQueryState(keys.info())?.status).toBe('error'));
    expect(screen.getByLabelText('Unknown')).toHaveTextContent('—');
  });
});
