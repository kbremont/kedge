# App Review information

What goes in App Store Connect's App Review Information. The review copy's
password is never committed: it's in Vaultwarden, and only App Store
Connect gets it.

## Sign-in information

Sign-in required: **yes**.

| Field | Value |
|---|---|
| User name | `https://kedge-review.lyle.sh` (the server address; Kedge has no user names) |
| Password | the review copy's passphrase, from Vaultwarden |

## Contact information

The developer's name, phone and email, entered in App Store Connect.

## Notes

> Kedge is the iOS app for a 5/3/1 strength training server that each lifter runs themselves. The server is open source (https://github.com/kbremont/kedge) and ships as a Docker image (ghcr.io/kbremont/kedge). For review, we run a server with a demo lifter's training already logged.
>
> To sign in:
> 1. Open Kedge. On the Connect screen, enter the server address: https://kedge-review.lyle.sh
> 2. Enter the password from the Sign-in Information, and tap Connect.
>
> What to try:
> - Today shows the next session, Bench Press in the anchor's second round. Tap Start to open it. Each set shows its weight, reps and the plates to load. Log each set, and a rest timer starts. The PR set asks how many reps you did. Finish returns to Today.
> - History is a table of every round, one column per lift. Tap a set to open that session, or a lift's name to see its chart of estimated maxes.
> - Settings opens each section on its own page, and links the privacy policy.
>
> This review server resets every night, so anything changed during review is undone by the next day.
>
> Kedge has no accounts, sign-up or purchases. The password belongs to the server and is set by whoever runs it. Kedge asks for Local Network access only when its server's address is on the lifter's own network. The review server is on the internet, so it doesn't need that.
