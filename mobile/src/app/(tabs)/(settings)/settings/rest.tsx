import { Host, Stepper } from '@expo/ui/swift-ui';
import { labelsHidden } from '@expo/ui/swift-ui/modifiers';
import { StyleSheet, Text, useColorScheme } from 'react-native';

import { useSettings } from '@/api/queries';
import type { Settings } from '@/api/schema.gen';
import { LoadState, useCardStyles } from '@/components/card-back';
import { Row, SectionPage, SectionTitle, useSettingsSection } from '@/components/settings';
import { fmtClock } from '@/lib/format';
import { displayScaleCap } from '@/theme';

export default function RestScreen() {
  const settings = useSettings();
  return (
    <>
      <SectionTitle title="Rest" />
      <LoadState query={settings}>{(s) => <Rest settings={s} />}</LoadState>
    </>
  );
}

const fields = ['restWarmupSeconds', 'restMainSeconds', 'restSuppSeconds'] as const;

const rest: { key: (typeof fields)[number]; label: string; sub?: string }[] = [
  { key: 'restWarmupSeconds', label: 'After warmups' },
  { key: 'restMainSeconds', label: 'After main sets' },
  { key: 'restSuppSeconds', label: 'After supplemental', sub: 'BBB and FSL' },
];

function Rest({ settings }: { settings: Settings }) {
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const section = useSettingsSection(settings, fields);
  const { form, edit } = section;

  return (
    <SectionPage section={section}>
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
    </SectionPage>
  );
}

const local = StyleSheet.create({
  clock: { minWidth: 48, textAlign: 'right' },
});
