import { Stack } from 'expo-router';

import { font, usePalette } from '@/theme';

// A tab's own stack, for its native large title. A header background colour
// hides the large title on iOS 26, so the screen's own background shows
// through instead. A page pushed in the tab, such as a Settings section,
// has an ink back button and bar buttons.
export function TabStack({ title }: { title: string }) {
  const p = usePalette();
  return (
    <Stack
      screenOptions={{
        title,
        headerLargeTitle: true,
        headerLargeTitleShadowVisible: false,
        headerShadowVisible: false,
        headerTintColor: p.ink,
        headerTitleStyle: { fontFamily: font.heavy, color: p.ink },
        headerLargeTitleStyle: { fontFamily: font.black, color: p.ink },
        contentStyle: { backgroundColor: p.ground },
      }}
    />
  );
}
