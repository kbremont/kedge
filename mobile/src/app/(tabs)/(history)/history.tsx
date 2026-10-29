import { router } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { Fragment } from 'react';
import { Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';

import { useHistory } from '@/api/queries';
import type { History, HistoryItem, LiftRef } from '@/api/schema.gen';
import { Label, LiftChip, LoadState, Page, useCardStyles } from '@/components/card-back';
import { cycleParts, cycleText, shortName, times } from '@/lib/format';
import { displayScaleCap, font, liftInk } from '@/theme';

// One line per round, newest first: the lifter's career read across like
// seasons on the back of a card, each lift in its own column.
type Line = { label: string; items: Record<string, HistoryItem> };

function careerLines(items: HistoryItem[]): Line[] {
  const lines: Line[] = [];
  for (const it of items) {
    const last = lines.at(-1);
    if (last && last.label === it.cycleLabel && !last.items[it.liftSlug]) {
      last.items[it.liftSlug] = it;
    } else {
      lines.push({ label: it.cycleLabel, items: { [it.liftSlug]: it } });
    }
  }
  return lines;
}

// "7th Week TM Test" reads as "TM Test" in a narrow column.
function cycleShort(cycle: string): string {
  return cycle.replace(/^7th Week /, '');
}

const openSession = (it: HistoryItem) =>
  router.push({ pathname: '/workout/[id]', params: { id: it.workoutId, back: 'History' } });

export default function HistoryScreen() {
  const history = useHistory();
  return <LoadState query={history}>{(h) => <Career h={h} />}</LoadState>;
}

function Career({ h }: { h: History }) {
  const { p, s } = useCardStyles();
  const { fontScale } = useWindowDimensions();
  const lines = careerLines(h.items);

  // At accessibility text sizes four columns no longer fit: each round becomes
  // a short list of its sessions instead.
  if (fontScale > 1.5) {
    return (
      <Page>
        <LiftLinks lifts={h.lifts} />
        {lines.length === 0 ? <Empty /> : null}
        {lines.map((line, i) => (
          <View key={i} style={[local.stackedLine, { borderBottomColor: p.rule }]}>
            <Text style={[s.name, local.stackedTitle]}>{cycleText(line.label)}</Text>
            {h.lifts.map((l) => {
              const it = line.items[l.slug];
              return it ? (
                <Pressable
                  key={l.slug}
                  accessibilityRole="button"
                  onPress={() => openSession(it)}
                  style={({ pressed }) => [local.stackedRow, pressed && s.pressed]}>
                  <LiftChip slug={l.slug} />
                  <Text style={[s.name, { flex: 1 }]}>{l.name}</Text>
                  <View style={{ alignItems: 'flex-end' }}>
                    <Text style={s.num}>{times(it.summary)}</Text>
                    <Text style={s.sub}>{it.date}</Text>
                  </View>
                </Pressable>
              ) : null;
            })}
          </View>
        ))}
      </Page>
    );
  }

  return (
    <Page stickyHeaderIndices={[0]}>
      <LiftHeads lifts={h.lifts} />
      {lines.length === 0 ? <Empty /> : null}
      {lines.map((line, i) => {
        const { cycle, round } = cycleParts(line.label);
        return (
          <View key={i} style={[local.line, { borderBottomColor: p.rule }]}>
            <View style={local.cRound} accessible accessibilityLabel={cycleText(line.label)}>
              <Label style={local.cycle}>{cycleShort(cycle)}</Label>
              {round ? (
                <Text style={[s.num, local.round]} maxFontSizeMultiplier={displayScaleCap}>
                  {round}
                </Text>
              ) : null}
            </View>
            {h.lifts.map((l) => {
              const it = line.items[l.slug];
              return (
                <Fragment key={l.slug}>
                  <View style={[local.colRule, { backgroundColor: p.rule }]} />
                  {it ? (
                    <Pressable
                      accessibilityRole="button"
                      accessibilityLabel={`${l.name}, ${cycleText(line.label)}, ${it.date}, ${times(it.summary)}`}
                      onPress={() => openSession(it)}
                      style={({ pressed }) => [local.cell, pressed && { backgroundColor: p.stock }]}>
                      <Text
                        style={[local.cellNum, { color: p.ink }]}
                        numberOfLines={1}
                        adjustsFontSizeToFit
                        minimumFontScale={0.8}
                        maxFontSizeMultiplier={displayScaleCap}>
                        {times(it.summary)}
                      </Text>
                      <Text style={[local.cellDate, { color: p.ink2 }]} maxFontSizeMultiplier={displayScaleCap}>
                        {it.date}
                      </Text>
                    </Pressable>
                  ) : (
                    <View style={local.cell} />
                  )}
                </Fragment>
              );
            })}
          </View>
        );
      })}
    </Page>
  );
}

// The column heads: each lift's colour, its name, and the way to its page.
// They stay pinned under the header while the career scrolls.
function LiftHeads({ lifts }: { lifts: LiftRef[] }) {
  const { p, s } = useCardStyles();
  return (
    <View style={[local.heads, { backgroundColor: p.ground, borderBottomColor: p.ink }]}>
      <View style={[local.cRound, local.headRound]}>
        <Label>Round</Label>
      </View>
      {lifts.map((l) => (
        <Fragment key={l.slug}>
          {/* Keeps the heads over the columns' hairline rules. */}
          <View style={local.colRule} />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`${l.name}: chart and past estimates`}
            onPress={() => router.push({ pathname: '/lift/[slug]', params: { slug: l.slug, back: 'History' } })}
            style={({ pressed }) => [local.head, pressed && s.pressed]}>
            <View style={[local.headBar, { backgroundColor: liftInk(p, l.slug).band }]} />
            <View style={local.headLabel}>
              <Label>{shortName(l.slug, l.name)}</Label>
              <SymbolView name="chevron.right" tintColor={p.ink2} size={10} weight="bold" />
            </View>
          </Pressable>
        </Fragment>
      ))}
    </View>
  );
}

// At accessibility text sizes the heads give way to two rows of lift links,
// each with room for its whole name.
function LiftLinks({ lifts }: { lifts: LiftRef[] }) {
  const { p, s } = useCardStyles();
  return (
    <View style={[local.links, { borderBottomColor: p.ink }]}>
      {lifts.map((l) => (
        <Pressable
          key={l.slug}
          accessibilityRole="button"
          accessibilityLabel={`${l.name}: chart and past estimates`}
          onPress={() => router.push({ pathname: '/lift/[slug]', params: { slug: l.slug, back: 'History' } })}
          style={({ pressed }) => [local.link, pressed && s.pressed]}>
          <View style={[local.headBar, { backgroundColor: liftInk(p, l.slug).band }]} />
          <View style={local.headLabel}>
            <Text style={s.name}>{shortName(l.slug, l.name)}</Text>
            <SymbolView name="chevron.right" tintColor={p.ink2} size={12} weight="bold" />
          </View>
        </Pressable>
      ))}
    </View>
  );
}

function Empty() {
  const { s } = useCardStyles();
  return (
    <View style={s.row}>
      <Text style={[s.name, s.muted, { flex: 1 }]}>No sessions yet. Each round you finish adds a line here.</Text>
    </View>
  );
}

const hairline = StyleSheet.hairlineWidth * 2;

const local = StyleSheet.create({
  heads: { flexDirection: 'row', alignItems: 'flex-end', borderBottomWidth: 3, paddingTop: 6 },
  headRound: { paddingBottom: 6 },
  head: { flex: 1, paddingLeft: 8, paddingRight: 6, paddingBottom: 6 },
  headBar: { height: 5, borderRadius: 1, marginBottom: 7 },
  headLabel: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  line: { flexDirection: 'row', alignItems: 'stretch', borderBottomWidth: hairline },
  cRound: { width: 62, justifyContent: 'center', paddingVertical: 8 },
  cycle: { fontSize: 13, letterSpacing: 0.6 },
  round: { fontSize: 22, lineHeight: 24 },
  colRule: { width: hairline },
  cell: { flex: 1, alignItems: 'flex-end', justifyContent: 'center', paddingVertical: 9, paddingHorizontal: 7 },
  cellNum: { fontFamily: font.heavy, fontSize: 20, fontVariant: ['tabular-nums'] },
  // A date reads as one word under its top set, so its digits keep their own
  // widths.
  cellDate: { fontFamily: font.semi, fontSize: 14 },
  links: { flexDirection: 'row', flexWrap: 'wrap', borderBottomWidth: 3, columnGap: 16, paddingBottom: 4 },
  link: { width: '46%', flexGrow: 1, paddingVertical: 8 },
  stackedLine: { paddingVertical: 10, borderBottomWidth: hairline },
  stackedTitle: { marginBottom: 4 },
  stackedRow: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingVertical: 6 },
});
