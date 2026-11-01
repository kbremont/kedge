import { router, Stack, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useRef, type ReactNode } from 'react';
import { Alert, Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';
import Swipeable, { type SwipeableMethods } from 'react-native-gesture-handler/ReanimatedSwipeable';

import { useDeleteEstimate, useLift } from '@/api/queries';
import type { Lift, PastEstimate } from '@/api/schema.gen';
import {
  Heading,
  Intro,
  LoadState,
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
import { displayScaleCap, liftInk } from '@/theme';

export default function LiftScreen() {
  const { slug, back } = useLocalSearchParams<{ slug: string; back?: string }>();
  const lift = useLift(slug);
  return (
    <>
      <Stack.Screen options={{ title: lift.data?.name ?? '', headerBackTitle: back ?? 'Today' }} />
      <LoadState query={lift}>{(l) => <LiftPage slug={slug} l={l} />}</LoadState>
    </>
  );
}

// `slug` is the route's, so a change answers into the cache this page and
// the add sheet read.
function LiftPage({ slug, l }: { slug: string; l: Lift }) {
  const { p, s } = useCardStyles();
  const { fontScale } = useWindowDimensions();
  const stacked = fontScale > 1.5;
  const best = l.bestE1rm > 0 ? fmtWeight(l.bestE1rm) : '—';

  // One delete at a time. The answer is the whole lift, so a second delete
  // sent meanwhile could land before the first and bring its row back.
  // `isPending` stills every row once the page redraws, and a second tap
  // can land before that, so while a request is out a tap sends nothing.
  const remove = useDeleteEstimate(slug);
  const sending = useRef(false);
  const deleteEstimate = (e: PastEstimate, row?: SwipeableMethods) => {
    if (sending.current) return;
    sending.current = true;
    remove.mutate(
      { id: e.id },
      {
        // The row slides back over its Delete, and the alert says why.
        onError: (error) => {
          row?.close();
          Alert.alert('Couldn’t delete the estimate', error.message);
        },
        onSettled: () => {
          sending.current = false;
        },
      },
    );
  };
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
            onPress={() => router.push(`/lift/${slug}/add-estimate`)}>
            <SymbolView name="plus.circle.fill" tintColor={p.ink} size={28} />
          </Pressable>
        }
      />
      {l.pastEstimates.length === 0 ? (
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
      {[...l.pastEstimates].reverse().map((e) => {
        const set = `${fmtWeight(e.weight)} × ${e.reps}`;
        const est = fmtWeight(e.estimate);
        return (
          <SwipeToDelete
            key={e.id}
            e={e}
            unit={l.unit}
            busy={remove.isPending}
            onDelete={(row) => deleteEstimate(e, row)}>
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
// While a delete is on its way the rows hold still: none swipes, and no
// Delete can be tapped, so nothing is sent twice. The row being deleted stays
// open until the answer removes it.
function SwipeToDelete({
  e,
  unit,
  busy,
  onDelete,
  children,
}: {
  e: PastEstimate;
  unit: string;
  busy: boolean;
  onDelete: (row?: SwipeableMethods) => void;
  children: ReactNode;
}) {
  const { p, s } = useCardStyles();
  return (
    <Swipeable
      enabled={!busy}
      friction={1.5}
      rightThreshold={40}
      overshootRight={false}
      renderRightActions={(_progress, _translation, row) => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Delete the estimate of ${e.date}`}
          accessibilityState={{ disabled: busy }}
          disabled={busy}
          onPress={() => onDelete(row)}
          style={({ pressed }) => [
            local.delete,
            { backgroundColor: p.danger },
            pressed && { opacity: 0.8 },
            busy && s.disabled,
          ]}>
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
