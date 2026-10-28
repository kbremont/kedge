// The Career Line building blocks: every screen is stock, ink, ruled stat
// lines and a band in the lift's colour.
import { SymbolView, type SFSymbol } from 'expo-symbols';
import { useMemo, type ComponentRef, type ReactNode, type Ref } from 'react';
import {
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
  type StyleProp,
  type TextStyle,
  type ViewStyle,
} from 'react-native';

import { displayScaleCap, font, liftInk, radius, space, usePalette, type Palette } from '@/theme';

const hairline = StyleSheet.hairlineWidth * 2;

function makeStyles(p: Palette) {
  return StyleSheet.create({
    page: { backgroundColor: p.ground },
    pageContent: { paddingHorizontal: space.page, paddingTop: 8, paddingBottom: 48 },
    panel: { backgroundColor: p.stock, borderRadius: radius.panel, overflow: 'hidden' },
    band: { paddingHorizontal: 16, paddingTop: 14, paddingBottom: 12 },
    bandTitle: {
      fontFamily: font.black,
      fontSize: 46,
      lineHeight: 46,
      letterSpacing: 0.4,
      textTransform: 'uppercase',
    },
    bandSub: { fontSize: 15, fontWeight: '600', marginTop: 4, opacity: 0.9 },
    slimBand: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
      paddingHorizontal: 14,
      paddingVertical: 9,
    },
    slimBandText: {
      fontFamily: font.heavy,
      fontSize: 20,
      letterSpacing: 0.6,
      textTransform: 'uppercase',
    },
    statRow: { flexDirection: 'row', borderBottomWidth: hairline, borderBottomColor: p.rule },
    stat: { flex: 1, paddingHorizontal: 14, paddingVertical: 10 },
    statRule: { borderRightWidth: hairline, borderRightColor: p.rule },
    label: {
      fontFamily: font.semi,
      fontSize: 14,
      letterSpacing: 0.8,
      textTransform: 'uppercase',
      color: p.ink2,
    },
    statValue: { fontFamily: font.heavy, fontSize: 32, color: p.ink, fontVariant: ['tabular-nums'] },
    statUnit: { fontFamily: font.semi, fontSize: 17, color: p.ink2 },
    button: {
      height: 54,
      borderRadius: radius.panel,
      backgroundColor: p.button,
      alignItems: 'center',
      justifyContent: 'center',
      flexDirection: 'row',
      gap: 8,
    },
    pressed: { opacity: 0.8 },
    buttonText: {
      fontFamily: font.heavy,
      fontSize: 22,
      letterSpacing: 0.6,
      textTransform: 'uppercase',
      color: p.onButton,
    },
    headingRow: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'space-between',
      marginTop: space.section,
      paddingBottom: 4,
      borderBottomWidth: 3,
      borderBottomColor: p.ink,
    },
    heading: {
      fontFamily: font.black,
      fontSize: 26,
      letterSpacing: 0.4,
      textTransform: 'uppercase',
      color: p.ink,
    },
    tableHead: {
      flexDirection: 'row',
      alignItems: 'flex-end',
      paddingTop: 10,
      paddingBottom: 4,
      borderBottomWidth: hairline,
      borderBottomColor: p.rule,
    },
    row: {
      flexDirection: 'row',
      alignItems: 'center',
      paddingVertical: 10,
      borderBottomWidth: hairline,
      borderBottomColor: p.rule,
    },
    num: { fontFamily: font.heavy, fontSize: 24, color: p.ink, fontVariant: ['tabular-nums'] },
    numSmall: { fontFamily: font.semi, fontSize: 18, color: p.ink, fontVariant: ['tabular-nums'] },
    name: { fontSize: 16, fontWeight: '600', color: p.ink },
    sub: { fontSize: 13, color: p.ink2, marginTop: 2 },
    muted: { color: p.ink2 },
    chip: { width: 10, height: 10, borderRadius: radius.chip },
    rest: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingTop: 14, paddingBottom: 10 },
    restLabel: {
      fontFamily: font.heavy,
      fontSize: 18,
      letterSpacing: 0.8,
      textTransform: 'uppercase',
      color: p.ink,
    },
    restTrack: { flex: 1, height: 6, backgroundColor: p.rule, borderRadius: 1 },
    restFill: { height: 6, borderRadius: 1 },
    restClock: { fontFamily: font.black, fontSize: 30, color: p.ink, fontVariant: ['tabular-nums'] },
    totals: {
      flexDirection: 'row',
      alignItems: 'center',
      paddingVertical: 10,
      borderTopWidth: 3,
      borderTopColor: p.ink,
      marginTop: -1,
    },
    totalsLabel: {
      flex: 1,
      fontFamily: font.heavy,
      fontSize: 18,
      letterSpacing: 0.6,
      textTransform: 'uppercase',
      color: p.ink,
    },
    totalsValue: { fontFamily: font.black, fontSize: 30, color: p.ink, fontVariant: ['tabular-nums'] },
  });
}

export function useCardStyles() {
  const p = usePalette();
  return useMemo(() => ({ p, s: makeStyles(p) }), [p]);
}

export function Page({
  children,
  ref,
}: {
  children: ReactNode;
  ref?: Ref<ComponentRef<typeof ScrollView>>;
}) {
  const { s } = useCardStyles();
  return (
    <ScrollView
      ref={ref}
      style={s.page}
      contentInsetAdjustmentBehavior="automatic"
      contentContainerStyle={s.pageContent}>
      {children}
    </ScrollView>
  );
}

export function Panel({ children, style }: { children: ReactNode; style?: StyleProp<ViewStyle> }) {
  const { s } = useCardStyles();
  return <View style={[s.panel, style]}>{children}</View>;
}

// The lift's colour band: the card's header, in large or slim form.
export function LiftBand({
  slug,
  title,
  subtitle,
  right,
  slim,
  style,
}: {
  slug: string;
  title: string;
  subtitle?: string;
  right?: string;
  slim?: boolean;
  style?: StyleProp<ViewStyle>;
}) {
  const { p, s } = useCardStyles();
  const ink = liftInk(p, slug);
  if (slim) {
    return (
      <View style={[s.slimBand, { backgroundColor: ink.band }, style]}>
        <Text style={[s.slimBandText, { color: ink.onBand }]} maxFontSizeMultiplier={displayScaleCap}>
          {title}
        </Text>
        {right ? (
          <Text style={[s.slimBandText, { color: ink.onBand }]} maxFontSizeMultiplier={displayScaleCap}>
            {right}
          </Text>
        ) : null}
      </View>
    );
  }
  return (
    <View style={[s.band, { backgroundColor: ink.band }, style]}>
      <Text
        style={[s.bandTitle, { color: ink.onBand }]}
        maxFontSizeMultiplier={displayScaleCap}
        accessibilityRole="header">
        {title}
      </Text>
      {subtitle ? <Text style={[s.bandSub, { color: ink.onBand }]}>{subtitle}</Text> : null}
    </View>
  );
}

export function StatRow({ children }: { children: ReactNode }) {
  const { s } = useCardStyles();
  return <View style={s.statRow}>{children}</View>;
}

export function Stat({
  label,
  value,
  unit,
  last,
}: {
  label: string;
  value: string;
  unit?: string;
  last?: boolean;
}) {
  const { s } = useCardStyles();
  return (
    <View style={[s.stat, !last && s.statRule]}>
      <Text style={s.label} maxFontSizeMultiplier={displayScaleCap}>
        {label}
      </Text>
      <Text style={s.statValue} maxFontSizeMultiplier={displayScaleCap}>
        {value}
        {unit ? <Text style={s.statUnit}> {unit}</Text> : null}
      </Text>
    </View>
  );
}

export function Label({ children, style }: { children: ReactNode; style?: StyleProp<TextStyle> }) {
  const { s } = useCardStyles();
  return (
    <Text style={[s.label, style]} maxFontSizeMultiplier={displayScaleCap}>
      {children}
    </Text>
  );
}

export function Button({
  label,
  icon,
  onPress,
  style,
}: {
  label: string;
  icon?: SFSymbol;
  onPress?: () => void;
  style?: StyleProp<ViewStyle>;
}) {
  const { p, s } = useCardStyles();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [s.button, pressed && s.pressed, style]}>
      {icon ? <SymbolView name={icon} tintColor={p.onButton} size={18} weight="bold" /> : null}
      <Text style={s.buttonText} maxFontSizeMultiplier={displayScaleCap}>
        {label}
      </Text>
    </Pressable>
  );
}

// A section heading: black condensed caps over a heavy ink rule.
export function Heading({ title, action }: { title: string; action?: ReactNode }) {
  const { s } = useCardStyles();
  return (
    <View style={s.headingRow}>
      <Text style={s.heading} accessibilityRole="header" maxFontSizeMultiplier={displayScaleCap}>
        {title}
      </Text>
      {action}
    </View>
  );
}

export type Column = { label: string; style: StyleProp<TextStyle> };

export function TableHead({ columns, style }: { columns: Column[]; style?: StyleProp<ViewStyle> }) {
  const { s } = useCardStyles();
  return (
    <View style={[s.tableHead, style]}>
      {columns.map((c) => (
        <Text key={c.label} style={[s.label, c.style]} maxFontSizeMultiplier={displayScaleCap}>
          {c.label}
        </Text>
      ))}
    </View>
  );
}

export function LiftChip({ slug }: { slug: string }) {
  const { p, s } = useCardStyles();
  return <View style={[s.chip, { backgroundColor: liftInk(p, slug).band }]} />;
}

// The rest bar is drawn to scale: its full length is the interval the server
// sent, and it drains in real seconds.
export function RestBar({ slug, left, total, clock }: { slug: string; left: number; total: number; clock: string }) {
  const { p, s } = useCardStyles();
  return (
    <View style={s.rest} accessibilityLabel={`Rest, ${clock} left`}>
      <Text style={s.restLabel}>Rest</Text>
      <View style={s.restTrack}>
        <View
          style={[s.restFill, { backgroundColor: liftInk(p, slug).line, width: `${(left / total) * 100}%` }]}
        />
      </View>
      <Text style={s.restClock} maxFontSizeMultiplier={displayScaleCap}>
        {clock}
      </Text>
    </View>
  );
}

// The card back's bottom line, like a career's totals under the seasons.
export function Totals({ label, value }: { label: string; value: string }) {
  const { s } = useCardStyles();
  return (
    <View style={s.totals}>
      <Text style={s.totalsLabel}>{label}</Text>
      <Text style={s.totalsValue} maxFontSizeMultiplier={displayScaleCap}>
        {value}
      </Text>
    </View>
  );
}
