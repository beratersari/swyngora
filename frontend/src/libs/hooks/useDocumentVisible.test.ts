import { describe, expect, it, vi, afterEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useDocumentVisible } from './useDocumentVisible';

describe('useDocumentVisible', () => {
  const original = document.visibilityState;

  afterEach(() => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => original,
    });
  });

  it('starts from document.visibilityState', () => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => 'visible',
    });
    const { result } = renderHook(() => useDocumentVisible());
    expect(result.current).toBe(true);
  });

  it('updates on visibilitychange and cleans up listener', () => {
    let state: DocumentVisibilityState = 'visible';
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => state,
    });
    const addSpy = vi.spyOn(document, 'addEventListener');
    const removeSpy = vi.spyOn(document, 'removeEventListener');
    const { result, unmount } = renderHook(() => useDocumentVisible());
    expect(result.current).toBe(true);

    state = 'hidden';
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });
    expect(result.current).toBe(false);

    unmount();
    expect(removeSpy).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
    addSpy.mockRestore();
    removeSpy.mockRestore();
  });
});
