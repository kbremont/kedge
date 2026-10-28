---
name: 531 Tracker
description: Training printed as the back of a sports card, read in a glance at the bar.
colors:
  chipboard: "#C9C3B7"
  chipboard-stock: "#D8D3C9"
  navy-ink: "#1B2240"
  navy-ink-soft: "#454B63"
  ink-rule: "rgba(27, 34, 64, 0.26)"
  danger-red: "#8A2A17"
  night-navy: "#121729"
  night-navy-stock: "#1C2238"
  chipboard-ink: "#E4DED2"
  chipboard-ink-soft: "#A9AEC2"
  chipboard-rule: "rgba(228, 222, 210, 0.18)"
  danger-coral: "#F07A62"
  band-white: "#FFFFFF"
  press-gold: "#D9A21B"
  press-gold-line: "#8A6206"
  press-gold-line-dark: "#E8B84A"
  deadlift-brick: "#C2412A"
  deadlift-brick-line: "#A8361F"
  deadlift-brick-dark: "#C24A33"
  deadlift-brick-line-dark: "#F07A62"
  bench-blue: "#2B5FA8"
  bench-blue-dark: "#3F72BD"
  bench-blue-line-dark: "#7FA6E6"
  squat-green: "#2E7A48"
  squat-green-line: "#276B3E"
  squat-green-dark: "#33844F"
  squat-green-line-dark: "#6CC08A"
typography:
  display-set:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "88px"
    fontWeight: 900
    lineHeight: "92px"
  display-band:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "46px"
    fontWeight: 900
    lineHeight: 1
    letterSpacing: "0.4px"
  headline:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "26px"
    fontWeight: 900
    letterSpacing: "0.4px"
  stat-value:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "32px"
    fontWeight: 800
    fontFeature: "tnum"
  number:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "24px"
    fontWeight: 800
    fontFeature: "tnum"
  number-small:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "18px"
    fontWeight: 600
    fontFeature: "tnum"
  band-slim:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "20px"
    fontWeight: 800
    letterSpacing: "0.6px"
  button:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "22px"
    fontWeight: 800
    letterSpacing: "0.6px"
  label:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "14px"
    fontWeight: 600
    letterSpacing: "0.8px"
  body-name:
    fontFamily: "SF Pro Text, -apple-system, system-ui"
    fontSize: "16px"
    fontWeight: 600
  body-sub:
    fontFamily: "SF Pro Text, -apple-system, system-ui"
    fontSize: "13px"
    fontWeight: 400
rounded:
  chip: "1px"
  panel: "6px"
spacing:
  page: "16px"
  gap: "12px"
  section: "30px"
components:
  page:
    backgroundColor: "{colors.chipboard}"
    padding: "8px 16px 48px"
  panel:
    backgroundColor: "{colors.chipboard-stock}"
    rounded: "{rounded.panel}"
  lift-band:
    backgroundColor: "{colors.bench-blue}"
    textColor: "{colors.band-white}"
    typography: "{typography.display-band}"
    padding: "14px 16px 12px"
  lift-band-slim:
    backgroundColor: "{colors.bench-blue}"
    textColor: "{colors.band-white}"
    typography: "{typography.band-slim}"
    padding: "9px 14px"
  stat:
    textColor: "{colors.navy-ink}"
    typography: "{typography.stat-value}"
    padding: "10px 14px"
  section-heading:
    textColor: "{colors.navy-ink}"
    typography: "{typography.headline}"
  button-primary:
    backgroundColor: "{colors.navy-ink}"
    textColor: "{colors.chipboard-stock}"
    typography: "{typography.button}"
    rounded: "{rounded.panel}"
    height: "54px"
  button-primary-dark:
    backgroundColor: "{colors.chipboard-ink}"
    textColor: "{colors.night-navy}"
    typography: "{typography.button}"
    rounded: "{rounded.panel}"
    height: "54px"
  rep-choice:
    backgroundColor: "{colors.chipboard}"
    textColor: "{colors.navy-ink}"
    typography: "{typography.number}"
    rounded: "{rounded.panel}"
    height: "52px"
  rep-choice-target:
    backgroundColor: "{colors.chipboard}"
    textColor: "{colors.bench-blue}"
    typography: "{typography.number}"
    rounded: "{rounded.panel}"
    height: "52px"
  rep-choice-pressed:
    backgroundColor: "{colors.bench-blue}"
    textColor: "{colors.band-white}"
  input-number:
    backgroundColor: "{colors.chipboard-stock}"
    textColor: "{colors.navy-ink}"
    typography: "{typography.stat-value}"
    rounded: "{rounded.panel}"
    height: "52px"
    padding: "0 12px"
  lift-chip:
    backgroundColor: "{colors.bench-blue}"
    rounded: "{rounded.chip}"
    size: "10px"
---

# Design System: 531 Tracker

## Overview

**Creative North Star: "Career Line"**

Every screen is the back of a sports card. The page is uncoated chipboard printed in one navy ink; a lift's identity arrives as a solid band of spot colour, the way a team colour heads a player's card; and every number sits in a ruled stat line, read across like a season's row. The lifter's history becomes a career, the best estimate becomes the totals line under the seasons, and the next set is the one stat printed large.

The system is dense and typographic rather than decorative. Condensed black caps carry headings and every figure; SF carries sentences and controls. There is no imagery, no gradient, no shadow and no glow: hierarchy comes from type size, ink weight, the heavy rule under a heading, and the one spot colour per lift. Dark appearance reverses the stock, not the idea: navy card, chipboard ink, the same lift inks.

The world sits inside native iOS structure. The tab bar, stack navigation, large titles, sheets, alerts, date pickers and SF Symbols are the system's own; the card back is printed on the content, not on the chrome.

**Key Characteristics:**
- Chipboard ground and navy ink in light; night navy ground and chipboard ink in dark.
- One spot colour per lift (press gold, deadlift brick, bench blue, squat green), used everywhere that lift appears and nowhere else.
- Sofia Sans Extra Condensed for headings and all numerals, tabular; SF for body text.
- Hairline-ruled stat rows; a 3pt ink rule under every section heading and over every totals line.
- Flat print: panels are a lighter stock, never lifted by shadow.
- Display numerals follow Dynamic Type to a 1.3x cap; body text scales fully.

## Colors

A two-ink print job on card stock, plus four spot inks that each belong to one lift.

### Primary
- **Navy Ink** (`navy-ink`): every heading, number, rule and the filled button in light appearance. The 3pt heading rule and the totals rule are this ink at full strength.
- **Chipboard Ink** (`chipboard-ink`): the same role in dark appearance; also the dark filled button.

### Secondary (the lift spot inks)
Each lift owns three values per appearance: a **band** fill for bands, chips and the PR target, an **on-band** text colour, and a **line** for strokes and marks drawn on the ground (chart line and points, the rest bar fill, rep choices that meet the target).
- **Press Gold** (`press-gold`; line `press-gold-line`, dark line `press-gold-line-dark`): the one band printed with navy ink on it in both appearances, because white does not hold on gold.
- **Deadlift Brick** (`deadlift-brick`, dark `deadlift-brick-dark`; lines `deadlift-brick-line`, `deadlift-brick-line-dark`): white on band.
- **Bench Blue** (`bench-blue`, dark `bench-blue-dark`; dark line `bench-blue-line-dark`; the light line is the band itself): white on band.
- **Squat Green** (`squat-green`, dark `squat-green-dark`; lines `squat-green-line`, `squat-green-line-dark`): white on band.

### Tertiary
- **Danger Red** (`danger-red`) / **Danger Coral** (`danger-coral`, dark): destructive actions and the "plates short" warning only. Text in it needs 4.5:1 against its surface; Danger Red holds 4.9:1 on the light ground and 5.9:1 on stock.

### Neutral
- **Chipboard** (`chipboard`): the light page ground, behind the native header and large title too.
- **Chipboard Stock** (`chipboard-stock`): panels printed a step lighter than the ground (the next-session card, the current set, the chart), and the light button's label.
- **Navy Ink Soft** (`navy-ink-soft`): labels, units, done sets, secondary lines, chart axis values.
- **Ink Rule** (`ink-rule`): the hairline between stat rows and columns, the rest bar track, chart gridlines.
- **Night Navy** (`night-navy`), **Night Navy Stock** (`night-navy-stock`), **Chipboard Ink Soft** (`chipboard-ink-soft`), **Chipboard Rule** (`chipboard-rule`): the same four roles in dark.
- **Band White** (`band-white`): text on the brick, blue and green bands.

### Named Rules
**The One Ink Per Lift Rule.** A lift's spot colour marks that lift and nothing else: its band, its chip, its chart line, its rest bar, its PR target. No spot ink is ever a generic accent, link colour or state colour, and no screen borrows another lift's ink.

**The Band, On-Band, Line Rule.** Fills use `band`, text on a fill uses `onBand`, and anything drawn on the ground or stock uses `line`. Never set a lift's band colour as text or a stroke on the ground.

## Typography

**Display Font:** Sofia Sans Extra Condensed, bundled at 600, 800 and 900 (PostScript `SofiaSansExtraCondensed-SemiBold`, `-ExtraBold`, `-Black`, registered so the native header can use them)
**Body Font:** SF Pro (the iOS system font)

**Character:** A tall, tight sports-programme numeral face against Apple's text face. The condensed face does the shouting and the counting; SF does the explaining.

### Hierarchy
- **Display Set** (900, 88pt, 92pt line): the current set's weight only. Its reps sit beside it at 56pt black with a 32pt semibold "×" in soft ink, and its unit at 24pt heavy in soft ink.
- **Display Band** (900, 46pt, line 1, +0.4 tracking, uppercase): the lift name in a full lift band.
- **Large Title** (900, iOS large-title size): the native large title on a tab root; the inline nav title uses 800 at 20pt.
- **Headline** (900, 26pt, +0.4, uppercase): section headings over the 3pt ink rule.
- **Stat Value** (800, 32pt, tabular): values in a stat row, with the unit at 17pt semibold in soft ink. Training-max columns on Today run at 34pt.
- **Totals / Rest Clock** (900, 30pt, tabular): the totals line's value and the rest countdown.
- **Number** (800, 24pt, tabular): numbers in table rows, rep choices, PR targets. **Number Small** (600, 18pt) for percentages and dates in a row.
- **Band Slim / Button** (800, 20pt and 22pt, +0.6, uppercase): slim band text, rest and totals labels (18pt), and button labels.
- **Label** (600, 14pt, +0.8, uppercase, soft ink): stat labels, column heads, field labels.
- **Body Name** (SF 600, 16pt): set names, session names, dates in prose rows, sheet links.
- **Body Sub** (SF 400, 13pt, soft ink): cycle lines under a name, notes, legends, the Logged line. Band subtitles are SF 600 at 15pt.

### Named Rules
**The Every Number Rule.** Every figure the lifter reads (weights, reps, percentages, maxes, dates in a table, clocks) is set in the condensed face with tabular numerals. Numbers inside a sentence stay in SF.

**The Display Cap Rule.** Text in the condensed face carries `maxFontSizeMultiplier` 1.3 so bands and the set weight stay on one line; SF body text scales fully. Above a 1.5x font scale, tables stack each row's label over its numbers and drop the cap on those numbers.

## Layout

A single scrolling column on the page ground, 16pt side margins, 8pt top and 48pt bottom padding inside the scroll view, with content inset adjusted under the native header. Sections open 30pt below the previous block. Values are iOS points.

Tables are built from rows, not cards: each row is a full-width line with a hairline rule below, roughly 10pt vertical padding, the name column flexible and numeric columns right-aligned at fixed widths (56 to 92pt). Stat rows divide their width equally with hairline column rules. Inside panels, content sits 14 to 16pt from the edge and buttons inset 12pt.

The session screen keeps the current set and its Log button in view: when the current set would fall below the fold, the table scrolls so the rest bar sits just under the header.

### Named Rules
**The Stat Line Rule.** Numbers live in ruled rows and columns, read across. Don't wrap a single stat in its own card.

## Elevation & Depth

None. The system is flat print: depth is a stock step (ground to the lighter panel stock) and nothing else. No shadows, no blur, no glow, and the native header's bottom shadow is turned off. The only translucent material on screen is the system's own tab bar and back button.

### Named Rules
**The Flat Print Rule.** A surface is either ground or stock. If something needs more emphasis, it gets more ink (a heavier rule, a band, larger type), never a shadow.

## Shapes

Two corners only: a soft 6pt corner on panels, bands that sit alone, buttons, rep choices and inputs, and a near-square 1pt corner on lift chips and the short lift bars. Panels clip their contents, so a band at the top of a panel takes the panel's corner and stays square at its bottom. Rules carry the form: a hairline (two device hairlines) between rows, 3pt solid ink under headings and over totals. Chart points are circles: filled for a logged PR set, a 2pt ring on stock for a past estimate.

## Components

### Buttons
Solid, square-shouldered, printed in the ink colour.
- **Shape:** soft corner (6pt), 54pt tall, full width of its container.
- **Primary:** navy ink fill with stock-coloured label in light; chipboard ink fill with night navy label in dark. Label in uppercase condensed 800 at 22pt; an optional 18pt bold SF Symbol leads it.
- **Pressed:** opacity 0.8. No scale, no colour change.
- **Text actions:** Undo is condensed 800 uppercase in ink with an SF Symbol, 44pt minimum height. Cancel in a sheet is SF 600 in soft ink. Discard is SF 600 in danger, centred, and confirms through a native destructive alert.

### Chips
- **Lift Chip:** a 10pt square of the lift's band ink (1pt corner) before a lift name in a list.
- **PR Target:** the PR set's rep target printed reversed out of the lift's band, at least 40pt wide, 7pt side padding.

### Cards / Containers
- **Corner Style:** soft (6pt).
- **Background:** stock on the ground.
- **Shadow Strategy:** none (see Elevation & Depth).
- **Border:** none; internal hairline rules divide content.
- **Internal Padding:** 14 to 16pt; a chart panel uses 12pt.

### Inputs / Fields
- **Number Field:** stock fill, hairline rule-coloured border, 6pt corner, 52pt tall, condensed 800 at 26pt tabular; a Label above it. Numeric keyboard.
- **Date:** the native SwiftUI DatePicker, seeded with the ink colour and the current appearance, in a ruled row with its Label.
- **Rep Picker:** a five-column grid of 52pt cells, 8pt gaps, ground fill, 2pt border. Cells that meet the target take the lift's line colour for border and number; others take the rule and ink. Pressed fills with the band and switches the number to on-band. One tap logs; there is no keyboard.

### Navigation
- **Tab bar:** native tabs with SF Symbols, tinted ink.
- **Stack header:** ground-coloured, no shadow, ink tint, title in condensed 800 at 20pt; back title "Today".
- **Tab root:** native large title in condensed 900. The header background stays unset there, because a header background hides the large title on iOS 26; the page ground shows through.
- **Sheets:** native form sheet with grabber, ground background, a 6pt lift bar across the top, a 30pt black uppercase title.

### Lift Band (signature)
The card's header in the lift's band ink. **Full:** lift name at 46pt black uppercase with an SF subtitle (the cycle and round). **Slim:** 20pt heavy uppercase text on both ends. On the session screen the slim band also carries the set table's column heads, printed reversed in on-band at 85% opacity.

### Stat Row
Equal columns divided by hairlines, each a Label over a 32pt value with a soft unit. The block, round and TM line on Today; training max and best estimate on a lift.

### Section Heading
26pt black uppercase over a 3pt ink rule, with an optional trailing action (an SF Symbol button such as `plus.circle.fill`).

### Set Table and Current Set
Done sets drop to soft ink with a checkmark beside their reps. The current set opens out of the table on a stock panel: 88pt weight, reps, an SF meta line, a hairline, then "Per side" plates in condensed 800 at 26pt, then the Log button or the rep picker. A "Logged" line with Undo sits between the rest bar and the current set.

### Rest Bar
Drawn to scale: the track (6pt, rule colour) is the full rest interval the server sent, and the fill in the lift's line colour drains in real seconds. "Rest" label at 18pt heavy, the clock at 30pt black.

### Totals Line
A 3pt ink rule over a row with an 18pt heavy uppercase label and a 30pt black value: the bottom line under a table, like career totals under the seasons.

### Lift Chart
The server's path drawn at 3pt in the lift's line colour with round joins, three rule-coloured gridlines, axis values in soft ink, point labels in condensed 800 placed above a point unless the line arrives from above. A legend in SF sub text names filled points and rings.

## Do's and Don'ts

### Do:
- **Do** print every number in Sofia Sans Extra Condensed with tabular numerals and cap its Dynamic Type growth at 1.3x.
- **Do** take lift colour from the lift's ink set (`band`, `onBand`, `line`) and use it everywhere that lift appears.
- **Do** put a 3pt ink rule under every section heading and over every totals line, and hairline rules between rows.
- **Do** separate surfaces only by the stock step: ground for the page, stock for panels.
- **Do** keep native iOS chrome native: tabs, stack headers, large titles, form sheets, alerts, date pickers and SF Symbols.
- **Do** stack a row's label over its numbers above a 1.5x font scale instead of shrinking or truncating.

### Don't:
- **Don't** use a lift's spot ink as a generic accent, link, selection or status colour.
- **Don't** add shadows, blur, gradients or glows to content; emphasis is ink and type.
- **Don't** wrap individual stats in their own cards; they belong in ruled rows and stat lines.
- **Don't** set a header background on a large-title tab root; it hides the large title on iOS 26.
- **Don't** add corner radii beyond the 6pt panel corner and the 1pt chip corner.
- **Don't** use a keyboard where a bounded set of answers can be tapped.
