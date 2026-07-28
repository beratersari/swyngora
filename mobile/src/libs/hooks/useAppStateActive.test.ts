import { describe, expect, it, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';

const listeners = new Set<(s: string) => void>();

vi.mock('react-native', () => ({
  AppState: {
    currentState: 'active',
    addEventListener: (_: string, cb: (s: string) => void) => {
      listeners.add(cb);
      return {
        remove: () => {
          listeners.delete(cb);
        },
      };
    },
  },
}));

import { useAppStateActive } from './useAppStateActive';

describe('useAppStateActive', () => {
  beforeEach(() => {
    listeners.clear();
  });

  it('starts active and tracks background', () => {
    const { result } = renderHook(() => useAppStateActive());
    expect(result.current).toBe(true);
    act(() => {
      listeners.forEach((cb) => cb('background'));
    });
    expect(result.current).toBe(false);
    act(() => {
      listeners.forEach((cb) => cb('active'));
    });
    expect(result.current).toBe(true);
  });
});
