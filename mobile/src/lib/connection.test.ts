import * as SecureStore from 'expo-secure-store';

import {
  checkApiVersion,
  getConnection,
  loadConnection,
  normalizeServerUrl,
  saveConnection,
  sessionEnded,
  signOut,
} from '@/lib/connection';

const later = '2099-01-01T00:00:00Z';

describe('normalizeServerUrl', () => {
  test.each([
    ['lift.lyle.sh', 'https://lift.lyle.sh'],
    ['raspberrypi:3000', 'http://raspberrypi:3000'],
    ['localhost:3000', 'http://localhost:3000'],
    ['192.168.1.20:3000', 'http://192.168.1.20:3000'],
    ['nas.local', 'http://nas.local'],
    ['[::1]:3000', 'http://[::1]:3000'],
    ['fe80::1', 'http://[fe80::1]'],
    ['raspberrypi.tail1e2f.ts.net', 'https://raspberrypi.tail1e2f.ts.net'],
    ['https://lift.lyle.sh/api/v1/', 'https://lift.lyle.sh'],
    ['raspberrypi:3000/api/v1', 'http://raspberrypi:3000'],
    ['https://lift.lyle.sh/', 'https://lift.lyle.sh'],
    ['HTTPS://Lift.Lyle.SH', 'https://lift.lyle.sh'],
    ['  lift.lyle.sh \n', 'https://lift.lyle.sh'],
    ['http://raspberrypi:3000', 'http://raspberrypi:3000'],
    ['https://raspberrypi:3000', 'https://raspberrypi:3000'],
    ['https://example.com/Kedge/?x=1', 'https://example.com/Kedge'],
  ])('%j is %s', (input, url) => {
    expect(normalizeServerUrl(input)).toEqual({ url });
  });

  test('refuses plain http to an address off the local network', () => {
    expect(normalizeServerUrl('http://lift.lyle.sh')).toEqual({
      error: 'Use https:// for lift.lyle.sh. Kedge sends plain http:// only to addresses on your own network.',
    });
  });

  test.each(['', '   ', 'ftp://lift.lyle.sh', 'lift lyle.sh', 'raspberrypi:99999', 'http:/lift.lyle.sh', '300.1.1.1'])(
    'refuses %j',
    (input) => {
      expect(normalizeServerUrl(input)).toHaveProperty('error');
    },
  );
});

describe('checkApiVersion', () => {
  test('an older server says to update the server', () => {
    expect(checkApiVersion({ apiVersion: 0, version: 'v0' })).toMatch(/Update the server/);
  });

  test('the same version passes', () => {
    expect(checkApiVersion({ apiVersion: 1, version: 'v1' })).toBeNull();
  });

  test('a newer server says to update Kedge', () => {
    expect(checkApiVersion({ apiVersion: 2, version: 'v2' })).toMatch(/Update Kedge/);
  });
});

describe('the saved connection', () => {
  beforeEach(async () => {
    await SecureStore.deleteItemAsync('kedge.connection');
  });

  test('starts signed out, once loaded', async () => {
    expect(getConnection().loaded).toBe(false);
    await loadConnection();
    expect(getConnection()).toMatchObject({ loaded: true, signedIn: false, url: '', token: '' });
  });

  test('is saved to the keychain', async () => {
    await saveConnection({ url: 'https://lift.lyle.sh', token: 'tok', expiresAt: later });
    expect(getConnection()).toMatchObject({ signedIn: true, url: 'https://lift.lyle.sh', token: 'tok' });
    expect(JSON.parse((await SecureStore.getItemAsync('kedge.connection')) ?? '')).toEqual({
      url: 'https://lift.lyle.sh',
      token: 'tok',
      expiresAt: later,
    });
  });

  test('loads what the last launch saved', async () => {
    await SecureStore.setItemAsync(
      'kedge.connection',
      JSON.stringify({ url: 'https://lift.lyle.sh', token: 'saved', expiresAt: later }),
    );
    await loadConnection();
    expect(getConnection()).toMatchObject({ loaded: true, signedIn: true, token: 'saved', reason: '' });
  });

  test('signing out keeps the address and the reason', async () => {
    await saveConnection({ url: 'http://raspberrypi:3000', token: 'tok', expiresAt: later });
    await signOut(sessionEnded);
    expect(getConnection()).toMatchObject({
      signedIn: false,
      token: '',
      url: 'http://raspberrypi:3000',
      reason: sessionEnded,
    });
    await loadConnection();
    expect(getConnection()).toMatchObject({ signedIn: false, url: 'http://raspberrypi:3000', reason: '' });
  });

  test('a session that ran out while closed opens signed out, saying why', async () => {
    await SecureStore.setItemAsync(
      'kedge.connection',
      JSON.stringify({ url: 'http://raspberrypi:3000', token: 'tok', expiresAt: '2020-01-01T00:00:00Z' }),
    );
    await loadConnection();
    expect(getConnection()).toMatchObject({
      loaded: true,
      signedIn: false,
      token: '',
      url: 'http://raspberrypi:3000',
      reason: sessionEnded,
    });
  });

  test('an expiry in the past is signed out', async () => {
    await saveConnection({ url: 'http://raspberrypi:3000', token: 'tok', expiresAt: '2020-01-01T00:00:00Z' });
    expect(getConnection().signedIn).toBe(false);
  });

  test('a keychain that cannot be read loads as empty', async () => {
    jest.mocked(SecureStore.getItemAsync).mockRejectedValueOnce(new Error('locked'));
    await loadConnection();
    expect(getConnection()).toMatchObject({ loaded: true, signedIn: false, url: '' });
  });
});
