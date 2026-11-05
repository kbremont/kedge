// Settings' building blocks: the root's section rows, the ruled lines of each
// section's page, and the form those pages share.
import { router, Stack, usePreventRemove } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useRef, useState, type ReactNode } from 'react';
import { Alert, Keyboard, Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';

import { useSaveSettings } from '@/api/queries';
import type { Settings } from '@/api/schema.gen';
import { Notice, Page, useCardStyles } from '@/components/card-back';
import { toForm, toInput, type Form, type Refusal } from '@/lib/settings-form';
import { displayScaleCap, font, usePalette } from '@/theme';

// A section of Settings on the root: its name over what it's set to now, and
// a chevron, since it opens the section's own page. The summary is drawn as
// `children` when it needs more than words, such as the lift chips, and
// `summary` is then what VoiceOver reads.
export function SectionRow({
  label,
  summary,
  children,
  onPress,
}: {
  label: string;
  summary: string;
  children?: ReactNode;
  onPress: () => void;
}) {
  const { p, s } = useCardStyles();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${label}, ${summary}`}
      onPress={onPress}
      style={({ pressed }) => [s.row, local.sectionRow, pressed && s.pressed]}>
      <View style={{ flex: 1 }}>
        <Text style={s.name}>{label}</Text>
        {children ?? <Text style={s.sub}>{summary}</Text>}
      </View>
      <SymbolView name="chevron.right" tintColor={p.ink2} size={11} weight="bold" style={local.chevron} />
    </Pressable>
  );
}

// A section page's title, inline in the navigation bar at the size of the
// app's other pushed pages, where the tab's root has its large one.
export function SectionTitle({ title }: { title: string }) {
  const p = usePalette();
  return (
    <Stack.Screen
      options={{
        title,
        headerLargeTitle: false,
        headerTitleStyle: { fontFamily: font.heavy, fontSize: 20, color: p.ink },
      }}
    />
  );
}

// A section's page: its Save in the navigation bar, and why the last save
// was refused, if it was, over the page's own lines.
export function SectionPage({ section, children }: { section: SettingsSection; children: ReactNode }) {
  return (
    <>
      <SaveButton section={section} />
      <Page>
        {section.problem ? (
          <Notice error style={settingStyles.problem}>
            {section.problem}
          </Notice>
        ) : null}
        {children}
      </Page>
    </>
  );
}

// It saves the page and goes back to the root, so it stays disabled until
// something has changed. It's filled with the ink, and iOS sets its label
// against that, so it isn't given a colour: a label colour of our own came
// out grey on the fill in both appearances.
export function SaveButton({ section }: { section: SettingsSection }) {
  const { p } = useCardStyles();
  return (
    <Stack.Toolbar placement="right">
      <Stack.Toolbar.Button
        variant="done"
        tintColor={p.ink}
        disabled={!section.dirty || section.busy}
        onPress={section.submit}>
        {section.busy ? 'Saving…' : 'Save'}
      </Stack.Toolbar.Button>
    </Stack.Toolbar>
  );
}

// One change at a time: `busy` disables the controls once the screen
// redraws, and a second tap can land before that, so while a request is out
// a tap sends nothing.
export function useSendOnce() {
  const sending = useRef(false);
  return (request: (settled: { onSettled: () => void }) => void) => {
    if (sending.current) return;
    sending.current = true;
    request({
      onSettled: () => {
        sending.current = false;
      },
    });
  };
}

export type SettingsSection = ReturnType<typeof useSettingsSection>;

// The form behind a section's page, which edits `fields` of the settings.
// Save sends the page's edits over the settings as last read, so the other
// sections go back as they're saved, then returns to the root. Leaving with
// edits asks first, by the back button or the swipe.
export function useSettingsSection(settings: Settings, fields: readonly (keyof Form)[]) {
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
  const [refused, setRefused] = useState<Refusal | null>(null);
  // Discard was chosen, so the page goes without asking again.
  const [leaving, setLeaving] = useState(false);

  const save = useSaveSettings();
  const send = useSendOnce();

  // A save on its way can't be taken back, so it doesn't ask either: the
  // answer still reaches the root's summaries.
  const allowLeave = usePreventRemove(dirty && !save.isPending && !leaving, ({ repeat }) => {
    Alert.alert('Discard changes?', 'These settings stay as they were.', [
      { text: 'Keep editing', style: 'cancel' },
      {
        text: 'Discard',
        style: 'destructive',
        onPress: () => {
          setLeaving(true);
          repeat();
        },
      },
    ]);
  });

  // Why the last save failed was about the form as it stood.
  const edit = (patch: Partial<Form>) => {
    setForm((f) => ({ ...f, ...patch }));
    setRefused(null);
    if (save.isError) save.reset();
  };
  const editBBB = (slug: string, patch: Partial<Form['bbb'][string]>) =>
    edit({ bbb: { ...form.bbb, [slug]: { ...form.bbb[slug], ...patch } } });

  const submit = () => {
    const input = toInput(
      { ...toForm(settings), ...Object.fromEntries(fields.map((k) => [k, form[k]])) },
      settings.bbb,
    );
    if ('message' in input) {
      setRefused(input);
      return;
    }
    // So why a save failed isn't behind the keyboard.
    Keyboard.dismiss();
    send((settled) =>
      save.mutate(input, {
        ...settled,
        // The root reads the settings as the server saved them, which can
        // differ from what was typed: the plates sorted, a blank bar weight
        // kept as it was.
        onSuccess: (answer) => {
          setBase(answer);
          setForm(toForm(answer));
          allowLeave();
          router.back();
        },
      }),
    );
  };

  return {
    // The settings the page was loaded from: a lift's name, the unit.
    base,
    form,
    edit,
    editBBB,
    refused,
    problem: refused?.message ?? save.error?.message,
    dirty,
    busy: save.isPending,
    submit,
  };
}

// A ruled settings line: what it is on the left, its control on the right.
// At accessibility text sizes the label goes over the control, so neither
// has to break.
export function Row({ label, sub, children }: { label: string; sub?: string; children: ReactNode }) {
  const { s } = useCardStyles();
  const stacked = useWindowDimensions().fontScale > 1.5;
  return (
    <View style={[s.row, local.settingRow, stacked && settingStyles.stackedRow]}>
      <View style={stacked ? undefined : { flex: 1 }}>
        <Text style={s.name}>{label}</Text>
        {sub ? <Text style={s.sub}>{sub}</Text> : null}
      </View>
      <View style={[settingStyles.control, stacked && settingStyles.stackedControl]}>{children}</View>
    </View>
  );
}

export function Unit({ children }: { children: ReactNode }) {
  const { s } = useCardStyles();
  return (
    <Text style={[s.statUnit, local.fieldUnit]} maxFontSizeMultiplier={displayScaleCap}>
      {children}
    </Text>
  );
}

// The lines and blocks every section's page is ruled in.
export const settingStyles = StyleSheet.create({
  intro: { marginTop: 10, marginBottom: 2 },
  nameRow: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: 7 },
  control: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  stackedRow: { flexDirection: 'column', alignItems: 'stretch', gap: 8 },
  stackedControl: { justifyContent: 'flex-end' },
  small: { width: 84, height: 46, fontSize: 24, textAlign: 'right' },
  block: { paddingVertical: 12, borderBottomWidth: StyleSheet.hairlineWidth * 2 },
  problem: { paddingBottom: 18 },
});

const local = StyleSheet.create({
  sectionRow: { minHeight: 64, gap: 12 },
  chevron: { marginLeft: 8 },
  settingRow: { minHeight: 64, gap: 12 },
  // Wide enough for the widest unit, kg, so every field lines up.
  fieldUnit: { width: 24 },
});
