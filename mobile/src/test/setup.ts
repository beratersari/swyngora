import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';
import React from 'react';
import { initI18n } from '@/libs/i18n';

// Ensure translations resolve in unit tests (default en).
initI18n();

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// ResizeObserver stub for charts
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver;

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
