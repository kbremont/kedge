import { router } from 'expo-router';
import { useEffect } from 'react';

// Dev only, and only when Metro runs with EXPO_PUBLIC_REMOTE_STEP=1: lets a
// screenshot script move between screens without deep links, which iOS
// answers with an "Open in…?" prompt. Polls a local file holding a path, or
// several separated by commas to push in order, and goes there when it
// changes.
const STEP_URL = 'http://127.0.0.1:8194/step.txt';

export function useRemoteStep() {
  useEffect(() => {
    if (!__DEV__ || process.env.EXPO_PUBLIC_REMOTE_STEP !== '1') return;
    let last = '';
    const id = setInterval(async () => {
      try {
        const res = await fetch(`${STEP_URL}?t=${Date.now()}`);
        if (!res.ok) return;
        const path = (await res.text()).trim();
        if (path === last) return;
        last = path;
        if (router.canDismiss()) router.dismissAll();
        for (const p of path.split(',')) {
          if (p && p !== '/') router.push(p as never);
        }
      } catch {
        // No stepper running.
      }
    }, 400);
    return () => clearInterval(id);
  }, []);
}
