import { SymbolView } from 'expo-symbols';
import { Alert, Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';

import { ApiError } from '@/api/client';
import { useSetTrainingMax, useSettings } from '@/api/queries';
import type { Settings } from '@/api/schema.gen';
import { Intro, LiftChip, LoadState, Page, useCardStyles } from '@/components/card-back';
import { SectionTitle, settingStyles, useSendOnce } from '@/components/settings';
import { fmtWeight, parseDecimal, weightField } from '@/lib/format';
import { displayScaleCap, font } from '@/theme';

export default function TrainingMaxesScreen() {
  const settings = useSettings();
  return (
    <>
      <SectionTitle title="Training maxes" />
      <LoadState query={settings}>{(s) => <TrainingMaxes settings={s} />}</LoadState>
    </>
  );
}

// Each training max saves through its own prompt, so the page has no Save.
function TrainingMaxes({ settings }: { settings: Settings }) {
  const { p, s } = useCardStyles();
  const stacked = useWindowDimensions().fontScale > 1.5;
  const unit = settings.unit;

  const setTM = useSetTrainingMax();
  const busy = setTM.isPending;
  const send = useSendOnce();

  // A training max changes future sessions, so each one is its own decision.
  // The answer is the settings with it, which redraws its row.
  const editTM = (slug: string, name: string, value: number) => {
    Alert.prompt(
      `${name} training max`,
      'Changes future sessions only. Finished sessions keep the weights they were lifted at.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Save',
          isPreferred: true,
          onPress: (v?: string) => {
            const n = parseDecimal(v ?? '');
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
      weightField(value),
      'decimal-pad',
    );
  };

  return (
    <Page>
      <Intro style={settingStyles.intro}>Changes future sessions only. Finished sessions keep their weights.</Intro>
      {settings.tms.map((tm) => (
        <Pressable
          key={tm.slug}
          accessibilityRole="button"
          accessibilityLabel={`${tm.name}, training max ${fmtWeight(tm.value)} ${unit}. Change`}
          accessibilityState={{ disabled: busy }}
          disabled={busy}
          onPress={() => editTM(tm.slug, tm.name, tm.value)}
          style={({ pressed }) => [s.row, stacked && settingStyles.stackedRow, pressed && s.pressed]}>
          <View style={settingStyles.nameRow}>
            <LiftChip slug={tm.slug} />
            <Text style={s.name}>{tm.name}</Text>
          </View>
          <View style={[settingStyles.control, stacked && settingStyles.stackedControl]}>
            <Text style={s.num} maxFontSizeMultiplier={stacked ? undefined : displayScaleCap}>
              {fmtWeight(tm.value)}
              <Text style={[local.unit, { color: p.ink2 }]}> {unit}</Text>
            </Text>
            <SymbolView name="square.and.pencil" tintColor={p.ink2} size={16} style={local.edit} />
          </View>
        </Pressable>
      ))}
    </Page>
  );
}

const local = StyleSheet.create({
  unit: { fontFamily: font.semi, fontSize: 17 },
  edit: { marginLeft: 4 },
});
