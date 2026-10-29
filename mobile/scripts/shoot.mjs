#!/usr/bin/env node
// Screenshots every gallery state in the iOS simulator, in light and dark,
// and lays them out as contact sheets.
//
// It needs the dev build installed on a booted simulator and Metro started
// with the remote stepper on (npm run start:shoot). Metro runs on its own
// port, 8193, so the dev client reopens it at launch once it has been picked
// from the launcher the first time. Don't start it with CI=1: that turns off
// Metro's file watching, and the captures keep showing the code as it was
// when Metro started. Deep links bring up an
// iOS "Open in…?" prompt, so the script moves between states through the
// stepper instead: it serves the next path on 127.0.0.1:8194, which the app
// polls (src/dev/remote-step.ts).
//
//   node scripts/shoot.mjs [--only today,history] [--appearance light,dark]
//                          [--out ../.context/mockups] [--sheet ../docs/mockups]
//                          [--device <udid>] [--wait 2200] [--sheet-only]
import { execFileSync } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:http';
import { dirname, join, resolve } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');

const args = Object.fromEntries(
  process.argv
    .slice(2)
    .join(' ')
    .split(/\s*--/)
    .filter(Boolean)
    .map((a) => {
      const [k, ...v] = a.split(/\s+/);
      return [k, v.join(' ')];
    }),
);
const out = resolve(root, args.out || '../.context/mockups');
const sheetDir = resolve(root, args.sheet || '../docs/mockups');
const appearances = (args.appearance || 'light,dark').split(',');
const wait = Number(args.wait || 2200);

// States that leave something native on screen (an alert) go last, and the
// app restarts after them.
const modal = new Set(['workout-discard']);
// A state that opens scrolled leaves its tab's large title collapsed, so the
// app restarts after it too.
const scrolled = new Set(
  readFileSync(join(root, 'src/gallery/states.ts'), 'utf8')
    .split(/\{\s*id: '/)
    .slice(1)
    .filter((chunk) => chunk.split(/\n  \},?\n/)[0].includes('scrollY'))
    .map((chunk) => chunk.slice(0, chunk.indexOf("'"))),
);
// Sheets and pushes several deep take a moment longer to settle.
const slow = (id) => /^(lift-add|lift-error|setup-leader|setup-anchor|workout-completed)$/.test(id);

const states = [
  ...readFileSync(join(root, 'src/gallery/states.ts'), 'utf8').matchAll(
    /id: '([^']+)',\s*screen: '([^']+)',\s*label: '([^']+)'/g,
  ),
].map(([, id, screen, label]) => ({ id, screen, label }));
const only = args.only ? args.only.split(',') : null;
const shots = states
  .filter((s) => !only || only.some((o) => s.id === o || s.id.startsWith(`${o}-`)))
  .sort((a, b) => Number(modal.has(a.id)) - Number(modal.has(b.id)));

function sh(cmd, ...a) {
  return execFileSync(cmd, a, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

// --sheet-only rebuilds the contact sheets from the screenshots already on
// disk, without the simulator.
if (!('sheet-only' in args)) await capture();

async function capture() {
  const device =
    args.device ||
    Object.values(JSON.parse(sh('xcrun', 'simctl', 'list', 'devices', 'booted', '-j')).devices)
      .flat()
      .find((d) => d.state === 'Booted')?.udid;
  if (!device) throw new Error('No booted simulator. Boot one and install the dev build first.');

  const config = JSON.parse(readFileSync(join(root, 'app.json'), 'utf8')).expo;
  const bundle = config.ios?.bundleIdentifier ?? `com.anonymous.${config.slug}`;

  try {
    await fetch('http://127.0.0.1:8193/status');
  } catch {
    throw new Error('Metro is not running. Start it with npm run start:shoot.');
  }

  let step = '/';
  const server = createServer((_, res) => res.end(step));
  await new Promise((ok, fail) => {
    server.once('error', (e) =>
      fail(e.code === 'EADDRINUSE' ? new Error('Port 8194 is taken; stop the other stepper first.') : e),
    );
    server.listen(8194, '127.0.0.1', ok);
  });

  // A clean status bar, and no dev tools button over the screens.
  sh('xcrun', 'simctl', 'status_bar', device, 'override', '--time', '9:41', '--batteryState', 'charged',
    '--batteryLevel', '100', '--cellularMode', 'active', '--cellularBars', '4', '--wifiBars', '3');
  sh('xcrun', 'simctl', 'spawn', device, 'defaults', 'write', bundle, 'EXDevMenuShowFloatingActionButton', '-bool', 'NO');

  async function relaunch() {
    step = '/';
    try {
      sh('xcrun', 'simctl', 'terminate', device, bundle);
    } catch {
      // Not running.
    }
    sh('xcrun', 'simctl', 'launch', device, bundle);
    await sleep(9000);
  }

  mkdirSync(out, { recursive: true });
  let n = 0;
  for (const appearance of appearances) {
    sh('xcrun', 'simctl', 'ui', device, 'appearance', appearance);
    await relaunch();
    for (const s of shots) {
      step = `/gallery?state=${s.id}&n=${n++}`;
      await sleep(wait + (slow(s.id) ? 900 : 0) + (modal.has(s.id) ? 600 : 0));
      const file = join(out, `${s.id}-${appearance}.png`);
      sh('xcrun', 'simctl', 'io', device, 'screenshot', file);
      console.log(file);
      if (modal.has(s.id) || scrolled.has(s.id)) await relaunch();
    }
  }
  server.close();
}

// One sheet per appearance, grouped by screen.
mkdirSync(sheetDir, { recursive: true });
for (const appearance of appearances) {
  const sections = [];
  for (const s of states.filter((x) => shots.includes(x))) {
    let sec = sections.at(-1);
    if (sec?.title !== s.screen) sections.push((sec = { title: s.screen, items: [] }));
    sec.items.push({ label: s.label, path: join(out, `${s.id}-${appearance}.png`) });
  }
  const spec = join(out, `sheet-${appearance}.json`);
  writeFileSync(
    spec,
    JSON.stringify({
      title: `Every screen and state · ${appearance}`,
      sections,
      columns: 6,
      thumbWidth: 360,
      dark: appearance === 'dark',
      font: join(root, 'assets/fonts/SofiaSansExtraCondensed-900.ttf'),
    }),
  );
  const file = join(sheetDir, appearance === 'light' ? 'overview.png' : `overview-${appearance}.png`);
  sh('swift', join(root, 'scripts/contact-sheet.swift'), spec, file);
  console.log(file);
}
