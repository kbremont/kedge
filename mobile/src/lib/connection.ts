import * as SecureStore from 'expo-secure-store';
import { useSyncExternalStore } from 'react';

import type { Info } from '@/api/schema.gen';

// The lifter's server and the session the app holds there. One keychain item
// keeps both, read into memory at launch, so requests and the sign-in gate
// never wait on the keychain.

export type Connection = {
  // The server's address without /api/v1, as normalizeServerUrl writes it.
  url: string;
  // The session token, or '' once signed out.
  token: string;
  // When the token stops working, in RFC 3339.
  expiresAt: string;
};

export type ConnectionState = Connection & {
  // Set once the keychain has been read. Until then nobody can say whether
  // the lifter is signed in.
  loaded: boolean;
  // A token that hasn't expired.
  signedIn: boolean;
  // Why the app signed out by itself, in a sentence Connect can show as it
  // is; '' after the lifter signs out, or before they ever sign in.
  reason: string;
};

// What Connect says once the server stops taking the session's token.
export const sessionEnded = 'Your session ended. Enter the server’s password to connect again.';

const storeKey = 'kedge.connection';
const none: Connection = { url: '', token: '', expiresAt: '' };

let state: ConnectionState = { ...none, loaded: false, signedIn: false, reason: '' };
const listeners = new Set<() => void>();

function set(next: Partial<ConnectionState>) {
  const merged = { ...state, ...next };
  // A token past its expiry is as good as none.
  state = { ...merged, signedIn: merged.token !== '' && Date.parse(merged.expiresAt) > Date.now() };
  listeners.forEach((l) => l());
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

// Reads the saved connection into memory. The root layout awaits it beside
// the fonts and keeps the splash screen up until both are in. A keychain that
// can't be read counts as empty: Connect asks for the server again.
export async function loadConnection(): Promise<void> {
  let saved = none;
  try {
    const raw = await SecureStore.getItemAsync(storeKey);
    if (raw) saved = { ...none, ...(JSON.parse(raw) as Partial<Connection>) };
  } catch {
    // Read as empty.
  }
  set({ ...saved, loaded: true, reason: '' });
  // A session that ran out while the app was closed opens Connect with the
  // address filled in, saying why.
  if (state.token && !state.signedIn) await signOut(sessionEnded);
}

export function useConnection(): ConnectionState {
  return useSyncExternalStore(subscribe, () => state);
}

// The same, for code outside React.
export function getConnection(): ConnectionState {
  return state;
}

// Keeps a new session. It is saved before the gate opens, so a launch right
// after signing in finds it.
export async function saveConnection(c: Connection): Promise<void> {
  await SecureStore.setItemAsync(storeKey, JSON.stringify(c));
  set({ ...c, reason: '' });
}

// Forgets the token but keeps the address, so Connect can fill it in. The
// gate closes at once; the keychain catches up after. If it can't be written
// there is nothing the lifter could do about it, and the token it still
// holds runs out by itself.
export async function signOut(reason = ''): Promise<void> {
  set({ token: '', expiresAt: '', reason });
  try {
    await SecureStore.setItemAsync(storeKey, JSON.stringify({ ...none, url: state.url }));
  } catch {
    // Signed out in memory all the same.
  }
}

const ipv4 = /^\d+(\.\d+){3}$/;
const hostname = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$/;

// Plain http:// goes only to an address on the lifter's own network, which is
// what iOS's NSAllowsLocalNetworking lets through: a short name such as
// raspberrypi, a .local name, localhost, or an IP address. Anything else
// needs HTTPS.
function isLocal(host: string): boolean {
  return !host.includes('.') || host.endsWith('.local') || host.startsWith('[') || ipv4.test(host);
}

const notAnAddress = 'That isn’t a server address. Try one such as raspberrypi:3000 or lift.example.com.';

// Turns what the lifter typed into the server's address: spaces trimmed, the
// host lowercased, a trailing / or pasted /api/v1 dropped, and http:// or
// https:// added when there is no scheme.
export function normalizeServerUrl(input: string): { url: string } | { error: string } {
  const text = input.trim();
  if (!text) return { error: 'Enter your server’s address.' };

  const scheme = /^([a-z][a-z0-9+.-]*):\/\//i.exec(text)?.[1].toLowerCase();
  if (scheme && scheme !== 'http' && scheme !== 'https') return { error: 'Use an http:// or https:// address.' };
  const rest = scheme ? text.slice(scheme.length + 3) : text;
  // The address ends where its path begins, and anything after a ? or # is
  // dropped.
  const [, authority, path] = /^([^/?#]*)([^?#]*)/.exec(rest) ?? ['', '', ''];

  let host = authority.toLowerCase();
  let port = '';
  if ((host.match(/:/g) ?? []).length > 1 && !host.startsWith('[')) {
    // An IPv6 address typed without brackets can't carry a port.
    host = `[${host}]`;
  } else {
    const m = /^(.*?)(?::(\d+))?$/.exec(host);
    host = m?.[1] ?? '';
    port = m?.[2] ?? '';
  }
  const valid = host.startsWith('[')
    ? /^\[[0-9a-f:.]+\]$/.test(host)
    : hostname.test(host) && (!ipv4.test(host) || host.split('.').every((n) => Number(n) <= 255));
  if (!valid || (port && !(Number(port) >= 1 && Number(port) <= 65535))) return { error: notAnAddress };

  const local = isLocal(host);
  if (scheme === 'http' && !local) {
    return { error: `Use https:// for ${host}. Kedge sends plain http:// only to addresses on your own network.` };
  }
  const base = path.replace(/\/+$/, '').replace(/\/api\/v1$/i, '').replace(/\/+$/, '');
  return { url: `${scheme ?? (local ? 'http' : 'https')}://${host}${port ? `:${port}` : ''}${base}` };
}

// The API version this build of the app speaks, the server's APIVersion.
export const supportedApiVersion = 1;

// Says what to update when the server and the app speak different versions,
// or null when they match. A server that reports no version reads as older.
export function checkApiVersion({ apiVersion }: Info): string | null {
  if (apiVersion > supportedApiVersion) {
    return 'This version of Kedge is too old for your server. Update Kedge, then connect again.';
  }
  if (!(apiVersion >= supportedApiVersion)) {
    return 'Your server is too old for this version of Kedge. Update the server, then connect again.';
  }
  return null;
}
