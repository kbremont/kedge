# 5/3/1 Tracker

A self-hosted tracker for Jim Wendler's 5/3/1, built for one person and a
phone at the gym.

The repo holds both halves:

| Directory | What it is |
|---|---|
| [`server/`](server/README.md) | The Go server: the 5/3/1 engine, SQLite, and the web UI. All training logic lives here. Ships as a Docker image. |
| [`mobile/`](mobile/README.md) | Kedge, the native iOS app, built with Expo and shipped through TestFlight with EAS. It only displays what the server computes. |

See [`server/README.md`](server/README.md) for the program, running and
deploying the server, configuration, and backups, and
[`mobile/README.md`](mobile/README.md) for building and releasing the app.

## Development

The root `Makefile` drives both halves:

```sh
make run      # the server at http://localhost:3000, AUTH_PASSWORD=dev
make check    # server: fmt, vet, staticcheck, test
make generate # templ, sqlc, and both ends of the API from server/api/openapi.yaml
make app      # the Expo dev server
```
