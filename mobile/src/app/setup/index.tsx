import { router } from 'expo-router';
import { useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';

import { Button, Field, Intro, LiftChip, Notice, Page, TableHead, useCardStyles } from '@/components/card-back';
import { useOverrides } from '@/gallery/store';
import { fmtWeight } from '@/lib/format';
import { sample } from '@/sample';

// First run, step one: a training max for each lift. Everything the app
// prescribes follows from these.
export default function SetupScreen() {
  const { version, o } = useOverrides();
  return <SetupTMs key={version} error={o.setupError} />;
}

function SetupTMs({ error: errorAtStart }: { error?: string }) {
  const { s } = useCardStyles();
  const setup = sample.setup;
  const [values, setValues] = useState(() =>
    Object.fromEntries(
      setup.lifts.map((l) => [l.slug, l.value > 0 ? fmtWeight(l.value) : errorAtStart ? demoTMs[l.slug] : '']),
    ),
  );
  // The server's message for a value it refused, shown as it is.
  const [error, setError] = useState(errorAtStart ?? '');

  // Sample only: POST /setup, which answers a missing value with a 400 naming
  // the lift.
  const next = () => {
    const missing = setup.lifts.find((l) => !(Number(values[l.slug]) > 0));
    if (missing) setError(`Enter a positive training max for ${missing.name}.`);
    else router.push('/setup/position');
  };

  return (
    <Page>
      <Intro>
        Start at 90% of a recent, honest one rep max. If in doubt, go lower. You can change these at any time.
      </Intro>
      <TableHead
        style={local.head}
        columns={[
          { label: 'Lift', style: { flex: 1 } },
          { label: `Training max, ${setup.unit}`, style: { textAlign: 'right' } },
        ]}
      />
      {setup.lifts.map((l) => (
        <View key={l.slug} style={[s.row, local.row]}>
          <View style={local.nameRow}>
            <LiftChip slug={l.slug} />
            <Text style={[s.name, local.name]}>{l.name}</Text>
          </View>
          <Field
            value={values[l.slug]}
            onChangeText={(v) => {
              setValues((x) => ({ ...x, [l.slug]: v }));
              setError('');
            }}
            keyboardType="decimal-pad"
            accessibilityLabel={`${l.name} training max, ${setup.unit}`}
            invalid={error !== '' && !(Number(values[l.slug]) > 0)}
            style={local.field}
          />
        </View>
      ))}
      {error ? (
        <Notice error style={local.error}>
          {error}
        </Notice>
      ) : null}
      <Button label="Continue" style={local.next} onPress={next} />
    </Page>
  );
}

// Gallery only: the error state with three lifts filled in and squat left
// empty.
const demoTMs: Record<string, string> = { press: '115', deadlift: '365', bench: '205', squat: '' };

const local = StyleSheet.create({
  head: { marginTop: 18 },
  row: { paddingVertical: 8 },
  nameRow: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: 8 },
  name: { fontSize: 17 },
  field: { width: 120, height: 54, fontSize: 30, textAlign: 'right' },
  error: { marginTop: 8, paddingBottom: 0 },
  next: { marginTop: 22 },
});
