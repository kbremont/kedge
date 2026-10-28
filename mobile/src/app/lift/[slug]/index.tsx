import { router, Stack, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';

import {
  Heading,
  Page,
  Panel,
  Stat,
  StatRow,
  TableHead,
  Totals,
  useCardStyles,
} from '@/components/card-back';
import { LiftChart } from '@/components/lift-chart';
import { fmtWeight } from '@/lib/format';
import { sample } from '@/sample';
import { displayScaleCap, liftInk } from '@/theme';

export default function LiftScreen() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  const { p, s } = useCardStyles();
  const { fontScale } = useWindowDimensions();
  const stacked = fontScale > 1.5;
  const l = sample.lifts[slug] ?? sample.lifts.bench;
  const best = fmtWeight(l.bestE1rm);
  return (
    <Page>
      <Stack.Screen options={{ title: l.name }} />
      <Panel>
        <View style={[local.bar, { backgroundColor: liftInk(p, l.slug).band }]} />
        <StatRow>
          <Stat label="Training max" value={fmtWeight(l.currentTm)} unit={l.unit} />
          <Stat label="Best est. max" value={best} unit={l.unit} last />
        </StatRow>
      </Panel>

      <Heading title="Estimated max" />
      <Panel style={local.chart}>
        <LiftChart chart={l.chart} slug={l.slug} />
      </Panel>

      <Heading
        title="Past estimates"
        action={
          <Pressable
            hitSlop={12}
            accessibilityRole="button"
            accessibilityLabel="Add a past estimate"
            onPress={() => router.push(`/lift/${l.slug}/add-estimate`)}>
            <SymbolView name="plus.circle.fill" tintColor={p.ink} size={28} />
          </Pressable>
        }
      />
      {stacked ? null : (
        <TableHead
          columns={[
            { label: 'Date', style: { flex: 1 } },
            { label: 'Set', style: local.cSet },
            { label: 'Est. max', style: local.cEst },
          ]}
        />
      )}
      {[...l.pastEstimates].reverse().map((e) => {
        const set = `${fmtWeight(e.weight)} × ${e.reps}`;
        const est = fmtWeight(e.estimate);
        return stacked ? (
          <View key={e.id} style={[local.stackedRow, { borderBottomColor: p.rule }]}>
            <Text style={s.name}>{e.date}</Text>
            {/* Numbers get their own line here, so they scale fully. */}
            <View style={local.stackedNums}>
              <Text style={s.num}>{set}</Text>
              <Text style={s.num}>{est}</Text>
            </View>
          </View>
        ) : (
          <View key={e.id} style={s.row}>
            <Text style={[s.name, { flex: 1 }]}>{e.date}</Text>
            <Text style={[s.num, local.cSet]} maxFontSizeMultiplier={displayScaleCap}>
              {set}
            </Text>
            <Text style={[s.num, local.cEst]} maxFontSizeMultiplier={displayScaleCap}>
              {est}
            </Text>
          </View>
        );
      })}
      <Totals label="Best, all sources" value={best} />

      <Heading title="Training max" />
      {l.tmHistory.map((t) => (
        <View key={`${t.date}-${t.value}`} style={s.row}>
          <View style={{ flex: 1 }}>
            <Text style={s.name}>{t.date}</Text>
            {t.note ? <Text style={s.sub}>{t.note}</Text> : null}
          </View>
          <Text style={s.num} maxFontSizeMultiplier={displayScaleCap}>
            {fmtWeight(t.value)}
          </Text>
        </View>
      ))}
    </Page>
  );
}

const local = StyleSheet.create({
  bar: { height: 6 },
  chart: { padding: 12, marginTop: 12 },
  cSet: { width: 92, textAlign: 'right' },
  cEst: { width: 88, textAlign: 'right' },
  stackedRow: { paddingVertical: 10, borderBottomWidth: StyleSheet.hairlineWidth * 2 },
  stackedNums: { flexDirection: 'row', justifyContent: 'space-between', marginTop: 2 },
});
