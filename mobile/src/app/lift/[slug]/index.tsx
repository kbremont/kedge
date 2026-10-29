import { router, Stack, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useState, type ReactNode } from 'react';
import { Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';
import Swipeable from 'react-native-gesture-handler/ReanimatedSwipeable';

import type { PastEstimate } from '@/api/schema.gen';
import {
  Heading,
  Intro,
  Page,
  Panel,
  Stat,
  StatRow,
  TableHead,
  Totals,
  useCardStyles,
} from '@/components/card-back';
import { LiftChart } from '@/components/lift-chart';
import { useOverrides } from '@/gallery/store';
import { fmtWeight } from '@/lib/format';
import { useLift } from '@/sample/use-screen';
import { displayScaleCap, liftInk } from '@/theme';

export default function LiftScreen() {
  const { slug, back } = useLocalSearchParams<{ slug: string; back?: string }>();
  const { version } = useOverrides();
  const l = useLift(slug);
  return (
    <>
      <Stack.Screen options={{ title: l.name, headerBackTitle: back ?? 'Today' }} />
      <LiftPage key={version} l={l} />
    </>
  );
}

function LiftPage({ l }: { l: ReturnType<typeof useLift> }) {
  const { p, s } = useCardStyles();
  const { fontScale } = useWindowDimensions();
  const stacked = fontScale > 1.5;
  // Sample only: DELETE /lifts/{slug}/estimates/{id} answers with the lift.
  const [deleted, setDeleted] = useState<number[]>([]);
  const estimates = l.pastEstimates.filter((e) => !deleted.includes(e.id));
  const best = l.bestE1rm > 0 ? fmtWeight(l.bestE1rm) : '—';
  return (
    <Page>
      <Panel>
        <View style={[local.bar, { backgroundColor: liftInk(p, l.slug).band }]} />
        <StatRow>
          <Stat label="Training max" value={fmtWeight(l.currentTm)} unit={l.unit} />
          <Stat label="Best est. max" value={best} unit={l.bestE1rm > 0 ? l.unit : undefined} last />
        </StatRow>
      </Panel>

      <Heading title="Estimated max" />
      {l.chart.points.length > 0 ? (
        <Panel style={local.chart}>
          <LiftChart chart={l.chart} slug={l.slug} />
        </Panel>
      ) : (
        <Panel style={[local.chart, local.chartEmpty]}>
          <Intro>
            Not enough data yet. The TM test and the anchor’s PR sets add points as you train, and you can add
            estimates from before the app below.
          </Intro>
        </Panel>
      )}

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
      {estimates.length === 0 ? (
        <View style={s.row}>
          <Text style={[s.name, s.muted]}>None yet.</Text>
        </View>
      ) : stacked ? null : (
        <TableHead
          columns={[
            { label: 'Date', style: { flex: 1 } },
            { label: 'Set', style: local.cSet },
            { label: 'Est. max', style: local.cEst },
          ]}
        />
      )}
      {[...estimates].reverse().map((e) => {
        const set = `${fmtWeight(e.weight)} × ${e.reps}`;
        const est = fmtWeight(e.estimate);
        return (
          <SwipeToDelete key={e.id} e={e} unit={l.unit} onDelete={() => setDeleted((x) => [...x, e.id])}>
            {stacked ? (
              <View style={[local.stackedRow, { borderBottomColor: p.rule }]}>
                <Text style={s.name}>{e.date}</Text>
                {/* Numbers get their own line here, so they scale fully. */}
                <View style={local.stackedNums}>
                  <Text style={s.num}>{set}</Text>
                  <Text style={s.num}>{est}</Text>
                </View>
              </View>
            ) : (
              <View style={s.row}>
                <Text style={[s.name, { flex: 1 }]}>{e.date}</Text>
                <Text style={[s.num, local.cSet]} maxFontSizeMultiplier={displayScaleCap}>
                  {set}
                </Text>
                <Text style={[s.num, local.cEst]} maxFontSizeMultiplier={displayScaleCap}>
                  {est}
                </Text>
              </View>
            )}
          </SwipeToDelete>
        );
      })}
      {l.bestE1rm > 0 ? <Totals label="Best, all sources" value={best} /> : null}

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

// A past estimate swipes left to a Delete printed in danger, as iOS lists do.
function SwipeToDelete({
  e,
  unit,
  onDelete,
  children,
}: {
  e: PastEstimate;
  unit: string;
  onDelete: () => void;
  children: ReactNode;
}) {
  const { p } = useCardStyles();
  return (
    <Swipeable
      friction={1.5}
      rightThreshold={40}
      overshootRight={false}
      renderRightActions={() => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Delete the estimate of ${e.date}`}
          onPress={onDelete}
          style={({ pressed }) => [local.delete, { backgroundColor: p.danger }, pressed && { opacity: 0.8 }]}>
          <SymbolView name="trash" tintColor={p.ground} size={18} weight="semibold" />
          <Text style={[local.deleteText, { color: p.ground }]}>Delete</Text>
        </Pressable>
      )}>
      <View
        style={{ backgroundColor: p.ground }}
        accessible
        accessibilityLabel={`${e.date}, ${fmtWeight(e.weight)} × ${e.reps}, estimated max ${fmtWeight(e.estimate)} ${unit}`}
        accessibilityActions={[{ name: 'delete', label: 'Delete' }]}
        onAccessibilityAction={(ev) => ev.nativeEvent.actionName === 'delete' && onDelete()}>
        {children}
      </View>
    </Swipeable>
  );
}

const local = StyleSheet.create({
  bar: { height: 6 },
  chart: { padding: 12, marginTop: 12 },
  chartEmpty: { padding: 16 },
  delete: { width: 96, alignItems: 'center', justifyContent: 'center', gap: 3 },
  deleteText: { fontSize: 13, fontWeight: '600' },
  cSet: { width: 92, textAlign: 'right' },
  cEst: { width: 88, textAlign: 'right' },
  stackedRow: { paddingVertical: 10, borderBottomWidth: StyleSheet.hairlineWidth * 2 },
  stackedNums: { flexDirection: 'row', justifyContent: 'space-between', marginTop: 2 },
});
