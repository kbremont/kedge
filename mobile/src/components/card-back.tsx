// The Career Line building blocks: every screen is stock, ink, ruled stat
// lines and a band in the lift's colour.
import { SymbolView, type SFSymbol } from 'expo-symbols';
import {
  Children,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ComponentRef,
  type ReactNode,
  type Ref,
} from 'react';
import {
  AccessibilityInfo,
  ActivityIndicator,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  useWindowDimensions,
  View,
  type ScrollViewProps,
  type StyleProp,
  type TextInputProps,
  type TextStyle,
  type ViewStyle,
} from 'react-native';

import type { Tone } from '@/api/schema.gen';
import { useOverrides } from '@/gallery/store';
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
    buttonSecondary: { backgroundColor: 'transparent', borderWidth: 2, borderColor: p.ink },
    pressed: { opacity: 0.8 },
    disabled: { opacity: 0.35 },
    textAction: {
      minHeight: 44,
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'center',
      gap: 7,
      paddingVertical: 10,
    },
    textActionText: { fontSize: 16, fontWeight: '600', color: p.ink },
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
    field: {
      height: 52,
      borderRadius: radius.panel,
      borderWidth: hairline,
      borderColor: p.rule,
      paddingHorizontal: 12,
      color: p.ink,
    },
    fieldNumber: { fontFamily: font.heavy, fontSize: 26, fontVariant: ['tabular-nums'] },
    fieldText: { fontSize: 17 },
    notice: { flexDirection: 'row', alignItems: 'flex-start', gap: 8, paddingVertical: 10 },
    noticeText: { flex: 1, fontSize: 15, fontWeight: '600', color: p.ink, lineHeight: 20 },
    tone: { width: 22, height: 22, borderRadius: radius.chip, alignItems: 'center', justifyContent: 'center' },
    intro: { fontSize: 15, lineHeight: 21, color: p.ink2 },
    loading: { marginTop: 40 },
    failed: { padding: 16 },
    failedNotice: { paddingVertical: 0 },
    failedButton: { marginTop: 14 },
  });
}

export function useCardStyles() {
  const p = usePalette();
  return useMemo(() => ({ p, s: makeStyles(p) }), [p]);
}

export function Page({
  children,
  ref,
  ...props
}: ScrollViewProps & {
  children: ReactNode;
  ref?: Ref<ComponentRef<typeof ScrollView>>;
}) {
  const { s } = useCardStyles();
  const own = useRef<ComponentRef<typeof ScrollView>>(null);
  // Gallery only: open a state partway down, to show what sits below the fold.
  const { scrollY } = useOverrides().o;
  useEffect(() => {
    if (scrollY === undefined) return;
    const id = setTimeout(() => {
      if (scrollY === 'end') own.current?.scrollToEnd({ animated: false });
      else own.current?.scrollTo({ y: scrollY, animated: false });
    }, 350);
    return () => clearTimeout(id);
  }, [scrollY]);
  return (
    <ScrollView
      ref={(node) => {
        own.current = node;
        if (typeof ref === 'function') ref(node);
        else if (ref) ref.current = node;
      }}
      style={s.page}
      contentInsetAdjustmentBehavior="automatic"
      automaticallyAdjustKeyboardInsets
      keyboardDismissMode="interactive"
      contentContainerStyle={s.pageContent}
      {...props}>
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
  secondary,
  disabled,
  style,
}: {
  label: string;
  icon?: SFSymbol;
  onPress?: () => void;
  // Printed as an ink outline instead of a solid: the second of two choices.
  secondary?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
}) {
  const { p, s } = useCardStyles();
  const color = secondary ? p.ink : p.onButton;
  // The label is read as written, not in the caps it's printed in, and
  // without the icon's own name, such as the checkmark's "selected".
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        s.button,
        secondary && s.buttonSecondary,
        pressed && s.pressed,
        disabled && s.disabled,
        style,
      ]}>
      {icon ? <SymbolView name={icon} tintColor={color} size={18} weight="bold" /> : null}
      <Text style={[s.buttonText, { color }]} maxFontSizeMultiplier={displayScaleCap}>
        {label}
      </Text>
    </Pressable>
  );
}

// A text action in SF, centred, with a 44pt target: ink for an alternative,
// soft ink for Cancel, danger for a destructive one.
export function TextAction({
  label,
  icon,
  danger,
  muted,
  disabled,
  onPress,
  style,
}: {
  label: string;
  icon?: SFSymbol;
  danger?: boolean;
  muted?: boolean;
  disabled?: boolean;
  onPress?: () => void;
  style?: StyleProp<ViewStyle>;
}) {
  const { p, s } = useCardStyles();
  const color = danger ? p.danger : muted ? p.ink2 : p.ink;
  // Read as its words alone, like Button.
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [s.textAction, pressed && s.pressed, disabled && s.disabled, style]}>
      {icon ? <SymbolView name={icon} tintColor={color} size={16} weight="semibold" /> : null}
      <Text style={[s.textActionText, { color }]}>{label}</Text>
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

// A field printed on the card: stock on the ground, ground on a stock panel.
// Numbers are set in the condensed face; text such as an address in SF.
// A field the server refused is outlined in danger, beside the message that
// says why.
export function Field({
  text,
  onPanel,
  invalid,
  style,
  ...props
}: TextInputProps & { text?: boolean; onPanel?: boolean; invalid?: boolean }) {
  const { p, s } = useCardStyles();
  return (
    <TextInput
      placeholderTextColor={p.ink2}
      selectionColor={p.ink}
      maxFontSizeMultiplier={text ? undefined : displayScaleCap}
      {...props}
      style={[
        s.field,
        text ? s.fieldText : s.fieldNumber,
        { backgroundColor: onPanel ? p.ground : p.stock },
        style,
        invalid && { borderColor: p.danger, borderWidth: 2 },
      ]}
    />
  );
}

// A line of news under or over a form: an error in danger, a confirmation in
// ink. Each carries its symbol, so colour is never the only signal.
// VoiceOver reads it as one element, its words alone. iOS has no live
// regions, so an error is announced as it appears and whenever it changes;
// Android's live region reads it, and news, by itself.
export function Notice({
  error,
  icon,
  children,
  style,
}: {
  error?: boolean;
  icon?: SFSymbol;
  children: ReactNode;
  style?: StyleProp<ViewStyle>;
}) {
  const { p, s } = useCardStyles();
  const color = error ? p.danger : p.ink;
  const text = Children.toArray(children)
    .filter((c) => typeof c === 'string' || typeof c === 'number')
    .join('');
  useEffect(() => {
    if (error && text && Platform.OS === 'ios') AccessibilityInfo.announceForAccessibility(text);
  }, [error, text]);
  return (
    <View
      style={[s.notice, style]}
      accessible
      accessibilityRole={error ? 'alert' : undefined}
      accessibilityLiveRegion="polite">
      <SymbolView
        name={icon ?? (error ? 'exclamationmark.triangle.fill' : 'checkmark.circle.fill')}
        tintColor={color}
        size={18}
        style={{ marginTop: 1 }}
        accessibilityElementsHidden
      />
      <Text style={[s.noticeText, { color }]}>{children}</Text>
    </View>
  );
}

const toneSymbol: Record<Tone, SFSymbol> = { up: 'arrow.up', hold: 'equal', down: 'arrow.down' };
const toneWord: Record<Tone, string> = { up: 'Up', hold: 'Hold', down: 'Down' };

// Which way a training max is going: a square ink stamp with an arrow, read
// beside the verdict's own words. It grows with their text and sits centred
// on their first line, whose height (at the default text size) is `line`.
export function ToneMark({ tone, size = 22, line = size }: { tone: Tone; size?: number; line?: number }) {
  const { p, s } = useCardStyles();
  const { fontScale } = useWindowDimensions();
  const box = size * Math.min(fontScale, 2);
  return (
    <View
      style={[
        s.tone,
        { backgroundColor: p.ink, width: box, height: box, marginTop: Math.max(0, (line * fontScale - box) / 2) },
      ]}
      accessible
      accessibilityLabel={toneWord[tone]}>
      <SymbolView name={toneSymbol[tone]} tintColor={p.ground} size={box * 0.62} weight="heavy" />
    </View>
  );
}

// An SF paragraph in soft ink that opens a section or a form.
export function Intro({ children, style }: { children: ReactNode; style?: StyleProp<TextStyle> }) {
  const { s } = useCardStyles();
  return <Text style={[s.intro, style]}>{children}</Text>;
}

// A screen's read, as a query hook returns it.
type Loadable<D> = {
  data: D | undefined;
  error: Error | null;
  isFetching: boolean;
  refetch: () => unknown;
};

// A screen until its first load arrives, then the screen itself:
//   <LoadState query={useHistory()}>{(h) => <History h={h} />}</LoadState>
// A quick answer shows nothing meanwhile, so the spinner only appears after
// a moment. A failed first load says why and offers to ask again. Once there
// is data it stays on screen, even when a refetch fails: the query then
// reports an error with its data still set, so data is checked first.
export function LoadState<D>({ query, children }: { query: Loadable<D>; children: (data: D) => ReactNode }) {
  const { p, s } = useCardStyles();
  // Asking again clears the query's error until the answer comes, so the
  // last one stays on screen meanwhile, over a button that says it's trying.
  const [failure, setFailure] = useState<Error | null>(null);
  if (query.error && query.error !== failure) setFailure(query.error);
  const shown = query.error ?? failure;
  const waiting = query.data === undefined && !shown;
  const [slow, setSlow] = useState(false);
  useEffect(() => {
    if (!waiting) return;
    const id = setTimeout(() => setSlow(true), 400);
    return () => clearTimeout(id);
  }, [waiting]);

  if (query.data !== undefined) return children(query.data);
  if (shown) {
    return (
      <Page>
        <Panel style={s.failed}>
          <Notice error style={s.failedNotice}>
            {shown.message}
          </Notice>
          <Button
            label={query.isFetching ? 'Trying again…' : 'Try again'}
            disabled={query.isFetching}
            style={s.failedButton}
            onPress={() => void query.refetch()}
          />
        </Panel>
      </Page>
    );
  }
  return (
    <Page>
      {slow ? <ActivityIndicator size="large" color={p.ink2} accessibilityLabel="Loading" style={s.loading} /> : null}
    </Page>
  );
}
