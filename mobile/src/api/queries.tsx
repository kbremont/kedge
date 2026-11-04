import {
  focusManager,
  hashKey,
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQuery,
  useQueryClient,
  type DataTag,
  type QueryFunction,
  type QueryKey,
} from '@tanstack/react-query';
import createFetchClient from 'openapi-fetch';
import createQueryClient, { type OpenapiQueryClient } from 'openapi-react-query';
import { createContext, use, useEffect, useMemo, useState, type ReactNode } from 'react';
import { AppState } from 'react-native';

import { answer, ApiError, makeClient, type ApiClient, type Fetch } from '@/api/client';
import type {
  PastEstimateInput,
  paths,
  SettingsInput,
  SetupHistoryInput,
  SetupInput,
  TMReviewInput,
  TMTestDecision,
  TrainingMaxInput,
} from '@/api/schema.gen';
import { checkApiVersion, saveConnection, useConnection } from '@/lib/connection';
import { clearRest, startRest } from '@/lib/rest';

// The screens' data: one read per screen, cached by React Query, and one
// change per write the API offers. Every change answers with the screen it
// leads to, which goes straight into that screen's cache.

// What each screen reads, as openapi-react-query's options for a given $api.
// The hooks fetch with the connection's; `keys` names the same queries with
// one that never fetches, so the cache can be reached without a connection.
const reads = ($api: OpenapiQueryClient<paths>) => ({
  info: () => $api.queryOptions('get', '/info'),
  dashboard: () => $api.queryOptions('get', '/dashboard'),
  workout: (id: number) => $api.queryOptions('get', '/workouts/{id}', { params: { path: { id } } }),
  plates: (weight: number) => $api.queryOptions('get', '/plates', { params: { query: { weight } } }),
  history: () => $api.queryOptions('get', '/history'),
  lift: (slug: string) => $api.queryOptions('get', '/lifts/{slug}', { params: { path: { slug } } }),
  settings: () => $api.queryOptions('get', '/settings'),
  setup: () => $api.queryOptions('get', '/setup'),
  setupPosition: () => $api.queryOptions('get', '/setup/position'),
  setupHistory: (cycle: number) =>
    $api.queryOptions('get', '/setup/history/{cycle}', { params: { path: { cycle } } }),
});

type Reads = ReturnType<typeof reads>;

// A read's key, [method, path] or [method, path, init], tagged with its data
// so setQueryData and getQueryData are typed by it.
function tag<D, K extends QueryKey>({ queryKey }: { queryKey: K; queryFn: QueryFunction<D, K, never> }) {
  return queryKey as DataTag<K, D>;
}

const named = reads(createQueryClient(createFetchClient<paths>()));

// The cache key of each screen's read. Path ids are numbers, as the API's
// are: a route converts its string param once, or the key won't match.
export const keys = {
  info: () => tag(named.info()),
  dashboard: () => tag(named.dashboard()),
  workout: (id: number) => tag(named.workout(id)),
  plates: (weight: number) => tag(named.plates(weight)),
  history: () => tag(named.history()),
  lift: (slug: string) => tag(named.lift(slug)),
  settings: () => tag(named.settings()),
  setup: () => tag(named.setup()),
  setupPosition: () => tag(named.setupPosition()),
  setupHistory: (cycle: number) => tag(named.setupHistory(cycle)),
};

// Puts a change's answer straight into its screen's cache, then marks every
// other query stale: those on screen refetch now, the rest when they next
// open. A fetch of the same screen still on its way could land on top of the
// answer, so it is cancelled. `left` is a query the change closed, such as a
// finished session: it isn't refetched while its screen goes back. The
// server's info only changes with its build, so changes leave it alone.
export function show<D>(queryClient: QueryClient, key: DataTag<QueryKey, D>, data: D, left?: QueryKey) {
  void queryClient.cancelQueries({ queryKey: key, exact: true });
  queryClient.setQueryData(key, data);
  const skip = [hashKey(key), hashKey(keys.info())];
  if (left) {
    skip.push(hashKey(left));
    void queryClient.cancelQueries({ queryKey: left, exact: true });
  }
  void queryClient.invalidateQueries({ predicate: (q) => !skip.includes(q.queryHash) });
}

// Removes a query once no screen shows it, so a session finished or discarded
// is fetched fresh if it is ever opened again.
function dropWhenUnwatched(queryClient: QueryClient, key: QueryKey) {
  const cache = queryClient.getQueryCache();
  const hash = hashKey(key);
  const drop = () => {
    const q = cache.get(hash);
    if (q?.getObserversCount()) return false;
    if (q) cache.remove(q);
    return true;
  };
  if (drop()) return;
  const stop = cache.subscribe((e) => {
    if (e.type === 'observerRemoved' && e.query.queryHash === hash && drop()) stop();
  });
}

// The app's cache. A query asks once more, and only when the server couldn't
// be reached: a refusal would only be refused again. A change is never resent
// by itself; the lifter sees it failed and decides.
export function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: (failures, error) => failures < 1 && error instanceof ApiError && error.status === 0 },
      mutations: { retry: false },
    },
  });
}

type Api = {
  // Requests for the current connection.
  client: ApiClient;
  // A client for another address or token, through the same fetch: signing
  // in talks to the server before there is a connection to save.
  clientFor: (c: { url: string; token?: string }) => ApiClient;
  // Each screen's read through `client`, for the hooks below.
  read: Reads;
};

const ApiContext = createContext<Api | null>(null);

export function useApi(): Api {
  const api = use(ApiContext);
  if (!api) throw new Error('useApi needs an ApiProvider above it.');
  return api;
}

// The data layer for everything under it, rebuilt for each connection.
// `fetch` and `queryClient` let a test stand in for the server and start
// from a seeded cache.
export function ApiProvider({
  children,
  fetch,
  queryClient,
}: {
  children: ReactNode;
  fetch?: Fetch;
  queryClient?: QueryClient;
}) {
  const [cache] = useState(() => queryClient ?? makeQueryClient());
  const { url, token, signedIn } = useConnection();
  const api = useMemo(() => {
    const clientFor = (c: { url: string; token?: string }) => makeClient(c, { fetch });
    const client = clientFor({ url, token });
    return { client, clientFor, read: reads(createQueryClient(client)) };
  }, [url, token, fetch]);

  // Signing out forgets every screen, so nothing of the old session shows
  // through to the next.
  useEffect(() => {
    if (!signedIn) cache.clear();
  }, [signedIn, cache]);

  // The screens on show refetch when the app comes back to the foreground.
  useEffect(() => {
    const sub = AppState.addEventListener('change', (s) => focusManager.setFocused(s === 'active'));
    return () => sub.remove();
  }, []);

  return (
    <QueryClientProvider client={cache}>
      <ApiContext value={api}>{children}</ApiContext>
    </QueryClientProvider>
  );
}

// openapi-react-query types a failure as the server's error body, but the
// client's middleware throws an ApiError in its place, so only the key and
// the fetch are passed on and the error stays an Error.
function useRead<D, K extends QueryKey>({ queryKey, queryFn }: { queryKey: K; queryFn: QueryFunction<D, K, never> }) {
  return useQuery({ queryKey, queryFn });
}

export function useInfo() {
  return useRead(useApi().read.info());
}

export function useDashboard() {
  return useRead(useApi().read.dashboard());
}

export function useWorkout(id: number) {
  return useRead(useApi().read.workout(id));
}

export function useHistory() {
  return useRead(useApi().read.history());
}

export function useLift(slug: string) {
  return useRead(useApi().read.lift(slug));
}

export function useSettings() {
  return useRead(useApi().read.settings());
}

export function useSetup() {
  return useRead(useApi().read.setup());
}

export function useSetupPosition() {
  return useRead(useApi().read.setupPosition());
}

export function useSetupHistory(cycle: number) {
  return useRead(useApi().read.setupHistory(cycle));
}

// A change to the server. Once it succeeds, `done` puts its answer in the
// cache by way of show. A 409 means training is held or the offer has
// passed, so the dashboard refetches to show why.
function useChange<V = void, D = unknown>(
  send: (client: ApiClient, vars: V) => Promise<D>,
  done: (queryClient: QueryClient, answer: D) => void,
) {
  const { client } = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: V) => send(client, vars),
    onSuccess: (data) => done(queryClient, data),
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409) {
        void queryClient.invalidateQueries({ queryKey: keys.dashboard() });
      }
      // What the change was about is gone, such as a session discarded in
      // the web UI, so every screen asks again and shows what's there now.
      if (error instanceof ApiError && error.status === 404) void queryClient.invalidateQueries();
    },
  });
}

// Signs in to the server at `url`, already normalized: checks it is a Kedge
// server on this app's API version, trades the password for a token, and
// fetches Today with it before saving the connection, so the gate opens onto
// a screen that is ready.
export function useLogin() {
  const { clientFor } = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ url, password }: { url: string; password: string }) => {
      const server = clientFor({ url });
      const info = await answer(server.GET('/info'));
      const outdated = checkApiVersion(info);
      if (outdated) throw new Error(outdated);
      const { token, expiresAt } = await answer(server.POST('/login', { body: { password } }));
      const dashboard = await answer(clientFor({ url, token }).GET('/dashboard'));
      queryClient.setQueryData(keys.info(), info);
      queryClient.setQueryData(keys.dashboard(), dashboard);
      await saveConnection({ url, token, expiresAt });
      return dashboard;
    },
  });
}

// Starts the next session, or resumes the one in progress.
export function useStartWorkout() {
  return useChange(
    (api) => answer(api.POST('/workouts')),
    (qc, w) => show(qc, keys.workout(w.id), w),
  );
}

// Logs a set and starts the rest the server gives for it, timed from the tap.
// How to load the bar for a weight being typed, or nothing to ask while
// `weight` is undefined. Each keystroke isn't worth asking about, so it waits
// for the typing to settle, and until the answer is for this very weight there
// is no answer: a load for another weight is worse than none.
export function usePlates(weight: number | undefined) {
  const { read } = useApi();
  const [settled, setSettled] = useState(weight);
  useEffect(() => {
    const wait = setTimeout(() => setSettled(weight), 300);
    return () => clearTimeout(wait);
  }, [weight]);
  const current = settled === weight;
  const { queryKey, queryFn } = read.plates(settled ?? 0);
  const query = useQuery({ queryKey, queryFn, enabled: current && weight !== undefined });
  return current && weight !== undefined ? { data: query.data, error: query.error } : { data: undefined, error: null };
}

export function useLogSet(id: number) {
  return useChange(
    async (api, { idx, reps, weight }: { idx: number; reps: number; weight?: number }) => {
      const tapped = Date.now();
      const w = await answer(
        api.PUT('/workouts/{id}/sets/{idx}', { params: { path: { id, idx } }, body: { reps, weight } }),
      );
      startRest(id, w.restSeconds, tapped);
      return w;
    },
    (qc, w) => show(qc, keys.workout(id), w),
  );
}

export function useUndoSet(id: number) {
  return useChange(
    (api, { idx }: { idx: number }) =>
      answer(api.DELETE('/workouts/{id}/sets/{idx}', { params: { path: { id, idx } } })),
    (qc, w) => {
      clearRest(id);
      show(qc, keys.workout(id), w);
    },
  );
}

// Finishing or discarding a session answers with Today. The session's own
// query is left alone while its screen goes back, so it isn't refetched into
// the finished session or a 404, and dropped once the screen is gone.
function closeSession(qc: QueryClient, id: number) {
  clearRest(id);
  dropWhenUnwatched(qc, keys.workout(id));
}

export function useFinishWorkout(id: number) {
  return useChange(
    (api) => answer(api.POST('/workouts/{id}/finish', { params: { path: { id } } })),
    (qc, d) => {
      show(qc, keys.dashboard(), d, keys.workout(id));
      closeSession(qc, id);
    },
  );
}

export function useDiscardWorkout(id: number) {
  return useChange(
    (api) => answer(api.DELETE('/workouts/{id}', { params: { path: { id } } })),
    (qc, d) => {
      show(qc, keys.dashboard(), d, keys.workout(id));
      closeSession(qc, id);
    },
  );
}

export function useAddEstimate(slug: string) {
  return useChange(
    (api, body: PastEstimateInput) => answer(api.POST('/lifts/{slug}/estimates', { params: { path: { slug } }, body })),
    (qc, l) => show(qc, keys.lift(slug), l),
  );
}

export function useDeleteEstimate(slug: string) {
  return useChange(
    (api, { id }: { id: number }) =>
      answer(api.DELETE('/lifts/{slug}/estimates/{id}', { params: { path: { slug, id } } })),
    (qc, l) => show(qc, keys.lift(slug), l),
  );
}

export function useSaveSettings() {
  return useChange(
    (api, body: SettingsInput) => answer(api.PUT('/settings', { body })),
    (qc, s) => show(qc, keys.settings(), s),
  );
}

export function useSetTrainingMax() {
  return useChange(
    (api, body: TrainingMaxInput) => answer(api.POST('/training-maxes', { body })),
    (qc, s) => show(qc, keys.settings(), s),
  );
}

export function useAcceptReview() {
  return useChange(
    (api, body: TMReviewInput) => answer(api.POST('/tm-review', { body })),
    (qc, d) => show(qc, keys.dashboard(), d),
  );
}

// Settles a failed TM test; `id` is the test session's, TMTestResult.workoutId.
export function useDecideTMTest() {
  return useChange(
    (api, { id, ...body }: { id: number } & TMTestDecision) =>
      answer(api.POST('/tm-tests/{id}', { params: { path: { id } }, body })),
    (qc, d) => show(qc, keys.dashboard(), d),
  );
}

export function useSkipCycle() {
  return useChange(
    (api) => answer(api.POST('/skip-cycle')),
    (qc, d) => show(qc, keys.dashboard(), d),
  );
}

export function useSaveSetup() {
  return useChange(
    (api, body: SetupInput) => answer(api.PUT('/setup', { body })),
    (qc, p) => show(qc, keys.setupPosition(), p),
  );
}

export function useSaveSetupHistory(cycle: number) {
  return useChange(
    (api, body: SetupHistoryInput) =>
      answer(api.PUT('/setup/history/{cycle}', { params: { path: { cycle } }, body })),
    (qc, d) => show(qc, keys.dashboard(), d),
  );
}
