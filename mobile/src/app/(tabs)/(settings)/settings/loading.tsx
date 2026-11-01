import { StyleSheet, Switch, Text, View } from 'react-native';

import { useSettings } from '@/api/queries';
import type { Settings } from '@/api/schema.gen';
import { Field, LoadState, useCardStyles } from '@/components/card-back';
import { Row, SectionPage, SectionTitle, settingStyles, Unit, useSettingsSection } from '@/components/settings';

export default function LoadingScreen() {
  const settings = useSettings();
  return (
    <>
      <SectionTitle title="Loading" />
      <LoadState query={settings}>{(s) => <Loading settings={s} />}</LoadState>
    </>
  );
}

const fields = ['barWeight', 'rounding', 'plates', 'warmups'] as const;

function Loading({ settings }: { settings: Settings }) {
  const { p, s } = useCardStyles();
  const section = useSettingsSection(settings, fields);
  const { form, edit, refused } = section;
  const unit = settings.unit;

  return (
    <SectionPage section={section}>
      <Row label="Bar">
        <Field
          value={form.barWeight}
          onChangeText={(barWeight) => edit({ barWeight })}
          keyboardType="decimal-pad"
          accessibilityLabel={`Bar weight, ${unit}`}
          invalid={refused?.field === 'barWeight'}
          style={settingStyles.small}
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
          style={settingStyles.small}
        />
        <Unit>{unit}</Unit>
      </Row>
      <View style={[settingStyles.block, { borderBottomColor: p.rule }]}>
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
    </SectionPage>
  );
}

const local = StyleSheet.create({
  plates: { marginTop: 8, fontSize: 24 },
});
