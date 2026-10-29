---
name: Kedge
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
  title:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "30px"
    fontWeight: 900
    lineHeight: "32px"
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
  number-field:
    fontFamily: "Sofia Sans Extra Condensed"
    fontSize: "26px"
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
  body-intro:
    fontFamily: "SF Pro Text, -apple-system, system-ui"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: "21px"
  body-notice:
    fontFamily: "SF Pro Text, -apple-system, system-ui"
    fontSize: "15px"
    fontWeight: 600
    lineHeight: "20px"
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
  ink-band:
    backgroundColor: "{colors.navy-ink}"
    textColor: "{colors.chipboard-stock}"
    typography: "{typography.title}"
    padding: "14px 16px"
  ink-band-dark:
    backgroundColor: "{colors.chipboard-ink}"
    textColor: "{colors.night-navy}"
    typography: "{typography.title}"
    padding: "14px 16px"
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
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.navy-ink}"
    typography: "{typography.button}"
    rounded: "{rounded.panel}"
    height: "54px"
  button-secondary-dark:
    backgroundColor: "transparent"
    textColor: "{colors.chipboard-ink}"
    typography: "{typography.button}"
    rounded: "{rounded.panel}"
    height: "54px"
  text-action:
    textColor: "{colors.navy-ink}"
    typography: "{typography.body-name}"
    height: "44px"
  text-action-muted:
    textColor: "{colors.navy-ink-soft}"
    typography: "{typography.body-name}"
    height: "44px"
  text-action-danger:
    textColor: "{colors.danger-red}"
    typography: "{typography.body-name}"
    height: "44px"
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
    typography: "{typography.number-field}"
    rounded: "{rounded.panel}"
    height: "52px"
    padding: "0 12px"
  input-number-on-panel:
    backgroundColor: "{colors.chipboard}"
    textColor: "{colors.navy-ink}"
    typography: "{typography.number-field}"
    rounded: "{rounded.panel}"
    height: "52px"
    padding: "0 12px"
  input-text:
    backgroundColor: "{colors.chipboard-stock}"
    textColor: "{colors.navy-ink}"
    rounded: "{rounded.panel}"
    height: "52px"
    padding: "0 12px"
  notice-error:
    textColor: "{colors.danger-red}"
    typography: "{typography.body-notice}"
    padding: "10px 0"
  notice-confirm:
    textColor: "{colors.navy-ink}"
    typography: "{typography.body-notice}"
    padding: "10px 0"
  tone-mark:
    backgroundColor: "{colors.navy-ink}"
    textColor: "{colors.chipboard}"
    rounded: "{rounded.chip}"
    size: "22px"
  tone-mark-small:
    backgroundColor: "{colors.navy-ink}"
    textColor: "{colors.chipboard}"
    rounded: "{rounded.chip}"
    size: "18px"
  career-cell:
    textColor: "{colors.navy-ink}"
    padding: "9px 7px"
  career-cell-pressed:
    backgroundColor: "{colors.chipboard-stock}"
  setting-row:
    textColor: "{colors.navy-ink}"
    typography: "{typography.body-name}"
    height: "64px"
  lift-chip:
    backgroundColor: "{colors.bench-blue}"
    rounded: "{rounded.chip}"
    size: "10px"
  plate-mark:
    size: "76px"
---

# Design System: Kedge

## Overview

**Creative North Star: "Career Line"**

Every screen is the back of a sports card. The page is uncoated chipboard printed in one navy ink; a lift's identity arrives as a solid band of spot colour, the way a team colour heads a player's card; and every number sits in a ruled stat line, read across like a season's row. The lifter's history becomes a career, the best estimate becomes the totals line under the seasons, and the next set is the one stat printed large.

The system is dense and typographic rather than decorative. Condensed black caps carry headings and every figure; SF carries sentences and controls. There is no imagery, no gradient, no shadow and no glow: hierarchy comes from type size, ink weight, the heavy rule under a heading, and the one spot colour per lift. Dark appearance reverses the stock, not the idea: navy card, chipboard ink, the same lift inks.

The world sits inside native iOS structure. The tab bar, stack navigation, large titles, sheets, alerts and prompts, swipe actions, date pickers, steppers, segmented pickers, switches and SF Symbols are the system's own, tinted from the inks; the card back is printed on the content, not on the chrome.

**Key Characteristics:**
- Chipboard ground and navy ink in light; night navy ground and chipboard ink in dark.
- One spot colour per lift (press gold, deadlift brick, bench blue, squat green), used everywhere that lift appears and nowhere else.
- Sofia Sans Extra Condensed for headings and all numerals, tabular; SF for body text.
- Hairline-ruled stat rows; a 3pt ink rule under every section heading and over every totals line.
- Flat print: panels are a lighter stock, never lifted by shadow.
- Display numerals follow Dynamic Type to a 1.3x cap; body text scales fully; above 1.5x, rows stack and their numbers scale fully.

## Colors

A two-ink print job on card stock, plus four spot inks that each belong to one lift, and one danger ink for what is wrong.

### Primary
- **Navy Ink** (`navy-ink`): every heading, number, rule and the filled button in light appearance. The 3pt heading rule and the totals rule are this ink at full strength. It also fills the ink band and the tone mark, and tints native controls.
- **Chipboard Ink** (`chipboard-ink`): the same role in dark appearance; also the dark filled button and the dark ink band.

### Secondary (the lift spot inks)
Each lift owns three values per appearance: a **band** fill for bands, chips, its plate in the mark and the PR set, an **on-band** text colour, and a **line** for strokes and marks drawn on the ground (chart line and points, the rest bar fill, rep choices that meet the target).
- **Press Gold** (`press-gold`; line `press-gold-line`, dark line `press-gold-line-dark`): the one band printed with navy ink on it in both appearances, because white does not hold on gold.
- **Deadlift Brick** (`deadlift-brick`, dark `deadlift-brick-dark`; lines `deadlift-brick-line`, `deadlift-brick-line-dark`): white on band.
- **Bench Blue** (`bench-blue`, dark `bench-blue-dark`; dark line `bench-blue-line-dark`; the light line is the band itself): white on band.
- **Squat Green** (`squat-green`, dark `squat-green-dark`; lines `squat-green-line`, `squat-green-line-dark`): white on band.

### Tertiary
- **Danger Red** (`danger-red`) / **Danger Coral** (`danger-coral`, dark): destructive actions (Discard, the swiped Delete), plates short, error messages and fields the server refused. Text in it needs 4.5:1 against its surface; Danger Red holds 4.9:1 on the light ground and 5.9:1 on stock. The swiped Delete prints the ground colour on a danger fill.

### Neutral
- **Chipboard** (`chipboard`): the light page ground, behind the native header and large title too; the fill of a field on a stock panel; the arrow in the tone mark.
- **Chipboard Stock** (`chipboard-stock`): panels printed a step lighter than the ground (the next-session card, the current set, the chart, the TM test and review cards), fields on the ground, a pressed career-table cell, and the light button's label.
- **Navy Ink Soft** (`navy-ink-soft`): labels, units, done sets in a live session, secondary lines, intros, chart axis values, chevrons, the held lock.
- **Ink Rule** (`ink-rule`): the hairline between stat rows and columns, field borders, the rest bar track, the off switch track, chart gridlines.
- **Night Navy** (`night-navy`), **Night Navy Stock** (`night-navy-stock`), **Chipboard Ink Soft** (`chipboard-ink-soft`), **Chipboard Rule** (`chipboard-rule`): the same four roles in dark.
- **Band White** (`band-white`): text on the brick, blue and green bands.

### Named Rules
**The One Ink Per Lift Rule.** A lift's spot colour marks that lift and nothing else: its band, its chip, its plate in the mark, its chart line, its rest bar, its PR set. No spot ink is ever a generic accent, link colour, verdict or state colour, and no screen borrows another lift's ink.

**The Band, On-Band, Line Rule.** Fills use `band`, text on a fill uses `onBand`, and anything drawn on the ground or stock uses `line`. Never set a lift's band colour as text or a stroke on the ground.

**The Ink Band Rule.** Content about every lift at once (the block review) is headed by the ink band: the button colours as a band. It is the one band not in a lift's colour, and it is never used for a single lift.

**The Uncoloured Verdict Rule.** A verdict on a training max (up, hold, down) is the ink tone mark plus the server's own words. It is never green, red or a lift's ink.

**The Danger Signal Rule.** Danger is never colour alone. It arrives as a word (Discard, Delete), as the `exclamationmark.triangle.fill` symbol (plates short, an error message), or as a 2pt outline on a field beside the message that says why.

## Typography

**Display Font:** Sofia Sans Extra Condensed, bundled at 600, 800 and 900 (PostScript `SofiaSansExtraCondensed-SemiBold`, `-ExtraBold`, `-Black`, registered so the native header can use them)
**Body Font:** SF Pro (the iOS system font)

**Character:** A tall, tight sports-programme numeral face against Apple's text face. The condensed face does the shouting and the counting; SF does the explaining.

### Hierarchy
- **Display Set** (900, 88pt, 92pt line): the current set's weight only. Its reps sit beside it at 56pt black with a 32pt semibold "×" in soft ink, and its unit at 24pt heavy in soft ink.
- **Masthead** (900, 64pt, line 1, +0.4, uppercase): the app's name on the connect screen only.
- **Display Band** (900, 46pt, line 1, +0.4 tracking, uppercase): the lift name in a full lift band.
- **Large Title** (900, iOS large-title size): the native large title on a tab root; the inline nav title uses 800 at 20pt.
- **Title** (900, 30pt, 32pt line, +0.4, uppercase): a sheet's title and the ink band's title.
- **Headline** (900, 26pt, +0.4, uppercase): section headings over the 3pt ink rule.
- **Stat Value** (800, 32pt, tabular): values in a stat row, with the unit at 17pt semibold in soft ink. Training-max columns on Today run at 34pt.
- **Totals / Rest Clock** (900, 30pt, tabular): the totals line's value and the rest countdown.
- **Number Field** (800, 26pt, tabular): numbers typed into a field, and "Per side" plates. Smaller fields in rows run at 24pt; the setup table at 30pt.
- **Number** (800, 24pt, tabular): numbers in table rows, rep choices, PR targets. Career-table cells run at 20pt. **Number Small** (600, 18pt) for percentages and dates in a row; a career-table cell's date runs at 14pt with the face's own digit widths, because it reads as one word under its top set rather than down a column.
- **Band Slim / Button** (800, 20pt and 22pt, +0.6, uppercase): slim band text, rest and totals labels (18pt), and button labels.
- **Label** (600, 14pt, +0.8, uppercase, soft ink): stat labels, column heads, field labels. The career table's cycle label runs at 13pt.
- **Body Name** (SF 600, 16pt): set names, session names, dates in prose rows, setting labels, sheet links, text actions.
- **Body Intro** (SF 400, 15pt, 21pt line, soft ink): the paragraph that opens a section, a form or an empty panel.
- **Body Notice** (SF 600, 15pt, 20pt line): a Notice, and a verdict line in the block review.
- **Body Sub** (SF 400, 13pt, soft ink): cycle lines under a name, notes, legends, the Logged line, evidence lines, setting subs, server values, the footer. Band subtitles and the ink band's intro are SF 600 at 15pt.

### Named Rules
**The Every Number Rule.** Every figure the lifter reads (weights, reps, percentages, maxes, dates in a table, clocks) is set in the condensed face with tabular numerals. Numbers inside a sentence stay in SF.

**The Display Cap Rule.** Text in the condensed face carries `maxFontSizeMultiplier` 1.3 so bands and the set weight stay on one line; SF body text scales fully. Above a 1.5x font scale, rows stack instead of shrinking: a set, a past estimate, a training max or a setting puts its label over its numbers or control, and History turns its four columns into per-round lists. Numbers on a stacked line drop the cap. The tone mark grows with the text beside it up to 2x.

## Layout

A single scrolling column on the page ground, 16pt side margins, 8pt top and 48pt bottom padding inside the scroll view, with content inset adjusted under the native header. Sections open 30pt below the previous block. Cards stacked at the top of Today (TM test results, the review, the next session) sit 12pt apart. Values are iOS points.

Tables are built from rows, not cards: each row is a full-width line with a hairline rule below, roughly 10pt vertical padding, the name column flexible and numeric columns right-aligned at fixed widths (56 to 96pt). Stat rows divide their width equally with hairline column rules. Inside panels, content sits 14 to 16pt from the edge and buttons inset 12pt; a second button sits 8pt below the first.

Setting rows are at least 64pt tall: the label and its sub on the left, the control on the right, 12pt apart. Single-choice and checklist rows are also 64pt and open under a 3pt ink rule. The connect screen runs without a header: 20pt side margins, the masthead 36pt below the safe area.

The career table is a grid inside the column: a 62pt round column, then four equal lift columns divided by hairlines, with the column heads pinned under the header while the rows scroll.

The session screen keeps the current set and its Log button in view: when the current set would fall below the fold, the table scrolls so the rest bar sits just under the header.

### Named Rules
**The Stat Line Rule.** Numbers live in ruled rows and columns, read across. Don't wrap a single stat in its own card.

## Elevation & Depth

None. The system is flat print: depth is a stock step (ground to the lighter panel stock) and nothing else. No shadows, no blur, no glow, and the native header's bottom shadow is turned off, large title included. The only translucent material on screen is the system's own tab bar and back button.

### Named Rules
**The Flat Print Rule.** A surface is either ground or stock. If something needs more emphasis, it gets more ink (a heavier rule, a band, larger type), never a shadow.

## Shapes

Two corners only: a soft 6pt corner on panels, bands that sit alone, buttons, rep choices and fields, and a near-square 1pt corner on lift chips, the short lift bars and the tone mark. Panels clip their contents, so a band at the top of a panel takes the panel's corner and stays square at its bottom. Rules carry the form: a hairline (two device hairlines) between rows and columns, 3pt solid ink under headings, over totals, under the career table's heads and under the masthead. Outlines are 2pt: the secondary button in ink, a rep choice in rule or line, a refused field in danger. Chart points are circles: filled for a logged PR set, a 2pt ring on stock for a past estimate. The plate mark is drawn from plain rectangles.

## Components

### Buttons
Solid, square-shouldered, printed in the ink colour.
- **Shape:** soft corner (6pt), 54pt tall, full width of its container.
- **Primary:** navy ink fill with stock-coloured label in light; chipboard ink fill with night navy label in dark. Label in uppercase condensed 800 at 22pt; an optional 18pt bold SF Symbol leads it.
- **Secondary:** the second of two choices ("Keep 205 lb", "Download backup"): transparent fill, a 2pt ink outline, the label and symbol in ink.
- **Pressed:** opacity 0.8. No scale, no colour change.
- **Disabled:** opacity 0.35, until the input is valid or something has changed ("Lower to …", "Accept training maxes", "Save settings", "Connect").
- **Text actions:** SF 600 at 16pt, centred, 44pt minimum height, an optional 16pt semibold SF Symbol before the label: ink for an alternative ("Skip it and start the next block", "Sign out"), soft ink for Cancel, danger for a destructive one (Discard, which confirms through a native destructive alert). Skip and Sign out confirm through native alerts too. Undo is the exception: condensed 800 uppercase in ink with an SF Symbol, 44pt minimum height.
- **Swipe Delete:** a past estimate swipes left to a 96pt danger action with a `trash` symbol and "Delete" in SF 600 at 13pt, both in the ground colour; VoiceOver gets the same action.

### Chips
- **Lift Chip:** a 10pt square of the lift's band ink (1pt corner) before a lift name in a list.
- **PR Target:** the PR set's rep target printed reversed out of the lift's band, at least 40pt wide, 7pt side padding. In a session opened from history the same chip carries the reps the PR set made.
- **Tone Mark:** which way a training max goes: `arrow.up`, `equal` or `arrow.down` reversed in the ground colour out of a square ink stamp (1pt corner), 22pt beside a headline or 18pt beside a verdict line. It grows with the text beside it up to 2x and centres on that text's first line. It is never the only signal: the server's words sit beside it.

### Cards / Containers
- **Corner Style:** soft (6pt).
- **Background:** stock on the ground.
- **Shadow Strategy:** none (see Elevation & Depth).
- **Border:** none; internal hairline rules divide content.
- **Internal Padding:** 14 to 16pt; a chart panel uses 12pt.
- **Held:** while a training max decision waits, the next-session card replaces its start button with a `lock.fill` in soft ink and "Settle your training maxes above to start." in soft Body Name.
- **Empty:** an empty panel or table says why in an Intro (the chart on stock) or a soft Body Name row ("No sessions yet.").

### Inputs / Fields
- **Field:** a field printed on the card: stock on the ground, ground on a stock panel. Hairline rule-coloured border, 6pt corner, 52pt tall, 12pt side padding. Numbers in condensed 800 at 26pt tabular, capped at 1.3x; text such as an address in SF at 17pt, scaling fully. Placeholder in soft ink, the caret and selection in ink. A Label sits above it; a unit beside it is condensed 600 at 17pt in soft ink, lowercase. Numeric keyboard for numbers.
- **Refused:** a field the server refused takes a 2pt danger outline, beside the error Notice that says why.
- **Notice:** a line of news under or over a form, SF 600 at 15pt with an 18pt symbol before it: an error in danger with `exclamationmark.triangle.fill`, announced as an alert; a confirmation in ink with `checkmark.circle.fill` ("Saved. The next session uses these settings."). A Save button that has saved gives way to its Notice until something changes again.
- **Native controls:** the SwiftUI DatePicker, Stepper (rest in 15s steps, beside an m:ss clock in condensed 800) and segmented Picker (the BBB mode) are seeded with the ink colour and the current appearance. The Switch's on track takes the button colour and its knob the on-button colour; off, the track is the rule colour.
- **Training max edits:** in Settings each training max is its own decision, edited one at a time through a native `Alert.prompt` with a decimal pad. The row shows the value in condensed 800 at 24pt with a soft unit and a `square.and.pencil` symbol.
- **Choices:** a single-choice list marks its selection with a trailing 20pt bold ink `checkmark`. A checklist leads each row with a 26pt `checkmark.circle.fill` in ink when ticked and a `circle` in soft ink when not; a ticked row adds a small reps field when the cycle asks for one.
- **Rep Picker:** a five-column grid of 52pt cells, 8pt gaps, ground fill, 2pt border. Cells that meet the target take the lift's line colour for border and number; others take the rule and ink. Pressed fills with the band and switches the number to on-band. One tap logs; there is no keyboard.

### Navigation
- **Tab bar:** native tabs with SF Symbols, tinted ink.
- **Tab root:** each tab is its own stack with the native large title in condensed 900 and the inline title in condensed 800, both ink, with no header shadow. The header background stays unset there, because a header background hides the large title on iOS 26; the page ground shows through.
- **Stack header:** ground-coloured, no shadow, ink tint, title in condensed 800 at 20pt; the back title names the screen that opened it ("Today", "History", "Back" in setup).
- **Rows that open something:** Recent rows, career cells, training max columns and History's column heads are tappable; rows and heads carry an 11pt bold `chevron.right` in soft ink.
- **Sheets:** native form sheet with grabber, ground background, a 6pt lift bar across the top, a 30pt black uppercase title.

### Lift Band (signature)
The card's header in the lift's band ink. **Full:** lift name at 46pt black uppercase with an SF subtitle (the cycle and round). **Slim:** 20pt heavy uppercase text on both ends; a TM test result reads "<LIFT> TM TEST" on the left and "tested × reps" on the right. On the session screen the slim band also carries the set table's column heads, printed reversed in on-band at 85% opacity, and takes its colour from the session's lift.

### Ink Band
The band for content about every lift: the block review. The button colours as a band (navy fill and stock text in light, chipboard ink fill and night navy text in dark), a 30pt black uppercase title over an SF 600 15pt intro at 90% opacity, 14pt by 16pt padding.

### TM Test Result
A stock card headed by the slim lift band. A tone mark beside the server's headline in Body Name, then the detail in Body Sub, indented to the headline. A failure adds a hairline and two columns, Now (Label over a 32pt stat value) and New training max (Label over a field on the panel, the labels on one line), then "Lower to … lb" as the primary button and "Keep … lb" as the secondary.

### Block Review
A stock card headed by the ink band, then a Lift / Now / New table: a chip and name, the current value in soft condensed 800 at 24pt, and an editable field on the panel (48pt tall, 24pt). Under each row a verdict line (18pt tone mark and the server's verdict in Body Notice) and an evidence line in Body Sub. "Accept training maxes" stays disabled until every field holds a positive value.

### Stat Row
Equal columns divided by hairlines, each a Label over a 32pt value with a soft unit. The block, round and TM line on Today; training max and best estimate on a lift.

### Section Heading
26pt black uppercase over a 3pt ink rule, with an optional trailing action (an SF Symbol button such as `plus.circle.fill`).

### Set Table and Current Set
In a live session, done sets drop to soft ink with a checkmark beside their reps. Opened from history, every set reads in full ink, the PR set's reps are reversed out of the lift's band, a "PR set · target N+" note sits under it, and a "PR set" totals line closes the table. The current set opens out of the table on a stock panel: 88pt weight, reps, an SF meta line, a hairline, then "Per side" plates in condensed 800 at 26pt, then the Log button or the rep picker. A "Logged" line with Undo sits between the rest bar and the current set. Plates short is marked in every row by a 13pt danger triangle before the weight, and in the current set by an error Notice under the plates.

### Career Table (signature)
History as the back of a card: one line per round, newest first. The left column carries the cycle as a Label and the round number in condensed 800; four lift columns follow, divided by hairlines, each cell the server's top set in condensed 800 at 20pt over its date, right-aligned. A pressed cell takes the stock colour. The column heads are pinned: each lift's 5pt band bar, its short name as a Label and a chevron, over a 3pt ink rule. Above a 1.5x font scale each round becomes a short list (chip, name, top set over date) and the heads become wrapping lift links in Body Name.

### Setting Row
A ruled settings line: the label in Body Name and an optional sub on the left, the control on the right (a field with its unit, a Stepper with its clock, a Switch, or a server value in Body Sub). Above a 1.5x font scale the label sits over the control. Settings closes with a centred footer in Body Sub, "Not affiliated with Jim Wendler."

### Rest Bar
Drawn to scale: the track (6pt, rule colour) is the full rest interval the server sent, and the fill in the lift's line colour drains in real seconds. "Rest" label at 18pt heavy, the clock at 30pt black.

### Totals Line
A 3pt ink rule over a row with an 18pt heavy uppercase label and a 30pt black value: the bottom line under a table, like career totals under the seasons.

### Lift Chart
The server's path drawn at 3pt in the lift's line colour with round joins, three rule-coloured gridlines, axis values in soft ink, point labels in condensed 800 placed above a point unless the line arrives from above. A legend in SF sub text names filled points and rings, only for the series present.

### Plate Mark and Masthead
The app's mark: a bar sleeve loaded with four plates, one in each lift's band ink, in the order of competition plates (deadlift red, bench blue, press yellow, squat green), stepping down in height; shaft, collar and sleeve in ink. The icon is the same mark at 1024 on the chipboard ground in light and the night navy ground in dark. On the connect screen the masthead is the mark at 76pt, the name at 64pt black uppercase and an SF 600 17pt tagline in soft ink, over a 3pt ink rule. The name it sets is Kedge, from `mobile/src/lib/brand.ts`. The icon is built for iOS 26 by the Icon Composer bundle `mobile/assets/kedge.icon`: the ground as its fill, then one layer for the bar in ink and one per plate in its lift's band ink, each with a light and a dark value, and no shadow, highlight or translucency of its own, so the system's tinted and clear versions come from the same shapes. `mobile/assets/icon/` holds the 1024 renders in light and dark. The splash screen is the mark alone at 76pt, centred on the ground: chipboard in light and night navy in dark, with the bar in ink and the plates in their band inks. Its images are drawn in `mobile/assets/splash/` on a transparent ground, and the splash plugin's background colour supplies the ground.

## Do's and Don'ts

### Do:
- **Do** print every number in Sofia Sans Extra Condensed with tabular numerals and cap its Dynamic Type growth at 1.3x, except numbers on a line stacked above a 1.5x font scale.
- **Do** take lift colour from the lift's ink set (`band`, `onBand`, `line`) and use it everywhere that lift appears.
- **Do** head content about every lift with the ink band, and content about one lift with that lift's band.
- **Do** print a verdict as the ink tone mark beside the server's own words.
- **Do** mark danger with a word, the `exclamationmark.triangle.fill` symbol or a 2pt outline beside its message.
- **Do** put a 3pt ink rule under every section heading and over every totals line, and hairline rules between rows.
- **Do** separate surfaces only by the stock step: ground for the page, stock for panels.
- **Do** keep native iOS chrome native: tabs, stack headers, large titles, form sheets, alerts and prompts, swipe actions, date pickers, steppers, segmented pickers, switches and SF Symbols.
- **Do** tint native controls from the inks: seed SwiftUI controls with the ink colour and the appearance; give a Switch the button and on-button colours.
- **Do** stack a row's label over its numbers or control above a 1.5x font scale instead of shrinking or truncating.

### Don't:
- **Don't** use a lift's spot ink as a generic accent, link, selection, verdict or status colour.
- **Don't** colour a verdict green or red.
- **Don't** add shadows, blur, gradients or glows to content; emphasis is ink and type.
- **Don't** wrap individual stats in their own cards; they belong in ruled rows and stat lines.
- **Don't** set a header background on a large-title tab root; it hides the large title on iOS 26.
- **Don't** add corner radii beyond the 6pt panel corner and the 1pt chip corner.
- **Don't** use a keyboard where a bounded set of answers can be tapped.
