# 5/3/1 Tracker

A self-hosted tracker for Jim Wendler's 5/3/1, built for one person and a
phone at the gym.

- **One static binary.** No Node, no Postgres, no Redis, no sidecar. The
  container is `FROM scratch` and about 11 MB.
- **No JavaScript.** Every interaction is a form post. The rest timer is a CSS
  animation.
- **SQLite.** Backup is one file, or one click.
- **Installs to the home screen** as a PWA. It needs the network to run.

## The program

The default block is ten rounds and then repeats:

| Rounds | Cycle | Main work | Supplemental |
|--------|-------|-----------|--------------|
| 1–3    | Leader 1 | 5's PRO — 5/5/5, no rep-max sets | BBB 5×10 @ 50% |
| 4–6    | Leader 2 | 5's PRO | BBB 5×10 @ 60% |
| 7      | Deload | 40/50/60 × 5 | — |
| 8–10   | Anchor | 5/3/1 with a PR set | FSL 5×5 |

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

### Training maxes never move on their own

The app will *offer* a bump at the end of a cycle (+5 upper, +10 lower) and
you can change any training max at any time from Settings. It never applies
one for you.

Training maxes are an append-only log, and sets store absolute weights
snapshotted when a session starts. Changing a training max today affects
tomorrow's sessions and leaves everything you have already lifted alone.

## Running it

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

```sh
make run      # http://localhost:3000, AUTH_PASSWORD=dev
make check    # fmt, vet, test
make generate # regenerate templ and sqlc output
```

Requires Go 1.27, plus [templ](https://templ.guide) and
[sqlc](https://sqlc.dev) if you are changing templates or queries:

```sh
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Generated files (`*_templ.go`, `internal/db/*.go`) are committed, so neither
tool is needed to build.

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
slow. `.github/workflows/release.yml` publishes to GHCR on every push to
`main`. The compose file already points `image:` at
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
