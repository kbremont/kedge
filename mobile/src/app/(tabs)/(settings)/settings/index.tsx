import { Host, Picker, Text as SwiftText } from '@expo/ui/swift-ui';
import { pickerStyle, tag } from '@expo/ui/swift-ui/modifiers';
import { router } from 'expo-router';
import { Alert, StyleSheet, Text, useColorScheme, View } from 'react-native';

import { useInfo, useSettings } from '@/api/queries';
import type { Settings } from '@/api/schema.gen';
import { Heading, Intro, LiftChip, LoadState, Page, TextAction, useCardStyles } from '@/components/card-back';
import { Row, SectionRow, settingStyles } from '@/components/settings';
import { setAppearance, useAppearance, type AppearanceChoice } from '@/lib/appearance';
import { running, runningText } from '@/lib/build';
import { signOut, useConnection } from '@/lib/connection';
import { fmtClock, fmtWeight, shortName } from '@/lib/format';

export default function SettingsScreen() {
  const settings = useSettings();
  return (
    <LoadState
      query={settings}
      keep={
        <>
          <AppearanceSection />
          <Server />
        </>
      }>
      {(s) => (
        <Page>
          <Sections settings={s} />
          <AppearanceSection />
          <Server />
        </Page>
      )}
    </LoadState>
  );
}

// The settings the server keeps, a row for each section, saying what it's
// set to. Each opens the section's own page.
function Sections({ settings }: { settings: Settings }) {
  const { s } = useCardStyles();
  const { unit } = settings;
  const overrides = settings.bbb.filter((b) => b.mode !== 'program');
  const rest = [
    { seconds: settings.restWarmupSeconds, after: 'after warmups' },
    { seconds: settings.restMainSeconds, after: 'after main sets' },
    { seconds: settings.restSuppSeconds, after: 'after supplemental' },
  ];

  return (
    <>
      <Heading title="Training" />
      <SectionRow
        label="Training maxes"
        summary={settings.tms.map((tm) => `${shortName(tm.slug, tm.name)} ${fmtWeight(tm.value)} ${unit}`).join(', ')}
        onPress={() => router.push('/settings/training-maxes')}>
        <View style={local.tms}>
          {settings.tms.map((tm) => (
            <View key={tm.slug} style={local.tm}>
              <LiftChip slug={tm.slug} />
              <Text style={[s.sub, local.tmValue]}>{fmtWeight(tm.value)}</Text>
            </View>
          ))}
        </View>
      </SectionRow>
      <SectionRow
        label="Loading"
        summary={[
          `${fmtWeight(settings.barWeight)} ${unit} bar`,
          `rounds to ${fmtWeight(settings.rounding)}`,
          settings.warmups ? 'warmups' : 'no warmups',
        ].join(' · ')}
        onPress={() => router.push('/settings/loading')}
      />
      <SectionRow
        label="Rest"
        summary={rest.map((r) => `${fmtClock(r.seconds)} ${r.after}`).join(', ')}
        onPress={() => router.push('/settings/rest')}>
        <Text style={s.sub}>{rest.map((r) => fmtClock(r.seconds)).join(' · ')}</Text>
      </SectionRow>
      <SectionRow
        label="Supplemental"
        summary={
          overrides.length === 0
            ? 'Program'
            : overrides
                .map(
                  (b) =>
                    `${shortName(b.slug, b.name)} ${fmtWeight(b.value)}${b.mode === 'percent' ? '%' : ` ${unit}`}`,
                )
                .join(' · ')
        }
        onPress={() => router.push('/settings/supplemental')}
      />
    </>
  );
}

const appearanceLabel: Record<AppearanceChoice, string> = { system: 'System', light: 'Light', dark: 'Dark' };

// Light or dark for this phone. It applies the moment it's picked and is
// kept on the phone, so it sits apart from the settings the server keeps,
// and shows even when those can't be read.
function AppearanceSection() {
  const { p } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const { choice } = useAppearance();
  return (
    <>
      <Heading title="Appearance" />
      <Intro style={settingStyles.intro}>System follows your iPhone’s setting.</Intro>
      <View style={[settingStyles.block, { borderBottomColor: p.rule }]}>
        <Host matchContents={{ vertical: true }} seedColor={p.ink} colorScheme={scheme}>
          <Picker
            selection={choice}
            onSelectionChange={(c: AppearanceChoice) => void setAppearance(c)}
            modifiers={[pickerStyle('segmented')]}>
            {(['system', 'light', 'dark'] as const).map((c) => (
              <SwiftText key={c} modifiers={[tag(c)]}>
                {appearanceLabel[c]}
              </SwiftText>
            ))}
          </Picker>
        </Host>
      </View>
    </>
  );
}

// The server the app is signed in to, what this install runs, and the way
// out. It needs nothing from the settings read, so it shows even when that
// read fails.
function Server() {
  const { s } = useCardStyles();
  const { url } = useConnection();
  // The server's build and API version: a dash until they're in, and if they
  // can't be fetched.
  const info = useInfo().data;

  // The gate closes onto Connect, with the address kept, and the provider
  // forgets every screen of this session.
  const confirmSignOut = () =>
    Alert.alert('Sign out?', 'You’ll need the server’s password to connect again.', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Sign out', style: 'destructive', onPress: () => void signOut() },
    ]);

  return (
    <>
      <Heading title="Server" />
      <Row label="Address">
        <Text style={[s.sub, local.value]} numberOfLines={1} ellipsizeMode="middle">
          {url}
        </Text>
      </Row>
      <Row label="Version">
        <Text style={[s.sub, local.value]} accessibilityLabel={info ? undefined : 'Unknown'}>
          {info ? `${info.version} · API ${info.apiVersion}` : '—'}
        </Text>
      </Row>
      {/* This install beside the server: its version and build, and whether
          it runs the bundle it was built with or an update. */}
      <Row label="App">
        <Text style={[s.sub, local.value]}>{runningText(running())}</Text>
      </Row>
      <TextAction
        label="Sign out"
        icon="rectangle.portrait.and.arrow.right"
        onPress={confirmSignOut}
        style={local.signOut}
      />

      <Text style={[s.sub, local.footer]}>Not affiliated with Jim Wendler.</Text>
    </>
  );
}

const local = StyleSheet.create({
  tms: { flexDirection: 'row', flexWrap: 'wrap', columnGap: 14, rowGap: 2, marginTop: 2 },
  tm: { flexDirection: 'row', alignItems: 'center', gap: 5 },
  tmValue: { marginTop: 0 },
  value: { marginTop: 0, maxWidth: 210, textAlign: 'right' },
  signOut: { marginTop: 14 },
  footer: { textAlign: 'center', marginTop: 28 },
});
