import { useQueryClient } from '@tanstack/react-query';
import { router, Stack, usePathname } from 'expo-router';
import { useEffect, useRef, useState } from 'react';

import { keys, useInfo } from '@/api/queries';
import { checkApiVersion, signOut, useConnection } from '@/lib/connection';
import { font, usePalette } from '@/theme';

// The app's screens behind the sign-in gate: Connect alone until there is a
// session, then everything but Connect. A 401 or a sign-out closes the gate
// by itself, and whichever screen was showing gives way to Connect.
export function RootStack() {
  const p = usePalette();
  const { signedIn } = useConnection();

  return (
    <>
      {signedIn ? (
        <>
          <VersionCheck />
          <SetupAfterSignIn />
        </>
      ) : null}
      <Stack
        screenOptions={{
          headerShadowVisible: false,
          headerStyle: { backgroundColor: p.ground },
          headerTintColor: p.ink,
          headerTitleStyle: { fontFamily: font.heavy, fontSize: 20, color: p.ink },
          headerBackTitle: 'Today',
          contentStyle: { backgroundColor: p.ground },
        }}>
        <Stack.Protected guard={signedIn} redirectTo="/connect">
          {/* Signing in fades Connect into Today, as signing out fades it back.
              A screen's animation is its own push and pop, so the screens
              pushed over the tabs still slide. */}
          <Stack.Screen name="(tabs)" options={{ headerShown: false, animation: 'fade' }} />
          <Stack.Screen
            name="setup/index"
            options={{ title: 'Training maxes', headerBackVisible: false, gestureEnabled: false }}
          />
          <Stack.Screen name="setup/position" options={{ title: 'Where are you?', headerBackTitle: 'Back' }} />
          <Stack.Screen name="setup/history" options={{ headerBackTitle: 'Back' }} />
          <Stack.Screen name="workout/[id]" />
          <Stack.Screen name="lift/[slug]/index" />
          <Stack.Screen
            name="lift/[slug]/add-estimate"
            options={{
              presentation: 'formSheet',
              headerShown: false,
              sheetAllowedDetents: [0.62],
              sheetGrabberVisible: true,
              contentStyle: { backgroundColor: p.ground },
            }}
          />
        </Stack.Protected>
        <Stack.Protected guard={!signedIn} redirectTo="/">
          <Stack.Screen name="connect" options={{ headerShown: false, gestureEnabled: false, animation: 'fade' }} />
        </Stack.Protected>
      </Stack>
    </>
  );
}

// Signing in to a server with no training maxes yet goes on to setup, over
// Today as Today's own Set up button opens it, so finishing setup comes back
// to Today. Connect can't go there itself: the gate closes it the moment the
// session is saved and redirects to Today, and a push sent sooner would race
// that redirect, so this waits for Today. Connect cached Today's answer just
// before saving; at launch nothing is cached, and Today says so itself.
function SetupAfterSignIn() {
  const queryClient = useQueryClient();
  const [needsSetup] = useState(() => queryClient.getQueryData(keys.dashboard())?.needsSetup === true);
  const pathname = usePathname();
  const opened = useRef(false);
  useEffect(() => {
    if (!needsSetup || opened.current || pathname !== '/') return;
    opened.current = true;
    router.push('/setup');
  }, [needsSetup, pathname]);
  return null;
}

// The server can be updated, or rolled back, while the app keeps its
// session, so its API version is checked again at every launch and every
// return to the foreground. A version this build can't speak signs out with
// what to update, which Connect shows. A server that can't be reached
// blocks nothing: each screen says so itself.
function VersionCheck() {
  const { data } = useInfo();
  useEffect(() => {
    const outdated = data && checkApiVersion(data);
    if (outdated) void signOut(outdated);
  }, [data]);
  return null;
}
