// Stand-ins for the native modules jest-expo leaves as empty native views or
// no-op functions, so tests can read what a screen shows and drive its
// controls.

// The keychain, in memory: what a test saves it can read back. Each test file
// starts with it empty.
jest.mock('expo-secure-store', () => {
  const items = new Map<string, string>();
  return {
    isAvailableAsync: jest.fn(async () => true),
    getItemAsync: jest.fn(async (key: string) => items.get(key) ?? null),
    setItemAsync: jest.fn(async (key: string, value: string) => {
      items.set(key, value);
    }),
    deleteItemAsync: jest.fn(async (key: string) => {
      items.delete(key);
    }),
    getItem: jest.fn((key: string) => items.get(key) ?? null),
    setItem: jest.fn((key: string, value: string) => {
      items.set(key, value);
    }),
  };
});

// The app's version and build, which jest-expo's stand-in gives as "mock".
// Tests run with __DEV__ set, so the App row reads as a development build.
jest.mock('expo-application', () => ({
  ...jest.requireActual('expo-application'),
  nativeApplicationVersion: '1.0.0',
  nativeBuildVersion: '1',
}));

// An SF Symbol draws nothing a test can read; a plain view keeps its props.
jest.mock('expo-symbols', () => {
  const { createElement } = jest.requireActual<typeof import('react')>('react');
  const { View } = jest.requireActual<typeof import('react-native')>('react-native');
  return { SymbolView: (props: object) => createElement(View, props) };
});

// SwiftUI views become plain views carrying their props, so a test can fire
// a control's handler: fireEvent(stepper, 'valueChange', 120) calls its
// onValueChange. A string `label` becomes the accessibility label, so
// getByLabelText finds a Stepper or Toggle, and Text renders its text. Any
// component name works, so a screen can use one this list never named.
jest.mock('@expo/ui/swift-ui', () => {
  const { createElement } = jest.requireActual<typeof import('react')>('react');
  const { Text, View } = jest.requireActual<typeof import('react-native')>('react-native');
  const made = new Map<string, unknown>();
  const make = (name: string) => {
    if (name === 'Text') return (props: object) => createElement(Text, props);
    return (props: { label?: unknown; testID?: string }) =>
      createElement(View, {
        testID: name,
        accessibilityLabel: typeof props.label === 'string' ? props.label : undefined,
        ...props,
      });
  };
  return new Proxy(
    {},
    {
      get(_, name) {
        if (name === '__esModule') return true;
        if (typeof name !== 'string' || !/^[A-Z]/.test(name)) return undefined;
        if (!made.has(name)) made.set(name, make(name));
        return made.get(name);
      },
    },
  );
});

// Modifiers only describe a SwiftUI view, so each is a plain record of its
// name and arguments.
jest.mock('@expo/ui/swift-ui/modifiers', () =>
  new Proxy(
    {},
    {
      get(_, name) {
        if (name === '__esModule') return true;
        if (typeof name !== 'string') return undefined;
        return (...args: unknown[]) => ({ modifier: name, args });
      },
    },
  ),
);
