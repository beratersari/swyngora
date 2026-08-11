import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useMediaQuery } from './useMediaQuery';

describe('useMediaQuery', () => {
  const listeners = new Set<(ev: MediaQueryListEvent) => void>();
  let matches = false;

  beforeEach(() => {
    matches = false;
    listeners.clear();
    vi.stubGlobal(
      'matchMedia',
      (query: string): MediaQueryList =>
        ({
          media: query,
          get matches() {
            return matches;
          },
          onchange: null,
          addEventListener: (_: string, cb: (ev: MediaQueryListEvent) => void) => {
            listeners.add(cb);
          },
          removeEventListener: (_: string, cb: (ev: MediaQueryListEvent) => void) => {
            listeners.delete(cb);
          },
          addListener: () => undefined,
          removeListener: () => undefined,
          dispatchEvent: () => true,
        }) as MediaQueryList,
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('tracks matchMedia changes', () => {
    const { result } = renderHook(() => useMediaQuery('(max-width: 720px)'));
    expect(result.current).toBe(false);
    matches = true;
    act(() => {
      listeners.forEach((cb) => cb({ matches: true } as MediaQueryListEvent));
    });
    expect(result.current).toBe(true);
  });
});
