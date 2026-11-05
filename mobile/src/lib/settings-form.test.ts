import settingsFixture from '@/api/fixtures/settings.json';
import type { Settings } from '@/api/schema.gen';
import { parseNumber, toForm, toInput, type Form } from '@/lib/settings-form';

const fixture = settingsFixture as Settings;

test('the form shows the settings as typed text, a BBB that follows the program blank', () => {
  expect(toForm(fixture)).toEqual({
    barWeight: '45',
    rounding: '5',
    plates: '45, 35, 25, 10, 5, 2.5',
    warmups: true,
    restWarmupSeconds: 60,
    restMainSeconds: 180,
    restSuppSeconds: 90,
    bbb: {
      press: { mode: 'program', value: '' },
      deadlift: { mode: 'weight', value: '185' },
      bench: { mode: 'program', value: '' },
      squat: { mode: 'percent', value: '55' },
    },
  });
});

test('a form left as it was sends the settings back as they are', () => {
  expect(toInput(toForm(fixture), fixture.bbb)).toEqual({
    barWeight: 45,
    rounding: 5,
    plates: [45, 35, 25, 10, 5, 2.5],
    warmups: true,
    restWarmupSeconds: 60,
    restMainSeconds: 180,
    restSuppSeconds: 90,
    bbb: {
      press: { mode: 'program', value: 0 },
      deadlift: { mode: 'weight', value: 185 },
      bench: { mode: 'program', value: 0 },
      squat: { mode: 'percent', value: 55 },
    },
  });
});

test.each([
  ['', 0],
  ['  ', 0],
  ['35', 35],
  [' 2.5 ', 2.5],
  ['4.5.', undefined],
  ['abc', undefined],
])('%j reads as %j', (text, n) => {
  expect(parseNumber(text)).toBe(n);
});

test('plates split on commas and spaces, in the order typed', () => {
  const input = toInput({ ...toForm(fixture), plates: '5, 45 ,25  10' }, fixture.bbb);
  expect(input).toMatchObject({ plates: [5, 45, 25, 10] });
});

test('a program BBB sends nothing typed in its hidden field', () => {
  const form = toForm(fixture);
  form.bbb.deadlift = { mode: 'program', value: 'abc' };
  expect(toInput(form, fixture.bbb)).toMatchObject({ bbb: { deadlift: { mode: 'program', value: 0 } } });
});

test.each<{ what: string; edit: (f: Form) => void; field: string; message: string }>([
  {
    what: 'a bar weight',
    edit: (f) => (f.barWeight = '4.5.'),
    field: 'barWeight',
    message: 'Enter the bar weight as a number.',
  },
  { what: 'a rounding', edit: (f) => (f.rounding = 'five'), field: 'rounding', message: 'Enter the rounding as a number.' },
  {
    what: 'a plate',
    edit: (f) => (f.plates = '45, abc'),
    field: 'plates',
    message: 'List the plates as numbers, separated by commas.',
  },
  {
    what: 'a BBB percentage',
    edit: (f) => (f.bbb.squat.value = '5..5'),
    field: 'bbb.squat',
    message: 'Enter the Squat BBB percentage as a number.',
  },
  {
    what: 'a BBB weight',
    edit: (f) => (f.bbb.deadlift.value = 'x'),
    field: 'bbb.deadlift',
    message: 'Enter the Deadlift BBB weight as a number.',
  },
])('$what that isn’t a number is refused, naming its field', ({ edit, field, message }) => {
  const form = toForm(fixture);
  edit(form);
  expect(toInput(form, fixture.bbb)).toEqual({ field, message });
});

test('a kilogram setup saves back unchanged, a 1.25 plate included', () => {
  const kg: Settings = { ...fixture, unit: 'kg', barWeight: 20, rounding: 2.5, plates: [25, 20, 15, 10, 5, 2.5, 1.25] };
  const input = toInput(toForm(kg), kg.bbb);
  expect(input).toMatchObject({ barWeight: 20, rounding: 2.5, plates: [25, 20, 15, 10, 5, 2.5, 1.25] });
});

test('a decimal comma reads as a point in a single number', () => {
  expect(parseNumber('2,5')).toBe(2.5);
});
