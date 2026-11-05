import { fmtWeight, parseDecimal, weightField } from '@/lib/format';

test('a weight that can be loaded is written in full, anything else to one decimal', () => {
  expect(fmtWeight(100)).toBe('100');
  expect(fmtWeight(62.5)).toBe('62.5');
  expect(fmtWeight(1.25)).toBe('1.25');
  expect(fmtWeight(61.25)).toBe('61.25');
  expect(fmtWeight(262.333333)).toBe('262.3');
});

test('a weight to edit is written exactly', () => {
  expect(weightField(1.25)).toBe('1.25');
  expect(weightField(0.625)).toBe('0.625');
});

test('a decimal comma reads as a point, and anything else that is not a number as NaN', () => {
  expect(parseDecimal('102,5')).toBe(102.5);
  expect(parseDecimal(' 102.5 ')).toBe(102.5);
  expect(parseDecimal('100')).toBe(100);
  expect(parseDecimal('.5')).toBe(0.5);
  expect(parseDecimal('')).toBeNaN();
  expect(parseDecimal('abc')).toBeNaN();
  expect(parseDecimal('1,2,5')).toBeNaN();
});
