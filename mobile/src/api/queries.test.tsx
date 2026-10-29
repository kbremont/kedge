import { useQueryClient } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react-native';
import type { ReactNode } from 'react';

import dashboardNext from '@/api/fixtures/dashboard-next.json';
import infoFixture from '@/api/fixtures/info.json';
import liftFixture from '@/api/fixtures/lift.json';
import workoutAmrap from '@/api/fixtures/workout-amrap.json';
import workoutCurrent from '@/api/fixtures/workout-current.json';
import {
  ApiProvider,
  keys,
  makeQueryClient,
  show,
  useDashboard,
  useFinishWorkout,
  useLogin,
  useLogSet,
  useStartWorkout,
  useWorkout,
} from '@/api/queries';
import type { Dashboard, Lift, Workout } from '@/api/schema.gen';
import { getConnection, saveConnection, signOut } from '@/lib/connection';
import { clearRest, useRest } from '@/lib/rest';
import { fakeServer, json, testToken, testUrl, type Routes } from '@/test-utils';

const dashboard = dashboardNext as Dashboard;
const current = workoutCurrent as Workout;
const amrap = workoutAmrap as Workout;
const id = current.id;

function setUp(routes: Routes, queryClient = makeQueryClient()) {
  const server = fakeServer(routes);
  const wrapper = ({ children }: { children: ReactNode }) => (
    <ApiProvider fetch={server.fetch} queryClient={queryClient}>
      {children}
    </ApiProvider>
  );
  return { server, queryClient, wrapper };
}

beforeEach(async () => {
  jest.useFakeTimers();
  clearRest(id);
  await saveConnection({ url: testUrl, token: testToken, expiresAt: '2099-01-01T00:00:00Z' });
});

afterEach(() => jest.useRealTimers());

test('a seeded cache is read under the same key the hook uses', async () => {
  const queryClient = makeQueryClient();
  queryClient.setQueryData(keys.workout(id), current);
  const { wrapper, server } = setUp({ [`GET /workouts/${id}`]: current }, queryClient);
  const { result } = await renderHook(() => useWorkout(id), { wrapper });
  expect(result.current.data).toBe(current);
  // Seeded data is stale at once, so the screen still checks with the server.
  await waitFor(() => expect(server.sent).toHaveLength(1));
  expect(server.sent[0]).toMatchObject({ method: 'GET', path: `/workouts/${id}`, auth: `Bearer ${testToken}` });
});

describe('query defaults', () => {
  test.each([
    ['the network failed', () => Promise.reject(new TypeError('Network request failed'))],
    // tailscale serve, while the server behind it is down.
    ['a proxy says it is down', () => new Response('502 Bad Gateway', { status: 502 })],
  ])('a query asks once more when the server could not be reached: %s', async (_, reply) => {
    let calls = 0;
    const { wrapper } = setUp({
      'GET /dashboard': () => {
        calls++;
        return reply();
      },
    });
    const { result } = await renderHook(() => useDashboard(), { wrapper });
    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 5_000 });
    expect(calls).toBe(2);
    expect(result.current.error).toMatchObject({ status: 0 });
  });

  test('a query refused is not asked again', async () => {
    const { wrapper, server } = setUp({ 'GET /dashboard': json({ error: 'Something went wrong.' }, 500) });
    const { result } = await renderHook(() => useDashboard(), { wrapper });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(server.sent).toHaveLength(1);
    expect(result.current.error?.message).toBe('Something went wrong.');
  });

  test('a change is never resent', async () => {
    let calls = 0;
    const { wrapper } = setUp({
      'POST /workouts': () => {
        calls++;
        return Promise.reject(new TypeError('Network request failed'));
      },
    });
    const { result } = await renderHook(() => useStartWorkout(), { wrapper });
    await act(() => result.current.mutateAsync().catch(() => {}));
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(calls).toBe(1);
  });
});

describe('show', () => {
  test("puts the answer in its screen's cache and marks every other query stale", () => {
    const queryClient = makeQueryClient();
    queryClient.setQueryData(keys.dashboard(), dashboard);
    queryClient.setQueryData(keys.workout(id), current);
    queryClient.setQueryData(keys.lift('bench'), liftFixture as Lift);
    queryClient.setQueryData(keys.info(), infoFixture);
    show(queryClient, keys.workout(id), amrap);
    expect(queryClient.getQueryData(keys.workout(id))).toEqual(amrap);
    expect(queryClient.getQueryState(keys.workout(id))?.isInvalidated).toBe(false);
    expect(queryClient.getQueryState(keys.dashboard())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(keys.lift('bench'))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(keys.info())?.isInvalidated).toBe(false);
  });

  test('an older fetch still on its way does not land on top of the answer', async () => {
    let release = () => {};
    const { wrapper, queryClient } = setUp({
      [`GET /workouts/${id}`]: () => new Promise((resolve) => (release = () => resolve(current))),
    });
    const { result } = await renderHook(() => useWorkout(id), { wrapper });
    await waitFor(() => expect(result.current.isFetching).toBe(true));
    show(queryClient, keys.workout(id), amrap);
    release();
    await waitFor(() => expect(result.current.data).toEqual(amrap));
    await act(() => jest.advanceTimersByTime(1_000));
    expect(queryClient.getQueryData(keys.workout(id))).toEqual(amrap);
  });

  test('leaves alone a query the change closed', () => {
    const queryClient = makeQueryClient();
    queryClient.setQueryData(keys.workout(id), current);
    show(queryClient, keys.dashboard(), dashboard, keys.workout(id));
    expect(queryClient.getQueryState(keys.workout(id))?.isInvalidated).toBe(false);
  });
});

describe('changes', () => {
  test("logging a set shows the session's answer and starts its rest", async () => {
    const { wrapper, server, queryClient } = setUp({ [`PUT /workouts/${id}/sets/4`]: amrap });
    queryClient.setQueryData(keys.dashboard(), dashboard);
    const { result } = await renderHook(() => ({ log: useLogSet(id), rest: useRest(id) }), { wrapper });
    await act(() => result.current.log.mutateAsync({ idx: 4, reps: 5 }));
    expect(server.sent[0]).toMatchObject({ method: 'PUT', path: `/workouts/${id}/sets/4`, body: { reps: 5 } });
    expect(queryClient.getQueryData(keys.workout(id))).toEqual(amrap);
    expect(queryClient.getQueryState(keys.dashboard())?.isInvalidated).toBe(true);
    expect(result.current.rest).toEqual({ left: amrap.restSeconds, total: amrap.restSeconds });
  });

  test("finishing shows Today and drops the session once it's off screen", async () => {
    const { wrapper, queryClient } = setUp({
      [`GET /workouts/${id}`]: current,
      [`POST /workouts/${id}/finish`]: dashboard,
    });
    queryClient.setQueryData(keys.workout(id), current);
    const { result, unmount } = await renderHook(() => ({ w: useWorkout(id), finish: useFinishWorkout(id) }), {
      wrapper,
    });
    await act(() => result.current.finish.mutateAsync());
    expect(queryClient.getQueryData(keys.dashboard())).toEqual(dashboard);
    expect(queryClient.getQueryData(keys.workout(id))).toBeDefined();
    await unmount();
    expect(queryClient.getQueryData(keys.workout(id))).toBeUndefined();
  });

  test('a 409 refetches the dashboard to show why', async () => {
    const { wrapper, server, queryClient } = setUp({
      'POST /workouts': json({ error: 'Training is waiting on a training max decision.' }, 409),
      'GET /dashboard': dashboard,
    });
    queryClient.setQueryData(keys.dashboard(), dashboard);
    const { result } = await renderHook(() => ({ start: useStartWorkout(), today: useDashboard() }), { wrapper });
    await waitFor(() => expect(server.sent).toHaveLength(1)); // Today's own refetch on mount
    await act(() => result.current.start.mutateAsync().catch(() => {}));
    await waitFor(() =>
      expect(result.current.start.error?.message).toBe('Training is waiting on a training max decision.'),
    );
    await waitFor(() => expect(server.sent.filter((s) => s.path === '/dashboard')).toHaveLength(2));
  });

  test('signing out empties the cache', async () => {
    const { wrapper, queryClient } = setUp({});
    queryClient.setQueryData(keys.dashboard(), dashboard);
    await renderHook(() => useQueryClient(), { wrapper });
    expect(queryClient.getQueryData(keys.dashboard())).toBe(dashboard);
    await act(() => signOut());
    expect(queryClient.getQueryData(keys.dashboard())).toBeUndefined();
  });
});

describe('signing in', () => {
  beforeEach(() => signOut());

  test('checks the server, signs in, and has Today ready before the gate opens', async () => {
    const { wrapper, server, queryClient } = setUp({
      'GET /info': infoFixture,
      'POST /login': { token: 'fresh', expiresAt: '2099-01-01T00:00:00Z' },
      'GET /dashboard': dashboard,
    });
    const { result } = await renderHook(() => useLogin(), { wrapper });
    await act(() => result.current.mutateAsync({ url: 'http://raspberrypi:3000', password: 'dev' }));
    expect(server.sent.map((s) => `${s.method} ${s.path} ${s.auth}`)).toEqual([
      'GET /info null',
      'POST /login null',
      'GET /dashboard Bearer fresh',
    ]);
    expect(server.sent[1].body).toEqual({ password: 'dev' });
    expect(getConnection()).toMatchObject({ signedIn: true, url: 'http://raspberrypi:3000', token: 'fresh' });
    expect(queryClient.getQueryData(keys.dashboard())).toEqual(dashboard);
  });

  test('stops at a server on another API version', async () => {
    const { wrapper, server } = setUp({ 'GET /info': { apiVersion: 2, version: 'next' } });
    const { result } = await renderHook(() => useLogin(), { wrapper });
    await act(() => result.current.mutateAsync({ url: 'http://raspberrypi:3000', password: 'dev' }).catch(() => {}));
    await waitFor(() => expect(result.current.error?.message).toMatch(/Update Kedge/));
    expect(server.sent).toHaveLength(1);
    expect(getConnection().signedIn).toBe(false);
  });

  test("shows the server's refusal as it is", async () => {
    const { wrapper } = setUp({
      'GET /info': infoFixture,
      'POST /login': json({ error: 'Incorrect password.' }, 401),
    });
    const { result } = await renderHook(() => useLogin(), { wrapper });
    await act(() => result.current.mutateAsync({ url: 'http://raspberrypi:3000', password: 'no' }).catch(() => {}));
    await waitFor(() => expect(result.current.error?.message).toBe('Incorrect password.'));
  });
});
