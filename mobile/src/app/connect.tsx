import { router } from 'expo-router';
import { useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { Button, Field, Intro, Label, Notice, useCardStyles } from '@/components/card-back';
import { PlateMark } from '@/components/plate-mark';
import { useOverrides } from '@/gallery/store';
import { appName } from '@/lib/brand';
import { displayScaleCap, font, space } from '@/theme';

// First run: the lifter's own server and its password. The app talks to
// nothing else.
export default function ConnectScreen() {
  const { version, o } = useOverrides();
  return <Connect key={version} error={o.connectError} />;
}

function Connect({ error: errorAtStart }: { error?: string }) {
  const { p, s } = useCardStyles();
  const insets = useSafeAreaInsets();
  const [url, setUrl] = useState(errorAtStart ? 'http://raspberrypi:3000' : '');
  const [password, setPassword] = useState(errorAtStart ? 'wrong-password' : '');
  // The server's message when it refuses, shown as it is.
  const [error, setError] = useState(errorAtStart ?? '');

  // Sample only: Phase 4 checks GET /info, then POST /login.
  const connect = () => {
    if (!url || !password) return;
    setError('');
    router.replace('/setup');
  };

  return (
    <ScrollView
      style={s.page}
      contentContainerStyle={[local.content, { paddingTop: insets.top + 36, paddingBottom: insets.bottom + 24 }]}
      automaticallyAdjustKeyboardInsets
      keyboardShouldPersistTaps="handled">
      <View style={[local.masthead, { borderBottomColor: p.ink }]}>
        <PlateMark size={76} />
        <Text
          style={[local.name, { color: p.ink }]}
          accessibilityRole="header"
          maxFontSizeMultiplier={displayScaleCap}>
          {appName}
        </Text>
        <Text style={[local.tagline, { color: p.ink2 }]}>5/3/1, kept on your own server.</Text>
      </View>

      <Label style={local.label}>Server address</Label>
      <Field
        text
        value={url}
        onChangeText={setUrl}
        placeholder="http://raspberrypi:3000"
        keyboardType="url"
        textContentType="URL"
        autoCapitalize="none"
        autoCorrect={false}
        returnKeyType="next"
        accessibilityLabel="Server address"
      />
      <Label style={local.label}>Password</Label>
      <Field
        text
        value={password}
        onChangeText={(v) => {
          setPassword(v);
          setError('');
        }}
        secureTextEntry
        textContentType="password"
        returnKeyType="go"
        onSubmitEditing={connect}
        accessibilityLabel="Password"
      />
      {error ? (
        <Notice error style={local.error}>
          {error}
        </Notice>
      ) : null}
      <Button label="Connect" disabled={!url || !password} style={error ? local.afterError : local.connect} onPress={connect} />
      <Intro style={local.note}>
        The app talks only to this server, over Tailscale or your own network. Nothing you log is sent anywhere
        else.
      </Intro>
    </ScrollView>
  );
}

const local = StyleSheet.create({
  content: { paddingHorizontal: space.page + 4 },
  masthead: { paddingBottom: 14, marginBottom: 14, borderBottomWidth: 3 },
  name: {
    fontFamily: font.black,
    fontSize: 64,
    lineHeight: 64,
    letterSpacing: 0.4,
    textTransform: 'uppercase',
    marginTop: 14,
  },
  tagline: { fontSize: 17, fontWeight: '600', marginTop: 2 },
  label: { marginTop: 16, marginBottom: 6 },
  error: { marginTop: 12, paddingBottom: 0 },
  connect: { marginTop: 24 },
  afterError: { marginTop: 12 },
  note: { marginTop: 18, fontSize: 13, lineHeight: 18 },
});
