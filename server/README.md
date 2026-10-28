# 5/3/1 Tracker: server

The Go server: the 5/3/1 engine, the SQLite database, and the web UI. Paths
below are relative to this directory unless they say otherwise.

- **One static binary.** No Node, no Postgres, no Redis, no sidecar. The
  container is `FROM scratch` and about 11 MB.
- **No JavaScript.** Every interaction is a form post. The rest timer is a CSS
  animation.
- **SQLite.** Backup is one file, or one click.
- **Installs to the home screen** as a PWA. It needs the network to run.

## The program

The default block is eleven rounds and then repeats:

| Rounds | Cycle | Main work | Supplemental |
|--------|-------|-----------|--------------|
| 1–3    | Leader 1 | 5's PRO — 5/5/5, no rep-max sets | BBB 5×10 @ 50% |
| 4–6    | Leader 2 | 5's PRO | BBB 5×10 @ 60% |
| 7      | 7th Week TM Test | 70%×5, 80%×3, 90%×1, 100%×3+ (AMRAP) | — |
| 8–10   | Anchor | 5/3/1 with a PR set | FSL 5×5 |
| 11     | 7th Week Deload (optional) | 70%×5, 80%×3–5, 90%×1, 100%×1 | — |

The two 7th weeks are the *5/3/1 Forever* protocols. The TM test always runs,
and its 100% AMRAP counts toward each lift's best estimated max and progress
chart, like an anchor PR set. The deload is offered when the anchor ends, and
can be skipped to go straight into the next block.

Warmups, when enabled in Settings, are 40/50/60% × 5.

BBB load can be overridden per lift from Settings, as a percentage of that
lift's training max or as a fixed weight, replacing the 50%/60% ramp above for
that lift. Rest is configurable separately for warmups, main sets, and
supplemental work (BBB and FSL).

Four sessions to a round, one lift each: press, deadlift, bench, squat.
Supplemental work is on the same lift as the main work.

A **round** is four sessions, not a calendar week. The two drift apart the first
time life gets in the way — round 1's bench and squat can easily land on the
following Monday and Tuesday — so the app counts rounds and leaves the calendar
alone.

Programs are data, not code — see `LeaderAnchorBBB` in
`internal/program/program.go`. Adding a template is a config change.

### Starting partway through

You almost certainly do not start using this on round 1, day 1. On first run,
after entering your training maxes, the app asks which cycle you are on and
which of that cycle's sessions you have already done. Cycles before the one you
pick are left out of the block entirely, so there is no old history to key in
and nothing invented to represent it. The first session you leave unticked is
where the app picks up.

### Past estimates

If you ran 5/3/1 on paper first, add those numbers to each lift's chart from
its page (tap a lift on the dashboard or in History, then the + next to Past
estimates): a date, a weight, and reps, which default to 1. An estimated max goes in as weight × 1; a notebook PR
set such as 135 × 8 is converted the same way a logged set is. Past estimates
show on the chart as hollow dots and count toward the lift's best estimated
max, which is also the number to beat on an AMRAP set. They are stored apart
from workouts, so they never become sessions or history, and they never feed a
TM test result, the anchor's PR sets or the block-start review.

### Training maxes only move when you say so

- **After each lift's TM test**, the dashboard reports the result straight
  away. 3–5 reps at 100% means the TM is right. 6 or more means it can go up,
  and that happens at the end of the block. Under 3 reps means the TM is too
  high: the app offers to lower it by 10% (editable) or keep it, and the anchor
  waits until you choose. The rest of the TM test week carries on regardless.
  Nothing goes up here, so the anchor never runs on a training max raised
  partway through the block.
- **At the start of each new block**, every lift gets a proposal from the
  anchor's PR sets and the TM test. Missing the minimum on any PR set (5+, 3+,
  1+) resets the TM by 10%. Making all of them earns a step (+5 upper, +10
  lower), or two steps if the TM test also went 6+. Every proposal is editable,
  nothing is written until you accept, and the block waits until you do.

You can still change any training max at any time from Settings.

Training maxes are an append-only log, and sets store absolute weights
snapshotted when a session starts. Changing a training max today affects
tomorrow's sessions and leaves everything you have already lifted alone.

## Running it

From `server/`:

```sh
cp .env.example .env      # set AUTH_PASSWORD
docker compose up -d
```

Then open the app, enter your training maxes, tell it where in the programme
you are, and start training.

### Serving it over HTTPS with Tailscale

A PWA will only install over HTTPS, and `http://192.168.x.x:3000` will not do.
If you already use Tailscale, this is the shortest path:

```sh
tailscale serve --bg 3000
```

That gives you `https://<machine>.<tailnet>.ts.net` with a real certificate,
reachable from your phone at the gym, with nothing exposed to the internet.
Enable **MagicDNS** and **HTTPS Certificates** in the Tailscale admin console
first.

The compose file binds the container to `127.0.0.1` on the assumption that
Tailscale is publishing it. Change the port mapping to `"3000:3000"` if you
want it on your LAN instead.

### Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `AUTH_PASSWORD` | *(required)* | The only credential. Changing it signs you out. |
| `DATA_DIR` | `/data` | Where `531.db` lives. |
| `ADDR` | `:3000` | Listen address. |
| `TZ` | `UTC` | Zone for displayed dates; storage is always UTC. |

### Backups

`Settings → Download backup` streams a `VACUUM INTO` snapshot, which is
consistent and safe to take while the app is running. Or copy `data/531.db`
while the container is stopped.

## Development

From the repo root:

```sh
make run      # http://localhost:3000, AUTH_PASSWORD=dev, data in server/data
make demo     # the same, against server/data-demo filled with a demo block
make check    # fmt, vet, staticcheck, test
make golden   # rewrite the API's golden responses
make generate # regenerate templ, sqlc, and the app's API types and fixtures
```

Requires Go 1.27. The code generators (templ, sqlc, and tygo) and staticcheck
are pinned by the `tool` block in `go.mod` and run as `go tool templ`,
`go tool sqlc` and so on, so nothing needs installing.

Generated files (`*_templ.go`, `internal/db/*.go`, and the app's
`mobile/src/api/types.gen.ts` and `fixtures/`) are committed, so none of the
generators is needed to build. CI fails if `make generate` changes anything.

The Go module is `github.com/kbremont/531-tracker/server`, rooted here, so
`go test ./...` run from this directory covers the whole server and nothing
else.

### Layout

```
cmd/server          entrypoint
internal/program    the 5/3/1 engine — pure, no I/O, heavily tested
internal/db         schema, migrations, sqlc-generated queries
internal/training   the program applied to the database; returns read models
internal/auth       session tokens, from a cookie or a Bearer header
internal/web        HTML: thin handlers, templ templates, embedded CSS and icons
internal/api        the app's JSON API: wire types, thin handlers, golden responses
internal/demo       fills an empty database with a demo block (-seed-demo)
internal/server     routes, /healthz, panic recovery
```

`internal/program` is where the arithmetic lives — percentages, rounding,
plate math, which rounds carry a rep-max set. It has no dependencies and is the
first place to look when something prescribes the wrong number.

`internal/training` is where the rules meet the database: where you are in the
block, starting and logging sessions, the training max decisions, onboarding.
Its methods return one read model per screen with everything worked out, so
the handlers in `internal/web` only parse forms and render, and
`internal/api` serves the same screens as JSON without repeating any of the
logic.

### The JSON API

The iOS app talks to `/api/v1`. Every route answers JSON, never a redirect,
and `Cache-Control: no-store`.

**Signing in.** `GET /api/v1/info` needs no session and answers
`{apiVersion, version}`. `POST /api/v1/login` with `{"password": "…"}`
answers `{token, expiresAt}` (RFC 3339), or a 401. Send the token as
`Authorization: Bearer <token>`. It is the same token as the browser's
cookie, so it lasts 60 days, and changing `AUTH_PASSWORD` invalidates it.
Nothing is stored on the server, so signing out is the app forgetting the
token.

**Versioning.** `apiVersion` only goes up for a change that breaks an
installed app: a field removed, renamed or retyped, or a route changed.
Adding a field or a route is not breaking, since the app ignores what it does
not know, and the server ignores request fields it does not know. The app
checks `info` first and asks for a server update when it is behind.

**Routes**, all under `/api/v1`:

| Route | Body | Answers |
|---|---|---|
| `GET info` *(public)* | | `Info` |
| `POST login` *(public)* | `{password}` | `{token, expiresAt}` |
| `GET dashboard` | | `Dashboard`, with `needsSetup` before training maxes are entered |
| `POST workouts` | | the `Workout` started, or the one in progress |
| `GET workouts/{id}` | | `Workout` |
| `PUT workouts/{id}/sets/{idx}` | `{reps}`, required (0 is a failed set) | `Workout`; `restSeconds` starts the rest timer |
| `DELETE workouts/{id}/sets/{idx}` | | `Workout` |
| `POST workouts/{id}/finish` | | `Dashboard` |
| `DELETE workouts/{id}` | | `Dashboard` |
| `GET history` | | `History` |
| `GET lifts/{slug}` | | `Lift` |
| `POST lifts/{slug}/estimates` | `{date, weight, reps}` | `Lift` |
| `DELETE lifts/{slug}/estimates/{id}` | | `Lift` |
| `GET settings`, `PUT settings` | `SettingsInput` | `Settings` |
| `POST training-maxes` | `{lift, value, note}` | `Settings` |
| `POST tm-review` | `{key, tms: {slug: value}}` | `Dashboard` |
| `POST tm-tests/{id}` | `{action: "keep" \| "lower", tm}` | `Dashboard` |
| `POST skip-cycle` | | `Dashboard` |
| `GET setup`, `PUT setup` | `{tms: {slug: value}}` | `Setup`; the PUT answers `SetupPosition` |
| `GET setup/position` | | `SetupPosition` |
| `GET setup/history/{cycle}`, `PUT` | `{done: [{round, slug, prReps}]}` | `SetupHistory`; the PUT answers `Dashboard` |
| `GET backup` | | the database, as `application/octet-stream` |

A change answers with the screen the web form would redirect to, so the app
never needs a second request. Input is read exactly as the forms read it:
for example, a settings number that is not positive keeps the current value.

**Errors** are `{"error": "…"}` with a message that can be shown as it is:

| Status | When |
|---|---|
| 400 | input training rejects, or a body that is not the JSON the route expects |
| 401 | no session, an expired one, or a wrong password |
| 404 | a workout, lift, estimate or cycle that does not exist, or an unknown path |
| 409 | training is held on a decision, or the decision is no longer on offer, or there is nothing to start; fetch the dashboard |
| 500 | logged on the server; the message is generic |

**The contract with the app.** The wire types in `internal/api/types.go` are
the source of truth. `internal/api/testdata/*.json` holds a golden response
for every screen state the app shows. After changing the API:

```sh
make golden    # rewrite the golden responses; review the diff
make generate  # types.gen.ts and fixtures/ in mobile/src/api
```

and commit the result. `go test ./internal/api` fails while a golden response
is out of date, and CI fails while the app's copies are.

### Demo data

`-seed-demo` fills an empty database with a demo lifter's block before
serving: both leaders, a passed TM test, part of the anchor, and past
estimates, dated over the weeks up to yesterday. A database with training
maxes is left alone, so the flag is safe to leave on. `make demo` runs it
against `server/data-demo`; the public demo server for App Review runs the
image with `server -seed-demo` and its own volume and password.

### Deploying to the Pi

Let CI cross-build for arm64 and pull the image; building on the Pi itself is
slow. `.github/workflows/release.yml` (at the repo root) publishes to GHCR on
every push to `main` that changes `server/`. The compose file already points `image:` at
`ghcr.io/kbremont/531-tracker:latest` and keeps the `build:` block alongside
it, so the same file works both ways:

| Goal | Command |
|---|---|
| Update the Pi | `docker compose pull && docker compose up -d` |
| Test the real container locally | `docker compose up --build` |
| Day-to-day development | `make run` (unchanged, no Docker) |

Always `pull` first and check it succeeded — if `pull` fails and no image is
cached locally, `up` silently falls back to building on the Pi, which is slow
and the thing this setup exists to avoid.
