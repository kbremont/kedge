import errorInput from '@/api/fixtures/error-input.json';
import { ApiError, fetchBackup, makeClient, unreachable, type Fetch } from '@/api/client';
import { getConnection, saveConnection, sessionEnded } from '@/lib/connection';
import { json } from '@/test-utils';

const url = 'http://raspberrypi:3000';
const later = '2099-01-01T00:00:00Z';

// Answers every request with `reply`, keeping the requests.
function answering(reply: () => Response | Promise<Response>) {
  const requests: Request[] = [];
  const fetch: Fetch = async (request) => {
    requests.push(request);
    return reply();
  };
  return { fetch, requests };
}

async function failure(request: Promise<unknown>): Promise<ApiError> {
  const error = await request.then(
    () => null,
    (e: unknown) => e,
  );
  expect(error).toBeInstanceOf(ApiError);
  return error as ApiError;
}

test('sends the token as a Bearer header, under /api/v1', async () => {
  const server = answering(() => json({ items: [], lifts: [] }));
  const client = makeClient({ url, token: 'tok' }, { fetch: server.fetch });
  const { data } = await client.GET('/history');
  expect(data).toEqual({ items: [], lifts: [] });
  expect(server.requests[0].url).toBe('http://raspberrypi:3000/api/v1/history');
  expect(server.requests[0].headers.get('Authorization')).toBe('Bearer tok');
});

test('sends no Authorization before signing in', async () => {
  const server = answering(() => json({ apiVersion: 1, version: 'v1' }));
  await makeClient({ url }, { fetch: server.fetch }).GET('/info');
  expect(server.requests[0].headers.get('Authorization')).toBeNull();
});

test("a refusal carries the server's message as it is", async () => {
  const server = answering(() => json(errorInput, 400));
  const client = makeClient({ url, token: 'tok' }, { fetch: server.fetch });
  const error = await failure(
    client.POST('/lifts/{slug}/estimates', {
      params: { path: { slug: 'bench' } },
      body: { date: '2099-01-01', weight: 200, reps: 5 },
    }),
  );
  expect(error).toMatchObject({ status: 400, message: errorInput.error, notKedge: false });
});

test('a 401 signs out but keeps the address', async () => {
  await saveConnection({ url, token: 'tok', expiresAt: later });
  const server = answering(() => json({ error: 'Sign in to continue.' }, 401));
  const error = await failure(makeClient({ url, token: 'tok' }, { fetch: server.fetch }).GET('/dashboard'));
  expect(error).toMatchObject({ status: 401, message: 'Sign in to continue.' });
  expect(getConnection()).toMatchObject({ signedIn: false, token: '', url, reason: sessionEnded });
});

test('a wrong password at sign-in does not sign out', async () => {
  await saveConnection({ url, token: 'tok', expiresAt: later });
  const server = answering(() => json({ error: 'Incorrect password.' }, 401));
  const error = await failure(
    makeClient({ url, token: 'tok' }, { fetch: server.fetch }).POST('/login', { body: { password: 'no' } }),
  );
  expect(error.message).toBe('Incorrect password.');
  expect(getConnection()).toMatchObject({ signedIn: true, token: 'tok' });
});

test("a 401 for a session already replaced doesn't end the new one", async () => {
  await saveConnection({ url, token: 'new', expiresAt: later });
  const server = answering(() => json({ error: 'Sign in to continue.' }, 401));
  await failure(makeClient({ url, token: 'old' }, { fetch: server.fetch }).GET('/dashboard'));
  expect(getConnection()).toMatchObject({ signedIn: true, token: 'new' });
});

test('a network failure is status 0', async () => {
  const server = answering(() => Promise.reject(new TypeError('Network request failed')));
  const error = await failure(makeClient({ url }, { fetch: server.fetch }).GET('/info'));
  expect(error).toMatchObject({ status: 0, message: unreachable });
});

describe('with a server that never answers', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  test('gives up after 15 seconds, as status 0', async () => {
    const fetch: Fetch = (request) =>
      new Promise((_, reject) => request.signal.addEventListener('abort', () => reject(new Error('aborted'))));
    const settled = jest.fn();
    const request = failure(makeClient({ url }, { fetch }).GET('/info')).then((e) => {
      settled();
      return e;
    });
    await jest.advanceTimersByTimeAsync(14_900);
    expect(settled).not.toHaveBeenCalled();
    await jest.advanceTimersByTimeAsync(100);
    expect(await request).toMatchObject({ status: 0, message: unreachable });
  });
});

test('something other than a Kedge server is told apart', async () => {
  const page = () => new Response('<html>Not Found</html>', { status: 404, headers: { 'Content-Type': 'text/html' } });
  const error = await failure(makeClient({ url }, { fetch: answering(page).fetch }).GET('/info'));
  expect(error).toMatchObject({
    status: 404,
    notKedge: true,
    message: 'No Kedge server answered at raspberrypi:3000. Check the address.',
  });

  // A web page that answers 200 is no better.
  const ok = () => new Response('<html></html>', { status: 200, headers: { 'Content-Type': 'text/html' } });
  expect(await failure(makeClient({ url }, { fetch: answering(ok).fetch }).GET('/info'))).toMatchObject({
    status: 200,
    notKedge: true,
  });
});

// tailscale serve's answer while the server behind it is down.
const proxy = (status: number) => () =>
  new Response(`${status} Bad Gateway`, { status, headers: { 'Content-Type': 'text/plain; charset=utf-8' } });

test.each([502, 503, 504])('a proxy answering %i for a server that is down is the server out of reach', async (s) => {
  const error = await failure(makeClient({ url }, { fetch: answering(proxy(s)).fetch }).GET('/dashboard'));
  expect(error).toMatchObject({ status: 0, message: unreachable, notKedge: false });
});

test("a 5xx from Kedge itself keeps the server's message", async () => {
  const server = answering(() => json({ error: 'Something went wrong. Check the server log.' }, 502));
  const error = await failure(makeClient({ url }, { fetch: server.fetch }).GET('/dashboard'));
  expect(error).toMatchObject({ status: 502, message: 'Something went wrong. Check the server log.' });
});

test.each([
  ['an empty body', '', { 'Content-Length': '0' }],
  ['an empty body without a length', '', {}],
  ['a null', 'null', {}],
  ['JSON cut short', '{"apiVersion": 1, "vers', {}],
])('a 2xx JSON answer with %s is not a Kedge server', async (_, body, headers) => {
  const reply = () => new Response(body, { headers: { 'Content-Type': 'application/json', ...headers } });
  const error = await failure(makeClient({ url }, { fetch: answering(reply).fetch }).GET('/info'));
  expect(error).toMatchObject({
    status: 200,
    notKedge: true,
    message: 'No Kedge server answered at raspberrypi:3000. Check the address.',
  });
});

test('a JSON answer reaches the caller whole after its check', async () => {
  const dashboard = { needsSetup: false, unit: 'lb', name: 'Café ×' };
  const { data } = await makeClient({ url }, { fetch: answering(() => json(dashboard)).fetch }).GET('/dashboard');
  expect(data).toEqual(dashboard);
});

test('a body that stops arriving is the server out of reach', async () => {
  const cut = () =>
    new Response(new ReadableStream({ pull: (c) => c.error(new TypeError('The network connection was lost.')) }), {
      headers: { 'Content-Type': 'application/json' },
    });
  const error = await failure(makeClient({ url }, { fetch: answering(cut).fetch }).GET('/info'));
  expect(error).toMatchObject({ status: 0, message: unreachable });
});

test('a backup comes through the middleware as bytes, with its file name', async () => {
  const bytes = new Uint8Array([83, 81, 76, 105, 116, 101]);
  const server = answering(
    () =>
      new Response(bytes, {
        headers: {
          'Content-Type': 'application/octet-stream',
          'Content-Disposition': 'attachment; filename="531-2026-10-28.db"',
        },
      }),
  );
  const backup = await fetchBackup(makeClient({ url, token: 'tok' }, { fetch: server.fetch }));
  expect(new Uint8Array(backup.bytes)).toEqual(bytes);
  expect(backup.name).toBe('531-2026-10-28.db');
  expect(server.requests[0].headers.get('Authorization')).toBe('Bearer tok');
});

test('a backup refused is an ApiError like any other', async () => {
  await saveConnection({ url, token: 'tok', expiresAt: later });
  const server = answering(() => json({ error: 'Sign in to continue.' }, 401));
  expect(await failure(fetchBackup(makeClient({ url, token: 'tok' }, { fetch: server.fetch })))).toMatchObject({
    status: 401,
  });
});
