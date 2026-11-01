import type { QueryClient } from '@tanstack/react-query';
import { renderRouter, type MockContextConfig, type RenderRouterOptions } from 'expo-router/testing-library';
import type { ReactNode } from 'react';

import { ApiProvider, makeQueryClient } from '@/api/queries';
import { saveConnection, signOut } from '@/lib/connection';

// For component tests: a stand-in for the server that answers from the golden
// fixtures and records what it was sent, and routes rendered signed in to it.

// An address Connect accepts as it is, so a test can sign in to it again.
export const testUrl = 'http://kedge.local';
export const testToken = 'test-token';

// A request as the stand-in server saw it, its path without /api/v1.
export type Sent = { method: string; path: string; body: unknown; auth: string | null };

// A fixture is sent as JSON with a 200, a Response as it is. A function
// answers from the request, or later: return a promise the test settles to
// hold a request in flight.
type Reply = Response | object;
export type Routes = Record<string, Reply | ((sent: Sent) => Reply | Promise<Reply>)>;

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

// Answers 'METHOD /path' from `routes`, and a 404 for anything else.
export function fakeServer(routes: Routes) {
  const sent: Sent[] = [];
  async function fetch(request: Request): Promise<Response> {
    const path = request.url.replace(/^\w+:\/\/[^/]+\/api\/v1/, '').replace(/[?#].*$/, '');
    const text = await request.text();
    const s: Sent = {
      method: request.method,
      path,
      body: text ? JSON.parse(text) : undefined,
      auth: request.headers.get('Authorization'),
    };
    sent.push(s);
    const route = routes[`${request.method} ${path}`];
    const reply = typeof route === 'function' ? await route(s) : route;
    if (reply === undefined) return json({ error: 'Not found.' }, 404);
    return reply instanceof Response ? reply : json(reply);
  }
  return { fetch, sent };
}

// Renders `routes` against `server`, signed in unless `signedIn` is false,
// with `queryClient`'s cache when the test seeds one. renderRouter turns on
// Jest's fake timers.
export async function renderApp(
  routes: MockContextConfig,
  {
    server,
    queryClient = makeQueryClient(),
    signedIn = true,
    ...options
  }: RenderRouterOptions & { server: ReturnType<typeof fakeServer>; queryClient?: QueryClient; signedIn?: boolean },
) {
  if (signedIn) await saveConnection({ url: testUrl, token: testToken, expiresAt: '2099-01-01T00:00:00Z' });
  else await signOut();
  return renderRouter(routes, {
    ...options,
    wrapper: ({ children }: { children: ReactNode }) => (
      <ApiProvider fetch={server.fetch} queryClient={queryClient}>
        {children}
      </ApiProvider>
    ),
  });
}
