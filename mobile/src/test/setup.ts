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

// lucide-react-native pulls react-native-svg (Flow syntax) — mock icons for Vitest/jsdom.
vi.mock('react-native-svg', () => {
  const Stub = ({ children, ...props }: { children?: React.ReactNode }) =>
    React.createElement('svg', props, children);
  return {
    __esModule: true,
    default: Stub,
    Svg: Stub,
    Path: Stub,
    Circle: Stub,
    Rect: Stub,
    G: Stub,
    Line: Stub,
    Polyline: Stub,
    Polygon: Stub,
  };
});

vi.mock('lucide-react-native', () => {
  const makeIcon = (name: string) => {
    function MockIcon({
      accessibilityLabel,
      size: _size,
      color: _color,
      strokeWidth: _sw,
      fill: _fill,
      ...rest
    }: {
      accessibilityLabel?: string;
      size?: number;
      color?: string;
      strokeWidth?: number;
      fill?: string;
    }) {
      return React.createElement('span', {
        role: 'img',
        'aria-label': accessibilityLabel ?? name,
        'data-icon': name,
        ...rest,
      });
    }
    MockIcon.displayName = name;
    return MockIcon;
  };

  // Explicit set — Proxy can hang some Vitest module inspects.
  const names = [
    'Star',
    'House',
    'Home',
    'ChartCandlestick',
    'TrendingUp',
    'SlidersHorizontal',
    'LayoutGrid',
    'ListOrdered',
    'X',
    'MessageCircle',
    'ChevronLeft',
    'Languages',
    'Search',
    'Filter',
    'RefreshCw',
  ];
  const exports: Record<string, unknown> = { __esModule: true };
  for (const n of names) exports[n] = makeIcon(n);
  return exports;
});
