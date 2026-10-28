# Product

<!-- impeccable:product-schema 1 -->

## Platform

ios

The native iOS app in `mobile/` (Expo, SDK 58) is the product that design work serves. The templ web UI in `server/internal/web` is legacy: it keeps working as a PWA, but it gets no further design work.

## Users

**Primary: the owner.** An experienced 5/3/1 lifter who runs the server on their own hardware (a Raspberry Pi, reached over Tailscale) and uses the app on their phone at the gym. Their main job is getting through a session: knowing what to lift next, loading the bar, logging each set, and resting. Away from the bar they review history and lift charts, change settings, and settle training max decisions.

**Secondary: other self-hosters.** 5/3/1 lifters who install the app from the App Store and run their own server. They come second, but they should be able to connect, set up and train without help.

**Incidental: Apple App Review**, which signs in to a public demo server seeded with a demo lifter's block.

## Product Purpose

A self-hosted tracker for Jim Wendler's 5/3/1. It tells the lifter which session is next, prescribes every set's weight, reps and plates, logs what they actually lifted, times their rest, charts their estimated maxes, and proposes training max changes at the points the program calls for them.

Success means the lifter never does 5/3/1 arithmetic or keeps a notebook, logging a session at the gym takes as little attention as possible, and every training max change is one they chose. The app shows the same numbers as the server's web UI on every screen.

## Positioning

- **Self-hosted and private.** The lifter's whole history is one SQLite file on their own server. There is no account and no third-party service, and the App Store privacy label is "Data Not Collected".
- **Training maxes only move when the lifter says so.** The app proposes changes after each TM test and at the start of each block, shows the evidence, lets every value be edited, and writes nothing until the lifter accepts. The block waits for that decision.
- **Rounds, not calendar weeks.** A round is four sessions, one per lift, so a missed day never breaks the schedule.
- **Start partway through.** On first run the lifter says which cycle they are on and which sessions they have done. There is no old history to key in and nothing invented to stand for it.
- **Past estimates.** Numbers from a paper notebook can go on each lift's chart without becoming sessions or history.

## Operating Context

- **At the gym**, phone in hand between sets. Each set comes down to: read the weight and plates, lift, log the reps, rest. The rest timer counts down against the clock, so it stays right if the phone sleeps.
- **Network required.** The app talks to the lifter's server over Tailscale (or their LAN), and there is no offline mode.
- **At block transitions**, the lifter makes a decision before training goes on: lowering or keeping a training max after a failed TM test, accepting the block review, or skipping the optional deload.
- **First run:** connect to a server URL with the password, enter training maxes, then choose where in the program to start.
- **Distribution:** EAS builds go out through TestFlight and the App Store, and JS-only changes ship as over-the-air updates.

## Capabilities and Constraints

**Same functionality as the web UI, and all training logic stays in Go.** The app only shows what the server computes, and every screen's data arrives fully worked out from `/api/v1`. The OpenAPI spec at `server/api/openapi.yaml` is the contract. A change answers with the next screen's data, and an error is a message the app can show as it is. The database schema doesn't change.

**The program.** Each block is eleven rounds: Leader 1 and 2 (5's PRO with BBB 5×10), a 7th Week TM Test (100% AMRAP), Anchor (5/3/1 with PR sets and FSL 5×5), and an optional 7th Week Deload. Every round is four sessions: press, deadlift, bench and squat. Programs are data, so more templates may come.

**Screens and states** (from the migration plan):

| Screen | States |
|---|---|
| connect | default; error |
| today | next up, in progress, TM test passed, TM test failed, block review, skip deload, blocked, empty |
| workout | current set, rest timer, AMRAP picker, 3–5 range picker, plates short, finish, completed |
| history | list; empty |
| lift | line, single point, empty, add sheet, error |
| settings | default; saved |
| setup | TMs, position, leader history, anchor history |

**Also:**
- Undoing a set and discarding a session.
- Optional warmups (40/50/60% × 5).
- A per-lift BBB override, as a percentage of the TM or a fixed weight.
- Separate rest times for warmups, main sets and supplemental work.
- Plate math, with a warning when the plates run short.
- Setting a training max directly at any time.
- Downloading a backup, which goes to the iOS share sheet.

**Units:** pounds only for now. The database has a unit column for later.

**Terminology:** use these terms consistently.
- **Structure:** a *session* is one lift's workout, a *round* is four sessions, and a *block* is the eleven-round sequence.
- **Cycles:** Leader 1, Leader 2, 7th Week TM Test, Anchor and 7th Week Deload.
- **Loads and sets:** *training max* (TM), *PR set*, *AMRAP*, *estimated max*, *past estimates*, BBB, FSL and 5's PRO.
- **Lifts:** press, deadlift, bench and squat.

**Undecided:**
- The app's name and its 1024px icon. Both get decided in the screen design phase.
- The visual design. See Brand Commitments.

## Brand Commitments

- An original app name, with "5/3/1" in the subtitle and description and a note that the app is not affiliated with Jim Wendler. The name isn't chosen yet, and the PWA's "5/3/1" title is not the app name.
- App Store category: Health & Fitness.
- No visual commitments. The migration plan's look notes (dark only, an amber accent, Geist, lucide icons, the football-survivor palette) were a starting idea, not decisions. The design phase decides the look.

## Evidence on Hand

- **Screen fixtures:** `mobile/src/api/fixtures/*.json` has a golden API response for every screen state the app shows, generated from the server's tests.
- **Demo data:** `make demo` runs the server against a database seeded with a realistic block. It covers both leaders, a passed TM test, part of the anchor, and past estimates.
- **The legacy web UI** at `make run` (`server/internal/web`) works as a reference for behavior and copy, not for the look.
- **Assets:** the PWA icons in `server/internal/web/static/` belong to the legacy web UI. The images in `mobile/assets/` are Expo template placeholders.
- **Not available:** no app name, app icon, App Store screenshots, testimonials, user counts or reviews. Don't invent any.

## Product Principles

1. **The server decides and the app shows.** The app never computes a weight, a rep target or a verdict. If a screen needs a number, it comes from the API.
2. **The bar comes first.** During a session, the next set and how to log it outrank everything else on screen. History, charts and settings can wait until the lifter is away from the bar.
3. **Decisions belong to the lifter.** Show what's proposed and why, keep it editable, and write nothing until the lifter confirms.
4. **Fit the lifter's real training.** Count rounds rather than weeks, start wherever they are, and never fill gaps with invented history.
5. **Their data stays theirs.** The data lives on their own server, and the app sends nothing anywhere else.
