import { Text, View } from 'react-native';

import { font, usePalette } from '@/theme';

// Designed after Today, the session and the lift page.
export default function SettingsScreen() {
  const p = usePalette();
  return (
    <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: p.ground }}>
      <Text style={{ fontFamily: font.black, fontSize: 26, textTransform: 'uppercase', color: p.ink }}>
        Settings
      </Text>
    </View>
  );
}
