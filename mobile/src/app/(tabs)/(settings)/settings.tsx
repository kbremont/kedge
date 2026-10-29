import { Host, Picker, Stepper, Text as SwiftText } from '@expo/ui/swift-ui';
import { labelsHidden, pickerStyle, tag } from '@expo/ui/swift-ui/modifiers';
import { SymbolView } from 'expo-symbols';
import { useState, type ReactNode } from 'react';
import { Alert, Pressable, StyleSheet, Switch, Text, useColorScheme, useWindowDimensions, View } from 'react-native';

import type { BBBMode, Settings } from '@/api/schema.gen';
import {
  Button,
  Field,
  Heading,
  Intro,
  LiftChip,
  Notice,
  Page,
  TextAction,
  useCardStyles,
} from '@/components/card-back';
import { useOverrides } from '@/gallery/store';
import { fmtClock, fmtWeight } from '@/lib/format';
import { sample } from '@/sample';
import { useSettings } from '@/sample/use-screen';
import { displayScaleCap, font } from '@/theme';

export default function SettingsScreen() {
  const { version, o } = useOverrides();
  const settings = useSettings();
  return <SettingsForm key={version} settings={settings} savedAtStart={o.settingsSaved ?? false} />;
}

// What the Save button sends: everything but the training maxes, which save
// one at a time.
type Form = {
  barWeight: string;
  rounding: string;
  plates: string;
  warmups: boolean;
  restWarmupSeconds: number;
  restMainSeconds: number;
  restSuppSeconds: number;
  bbb: Record<string, { mode: BBBMode; value: string }>;
};

function toForm(s: Settings): Form {
  return {
    barWeight: fmtWeight(s.barWeight),
    rounding: fmtWeight(s.rounding),
    plates: s.plates.map(fmtWeight).join(', '),
    warmups: s.warmups,
    restWarmupSeconds: s.restWarmupSeconds,
    restMainSeconds: s.restMainSeconds,
    restSuppSeconds: s.restSuppSeconds,
    bbb: Object.fromEntries(
      s.bbb.map((b) => [b.slug, { mode: b.mode, value: b.value > 0 ? fmtWeight(b.value) : '' }]),
    ),
  };
}

const modeLabel: Record<BBBMode, string> = { program: 'Program', percent: '% of TM', weight: 'Weight' };

function SettingsForm({ settings, savedAtStart }: { settings: Settings; savedAtStart: boolean }) {
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const [tms, setTms] = useState(settings.tms);
  const [saved, setSaved] = useState(() => toForm(settings));
  const [form, setForm] = useState(saved);
  const [justSaved, setJustSaved] = useState(savedAtStart);
  const stacked = useWindowDimensions().fontScale > 1.5;
  const dirty = JSON.stringify(form) !== JSON.stringify(saved);
  const unit = settings.unit;

  const edit = (patch: Partial<Form>) => {
    setForm((f) => ({ ...f, ...patch }));
    setJustSaved(false);
  };
  const editBBB = (slug: string, patch: Partial<Form['bbb'][string]>) =>
    edit({ bbb: { ...form.bbb, [slug]: { ...form.bbb[slug], ...patch } } });

  // A training max changes future sessions, so each one is its own decision.
  const editTM = (slug: string, name: string, value: number) =>
    Alert.prompt(
      `${name} training max`,
      'Changes future sessions only. Finished sessions keep the weights they were lifted at.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Save',
          isPreferred: true,
          // Sample only: PUT /lifts/{slug}/training-max.
          onPress: (v?: string) => {
            const n = Number(v);
            if (n > 0) setTms((xs) => xs.map((t) => (t.slug === slug ? { ...t, value: n } : t)));
          },
        },
      ],
      'plain-text',
      fmtWeight(value),
      'decimal-pad',
    );

  // Sample only: PUT /settings answers with the settings as saved.
  const save = () => {
    setSaved(form);
    setJustSaved(true);
  };

  const backup = () =>
    // Sample only: GET /backup, then the share sheet.
    Alert.alert('Backups come with the server', 'Once the app is connected, the backup opens in the share sheet.');

  const signOut = () =>
    Alert.alert('Sign out?', 'You’ll need the server’s password to connect again.', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Sign out', style: 'destructive' },
    ]);

  const rest: { key: 'restWarmupSeconds' | 'restMainSeconds' | 'restSuppSeconds'; label: string; sub?: string }[] =
    [
      { key: 'restWarmupSeconds', label: 'After warmups' },
      { key: 'restMainSeconds', label: 'After main sets' },
      { key: 'restSuppSeconds', label: 'After supplemental', sub: 'BBB and FSL' },
    ];

  return (
    <Page>
      <Heading title="Training maxes" />
      <Intro style={local.intro}>Changes future sessions only. Finished sessions keep their weights.</Intro>
      {tms.map((tm) => (
        <Pressable
          key={tm.slug}
          accessibilityRole="button"
          accessibilityLabel={`${tm.name}, training max ${fmtWeight(tm.value)} ${unit}. Change`}
          onPress={() => editTM(tm.slug, tm.name, tm.value)}
          style={({ pressed }) => [s.row, stacked && local.stackedRow, pressed && s.pressed]}>
          <View style={local.nameRow}>
            <LiftChip slug={tm.slug} />
            <Text style={s.name}>{tm.name}</Text>
          </View>
          <View style={[local.control, stacked && local.stackedControl]}>
            <Text style={s.num} maxFontSizeMultiplier={stacked ? undefined : displayScaleCap}>
              {fmtWeight(tm.value)}
              <Text style={[local.unit, { color: p.ink2 }]}> {unit}</Text>
            </Text>
            <SymbolView name="square.and.pencil" tintColor={p.ink2} size={16} style={local.edit} />
          </View>
        </Pressable>
      ))}

      <Heading title="Loading" />
      <Row label="Bar">
        <Field
          value={form.barWeight}
          onChangeText={(barWeight) => edit({ barWeight })}
          keyboardType="decimal-pad"
          accessibilityLabel={`Bar weight, ${unit}`}
          style={local.small}
        />
        <Unit>{unit}</Unit>
      </Row>
      <Row label="Round weights to">
        <Field
          value={form.rounding}
          onChangeText={(rounding) => edit({ rounding })}
          keyboardType="decimal-pad"
          accessibilityLabel={`Round weights to, ${unit}`}
          style={local.small}
        />
        <Unit>{unit}</Unit>
      </Row>
      <View style={[local.block, { borderBottomColor: p.rule }]}>
        <Text style={s.name}>Plates, per side</Text>
        <Text style={s.sub}>The plates you have, heaviest first.</Text>
        <Field
          value={form.plates}
          onChangeText={(plates) => edit({ plates })}
          keyboardType="numbers-and-punctuation"
          accessibilityLabel={`Plates per side, ${unit}`}
          style={local.plates}
        />
      </View>
      <Row label="Warmup sets" sub="40, 50 and 60% × 5 before the main sets">
        <Switch
          value={form.warmups}
          onValueChange={(warmups) => edit({ warmups })}
          trackColor={{ true: p.button, false: p.rule }}
          thumbColor={form.warmups ? p.onButton : undefined}
          accessibilityLabel="Warmup sets"
        />
      </Row>

      <Heading title="Rest" />
      {rest.map((r) => (
        <Row key={r.key} label={r.label} sub={r.sub}>
          <Text style={[s.num, local.clock]} maxFontSizeMultiplier={displayScaleCap}>
            {fmtClock(form[r.key])}
          </Text>
          <Host matchContents seedColor={p.ink} colorScheme={scheme}>
            <Stepper
              label={r.label}
              value={form[r.key]}
              step={15}
              min={15}
              max={600}
              onValueChange={(v) => edit({ [r.key]: v })}
              modifiers={[labelsHidden()]}
            />
          </Host>
        </Row>
      ))}

      <Heading title="Supplemental" />
      <Intro style={local.intro}>
        The weight of the leaders’ BBB sets. Program follows each cycle’s own percentage.
      </Intro>
      {settings.bbb.map((b) => {
        const v = form.bbb[b.slug];
        return (
          <View key={b.slug} style={[local.block, { borderBottomColor: p.rule }]}>
            <View style={local.bbbTop}>
              <View style={local.nameRow}>
                <LiftChip slug={b.slug} />
                <Text style={s.name}>{b.name}</Text>
              </View>
              {v.mode === 'program' ? null : (
                <View style={local.bbbValue}>
                  <Field
                    value={v.value}
                    onChangeText={(value) => editBBB(b.slug, { value })}
                    keyboardType="decimal-pad"
                    accessibilityLabel={`${b.name} BBB, ${v.mode === 'percent' ? 'percent of training max' : unit}`}
                    style={local.small}
                  />
                  <Unit>{v.mode === 'percent' ? '%' : unit}</Unit>
                </View>
              )}
            </View>
            <Host matchContents={{ vertical: true }} seedColor={p.ink} colorScheme={scheme} style={local.segments}>
              <Picker
                selection={v.mode}
                onSelectionChange={(mode: BBBMode) => editBBB(b.slug, { mode })}
                modifiers={[pickerStyle('segmented')]}>
                {(['program', 'percent', 'weight'] as const).map((m) => (
                  <SwiftText key={m} modifiers={[tag(m)]}>
                    {modeLabel[m]}
                  </SwiftText>
                ))}
              </Picker>
            </Host>
          </View>
        );
      })}

      {justSaved && !dirty ? (
        <Notice style={local.saved}>Saved. The next session uses these settings.</Notice>
      ) : (
        <Button label="Save settings" disabled={!dirty} style={local.save} onPress={save} />
      )}

      <Heading title="Data" />
      <Intro style={local.intro}>
        A copy of the whole database, safe to take while you train. It opens in the share sheet.
      </Intro>
      <Button secondary icon="square.and.arrow.up" label="Download backup" style={local.backup} onPress={backup} />

      <Heading title="Server" />
      <Row label="Address">
        <Text style={[s.sub, local.value]} numberOfLines={1} ellipsizeMode="middle">
          {sample.server}
        </Text>
      </Row>
      <Row label="Version">
        <Text style={[s.sub, local.value]}>
          {sample.info.version} · API {sample.info.apiVersion}
        </Text>
      </Row>
      <TextAction label="Sign out" icon="rectangle.portrait.and.arrow.right" onPress={signOut} style={local.signOut} />

      <Text style={[s.sub, local.footer]}>Not affiliated with Jim Wendler.</Text>
    </Page>
  );
}

// A ruled settings line: what it is on the left, its control on the right.
// At accessibility text sizes the label goes over the control, so neither
// has to break.
function Row({ label, sub, children }: { label: string; sub?: string; children: ReactNode }) {
  const { s } = useCardStyles();
  const stacked = useWindowDimensions().fontScale > 1.5;
  return (
    <View style={[s.row, local.settingRow, stacked && local.stackedRow]}>
      <View style={stacked ? undefined : { flex: 1 }}>
        <Text style={s.name}>{label}</Text>
        {sub ? <Text style={s.sub}>{sub}</Text> : null}
      </View>
      <View style={[local.control, stacked && local.stackedControl]}>{children}</View>
    </View>
  );
}

function Unit({ children }: { children: ReactNode }) {
  const { s } = useCardStyles();
  return (
    <Text style={[s.statUnit, local.fieldUnit]} maxFontSizeMultiplier={displayScaleCap}>
      {children}
    </Text>
  );
}

const local = StyleSheet.create({
  intro: { marginTop: 10, marginBottom: 2 },
  nameRow: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: 7 },
  unit: { fontFamily: font.semi, fontSize: 17 },
  edit: { marginLeft: 4 },
  settingRow: { minHeight: 64, gap: 12 },
  control: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  stackedRow: { flexDirection: 'column', alignItems: 'stretch', gap: 8 },
  stackedControl: { justifyContent: 'flex-end' },
  small: { width: 84, height: 46, fontSize: 24, textAlign: 'right' },
  fieldUnit: { width: 20 },
  block: { paddingVertical: 12, borderBottomWidth: StyleSheet.hairlineWidth * 2 },
  plates: { marginTop: 8, fontSize: 24 },
  clock: { minWidth: 48, textAlign: 'right' },
  bbbTop: { flexDirection: 'row', alignItems: 'center', minHeight: 46 },
  bbbValue: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  segments: { marginTop: 10 },
  save: { marginTop: 24 },
  saved: { marginTop: 18 },
  backup: { marginTop: 14 },
  value: { marginTop: 0, maxWidth: 210, textAlign: 'right' },
  signOut: { marginTop: 14 },
  footer: { textAlign: 'center', marginTop: 28 },
});
