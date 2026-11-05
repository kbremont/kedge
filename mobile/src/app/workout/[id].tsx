import { router, Stack, useLocalSearchParams } from 'expo-router';
import { useHeaderHeight } from 'expo-router/react-navigation';
import { SymbolView } from 'expo-symbols';
import { useRef, useState, type ComponentRef } from 'react';
import {
  Alert,
  Keyboard,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  useWindowDimensions,
  View,
  type LayoutChangeEvent,
  type StyleProp,
  type TextInputProps,
  type TextStyle,
  type ViewStyle,
} from 'react-native';

import { ApiError } from '@/api/client';
import { useDiscardWorkout, useFinishWorkout, useLogSet, usePlates, useUndoSet, useWorkout } from '@/api/queries';
import type { Set, Workout } from '@/api/schema.gen';
import {
  Button,
  Field,
  Label,
  LoadState,
  Notice,
  Page,
  Panel,
  RestBar,
  Totals,
  useCardStyles,
} from '@/components/card-back';
import { cycleText, fmtClock, fmtPct, fmtWeight, parseDecimal, plateList, repsText, times } from '@/lib/format';
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
  // A session the server no longer has. LoadState keeps a screen's data
  // through a failed refetch, but this one can't be logged any more.
  const gone = workout.error instanceof ApiError && workout.error.status === 404;
  const back = params.back ?? 'Today';
  return (
    <>
      <Stack.Screen options={{ title: workout.data?.liftName ?? '', headerBackTitle: back }} />
      {gone ? (
        <Gone back={back} />
      ) : (
        <LoadState query={workout}>{(w) => <Session w={w} rest={rest} />}</LoadState>
      )}
    </>
  );
}

// Discarded somewhere else, such as in the web UI, while it was open here:
// said the way a failed first load is, with the way back.
function Gone({ back }: { back: string }) {
  const { s } = useCardStyles();
  return (
    <Page>
      <Panel style={s.failed}>
        <Notice error style={s.failedNotice}>
          This session was discarded somewhere else, so it’s no longer on your server.
        </Notice>
        <Button label={`Back to ${back}`} style={s.failedButton} onPress={() => router.back()} />
      </Panel>
    </Page>
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
  const frame = useRef({ width: 0, height: 0 });

  const logSet = useLogSet(w.id);
  const undoSet = useUndoSet(w.id);
  const finish = useFinishWorkout(w.id);
  const discard = useDiscardWorkout(w.id);
  // One request at a time: until the server answers, the current set is
  // still the one just tapped, so a second tap must not log it again or
  // reach the set after.
  const busy = logSet.isPending || undoSet.isPending || finish.isPending || discard.isPending;
  // `busy` disables the controls once the screen redraws, and a second tap
  // can land before that, so while a request is out a tap sends nothing.
  const sending = useRef(false);
  const send = (request: (settled: { onSettled: () => void }) => void) => {
    if (sending.current) return;
    sending.current = true;
    request({
      onSettled: () => {
        sending.current = false;
      },
    });
  };
  // The rest the last set logged here started, timed from the tap. Tapping
  // Log ends the rest before it, and the next one waits until the set is
  // logged, so it never shows over the set just tapped.
  const { left, total } = rest;
  const resting = left > 0 && !logSet.isPending;
  const current = w.completed ? undefined : w.sets.find((x) => x.idx === w.currentIdx);
  // A session opened from history: every set as it was lifted, and the PR
  // set's result as the bottom line.
  const pr = w.completed ? w.sets.find((x) => x.isAmrap && x.done) : undefined;
  // The set logged last can be taken back: the one before the current set,
  // or once every set is logged, the last of them, such as a TM test's
  // AMRAP.
  const undoable = w.completed ? undefined : w.sets.findLast((x) => x.done && (!current || x.idx < current.idx));
  // A failure is news about its own set. If the session has moved on since,
  // say by a refetch, it isn't shown beside another.
  const logError = logSet.error && logSet.variables?.idx === current?.idx ? logSet.error : null;
  const undoError = undoSet.error && undoSet.variables?.idx === undoable?.idx ? undoSet.error : null;

  // A new set request replaces the news of the last one that failed.
  const log = (set: Set, reps: number, weight?: number) =>
    send((settled) => {
      undoSet.reset();
      logSet.mutate({ idx: set.idx, reps, weight }, settled);
    });

  const undo = (set: Set) =>
    send((settled) => {
      logSet.reset();
      undoSet.mutate({ idx: set.idx }, settled);
    });

  const end = (mutation: typeof finish) =>
    send((settled) => mutation.mutate(undefined, { ...settled, onSuccess: () => router.back() }));

  const confirmDiscard = () =>
    Alert.alert('Discard this session?', 'The sets logged in it are removed. This can’t be undone.', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Discard session', style: 'destructive', onPress: () => end(discard) },
    ]);

  // Keeps the current set and its Log button on screen as the session moves
  // down the table: once they would end below the fold, the table scrolls to
  // put the rest bar under the header. Near the end of the table that aims
  // past the page's end, and Page leaves scrollTo unclamped for the keyboard,
  // so this zooms to the rect instead, which UIKit keeps within the page.
  const keepInView = (e: LayoutChangeEvent) => {
    const { y, height } = e.nativeEvent.layout;
    if (y + height > windowHeight - headerHeight - 40) {
      scroll.current?.scrollResponderZoomTo({
        x: 0,
        y: Math.max(0, y - headerHeight - 8),
        width: frame.current.width,
        height: frame.current.height,
        animated: true,
      });
    }
  };

  return (
    <Page
      ref={scroll}
      onLayout={(e) => {
        frame.current = e.nativeEvent.layout;
      }}>
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
                <Undo set={undoable} disabled={busy} error={undoError} onPress={() => undo(undoable)} />
              ) : null}
              <CurrentSet
                set={set}
                slug={slug}
                bestE1rm={w.bestE1rm}
                unit={w.unit}
                disabled={busy}
                error={logError}
                refused={logError instanceof ApiError && logError.status === 400 ? logSet.variables : undefined}
                onLog={(reps, weight) => log(set, reps, weight)}
              />
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
          {undoable ? (
            <Undo
              set={undoable}
              disabled={busy}
              error={undoError}
              onPress={() => undo(undoable)}
              style={local.lastLogged}
            />
          ) : null}
          {finish.error ? (
            <Notice error style={local.endError}>
              {finish.error.message}
            </Notice>
          ) : null}
          <Button
            label="Finish session"
            disabled={busy}
            style={finish.error ? local.afterError : undoable ? local.afterLogged : local.finish}
            onPress={() => end(finish)}
          />
        </View>
      )}
    </Page>
  );
}

// The set logged last, with a way to take it back, and why taking it back
// failed if it did.
function Undo({
  set,
  disabled,
  error,
  onPress,
  style,
}: {
  set: Set;
  disabled: boolean;
  error: Error | null;
  onPress: () => void;
  style?: StyleProp<ViewStyle>;
}) {
  const { p, s } = useCardStyles();
  return (
    <>
      <View style={[local.loggedRow, style]}>
        <Text style={[s.sub, local.loggedText]}>
          Logged {fmtWeight(set.weight)} × {set.actualReps}
        </Text>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Undo ${fmtWeight(set.weight)} × ${set.actualReps}`}
          accessibilityState={{ disabled }}
          disabled={disabled}
          onPress={onPress}
          style={({ pressed }) => [local.undo, pressed && s.pressed, disabled && s.disabled]}>
          <SymbolView name="arrow.uturn.backward" tintColor={p.ink} size={14} weight="bold" />
          <Text style={[local.undoText, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
            Undo
          </Text>
        </Pressable>
      </View>
      {error ? (
        <Notice error style={local.undoError}>
          {error.message}
        </Notice>
      ) : null}
    </>
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

// What's typed, as a number, or undefined until it is one, so a blank field
// never logs as 0. How much is the server's to refuse, in its own words.
function typedWeight(text: string): number | undefined {
  const n = parseDecimal(text);
  return Number.isFinite(n) ? n : undefined;
}

function typedReps(text: string): number | undefined {
  const t = text.trim();
  return /^\d+$/.test(t) ? Number(t) : undefined;
}

// A number on the current set that can be changed, in the type it's read in. A
// TextInput gives Yoga no baseline and doesn't size to its text, so a hidden
// copy of the text does both, and the input lies over it. `drop` puts the
// input's digits on the copy's baseline, since it centres them rather than
// honouring a line height, and `rise` puts the underline just under them. Both
// are in points at the base text size, and scale with the type.
function DisplayEntry({
  type,
  drop = 0,
  rise,
  value,
  disabled,
  invalid,
  ...props
}: {
  type: StyleProp<TextStyle>;
  drop?: number;
  rise: number;
  value: string;
  disabled: boolean;
  invalid: boolean;
} & Pick<TextInputProps, 'onChangeText' | 'keyboardType' | 'maxLength' | 'accessibilityLabel'>) {
  const { p } = useCardStyles();
  const { fontScale } = useWindowDimensions();
  const scale = Math.min(fontScale, displayScaleCap);
  return (
    <View>
      <Text
        style={[type, local.entrySizer]}
        numberOfLines={1}
        maxFontSizeMultiplier={displayScaleCap}
        accessibilityElementsHidden
        importantForAccessibility="no-hide-descendants">
        {value || '0'}
      </Text>
      <TextInput
        {...props}
        style={[type, local.entryInput, { color: p.ink, top: drop * scale, bottom: -drop * scale }]}
        value={value}
        editable={!disabled}
        selectTextOnFocus
        selectionColor={p.ink}
        // A number pad has no return key, and with the field above the keyboard
        // the plates and Log can be under it. On iOS this adds a bar with Done.
        returnKeyType="done"
        maxFontSizeMultiplier={displayScaleCap}
      />
      <View
        testID={`${props.accessibilityLabel} line`}
        pointerEvents="none"
        style={[local.entryLine, { bottom: rise * scale, backgroundColor: invalid ? p.danger : p.rule }]}
      />
    </View>
  );
}

// The set at the bar. Its weight and reps start as suggested and can be changed
// on the spot, and nothing is saved until it's logged. It's drawn afresh for
// each set, so an edit never reaches the next one, and undoing a set goes back
// to its suggestion. A PR set or TM test keeps its weight, as the server says,
// because the verdicts read its reps as done at that weight.
function CurrentSet({
  set,
  slug,
  bestE1rm,
  unit,
  disabled,
  error,
  refused,
  onLog,
}: {
  set: Set;
  slug: string;
  bestE1rm: number;
  unit: string;
  disabled: boolean;
  error: Error | null;
  // What the server just refused, outlined only while it's still typed.
  refused?: { reps: number; weight?: number };
  onLog: (reps: number, weight?: number) => void;
}) {
  const { p } = useCardStyles();
  const { fontScale } = useWindowDimensions();
  const [weightText, setWeightText] = useState(String(set.weight));
  const [repsDraft, setRepsDraft] = useState(String(set.targetReps));
  const weight = set.weightEditable ? typedWeight(weightText) : set.weight;
  const reps = typedReps(repsDraft);
  const changed = weight !== set.weight;

  // The suggestion comes with its plates. Any other weight waits for the
  // server's, and shows none until they're for this weight.
  const asked = usePlates(changed ? weight : undefined);
  const load = changed ? asked.data : set;

  const logAt = (count: number) => {
    Keyboard.dismiss();
    onLog(count, changed ? weight : undefined);
  };

  return (
    <View style={[local.current, { backgroundColor: p.stock }]}>
      {/* The weight's line sets this height. Stated, because on first layout
          the baseline of a DisplayEntry leaves the row taller until it's edited. */}
      <View style={[local.currentTop, { height: weightLine * Math.min(fontScale, displayScaleCap) }]}>
        {set.weightEditable ? (
          <View style={local.weightEntry}>
            <DisplayEntry
              type={local.currentWeight}
              drop={-8}
              rise={21}
              value={weightText}
              onChangeText={setWeightText}
              keyboardType="decimal-pad"
              maxLength={7}
              accessibilityLabel={`Weight, ${unit}`}
              disabled={disabled}
              invalid={changed && weight !== undefined && refused?.weight === weight}
            />
            <Text style={[local.currentUnit, { color: p.ink2 }]} maxFontSizeMultiplier={displayScaleCap}>
              {' '}
              {unit}
            </Text>
          </View>
        ) : (
          <Text
            style={[local.currentWeight, { color: p.ink }]}
            maxFontSizeMultiplier={displayScaleCap}
            accessibilityLabel={`${fmtWeight(set.weight)} ${unit}, ${repsText(set.repTarget)}`}>
            {fmtWeight(set.weight)}
            <Text style={[local.currentUnit, { color: p.ink2 }]}> {unit}</Text>
          </Text>
        )}
        {set.asksReps ? (
          <Text style={[local.currentReps, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
            <Text style={[local.currentTimes, { color: p.ink2 }]}>× </Text>
            {set.repTarget}
          </Text>
        ) : (
          <View style={local.repsEntry}>
            <Text style={[local.currentTimes, { color: p.ink2 }]} maxFontSizeMultiplier={displayScaleCap}>
              ×{' '}
            </Text>
            <DisplayEntry
              type={local.currentReps}
              rise={13}
              value={repsDraft}
              onChangeText={setRepsDraft}
              keyboardType="number-pad"
              maxLength={3}
              accessibilityLabel="Reps"
              disabled={disabled}
              invalid={reps !== undefined && reps !== set.targetReps && refused?.reps === reps}
            />
          </View>
        )}
      </View>
      <Text style={[local.currentMeta, { color: p.ink2 }]}>
        {set.tag} · {fmtPct(set.pct)} of training max
        {set.isAmrap ? ' · PR set' : ''}
        {changed ? ` · suggested ${fmtWeight(set.weight)}` : ''}
      </Text>
      <View style={[local.plateRow, { borderTopColor: p.rule }]}>
        <Label>Per side</Label>
        <Text style={[local.plateText, { color: p.ink }]} maxFontSizeMultiplier={displayScaleCap}>
          {load ? plateList(load.plates).join('  ·  ') || 'Empty bar' : '–'}
        </Text>
      </View>
      {load && load.unloadable > 0 && weight !== undefined ? (
        <Notice error style={local.short}>
          {fmtWeight(load.unloadable)} {unit} short: your plates can’t load {fmtWeight(weight)} {unit} exactly.
        </Notice>
      ) : null}
      {!asked.data && asked.error ? (
        <Notice error style={local.short}>
          {asked.error.message}
        </Notice>
      ) : null}
      {error ? (
        <Notice error style={local.short}>
          {error.message}
        </Notice>
      ) : null}
      {set.asksReps ? (
        <RepPicker
          set={set}
          slug={slug}
          bestE1rm={bestE1rm}
          unit={unit}
          disabled={disabled || weight === undefined}
          refused={refused?.reps}
          onPick={logAt}
        />
      ) : (
        <Button
          icon="checkmark"
          label={reps === undefined ? 'Log' : `Log ${repsText(reps)}`}
          disabled={disabled || weight === undefined || reps === undefined}
          style={local.log}
          onPress={() => {
            if (reps !== undefined) logAt(reps);
          }}
        />
      )}
    </View>
  );
}

// One tap per plausible answer, with the answers that meet the prescription
// printed in the lift's colour, and a field for any other: a PR set can beat
// every choice, or fall short of the first.
function RepPicker({
  set,
  slug,
  bestE1rm,
  unit,
  disabled,
  refused,
  onPick,
}: {
  set: Set;
  slug: string;
  bestE1rm: number;
  unit: string;
  disabled: boolean;
  // The count the server just refused, outlined only while it's still typed.
  refused?: number;
  onPick: (reps: number) => void;
}) {
  const { p, s } = useCardStyles();
  const ink = liftInk(p, slug);
  const [typed, setTyped] = useState('');
  const count = typedReps(typed);
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
                accessibilityLabel={`Log ${repsText(c.reps)}${c.target ? ', meets the target' : ''}`}
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
      <Label style={local.typedLabel}>Ran out of buttons?</Label>
      {/* Log sits beside the field because a number pad has no return key:
          whatever brings the field above the keyboard brings Log with it. */}
      <View style={local.typedRow}>
        <Field
          onPanel
          style={local.typedField}
          value={typed}
          onChangeText={setTyped}
          keyboardType="number-pad"
          maxLength={3}
          accessibilityLabel="Reps"
          editable={!disabled}
          invalid={count !== undefined && count === refused}
        />
        <Button
          label="Log"
          disabled={disabled || count === undefined}
          style={local.typedLog}
          onPress={() => {
            if (count !== undefined) onPick(count);
          }}
        />
      </View>
    </View>
  );
}

const weightLine = 92;

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
  currentWeight: { fontFamily: font.black, fontSize: 88, lineHeight: weightLine },
  currentUnit: { fontFamily: font.heavy, fontSize: 24 },
  currentReps: { fontFamily: font.black, fontSize: 56 },
  currentTimes: { fontFamily: font.semi, fontSize: 32 },
  weightEntry: { flexDirection: 'row', alignItems: 'baseline', flexShrink: 1 },
  repsEntry: { flexDirection: 'row', alignItems: 'baseline' },
  entrySizer: { opacity: 0, minWidth: 28, paddingRight: 2 },
  entryInput: { ...StyleSheet.absoluteFill, padding: 0, lineHeight: undefined },
  entryLine: { position: 'absolute', left: 0, right: 0, height: StyleSheet.hairlineWidth * 2 },
  loggedRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: -10,
    marginBottom: 2,
  },
  loggedText: { marginTop: 0, flexShrink: 1 },
  // Above Finish there's no rest bar or gap to pull up into, so the row
  // sits as far under the last set as it does under the rest gap.
  lastLogged: { marginTop: 2 },
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
  typedLabel: { marginTop: 14 },
  typedRow: { flexDirection: 'row', gap: 8, marginTop: 6 },
  typedField: { flex: 1 },
  typedLog: { width: 96, height: 52 },
  discard: { alignItems: 'center', paddingVertical: 18, marginTop: 12 },
  discardText: { fontSize: 16, fontWeight: '600' },
  finish: { marginTop: 20 },
  afterLogged: { marginTop: 10 },
  endError: { marginTop: 12, paddingBottom: 0 },
  afterError: { marginTop: 12 },
});
