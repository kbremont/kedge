import { useQueryClient } from '@tanstack/react-query';
import { router } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { keys, useSetupPosition } from '@/api/queries';
import type { SetupPosition } from '@/api/schema.gen';
import { Button, Intro, LoadState, Page, useCardStyles } from '@/components/card-back';
import { timesIn } from '@/lib/format';

// First run, step two: which cycle the lifter is on now. Anything before it
// stays out of the block, so there is no old history to enter.
export default function SetupPositionScreen() {
  return <LoadState query={useSetupPosition()}>{(pos) => <Position pos={pos} />}</LoadState>;
}

function Position({ pos: { cycles } }: { pos: SetupPosition }) {
  const { p, s } = useCardStyles();
  const queryClient = useQueryClient();
  const [choice, setChoice] = useState<number | 'fresh'>('fresh');
  const options = [
    { key: 'fresh' as const, label: 'Starting fresh', detail: `At the start of ${cycles[0]?.label ?? 'the block'}` },
    ...cycles.map((c) => ({ key: c.index, label: c.label, detail: timesIn(c.detail) })),
  ];

  // Starting fresh sends nothing: the block already starts at its first
  // cycle. Today is asked again for its first session. Any other cycle
  // goes on to its checklist, which saves the choice with it.
  const next = () => {
    if (choice === 'fresh') {
      void queryClient.invalidateQueries({ queryKey: keys.dashboard() });
      router.dismissTo('/');
    } else {
      router.push({ pathname: '/setup/history', params: { cycle: choice } });
    }
  };

  return (
    <Page>
      <Intro style={local.intro}>
        Pick the cycle you are on now. Anything before it is left out of this block, so there is no old history
        to enter.
      </Intro>
      <View style={[local.list, { borderTopColor: p.ink }]} accessibilityRole="radiogroup">
        {options.map((o) => {
          const selected = choice === o.key;
          return (
            <Pressable
              key={o.key}
              accessibilityRole="radio"
              accessibilityState={{ checked: selected }}
              onPress={() => setChoice(o.key)}
              style={({ pressed }) => [s.row, local.row, pressed && s.pressed]}>
              <View style={{ flex: 1 }}>
                <Text style={[s.name, local.name]}>{o.label}</Text>
                <Text style={s.sub}>{o.detail}</Text>
              </View>
              {selected ? <SymbolView name="checkmark" tintColor={p.ink} size={20} weight="bold" /> : null}
            </Pressable>
          );
        })}
      </View>
      <Button label={choice === 'fresh' ? 'Start training' : 'Continue'} style={local.next} onPress={next} />
    </Page>
  );
}

const local = StyleSheet.create({
  intro: { marginBottom: 16 },
  list: { borderTopWidth: 3 },
  row: { minHeight: 64, gap: 12 },
  name: { fontSize: 17 },
  next: { marginTop: 22 },
});
