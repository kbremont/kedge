// The server's training.FormatWeight: no trailing ".0", one decimal otherwise.
export function fmtWeight(v: number): string {
  return Number.isInteger(v) ? String(v) : v.toFixed(1);
}

// The server's training.FormatPct.
export function fmtPct(p: number): string {
  return `${Math.round(p * 100)}%`;
}

// One side's plates, heaviest first, as the server lists them.
export function plateList(plates: string): string[] {
  return plates ? plates.split(', ') : [];
}

// The server writes "Anchor - Round 2"; the screens set it with a middle dot.
export function cycleText(label: string): string {
  return label.replace(' - ', ' · ');
}

// The cycle and the round of a label such as "Anchor - Round 2", apart. A
// one-round cycle such as "7th Week TM Test" has no round.
export function cycleParts(label: string): { cycle: string; round: string } {
  const [cycle, round = ''] = label.split(' - ');
  return { cycle, round: round.replace('Round ', '') };
}

// The program writes sets as "5x10"; the screens set a real multiplication
// sign.
export function timesIn(text: string): string {
  return text.replace(/(\d)x(\d)/g, '$1×$2');
}

// Column heads and short labels: the lift's everyday name.
const short: Record<string, string> = {
  press: 'Press',
  deadlift: 'Deadlift',
  bench: 'Bench',
  squat: 'Squat',
};

export function shortName(slug: string, name: string): string {
  return short[slug] ?? name;
}

export function fmtClock(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
}

// Server summaries write "330 x 7"; the screens set a real multiplication sign.
export function times(summary: string): string {
  return summary.replace(' x ', ' × ');
}

// A count or target of reps in words: "1 rep", "5 reps", "3+ reps". The
// program's singles, such as the 1+ week's top set and the TM test's 90%,
// take the singular.
export function repsText(reps: number | string): string {
  return `${reps} ${String(reps) === '1' ? 'rep' : 'reps'}`;
}
