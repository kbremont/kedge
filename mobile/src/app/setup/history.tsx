import { router, Stack, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { Fragment, useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import type { SetupHistory } from '@/api/schema.gen';
import { Button, Field, Heading, Intro, Label, LiftChip, Page, useCardStyles } from '@/components/card-back';
import { useOverrides } from '@/gallery/store';
import { fmtWeight } from '@/lib/format';
import { useSetupHistory } from '@/sample/use-screen';

// First run, step three: the sessions of the chosen cycle already done. The
// first one left unticked is where the app picks up.
export default function SetupHistoryScreen() {
  const { cycle } = useLocalSearchParams<{ cycle: string }>();
  const { version, o } = useOverrides();
  const h = useSetupHistory(Number(cycle));
  return (
    <>
      <Stack.Screen options={{ title: h.cycleLabel }} />
      <Checklist key={version} h={h} tickedAtStart={o.ticked ?? {}} />
    </>
  );
}

function Checklist({ h, tickedAtStart }: { h: SetupHistory; tickedAtStart: Record<string, string> }) {
  const { p, s } = useCardStyles();
  // Done sessions by "round-slug", with the reps of their AMRAP set when the
  // cycle asks for them.
  const [done, setDone] = useState<Record<string, string>>(tickedAtStart);
  const toggle = (key: string) =>
    setDone(({ [key]: was, ...rest }) => (was === undefined ? { ...rest, [key]: '' } : rest));

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
                      onChangeText={(v) => setDone((x) => ({ ...x, [key]: v }))}
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
      {/* Sample only: POST /setup/history with the ticked sessions. */}
      <Button label="Save and start" style={local.save} onPress={() => router.dismissTo('/')} />
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
  save: { marginTop: 24 },
});
