import { Text, useWindowDimensions, View } from 'react-native';
import Svg, { Circle, Line, Path, Text as SvgText } from 'react-native-svg';

import type { Chart } from '@/api/schema.gen';
import { useCardStyles } from '@/components/card-back';
import { displayScaleCap, font, liftInk } from '@/theme';

// Draws the chart exactly as the server laid it out: the path and points
// arrive in a width x height space, and the axis labels are its yMin/yMax.
// A PR set logged in the app is a filled dot; a past estimate is a ring.
export function LiftChart({ chart, slug }: { chart: Chart; slug: string }) {
  const { p, s } = useCardStyles();
  const line = liftInk(p, slug).line;
  // Labels follow Dynamic Type up to the display cap, like every numeral.
  const k = Math.min(useWindowDimensions().fontScale, displayScaleCap);
  const padTop = 22 * k;
  const padBottom = 8 + 18 * k;
  // A label sits above its point unless the line runs in from above, where
  // it would cross the label; then it goes below.
  const labelBelow = (i: number) => {
    const ys = [chart.points[i - 1]?.y, chart.points[i + 1]?.y].filter((y) => y !== undefined);
    return ys.length > 0 && ys.every((y) => y < chart.points[i].y);
  };
  const padLeft = 30 + 6 * k;
  const padRight = 14;
  const w = chart.width + padLeft + padRight;
  const h = chart.height + padTop + padBottom;
  return (
    <View>
      <Svg width="100%" viewBox={`0 0 ${w} ${h}`} style={{ aspectRatio: w / h }}>
        {[0, 0.5, 1].map((t) => (
          <Line
            key={t}
            x1={padLeft}
            x2={w - padRight}
            y1={padTop + t * chart.height}
            y2={padTop + t * chart.height}
            stroke={p.rule}
            strokeWidth={1}
          />
        ))}
        {[
          { v: chart.yMax, y: padTop + 4 },
          { v: chart.yMin, y: padTop + chart.height + 4 },
        ].map((a) => (
          <SvgText
            key={a.v}
            x={padLeft - 6}
            y={a.y}
            fill={p.ink2}
            fontFamily={font.semi}
            fontSize={14 * k}
            textAnchor="end">
            {a.v}
          </SvgText>
        ))}
        <Path
          d={chart.path}
          transform={`translate(${padLeft} ${padTop})`}
          stroke={line}
          strokeWidth={3}
          fill="none"
          strokeLinejoin="round"
          strokeLinecap="round"
        />
        {chart.points.map((pt) => (
          <Circle
            key={`${pt.x}-${pt.y}`}
            cx={pt.x + padLeft}
            cy={pt.y + padTop}
            r={5}
            stroke={line}
            strokeWidth={2}
            fill={pt.past ? p.stock : line}
          />
        ))}
        {chart.points.map((pt, i) => (
          <SvgText
            key={`l-${pt.x}-${pt.y}`}
            x={pt.x + padLeft}
            y={pt.y + padTop + (labelBelow(i) ? 8 + 15 * k : -6 - 5 * k)}
            fill={p.ink}
            fontFamily={font.heavy}
            fontSize={15 * k}
            textAnchor="middle">
            {pt.label}
          </SvgText>
        ))}
      </Svg>
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6, marginTop: 6, marginLeft: padLeft }}>
        <View style={{ width: 10, height: 10, borderRadius: 5, backgroundColor: line }} />
        <Text style={s.sub}>PR sets</Text>
        <View
          style={{ width: 10, height: 10, borderRadius: 5, borderWidth: 2, borderColor: line, marginLeft: 10 }}
        />
        <Text style={s.sub}>Past estimates</Text>
      </View>
    </View>
  );
}
