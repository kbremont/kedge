import { TabStack } from '@/components/tab-stack';

// The tab opens on the root's section rows, under the pages they open.
export const unstable_settings = { anchor: 'settings/index' };

export default function SettingsLayout() {
  return <TabStack title="Settings" />;
}
