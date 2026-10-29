import { DatePicker, Host } from '@expo/ui/swift-ui';
import { router, useLocalSearchParams } from 'expo-router';
import { useRef, useState } from 'react';
import { StyleSheet, Text, useColorScheme, View } from 'react-native';

import { useAddEstimate, useLift } from '@/api/queries';
import type { Lift } from '@/api/schema.gen';
import { Button, Field, Label, LoadState, Notice, Page, TextAction, useCardStyles } from '@/components/card-back';
import { font, liftInk } from '@/theme';

// The day the lifter picked, as the API writes days (YYYY-MM-DD). Read in the
// phone's own time zone: toISOString would give the UTC day, a day off for an
// evening west of Greenwich or a morning east of it.
function localDay(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// A past estimate: a date, a weight, and reps, which default to 1 for an
// estimated max. The server converts it as it does a logged set. The sheet
// opens over the lift's page, so the lift is already cached.
export default function AddEstimateSheet() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  const lift = useLift(slug);
  return <LoadState query={lift}>{(l) => <Sheet slug={slug} l={l} />}</LoadState>;
}

function Sheet({ slug, l }: { slug: string; l: Lift }) {
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  // The server's day, so the picker can't reach a day it would refuse.
  const today = new Date(`${l.today}T12:00:00`);
  const [date, setDate] = useState(today);
  const [weight, setWeight] = useState('');
  const [reps, setReps] = useState('1');
  const add = useAddEstimate(slug);
  // The server's message for an estimate it refused, shown as it is; the
  // sheet stays open so it can be fixed.
  const error = add.error?.message ?? '';
  // A second tap can land before the button redraws disabled, and mustn't
  // add the estimate twice.
  const sending = useRef(false);

  const save = () => {
    if (sending.current) return;
    sending.current = true;
    add.mutate(
      { date: localDay(date), weight: Number(weight), reps: Number(reps) },
      {
        onSuccess: () => router.back(),
        onSettled: () => {
          sending.current = false;
        },
      },
    );
  };

  return (
    // A Page, so all of it can be scrolled to above the keyboard: a number
    // pad has no return key, and at large text sizes, or under a Notice, Add
    // estimate and Cancel fall below the pad's top.
    <Page contentContainerStyle={local.sheet}>
      <View style={[local.bar, { backgroundColor: liftInk(p, l.slug).band }]} />
      <Text style={[local.title, { color: p.ink }]} accessibilityRole="header">
        Past estimate
      </Text>
      <Text style={s.sub}>{l.name}. It goes on the chart and counts toward the best estimated max.</Text>

      <View style={[local.dateRow, { borderBottomColor: p.rule }]}>
        <Label>Date</Label>
        <Host matchContents seedColor={p.ink} colorScheme={scheme}>
          <DatePicker selection={date} range={{ end: today }} onDateChange={setDate} />
        </Host>
      </View>

      <View style={local.pair}>
        <View style={{ flex: 1 }}>
          <Label>Weight, {l.unit}</Label>
          <Field
            style={local.input}
            value={weight}
            onChangeText={setWeight}
            keyboardType="decimal-pad"
            accessibilityLabel={`Weight in ${l.unit}`}
            invalid={error !== '' && !(Number(weight) > 0)}
          />
        </View>
        <View style={{ width: 110 }}>
          <Label>Reps</Label>
          <Field
            style={local.input}
            value={reps}
            onChangeText={setReps}
            keyboardType="number-pad"
            accessibilityLabel="Reps"
            invalid={error !== '' && !(Number(reps) >= 1)}
          />
        </View>
      </View>

      {error ? (
        <Notice error style={local.error}>
          {error}
        </Notice>
      ) : null}
      <Button
        label="Add estimate"
        disabled={add.isPending}
        style={error ? local.saveAfterError : local.save}
        onPress={save}
      />
      <TextAction muted label="Cancel" onPress={() => router.back()} style={local.cancel} />
    </Page>
  );
}

const local = StyleSheet.create({
  sheet: { paddingHorizontal: 20, paddingTop: 0, paddingBottom: 20 },
  bar: { height: 6, marginHorizontal: -20, marginBottom: 18 },
  title: { fontFamily: font.black, fontSize: 30, textTransform: 'uppercase', letterSpacing: 0.4 },
  dateRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 18,
    paddingBottom: 12,
    borderBottomWidth: StyleSheet.hairlineWidth * 2,
  },
  pair: { flexDirection: 'row', gap: 12, marginTop: 16 },
  input: { marginTop: 4 },
  error: { marginTop: 12, paddingBottom: 0 },
  save: { marginTop: 22 },
  saveAfterError: { marginTop: 12 },
  cancel: { marginTop: 4 },
});
