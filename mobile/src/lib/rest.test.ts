import { act, renderHook } from '@testing-library/react-native';

import { clearRest, startRest, useRest } from '@/lib/rest';

beforeEach(() => {
  jest.useFakeTimers();
  clearRest(7);
  clearRest(8);
});

afterEach(() => jest.useRealTimers());

test('a session opened fresh has no rest', async () => {
  const { result } = await renderHook(() => useRest(7));
  expect(result.current).toEqual({ left: 0, total: 0 });
});

test('counts down from the rest the server gave', async () => {
  const { result } = await renderHook(() => useRest(7));
  await act(() => startRest(7, 90));
  expect(result.current).toEqual({ left: 90, total: 90 });
  await act(() => jest.advanceTimersByTime(30_000));
  expect(result.current).toEqual({ left: 60, total: 90 });
});

test('runs from the tap, not from when the answer came', async () => {
  const tapped = Date.now();
  jest.advanceTimersByTime(2_000);
  const { result } = await renderHook(() => useRest(7));
  await act(() => startRest(7, 90, tapped));
  expect(result.current.left).toBe(88);
});

test('stays right across a sleep that stopped the timers', async () => {
  const { result } = await renderHook(() => useRest(7));
  await act(() => startRest(7, 180));
  // The clock moves on while no timer fires, as when the phone sleeps.
  jest.setSystemTime(Date.now() + 100_000);
  await act(() => jest.advanceTimersByTime(250));
  expect(result.current.left).toBe(80);
});

test('is kept when the session is left and opened again', async () => {
  const first = await renderHook(() => useRest(7));
  await act(() => startRest(7, 120));
  await first.unmount();
  jest.advanceTimersByTime(45_000);
  const { result } = await renderHook(() => useRest(7));
  expect(result.current.left).toBe(75);
});

test('undo clears it, and other sessions keep theirs', async () => {
  const { result } = await renderHook(() => [useRest(7), useRest(8)]);
  await act(() => {
    startRest(7, 90);
    startRest(8, 60);
  });
  await act(() => clearRest(7));
  expect(result.current).toEqual([
    { left: 0, total: 0 },
    { left: 60, total: 60 },
  ]);
});

test('stops ticking once it runs out', async () => {
  let renders = 0;
  const { result } = await renderHook(() => {
    renders++;
    return useRest(7);
  });
  await act(() => startRest(7, 5));
  await act(() => jest.advanceTimersByTime(6_000));
  expect(result.current).toEqual({ left: 0, total: 5 });
  const settled = renders;
  await act(() => jest.advanceTimersByTime(10_000));
  expect(renders).toBe(settled);
});
