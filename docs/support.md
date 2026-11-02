# Kedge support

Kedge is an iOS app for Jim Wendler's 5/3/1. It tells you which session is
next, prescribes every set's weight, reps and plates, logs what you lifted,
times your rest, charts your estimated maxes, and proposes training max
changes when the program calls for them.

All of that is worked out by a server you run yourself, and the app shows
what it computes. Your whole history is one SQLite file on that server.

Kedge isn't affiliated with or endorsed by Jim Wendler.

## Try it first

Before running a server, connect to the public demo:

- **Server address:** `https://kedge-demo.lyle.sh`
- **Password:** `demo`

It's a demo lifter partway through a block, with both leaders, a TM test and
part of the anchor behind them. Everyone who signs in shares that lifter, and
it's wiped and refilled every hour, so don't enter anything personal.

## Run your own server

The server is one small Docker image, `ghcr.io/kbremont/kedge`, for amd64
and arm64 (a Raspberry Pi is plenty). A `docker-compose.yml`:

```yaml
services:
  kedge:
    image: ghcr.io/kbremont/kedge:latest
    restart: unless-stopped
    environment:
      # The only credential. Changing it signs every device out.
      AUTH_PASSWORD: ${AUTH_PASSWORD:?set AUTH_PASSWORD in .env}
      # Dates are shown in this zone; the database stores UTC.
      TZ: America/New_York
    volumes:
      - ./data:/data
    ports:
      - "127.0.0.1:3000:3000"
    healthcheck:
      test: ["CMD", "/server", "-healthcheck"]
      interval: 30s
```

Put a strong `AUTH_PASSWORD` in `.env` beside it, then `docker compose up -d`.

Then make it reachable from your phone at the gym. The usual way is a VPN
such as Tailscale or NetBird, with HTTPS in front of the server: Tailscale
Serve, Caddy or another reverse proxy. The
[server's README](../server/README.md) covers configuration, HTTPS through
Tailscale, backups and updates.

## Connecting

On Connect, enter the server's address and its password.

- **The address** can be typed without `https://`, and Kedge adds it. A
  pasted `/api/v1` on the end is dropped.
- **Use HTTPS** for any address on the internet or a VPN's domain.
- **Plain `http://`** works only for an address on your own network: a
  short name such as `raspberrypi:3000`, a `.local` name, `localhost` or an
  IP address. For those, Kedge adds `http://` itself. iOS asks once for
  Local Network access the first time.

If Kedge can't connect, it shows why:

- **The password was refused:** check `AUTH_PASSWORD`, and remember that a
  running container keeps the password it started with until it's
  recreated.
- **The server couldn't be reached:** check that the phone is on the VPN or
  network, and that the address opens `/api/v1/info` in Safari.
- **"Update the server" or "Update Kedge":** the app and the server speak
  different versions of their API. Update whichever it names.

## Get help

Open an issue at
[github.com/kbremont/kedge/issues](https://github.com/kbremont/kedge/issues).
The [privacy policy](privacy.md) says what the app keeps and sends.
