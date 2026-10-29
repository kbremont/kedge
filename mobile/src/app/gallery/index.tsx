import { Redirect, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { Fragment, useEffect } from 'react';
import { Pressable, Text } from 'react-native';

import { Heading, Page, useCardStyles } from '@/components/card-back';
import { galleryStates, openGalleryState } from '@/gallery/states';

// Dev only: every screen and state in the design. /gallery?state=<id> opens
// one directly, for the screenshot script.
export default function GalleryScreen() {
  const { state } = useLocalSearchParams<{ state?: string }>();
  const { p, s } = useCardStyles();

  useEffect(() => {
    if (!__DEV__ || !state) return;
    const id = setTimeout(() => openGalleryState(state), 0);
    return () => clearTimeout(id);
  }, [state]);

  if (!__DEV__) return <Redirect href="/" />;

  const screens = [...new Set(galleryStates.map((x) => x.screen))];
  return (
    <Page>
      {screens.map((screen) => (
        <Fragment key={screen}>
          <Heading title={screen} />
          {galleryStates
            .filter((x) => x.screen === screen)
            .map((x) => (
              <Pressable
                key={x.id}
                accessibilityRole="button"
                onPress={() => openGalleryState(x.id)}
                style={({ pressed }) => [s.row, pressed && s.pressed]}>
                <Text style={[s.name, { flex: 1 }]}>{x.label}</Text>
                <Text style={[s.sub, { marginTop: 0, marginRight: 8 }]}>{x.id}</Text>
                <SymbolView name="chevron.right" tintColor={p.ink2} size={11} weight="bold" />
              </Pressable>
            ))}
        </Fragment>
      ))}
    </Page>
  );
}
