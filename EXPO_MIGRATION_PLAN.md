# Native iOS app (Expo) for the 5/3/1 tracker, on the existing Go backend, in one repo

## Context

The 5/3/1 tracker is a Go + templ web app with hand-written CSS, used as a phone PWA over Tailscale. The user wants it to look cleaner, borrowing the tools and look of the football-survivor app: Tailwind-style design tokens, Geist, lucide icons, and design review before code.

After discussion, the target is a **native iOS app built with Expo, published to the App Store**, talking to the **self-hosted Go server**. The Go server keeps its engine, SQLite database and tests.

Decisions made with the user:
- **Stack:** Expo app on the Go backend, in **one repo**.
- **Layouts:** the Go side and the Expo/TypeScript side each follow their community's usual layout and code style.
- **Look:** chosen in Phase 3 from three directions built as real screens: "Career Line", recorded in `DESIGN.md`. It replaces the first idea of dark only with an amber accent. Screens are designed and approved before they're wired up.
- **Releases:** EAS for builds and over-the-air updates (self-hosting updates may come later).
- **App Store:** a public demo server for Apple's review; an original app name, with 5/3/1 in the description and a not-affiliated note.
- **Audience:** mostly the user, so no extra hardening for other self-hosters yet.

**Requirement:** the same functionality. All training logic stays in Go and the app only displays what the server computes. The database file, schema and migrations don't change.

## Repository layout

```
531-tracker/
├── README.md                 overview of both halves, links into server/ and mobile/
├── AGENTS.md, CLAUDE.md      repo map and conventions (CLAUDE.md = "@AGENTS.md", as in survivor)
├── PRODUCT.md, DESIGN.md     product context and the design system, read by the Impeccable plugin
├── .impeccable/              design.json (the system's tokens) and surfaces/ (the direction each surface
│                             was reviewed against); captures and runtime state are ignored
├── Makefile                  runs both halves: generate, check, run, app
├── .claude/settings.json     Expo and Impeccable plugins, enabled for this project
├── .github/workflows/        ci.yml (server and mobile jobs, path-filtered), release.yml (server image)
├── docs/                     design overview screenshots, App Store listing text and review notes, privacy policy
│
├── server/                   Go module: github.com/kbremont/531-tracker/server
│   ├── go.mod, go.sum        tool directives: templ, sqlc, oapi-codegen, staticcheck, oasdiff
│   ├── README.md             the program, running, config, backups, Pi deploy (moved from today's README)
│   ├── Dockerfile, .dockerignore, docker-compose.yml, .env.example
│   ├── sqlc.yaml
│   ├── api/openapi.yaml      the API's spec, written by hand: the contract between the halves
│   ├── cmd/server/main.go    entry point, plus -healthcheck and -seed-demo flags
│   ├── internal/
│   │   ├── program/          pure 5/3/1 engine (unchanged)
│   │   ├── db/               migrations/, query.sql, sqlc output, Open, Migrate (unchanged)
│   │   ├── training/         NEW: the rules applied to the database: state, sessions, TM review, onboarding,
│   │   │                     estimates, chart, settings, history, backup; read models
│   │   ├── auth/             NEW: session tokens (HMAC), password check, cookie or Bearer middleware
│   │   ├── web/              HTML: thin handlers + templ templates + format helpers + embedded static/
│   │   ├── api/              NEW: JSON v1: oapi.gen.go (generated from the spec), handlers, error mapping,
│   │   │                     testdata/*.json
│   │   ├── demo/             NEW: seeds a realistic demo database
│   │   └── server/           assembles routes, middleware, /healthz, panic recovery
│   └── tools/genicons/
│
└── mobile/                   Expo app (create-expo-app default template, SDK 58)
    ├── package.json, package-lock.json, .nvmrc (24), tsconfig.json (strict, "@/*" → src/*)
    ├── app.config.ts, eas.json, eslint.config.js
    ├── AGENTS.md             from the template: points agents at the SDK-matched docs
    ├── assets/               kedge.icon (the app icon, Icon Composer), icon/ (its 1024 renders), splash/ (the
    │                         splash screen's mark), fonts/ (Sofia Sans Extra Condensed, with its OFL licence)
    ├── scripts/              shoot.mjs (simulator screenshots of every gallery state), contact-sheet.swift
    └── src/
        ├── app/              Expo Router screens
        │   ├── _layout.tsx   root stack, query client, sign-in gate
        │   ├── connect.tsx
        │   ├── (tabs)/       _layout.tsx (NativeTabs); (today)/, (history)/ and (settings)/, each a
        │   │                 stack for its large title
        │   ├── workout/[id].tsx
        │   ├── lift/[slug]/  index.tsx, add-estimate.tsx (native sheet)
        │   ├── setup/        index.tsx, position.tsx, history.tsx
        │   └── gallery/      dev-only: every screen and state, by ?state=
        ├── theme/            the design system's tokens: both palettes, the lift inks, fonts, the
        │                     Dynamic Type cap
        ├── components/       card-back.tsx (page, panel, lift band, stat line, headings, table head,
        │                     rest bar, totals, buttons, fields, notices, tone mark), lift-chart.tsx,
        │                     plate-mark.tsx (the app's mark), tab-stack.tsx
        ├── api/              client.ts (openapi-fetch, Bearer, errors), queries.ts (openapi-react-query
        │                     hooks), schema.gen.ts (generated), fixtures/ (generated)
        ├── sample/           dev: the -seed-demo block fetched through the API, and use-screen.ts, each
        │                     screen's data hook until Phase 4's queries replace it
        ├── gallery/          dev: the gallery's states and the store that puts their data in place
        ├── dev/              remote-step.ts: lets the screenshot script move between screens
        └── lib/              format.ts (the server's formatting rules, the rest clock), brand.ts (the
                              app's name), connection.ts (server URL and token in the keychain)
```

### One repo, not separate repos

Home Assistant splits its server, frontend, iOS and Android into separate repos because it has:
- thousands of contributors on separate teams and languages
- apps that mostly show its web frontend and rarely change at the same time as the server
- a public API used by many clients it doesn't control

None of that applies here. There's one developer, one app that you control, and an API shaped to that app's screens, so almost every feature touches both sides.

Immich, a self-hosted app with App Store apps, keeps `server/`, `web/` and `mobile/` in one repo with a generated API contract. This plan follows that pattern.

The plan keeps what matters from the split approach:
- a versioned API, plus the `info` check for an app and server on different versions
- separate release pipelines, using path filters
- a defined contract, made of generated types and sample responses

Split `mobile/` out later with `git subtree split` if other people start contributing to one side, the API gets other clients, or the two sides need different visibility.

### Why this shape

- **Each half stands on its own.**
  - `cd server && go test ./...` never walks into `node_modules`.
  - Metro, Jest and EAS only see `mobile/`.
  - `eas.json` lives in the app directory, as Expo's monorepo guide says.
  - There's only one JavaScript package, so no npm workspaces and no Metro `watchFolders` config.
- **Go conventions:**
  - The module sits in a subdirectory, with a path to match.
  - `cmd/` and `internal/`, packages named for what they provide without stutter (`training.Service`, `auth.Sessions`, `api.Handler`).
  - `context.Context` first, errors wrapped with `%w`, typed or sentinel errors that the web and API layers translate.
  - Table-driven tests, `testdata/` golden files with an `-update` flag.
  - Dev tools pinned with go.mod `tool` directives and run as `go tool …`, so nothing gets installed globally.
  - `staticcheck` added to the checks.
- **Expo/TypeScript conventions:**
  - The default template's `src/app` routes, with `components/`, `api/` and `lib/` beside them.
  - Strict TypeScript, the `@/` path alias, `npx expo lint` (eslint-config-expo), Prettier.
  - `jest-expo` + React Native Testing Library, with tests next to the code they cover.
  - Dependencies added with `npx expo install` so versions match the SDK.
- **Contract between the halves:**
  - The OpenAPI 3.0 spec in `server/api/openapi.yaml`, written by hand, is the source of truth: routes, parameters, request bodies, and every answer with its status.
  - `make generate` runs templ, sqlc, **oapi-codegen** (spec → the server's wire types and a typed method per operation in `internal/api/oapi.gen.go`) and **openapi-typescript** (spec → `mobile/src/api/schema.gen.ts`), then copies `server/internal/api/testdata/*.json` into `mobile/src/api/fixtures/`.
  - The server checks every request against the spec, and the Go tests check every answer against it, golden responses included.
  - Generated files are committed and CI checks they're current, the same as templ and sqlc output. CI also runs oasdiff against the base branch and fails on a breaking change unless `info.version` changes with it.
  - Wire types are kept separate from the internal read models, so renaming a Go field can't silently break installed apps. `/api/v1` and the `info` endpoint guard against an app and server on different versions.

## Phases

Each restructure step must leave `make check` green, and tests may only change by import path or the package they moved to. I'd suggest three PRs (restructure, API, app), opened only when the user asks.

### 0. Setup ✅ Complete

Everything below is done, including the user steps. fnm switches to Node 24 in `mobile/`, and the Expo MCP server is connected.

- **Agent steps:**
  - Run `npx create-expo-app@latest mobile` (SDK 58), then `npx expo install expo-dev-client`.
  - Add the `expo-mcp` dev dependency.
  - Install the Expo plugin with `claude plugin install expo@claude-plugins-official --scope project`.
  - Pin Node 24 in `mobile/.nvmrc`.
- **User steps:**
  - Create a free Expo account and sign in with `/mcp` in an interactive Claude Code session.
  - Enroll in the Apple Developer Program early.

Already installed: Xcode 27.1, an iOS 26.4 simulator, CocoaPods.

### 1. Restructure the Go server (no behavior change) ✅ Complete

Done in five commits, one per step, each with `make check` green. A black-box transcript of about 2,200 requests across every route matched the old server byte for byte, apart from one space of whitespace in the onboarding page's markup. Where the result differs from the plan below:

- `estimates_test` stays in `internal/server` next to `flow_test`, because it drives the HTTP routes. `state_test` and `chart_test` moved to `training`. The rep picker and rep target tests moved to `training` too, now that those are methods on `training.Set`.
- The flow tests' harness reads through `training.Service` and the database instead of the server's private fields. The assertions are unchanged.
- Templates render `training`'s read models directly. Page-only state (`?rest=1`, `?saved=1`, `?error=1`, setup form errors) lives in small `web` wrappers that embed them. Form field names stay in `web`.
- The image build no longer runs `go mod download`, which would also fetch the tools' dependencies (over 1 GB). `go build` fetches only what the binary imports, using BuildKit cache mounts.

1. **Move the code.** `git mv` `cmd/ internal/ tools/ go.mod go.sum sqlc.yaml Dockerfile .dockerignore docker-compose.yml .env.example` into `server/`.
   - Change the module path to `github.com/kbremont/531-tracker/server` and rewrite imports.
   - Re-run templ and sqlc and commit the output.
   - Point CI/release at `server/`: working directory, Docker `context: server`, path filters. The image stays `ghcr.io/kbremont/531-tracker`.
   - Split the README: server details go to `server/README.md`.
2. **Tools.** Add `tool` directives for templ, sqlc, tygo and staticcheck (sqlc stays a `go install` if its cgo dependencies cause trouble). The `Makefile` and CI use `go tool …`. Add `staticcheck` to `make check` and CI.
3. **Extract `internal/auth`** from `server/auth.go`: the same `s531` cookie, the same `exp.hmac` token and secret derivation, and middleware that also accepts `Authorization: Bearer`. API requests get a JSON 401; pages still redirect to `/login`.
4. **Extract `internal/training`** from `internal/server/{state,session,review,onboarding,estimates,chart,handlers,backup}.go`.
   - It holds `training.New(conn, log)` and methods like `Dashboard`, `StartWorkout`, `LogSet`, `UndoSet`, `FinishWorkout`, `AbandonWorkout`, `Workout`, `History`, `Lift`, `AddPastEstimate`, `DeletePastEstimate`, `Settings`, `SaveSettings`, `SetTrainingMax`, `AcceptReview`, `DecideTMTest`, `SkipCycle`, the `Setup*` methods, and `Backup`.
   - It returns read models and errors such as `ErrNotFound`, `ErrHeld` and `*InputError`.
   - Move the logic now inside templates (`asksReps`, `repChoices`, `isTargetRep`, `RepTarget`, the plate line) into the read models.
   - `state_test`, `chart_test` and `estimates_test` move with the code.
5. **Rename `internal/views` → `internal/web`.** The HTML handlers become thin: parse the form, call `training`, redirect or render exactly as today.
   - `internal/assets/static` moves to `internal/web/static`.
   - `pages_test` and `workout_test` move along.
   - `internal/server` keeps only route wiring, `/healthz` and panic recovery, plus `flow_test.go` driving the whole HTTP stack.

### 2. JSON API v1 (`internal/api`) ✅ Complete

Done in six commits, each with `make check` green. `flow_test`, `pages_test` and `workout_test` are unchanged. A smoke test against the demo database covered info, sign-in, a session (log, undo, finish), a rejected estimate, a backup that opens in `sqlite3`, and the web UI through a cookie. On every lift page, the web and the API showed the same numbers. Where the result differs from the plan below:

- **Small changes in `training`:**
  - `training.Service` has a clock (`WithClock`) for `Lift.Today`, past estimate validation and the backup's filename.
  - `ChartWidth`/`ChartHeight` are exported.
  - `SetTrainingMax` returns `ErrNotFound` for an unknown lift. The web still answers it with a 500.
- **Extra TS unions:** besides tone, BBB mode, set tag and TM-test action, `phase` is a union too (`"leader" | "tm_test" | "anchor" | "deload"`). The unions are written with `//tygo:emit` on the struct and `tstype` on the field.
- **History lifts:** `history.lifts` is `{slug, name}[]` (`LiftRef`), not training maxes with a meaningless 0.
- **Cleaned-up numbers:** chart points are rounded to a tenth, as the web draws them and as the path is written. A BBB percent read back from its stored fraction is cleaned of float noise (55, not 55.00000000000001). Other numbers, such as `bestE1rm`, are sent unrounded, and the app formats them as the web's `FmtWeight` does.
- **Unknown paths:** without a token, an unknown `/api/` path gets a 401, because the session check comes first. With a token it gets a 404, and so does a known path with the wrong method.
- **Extra tests:**
  - onboarding partway into the anchor
  - adding and deleting a past estimate
  - a settings round trip covering each BBB mode
  - a failed TM test that's already settled, and a stale review key, both 409
  - `TestScreens`, which also fails on a `testdata/` file that no screen writes (`-update` removes it)
- **Demo data:** the demo also sets warmups and rest times. It runs six anchor sessions, so bench's Anchor round 2 is next. `.dockerignore` skips `data-demo/`.
- **Type checking:** `release.yml`'s generate job still checks templ and sqlc only; the PR CI job covers tygo and the fixtures. `npx tsc --noEmit` over the whole app still fails on three errors that were already in the Expo template (`*.module.css` and `global.css` declarations, and a ref type in `app-tabs.web.tsx`). `types.gen.ts` type-checks, and every fixture was checked against its type, with string unions widened, which is all JSON imports infer. Phase 3 should add a `.d.ts` for the CSS imports before the mobile CI job lands. *(No longer needed: Phase 3 removed the template files that imported CSS, and `tsc` now reports no errors.)*

- **Endpoints:**
  - `GET info` is public and returns `{apiVersion, version}`; `POST login {password}` returns `{token, expiresAt}`.
  - Dashboard, which reports `needsSetup` instead of redirecting.
  - Workouts: start, get, log set, undo, finish, abandon.
  - History; lift, plus adding and deleting past estimates.
  - Settings get and put; set training max; accept TM review; TM test decision; skip cycle.
  - Setup, setup position and setup history; backup.
- **Behavior:**
  - Requests that change data return the updated screen data.
  - Valid input behaves exactly like the web form; invalid input gets a 400 `{error}` with a message.
  - Wire types carry explicit json tags and are converted from `training` read models.
- **Tests:**
  - `api_test.go` repeats the `flow_test` scenarios through the API: TM test through to the next block, a failed test holding only the anchor, block review proposals, the 7th week deload including skipping it, setup reps.
  - Golden responses for every screen state go in `testdata/` (`go test ./internal/api -update`).
- **Demo data:** `internal/demo` plus the `-seed-demo` flag fill an empty database with a realistic block.
- **Generation:** `make generate` writes `mobile/src/api/types.gen.ts` and the fixtures. *(Since replaced: see 2a.)*

### 2a. Describe the API with OpenAPI ✅ Complete

Phase 2's contract was code-first: Go structs turned into TypeScript by tygo, which covered data shapes but not routes, methods, parameters or status codes. It's now spec-first, in six commits, each with `make check` and `go test -race` green:

1. `server/api/openapi.yaml` describes the API as it shipped, and the test harness checks every answer against it with kin-openapi. All 27 golden responses passed against the hand-written handlers before any code moved.
2. oapi-codegen (strict server) generates `internal/api/oapi.gen.go`, `types.go` is gone, and nethttp-middleware checks every request against the spec after the session check. The goldens changed only in key order (the generated structs are alphabetical); sorted, each is identical.
3. openapi-typescript writes `mobile/src/api/schema.gen.ts` in place of tygo's `types.gen.ts`. The root type names are unchanged, every fixture type-checks against them, and `tsc` still shows only the three template errors.
4. A CI job runs `oasdiff breaking` against the base branch's spec. Renaming a response property made it fail with an ERR; adding one didn't.
5. `Dashboard.next` and `review` get `type: object` beside `nullable`, which Redocly's linter requires for 3.0.3. `redocly lint` now reports no errors, only three expected warnings (no license, `getInfo` has no 4xx, and the 405 response is unused by design).
6. These docs.

The same request script ran against `main` and this branch, each on a fresh `-seed-demo` database: every success answered identically, apart from key order. The only differences were the stricter requests below and the validator's wording for a missing or malformed body. Backups still open in `sqlite3` and leave no snapshot behind, the web UI still works through a cookie, and the Docker image answers `/healthz`, `/api/v1/info` and `/api/v1/openapi.json`.

Where the result differs from what was planned, or is worth knowing:

- **Stricter requests:** a body needs `Content-Type: application/json` and the fields the spec marks required. An unknown BBB mode or TM-test action is a 400 rather than the default. A known path with the wrong method is a 405 with `Allow`, where it was a 404. A path segment that doesn't fit its parameter is still a 404.
- **`api.Handler` is now `api.Server`,** because the generated code declares a `Handler` function in the same package.
- **Generator settings:** enum constants are prefixed (`PhaseTMTest`, `ToneUp`), `BBB` joins the planned initialisms, and `skip-prune` keeps the 405 response, which no operation lists, in the embedded spec. oapi-codegen capitalizes operation ids as it embeds the spec, so `/api/v1/openapi.json` has `GetDashboard` where the file has `getDashboard`; `TestSpec` allows for exactly that.
- **TypeScript:** openapi-typescript 7 declares a TypeScript 5 peer; an npm `overrides` entry lets it use the app's TypeScript 6. `--root-types-keep-casing` keeps names such as `LiftTM` and `BBBMode`, which it would otherwise rewrite as `LiftTm`.
- **CI:** the generate job sets up Node from `mobile/.nvmrc` and runs `npm ci` before `make generate`, and also runs when `mobile/package.json` or its lockfile change (the spec is already under `server/**`). `release.yml`'s generate check now covers `oapi.gen.go`.
- **The spec is served** at `/api/v1/openapi.json`, without a session and with `Access-Control-Allow-Origin: *`, for docs viewers.

### 3. Design the screens. **Checkpoint: user approval** ✅ Complete

Every state in the table below is designed and built on the Career Line look, the app is named Kedge, its icon and splash screen are chosen and wired in, and the contact sheets are in `docs/mockups/`. The user approved the screens on 2026-10-28. The visual design bullets further down are the original plan and no longer apply.

- **Product context.** `PRODUCT.md` records what the design work has to respect:
  - the iOS app is the product and the web UI is legacy
  - the owner comes first and other self-hosters second
  - the server decides every number
  - what is still undecided, and what must not be invented
- **The look.** Three directions were built as real screens and compared in the simulator. The user chose **Career Line**, which reads training as the back of a sports card:
  - **Colour:** chipboard stock and navy ink in light, the two swapped in dark, and one spot colour per lift (press gold, deadlift brick, bench blue, squat green) everywhere that lift appears.
  - **Type:** Sofia Sans Extra Condensed sets headings and every number; SF sets body text.
  - **Lists:** set as ruled stat lines, under headings with a heavy ink rule.
  - **Records:** `DESIGN.md` and `.impeccable/design.json` describe the system as built, and the other two directions are kept outside the repo.
- **The screens:**
  - **Connect:** the app's mark and name over a heavy rule, the server address and password, and the server's own message when it refuses.
  - **Today:** a TM test result arrives as a card in its lift's band, with a verdict mark (an arrow in an ink square, beside the server's words). A failure holds a "new training max" field, "Lower to …" and "Keep …". The block review is printed under a navy band: one row per lift with its current and proposed training max (editable), the verdict and the evidence, then "Accept training maxes". While either waits, the next session's card says what it is waiting on instead of offering to start. An optional cycle offers "Skip it and start the next block", confirmed with an alert.
  - **History:** a career table. Each round is a line, newest first, and the four lifts are columns of top sets with their dates. A cell opens that session, and a column head opens the lift. The heads stay pinned while the table scrolls. At accessibility text sizes each round becomes a short list instead.
  - **A finished session** opened from history reads every set in full ink, prints the PR set's reps reversed out of the lift's colour, and closes with a "PR set" totals line.
  - **Settings:** training maxes save one at a time through a native prompt. Loading, rest (native steppers in 15-second steps), warmups (a switch) and the BBB override (a segmented control and a value) save together with one Save button, which then turns into a "Saved" line. Data, server and sign out follow.
  - **Setup:** training maxes, the cycle the lifter is on, then a checklist of that cycle's sessions, with PR set reps for the anchor and the TM test.
  - **Lift:** the empty chart says why it is empty, and past estimates swipe left to delete. The add-estimate sheet shows the server's message when it refuses an entry, and stays open.
- **Gallery and screenshots:** `/gallery` lists every state and `/gallery?state=<id>` opens one in its real route, with the native chrome. `npm run shoot` (with Metro started by `npm run start:shoot`) captures every state in light and dark into `.context/mockups/` and builds `docs/mockups/overview.png` and `overview-dark.png`.
- **Name: Kedge.** A kedge is a small anchor a ship drops ahead and hauls itself up to, the way the program moves a lifter forward. 5/3/1 goes in the subtitle. No App Store app is called Kedge. Two near things to know: KEDGE Business School (France) has an alumni app ("KEDGE Alumni") and a US mark in education, and Kedge Holding B.V. holds a live US registration for KEDGE in classes 9 and 19. Neither has been checked by a lawyer; reserve the name in App Store Connect early. Other names considered: Barline, Weigh Anchor, Anchor Line, Collar, Chalkline; "Lift Card", "Statline", "The Plus Set" (a 5/3/1 tracker), Ballast and Holdfast are taken.
- **Icon: the loaded sleeve.** A bar's sleeve loaded with four plates, one per lift's ink, in the order of competition plates (red, blue, yellow, green), on chipboard in light and navy in dark. It was chosen from four options (the others: an anchor whose stock is a loaded barbell, an anchor over the four-lift stripe, and a crop of the career table). `mobile/assets/kedge.icon` is the Icon Composer bundle `app.json` points at: the ground, the bar and one layer per plate, each with a light and a dark value, flat, with no shadow or highlight of its own; `actool` compiles it to light, dark and tinted renditions. The 1024 renders are in `mobile/assets/icon/`, and the same mark heads the connect screen.
- **Reviewed.** Impeccable's finish review passed the first three screens after three rounds of fixes, and the rest after a round of eight fixes (the TM test card's alignment, refused fields outlined in danger, the stamp and settings at accessibility sizes, History's heads at accessibility sizes, the chart legend, the capture order, and `DESIGN.md`), judged on simulator screenshots in light, dark and at an accessibility text size. `DESIGN.md` and `.impeccable/design.json` now record the stamp, the ink band, the outline button, errors in danger, the native controls' tint, stacking above 1.5x, the career table and the plate mark.

- **After approval,** three loose ends were closed, one commit each:
  - **Splash screen:** the plate mark at 76pt, centred on chipboard in light and night navy in dark (the splash plugin's `dark` option), with the bar in ink and the plates in the lift band inks. `mobile/assets/splash/` has each image as an SVG drawn from `plate-mark.tsx` and a 1024 PNG on a transparent ground, with its origin embedded. QuickLook (`qlmanage`) flattens an SVG onto white, so the PNGs come from AppKit's own SVG support. After a rebuild the simulator showed it in both appearances; until the simulator was rebooted, the launch animation still showed iOS's cached snapshot of the old blue screen, which is the system's launch-screen cache, not the build.
  - **Template leftovers:** the Expo template's tab bar, themed text and views, hint row, web badge, collapsible, external link, animated splash overlay, colour-scheme hooks, `constants/theme.ts` and `global.css` are gone, with the images only they used, the template's `icon.png` and `reset-project.js`. `npx tsc --noEmit` and `npx expo lint` now report nothing, and no CSS imports remain. The Android adaptive icon and the favicon stay, because `app.json` uses them, and now record that they are the template's placeholders, so `impeccable embed-prompt --scan mobile/assets` finds no image without its origin.
  - **PR target:** its corner was 3pt, outside `DESIGN.md`'s two corners; it now takes the 1pt chip corner (`radius.chip`), checked in the simulator on a completed session.

The name, icon and splash screen are in `app.json`; the slug, scheme and bundle id follow in Phase 5.

Where the result differs from the plan below:

- **Visual design:** Career Line replaces the survivor palette, amber, Geist, lucide icons, eyebrow lines and `rounded-2xl` cards. Icons are SF Symbols, following Apple's guidelines. Verdicts (up, hold, down) are an arrow in an ink square beside the server's words, not green, grey and red: in Career Line colour belongs to the lifts.
- **No NativeWind:** the tokens are a TypeScript module, components use `StyleSheet`, and the palette follows the system's light or dark setting.
- **Sample data:** until Phase 4, the screens read `src/sample`, which is the `-seed-demo` block fetched through the API, for all four lifts, with BBB overrides for deadlift and squat saved in settings. Logging, undo, discard, TM decisions, the review, deleting estimates and the add-estimate sheet only change that sample. Each screen reads its data through a hook in `src/sample/use-screen.ts`, which Phase 4 swaps for the API's queries. `src/api/fixtures` stays the source for states the demo block doesn't reach.
- **API change:** the session (`Workout`) and a TM test result (`TMTestResult`) gained `liftSlug`, so the session's band and the result card take their lift's colour; the session screen had bench hard-coded. It's an additive change, so `oasdiff` passes without a version bump.
- **Tabs:** History and Settings live in `(history)/history.tsx` and `(settings)/settings.tsx`, each with its own stack, because the native large title needs one.
- **Screenshots:** opening the app's scheme with `xcrun simctl openurl` brings up an iOS "Open in…?" prompt, so `shoot.mjs` drives a dev-only stepper instead: `src/dev/remote-step.ts`, active only when Metro runs with `EXPO_PUBLIC_REMOTE_STEP=1`, polls the script's local server for the path to open. The script also hides the dev client's floating button and fixes the status bar at 9:41.
- **Phase 4 work done early:** the session screen already keeps the current set in view and counts the rest timer against the clock, as Phase 4 asks.
- **iOS 26 header:** a header background colour hides the native large title (react-native-screens), so the tab roots' headers have none and the screen's background shows through.
- **New dependencies:** `react-native-svg` for the chart and the mark. The native date picker, steppers and segmented control are `@expo/ui`'s SwiftUI views; swipe-to-delete uses `react-native-gesture-handler`, already in the template.

- **Build:** real Expo screens fed from the fixtures, collected in the dev-only `gallery` route (survivor's `?state=` equivalent).
- **Screenshots:** `mobile/scripts/shoot.mjs` boots the simulator, opens each state by deep link (`xcrun simctl openurl`) and screenshots it (`simctl io … screenshot`). Output goes to `.context/mockups/` and `docs/mockups/overview.png`. Agents can also tap through with the Expo MCP local tools.
- **Visual design:**
  - **Colors:** survivor's dark palette (`#0b0b0c` background, `#161618` cards, `#1c1c1f` raised, `#232327` muted, zinc text greys, white-alpha borders). Amber `#f5a524` is the primary color, with `#1a1206` text on it.
  - **Verdicts:** up green, down red, hold grey, always with an icon and a label.
  - **Type and layout:** Geist, 34px titles with an eyebrow line, `rounded-2xl` cards, grouped rows with lucide icons.
  - **Navigation:** a native tab bar (Today, History, Settings), pushed screens with a back button, a native sheet for adding a past estimate.
  - **Controls:** a segmented control for BBB mode, a switch for warmups, the native date picker, 54px buttons.
- **Screens and states:**

  | Screen | States | Gallery states |
  |---|---|---|
  | connect | default; error | `connect`, `connect-error` |
  | today | next up, in progress, TM test passed, TM test failed, block review, skip deload, blocked, empty | `today`, `today-next`, `today-in-progress`, `today-tm-test-passed`, `today-tm-test-failed` (blocked shows here and under the review), `today-block-review`, `today-block-review-end`, `today-skip-deload`, `today-empty` |
  | workout | current set, rest timer, AMRAP picker, 3–5 range picker, plates short, finish, completed | `workout` (current set and rest timer), `workout-amrap`, `workout-amrap-press`, `workout-range`, `workout-plates-short`, `workout-finish`, `workout-discard`, `workout-completed` |
  | history | list; empty | `history`, `history-scrolled`, `history-empty` |
  | lift | line, single point, empty, add sheet, error | `lift`, `lift-end`, `lift-deadlift`, `lift-single`, `lift-empty`, `lift-add`, `lift-error` |
  | settings | default; saved | `settings`, `settings-rest`, `settings-supplemental`, `settings-end`, `settings-saved` |
  | setup | TMs, position, leader history, anchor history | `setup`, `setup-error`, `setup-position`, `setup-leader`, `setup-anchor` |

- **Also decided here:** the app name and the 1024px icon.
- **Stop:** show the contact sheet and iterate until the user approves.

### 4. Wire up the live API

- **Client:** `api/client.ts` is `openapi-fetch` over `schema.gen.ts`'s `paths`, so every path, parameter, body and response is checked at compile time, with a middleware that adds the Bearer token. A 401 sends the user to sign in; an old server shows an "update your server" message (checked via `info`).
- **Connect:** normalize the URL, check `info`, sign in, and keep the URL and token in `expo-secure-store`.
- **Data:** TanStack Query hooks from `openapi-react-query` over the same client, with returned screen data written straight into the cache, and a refetch when the app comes to the foreground.
- **Rest timer:**
  - starts when a set is logged, for the `restSeconds` the server returns
  - counts down against the clock, so it stays right if the phone sleeps
  - sits above the current set, and the list scrolls to it
- **Backup:** download with the token, then hand the file to the iOS share sheet.
- **iOS settings in `app.config.ts`:** `NSAllowsLocalNetworking` plus the local-network permission text, HTTPS everywhere else, and `ITSAppUsesNonExemptEncryption: false`.
- **Component tests:** carry over the intent of `pages_test` and `workout_test`: rest timer position, the rep picker's choices and highlighted targets, TM result cards and the decision form, skip offered only when allowed, chart edge cases, and the estimate error reopening the sheet.

### 5. EAS and TestFlight

- **Agent steps:** `mobile/eas.json` with `development` (simulator), `preview` and `production` profiles; `eas update:configure` with the runtime version set by `fingerprint`; bundle id `com.kbremont.kedge`, with the slug and scheme renamed to match.
- **User steps** (Apple two-factor sign-in): `eas login`, then `eas build -p ios --profile production`, then `eas submit -p ios`.
- **Updates:** `eas update --channel production` for JS-only changes; a rebuild for native changes or every 90 days.

### 6. Demo server and App Store

- **Demo server:** the same image with its own volume, seeded with `-seed-demo` and a demo password. It's public through Tailscale Funnel on a separate port, or a small VPS; the real server stays tailnet-only.
- **Listing:**
  - original name; 5/3/1 in the subtitle and description, with "not affiliated with Jim Wendler"
  - Health & Fitness category
  - privacy label "Data Not Collected" (check that nothing else in the app sends data)
  - a privacy policy and support page
  - screenshots at Apple's required sizes, taken from the simulator against demo data
  - review notes with the demo URL and password
  - the listing text kept in `docs/app-store/`
- **Order:** TestFlight first, then App Review.

### 7. CI and docs

- **`ci.yml`:**
  - **server job** (on `server/**` changes): gofmt, vet, staticcheck, `go test -race`, and the generate-then-`git diff` check, which sets up Node and runs `npm ci` in `mobile/` so it also covers `oapi.gen.go`, `schema.gen.ts` and the fixtures.
  - **breaking job:** `oasdiff breaking` against the base branch's spec, failing unless `info.version` changes with a break.
  - **mobile job** (on `mobile/**` changes): `npm ci`, `tsc --noEmit`, `expo lint`, `jest`, `npx expo-doctor`, `npx expo export --platform ios`.
  - **image job:** the existing smoke test, with the server as build context.
- **`release.yml`:** builds the server image only, filtered to `server/**`.
- **Docs:** root `AGENTS.md`/`CLAUDE.md` with the repo map and conventions: logic lives in Go and the app only displays; API changes run `make generate`; the UI matches the approved gallery. The root README is an overview, with details in `server/README.md` and `mobile/README.md`.

## Verification

1. **Restructure:** after each step in Phase 1, `make check` passes (gofmt, vet, staticcheck, every existing Go test). `docker build server` works and the image's smoke test hits `/healthz`. The web UI at `make run` behaves identically: click through a session, settings, and a lift page.
2. **API:** the API tests and golden files pass, and `make generate` leaves no diff.
3. **App checks:** in `mobile/`, `tsc`, lint, `jest`, `expo-doctor` and `expo export` pass.
4. **End to end in the simulator** against `make run`, screenshotted to `.context/`:
   - connect with a wrong password, then the right one
   - onboarding mid-programme
   - a full session with rest timer, AMRAP, undo and finish; separately, discard
   - a TM test failure, both lowering and keeping
   - accepting the block review; skipping the deload
   - settings, including each BBB mode
   - past estimates with good and bad input, and a delete
   - sharing a backup
5. **Parity:** point the server at a copy of the user's real database (a backup from the Pi). The app and the web UI show the same numbers on Today, History, each lift page and old workouts.
6. **Ship:** the TestFlight build works on the phone over Tailscale; an `eas update` arrives without a rebuild; the demo server is reachable with Tailscale off.
