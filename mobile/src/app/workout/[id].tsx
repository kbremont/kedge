import { router, Stack, useLocalSearchParams } from 'expo-router';
import { useHeaderHeight } from 'expo-router/react-navigation';
import { SymbolView } from 'expo-symbols';
import { useRef, type ComponentRef } from 'react';
import {
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  useWindowDimensions,
  View,
  type LayoutChangeEvent,
} from 'react-native';

import { useDiscardWorkout, useFinishWorkout, useLogSet, useUndoSet, useWorkout } from '@/api/queries';
import type { Set, Workout } from '@/api/schema.gen';
import { Button, Label, LoadState, Notice, Page, RestBar, Totals, useCardStyles } from '@/components/card-back';
import { cycleText, fmtClock, fmtPct, fmtWeight, plateList, times } from '@/lib/format';
import { useRest } from '@/lib/rest';
import { displayScaleCap, font, liftInk, radius } from '@/theme';

export default function WorkoutScreen() {
  const params = useLocalSearchParams<{ id: string; back?: string }>();
  // The cache keys a session by the API's number, not the route's string.
  const id = Number(params.id);
  const workout = useWorkout(id);
  // A logged set's rest starts as its answer arrives, a moment before the
  // answer reaches the screen. Read here, beside the session's data, a new
  // rest redraws the session too, so the bar and the sets are never from
  // different moments.
  const rest = useRest(id);
  return (
    <>
      <Stack.Screen options={{ title: workout.data?.liftName ?? '', headerBackTitle: params.back ?? 'Today' }} />
      <LoadState query={workout}>{(w) => <Session w={w} rest={rest} />}</LoadState>
    </>
  );
}

function Session({ w, rest }: { w: Workout; rest: { left: number; total: number } }) {
  const { p, s } = useCardStyles();
  const slug = w.liftSlug;
  const ink = liftInk(p, slug);
  const { fontScale, height: windowHeight } = useWindowDimensions();
  // At accessibility text sizes each set stacks its label over its numbers,
  // so labels never outgrow the figures they name.
  const stacked = fontScale > 1.5;
  const headerHeight = useHeaderHeight();
  const scroll = useRef<ComponentRef<typeof ScrollView>>(null);

  const logSet = useLogSet(w.id);
  const undoSet = useUndoSet(w.id);
  const finish = useFinishWorkout(w.id);
  const discard = useDiscardWorkout(w.id);
  // One request at a time: until the server answers, the current set is
  // still the one just tapped, so a second tap must not log it again or
  // reach the set after.
  const busy = logSet.isPending || undoSet.isPending || finish.isPending || discard.isPending;
  // The rest the last set logged here started, timed from the tap. Tapping
  // Log ends the rest before it, and the next one waits until the set is
  // logged, so it never shows over the set just tapped.
  const { left, total } = rest;
  const resting = left > 0 && !logSet.isPending;
  const current = w.completed ? undefined : w.sets.find((x) => x.idx === w.currentIdx);
  // A session opened from history: every set as it was lifted, and the PR
  // set's result as the bottom line.
  const pr = w.completed ? w.sets.find((x) => x.isAmrap && x.done) : undefined;
  // The set logged before the current one can be taken back.
  const undoable = current ? w.sets.findLast((x) => x.done && x.idx < current.idx) : undefined;

  // A new set request replaces the news of the last one that failed.
  const log = (set: Set, reps: number) => {
    undoSet.reset();
    logSet.mutate({ idx: set.idx, reps });
  };

  const undo = (set: Set) => {
    logSet.reset();
    undoSet.mutate({ idx: set.idx });
  };

  const confirmDiscard = () =>
    Alert.alert('Discard this session?', 'The sets logged in it are removed. This can’t be undone.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Discard session',
        style: 'destructive',
        onPress: () => discard.mutate(undefined, { onSuccess: () => router.back() }),
      },
    ]);

  // Keeps the current set and its Log button on screen as the session moves
  // down the table: once they would end below the fold, the table scrolls to
  // put the rest bar under the header.
  const keepInView = (e: LayoutChangeEvent) => {
    const { y, height } = e.nativeEvent.layout;
    if (y + height > windowHeight - headerHeight - 40) {
      scroll.current?.scrollTo({ y: Math.max(0, y - headerHeight - 8) });
    }
  };

  return (
    <Page ref={scroll}>
      <View style={[local.band, { backgroundColor: ink.band }]}>
        <View style={local.bandTop}>
          <Text style={[local.bandText, { color: ink.onBand }]} maxFontSizeMultiplier={displayScaleCap}>
            {cycleText(w.cycleLabel)}
          </Text>
          <Text style={[local.bandText, { color: ink.onBand }]} maxFontSizeMultiplier={displayScaleCap}>
            TM {fmtWeight(w.trainingMax)} {w.unit}
          </Text>
        </View>
        {stacked ? null : (
          <View style={local.cols}>
            {[
              { label: 'Set', style: local.cSet },
              { label: '%', style: local.cPct },
              { label: w.unit, style: local.cLb },
              { label: 'Reps', style: local.cReps },
            ].map((c) => (
              <Label key={c.label} style={[c.style, { color: ink.onBand, opacity: 0.85 }]}>
                {c.label}
              </Label>
            ))}
          </View>
        )}
      </View>

      {w.sets.map((set) => {
        if (set.idx === current?.idx) {
          return (
            <View key={set.idx} onLayout={keepInView}>
              {resting ? (
                <RestBar slug={slug} left={left} total={total} clock={fmtClock(left)} />
              ) : (
                <View style={local.restGap} />
              )}
              {undoable ? (
                <View style={local.loggedRow}>
                  <Text style={[s.sub, local.loggedText]}>
                    Logged {fmtWeight(undoable.weight)} × {undoable.actualReps}
                  </Text>
                  <Pressable
                    accessibilityRole="button"
                    accessibilityLabel={`Undo ${fmtWeight(undoable.weight)} × ${undoable.actualReps}`}
                    accessibilityState={{ disabled: busy }}
                    disabled={busy}
                    onPress={() => undo(undoable)}
                    style={({ pressed }) => [local.undo, pressed && s.pressed, busy && s.disabled]}>
                    <SymbolView name="arrow.uturn.backward" tintColor={p.ink} size={14} weight="bold" />
                    <Text style={[local.undoText, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
                      Undo
                    </Text>
                  </Pressable>
                </View>
              ) : null}
              {undoSet.error ? (
                <Notice error style={local.undoError}>
                  {undoSet.error.message}
                </Notice>
              ) : null}
              <View style={[local.current, { backgroundColor: p.stock }]}>
                <View style={local.currentTop}>
                  <Text
                    style={[local.currentWeight, { color: p.ink }]}
                    maxFontSizeMultiplier={displayScaleCap}
                    accessibilityLabel={`${fmtWeight(set.weight)} ${w.unit}, ${set.repTarget} reps`}>
                    {fmtWeight(set.weight)}
                    <Text style={[local.currentUnit, { color: p.ink2 }]}> {w.unit}</Text>
                  </Text>
                  <Text style={[local.currentReps, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
                    <Text style={[local.currentTimes, { color: p.ink2 }]}>× </Text>
                    {set.repTarget}
                  </Text>
                </View>
                <Text style={[local.currentMeta, { color: p.ink2 }]}>
                  {set.tag} · {fmtPct(set.pct)} of training max
                  {set.isAmrap ? ' · PR set' : ''}
                </Text>
                <View style={[local.plateRow, { borderTopColor: p.rule }]}>
                  <Label>Per side</Label>
                  <Text style={[local.plateText, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
                    {plateList(set.plates).join('  ·  ') || 'Empty bar'}
                  </Text>
                </View>
                {set.unloadable > 0 ? (
                  <Notice error style={local.short}>
                    {fmtWeight(set.unloadable)} {w.unit} short: your plates can’t load {fmtWeight(set.weight)}{' '}
                    {w.unit} exactly.
                  </Notice>
                ) : null}
                {logSet.error ? (
                  <Notice error style={local.short}>
                    {logSet.error.message}
                  </Notice>
                ) : null}
                {set.asksReps ? (
                  <RepPicker
                    set={set}
                    slug={slug}
                    bestE1rm={w.bestE1rm}
                    unit={w.unit}
                    disabled={busy}
                    onPick={(r) => log(set, r)}
                  />
                ) : (
                  <Button
                    icon="checkmark"
                    label={`Log ${set.targetReps} reps`}
                    disabled={busy}
                    style={local.log}
                    onPress={() => log(set, set.targetReps)}
                  />
                )}
              </View>
            </View>
          );
        }
        return (
          <SetRow
            key={set.idx}
            set={set}
            slug={slug}
            stacked={stacked}
            completed={w.completed}
            bestE1rm={w.bestE1rm}
            unit={w.unit}
          />
        );
      })}

      {w.completed ? (
        pr ? (
          <Totals label="PR set" value={`${fmtWeight(pr.weight)} × ${pr.actualReps}`} />
        ) : null
      ) : current ? (
        <>
          {discard.error ? (
            <Notice error style={local.endError}>
              {discard.error.message}
            </Notice>
          ) : null}
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ disabled: busy }}
            disabled={busy}
            style={[local.discard, busy && s.disabled]}
            onPress={confirmDiscard}>
            <Text style={[local.discardText, { color: p.danger }]}>Discard this session</Text>
          </Pressable>
        </>
      ) : (
        <View onLayout={() => scroll.current?.scrollToEnd()}>
          {finish.error ? (
            <Notice error style={local.endError}>
              {finish.error.message}
            </Notice>
          ) : null}
          <Button
            label="Finish session"
            disabled={busy}
            style={finish.error ? local.afterError : local.finish}
            onPress={() => finish.mutate(undefined, { onSuccess: () => router.back() })}
          />
        </View>
      )}
    </Page>
  );
}

function SetRow({
  set,
  slug,
  stacked,
  completed,
  bestE1rm,
  unit,
}: {
  set: Set;
  slug: string;
  stacked: boolean;
  completed: boolean;
  bestE1rm: number;
  unit: string;
}) {
  const { p, s } = useCardStyles();
  const ink = liftInk(p, slug);
  // In a session the done sets step back; from history every set reads alike.
  const muted = set.done && !completed && s.muted;
  // Stacked rows give the numbers their own line, so they scale fully and
  // stay heavier than the label above them.
  const cap = stacked ? undefined : displayScaleCap;
  const short =
    set.unloadable > 0 ? (
      <SymbolView
        name="exclamationmark.triangle.fill"
        tintColor={p.danger}
        size={13}
        accessibilityLabel={`${fmtWeight(set.unloadable)} ${unit} short`}
      />
    ) : null;
  const reps = completed && set.isAmrap ? (
    // The PR set's result, printed reversed out of the lift's colour.
    <View style={local.repsCell}>
      <View style={[local.prChip, { backgroundColor: ink.band }]}>
        <Text style={[s.num, { color: ink.onBand }]} maxFontSizeMultiplier={cap}>
          {set.actualReps}
        </Text>
      </View>
    </View>
  ) : completed ? (
    <Text style={s.num} maxFontSizeMultiplier={cap}>
      {set.actualReps}
    </Text>
  ) : set.done ? (
    <View style={local.repsCell}>
      <SymbolView name="checkmark" tintColor={p.ink2} size={13} weight="bold" />
      <Text style={[s.num, s.muted]} maxFontSizeMultiplier={cap}>
        {set.actualReps}
      </Text>
    </View>
  ) : set.isAmrap ? (
    // League-leading emphasis: the PR set's target is printed reversed out of
    // the lift's colour.
    <View style={local.repsCell}>
      <View style={[local.prChip, { backgroundColor: ink.band }]}>
        <Text style={[s.num, { color: ink.onBand }]} maxFontSizeMultiplier={cap}>
          {set.repTarget}
        </Text>
      </View>
    </View>
  ) : (
    <Text style={s.num} maxFontSizeMultiplier={cap}>
      {set.repTarget}
    </Text>
  );
  return (
    <View style={[local.setRow, { borderBottomColor: p.rule }]}>
      {stacked ? (
        <>
          <Text style={[s.name, muted]}>
            {set.tag} · {fmtPct(set.pct)}
          </Text>
          <View style={local.stackedNums}>
            <View style={local.weightCell}>
              {short}
              <Text style={[s.num, muted]} maxFontSizeMultiplier={cap}>
                {fmtWeight(set.weight)} {unit}
              </Text>
            </View>
            {reps}
          </View>
        </>
      ) : (
        <View style={local.cols}>
          <Text style={[s.name, local.cSet, muted]}>{set.tag}</Text>
          <Text style={[s.numSmall, local.cPct, muted]} maxFontSizeMultiplier={displayScaleCap}>
            {fmtPct(set.pct)}
          </Text>
          <View style={[local.cLb, local.weightCell]}>
            {short}
            <Text style={[s.num, muted]} maxFontSizeMultiplier={displayScaleCap}>
              {fmtWeight(set.weight)}
            </Text>
          </View>
          <View style={local.cRepsBox}>{reps}</View>
        </View>
      )}
      {set.isAmrap && !set.done && bestE1rm > 0 ? (
        <Text style={[s.sub, local.prNote]}>
          PR set · best estimated max {fmtWeight(bestE1rm)} {unit}
        </Text>
      ) : completed && set.isAmrap ? (
        <Text style={[s.sub, local.prNote]}>PR set · target {times(set.repTarget)}</Text>
      ) : null}
    </View>
  );
}

// One tap per plausible answer instead of a keyboard; the answers that meet
// the prescription are printed in the lift's colour.
function RepPicker({
  set,
  slug,
  bestE1rm,
  unit,
  disabled,
  onPick,
}: {
  set: Set;
  slug: string;
  bestE1rm: number;
  unit: string;
  disabled: boolean;
  onPick: (reps: number) => void;
}) {
  const { p, s } = useCardStyles();
  const ink = liftInk(p, slug);
  const perRow = 5;
  const rows = Array.from({ length: Math.ceil(set.repChoices.length / perRow) }, (_, i) =>
    set.repChoices.slice(i * perRow, (i + 1) * perRow),
  );
  return (
    <View style={local.picker}>
      <Text style={[s.name, { marginBottom: 2 }]}>How many reps? Target {set.repTarget}.</Text>
      {bestE1rm > 0 ? (
        <Text style={s.sub}>
          Best estimated max {fmtWeight(bestE1rm)} {unit}
        </Text>
      ) : null}
      <View style={[local.repGrid, disabled && s.disabled]}>
        {rows.map((row, i) => (
          <View key={i} style={local.repRow}>
            {row.map((c) => (
              <Pressable
                key={c.reps}
                accessibilityRole="button"
                accessibilityLabel={`Log ${c.reps} reps${c.target ? ', meets the target' : ''}`}
                accessibilityState={{ disabled }}
                disabled={disabled}
                onPress={() => onPick(c.reps)}
                style={({ pressed }) => [
                  local.rep,
                  { backgroundColor: p.ground },
                  { borderColor: c.target ? ink.line : p.rule },
                  pressed && { backgroundColor: ink.band },
                ]}>
                {({ pressed }) => (
                  <Text
                    style={[s.num, { color: pressed ? ink.onBand : c.target ? ink.line : p.ink }]}
                    maxFontSizeMultiplier={displayScaleCap}>
                    {c.reps}
                  </Text>
                )}
              </Pressable>
            ))}
            {Array.from({ length: perRow - row.length }, (_, k) => (
              <View key={`pad-${k}`} style={local.repPad} />
            ))}
          </View>
        ))}
      </View>

    </View>
  );
}

const local = StyleSheet.create({
  band: { borderRadius: radius.panel, paddingHorizontal: 14, paddingTop: 9, paddingBottom: 7, marginBottom: 2 },
  bandTop: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  bandText: { fontFamily: font.heavy, fontSize: 20, letterSpacing: 0.6, textTransform: 'uppercase' },
  cols: { flexDirection: 'row', alignItems: 'center', marginTop: 4 },
  cSet: { flex: 1 },
  cPct: { width: 56, textAlign: 'right' },
  cLb: { width: 76, textAlign: 'right' },
  weightCell: { flexDirection: 'row', alignItems: 'center', justifyContent: 'flex-end', gap: 5 },
  cReps: { width: 76, textAlign: 'right' },
  cRepsBox: { width: 76, alignItems: 'flex-end' },
  setRow: { paddingVertical: 8, borderBottomWidth: StyleSheet.hairlineWidth * 2 },
  stackedNums: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 2 },
  repsCell: { flexDirection: 'row', justifyContent: 'flex-end', alignItems: 'center', gap: 6 },
  prChip: { borderRadius: radius.chip, paddingHorizontal: 7, minWidth: 40, alignItems: 'center' },
  prNote: { marginTop: 4, textAlign: 'right' },
  restGap: { height: 12 },
  current: { borderRadius: radius.panel, paddingTop: 10, marginBottom: 4 },
  currentTop: {
    flexDirection: 'row',
    alignItems: 'baseline',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
  },
  currentWeight: { fontFamily: font.black, fontSize: 88, lineHeight: 92 },
  currentUnit: { fontFamily: font.heavy, fontSize: 24 },
  currentReps: { fontFamily: font.black, fontSize: 56 },
  currentTimes: { fontFamily: font.semi, fontSize: 32 },
  loggedRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: -10,
    marginBottom: 2,
  },
  loggedText: { marginTop: 0, flexShrink: 1 },
  undo: { flexDirection: 'row', alignItems: 'center', gap: 6, minHeight: 44, paddingLeft: 12 },
  undoText: { fontFamily: font.heavy, fontSize: 18, letterSpacing: 0.6, textTransform: 'uppercase' },
  undoError: { paddingTop: 0 },
  currentMeta: { fontSize: 15, paddingHorizontal: 16, marginTop: -2 },
  plateRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginTop: 12,
    marginHorizontal: 16,
    paddingTop: 10,
    borderTopWidth: StyleSheet.hairlineWidth * 2,
  },
  plateText: { fontFamily: font.heavy, fontSize: 26, fontVariant: ['tabular-nums'] },
  short: { marginHorizontal: 16, paddingBottom: 0 },
  log: { margin: 12 },
  picker: { padding: 16, paddingTop: 14 },
  repGrid: { gap: 8, marginTop: 10 },
  repRow: { flexDirection: 'row', gap: 8 },
  repPad: { flex: 1 },
  rep: {
    flex: 1,
    height: 52,
    borderWidth: 2,
    borderRadius: radius.panel,
    alignItems: 'center',
    justifyContent: 'center',
  },
  discard: { alignItems: 'center', paddingVertical: 18, marginTop: 12 },
  discardText: { fontSize: 16, fontWeight: '600' },
  finish: { marginTop: 20 },
  endError: { marginTop: 12, paddingBottom: 0 },
  afterError: { marginTop: 12 },
});
