import { runningText, type Running } from '@/lib/build';

const release: Running = { version: '1.0.0', build: '3', development: false, embedded: true, createdAt: null };
const now = new Date(2026, 10, 20);

test('a release build on its own bundle says so', () => {
  expect(runningText({ ...release, createdAt: new Date(2026, 10, 1) }, now)).toBe('1.0.0 (3) · built in');
});

test('a release build on an update gives the day it was published', () => {
  const r = { ...release, embedded: false, createdAt: new Date(2026, 10, 2, 18, 30) };
  expect(runningText(r, now)).toBe('1.0.0 (3) · update Nov 2');
});

test('an update from another year gives the year too', () => {
  const r = { ...release, embedded: false, createdAt: new Date(2025, 11, 30) };
  expect(runningText(r, now)).toBe('1.0.0 (3) · update Dec 30, 2025');
});

test('a development build says so, whatever expo-updates reports', () => {
  expect(runningText({ ...release, build: '1', development: true }, now)).toBe('1.0.0 (1) · development');
});

test('a build with no version or build number still reads', () => {
  expect(runningText({ ...release, version: null, build: null }, now)).toBe('— · built in');
});
