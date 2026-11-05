import type { BBBMode, Settings, SettingsInput } from '@/api/schema.gen';
import { parseDecimal, weightField } from '@/lib/format';

// What Settings' pages save: everything but the training maxes, which save
// one at a time. Each page edits its own part and sends the rest as saved.
export type Form = {
  barWeight: string;
  rounding: string;
  plates: string;
  warmups: boolean;
  restWarmupSeconds: number;
  restMainSeconds: number;
  restSuppSeconds: number;
  bbb: Record<string, { mode: BBBMode; value: string }>;
};

export function toForm(s: Settings): Form {
  return {
    barWeight: weightField(s.barWeight),
    rounding: weightField(s.rounding),
    plates: s.plates.map(weightField).join(', '),
    warmups: s.warmups,
    restWarmupSeconds: s.restWarmupSeconds,
    restMainSeconds: s.restMainSeconds,
    restSuppSeconds: s.restSuppSeconds,
    bbb: Object.fromEntries(
      s.bbb.map((b) => [b.slug, { mode: b.mode, value: b.value > 0 ? weightField(b.value) : '' }]),
    ),
  };
}

// A field the form won't send as it stands, and what to put there.
export type Refusal = { field: string; message: string };

// A number as typed, with a decimal comma read as a point. Blank is 0, which
// the server reads as "keep what's there", and for a BBB value as "follow the
// program".
export function parseNumber(text: string): number | undefined {
  const t = text.trim();
  if (!t) return 0;
  const n = parseDecimal(t);
  return Number.isFinite(n) ? n : undefined;
}

// The form as PUT /settings takes it. Something typed that isn't a number is
// refused here, as the web's form refuses plates it can't read: JSON would
// carry it as null, and the server's answer to that names the schema, not
// the field.
export function toInput(f: Form, lifts: Settings['bbb']): SettingsInput | Refusal {
  const barWeight = parseNumber(f.barWeight);
  if (barWeight === undefined) return { field: 'barWeight', message: 'Enter the bar weight as a number.' };
  const rounding = parseNumber(f.rounding);
  if (rounding === undefined) return { field: 'rounding', message: 'Enter the rounding as a number.' };
  // Split as the web splits them, on commas and spaces. A comma here always
  // separates plates, so "1,25" is two of them; the page asks for a point.
  const plates = f.plates.split(/[\s,]+/).filter(Boolean).map(Number);
  if (!plates.every(Number.isFinite)) {
    return { field: 'plates', message: 'List the plates as numbers, separated by commas.' };
  }
  const bbb: SettingsInput['bbb'] = {};
  for (const l of lifts) {
    const { mode, value } = f.bbb[l.slug];
    // Program follows the cycle's own percentage, so its field is hidden and
    // nothing in it is sent.
    const n = mode === 'program' ? 0 : parseNumber(value);
    if (n === undefined) {
      const what = mode === 'percent' ? 'percentage' : 'weight';
      return { field: `bbb.${l.slug}`, message: `Enter the ${l.name} BBB ${what} as a number.` };
    }
    bbb[l.slug] = { mode, value: n };
  }
  return {
    barWeight,
    rounding,
    plates,
    warmups: f.warmups,
    restWarmupSeconds: f.restWarmupSeconds,
    restMainSeconds: f.restMainSeconds,
    restSuppSeconds: f.restSuppSeconds,
    bbb,
  };
}
