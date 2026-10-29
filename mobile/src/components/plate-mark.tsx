import Svg, { Rect } from 'react-native-svg';

import { liftInk, usePalette } from '@/theme';

// The app's mark: a bar's sleeve loaded with four plates, one in each lift's
// ink. Deadlift, bench, press and squat stand in the order of competition
// plates (red, blue, yellow, green). The icon is drawn from the same shapes.
export const plateMarkShapes = {
  shaft: { x: 4, y: 46, width: 26, height: 8 },
  collar: { x: 28, y: 37, width: 8, height: 26 },
  plates: [
    { slug: 'deadlift', x: 38, y: 12, width: 10, height: 76 },
    { slug: 'bench', x: 50, y: 18, width: 10, height: 64 },
    { slug: 'press', x: 62, y: 24, width: 10, height: 52 },
    { slug: 'squat', x: 74, y: 30, width: 10, height: 40 },
  ],
  sleeve: { x: 84, y: 44, width: 12, height: 12 },
} as const;

export function PlateMark({ size }: { size: number }) {
  const p = usePalette();
  const m = plateMarkShapes;
  return (
    <Svg width={size} height={size} viewBox="0 0 100 100" accessibilityElementsHidden>
      <Rect {...m.shaft} rx={1} fill={p.ink} />
      <Rect {...m.collar} rx={1.5} fill={p.ink} />
      {m.plates.map(({ slug, ...r }) => (
        <Rect key={slug} {...r} rx={1.5} fill={liftInk(p, slug).band} />
      ))}
      <Rect {...m.sleeve} rx={1} fill={p.ink} />
    </Svg>
  );
}
