import { useRef, useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { useLogin } from '@/api/queries';
import { Button, Field, Intro, Label, Notice, useCardStyles } from '@/components/card-back';
import { PlateMark } from '@/components/plate-mark';
import { appName } from '@/lib/brand';
import { normalizeServerUrl, sessionEnded, useConnection } from '@/lib/connection';
import { displayScaleCap, font, space } from '@/theme';

// First run, and after a sign-out: the lifter's own server and its password.
// The app talks to nothing else. Once the session is saved, the gate opens
// onto Today or setup by itself.
export default function ConnectScreen() {
  const { p, s } = useCardStyles();
  const insets = useSafeAreaInsets();
  const connection = useConnection();
  const login = useLogin();
  // After a sign-out the address is the one the app last used.
  const [url, setUrl] = useState(connection.url);
  const [password, setPassword] = useState('');
  // An address refused before anything is sent, such as plain http:// to a
  // server out on the internet.
  const [urlError, setUrlError] = useState('');
  // Why the app signed out by itself stands until the lifter tries again;
  // from then on the answer to that try speaks instead.
  const [tried, setTried] = useState(false);
  // A second tap can land before the button redraws disabled, and mustn't
  // sign in twice.
  const sending = useRef(false);

  const failure = login.error;
  const message = urlError || failure?.message || (tried ? '' : connection.reason);
  // A session that ended is news, not a fault: the password is all it takes.
  // Everything else here keeps Kedge from working until something changes.
  const news = message === sessionEnded;
  const badUrl = urlError !== '' || (failure instanceof ApiError && failure.notKedge);
  const badPassword = failure instanceof ApiError && failure.status === 401;

  // A refusal is shown as the server wrote it, with the field it's about
  // outlined, until the lifter changes what they typed.
  const edit = () => {
    setUrlError('');
    if (login.isError) login.reset();
  };

  const connect = () => {
    if (!url || !password || sending.current) return;
    setTried(true);
    const normalized = normalizeServerUrl(url);
    if ('error' in normalized) {
      setUrlError(normalized.error);
      return;
    }
    sending.current = true;
    // On success the gate closes Connect, so only a failure comes back here.
    login.mutate(
      { url: normalized.url, password },
      {
        onSettled: () => {
          sending.current = false;
        },
      },
    );
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
        onChangeText={(v) => {
          setUrl(v);
          edit();
        }}
        invalid={badUrl}
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
          edit();
        }}
        invalid={badPassword}
        secureTextEntry
        textContentType="password"
        returnKeyType="go"
        onSubmitEditing={connect}
        accessibilityLabel="Password"
      />
      {message ? (
        <Notice error={!news} icon={news ? 'key.fill' : undefined} style={local.notice}>
          {message}
        </Notice>
      ) : null}
      <Button
        label={login.isPending ? 'Connecting…' : 'Connect'}
        disabled={!url || !password || login.isPending}
        style={message ? local.afterNotice : local.connect}
        onPress={connect}
      />
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
  notice: { marginTop: 12, paddingBottom: 0 },
  connect: { marginTop: 24 },
  afterNotice: { marginTop: 12 },
  note: { marginTop: 18, fontSize: 13, lineHeight: 18 },
});
