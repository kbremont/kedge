import { useEffect, useState, useSyncExternalStore } from 'react';

// The rest between sets, by session: when the lifter tapped Log, and how long
// the server says to rest after that set. It lives outside the session
// screen, so leaving the session and coming back keeps the right time, and
// only a set logged since the app started has one, so a session opened fresh
// shows no timer.

type Rest = { startedAt: number; seconds: number };

const rests = new Map<number, Rest>();
const listeners = new Set<() => void>();

function changed() {
  listeners.forEach((l) => l());
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

export function startRest(workoutId: number, seconds: number, startedAt = Date.now()) {
  rests.set(workoutId, { startedAt, seconds });
  changed();
}

// Undoing the set takes its rest with it.
export function clearRest(workoutId: number) {
  if (rests.delete(workoutId)) changed();
}

// Seconds of a session's rest left, counted against the clock so it stays
// right if the phone sleeps, and its whole length for the bar. Both are 0
// when there is no rest.
export function useRest(workoutId: number): { left: number; total: number } {
  const rest = useSyncExternalStore(subscribe, () => rests.get(workoutId));
  const [now, setNow] = useState(Date.now);

  // Ticks until the rest runs out. A new rest restarts it.
  useEffect(() => {
    if (!rest) return;
    const end = rest.startedAt + rest.seconds * 1000;
    const id = setInterval(() => {
      const t = Date.now();
      setNow(t);
      if (t >= end) clearInterval(id);
    }, 250);
    return () => clearInterval(id);
  }, [rest]);

  if (!rest) return { left: 0, total: 0 };
  // Until the first tick `now` can be from before the rest began.
  const elapsed = Math.max(0, Math.floor((now - rest.startedAt) / 1000));
  return { left: Math.max(0, rest.seconds - elapsed), total: rest.seconds };
}
