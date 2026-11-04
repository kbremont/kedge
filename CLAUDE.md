# Kedge

The app and how it's released are in `mobile/README.md`, its Expo rules in
`mobile/AGENTS.md`, and the server in `server/README.md`.

## Testing on the owner's phone

When the owner wants to try this workspace's changes on their phone, run
`make dev` from the repo's root as a background command, then tell them to open
Kedge Dev. It starts the demo server and Metro, and stops any left over from
another workspace. No cable or publishing is involved.

If it prints that Kedge Dev isn't installed or that native code differs, ask
them to plug in their phone, then run `make dev-install`. Never use `eas update`
to reach their phone before a merge: the `production` channel reaches every
install.

## Testing in the simulator

Keep the simulator inside Claude Desktop, using its simulator tools: build with
`xcodebuild` or the build tool, and install with the control tool's `launch`.
Don't use `expo run:ios` or `make dev-install` against a simulator, or press `i`
in Metro: Expo also opens Xcode's Device Hub in a window of its own.
