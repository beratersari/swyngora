import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { defaultMetricIds } from '@/libs/utils/spotMetrics';
import { useSpotMetricColumns } from './useSpotMetricColumns';

describe('useSpotMetricColumns', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('starts with surface defaults', () => {
    const { result } = renderHook(() => useSpotMetricColumns('watchlist'));
    expect(result.current.metricIds).toEqual(defaultMetricIds('watchlist'));
    expect(result.current.metrics.map((m) => m.id)).toEqual(result.current.metricIds);
  });

  it('persists selection changes', () => {
    const { result } = renderHook(() => useSpotMetricColumns('markets'));
    act(() => {
      result.current.setMetricIds(['lastPrice', 'highPrice']);
    });
    expect(result.current.metricIds).toEqual(['lastPrice', 'highPrice']);
    expect(JSON.parse(localStorage.getItem('swyngora.metricColumns.markets')!)).toEqual([
      'lastPrice',
      'highPrice',
    ]);
  });

  it('resetToDefaults restores catalog defaults', () => {
    const { result } = renderHook(() => useSpotMetricColumns('markets'));
    act(() => {
      result.current.setMetricIds(['lastPrice']);
    });
    act(() => {
      result.current.resetToDefaults();
    });
    expect(result.current.metricIds).toEqual(defaultMetricIds('markets'));
  });
});
