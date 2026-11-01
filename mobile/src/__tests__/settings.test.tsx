import { focusManager } from '@tanstack/react-query';
import { router } from 'expo-router';
import * as SecureStore from 'expo-secure-store';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import { Alert, Appearance, type AlertButton } from 'react-native';

import settingsFixture from '@/api/fixtures/settings.json';
import { keys, makeQueryClient } from '@/api/queries';
import type { BBBMode, Settings, SettingsInput, TrainingMaxInput } from '@/api/schema.gen';
import SettingsLayout, { unstable_settings } from '@/app/(tabs)/(settings)/_layout';
import SettingsScreen from '@/app/(tabs)/(settings)/settings/index';
import LoadingScreen from '@/app/(tabs)/(settings)/settings/loading';
import RestScreen from '@/app/(tabs)/(settings)/settings/rest';
import SupplementalScreen from '@/app/(tabs)/(settings)/settings/supplemental';
import TrainingMaxesScreen from '@/app/(tabs)/(settings)/settings/training-maxes';
import { fakeServer, json, renderApp, testUrl, type Routes } from '@/test-utils';

// Settings against a stand-in server: the root's rows and what they say, each
// section's page sent as numbers over the settings as saved, the form
// reloaded from what the server saved and never from under an edit, leaving
// a page with edits, and the training max prompt. The server rows and
// sign-out are in connect.test.tsx.

const fixture = settingsFixture as Settings;
const routes = {
  _layout: { default: SettingsLayout, unstable_settings },
  'settings/index': SettingsScreen,
  'settings/training-maxes': TrainingMaxesScreen,
  'settings/loading': LoadingScreen,
  'settings/rest': RestScreen,
  'settings/supplemental': SupplementalScreen,
};

// The settings as the server has them: a change answers with them, and a
// refetch reads them.
let stored: Settings;

// What training.SaveSettings makes of a change: a number that isn't positive
// keeps what's there, plates with none positive are refused, and the rest
// are sorted heaviest first. A BBB value that isn't positive goes back to
// the program.
function saveSettings(input: SettingsInput) {
  const plates = input.plates.filter((x) => x > 0).sort((a, b) => b - a);
  if (plates.length === 0) return json({ error: 'List at least one plate.' }, 400);
  const keep = (v: number, was: number) => (v > 0 ? v : was);
  stored = {
    ...stored,
    barWeight: keep(input.barWeight, stored.barWeight),
    rounding: keep(input.rounding, stored.rounding),
    plates,
    warmups: input.warmups,
    restWarmupSeconds: keep(input.restWarmupSeconds, stored.restWarmupSeconds),
    restMainSeconds: keep(input.restMainSeconds, stored.restMainSeconds),
    restSuppSeconds: keep(input.restSuppSeconds, stored.restSuppSeconds),
    bbb: stored.bbb.map((b) => {
      const { mode, value } = input.bbb[b.slug];
      return value > 0 ? { ...b, mode, value } : { ...b, mode: 'program', value: 0 };
    }),
  };
  return stored;
}

function setTrainingMax({ lift, value }: TrainingMaxInput) {
  stored = { ...stored, tms: stored.tms.map((t) => (t.slug === lift ? { ...t, value } : t)) };
  return stored;
}

// Opens Settings on the fixture, seeded as if read before, and waits for the
// read on opening to come back, so a test's edits come after it.
async function openSettings(more: Routes = {}) {
  stored = fixture;
  const server = fakeServer({
    'GET /settings': () => stored,
    'PUT /settings': (sent) => saveSettings(sent.body as SettingsInput),
    'POST /training-maxes': (sent) => setTrainingMax(sent.body as TrainingMaxInput),
    ...more,
  });
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.settings(), fixture);
  await renderApp(routes, { server, queryClient, initialUrl: '/settings' });
  await waitFor(() => expect(queryClient.isFetching()).toBe(0));
  return { server, queryClient };
}

// The root's row for a section, read by its name and then what it says.
const sectionRow = (name: string) => screen.getByRole('button', { name: new RegExp(`^${name}, `) });

// Opens Settings, then a section's page from its row, and waits for the
// page's own read on opening.
async function openPage(name: string, path: string, more: Routes = {}) {
  const opened = await openSettings(more);
  await fireEvent.press(sectionRow(name));
  expect(screen).toHavePathname(path);
  await waitFor(() => expect(opened.queryClient.isFetching()).toBe(0));
  return opened;
}

// What `server` was sent as `route`, such as 'PUT /settings'.
function sentTo(server: ReturnType<typeof fakeServer>, route: string) {
  return server.sent.filter((x) => `${x.method} ${x.path}` === route);
}

// Holds a request in flight until the test answers it.
function held<T extends object>() {
  let answer: (reply: T) => void = () => {};
  const reply = () => new Promise<T>((resolve) => (answer = resolve));
  return { reply, answer: (r: T) => act(async () => answer(r)) };
}

const type = (label: string, text: string) => fireEvent.changeText(screen.getByLabelText(label), text);

// Each lift's BBB mode is a segmented control, one per lift in the
// settings' order.
const pickMode = (slug: string, mode: BBBMode) =>
  fireEvent(screen.getAllByTestId('Picker')[fixture.bbb.findIndex((b) => b.slug === slug)], 'selectionChange', mode);

// The Save in the navigation bar, which the test setup draws in place.
const saveButton = () => screen.getByRole('button', { name: 'Save' });
const save = () => fireEvent.press(saveButton());

// The body of the last PUT /settings.
const sentSettings = (server: ReturnType<typeof fakeServer>) => sentTo(server, 'PUT /settings').at(-1)?.body;

// The app comes back to the foreground, and the screens open refetch.
const comeBack = () =>
  act(() => {
    focusManager.setFocused(false);
    focusManager.setFocused(true);
  });

// The back button or the swipe, and the prompt it may raise.
async function goBack() {
  const alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
  await act(() => router.back());
  const call = alert.mock.calls.find(([title]) => title === 'Discard changes?');
  alert.mockRestore();
  return call && (call[2] as AlertButton[]);
}
const choose = (buttons: AlertButton[], text: string) =>
  act(async () => buttons.find((b) => b.text === text)?.onPress?.());

// Opens the prompt for a lift's training max, as the row's tap does, and
// returns its buttons.
async function promptFor(name: RegExp) {
  const prompt = jest.spyOn(Alert, 'prompt').mockImplementation(() => {});
  await fireEvent.press(screen.getByRole('button', { name }));
  const [, , buttons, , defaultValue] = prompt.mock.calls.at(-1)!;
  return { buttons: buttons as AlertButton[], defaultValue };
}
const enter = (buttons: AlertButton[], text: string) =>
  act(async () => (buttons[1].onPress as (v: string) => void)(text));

afterEach(() => {
  jest.restoreAllMocks();
  focusManager.setFocused(undefined);
});

describe('the root', () => {
  test('each section’s row says what it’s set to', async () => {
    await openSettings();

    expect(sectionRow('Training maxes')).toHaveAccessibleName(
      'Training maxes, Press 100 lb, Deadlift 300 lb, Bench 200 lb, Squat 250 lb',
    );
    expect(sectionRow('Loading')).toHaveAccessibleName('Loading, 45 lb bar · rounds to 5 · warmups');
    expect(screen.getByText('45 lb bar · rounds to 5 · warmups')).toBeOnTheScreen();
    expect(sectionRow('Rest')).toHaveAccessibleName(
      'Rest, 1:00 after warmups, 3:00 after main sets, 1:30 after supplemental',
    );
    expect(screen.getByText('1:00 · 3:00 · 1:30')).toBeOnTheScreen();
    expect(sectionRow('Supplemental')).toHaveAccessibleName('Supplemental, Deadlift 185 lb · Squat 55%');
    expect(screen.getByText('Deadlift 185 lb · Squat 55%')).toBeOnTheScreen();
  });

  test('Supplemental says Program when no lift overrides it, and Loading says when warmups are off', async () => {
    await openSettings({
      'GET /settings': () => ({
        ...fixture,
        warmups: false,
        bbb: fixture.bbb.map((b) => ({ ...b, mode: 'program', value: 0 })),
      }),
    });

    expect(await screen.findByText('Program')).toBeOnTheScreen();
    expect(screen.getByText('45 lb bar · rounds to 5 · no warmups')).toBeOnTheScreen();
  });

  test.each([
    { name: 'Training maxes', path: '/settings/training-maxes', shows: 'Squat, training max 250 lb. Change' },
    { name: 'Loading', path: '/settings/loading', shows: 'Bar weight, lb' },
    { name: 'Rest', path: '/settings/rest', shows: 'After warmups' },
    { name: 'Supplemental', path: '/settings/supplemental', shows: 'Squat BBB, percent of training max' },
  ])('$name opens its page', async ({ name, path, shows }) => {
    await openPage(name, path);
    expect(screen.getByLabelText(shows)).toBeOnTheScreen();
  });
});

describe('saving', () => {
  test('Save sends the page as numbers, and the root shows what the server saved', async () => {
    const { server } = await openPage('Loading', '/settings/loading');
    await type('Bar weight, lb', '');
    await type('Round weights to, lb', '2.5');
    await type('Plates per side, lb', '5, 45 ,25  10');
    await fireEvent(screen.getByLabelText('Warmup sets'), 'valueChange', false);
    await save();

    await waitFor(() => expect(sentTo(server, 'PUT /settings')).toHaveLength(1));
    expect(sentSettings(server)).toEqual({
      barWeight: 0,
      rounding: 2.5,
      plates: [5, 45, 25, 10],
      warmups: false,
      restWarmupSeconds: 60,
      restMainSeconds: 180,
      restSuppSeconds: 90,
      bbb: {
        press: { mode: 'program', value: 0 },
        deadlift: { mode: 'weight', value: 185 },
        bench: { mode: 'program', value: 0 },
        squat: { mode: 'percent', value: 55 },
      },
    });
    // The blank bar weight was kept.
    expect(await screen.findByText('45 lb bar · rounds to 2.5 · no warmups')).toBeOnTheScreen();
    expect(screen).toHavePathname('/settings');

    // And the plates come back sorted.
    await fireEvent.press(sectionRow('Loading'));
    expect(screen.getByLabelText('Plates per side, lb')).toHaveDisplayValue('45, 25, 10, 5');
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('45');
    expect(screen.getByLabelText('Round weights to, lb')).toHaveDisplayValue('2.5');
  });

  test('Rest sends its steppers’ seconds', async () => {
    const { server } = await openPage('Rest', '/settings/rest');
    await fireEvent(screen.getByLabelText('After warmups'), 'valueChange', 75);
    await fireEvent(screen.getByLabelText('After supplemental'), 'valueChange', 120);
    await save();

    expect(await screen.findByText('1:15 · 3:00 · 2:00')).toBeOnTheScreen();
    expect(screen).toHavePathname('/settings');
    expect(sentSettings(server)).toMatchObject({ restWarmupSeconds: 75, restMainSeconds: 180, restSuppSeconds: 120 });
  });

  test.each<{ name: string; mode: BBBMode; text: string | null; sent: { mode: BBBMode; value: number } }>([
    { name: 'program, with no value to type', mode: 'program', text: null, sent: { mode: 'program', value: 0 } },
    { name: 'percent', mode: 'percent', text: '60', sent: { mode: 'percent', value: 60 } },
    { name: 'percent, left blank', mode: 'percent', text: '', sent: { mode: 'percent', value: 0 } },
    { name: 'weight', mode: 'weight', text: '205', sent: { mode: 'weight', value: 205 } },
    { name: 'weight, left blank', mode: 'weight', text: '', sent: { mode: 'weight', value: 0 } },
  ])('a BBB in $name sends $sent', async ({ mode, text, sent }) => {
    const { server } = await openPage('Supplemental', '/settings/supplemental');
    // Deadlift starts at 185 lb.
    if (mode !== 'weight') await pickMode('deadlift', mode);
    if (text !== null) await type(`Deadlift BBB, ${mode === 'percent' ? 'percent of training max' : 'lb'}`, text);
    if (mode === 'program') expect(screen.queryByLabelText(/^Deadlift BBB/)).not.toBeOnTheScreen();
    await save();

    await waitFor(() => expect(sentTo(server, 'PUT /settings')).toHaveLength(1));
    expect(sentSettings(server)).toMatchObject({
      bbb: { deadlift: sent, press: { mode: 'program', value: 0 }, squat: { mode: 'percent', value: 55 } },
    });
  });

  test('Save is disabled until something changes, and an edit put back disables it again', async () => {
    await openPage('Loading', '/settings/loading');
    expect(saveButton()).toBeDisabled();

    await type('Bar weight, lb', '35');
    expect(saveButton()).toBeEnabled();
    await type('Bar weight, lb', '45');
    expect(saveButton()).toBeDisabled();
  });

  test('a page sends its own lines as shown, over the other sections as last read', async () => {
    const { server } = await openPage('Rest', '/settings/rest');
    await fireEvent(screen.getByLabelText('After warmups'), 'valueChange', 75);
    // Changed elsewhere, such as in the web UI, while the page is open.
    stored = { ...fixture, barWeight: 55, restMainSeconds: 240 };
    await comeBack();
    // The root, behind the page, has read the change, and the page keeps
    // what it shows.
    expect(await screen.findByText('55 lb bar · rounds to 5 · warmups', { includeHiddenElements: true })).toBeTruthy();
    expect(screen.getByLabelText('After warmups')).toHaveProp('value', 75);
    expect(screen.getByLabelText('After main sets')).toHaveProp('value', 180);
    await save();

    await waitFor(() => expect(sentTo(server, 'PUT /settings')).toHaveLength(1));
    expect(sentSettings(server)).toMatchObject({ barWeight: 55, restWarmupSeconds: 75, restMainSeconds: 180 });
  });

  test('Save is disabled while it’s sent, and a second tap sends nothing more', async () => {
    const put = held<Settings>();
    const { server } = await openPage('Loading', '/settings/loading', { 'PUT /settings': put.reply });
    await type('Bar weight, lb', '35');
    const button = saveButton();
    await fireEvent.press(button);
    await fireEvent.press(button);

    expect(await screen.findByRole('button', { name: 'Saving…' })).toBeDisabled();
    expect(sentTo(server, 'PUT /settings')).toHaveLength(1);

    await put.answer({ ...fixture, barWeight: 35 });
    expect(await screen.findByText('35 lb bar · rounds to 5 · warmups')).toBeOnTheScreen();
    expect(screen).toHavePathname('/settings');
  });

  test('a refusal shows the server’s message at the top of the page, which stays open with what was typed', async () => {
    await openPage('Loading', '/settings/loading');
    await type('Bar weight, lb', '35');
    await type('Plates per side, lb', '');
    await save();

    const message = await screen.findByText('List at least one plate.');
    const notice = message.parent!;
    expect(notice).toHaveProp('accessibilityRole', 'alert');
    // First on the page, above the lines it's about.
    expect(notice.parent!.children[0]).toBe(notice);
    expect(screen).toHavePathname('/settings/loading');
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('35');
    expect(screen.getByLabelText('Plates per side, lb')).toHaveDisplayValue('');
    expect(saveButton()).toBeEnabled();

    // Fixing the form clears it.
    await type('Plates per side, lb', '45');
    expect(screen.queryByText('List at least one plate.')).not.toBeOnTheScreen();
  });

  test.each([
    {
      what: 'a plate',
      page: ['Loading', '/settings/loading'],
      label: 'Plates per side, lb',
      text: '45, abc',
      message: 'List the plates as numbers, separated by commas.',
    },
    {
      what: 'a bar weight',
      page: ['Loading', '/settings/loading'],
      label: 'Bar weight, lb',
      text: '4.5.',
      message: 'Enter the bar weight as a number.',
    },
    {
      what: 'a BBB value',
      page: ['Supplemental', '/settings/supplemental'],
      label: 'Squat BBB, percent of training max',
      text: '5..5',
      message: 'Enter the Squat BBB percentage as a number.',
    },
  ])('$what that isn’t a number is refused before anything is sent', async ({ page, label, text, message }) => {
    const { server } = await openPage(page[0], page[1]);
    await type(label, text);
    await save();

    expect(await screen.findByText(message)).toBeOnTheScreen();
    expect(screen.getByLabelText(label)).toHaveStyle({ borderWidth: 2 });
    expect(screen.getByLabelText(label)).toHaveDisplayValue(text);
    expect(sentTo(server, 'PUT /settings')).toHaveLength(0);
    expect(screen).toHavePathname(page[1]);

    await type(label, '50');
    expect(screen.queryByText(message)).not.toBeOnTheScreen();
    expect(screen.getByLabelText(label)).not.toHaveStyle({ borderWidth: 2 });
  });
});

describe('leaving a page', () => {
  test('Back with edits asks first: Keep editing stays, and Discard leaves with nothing sent', async () => {
    const { server } = await openPage('Loading', '/settings/loading');
    await type('Bar weight, lb', '35');

    let buttons = await goBack();
    expect(buttons?.map((b) => [b.text, b.style])).toEqual([
      ['Keep editing', 'cancel'],
      ['Discard', 'destructive'],
    ]);
    await choose(buttons!, 'Keep editing');
    expect(screen).toHavePathname('/settings/loading');
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('35');

    buttons = await goBack();
    await choose(buttons!, 'Discard');
    expect(screen).toHavePathname('/settings');
    expect(sentTo(server, 'PUT /settings')).toHaveLength(0);
    expect(screen.getByText('45 lb bar · rounds to 5 · warmups')).toBeOnTheScreen();

    // The page opens again on the settings as saved.
    await fireEvent.press(sectionRow('Loading'));
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('45');
  });

  test('Back without edits leaves at once', async () => {
    await openPage('Rest', '/settings/rest');
    expect(await goBack()).toBeUndefined();
    expect(screen).toHavePathname('/settings');
  });
});

describe('refetching', () => {
  // Changed elsewhere, such as in the web UI.
  const changed = () => (stored = { ...fixture, barWeight: 55, rounding: 2.5 });

  test('a refetch leaves an edit in progress alone', async () => {
    const { server, queryClient } = await openPage('Loading', '/settings/loading');
    const reads = sentTo(server, 'GET /settings').length;
    await type('Bar weight, lb', '35');
    changed();
    await comeBack();

    await waitFor(() => expect(sentTo(server, 'GET /settings')).toHaveLength(reads + 1));
    await waitFor(() => expect(queryClient.isFetching()).toBe(0));
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('35');
    expect(screen.getByLabelText('Round weights to, lb')).toHaveDisplayValue('5');
  });

  test('a refetch reloads a page nobody has touched', async () => {
    await openPage('Loading', '/settings/loading');
    changed();
    await comeBack();

    expect(await screen.findByDisplayValue('2.5')).toBeOnTheScreen();
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('55');
    expect(saveButton()).toBeDisabled();
  });
});

describe('training maxes', () => {
  test('the prompt sends the lift and value, and the row and the root redraw from the answer', async () => {
    const { server } = await openPage('Training maxes', '/settings/training-maxes');
    // The page has no Save of its own: each prompt is its own.
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeOnTheScreen();
    const { buttons, defaultValue } = await promptFor(/^Squat, training max 250 lb/);
    expect(defaultValue).toBe('250');
    await enter(buttons, '255');

    expect(await screen.findByRole('button', { name: 'Squat, training max 255 lb. Change' })).toBeOnTheScreen();
    expect(sentTo(server, 'POST /training-maxes').map((s) => s.body)).toEqual([{ lift: 'squat', value: 255 }]);

    expect(await goBack()).toBeUndefined();
    expect(screen).toHavePathname('/settings');
    expect(sectionRow('Training maxes')).toHaveAccessibleName(/Squat 255 lb$/);
  });

  test('a value that isn’t a positive number sends nothing', async () => {
    const { server } = await openPage('Training maxes', '/settings/training-maxes');
    const { buttons } = await promptFor(/^Squat/);
    for (const text of ['', '0', '-5', 'abc']) await enter(buttons, text);

    expect(sentTo(server, 'POST /training-maxes')).toHaveLength(0);
    expect(screen.getByRole('button', { name: 'Squat, training max 250 lb. Change' })).toBeOnTheScreen();
  });

  test('a refusal shows the server’s message in an alert', async () => {
    await openPage('Training maxes', '/settings/training-maxes', {
      'POST /training-maxes': json({ error: 'Enter a positive training max.' }, 400),
    });
    const alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const { buttons } = await promptFor(/^Squat/);
    await enter(buttons, '255');

    await waitFor(() => expect(alert).toHaveBeenCalledWith('Training max not saved', 'Enter a positive training max.'));
    expect(screen.getByRole('button', { name: 'Squat, training max 250 lb. Change' })).toBeOnTheScreen();
  });
});

describe('appearance', () => {
  // The root's only segmented control.
  const appearancePicker = () => screen.getByTestId('Picker');

  test('a pick applies at once and is kept on the phone, with nothing sent to the server', async () => {
    const setColorScheme = jest.spyOn(Appearance, 'setColorScheme').mockImplementation(() => {});
    const { server } = await openSettings();
    expect(appearancePicker()).toHaveProp('selection', 'system');

    await fireEvent(appearancePicker(), 'selectionChange', 'dark');
    expect(setColorScheme).toHaveBeenLastCalledWith('dark');
    expect(appearancePicker()).toHaveProp('selection', 'dark');
    expect(await SecureStore.getItemAsync('kedge.appearance')).toBe('dark');
    expect(sentTo(server, 'PUT /settings')).toEqual([]);

    await fireEvent(appearancePicker(), 'selectionChange', 'system');
    expect(setColorScheme).toHaveBeenLastCalledWith('unspecified');
    setColorScheme.mockRestore();
  });
});

describe('first load', () => {
  test('a spinner appears only after a moment, then the sections', async () => {
    const load = held<Settings>();
    await renderApp(routes, { server: fakeServer({ 'GET /settings': load.reply }), initialUrl: '/settings' });

    await act(() => jest.advanceTimersByTime(399));
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
    await act(() => jest.advanceTimersByTime(1));
    expect(screen.getByLabelText('Loading')).toBeOnTheScreen();

    await load.answer(fixture);
    expect(await screen.findByText('45 lb bar · rounds to 5 · warmups')).toBeOnTheScreen();
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
  });

  test('a failed first load says why, and Try again asks again', async () => {
    let up = false;
    const failure = 'Something went wrong. Check the server log.';
    await renderApp(routes, {
      server: fakeServer({ 'GET /settings': () => (up ? fixture : json({ error: failure }, 500)) }),
      initialUrl: '/settings',
    });

    expect(await screen.findByText(failure)).toBeOnTheScreen();
    up = true;
    await fireEvent.press(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByText('45 lb bar · rounds to 5 · warmups')).toBeOnTheScreen();
    expect(screen.queryByText(failure)).not.toBeOnTheScreen();
  });

  test('a server that can’t be reached still shows Appearance, its address and Sign out', async () => {
    await renderApp(routes, {
      server: fakeServer({ 'GET /settings': json({ error: 'Something went wrong. Check the server log.' }, 500) }),
      initialUrl: '/settings',
    });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: /^Loading, / })).not.toBeOnTheScreen();
    expect(screen.getByText('System follows your iPhone’s setting.')).toBeOnTheScreen();
    expect(screen.getAllByTestId('Picker')).toHaveLength(1);
    expect(screen.getByText(testUrl)).toBeOnTheScreen();
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeOnTheScreen();
  });
});
