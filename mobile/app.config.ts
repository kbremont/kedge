import type { ConfigContext, ExpoConfig } from 'expo/config';

// app.json is the app. APP_VARIANT=development, set by `make dev` and
// `make dev-install`, makes it Kedge Dev: an app of its own beside the
// production one on a phone, with its own name, keychain and links. Without
// the variable this returns app.json untouched, so production's fingerprint
// doesn't depend on this file.
export default ({ config }: ConfigContext): ExpoConfig => {
  const app = config as ExpoConfig;
  if (process.env.APP_VARIANT !== 'development') return app;
  return {
    ...app,
    name: 'Kedge Dev',
    scheme: 'kedge-dev',
    ios: {
      ...app.ios,
      // No appleTeamId: with one set, `expo run:ios` treats signing as done and
      // doesn't let xcodebuild create this bundle id's provisioning profile.
      bundleIdentifier: 'sh.lyle.kedge.dev',
    },
  };
};
