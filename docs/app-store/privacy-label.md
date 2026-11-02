# App Privacy answers

How to fill in App Store Connect's App Privacy questionnaire for Kedge, and
why. The privacy policy itself is [docs/privacy.md](../privacy.md).

## Why Kedge declares anything

Nothing the lifter logs leaves their own server. But `expo-updates` checks
Expo's update server (`u.expo.dev`) at every launch, and Expo keeps those
requests for its usage insights. Apple counts data that a third-party
partner keeps longer than it takes to answer the request as collected. From
`FileDownloader.swift` in expo-updates, each check sends:

- **`EAS-Client-ID`:** a random UUID made once per install and kept in
  UserDefaults (`EASClientID.swift` in expo-eas-client). EAS also uses it to
  split staged rollouts.
- **`Expo-Current-Update-ID`, `Expo-Embedded-Update-ID`,
  `Expo-Recent-Failed-Update-IDs`, `Expo-Runtime-Version`,
  `Expo-Platform`:** which updates the install runs and has failed to
  launch.
- **`Expo-Fatal-Error`:** after a crash, the crash's error message, up to
  1024 characters.

## The answers

**Do you or your third-party partners collect data from this app?** Yes.

| Data type | Why | Purposes | Linked to the user | Tracking |
|---|---|---|---|---|
| Identifiers → Device ID | `EAS-Client-ID`, an id for the install | App Functionality, Analytics | No | No |
| Usage Data → Product Interaction | Each launch's update check, which Expo counts as launches and installs | App Functionality, Analytics | No | No |
| Diagnostics → Crash Data | `Expo-Fatal-Error` after a crash | App Functionality, Analytics | No | No |

- **App Functionality** covers delivering updates, splitting staged
  rollouts and rolling back an update that fails to launch.
- **Analytics** covers the counts EAS shows the developer: installs and
  launches per update, and how often an update fails.

The label then reads **Data Not Linked to You: Identifiers, Usage Data,
Diagnostics**.

## Not declared

- **What the lifter logs:** training maxes, sets and settings go only to
  the server they connect to, which they run.
- **The public demo:** the app doesn't point anyone to it. It's a server
  address like any other, listed on the support page. What's logged there
  is a shared demo lifter's, wiped every hour, with no access logs kept.
- **The IP address** that every request carries: neither the app nor Expo
  uses it for location.

## When to revisit

- **A package that sends data:** adding one, such as expo-insights,
  expo-observe, or a crash reporter, changes these answers.
- **Updates moving off Expo:** if updates were self-hosted, with nothing
  kept beyond each request, the label could go back to Data Not Collected.
