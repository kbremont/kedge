---
version: 1
slug: "mobile-src-app-tabs-today-index-tsx"
primary_target: "mobile/src/app/(tabs)/(today)/index.tsx"
related_targets: ["mobile/src/app/workout/[id].tsx","mobile/src/app/lift/[slug].tsx"]
---

# iOS app: working screens

**Mode:** Operate. The lifter completes a session at the bar and settles training max decisions away from it.

**Scope:** the app's screens, starting with Today, the session screen and the lift page; connect, setup, history and settings follow in the same world.

**Audience and task:** the owner first, other self-hosters second (PRODUCT.md). At the bar: read weight, plates and reps, lift, log, rest. Away from it: history, the estimated-max chart, settings, decisions.

**Constraints:** native iOS structure (tab bar, stack, large titles, SF Symbols, system controls); the server computes every number; light and dark appearances both ship.

**Memorable moment:** the session as a stat table, the current set opening out of it at 88pt with its plates and log button.

**Unresolved:** app name and icon; Dynamic Type ceiling for the 88pt set weight.

## Direction contract

THESIS: Training read as a sports card's career stat line. Refuses the category's dark dashboard of cards with one neon accent.

OWN-WORLD: Chipboard stock ground, navy ink, one spot colour per lift (press gold, deadlift brick, bench blue, squat green) used everywhere that lift appears. Sofia Sans Extra Condensed for headings and numbers, SF for body. Ruled stat rows, 3pt ink rules under headings, bands in the lift's colour. Dark: navy stock, chipboard ink.

STORY: The lifter sees where they are in the block, what the next set asks, and their numbers over time, and logs each set with one tap.

FIRST VIEWPORT: Today: large title, the next lift's colour band with its name at 46pt, a block/round/TM stat line, then Start session full width. Session: the set table, a to-scale rest bar, the current set at 88pt with its plates and Log button.

FORM: sports card back, candidate 6 of 7. seed:9cf574db71572f32

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
