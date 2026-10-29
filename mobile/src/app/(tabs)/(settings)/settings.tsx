import { Host, Picker, Stepper, Text as SwiftText } from '@expo/ui/swift-ui';
import { labelsHidden, pickerStyle, tag } from '@expo/ui/swift-ui/modifiers';
import { SymbolView } from 'expo-symbols';
import { useRef, useState, type ReactNode } from 'react';
import { Alert, Pressable, StyleSheet, Switch, Text, useColorScheme, useWindowDimensions, View } from 'react-native';

import { ApiError } from '@/api/client';
import { useInfo, useSaveSettings, useSetTrainingMax, useSettings } from '@/api/queries';
import type { BBBMode, Settings, SettingsInput } from '@/api/schema.gen';
import {
  Button,
  Field,
  Heading,
  Intro,
  LiftChip,
  LoadState,
  Notice,
  Page,
  TextAction,
  useCardStyles,
} from '@/components/card-back';
import { signOut, useConnection } from '@/lib/connection';
import { fmtClock, fmtWeight } from '@/lib/format';
import { displayScaleCap, font } from '@/theme';

export default function SettingsScreen() {
  const settings = useSettings();
  return (
    <LoadState query={settings} keep={<Server />}>
      {(s) => <SettingsForm settings={s} />}
    </LoadState>
  );
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

// A field the form won't send as it stands, and what to put there.
type Refusal = { field: string; message: string };

// A number as typed. Blank is 0, which the server reads as "keep what's
// there", and for a BBB value as "follow the program".
function parseNumber(text: string): number | undefined {
  const t = text.trim();
  if (!t) return 0;
  const n = Number(t);
  return Number.isFinite(n) ? n : undefined;
}

// The form as PUT /settings takes it. Something typed that isn't a number is
// refused here, as the web's form refuses plates it can't read: JSON would
// carry it as null, and the server's answer to that names the schema, not
// the field.
function toInput(f: Form, lifts: Settings['bbb']): SettingsInput | Refusal {
  const barWeight = parseNumber(f.barWeight);
  if (barWeight === undefined) return { field: 'barWeight', message: 'Enter the bar weight as a number.' };
  const rounding = parseNumber(f.rounding);
  if (rounding === undefined) return { field: 'rounding', message: 'Enter the rounding as a number.' };
  // Split as the web splits them, on commas and spaces.
  const plates = f.plates.split(/[\s,]+/).filter(Boolean).map(Number);
  if (!plates.every(Number.isFinite)) {
    return { field: 'plates', message: 'List the plates as numbers, separated by commas.' };
  }
  const bbb: SettingsInput['bbb'] = {};
  for (const l of lifts) {
    const { mode, value } = f.bbb[l.slug];
    // Program follows the cycle's own percentage, so its field is hidden and
    // nothing in it is sent.
    const n = mode === 'program' ? 0 : parseNumber(value);
    if (n === undefined) {
      const what = mode === 'percent' ? 'percentage' : 'weight';
      return { field: `bbb.${l.slug}`, message: `Enter the ${l.name} BBB ${what} as a number.` };
    }
    bbb[l.slug] = { mode, value: n };
  }
  return {
    barWeight,
    rounding,
    plates,
    warmups: f.warmups,
    restWarmupSeconds: f.restWarmupSeconds,
    restMainSeconds: f.restMainSeconds,
    restSuppSeconds: f.restSuppSeconds,
    bbb,
  };
}

const modeLabel: Record<BBBMode, string> = { program: 'Program', percent: '% of TM', weight: 'Weight' };

function SettingsForm({ settings }: { settings: Settings }) {
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const stacked = useWindowDimensions().fontScale > 1.5;
  const unit = settings.unit;

  // The settings the form was loaded from, and the lifter's edits to them. A
  // refetch, such as on coming back to the app, reloads the form only while
  // it's untouched, so it never undoes what the lifter is typing.
  const [base, setBase] = useState(settings);
  const saved = toForm(base);
  const [form, setForm] = useState(saved);
  const dirty = JSON.stringify(form) !== JSON.stringify(saved);
  if (settings !== base && !dirty) {
    setBase(settings);
    setForm(toForm(settings));
  }
  const [justSaved, setJustSaved] = useState(false);
  const [refused, setRefused] = useState<Refusal | null>(null);

  const save = useSaveSettings();
  const setTM = useSetTrainingMax();
  // One change at a time: each answers with all the settings, so one landing
  // after the other could put back what the other changed.
  const busy = save.isPending || setTM.isPending;
  // `busy` disables the controls once the screen redraws, and a second tap
  // can land before that, so while a request is out a tap sends nothing.
  const sending = useRef(false);
  const send = (request: (settled: { onSettled: () => void }) => void) => {
    if (sending.current) return;
    sending.current = true;
    request({
      onSettled: () => {
        sending.current = false;
      },
    });
  };

  // News of the last save, or why it failed, was about the form as it stood.
  const edit = (patch: Partial<Form>) => {
    setForm((f) => ({ ...f, ...patch }));
    setJustSaved(false);
    setRefused(null);
    if (save.isError) save.reset();
  };
  const editBBB = (slug: string, patch: Partial<Form['bbb'][string]>) =>
    edit({ bbb: { ...form.bbb, [slug]: { ...form.bbb[slug], ...patch } } });

  // A training max changes future sessions, so each one is its own decision.
  // The answer is the settings with it, which redraws its row.
  const editTM = (slug: string, name: string, value: number) =>
    Alert.prompt(
      `${name} training max`,
      'Changes future sessions only. Finished sessions keep the weights they were lifted at.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Save',
          isPreferred: true,
          onPress: (v?: string) => {
            const n = Number(v);
            if (!(n > 0)) return;
            send((settled) =>
              setTM.mutate(
                { lift: slug, value: n },
                {
                  ...settled,
                  // A 401 has signed out already, and Connect says why.
                  onError: (e) => {
                    if (!(e instanceof ApiError && e.status === 401)) Alert.alert('Training max not saved', e.message);
                  },
                },
              ),
            );
          },
        },
      ],
      'plain-text',
      fmtWeight(value),
      'decimal-pad',
    );

  const submit = () => {
    const input = toInput(form, base.bbb);
    if ('message' in input) {
      setRefused(input);
      return;
    }
    send((settled) =>
      save.mutate(input, {
        ...settled,
        // The form reloads from the settings as the server saved them, which
        // can differ from what was typed: the plates sorted, a blank bar
        // weight kept as it was.
        onSuccess: (answer) => {
          setBase(answer);
          setForm(toForm(answer));
          setJustSaved(true);
        },
      }),
    );
  };
  const problem = refused?.message ?? save.error?.message;

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
      {settings.tms.map((tm) => (
        <Pressable
          key={tm.slug}
          accessibilityRole="button"
          accessibilityLabel={`${tm.name}, training max ${fmtWeight(tm.value)} ${unit}. Change`}
          accessibilityState={{ disabled: busy }}
          disabled={busy}
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
          invalid={refused?.field === 'barWeight'}
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
          invalid={refused?.field === 'rounding'}
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
          invalid={refused?.field === 'plates'}
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
      {base.bbb.map((b) => {
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
                    invalid={refused?.field === `bbb.${b.slug}`}
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

      {problem ? (
        <Notice error style={local.saveError}>
          {problem}
        </Notice>
      ) : null}
      {justSaved && !dirty ? (
        <Notice style={local.saved}>Saved. The next session uses these settings.</Notice>
      ) : (
        <Button
          label={save.isPending ? 'Saving…' : 'Save settings'}
          disabled={!dirty || busy}
          style={problem ? local.afterNotice : local.save}
          onPress={submit}
        />
      )}

      <Server />
    </Page>
  );
}

// The server the app is signed in to, and the way out of it. It needs
// nothing from the settings read, so it shows even when that read fails.
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
  saveError: { marginTop: 18, paddingBottom: 0 },
  afterNotice: { marginTop: 12 },
  saved: { marginTop: 18 },
  value: { marginTop: 0, maxWidth: 210, textAlign: 'right' },
  signOut: { marginTop: 14 },
  footer: { textAlign: 'center', marginTop: 28 },
});
