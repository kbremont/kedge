import { Host, Picker, Text as SwiftText } from '@expo/ui/swift-ui';
import { pickerStyle, tag } from '@expo/ui/swift-ui/modifiers';
import { router } from 'expo-router';
import { useRef, useState } from 'react';
import { Keyboard, StyleSheet, Text, useColorScheme, View } from 'react-native';

import { ApiError } from '@/api/client';
import { useSaveSetup, useSetup } from '@/api/queries';
import type { Setup, SetupInput } from '@/api/schema.gen';
import {
  Button,
  Field,
  Intro,
  LiftChip,
  LoadState,
  Notice,
  Page,
  TableHead,
  useCardStyles,
} from '@/components/card-back';
import { parseDecimal, weightField } from '@/lib/format';

// First run, step one: a training max for each lift. Everything the app
// prescribes follows from these.
export default function SetupScreen() {
  return <LoadState query={useSetup()}>{(setup) => <SetupTMs setup={setup} />}</LoadState>;
}

type Unit = NonNullable<SetupInput['unit']>;

function SetupTMs({ setup }: { setup: Setup }) {
  const { p, s } = useCardStyles();
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';
  const save = useSaveSetup();
  // Offered until anything is logged, since weights are never converted. A
  // server from before kilograms doesn't say, and is in pounds.
  const canChangeUnit = setup.canChangeUnit === true;
  const [unit, setUnit] = useState<Unit>(setup.unit === 'kg' ? 'kg' : 'lb');
  // Running setup again starts from the training maxes already saved.
  const [values, setValues] = useState(() =>
    Object.fromEntries(setup.lifts.map((l) => [l.slug, l.value > 0 ? weightField(l.value) : ''])),
  );
  // A second tap can land before the button redraws disabled, and mustn't
  // save twice.
  const sending = useRef(false);
  // The server refuses a value that isn't positive, naming the first such
  // lift; every field like it is outlined.
  const refused = save.error instanceof ApiError && save.error.status === 400;

  // The server names the first lift it refuses. A field that isn't a number
  // goes as 0, so it is refused as an empty one is, not as a bad body.
  const next = () => {
    if (sending.current) return;
    sending.current = true;
    // The keyboard would otherwise leave over the next screen.
    Keyboard.dismiss();
    const tms = Object.fromEntries(setup.lifts.map((l) => [l.slug, parseDecimal(values[l.slug]) || 0]));
    save.mutate(
      canChangeUnit ? { tms, unit } : { tms },
      {
        onSuccess: () => router.push('/setup/position'),
        onSettled: () => {
          sending.current = false;
        },
      },
    );
  };

  return (
    <Page>
      <Intro>
        Start at 90% of a recent, honest one rep max. If in doubt, go lower. You can change these at any time.
      </Intro>
      {canChangeUnit ? (
        <View style={local.unit}>
          <Host matchContents={{ vertical: true }} seedColor={p.ink} colorScheme={scheme}>
            <Picker
              selection={unit}
              onSelectionChange={(u: Unit) => {
                if (u === unit) return;
                setUnit(u);
                // A number typed in one unit means nothing in the other.
                setValues(Object.fromEntries(setup.lifts.map((l) => [l.slug, ''])));
                if (save.isError) save.reset();
              }}
              modifiers={[pickerStyle('segmented')]}>
              {(['lb', 'kg'] as const).map((u) => (
                <SwiftText key={u} modifiers={[tag(u)]}>
                  {unitLabel[u]}
                </SwiftText>
              ))}
            </Picker>
          </Host>
          <Text style={[s.sub, local.unitNote]}>Kept once you log a session.</Text>
        </View>
      ) : null}
      <TableHead
        style={local.head}
        columns={[
          { label: 'Lift', style: { flex: 1 } },
          { label: `Training max, ${unit}`, style: { textAlign: 'right' } },
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
              // A refusal stands until the lifter changes what they typed.
              if (save.isError) save.reset();
            }}
            keyboardType="decimal-pad"
            accessibilityLabel={`${l.name} training max, ${unit}`}
            invalid={refused && !(parseDecimal(values[l.slug]) > 0)}
            style={local.field}
          />
        </View>
      ))}
      {save.error ? (
        <Notice error style={local.error}>
          {save.error.message}
        </Notice>
      ) : null}
      <Button label="Continue" disabled={save.isPending} style={local.next} onPress={next} />
    </Page>
  );
}

const unitLabel: Record<Unit, string> = { lb: 'Pounds', kg: 'Kilograms' };

const local = StyleSheet.create({
  unit: { marginTop: 18 },
  unitNote: { marginTop: 6 },
  head: { marginTop: 18 },
  row: { paddingVertical: 8 },
  nameRow: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: 8 },
  name: { fontSize: 17 },
  field: { width: 120, height: 54, fontSize: 30, textAlign: 'right' },
  error: { marginTop: 8, paddingBottom: 0 },
  next: { marginTop: 22 },
});
