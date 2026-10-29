import { DatePicker, Host } from '@expo/ui/swift-ui';
import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { StyleSheet, Text, useColorScheme, View } from 'react-native';

import inputError from '@/api/fixtures/error-input.json';
import { Button, Field, Label, Notice, TextAction, useCardStyles } from '@/components/card-back';
import { useOverrides } from '@/gallery/store';
import { useLift } from '@/sample/use-screen';
import { font, liftInk } from '@/theme';

// A past estimate: a date, a weight, and reps, which default to 1 for an
// estimated max. The server converts it as it does a logged set.
export default function AddEstimateSheet() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const { o } = useOverrides();
  const l = useLift(slug);
  const today = new Date(`${l.today}T12:00:00`);
  const [date, setDate] = useState(today);
  const [weight, setWeight] = useState(o.estimateError ? '0' : '');
  const [reps, setReps] = useState('1');
  // The server's message for an estimate it refused, shown as it is; the
  // sheet stays open so it can be fixed.
  const [error, setError] = useState(o.estimateError ?? '');

  // Sample only: POST /lifts/{slug}/estimates, which answers a bad entry with
  // a 400 and this message.
  const add = () => {
    if (Number(weight) > 0 && Number(reps) >= 1) router.back();
    else setError(inputError.error);
  };

  return (
    <View style={[local.sheet, { backgroundColor: p.ground }]}>
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
      <Button label="Add estimate" style={error ? local.saveAfterError : local.save} onPress={add} />
      <TextAction muted label="Cancel" onPress={() => router.back()} style={local.cancel} />
    </View>
  );
}

const local = StyleSheet.create({
  sheet: { flex: 1, paddingHorizontal: 20, paddingTop: 0 },
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
