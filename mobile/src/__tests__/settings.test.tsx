import { focusManager } from '@tanstack/react-query';
import { Stack } from 'expo-router';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import { Alert, type AlertButton } from 'react-native';

import settingsFixture from '@/api/fixtures/settings.json';
import { keys, makeQueryClient } from '@/api/queries';
import type { BBBMode, Settings, SettingsInput, TrainingMaxInput } from '@/api/schema.gen';
import SettingsScreen from '@/app/(tabs)/(settings)/settings';
import { fakeServer, json, renderApp, type Routes } from '@/test-utils';

// Settings against a stand-in server: the form sent as numbers, the form
// reloaded from what the server saved and never from under an edit, and the
// training max prompt. The server rows and sign-out are in connect.test.tsx.

const fixture = settingsFixture as Settings;
const routes = { _layout: () => <Stack />, index: SettingsScreen };

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
  await renderApp(routes, { server, queryClient });
  await waitFor(() => expect(queryClient.isFetching()).toBe(0));
  return { server, queryClient };
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

const save = () => fireEvent.press(screen.getByRole('button', { name: 'Save settings' }));

// The body of the last PUT /settings.
const sentSettings = (server: ReturnType<typeof fakeServer>) => sentTo(server, 'PUT /settings').at(-1)?.body;

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

describe('saving', () => {
  test('Save sends the form as numbers, and the form reloads from what the server saved', async () => {
    const { server } = await openSettings();
    await type('Bar weight, lb', '');
    await type('Round weights to, lb', '2.5');
    await type('Plates per side, lb', '5, 45 ,25  10');
    await fireEvent(screen.getByLabelText('Warmup sets'), 'valueChange', false);
    await fireEvent(screen.getByLabelText('After warmups'), 'valueChange', 75);
    await save();

    await waitFor(() => expect(sentTo(server, 'PUT /settings')).toHaveLength(1));
    expect(sentSettings(server)).toEqual({
      barWeight: 0,
      rounding: 2.5,
      plates: [5, 45, 25, 10],
      warmups: false,
      restWarmupSeconds: 75,
      restMainSeconds: 180,
      restSuppSeconds: 90,
      bbb: {
        press: { mode: 'program', value: 0 },
        deadlift: { mode: 'weight', value: 185 },
        bench: { mode: 'program', value: 0 },
        squat: { mode: 'percent', value: 55 },
      },
    });
    // The blank bar weight was kept, and the plates come back sorted.
    expect(await screen.findByDisplayValue('45, 25, 10, 5')).toBeOnTheScreen();
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('45');
    expect(screen.getByLabelText('Round weights to, lb')).toHaveDisplayValue('2.5');
  });

  test.each<{ name: string; mode: BBBMode; text: string | null; sent: { mode: BBBMode; value: number } }>([
    { name: 'program, with no value to type', mode: 'program', text: null, sent: { mode: 'program', value: 0 } },
    { name: 'percent', mode: 'percent', text: '60', sent: { mode: 'percent', value: 60 } },
    { name: 'percent, left blank', mode: 'percent', text: '', sent: { mode: 'percent', value: 0 } },
    { name: 'weight', mode: 'weight', text: '205', sent: { mode: 'weight', value: 205 } },
    { name: 'weight, left blank', mode: 'weight', text: '', sent: { mode: 'weight', value: 0 } },
  ])('a BBB in $name sends $sent', async ({ mode, text, sent }) => {
    const { server } = await openSettings();
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

  test('a save says so in place of the button, until the next edit', async () => {
    await openSettings();
    expect(screen.getByRole('button', { name: 'Save settings' })).toBeDisabled();
    await type('Bar weight, lb', '35');
    await save();

    expect(await screen.findByText('Saved. The next session uses these settings.')).toBeOnTheScreen();
    expect(screen.queryByRole('button', { name: 'Save settings' })).not.toBeOnTheScreen();

    await type('Bar weight, lb', '33');
    expect(screen.queryByText('Saved. The next session uses these settings.')).not.toBeOnTheScreen();
    expect(screen.getByRole('button', { name: 'Save settings' })).toBeEnabled();
  });

  test('Save is disabled while it’s sent, and a second tap sends nothing more', async () => {
    const put = held<Settings>();
    const { server } = await openSettings({ 'PUT /settings': put.reply });
    await type('Bar weight, lb', '35');
    const button = screen.getByRole('button', { name: 'Save settings' });
    await fireEvent.press(button);
    await fireEvent.press(button);

    expect(await screen.findByRole('button', { name: 'Saving…' })).toBeDisabled();
    expect(sentTo(server, 'PUT /settings')).toHaveLength(1);

    await put.answer({ ...fixture, barWeight: 35 });
    expect(await screen.findByText('Saved. The next session uses these settings.')).toBeOnTheScreen();
  });

  test('a refusal shows the server’s message by Save, and keeps what was typed', async () => {
    await openSettings();
    await type('Bar weight, lb', '35');
    await type('Plates per side, lb', '');
    await save();

    const message = await screen.findByText('List at least one plate.');
    const notice = message.parent!;
    expect(notice).toHaveProp('accessibilityRole', 'alert');
    // Right above the button it's about.
    const shown = notice.parent!.children;
    expect(shown[shown.indexOf(notice) + 1]).toBe(screen.getByRole('button', { name: 'Save settings' }));
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('35');
    expect(screen.getByLabelText('Plates per side, lb')).toHaveDisplayValue('');
    expect(screen.getByRole('button', { name: 'Save settings' })).toBeEnabled();

    // Fixing the form clears it.
    await type('Plates per side, lb', '45');
    expect(screen.queryByText('List at least one plate.')).not.toBeOnTheScreen();
  });

  test.each([
    {
      what: 'a plate',
      label: 'Plates per side, lb',
      text: '45, abc',
      message: 'List the plates as numbers, separated by commas.',
    },
    { what: 'a bar weight', label: 'Bar weight, lb', text: '4.5.', message: 'Enter the bar weight as a number.' },
    {
      what: 'a BBB value',
      label: 'Squat BBB, percent of training max',
      text: '5..5',
      message: 'Enter the Squat BBB percentage as a number.',
    },
  ])('$what that isn’t a number is refused before anything is sent', async ({ label, text, message }) => {
    const { server } = await openSettings();
    await type(label, text);
    await save();

    expect(await screen.findByText(message)).toBeOnTheScreen();
    expect(screen.getByLabelText(label)).toHaveStyle({ borderWidth: 2 });
    expect(screen.getByLabelText(label)).toHaveDisplayValue(text);
    expect(sentTo(server, 'PUT /settings')).toHaveLength(0);

    await type(label, '50');
    expect(screen.queryByText(message)).not.toBeOnTheScreen();
    expect(screen.getByLabelText(label)).not.toHaveStyle({ borderWidth: 2 });
  });
});

describe('refetching', () => {
  // The app comes back to the foreground, and the screen on show refetches.
  const comeBack = () =>
    act(() => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
    });

  // Changed elsewhere, such as in the web UI. The training maxes aren't part
  // of the form, so their rows show when the refetch has reached the screen.
  const changed = () =>
    (stored = {
      ...fixture,
      barWeight: 55,
      rounding: 2.5,
      tms: fixture.tms.map((t) => (t.slug === 'squat' ? { ...t, value: 260 } : t)),
    });

  test('a refetch leaves an edit in progress alone', async () => {
    const { server } = await openSettings();
    await type('Bar weight, lb', '35');
    changed();
    await comeBack();

    expect(await screen.findByRole('button', { name: 'Squat, training max 260 lb. Change' })).toBeOnTheScreen();
    expect(sentTo(server, 'GET /settings')).toHaveLength(2);
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('35');
    expect(screen.getByLabelText('Round weights to, lb')).toHaveDisplayValue('5');
  });

  test('a refetch reloads a form nobody has touched', async () => {
    await openSettings();
    changed();
    await comeBack();

    expect(await screen.findByRole('button', { name: 'Squat, training max 260 lb. Change' })).toBeOnTheScreen();
    expect(screen.getByLabelText('Round weights to, lb')).toHaveDisplayValue('2.5');
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('55');
    expect(screen.getByRole('button', { name: 'Save settings' })).toBeDisabled();
  });
});

describe('training maxes', () => {
  test('the prompt sends the lift and value, and the row redraws from the answer', async () => {
    const { server } = await openSettings();
    // An edit in progress stays through the answer.
    await type('Bar weight, lb', '35');
    const { buttons, defaultValue } = await promptFor(/^Squat, training max 250 lb/);
    expect(defaultValue).toBe('250');
    await enter(buttons, '255');

    expect(await screen.findByRole('button', { name: 'Squat, training max 255 lb. Change' })).toBeOnTheScreen();
    expect(sentTo(server, 'POST /training-maxes').map((s) => s.body)).toEqual([{ lift: 'squat', value: 255 }]);
    expect(screen.getByLabelText('Bar weight, lb')).toHaveDisplayValue('35');
  });

  test('a value that isn’t a positive number sends nothing', async () => {
    const { server } = await openSettings();
    const { buttons } = await promptFor(/^Squat/);
    for (const text of ['', '0', '-5', 'abc']) await enter(buttons, text);

    expect(sentTo(server, 'POST /training-maxes')).toHaveLength(0);
    expect(screen.getByRole('button', { name: 'Squat, training max 250 lb. Change' })).toBeOnTheScreen();
  });

  test('a refusal shows the server’s message in an alert', async () => {
    await openSettings({ 'POST /training-maxes': json({ error: 'Enter a positive training max.' }, 400) });
    const alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const { buttons } = await promptFor(/^Squat/);
    await enter(buttons, '255');

    await waitFor(() => expect(alert).toHaveBeenCalledWith('Training max not saved', 'Enter a positive training max.'));
    expect(screen.getByRole('button', { name: 'Squat, training max 250 lb. Change' })).toBeOnTheScreen();
  });
});

describe('first load', () => {
  test('a spinner appears only after a moment, then the form', async () => {
    const load = held<Settings>();
    await renderApp(routes, { server: fakeServer({ 'GET /settings': load.reply }) });

    await act(() => jest.advanceTimersByTime(399));
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
    await act(() => jest.advanceTimersByTime(1));
    expect(screen.getByLabelText('Loading')).toBeOnTheScreen();

    await load.answer(fixture);
    expect(await screen.findByLabelText('Bar weight, lb')).toHaveDisplayValue('45');
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
  });

  test('a failed first load says why, and Try again asks again', async () => {
    let up = false;
    const failure = 'Something went wrong. Check the server log.';
    await renderApp(routes, {
      server: fakeServer({ 'GET /settings': () => (up ? fixture : json({ error: failure }, 500)) }),
    });

    expect(await screen.findByText(failure)).toBeOnTheScreen();
    up = true;
    await fireEvent.press(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByLabelText('Bar weight, lb')).toHaveDisplayValue('45');
    expect(screen.queryByText(failure)).not.toBeOnTheScreen();
  });
});
