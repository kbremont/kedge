import { render, screen } from '@testing-library/react-native';
import { AccessibilityInfo } from 'react-native';

import { Button, Notice, TextAction } from '@/components/card-back';

// How the building blocks read to VoiceOver.

const announce = jest.mocked(AccessibilityInfo.announceForAccessibility);

beforeEach(() => announce.mockClear());

describe('Notice', () => {
  test('an error is one alert, announced as it appears and whenever it changes', async () => {
    const view = await render(<Notice error>Couldn’t reach your server.</Notice>);
    expect(screen.getByRole('alert')).toHaveProp('accessible', true);
    expect(announce.mock.calls).toEqual([['Couldn’t reach your server.']]);

    await view.rerender(<Notice error>Incorrect password.</Notice>);
    await view.rerender(<Notice error>Incorrect password.</Notice>);
    expect(announce.mock.calls).toEqual([['Couldn’t reach your server.'], ['Incorrect password.']]);
  });

  test('an error written in parts is announced whole', async () => {
    const short = 2.5;
    await render(
      <Notice error>
        {short} lb short: your plates can’t load {187.5} lb exactly.
      </Notice>,
    );
    expect(announce).toHaveBeenCalledWith('2.5 lb short: your plates can’t load 187.5 lb exactly.');
  });

  test('news is one element too, but not announced on iOS', async () => {
    await render(<Notice>Saved. The next session uses these settings.</Notice>);
    expect(screen.getByText('Saved. The next session uses these settings.').parent).toHaveProp('accessible', true);
    expect(screen.queryByRole('alert')).not.toBeOnTheScreen();
    expect(announce).not.toHaveBeenCalled();
  });
});

// The native icon would add its own name to the label, such as "selected"
// for the checkmark, and the caps the label is printed in would be read.
test.each([
  ['Button', <Button key="b" icon="checkmark" label="Log 5 reps" />, 'Log 5 reps'],
  ['TextAction', <TextAction key="t" icon="rectangle.portrait.and.arrow.right" label="Sign out" />, 'Sign out'],
])('a %s with an icon reads as its words alone', async (_, element, label) => {
  await render(element);
  expect(screen.getByRole('button', { name: label })).toHaveProp('accessibilityLabel', label);
});
