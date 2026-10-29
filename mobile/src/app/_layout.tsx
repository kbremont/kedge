import { useFonts } from 'expo-font';
import { Stack } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import { StatusBar } from 'expo-status-bar';
import { useEffect } from 'react';
import { GestureHandlerRootView } from 'react-native-gesture-handler';

import { useRemoteStep } from '@/dev/remote-step';
import { font, fontFiles, usePalette } from '@/theme';

SplashScreen.preventAutoHideAsync();

export const unstable_settings = { anchor: '(tabs)' };

export default function RootLayout() {
  const [loaded] = useFonts(fontFiles);
  const p = usePalette();
  useRemoteStep();

  useEffect(() => {
    if (loaded) SplashScreen.hideAsync();
  }, [loaded]);

  if (!loaded) return null;

  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <StatusBar style="auto" />
      <Stack
        screenOptions={{
          headerShadowVisible: false,
          headerStyle: { backgroundColor: p.ground },
          headerTintColor: p.ink,
          headerTitleStyle: { fontFamily: font.heavy, fontSize: 20, color: p.ink },
          headerBackTitle: 'Today',
          contentStyle: { backgroundColor: p.ground },
        }}>
        <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
        <Stack.Screen name="connect" options={{ headerShown: false, gestureEnabled: false, animation: 'fade' }} />
        <Stack.Screen
          name="setup/index"
          options={{ title: 'Training maxes', headerBackVisible: false, gestureEnabled: false }}
        />
        <Stack.Screen name="setup/position" options={{ title: 'Where are you?', headerBackTitle: 'Back' }} />
        <Stack.Screen name="setup/history" options={{ headerBackTitle: 'Back' }} />
        <Stack.Screen name="workout/[id]" />
        <Stack.Screen name="lift/[slug]/index" />
        <Stack.Screen name="gallery/index" options={{ title: 'Gallery' }} />
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
      </Stack>
    </GestureHandlerRootView>
  );
}
