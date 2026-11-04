# Kedge

The iOS app, built with Expo (SDK 58). It shows what
the Go server in [`../server`](../server/README.md) computes and keeps no
training logic of its own. Bundle id `sh.lyle.kedge`; EAS project
[`@lyle15/kedge`](https://expo.dev/accounts/lyle15/projects/kedge); App Store
Connect app `6829543729`.

## Develop

```sh
npm ci
npx expo run:ios       # build the dev client and install it in the simulator
npx expo start         # Metro, for the dev client
npx tsc --noEmit && npx expo lint && npx jest
```

`ios/` is generated (Continuous Native Generation) and ignored: configure
native behaviour in `app.json`. (`app.config.ts` only makes it Kedge Dev,
below.) After adding a package with native code, or
taking a new SDK release, rebuild the dev client with
`npx expo prebuild --clean -p ios` and `npx expo run:ios`. Restart Metro
with `--clear` after `npx expo install` changes packages under it.

## On your phone

**Kedge Dev** is an app of its own, for testing a workspace on a real phone
before it's merged. It has its own name and bundle id
(`sh.lyle.kedge.dev`), so it sits beside the production app rather than
replacing it, and signs in separately. From the repo's root:

```sh
make dev           # the demo server and Metro, from this workspace
make dev-install   # build it and install it on a plugged-in phone
```

`make dev` is the everyday one. Open Kedge Dev on the same Wi-Fi as the Mac,
and what you save appears on the phone, with no cable and nothing published.
It stops anything left on ports 8081 and 3000 by another workspace, so the
last workspace to run it is the one the phone shows. It prints the server's
address and password, which is `dev`; each workspace has its own demo data,
so a new one may ask for the password again.

`make dev-install` is for the first time and after native changes. `make dev`
compares the workspace's fingerprint with the installed one and says when one
is needed, since Metro itself won't: without a rebuild, a native change shows
up as a crash or a missing module. The first install needs Developer Mode on
the phone and the Apple account signed in to Xcode. It leaves Kedge Dev's
project in that workspace's `ios/`, so run `npx expo prebuild --clean -p ios`
before going back to `npx expo run:ios` there.

Kedge Dev is a Debug build, so it needs Metro on the Mac and won't load without
it. It's built only with these targets: EAS builds and updates don't set
`APP_VARIANT`, and `app.config.ts` leaves `app.json` untouched without it, so
production's fingerprint doesn't change.

## Release

Run EAS as `npx eas-cli@latest …` from this directory, logged in with
`eas login`. `eas.json` has three build profiles:

| Profile | Build | Update channel |
|---|---|---|
| `development` | the dev client, for the simulator | none: it loads from Metro |
| `preview` | a release build for the simulator | `preview` |
| `production` | for the App Store and TestFlight | `production` |

Builds run on the Node version in `.nvmrc`. Settings' App row says what an
install is running: its version and build, then `built in`, `update <date>`
or `development`.

### Apple credentials

The Apple account signs in with security keys, which EAS's Apple ID sign-in
can't use, so EAS talks to Apple with an App Store Connect API key: a team
key with Admin access, kept outside the repo. Export it before a production
build or a submit:

```sh
export EXPO_ASC_API_KEY_PATH=~/.appstoreconnect/private_keys/AuthKey_<KEY_ID>.p8
export EXPO_ASC_KEY_ID=<KEY_ID>
export EXPO_ASC_ISSUER_ID=<ISSUER_ID>
export EXPO_APPLE_TEAM_ID=6ZQGF8WCR4
export EXPO_APPLE_TEAM_TYPE=INDIVIDUAL
```

EAS keeps the distribution certificate and provisioning profile it made, and
`eas.json`'s submit profile reads the same three `EXPO_ASC_*` variables. A
key can't create an App Store Connect app; that one was made in the browser.

### A build

Native changes (a package with native code, an SDK release, most of
`app.json`) need a new build. TestFlight builds also expire after 90 days.

```sh
npx eas-cli@latest build -p ios --profile production --auto-submit   # to TestFlight
npx eas-cli@latest build -p ios --profile preview                    # keep preview on the same runtime
```

The build number rises by itself. Apple takes a few minutes to process an
upload; then it's in TestFlight for the internal testers.

### An update

JavaScript, styling and images can go out as an update, without a build. It
goes to `preview` first, is checked in the preview build, then the same
bundle is republished to `production`:

```sh
npx eas-cli@latest update -p ios --channel preview --environment preview --message "…"
npx eas-cli@latest update:republish -p ios --group <update group id> --destination-channel production
```

`eas update` prints the update group id. (`update:republish --channel preview`
picks the group interactively instead.)

The preview build and the dev client share the bundle id, so the preview
build goes on a simulator of its own:
`npx eas-cli@latest build:run -p ios --profile preview --latest --simulator <name or UDID>`.

A release build downloads an update in the background at launch and runs it
from the next launch, so close it fully and open it again, up to twice.

The runtime version is a fingerprint of everything native
(`runtimeVersion.policy: fingerprint`). An update only reaches builds with
the same fingerprint, so after a native change it reaches no one until there
is a new build. `eas update` prints the runtime it published for. Check a
build against the working tree with
`npx eas-cli@latest fingerprint:compare --build-id <id>`, or two builds with
two `--build-id`s.

## The App Store

Kedge is listed as **Kedge Strength**, since "Kedge" was taken; the name
under the icon stays Kedge. Everything App Store Connect holds lives in
[`../docs/app-store/`](../docs/app-store/):

- **`listing.md`:** every field of the listing.
- **`privacy-label.md`:** the App Privacy answers, and why. Revisit them
  before adding any package that sends data.
- **`review-notes.md`:** App Review's sign-in and notes.
- **`screenshots/`:** the captioned screenshots.

The privacy policy and support page are
[`../docs/privacy.md`](../docs/privacy.md) and
[`../docs/support.md`](../docs/support.md), read on GitHub. Settings links
the policy through `privacyPolicyUrl` in `src/lib/brand.ts`, so keep its
path: a new one reaches installed apps only through an update.

App Review signs in to its own demo server, `https://kedge-review.lyle.sh`,
whose passphrase is in Vaultwarden and App Store Connect only. The public
demo, `https://kedge-demo.lyle.sh` with the password `demo`, is for anyone
trying Kedge. Both run on the homelab's VPS with `-demo` (see the homelab's
`vps/README.md`).

### A new version for review

JavaScript reaches installed apps as an update. A new version is for
native changes, or for changing the listing or screenshots, which belong
to a version.

1. **The version:** raise `version` in `app.json`. The version in App Store
   Connect has to match it exactly (`1.0.0`, not `1.0`).
2. **The demo:** from the homelab's `vps/`, run `make update-kedge`, so
   App Review gets the server this build expects. Check
   `curl https://kedge-review.lyle.sh/api/v1/info`.
3. **The build:** export the Apple credentials above, with the issuer ID
   from App Store Connect (Users and Access → Integrations), then run
   `npx eas-cli@latest build -p ios --profile production --auto-submit`.
4. **The phone:** check it on TestFlight. It should sign in to your own
   server, and to `kedge-review` with NetBird off.
5. **App Store Connect:** choose the build, update anything in
   `docs/app-store/` that changed, add it for review, and release it by
   hand once it's approved.

**Screenshots:** they go in "iPhone with Dynamic Island (medium display)" at
1206×2622, which refuses 6.9" (1320×2868) images. Retake them whenever a
screen they show changes:
1. **Capture:** run the preview build on the iPhone 17 Pro Max simulator,
   signed in to a `-demo` server. Set the status bar with
   `xcrun simctl status_bar <udid> override --time 9:41 …`.
2. **Compose:** set each caption like a section heading, over the screen in
   a flat ink phone edge. Each PNG records in its metadata how it was made.
