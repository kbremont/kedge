import { Host, Picker, Text as SwiftText } from '@expo/ui/swift-ui';
import { pickerStyle, tag } from '@expo/ui/swift-ui/modifiers';
import { StyleSheet, Text, useColorScheme, View } from 'react-native';

import { useSettings } from '@/api/queries';
import type { BBBMode, Settings } from '@/api/schema.gen';
import { Field, Intro, LiftChip, LoadState, useCardStyles } from '@/components/card-back';
import { SectionPage, SectionTitle, settingStyles, Unit, useSettingsSection } from '@/components/settings';

export default function SupplementalScreen() {
  const settings = useSettings();
  return (
    <>
      <SectionTitle title="Supplemental" />
      <LoadState query={settings}>{(s) => <Supplemental settings={s} />}</LoadState>
    </>
  );
}

const fields = ['bbb'] as const;

const modeLabel: Record<BBBMode, string> = { program: 'Program', percent: '% of TM', weight: 'Weight' };

function Supplemental({ settings }: { settings: Settings }) {
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const section = useSettingsSection(settings, fields);
  const { base, form, editBBB, refused } = section;
  const unit = settings.unit;

  return (
    <SectionPage section={section}>
      <Intro style={settingStyles.intro}>
        The weight of the leaders’ BBB sets. Program follows each cycle’s own percentage.
      </Intro>
      {base.bbb.map((b) => {
        const v = form.bbb[b.slug];
        return (
          <View key={b.slug} style={[settingStyles.block, { borderBottomColor: p.rule }]}>
            <View style={local.bbbTop}>
              <View style={settingStyles.nameRow}>
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
                    style={settingStyles.small}
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
    </SectionPage>
  );
}

const local = StyleSheet.create({
  bbbTop: { flexDirection: 'row', alignItems: 'center', minHeight: 46 },
  bbbValue: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  segments: { marginTop: 10 },
});
