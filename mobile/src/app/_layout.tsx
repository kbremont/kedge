import { useFonts } from 'expo-font';
import * as SplashScreen from 'expo-splash-screen';
import { StatusBar } from 'expo-status-bar';
import { useEffect } from 'react';
import { GestureHandlerRootView } from 'react-native-gesture-handler';

import { ApiProvider } from '@/api/queries';
import { RootStack } from '@/components/root-stack';
import { loadConnection, useConnection } from '@/lib/connection';
import { fontFiles } from '@/theme';

SplashScreen.preventAutoHideAsync();
// The saved session is read while the fonts load, so the splash screen gives
// way straight to the right first screen: Today, or Connect.
void loadConnection();

export const unstable_settings = { anchor: '(tabs)' };

export default function RootLayout() {
  const [fontsLoaded] = useFonts(fontFiles);
  const { loaded } = useConnection();
  const ready = fontsLoaded && loaded;

  useEffect(() => {
    if (ready) SplashScreen.hideAsync();
  }, [ready]);

  if (!ready) return null;

  return (
    <ApiProvider>
      <GestureHandlerRootView style={{ flex: 1 }}>
        <StatusBar style="auto" />
        <RootStack />
      </GestureHandlerRootView>
    </ApiProvider>
  );
}
