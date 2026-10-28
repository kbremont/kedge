import { DatePicker, Host } from '@expo/ui/swift-ui';
import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, Text, TextInput, useColorScheme, View } from 'react-native';

import { Button, Label, useCardStyles } from '@/components/card-back';
import { sample } from '@/sample';
import { font, liftInk, radius } from '@/theme';

// A past estimate: a date, a weight, and reps, which default to 1 for an
// estimated max. The server converts it as it does a logged set.
export default function AddEstimateSheet() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const l = sample.lifts[slug] ?? sample.lifts.bench;
  const today = new Date(`${l.today}T12:00:00`);
  const [date, setDate] = useState(today);
  const [weight, setWeight] = useState('');
  const [reps, setReps] = useState('1');

  const field = [local.input, { backgroundColor: p.stock, color: p.ink, borderColor: p.rule }];
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
          <TextInput
            style={field}
            value={weight}
            onChangeText={setWeight}
            keyboardType="decimal-pad"
            accessibilityLabel={`Weight in ${l.unit}`}
          />
        </View>
        <View style={{ width: 110 }}>
          <Label>Reps</Label>
          <TextInput
            style={field}
            value={reps}
            onChangeText={setReps}
            keyboardType="number-pad"
            accessibilityLabel="Reps"
          />
        </View>
      </View>

      {/* Sample only: POST /lifts/{slug}/estimates once the API is wired. */}
      <Button label="Add estimate" style={local.save} onPress={() => router.back()} />
      <Pressable accessibilityRole="button" style={local.cancel} onPress={() => router.back()}>
        <Text style={[s.name, s.muted]}>Cancel</Text>
      </Pressable>
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
  input: {
    height: 52,
    marginTop: 4,
    borderRadius: radius.panel,
    borderWidth: StyleSheet.hairlineWidth * 2,
    paddingHorizontal: 12,
    fontFamily: font.heavy,
    fontSize: 26,
    fontVariant: ['tabular-nums'],
  },
  save: { marginTop: 22 },
  cancel: { alignItems: 'center', paddingVertical: 14 },
});
