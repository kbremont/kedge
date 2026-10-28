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
make check    # fmt, vet, staticcheck, test
make generate # regenerate templ and sqlc output
```

Requires Go 1.27. The code generators (templ, sqlc, and tygo) and staticcheck
are pinned by the `tool` block in `go.mod` and run as `go tool templ`,
`go tool sqlc` and so on, so nothing needs installing.

Generated files (`*_templ.go`, `internal/db/*.go`) are committed, so none of
the generators is needed to build.

The Go module is `github.com/kbremont/531-tracker/server`, rooted here, so
`go test ./...` run from this directory covers the whole server and nothing
else.

### Layout

```
cmd/server          entrypoint
internal/program    the 5/3/1 engine — pure, no I/O, heavily tested
internal/db         schema, migrations, sqlc-generated queries
internal/server     HTTP handlers and orchestration
internal/views      templ templates and view models
internal/assets     embedded CSS and icons
```

`internal/program` is where the arithmetic lives — percentages, rounding,
plate math, which rounds carry a rep-max set. It has no dependencies and is the
first place to look when something prescribes the wrong number.

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
