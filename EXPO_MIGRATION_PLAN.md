# Native iOS app (Expo) for the 5/3/1 tracker, on the existing Go backend, in one repo

## Context

The 5/3/1 tracker is a Go + templ web app with hand-written CSS, used as a phone PWA over Tailscale. The user wants it to look cleaner, borrowing the tools and look of the football-survivor app: Tailwind-style design tokens, Geist, lucide icons, and design review before code.

After discussion, the target is a **native iOS app built with Expo, published to the App Store**, talking to the **self-hosted Go server**. The Go server keeps its engine, SQLite database and tests.

Decisions made with the user:
- **Stack:** Expo app on the Go backend, in **one repo**.
- **Layouts:** the Go side and the Expo/TypeScript side each follow their community's usual layout and code style.
- **Look:** dark only, amber accent; screens designed and approved before they're wired up.
- **Releases:** EAS for builds and over-the-air updates (self-hosting updates may come later).
- **App Store:** a public demo server for Apple's review; an original app name, with 5/3/1 in the description and a not-affiliated note.
- **Audience:** mostly the user, so no extra hardening for other self-hosters yet.

**Requirement:** the same functionality. All training logic stays in Go and the app only displays what the server computes. The database file, schema and migrations don't change.

## Repository layout

```
531-tracker/
├── README.md                 overview of both halves, links into server/ and mobile/
├── AGENTS.md, CLAUDE.md      repo map and conventions (CLAUDE.md = "@AGENTS.md", as in survivor)
├── Makefile                  runs both halves: generate, check, run, app
├── .claude/settings.json     Expo plugin, enabled for this project
├── .github/workflows/        ci.yml (server and mobile jobs, path-filtered), release.yml (server image)
├── docs/                     design overview screenshots, App Store listing text and review notes, privacy policy
│
├── server/                   Go module: github.com/kbremont/531-tracker/server
│   ├── go.mod, go.sum        tool directives: templ, sqlc, tygo, staticcheck
│   ├── README.md             the program, running, config, backups, Pi deploy (moved from today's README)
│   ├── Dockerfile, .dockerignore, docker-compose.yml, .env.example
│   ├── sqlc.yaml, tygo.yaml
│   ├── cmd/server/main.go    entry point, plus -healthcheck and -seed-demo flags
│   ├── internal/
│   │   ├── program/          pure 5/3/1 engine (unchanged)
│   │   ├── db/               migrations/, query.sql, sqlc output, Open, Migrate (unchanged)
│   │   ├── training/         NEW: the rules applied to the database: state, sessions, TM review, onboarding,
│   │   │                     estimates, chart, settings, history, backup; read models
│   │   ├── auth/             NEW: session tokens (HMAC), password check, cookie or Bearer middleware
│   │   ├── web/              HTML: thin handlers + templ templates + format helpers + embedded static/
│   │   ├── api/              NEW: JSON v1: handlers, wire types (json tags), error mapping, testdata/*.json
│   │   ├── demo/             NEW: seeds a realistic demo database
│   │   └── server/           assembles routes, middleware, /healthz, panic recovery
│   └── tools/genicons/
│
└── mobile/                   Expo app (create-expo-app default template, SDK 58)
    ├── package.json, package-lock.json, .nvmrc (24), tsconfig.json (strict, "@/*" → src/*)
    ├── app.config.ts, eas.json, metro.config.js, global.css (NativeWind tokens), eslint.config.js
    ├── AGENTS.md             from the template: points agents at the SDK-matched docs
    ├── assets/               icon, splash, Geist font files
    ├── scripts/shoot.mjs     simulator screenshots and a contact sheet
    └── src/
        ├── app/              Expo Router screens
        │   ├── _layout.tsx   root stack, query client, sign-in gate
        │   ├── connect.tsx
        │   ├── (tabs)/       _layout.tsx, index.tsx (Today), history.tsx, settings.tsx
        │   ├── workout/[id].tsx
        │   ├── lift/[slug]/  index.tsx, add-estimate.tsx (native sheet)
        │   ├── setup/        index.tsx, position.tsx, history.tsx
        │   └── gallery/      dev-only: every screen and state from fixtures
        ├── components/       ui/ (Button, Card, ListRow, Segmented, Switch, Field, PageHeader)
        │                     workout/ (SetRow, RepPicker, RestTimer), today/ (NextUpCard,
        │                     TMTestResult, BlockReview), lift/ (LiftChart, PastEstimates)
        ├── api/              client.ts (fetch, Bearer, errors), queries.ts (TanStack Query hooks),
        │                     types.gen.ts (generated), fixtures/ (generated)
        └── lib/              connection.ts (server URL and token in the keychain), small helpers
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
  - Strict TypeScript, the `@/` path alias, `npx expo lint` (eslint-config-expo), Prettier with the Tailwind plugin.
  - `jest-expo` + React Native Testing Library, with tests next to the code they cover.
  - Dependencies added with `npx expo install` so versions match the SDK.
- **Contract between the halves:**
  - The Go wire types in `internal/api` are the source of truth.
  - `make generate` runs templ, sqlc, and **tygo** (Go structs → `mobile/src/api/types.gen.ts`), then copies `server/internal/api/testdata/*.json` into `mobile/src/api/fixtures/`.
  - Generated files are committed and CI checks they're current, the same as templ and sqlc output today.
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

### 1. Restructure the Go server (no behavior change)

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

### 2. JSON API v1 (`internal/api`)

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
- **Generation:** `make generate` writes `mobile/src/api/types.gen.ts` and the fixtures.

### 3. Design the screens. **Checkpoint: user approval**

- **Build:** real Expo screens fed from the fixtures, collected in the dev-only `gallery` route (survivor's `?state=` equivalent).
- **Screenshots:** `mobile/scripts/shoot.mjs` boots the simulator, opens each state by deep link (`xcrun simctl openurl`) and screenshots it (`simctl io … screenshot`). Output goes to `.context/mockups/` and `docs/mockups/overview.png`. Agents can also tap through with the Expo MCP local tools.
- **Visual design:**
  - **Colors:** survivor's dark palette (`#0b0b0c` background, `#161618` cards, `#1c1c1f` raised, `#232327` muted, zinc text greys, white-alpha borders). Amber `#f5a524` is the primary color, with `#1a1206` text on it.
  - **Verdicts:** up green, down red, hold grey, always with an icon and a label.
  - **Type and layout:** Geist, 34px titles with an eyebrow line, `rounded-2xl` cards, grouped rows with lucide icons.
  - **Navigation:** a native tab bar (Today, History, Settings), pushed screens with a back button, a native sheet for adding a past estimate.
  - **Controls:** a segmented control for BBB mode, a switch for warmups, the native date picker, 54px buttons.
- **Screens and states:**

  | Screen | States |
  |---|---|
  | connect | default; error |
  | today | next up, in progress, TM test passed, TM test failed, block review, skip deload, blocked, empty |
  | workout | current set, rest timer, AMRAP picker, 3–5 range picker, plates short, finish, completed |
  | history | list; empty |
  | lift | line, single point, empty, add sheet, error |
  | settings | default; saved |
  | setup | TMs, position, leader history, anchor history |

- **Also decided here:** the app name and the 1024px icon.
- **Stop:** show the contact sheet and iterate until the user approves.

### 4. Wire up the live API

- **Client:** `api/client.ts` is a typed fetch with the Bearer token. A 401 sends the user to sign in; an old server shows an "update your server" message (checked via `info`).
- **Connect:** normalize the URL, check `info`, sign in, and keep the URL and token in `expo-secure-store`.
- **Data:** TanStack Query hooks, with returned screen data written straight into the cache, and a refetch when the app comes to the foreground.
- **Rest timer:**
  - starts when a set is logged, for the `restSeconds` the server returns
  - counts down against the clock, so it stays right if the phone sleeps
  - sits above the current set, and the list scrolls to it
- **Backup:** download with the token, then hand the file to the iOS share sheet.
- **iOS settings in `app.config.ts`:** `NSAllowsLocalNetworking` plus the local-network permission text, HTTPS everywhere else, and `ITSAppUsesNonExemptEncryption: false`.
- **Component tests:** carry over the intent of `pages_test` and `workout_test`: rest timer position, the rep picker's choices and highlighted targets, TM result cards and the decision form, skip offered only when allowed, chart edge cases, and the estimate error reopening the sheet.

### 5. EAS and TestFlight

- **Agent steps:** `mobile/eas.json` with `development` (simulator), `preview` and `production` profiles; `eas update:configure` with the runtime version set by `fingerprint`; bundle id `com.kbremont.<name>`.
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
  - **server job** (on `server/**` changes): gofmt, vet, staticcheck, `go test -race`, and the generate-then-`git diff` check, which now also covers `types.gen.ts` and the fixtures.
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
