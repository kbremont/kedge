# Kedge privacy policy

Effective November 1, 2026.

Kedge is an iOS app for Jim Wendler's 5/3/1 that works with a server you
run yourself. It's made by Kyle Bremont, an individual developer. This
policy covers the app and the public demo server.

## In short

Everything you log goes to your own server and nowhere else. The app has no
account, no ads, no tracking and no analytics. The one other place it
contacts is Expo, the service that delivers updates to the app's code, and
this page says exactly what that request carries.

## What stays on your phone

- **Your server's address and your session,** in the iOS keychain. The
  session is a token your server issued when you signed in, and it lasts 60
  days.
- **Your Appearance choice** (System, Light or Dark), in the keychain too.
- **The update identifier** described under Expo, below.

Nothing you log is stored on the phone.

## What goes to your server

Everything you log goes to the server you connect to and is stored there:
your training maxes, sessions and sets, past estimates and settings. You
run that server, and the developer has no access to it.

## What goes to Expo

Kedge receives updates to its code through Expo's EAS Update service. Each
time the app opens, it asks Expo's update server whether an update is
ready. That request carries:

- a random identifier created for this install of the app, which isn't
  tied to you, your Apple ID or your device's advertising identifier
- the platform (iOS), the version of the app's native code, and which
  updates the app has and is running
- after the app has crashed, the crash's error message

Like any request on the internet, it also reveals your IP address. Expo uses
this to deliver updates and to show the developer how many installs run each
update and how often an update fails. None of it includes anything you log.
See [Expo's privacy policy](https://expo.dev/privacy).

## What Kedge doesn't do

- No account or sign-up. The password you enter is your own server's.
- No ads, no tracking across apps or websites, and no analytics or
  crash-reporting service beyond the update check above.
- Nothing is sold or shared.

## The public demo

The support page lists a public demo server, `https://kedge-demo.lyle.sh`,
for trying Kedge before running your own. If you connect to it, what you
log goes to that demo server, which the developer runs.

- Everyone who signs in shares one demo lifter, so others can see what you
  log there. Don't enter anything personal.
- The demo is wiped and refilled every hour.
- It keeps no access logs.

## Deleting your data

- **On your server:** your data is yours to delete there, for example by
  removing its database file.
- **On your phone:** signing out removes your session, and keeps the
  server's address so you can connect again. Deleting the app removes the
  update identifier. iOS can keep an app's keychain items after the app is
  deleted, and Kedge's hold only the address, a session that expires and
  the Appearance choice.
- **At Expo:** Expo keeps update-check records under its own policy, and
  they hold no training data.

## Children

Kedge isn't directed at children under 13 and doesn't knowingly collect
their information.

## Changes

Changes to this policy are made in this file, and its history is in the
repository.

## Contact

Open an issue at
[github.com/kbremont/kedge/issues](https://github.com/kbremont/kedge/issues).
