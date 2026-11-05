# <img src="docs/kedge-icon.png" alt="" width="40" height="40" align="absmiddle"> Kedge

The self-hosted 5/3/1 logbook: an iOS app and a small server for Jim
Wendler's 5/3/1.

The server works out every session: which one is next, each set's weight,
reps and plates, and the training max changes the program calls for. It
keeps your whole history in one SQLite file on your own hardware. The app
shows what it computes, logs what you lifted and times your rest, against
the clock, so it stays right while the phone sleeps.

- **Rounds, not calendar weeks.** Four sessions to a round, one lift each,
  so a missed day never breaks the schedule.
- **Start partway through.** Say which cycle you're on and which sessions
  you've done. There's no old history to key in.
- **Training maxes move when you say so.** After each TM test and at the
  start of each block, Kedge proposes changes and shows the evidence.
  Nothing is written until you accept.
- **The program:** leader and anchor cycles with 5's PRO, BBB and FSL, a
  7th Week TM test and an optional deload. Warmups, per-lift BBB overrides
  and separate rest times are settings.
- **History and charts:** a career table of every round, a chart of each
  lift's estimated max, and room for the numbers from your paper notebook.

No account, and nothing you log leaves your server. Train in pounds or kilograms.

## Try it

Kedge is on its way to the App Store as **Kedge Strength**. Once it's there,
you can try it before running anything: on the Connect screen, enter
`https://kedge-demo.lyle.sh` with the password `demo`. It's a demo lifter
partway through a block, shared with everyone who signs in and wiped every
hour.

## Run your own server

The server is one small Docker image, `ghcr.io/kbremont/kedge`, for amd64
and arm64 (a Raspberry Pi is plenty). [`docs/support.md`](docs/support.md)
has a compose file and covers reaching it from your phone and connecting.
[`server/README.md`](server/README.md) covers the program in detail,
configuration, HTTPS through Tailscale, backups and updates. The server
also has its own web UI, for the browser.

## The repo

| Directory | What it is |
|---|---|
| [`server/`](server/README.md) | The Go server: the 5/3/1 engine, SQLite, the JSON API and the web UI. All training logic lives here. Ships as a Docker image. |
| [`mobile/`](mobile/README.md) | Kedge, the native iOS app, built with Expo and shipped with EAS. It only displays what the server computes. |
| [`docs/`](docs) | The [privacy policy](docs/privacy.md), the [support page](docs/support.md), the App Store listing, and the approved screen designs. |

[`server/api/openapi.yaml`](server/api/openapi.yaml) is the contract between
the two halves. Both ends are generated from it.

## Development

The root `Makefile` drives both halves:

```sh
make run      # the server at http://localhost:3000, AUTH_PASSWORD=dev
make demo     # the same, against a database seeded with a demo block
make check    # server: fmt, vet, staticcheck, test
make generate # templ, sqlc, and both ends of the API from server/api/openapi.yaml
make app      # the Expo dev server
```

Go 1.27 for the server, Node 24 (`mobile/.nvmrc`) for the app. Bug reports
and ideas go in [Issues](https://github.com/kbremont/kedge/issues).

## License

[MIT](LICENSE). The app's fonts, Sofia Sans Extra Condensed, are under the
[SIL Open Font License](mobile/assets/fonts/OFL.txt). A few files in
`mobile/` come from Expo's `create-expo-app` template, under its MIT notice
in [`mobile/LICENSE-expo-template`](mobile/LICENSE-expo-template).

Kedge is not affiliated with or endorsed by Jim Wendler. 5/3/1 is his
training program.
