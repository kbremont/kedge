import { Stack } from 'expo-router';

import { font, usePalette } from '@/theme';

export default function TodayLayout() {
  const p = usePalette();
  return (
    <Stack
      screenOptions={{
        title: 'Today',
        // A header background colour hides the large title on iOS 26; the
        // screen's own background shows through instead.
        headerLargeTitle: true,
        headerLargeTitleShadowVisible: false,
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: font.heavy, color: p.ink },
        headerLargeTitleStyle: { fontFamily: font.black, color: p.ink },
        contentStyle: { backgroundColor: p.ground },
      }}
    />
  );
}
