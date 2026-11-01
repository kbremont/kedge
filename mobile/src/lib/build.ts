import * as Application from 'expo-application';
import * as Updates from 'expo-updates';

// What this install is running, for Settings' App row: the version and build
// TestFlight and the App Store know it by, then where its code came from. A
// release build runs the bundle it was built with until an EAS update
// arrives, and runs that from the next launch; a development build loads its
// code from Metro.

export type Running = {
  version: string | null;
  build: string | null;
  // Code from Metro. A dev client still reports expo-updates as enabled, on
  // a bundle of its own, so __DEV__ is what tells.
  development: boolean;
  // The build's own bundle, rather than a downloaded update.
  embedded: boolean;
  // When the update running was published.
  createdAt: Date | null;
};

export function running(): Running {
  return {
    version: Application.nativeApplicationVersion,
    build: Application.nativeBuildVersion,
    development: __DEV__ || !Updates.isEnabled,
    embedded: Updates.isEmbeddedLaunch,
    createdAt: Updates.createdAt,
  };
}

// "1.0.0 (3) · built in", "1.0.0 (3) · update Nov 2", or
// "1.0.0 (1) · development". An update from another year gives the year.
export function runningText(r: Running, now = new Date()): string {
  const version = r.version ?? '—';
  const build = r.build ? ` (${r.build})` : '';
  return `${version}${build} · ${source(r, now)}`;
}

function source(r: Running, now: Date): string {
  if (r.development) return 'development';
  if (r.embedded || !r.createdAt) return 'built in';
  const date = r.createdAt.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: r.createdAt.getFullYear() === now.getFullYear() ? undefined : 'numeric',
  });
  return `update ${date}`;
}
