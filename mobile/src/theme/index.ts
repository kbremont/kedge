// Career Line: the back of a sports card. Chipboard stock and navy ink in
// light; navy stock and chipboard ink in dark. Each lift owns one spot colour
// in both.
import { useColorScheme } from 'react-native';

export type LiftSlug = 'press' | 'deadlift' | 'bench' | 'squat';

export type Palette = {
  // The page.
  ground: string;
  // Panels printed a step lighter than the ground: the next session, the
  // current set, the chart.
  stock: string;
  ink: string;
  // Labels, units, done sets, secondary lines.
  ink2: string;
  // Hairline rules between stat rows.
  rule: string;
  // Filled buttons.
  button: string;
  onButton: string;
  danger: string;
  lift: Record<LiftSlug, LiftInk>;
};

export type LiftInk = {
  // Band fills and chips.
  band: string;
  // Text on the band.
  onBand: string;
  // Strokes and marks on the ground: chart lines, the rest bar.
  line: string;
};

const light: Palette = {
  ground: '#C9C3B7',
  stock: '#D8D3C9',
  ink: '#1B2240',
  ink2: '#454B63',
  rule: 'rgba(27, 34, 64, 0.26)',
  button: '#1B2240',
  onButton: '#D8D3C9',
  danger: '#8A2A17',
  lift: {
    press: { band: '#D9A21B', onBand: '#1B2240', line: '#8A6206' },
    deadlift: { band: '#C2412A', onBand: '#FFFFFF', line: '#A8361F' },
    bench: { band: '#2B5FA8', onBand: '#FFFFFF', line: '#2B5FA8' },
    squat: { band: '#2E7A48', onBand: '#FFFFFF', line: '#276B3E' },
  },
};

const dark: Palette = {
  ground: '#121729',
  stock: '#1C2238',
  ink: '#E4DED2',
  ink2: '#A9AEC2',
  rule: 'rgba(228, 222, 210, 0.18)',
  button: '#E4DED2',
  onButton: '#121729',
  danger: '#F07A62',
  lift: {
    press: { band: '#D9A21B', onBand: '#1B2240', line: '#E8B84A' },
    deadlift: { band: '#C24A33', onBand: '#FFFFFF', line: '#F07A62' },
    bench: { band: '#3F72BD', onBand: '#FFFFFF', line: '#7FA6E6' },
    squat: { band: '#33844F', onBand: '#FFFFFF', line: '#6CC08A' },
  },
};

export function usePalette(): Palette {
  return useColorScheme() === 'dark' ? dark : light;
}

export function liftInk(p: Palette, slug: string): LiftInk {
  return p.lift[slug as LiftSlug] ?? p.lift.bench;
}

// Sofia Sans Extra Condensed carries headings and every number; SF carries
// body text and controls. Registered under their PostScript names so the
// native header can find them too.
export const font = {
  semi: 'SofiaSansExtraCondensed-SemiBold',
  heavy: 'SofiaSansExtraCondensed-ExtraBold',
  black: 'SofiaSansExtraCondensed-Black',
};

export const fontFiles = {
  [font.semi]: require('@/assets/fonts/SofiaSansExtraCondensed-600.ttf'),
  [font.heavy]: require('@/assets/fonts/SofiaSansExtraCondensed-800.ttf'),
  [font.black]: require('@/assets/fonts/SofiaSansExtraCondensed-900.ttf'),
};

// Display numerals grow with Dynamic Type only this far, so the set weight
// and the lift band stay on one line; body text scales fully.
export const displayScaleCap = 1.3;

export const space = { page: 16, gap: 12, section: 30 };

export const radius = { panel: 6, chip: 1 };
