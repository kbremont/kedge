import { Stack } from 'expo-router';
import { act, fireEvent, screen, waitFor } from 'expo-router/testing-library';
import type { ReactNode } from 'react';
import { Alert } from 'react-native';

import { unreachable } from '@/api/client';
import errorInput from '@/api/fixtures/error-input.json';
import liftEmpty from '@/api/fixtures/lift-empty.json';
import liftFixture from '@/api/fixtures/lift.json';
import liftSingle from '@/api/fixtures/lift-single.json';
import { keys, makeQueryClient } from '@/api/queries';
import type { Lift } from '@/api/schema.gen';
import AddEstimateSheet from '@/app/lift/[slug]/add-estimate';
import LiftScreen from '@/app/lift/[slug]/index';
import { fakeServer, json, renderApp, type Routes } from '@/test-utils';

// A lift's page and its add sheet against a stand-in server: the chart as
// the server lays it out, deleting a past estimate, and adding one.

// The real swipeable row needs Reanimated, whose Jest mock can't load here.
// The stand-in draws each row with its Delete beside it, as if swiped open,
// and keeps whether it may be swiped and what the screen asked of it.
const mockClose = jest.fn();
jest.mock('react-native-gesture-handler/ReanimatedSwipeable', () => {
  const { createElement } = jest.requireActual<typeof import('react')>('react');
  const { View } = jest.requireActual<typeof import('react-native')>('react-native');
  const row = { close: () => mockClose(), openLeft() {}, openRight() {}, reset() {} };
  return {
    __esModule: true,
    default: ({
      enabled,
      renderRightActions,
      children,
    }: {
      enabled?: boolean;
      renderRightActions?: (progress: unknown, translation: unknown, methods: typeof row) => ReactNode;
      children: ReactNode;
    }) => {
      // `enabled` rides along on the view, for a test to read.
      const props = { testID: 'swipeable', enabled };
      return createElement(View, props, renderRightActions?.(null, null, row), children);
    },
  };
});

const lift = liftFixture as Lift;

const routes = {
  _layout: () => <Stack />,
  'lift/[slug]/index': LiftScreen,
  'lift/[slug]/add-estimate': AddEstimateSheet,
};

// The lift as the server has it: a change answers with it, and a refetch
// reads it.
let current: Lift;

// Opens the page on `l`, seeded as it is when opened from Today.
async function openLift(l: Lift, more: Routes = {}) {
  current = l;
  const server = fakeServer({ [`GET /lifts/${l.slug}`]: () => current, ...more });
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.lift(l.slug), l);
  await renderApp(routes, { initialUrl: `/lift/${l.slug}`, server, queryClient });
  return { server, queryClient };
}

// What `server` was sent as `route`, such as 'DELETE /lifts/press/estimates/1'.
function sentTo(server: ReturnType<typeof fakeServer>, route: string) {
  return server.sent.filter((x) => `${x.method} ${x.path}` === route);
}

// Holds a request in flight until the test answers it.
function held<T extends object>() {
  let answer: (reply: T) => void = () => {};
  const reply = () => new Promise<T>((resolve) => (answer = resolve));
  return { reply, answer: (r: T) => act(async () => answer(r)) };
}

// The chart's points, as react-native-svg draws them.
const points = () => screen.container.queryAll((i) => i.type === 'RNSVGCircle');

// The lift once estimate `id` is gone, as the server answers.
function without(l: Lift, id: number): Lift {
  const pastEstimates = l.pastEstimates.filter((e) => e.id !== id);
  return { ...l, pastEstimates, bestE1rm: Math.max(0, ...pastEstimates.map((e) => e.estimate)) };
}

afterEach(() => {
  jest.restoreAllMocks();
  mockClose.mockClear();
});

describe('the chart', () => {
  test.each([
    { name: 'lift', l: lift, count: 5 },
    { name: 'lift-single', l: liftSingle as Lift, count: 1 },
  ])('$name draws its $count points as the server laid them out', async ({ l, count }) => {
    await openLift(l);
    expect(points()).toHaveLength(count);
    expect(screen.queryByText(/^Not enough data yet/)).not.toBeOnTheScreen();
  });

  test('lift-empty says there isn’t enough data yet, and has no estimates', async () => {
    await openLift(liftEmpty as Lift);
    expect(points()).toHaveLength(0);
    expect(screen.getByText(/^Not enough data yet\. The TM test and the anchor’s PR sets/)).toBeOnTheScreen();
    expect(screen.getByText('None yet.')).toBeOnTheScreen();
    expect(screen.queryByText('Best, all sources')).not.toBeOnTheScreen();
  });
});

describe('deleting a past estimate', () => {
  const route = `DELETE /lifts/${lift.slug}/estimates/1`;

  test('sends DELETE, and the page shows the lift the server answers with', async () => {
    const remove = held<Lift>();
    const { server } = await openLift(lift, { [route]: remove.reply });
    expect(screen.getByText('Feb 10, 2025')).toBeOnTheScreen();

    await fireEvent.press(screen.getByRole('button', { name: 'Delete the estimate of Feb 10, 2025' }));
    await waitFor(() => expect(sentTo(server, route)).toHaveLength(1));
    // The row stays until the answer comes.
    expect(screen.getByText('Feb 10, 2025')).toBeOnTheScreen();

    current = without(lift, 1);
    await remove.answer(current);
    await waitFor(() => expect(screen.queryByText('Feb 10, 2025')).not.toBeOnTheScreen());
    expect(screen.getByText('Sep 15, 2025')).toBeOnTheScreen();
  });

  test('while one is on its way, no row swipes and no Delete sends again', async () => {
    const remove = held<Lift>();
    const { server } = await openLift(lift, { [route]: remove.reply });
    const del = screen.getByRole('button', { name: 'Delete the estimate of Feb 10, 2025' });
    // A second tap before the page redraws.
    await fireEvent.press(del);
    await fireEvent.press(del);

    await waitFor(() => expect(del).toBeDisabled());
    expect(screen.getByRole('button', { name: 'Delete the estimate of Sep 15, 2025' })).toBeDisabled();
    for (const row of screen.getAllByTestId('swipeable')) expect(row.props.enabled).toBe(false);
    // VoiceOver's Delete action is held back too.
    await fireEvent(screen.getByLabelText(/^Sep 15, 2025, /), 'accessibilityAction', {
      nativeEvent: { actionName: 'delete' },
    });

    current = without(lift, 1);
    await remove.answer(current);
    await waitFor(() => expect(screen.queryByText('Feb 10, 2025')).not.toBeOnTheScreen());
    expect(server.sent.filter((x) => x.method === 'DELETE')).toHaveLength(1);
    for (const row of screen.getAllByTestId('swipeable')) expect(row.props.enabled).toBe(true);
  });

  test('a failure shows the server’s message in an alert, and the row slides back', async () => {
    const alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    const failed = 'Something went wrong. Check the server log.';
    const { server } = await openLift(lift, { [route]: () => json({ error: failed }, 500) });

    await fireEvent.press(screen.getByRole('button', { name: 'Delete the estimate of Feb 10, 2025' }));
    await waitFor(() => expect(alert).toHaveBeenCalledWith('Couldn’t delete the estimate', failed));
    expect(mockClose).toHaveBeenCalledTimes(1);
    expect(screen.getByText('Feb 10, 2025')).toBeOnTheScreen();

    // It can be tried again.
    const del = screen.getByRole('button', { name: 'Delete the estimate of Feb 10, 2025' });
    await waitFor(() => expect(del).toBeEnabled());
    await fireEvent.press(del);
    await waitFor(() => expect(sentTo(server, route)).toHaveLength(2));
  });

  test('VoiceOver’s Delete action sends the same request', async () => {
    const { server } = await openLift(lift, { [route]: () => (current = without(lift, 1)) });
    await fireEvent(screen.getByLabelText('Feb 10, 2025, 105 × 1, estimated max 105 lb'), 'accessibilityAction', {
      nativeEvent: { actionName: 'delete' },
    });
    await waitFor(() => expect(screen.queryByText('Feb 10, 2025')).not.toBeOnTheScreen());
    expect(sentTo(server, route)).toHaveLength(1);
  });
});

describe('adding a past estimate', () => {
  const route = `POST /lifts/${lift.slug}/estimates`;
  const added = {
    ...lift,
    pastEstimates: [...lift.pastEstimates, { id: 3, date: 'Mar 16, 2026', weight: 115, reps: 3, estimate: 126.5 }],
    bestE1rm: 126.5,
  };

  // Opens the page, then its add sheet from the + by Past estimates.
  async function openSheet(more: Routes = {}) {
    const opened = await openLift(lift, more);
    await fireEvent.press(screen.getByRole('button', { name: 'Add a past estimate' }));
    await waitFor(() => expect(screen).toHavePathname(`/lift/${lift.slug}/add-estimate`));
    return opened;
  }

  const fill = async (weight: string, reps: string) => {
    await fireEvent.changeText(screen.getByLabelText('Weight in lb'), weight);
    await fireEvent.changeText(screen.getByLabelText('Reps'), reps);
  };

  test('sends the date, weight and reps, goes back, and the page shows the new estimate', async () => {
    const { server } = await openSheet({ [route]: () => (current = added) });
    await fill('115', '3');
    await fireEvent.press(screen.getByRole('button', { name: 'Add estimate' }));

    await waitFor(() => expect(screen).toHavePathname(`/lift/${lift.slug}`));
    // Left as it opened, the date is the server's today.
    expect(sentTo(server, route)[0].body).toEqual({ date: '2026-03-16', weight: 115, reps: 3 });
    expect(await screen.findByLabelText('Mar 16, 2026, 115 × 3, estimated max 126.5 lb')).toBeOnTheScreen();
  });

  test('a refusal shows the server’s message, and the sheet stays open to fix it', async () => {
    const { server } = await openSheet({ [route]: () => json(errorInput, 400) });
    await fill('0', '1');
    await fireEvent.press(screen.getByRole('button', { name: 'Add estimate' }));

    expect(await screen.findByText(errorInput.error)).toBeOnTheScreen();
    expect(screen).toHavePathname(`/lift/${lift.slug}/add-estimate`);
    expect(screen.getByLabelText('Weight in lb')).toHaveDisplayValue('0');
    expect(sentTo(server, route)).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Add estimate' })).toBeEnabled();
  });

  test('a server out of reach says so, and the sheet stays open', async () => {
    await openSheet({ [route]: () => Promise.reject(new TypeError('Network request failed')) });
    await fill('115', '3');
    await fireEvent.press(screen.getByRole('button', { name: 'Add estimate' }));

    expect(await screen.findByText(unreachable)).toBeOnTheScreen();
    expect(screen).toHavePathname(`/lift/${lift.slug}/add-estimate`);
  });

  // Late in the evening west of Greenwich, or just after midnight east of
  // it, the UTC day is another day: the one picked is what's sent.
  test.each([
    { name: 'late in the evening', hour: 23, minute: 30 },
    { name: 'just after midnight', hour: 0, minute: 30 },
  ])('a day picked $name is sent as that day', async ({ hour, minute }) => {
    const { server } = await openSheet({ [route]: () => (current = added) });
    await fireEvent(screen.getByTestId('DatePicker'), 'dateChange', new Date(2026, 2, 15, hour, minute));
    await fill('115', '3');
    await fireEvent.press(screen.getByRole('button', { name: 'Add estimate' }));

    await waitFor(() => expect(sentTo(server, route)).toHaveLength(1));
    expect(sentTo(server, route)[0].body).toMatchObject({ date: '2026-03-15' });
  });

  test('Add estimate is disabled while it is on its way, and a second tap sends nothing', async () => {
    const post = held<Lift>();
    const { server } = await openSheet({ [route]: post.reply });
    await fill('115', '3');
    const add = screen.getByRole('button', { name: 'Add estimate' });
    await fireEvent.press(add);
    await fireEvent.press(add);

    await waitFor(() => expect(add).toBeDisabled());
    expect(screen).toHavePathname(`/lift/${lift.slug}/add-estimate`);
    current = added;
    await post.answer(added);
    await waitFor(() => expect(screen).toHavePathname(`/lift/${lift.slug}`));
    expect(sentTo(server, route)).toHaveLength(1);
  });

  test('Cancel goes back and sends nothing', async () => {
    const { server } = await openSheet();
    await fill('115', '3');
    await fireEvent.press(screen.getByRole('button', { name: 'Cancel' }));
    await waitFor(() => expect(screen).toHavePathname(`/lift/${lift.slug}`));
    expect(sentTo(server, route)).toHaveLength(0);
  });
});

describe('first load', () => {
  test('a spinner appears only after a moment, then the page', async () => {
    const load = held<Lift>();
    await renderApp(routes, {
      initialUrl: `/lift/${lift.slug}`,
      server: fakeServer({ [`GET /lifts/${lift.slug}`]: load.reply }),
    });

    await act(() => jest.advanceTimersByTime(399));
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
    await act(() => jest.advanceTimersByTime(1));
    expect(screen.getByLabelText('Loading')).toBeOnTheScreen();

    await load.answer(lift);
    expect(await screen.findByText('Best, all sources')).toBeOnTheScreen();
    expect(screen.queryByLabelText('Loading')).not.toBeOnTheScreen();
  });

  test('a failed first load says why, and Try again asks again', async () => {
    const failed = 'Something went wrong. Check the server log.';
    let up = false;
    const server = fakeServer({ [`GET /lifts/${lift.slug}`]: () => (up ? lift : json({ error: failed }, 500)) });
    await renderApp(routes, { initialUrl: `/lift/${lift.slug}`, server });

    expect(await screen.findByText(failed)).toBeOnTheScreen();
    up = true;
    await fireEvent.press(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByText('Best, all sources')).toBeOnTheScreen();
    expect(screen.queryByText(failed)).not.toBeOnTheScreen();
    expect(sentTo(server, `GET /lifts/${lift.slug}`)).toHaveLength(2);
  });

  test('a refetch that fails leaves the page as it was', async () => {
    const { queryClient } = await openLift(lift, {
      [`GET /lifts/${lift.slug}`]: json({ error: 'Something went wrong. Check the server log.' }, 500),
    });
    await waitFor(() => expect(queryClient.getQueryState(keys.lift(lift.slug))?.status).toBe('error'));
    expect(screen.getByText('Best, all sources')).toBeOnTheScreen();
    expect(screen.queryByText('Something went wrong. Check the server log.')).not.toBeOnTheScreen();
  });
});
