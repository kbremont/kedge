# App Store listing

The text App Store Connect holds for Kedge (app `6829543729`), kept here so
it changes with the app. Paste each field as it is. Limits are App Store
Connect's.

## App Information

| Field | Value |
|---|---|
| Name (30) | Kedge Strength |
| Subtitle (30) | The self-hosted 5/3/1 logbook |
| Primary category | Health & Fitness |
| Secondary category | none |
| Content rights | No third-party content |
| Age rating | 4+: every answer in the questionnaire is None or No (no user-generated content shown to others, no web browsing, no ads, no purchases, no medical or treatment claims) |

"Kedge" was taken in App Store Connect, so the store name is Kedge Strength.
The name under the icon stays Kedge (`app.json`).

## Pricing and Availability

- **Price:** Free.
- **Availability:** all countries and regions.
- **EU Digital Services Act:** "not a trader". Kedge is an individual's
  free, non-commercial app. A trader's address, phone and email are shown on
  the EU storefronts.

## Version information

**Promotional text (170):**

> Your 5/3/1, worked out set by set by a server you run yourself. Try the public demo from the support page before hosting your own.

**Description (4000):**

> Kedge runs Jim Wendler's 5/3/1 for you, from a server you host yourself.
>
> It tells you which session is next and prescribes every set: the weight, the reps and the plates to load. Log what you actually lifted, and a rest timer counts down against the clock, so it stays right while your phone sleeps.
>
> BUILT FOR HOW YOU TRAIN
> • Rounds, not calendar weeks: four sessions to a round, one lift each, so a missed day never breaks the schedule.
> • Start partway through. Tell it which cycle you're on and which sessions you've done, with no old history to key in.
> • Leader and anchor cycles with 5's PRO, BBB and FSL, a 7th Week TM test and an optional deload.
> • Optional warmups, a BBB override for each lift, and separate rest times for warmups, main sets and supplemental work.
> • Plate math for every set, with a warning when your plates run short.
>
> TRAINING MAXES MOVE WHEN YOU SAY SO
> After each TM test and at the start of each block, Kedge proposes changes and shows the evidence. Every value is editable, and nothing is written until you accept.
>
> YOUR CAREER
> History is a table of every round, one column per lift. Each lift has a chart of its estimated max, with room for the numbers from your paper notebook.
>
> YOUR SERVER, YOUR DATA
> Kedge connects to a server you run: one small Docker image, with your whole history in a single SQLite file. There's no account, and nothing you log goes anywhere else. Want to try it first? The support page has a public demo server.
>
> Train in pounds or kilograms.
>
> Kedge is not affiliated with or endorsed by Jim Wendler. 5/3/1 is his training program.

**Keywords (100):**

> 531,strength,barbell,powerlifting,training max,workout log,lifting,self-hosted,BBB,AMRAP,plates

**URLs:**

| Field | URL |
|---|---|
| Support URL | https://github.com/kbremont/kedge/blob/main/docs/support.md |
| Marketing URL | https://github.com/kbremont/kedge |
| Privacy Policy URL | https://github.com/kbremont/kedge/blob/main/docs/privacy.md |

The repo has to be public before these open for App Review. Keep the
paths: the app links the privacy policy itself (`mobile/src/lib/brand.ts`).

**Copyright:** 2026 Kyle Bremont

**Version release:** manually release this version.

## Screenshots

Upload these into "iPhone with Dynamic Island (medium display)", the
6.1"/6.3" slot, at 1206×2622, from `screenshots/` in this folder, in order:

| File | Caption | Screen |
|---|---|---|
| `1-session.png` | Every set, without the math | bench's session at its first main set, rest timer running |
| `2-today.png` | Know what's next | Today, with bench's anchor session next |
| `3-history.png` | Every round, every lift | History's career table |
| `4-deadlift.png` | Watch your max climb | deadlift's chart of PR sets and past estimates |
| `5-session-dark.png` | Light or dark | the session in dark |

Each caption is set like the app's section headings, in black condensed
caps over a heavy rule. Below it, the screen sits in a flat phone edge in
the ink, on the chipboard ground (night navy for the dark one), so the
chipboard screens don't blend into the page. The screens are 1320×2868
captures from the preview build on the iPhone 17 Pro Max simulator, signed
in to a server seeded with `-demo`: Today, History and deadlift's chart on
the seed where every estimated max climbs (#22), and the two session shots
on the public demo, whose bench session is the same on either seed. Each PNG
records how it was made.

Kedge runs on iPhone only, so there's no iPad set.

## Also

- **App Privacy:** see [privacy-label.md](privacy-label.md).
- **App Review:** see [review-notes.md](review-notes.md).
- **Export compliance** is answered by the build:
  `ITSAppUsesNonExemptEncryption` is false (`app.json`).
