import { act, renderHook } from '@testing-library/react-native';
import * as SecureStore from 'expo-secure-store';
import { Appearance } from 'react-native';

import { loadAppearance, setAppearance, useAppearance } from '@/lib/appearance';

let setColorScheme: jest.SpyInstance;

beforeEach(async () => {
  setColorScheme = jest.spyOn(Appearance, 'setColorScheme').mockImplementation(() => {});
  await SecureStore.deleteItemAsync('kedge.appearance');
});

afterEach(() => setColorScheme.mockRestore());

test('with nothing kept, the app follows the phone', async () => {
  await loadAppearance();
  const { result } = await renderHook(() => useAppearance());
  expect(result.current).toEqual({ choice: 'system', loaded: true });
  expect(setColorScheme).not.toHaveBeenCalled();
});

test('a kept choice is applied at launch', async () => {
  await SecureStore.setItemAsync('kedge.appearance', 'dark');
  await loadAppearance();
  const { result } = await renderHook(() => useAppearance());
  expect(result.current).toEqual({ choice: 'dark', loaded: true });
  expect(setColorScheme).toHaveBeenCalledWith('dark');
});

test('a pick applies at once and is kept, and System hands back to the phone', async () => {
  await loadAppearance();
  const { result } = await renderHook(() => useAppearance());

  await act(() => setAppearance('light'));
  expect(setColorScheme).toHaveBeenLastCalledWith('light');
  expect(result.current.choice).toBe('light');
  expect(await SecureStore.getItemAsync('kedge.appearance')).toBe('light');

  await act(() => setAppearance('system'));
  expect(setColorScheme).toHaveBeenLastCalledWith('unspecified');
  expect(result.current.choice).toBe('system');
  expect(await SecureStore.getItemAsync('kedge.appearance')).toBe('system');
});

test('a keychain that can’t be read, or holds something else, leaves the phone’s setting', async () => {
  jest.mocked(SecureStore.getItemAsync).mockRejectedValueOnce(new Error('locked'));
  await loadAppearance();
  await SecureStore.setItemAsync('kedge.appearance', 'sepia');
  await loadAppearance();
  const { result } = await renderHook(() => useAppearance());
  expect(result.current).toEqual({ choice: 'system', loaded: true });
  expect(setColorScheme).not.toHaveBeenCalled();
});

test('a pick holds even when the keychain can’t be written', async () => {
  jest.mocked(SecureStore.setItemAsync).mockRejectedValueOnce(new Error('locked'));
  const { result } = await renderHook(() => useAppearance());
  await act(() => setAppearance('dark'));
  expect(setColorScheme).toHaveBeenLastCalledWith('dark');
  expect(result.current.choice).toBe('dark');
  await act(() => setAppearance('system'));
});
