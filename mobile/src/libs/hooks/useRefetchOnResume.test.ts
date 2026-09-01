import { describe, expect, it, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useRefetchOnResume } from './useRefetchOnResume';

describe('useRefetchOnResume', () => {
  it('refetches when active goes from false to true', () => {
    const refetch = vi.fn();
    const { rerender } = renderHook(
      ({ active }: { active: boolean }) => useRefetchOnResume(refetch, active),
      { initialProps: { active: false } },
    );
    expect(refetch).not.toHaveBeenCalled();
    rerender({ active: true });
    expect(refetch).toHaveBeenCalledTimes(1);
    rerender({ active: true });
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
