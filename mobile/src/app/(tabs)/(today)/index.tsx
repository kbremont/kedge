import { router } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import {
  Button,
  Heading,
  Label,
  LiftBand,
  LiftChip,
  Page,
  Panel,
  Stat,
  StatRow,
  TableHead,
  useCardStyles,
} from '@/components/card-back';
import { cycleText, fmtWeight, times } from '@/lib/format';
import { sample } from '@/sample';
import { displayScaleCap, font, liftInk, type LiftSlug } from '@/theme';

const shortName: Record<LiftSlug, string> = {
  press: 'Press',
  deadlift: 'Deadlift',
  bench: 'Bench',
  squat: 'Squat',
};

export default function TodayScreen() {
  const { p, s } = useCardStyles();
  const d = sample.dashboard;
  const next = d.next;
  return (
    <Page>
      {next ? (
        <Panel>
          <LiftBand slug={next.liftSlug} title={next.liftName} subtitle={cycleText(next.cycleLabel)} />
          <StatRow>
            <Stat label="Block" value={String(next.blockNumber)} />
            <Stat label="Round" value={String(next.roundOfBlock)} unit={`of ${next.roundsInBlock}`} />
            <Stat label="TM" value={fmtWeight(next.trainingMax)} unit={d.unit} last />
          </StatRow>
          <Button
            label={next.inProgress ? 'Resume session' : 'Start session'}
            style={local.start}
            onPress={() => router.push(`/workout/${sample.workout.id}`)}
          />
        </Panel>
      ) : null}

      <Heading title="Training maxes" />
      <View style={[local.tmRow, { borderBottomColor: p.rule }]}>
        {d.tms.map((tm, i) => (
          <Pressable
            key={tm.slug}
            accessibilityRole="button"
            accessibilityLabel={`${tm.name}, training max ${fmtWeight(tm.value)} ${d.unit}`}
            style={({ pressed }) => [
              local.tmCol,
              i > 0 && [local.tmColRule, { borderLeftColor: p.rule }],
              pressed && s.pressed,
            ]}
            onPress={() => router.push(`/lift/${tm.slug}`)}>
            <View style={[local.tmBar, { backgroundColor: liftInk(p, tm.slug).band }]} />
            <View style={local.tmLabel}>
              <Label>{shortName[tm.slug as LiftSlug] ?? tm.name}</Label>
              <SymbolView name="chevron.right" tintColor={p.ink2} size={11} weight="bold" />
            </View>
            <Text style={[local.tmValue, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
              {fmtWeight(tm.value)}
            </Text>
          </Pressable>
        ))}
      </View>

      <Heading title="Recent" />
      <TableHead
        columns={[
          { label: 'Date', style: { width: 64 } },
          { label: 'Session', style: { flex: 1 } },
          { label: 'Top set', style: { textAlign: 'right' } },
        ]}
      />
      {d.recent.slice(0, 5).map((h) => (
        <View key={h.workoutId} style={s.row}>
          <Text maxFontSizeMultiplier={displayScaleCap} style={[s.numSmall, s.muted, { width: 64 }]}>{h.date}</Text>
          <View style={{ flex: 1 }}>
            <View style={local.nameRow}>
              <LiftChip slug={h.liftSlug} />
              <Text style={s.name}>{h.liftName}</Text>
            </View>
            <Text style={s.sub}>{cycleText(h.cycleLabel)}</Text>
          </View>
          <Text maxFontSizeMultiplier={displayScaleCap} style={s.num}>{times(h.summary)}</Text>
        </View>
      ))}
    </Page>
  );
}

const local = StyleSheet.create({
  start: { margin: 12 },
  tmRow: { flexDirection: 'row', borderBottomWidth: StyleSheet.hairlineWidth * 2 },
  tmCol: { flex: 1, paddingTop: 10, paddingBottom: 8, paddingHorizontal: 10 },
  tmColRule: { borderLeftWidth: StyleSheet.hairlineWidth * 2 },
  tmBar: { height: 5, borderRadius: 1, marginBottom: 8 },
  tmLabel: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  tmValue: { fontFamily: font.heavy, fontSize: 34, fontVariant: ['tabular-nums'] },
  nameRow: { flexDirection: 'row', alignItems: 'center', gap: 7 },
});
