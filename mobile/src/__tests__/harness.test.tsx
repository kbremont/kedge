import { Host, Stepper, Text as SwiftText } from '@expo/ui/swift-ui';
import { labelsHidden } from '@expo/ui/swift-ui/modifiers';
import { SymbolView } from 'expo-symbols';
import { fireEvent, screen } from 'expo-router/testing-library';
import { useState } from 'react';

import ConnectScreen from '@/app/connect';
import { fakeServer, renderApp } from '@/test-utils';

// Smoke tests of what the screen tests build on: routes rendered through
// renderRouter inside the ApiProvider, and the native views jest.setup.ts
// stands in for.

test('a route renders inside the ApiProvider', async () => {
  await renderApp({ connect: ConnectScreen }, { initialUrl: '/connect', server: fakeServer({}), signedIn: false });

  expect(screen.getByRole('header', { name: 'Kedge' })).toBeOnTheScreen();
  const connect = screen.getByRole('button', { name: 'Connect' });
  expect(connect).toBeDisabled();
  await fireEvent.changeText(screen.getByLabelText('Server address'), 'raspberrypi:3000');
  await fireEvent.changeText(screen.getByLabelText('Password'), 'dev');
  expect(connect).toBeEnabled();
});

test('a SwiftUI control is found by its label and driven through its handler', async () => {
  function Rest() {
    const [seconds, setSeconds] = useState(90);
    return (
      <>
        <SymbolView name="timer" size={14} />
        <Host matchContents>
          <SwiftText>{`${seconds} s`}</SwiftText>
          <Stepper label="Rest" value={seconds} step={15} onValueChange={setSeconds} modifiers={[labelsHidden()]} />
        </Host>
      </>
    );
  }
  await renderApp({ index: Rest }, { server: fakeServer({}) });

  expect(screen.getByText('90 s')).toBeOnTheScreen();
  await fireEvent(screen.getByLabelText('Rest'), 'valueChange', 105);
  expect(screen.getByText('105 s')).toBeOnTheScreen();
});
