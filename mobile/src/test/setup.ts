import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';
import React from 'react';

vi.mock('react-native-safe-area-context', () => {
  const SafeAreaView = ({
    children,
    style,
  }: {
    children?: React.ReactNode;
    style?: unknown;
    edges?: string[];
  }) => React.createElement('div', { style, 'data-testid': 'safe-area' }, children);

  return {
    SafeAreaProvider: ({ children }: { children?: React.ReactNode }) =>
      React.createElement(React.Fragment, null, children),
    SafeAreaView,
    useSafeAreaInsets: () => ({ top: 0, right: 0, bottom: 0, left: 0 }),
    initialWindowMetrics: {
      frame: { x: 0, y: 0, width: 0, height: 0 },
      insets: { top: 0, right: 0, bottom: 0, left: 0 },
    },
  };
});
