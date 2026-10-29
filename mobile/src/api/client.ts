import createClient, { type Client, type Middleware } from 'openapi-fetch';

import type { paths } from '@/api/schema.gen';
import { getConnection, sessionEnded, signOut } from '@/lib/connection';

// Requests to the lifter's server, typed by the API's spec. Every failure
// comes back as an ApiError whose message can be shown as it is.

export type ApiClient = Client<paths>;
export type Fetch = (request: Request) => Promise<Response>;

export class ApiError extends Error {
  // The answer's HTTP status, or 0 when none came.
  readonly status: number;
  // Set when what answered wasn't a Kedge server, such as another app, a web
  // page or a proxy's error.
  readonly notKedge: boolean;

  constructor(status: number, message: string, notKedge = false) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.notKedge = notKedge;
  }
}

export const unreachable = 'Couldn’t reach your server. Check that it’s running and that you’re on its network.';

// How long a request waits for an answer, so a stalled connection fails a
// set instead of hanging it.
const timeoutMs = 15_000;

export function makeClient(
  { url, token = '' }: { url: string; token?: string },
  { fetch = globalThis.fetch }: { fetch?: Fetch } = {},
): ApiClient {
  const client = createClient<paths>({ baseUrl: `${url}/api/v1`, fetch: withTimeout(fetch) });
  client.use(middleware(url, token));
  return client;
}

// The middleware throws for anything but a 2xx, so a request that returns
// has its data.
export async function answer<T>(request: Promise<{ data?: T }>): Promise<T> {
  return (await request).data as T;
}

// The database as the server snapshots it, through the same client as every
// other request so it is signed in and refused the same way. The server names
// the file after the day.
export async function fetchBackup(client: ApiClient): Promise<{ bytes: ArrayBuffer; name: string }> {
  const { data, response } = await client.GET('/backup', { parseAs: 'arrayBuffer' });
  const disposition = response.headers.get('Content-Disposition') ?? '';
  const name = /filename="?([^";]+)"?/.exec(disposition)?.[1] ?? '531-backup.db';
  return { bytes: data as ArrayBuffer, name };
}

// Gives up on a request with no answer in time. The caller's own signal,
// React Query's when it cancels, still aborts it too. The clock stops once
// the headers are in: a body follows quickly, and a backup's can take longer
// than the timeout to arrive.
function withTimeout(fetch: Fetch): Fetch {
  return async (request) => {
    const controller = new AbortController();
    const abort = () => controller.abort();
    if (request.signal.aborted) abort();
    request.signal.addEventListener('abort', abort);
    const timer = setTimeout(abort, timeoutMs);
    try {
      return await fetch(new Request(request, { signal: controller.signal }));
    } finally {
      clearTimeout(timer);
      request.signal.removeEventListener('abort', abort);
    }
  };
}

function middleware(url: string, token: string): Middleware {
  const notKedge = (status: number) =>
    new ApiError(status, `No Kedge server answered at ${url.replace(/^https?:\/\//, '')}. Check the address.`, true);

  return {
    onRequest({ request }) {
      if (token) request.headers.set('Authorization', `Bearer ${token}`);
    },

    async onResponse({ response, schemaPath, options }) {
      if (response.ok) {
        // A 2xx that isn't JSON, such as a web page at that address.
        if (options.parseAs === 'json' && !response.headers.get('Content-Type')?.includes('json')) {
          throw notKedge(response.status);
        }
        return;
      }
      // The server no longer knows the token, so the gate closes and Connect
      // says why. A wrong password at sign-in is only a refusal, and a 401
      // for a session the lifter has since replaced mustn't end the new one.
      if (response.status === 401 && schemaPath !== '/login' && token && getConnection().token === token) {
        void signOut(sessionEnded);
      }
      // The answer goes no further than this throw, so its body is read here
      // and not cloned. The server's {error} is written to be shown as it is.
      const body: unknown = await response.json().catch(() => null);
      if (body && typeof body === 'object' && 'error' in body && typeof body.error === 'string') {
        throw new ApiError(response.status, body.error);
      }
      throw notKedge(response.status);
    },

    // No answer at all: the network failed, or the timeout gave up.
    onError() {
      return new ApiError(0, unreachable);
    },
  };
}
