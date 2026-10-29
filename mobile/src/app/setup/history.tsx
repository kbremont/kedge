import { router, Stack, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { Fragment, useRef, useState } from 'react';
import { Keyboard, Pressable, StyleSheet, Text, View } from 'react-native';

import { useSaveSetupHistory, useSetupHistory } from '@/api/queries';
import type { PastSession, SetupHistory } from '@/api/schema.gen';
import {
  Button,
  Field,
  Heading,
  Intro,
  Label,
  LiftChip,
  LoadState,
  Notice,
  Page,
  useCardStyles,
} from '@/components/card-back';
import { fmtWeight } from '@/lib/format';

// First run, step three: the sessions of the chosen cycle already done. The
// first one left unticked is where the app picks up.
export default function SetupHistoryScreen() {
  // The cache keys a checklist by the API's cycle number, not the route's
  // string.
  const cycle = Number(useLocalSearchParams<{ cycle: string }>().cycle);
  const history = useSetupHistory(cycle);
  return (
    <>
      <Stack.Screen options={{ title: history.data?.cycleLabel ?? '' }} />
      <LoadState query={history}>{(h) => <Checklist h={h} />}</LoadState>
    </>
  );
}

function Checklist({ h }: { h: SetupHistory }) {
  const { p, s } = useCardStyles();
  const save = useSaveSetupHistory(h.cycleIndex);
  const busy = save.isPending;
  // Done sessions by "round-slug", with the reps of their AMRAP set when the
  // cycle asks for them.
  const [done, setDone] = useState<Record<string, string>>({});
  // A refusal stands until the lifter changes what they entered.
  const edit = (change: (was: Record<string, string>) => Record<string, string>) => {
    setDone(change);
    if (save.isError) save.reset();
  };
  const toggle = (key: string) =>
    edit(({ [key]: was, ...rest }) => (was === undefined ? { ...rest, [key]: '' } : rest));
  // A second tap can land before the button redraws disabled, and mustn't
  // save twice.
  const sending = useRef(false);

  // The ticked sessions in the checklist's order. Reps go only where the
  // cycle asks for them, and only as a count: a field left blank records
  // the set's target, as the placeholder says.
  const sessions = (): PastSession[] =>
    h.rounds.flatMap((rd) =>
      rd.days.flatMap((d) => {
        const reps = done[`${rd.round}-${d.slug}`];
        if (reps === undefined) return [];
        const n = Number(reps);
        return h.asksReps && Number.isInteger(n) && n > 0
          ? [{ round: rd.round, slug: d.slug, prReps: n }]
          : [{ round: rd.round, slug: d.slug }];
      }),
    );

  const saveAndStart = () => {
    if (sending.current) return;
    sending.current = true;
    // Today opens on success, and the keyboard mustn't still be leaving
    // over it.
    Keyboard.dismiss();
    save.mutate(
      { done: sessions() },
      {
        onSuccess: () => router.dismissTo('/'),
        onSettled: () => {
          sending.current = false;
        },
      },
    );
  };

  return (
    <Page>
      <Intro>
        Tick the sessions you have already finished. The first one you leave unticked is where the app will pick
        up.
      </Intro>
      {h.rounds.map((rd) => (
        <Fragment key={rd.round}>
          {h.rounds.length > 1 ? <Heading title={`Round ${rd.round}`} /> : <View style={[local.rule, { backgroundColor: p.ink }]} />}
          {rd.days.map((d) => {
            const key = `${rd.round}-${d.slug}`;
            const ticked = done[key] !== undefined;
            return (
              <View key={key} style={[s.row, local.row]}>
                <Pressable
                  accessibilityRole="checkbox"
                  accessibilityState={{ checked: ticked }}
                  accessibilityLabel={`${d.name}, round ${rd.round}, top set ${fmtWeight(d.topWeight)} ${h.unit}`}
                  disabled={busy}
                  onPress={() => toggle(key)}
                  style={({ pressed }) => [local.tick, pressed && s.pressed]}>
                  <SymbolView
                    name={ticked ? 'checkmark.circle.fill' : 'circle'}
                    tintColor={ticked ? p.ink : p.ink2}
                    size={26}
                  />
                  <View style={{ flex: 1 }}>
                    <View style={local.nameRow}>
                      <LiftChip slug={d.slug} />
                      <Text style={s.name}>{d.name}</Text>
                    </View>
                    <Text style={s.sub}>
                      Top set {fmtWeight(d.topWeight)} {h.unit}
                    </Text>
                  </View>
                </Pressable>
                {h.asksReps && ticked ? (
                  <View style={local.reps}>
                    <Label style={local.repsLabel}>{h.repsLabel}</Label>
                    <Field
                      value={done[key]}
                      onChangeText={(v) => edit((x) => ({ ...x, [key]: v }))}
                      editable={!busy}
                      placeholder={String(d.prTarget)}
                      keyboardType="number-pad"
                      accessibilityLabel={`${d.name}, round ${rd.round}, ${h.repsLabel}`}
                      style={local.repsField}
                    />
                  </View>
                ) : null}
              </View>
            );
          })}
        </Fragment>
      ))}
      {save.error ? (
        <Notice error style={local.error}>
          {save.error.message}
        </Notice>
      ) : null}
      <Button label="Save and start" disabled={busy} style={local.save} onPress={saveAndStart} />
    </Page>
  );
}

const local = StyleSheet.create({
  rule: { height: 3, marginTop: 18 },
  row: { minHeight: 64, gap: 12 },
  tick: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: 12, minHeight: 44 },
  nameRow: { flexDirection: 'row', alignItems: 'center', gap: 7 },
  reps: { alignItems: 'flex-end' },
  repsLabel: { fontSize: 12, marginBottom: 3 },
  repsField: { width: 76, height: 44, fontSize: 24, textAlign: 'right' },
  error: { marginTop: 8, paddingBottom: 0 },
  save: { marginTop: 24 },
});
