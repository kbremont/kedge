import * as SecureStore from 'expo-secure-store';
import { useSyncExternalStore } from 'react';
import { Appearance } from 'react-native';

// Light or dark: the phone's own setting, unless the lifter picks one in
// Settings. The choice is the phone's, not the server's, so it's kept in the
// keychain beside the connection. React Native's appearance override applies
// it to the whole app, the native tab bar, controls and alerts included, and
// usePalette follows. The launch screen is drawn before any of this runs, so
// it always follows the phone.

export type AppearanceChoice = 'system' | 'light' | 'dark';

export type AppearanceState = {
  choice: AppearanceChoice;
  // Set once the keychain has been read, so the first screen is drawn in
  // the right appearance.
  loaded: boolean;
};

const storeKey = 'kedge.appearance';

let state: AppearanceState = { choice: 'system', loaded: false };
const listeners = new Set<() => void>();

function set(next: Partial<AppearanceState>) {
  state = { ...state, ...next };
  listeners.forEach((l) => l());
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

function apply(choice: AppearanceChoice) {
  Appearance.setColorScheme(choice === 'system' ? 'unspecified' : choice);
}

// Reads the saved choice and applies it. The root layout waits for it behind
// the splash screen with the fonts and the connection. A keychain that can't
// be read leaves the phone's setting.
export async function loadAppearance(): Promise<void> {
  let choice: AppearanceChoice = 'system';
  try {
    const saved = await SecureStore.getItemAsync(storeKey);
    if (saved === 'light' || saved === 'dark') choice = saved;
  } catch {
    // The phone's setting.
  }
  if (choice !== 'system') apply(choice);
  set({ choice, loaded: true });
}

// Applies a choice at once, then keeps it. If the keychain can't be written
// it still holds until the app is closed.
export async function setAppearance(choice: AppearanceChoice): Promise<void> {
  apply(choice);
  set({ choice });
  try {
    await SecureStore.setItemAsync(storeKey, choice);
  } catch {
    // Applied all the same.
  }
}

export function useAppearance(): AppearanceState {
  return useSyncExternalStore(subscribe, () => state);
}
