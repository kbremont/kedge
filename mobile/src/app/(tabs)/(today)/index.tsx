import { router } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useEffect, useState } from 'react';
import { Alert, Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';

import { useAcceptReview, useDashboard, useDecideTMTest, useSkipCycle, useStartWorkout } from '@/api/queries';
import type { Dashboard, NextUp, TMReview, TMTestResult } from '@/api/schema.gen';
import {
  Button,
  Field,
  Heading,
  Intro,
  Label,
  LiftBand,
  LiftChip,
  LoadState,
  Notice,
  Page,
  Panel,
  Stat,
  StatRow,
  TableHead,
  TextAction,
  ToneMark,
  useCardStyles,
} from '@/components/card-back';
import { cycleText, fmtWeight, shortName, times } from '@/lib/format';
import { displayScaleCap, font, liftInk, radius, space } from '@/theme';

export default function TodayScreen() {
  const dashboard = useDashboard();
  return <LoadState query={dashboard}>{(d) => <Today d={d} />}</LoadState>;
}

function Today({ d }: { d: Dashboard }) {
  const { p, s } = useCardStyles();

  if (d.needsSetup) {
    return (
      <Page>
        <Panel style={local.pad}>
          <Intro>Enter a training max for each lift to start the block.</Intro>
          <Button label="Set up" style={{ marginTop: 14 }} onPress={() => router.push('/setup')} />
        </Panel>
      </Page>
    );
  }

  return (
    <Page>
      <View style={local.stack}>
        {d.testResults.map((r) => (
          <TestResult key={r.workoutId} r={r} />
        ))}
        {/* A new review starts from its own proposals. */}
        {d.review ? <Review key={d.review.key} review={d.review} /> : null}
        {d.next ? (
          <NextCard next={d.next} unit={d.unit} held={d.next.blocked} />
        ) : (
          <Panel style={local.pad}>
            <Intro>Nothing scheduled. Check your program in Settings.</Intro>
          </Panel>
        )}
      </View>

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
              <Label>{shortName(tm.slug, tm.name)}</Label>
              <SymbolView name="chevron.right" tintColor={p.ink2} size={11} weight="bold" />
            </View>
            <Text style={[local.tmValue, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
              {fmtWeight(tm.value)}
            </Text>
          </Pressable>
        ))}
      </View>

      <Heading title="Recent" />
      {d.recent.length === 0 ? (
        <View style={s.row}>
          <Text style={[s.name, s.muted]}>No sessions yet.</Text>
        </View>
      ) : (
        <>
          <TableHead
            columns={[
              { label: 'Date', style: { width: 64 } },
              { label: 'Session', style: { flex: 1 } },
              { label: 'Top set', style: { textAlign: 'right', marginRight: 19 } },
            ]}
          />
          {d.recent.slice(0, 5).map((h) => (
            <Pressable
              key={h.workoutId}
              accessibilityRole="button"
              accessibilityLabel={`${h.liftName}, ${cycleText(h.cycleLabel)}, ${h.date}, top set ${times(h.summary)}`}
              onPress={() => router.push({ pathname: '/workout/[id]', params: { id: h.workoutId } })}
              style={({ pressed }) => [s.row, pressed && s.pressed]}>
              <Text maxFontSizeMultiplier={displayScaleCap} style={[s.numSmall, s.muted, { width: 64 }]}>
                {h.date}
              </Text>
              <View style={{ flex: 1 }}>
                <View style={local.nameRow}>
                  <LiftChip slug={h.liftSlug} />
                  <Text style={s.name}>{h.liftName}</Text>
                </View>
                <Text style={s.sub}>{cycleText(h.cycleLabel)}</Text>
              </View>
              <Text maxFontSizeMultiplier={displayScaleCap} style={s.num}>
                {times(h.summary)}
              </Text>
              <SymbolView name="chevron.right" tintColor={p.ink2} size={11} weight="bold" style={local.rowChevron} />
            </Pressable>
          ))}
        </>
      )}
    </Page>
  );
}

// The next session in its lift's band. While a training max decision is
// waiting, the start button gives way to what it waits on. Start and Resume
// both ask the server for the session, so a stale card is refused with why.
function NextCard({ next, unit, held }: { next: NextUp; unit: string; held: boolean }) {
  const { p, s } = useCardStyles();
  const start = useStartWorkout();
  const skip = useSkipCycle();
  const busy = start.isPending || skip.isPending;
  const cycle = cycleText(next.cycleLabel);

  // A refusal to start stays beside the hold it ran into, which Today
  // refetches to show, and goes once that hold is settled.
  const { reset } = start;
  useEffect(() => {
    if (!held) reset();
  }, [held, reset]);

  const confirmSkip = () =>
    Alert.alert(`Skip the ${next.cycleLabel}?`, 'The next block starts now.', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Skip it', style: 'destructive', onPress: () => skip.mutate() },
    ]);
  return (
    <Panel>
      <LiftBand
        slug={next.liftSlug}
        title={next.liftName}
        subtitle={next.inProgress ? `${cycle} · In progress` : cycle}
      />
      <StatRow>
        <Stat label="Block" value={String(next.blockNumber)} />
        <Stat label="Round" value={String(next.roundOfBlock)} unit={`of ${next.roundsInBlock}`} />
        <Stat label="TM" value={fmtWeight(next.trainingMax)} unit={unit} last />
      </StatRow>
      {start.error ? (
        <Notice error style={local.cardError}>
          {start.error.message}
        </Notice>
      ) : null}
      {held ? (
        <View style={local.held}>
          <SymbolView name="lock.fill" tintColor={p.ink2} size={15} />
          <Text style={[s.name, s.muted, { flex: 1 }]}>Settle your training maxes above to start.</Text>
        </View>
      ) : (
        <>
          <Button
            label={next.inProgress ? 'Resume session' : 'Start session'}
            disabled={busy}
            style={local.start}
            onPress={() =>
              start.mutate(undefined, {
                onSuccess: (w) => router.push({ pathname: '/workout/[id]', params: { id: w.id } }),
              })
            }
          />
          {next.canSkip ? (
            <TextAction
              label="Skip it and start the next block"
              disabled={busy}
              onPress={confirmSkip}
              style={local.skip}
            />
          ) : null}
          {skip.error ? (
            <Notice error style={local.skipError}>
              {skip.error.message}
            </Notice>
          ) : null}
        </>
      )}
    </Panel>
  );
}

// One lift's TM test, as soon as it is done. A pass is news; a failure asks
// whether to lower the training max, and nothing is written until the lifter
// chooses. Today comes back without the question.
function TestResult({ r }: { r: TMTestResult }) {
  const { p, s } = useCardStyles();
  const indent = useMarkIndent(22);
  const decide = useDecideTMTest();
  const [tm, setTm] = useState(fmtWeight(r.proposed));
  const valid = Number(tm) > 0;
  return (
    <Panel>
      <LiftBand slim slug={r.liftSlug} title={`${r.liftName} TM test`} right={`${fmtWeight(r.tested)} × ${r.reps}`} />
      <View style={local.result}>
        <View style={local.verdictRow}>
          <ToneMark tone={r.tone} line={22} />
          <Text style={[s.name, local.headline]}>{r.headline}</Text>
        </View>
        <Text style={[s.sub, local.detail, { marginLeft: indent }]}>{r.detail}</Text>
      </View>
      {r.needsDecision ? (
        <>
          <View style={[local.decide, { borderTopColor: p.rule }]}>
            <View style={{ flex: 1 }}>
              <Label>Now</Label>
              <View style={local.nowValue}>
                <Text style={s.statValue} maxFontSizeMultiplier={displayScaleCap}>
                  {fmtWeight(r.current)}
                  <Text style={s.statUnit}> {r.unit}</Text>
                </Text>
              </View>
            </View>
            <View style={local.newTm}>
              <Label>New training max</Label>
              <Field
                onPanel
                value={tm}
                onChangeText={setTm}
                keyboardType="decimal-pad"
                accessibilityLabel={`New ${r.liftName} training max, ${r.unit}`}
                invalid={!valid}
                style={local.newTmField}
              />
            </View>
          </View>
          {decide.error ? (
            <Notice error style={local.decideError}>
              {decide.error.message}
            </Notice>
          ) : null}
          <Button
            label={valid ? `Lower to ${tm} ${r.unit}` : 'Lower training max'}
            disabled={!valid || decide.isPending}
            style={local.decideButton}
            onPress={() => decide.mutate({ id: r.workoutId, action: 'lower', tm: Number(tm) })}
          />
          <Button
            secondary
            label={`Keep ${fmtWeight(r.current)} ${r.unit}`}
            disabled={decide.isPending}
            style={[local.decideButton, local.lastButton]}
            onPress={() => decide.mutate({ id: r.workoutId, action: 'keep' })}
          />
        </>
      ) : null}
    </Panel>
  );
}

// The start of a block: a training max for every lift, each with its evidence,
// all editable. The block waits until they are accepted, as edited.
function Review({ review }: { review: TMReview }) {
  const { p, s } = useCardStyles();
  const indent = useMarkIndent(18);
  const accept = useAcceptReview();
  const [values, setValues] = useState(() =>
    Object.fromEntries(review.rows.map((r) => [r.slug, fmtWeight(r.proposed)])),
  );
  const valid = review.rows.every((r) => Number(values[r.slug]) > 0);
  const tms = Object.fromEntries(review.rows.map((r) => [r.slug, Number(values[r.slug])]));
  return (
    <Panel>
      <View style={[local.inkBand, { backgroundColor: p.button }]}>
        <Text
          style={[local.inkTitle, { color: p.onButton }]}
          accessibilityRole="header"
          maxFontSizeMultiplier={displayScaleCap}>
          {review.title}
        </Text>
        <Text style={[local.inkSub, { color: p.onButton }]}>{review.intro}</Text>
      </View>
      <TableHead
        style={local.reviewHead}
        columns={[
          { label: 'Lift', style: { flex: 1 } },
          { label: 'Now', style: local.cNow },
          { label: `New, ${review.unit}`, style: local.cNew },
        ]}
      />
      {review.rows.map((row) => (
        <View key={row.slug} style={[local.reviewRow, { borderBottomColor: p.rule }]}>
          <View style={local.reviewTop}>
            <View style={[local.nameRow, { flex: 1 }]}>
              <LiftChip slug={row.slug} />
              <Text style={s.name}>{row.name}</Text>
            </View>
            <Text style={[s.num, s.muted, local.cNow]} maxFontSizeMultiplier={displayScaleCap}>
              {fmtWeight(row.current)}
            </Text>
            <Field
              onPanel
              value={values[row.slug]}
              onChangeText={(v) => setValues((x) => ({ ...x, [row.slug]: v }))}
              keyboardType="decimal-pad"
              accessibilityLabel={`New ${row.name} training max, ${review.unit}`}
              invalid={!(Number(values[row.slug]) > 0)}
              style={[local.cNew, local.reviewField]}
            />
          </View>
          <View style={[local.verdictRow, local.reviewVerdict]}>
            <ToneMark tone={row.tone} size={18} line={18} />
            <Text style={[local.verdict, { color: p.ink }]}>{row.verdict}</Text>
          </View>
          <Text style={[s.sub, local.evidence, { marginLeft: indent }]}>{row.evidence}</Text>
        </View>
      ))}
      {accept.error ? (
        <Notice error style={local.cardError}>
          {accept.error.message}
        </Notice>
      ) : null}
      <Button
        label="Accept training maxes"
        disabled={!valid || accept.isPending}
        style={local.start}
        onPress={() => accept.mutate({ key: review.key, tms })}
      />
    </Panel>
  );
}

// Lines under a verdict start where its words do, past the tone mark, which
// grows with the text up to 2x.
function useMarkIndent(size: number) {
  return size * Math.min(useWindowDimensions().fontScale, 2) + 10;
}

const hairline = StyleSheet.hairlineWidth * 2;

const local = StyleSheet.create({
  stack: { gap: space.gap },
  pad: { padding: 16 },
  start: { margin: 12 },
  skip: { marginTop: -6, marginBottom: 4 },
  cardError: { marginHorizontal: 16, paddingTop: 12, paddingBottom: 0 },
  skipError: { marginHorizontal: 16, paddingTop: 0, paddingBottom: 14 },
  held: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingHorizontal: 16, paddingVertical: 18 },
  result: { paddingHorizontal: 14, paddingTop: 12, paddingBottom: 14 },
  verdictRow: { flexDirection: 'row', alignItems: 'flex-start', gap: 10 },
  headline: { flex: 1, lineHeight: 22 },
  detail: { marginTop: 4, lineHeight: 18 },
  decide: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 12,
    marginHorizontal: 14,
    paddingTop: 12,
    borderTopWidth: hairline,
  },
  newTm: { width: 150 },
  newTmField: { marginTop: 4 },
  nowValue: { height: 52, marginTop: 4, justifyContent: 'center' },
  decideButton: { marginHorizontal: 12, marginTop: 12 },
  decideError: { marginHorizontal: 14, paddingBottom: 0 },
  lastButton: { marginTop: 8, marginBottom: 12 },
  inkBand: { paddingHorizontal: 16, paddingTop: 14, paddingBottom: 14 },
  inkTitle: { fontFamily: font.black, fontSize: 30, lineHeight: 32, letterSpacing: 0.4, textTransform: 'uppercase' },
  inkSub: { fontSize: 15, fontWeight: '600', lineHeight: 20, marginTop: 6, opacity: 0.9 },
  reviewHead: { paddingHorizontal: 14 },
  reviewRow: { paddingHorizontal: 14, paddingVertical: 12, borderBottomWidth: hairline },
  reviewTop: { flexDirection: 'row', alignItems: 'center' },
  reviewVerdict: { marginTop: 8 },
  verdict: { flex: 1, fontSize: 15, fontWeight: '600', lineHeight: 18 },
  evidence: { marginTop: 4 },
  cNow: { width: 56, textAlign: 'right' },
  cNew: { width: 96, marginLeft: 12, textAlign: 'right' },
  reviewField: { height: 48, fontSize: 24, borderRadius: radius.panel },
  tmRow: { flexDirection: 'row', borderBottomWidth: hairline },
  tmCol: { flex: 1, paddingTop: 10, paddingBottom: 8, paddingHorizontal: 10 },
  tmColRule: { borderLeftWidth: hairline },
  tmBar: { height: 5, borderRadius: 1, marginBottom: 8 },
  tmLabel: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  tmValue: { fontFamily: font.heavy, fontSize: 34, fontVariant: ['tabular-nums'] },
  nameRow: { flexDirection: 'row', alignItems: 'center', gap: 7 },
  rowChevron: { marginLeft: 8 },
});
