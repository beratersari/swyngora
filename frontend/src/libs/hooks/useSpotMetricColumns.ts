import { useCallback, useMemo, useState } from 'react';
import {
  defaultMetricIds,
  loadMetricIds,
  metricsForSurface,
  normalizeMetricIds,
  resolveMetricDefs,
  saveMetricIds,
  type SpotMetricDef,
  type SpotMetricId,
  type SpotMetricSurface,
} from '@/libs/utils/spotMetrics';

export type UseSpotMetricColumnsResult = {
  surface: SpotMetricSurface;
  /** Currently visible metric ids (ordered). */
  metricIds: SpotMetricId[];
  /** Resolved defs for rendering columns. */
  metrics: SpotMetricDef[];
  /** All metrics available on this surface (for the picker). */
  available: SpotMetricDef[];
  setMetricIds: (ids: SpotMetricId[]) => void;
  resetToDefaults: () => void;
};

/**
 * Persistable metric column selection for Markets / Watchlist.
 */
export function useSpotMetricColumns(surface: SpotMetricSurface): UseSpotMetricColumnsResult {
  const [metricIds, setMetricIdsState] = useState<SpotMetricId[]>(() => loadMetricIds(surface));

  const setMetricIds = useCallback(
    (ids: SpotMetricId[]) => {
      const next = normalizeMetricIds(surface, ids);
      setMetricIdsState(next);
      saveMetricIds(surface, next);
    },
    [surface],
  );

  const resetToDefaults = useCallback(() => {
    const next = defaultMetricIds(surface);
    setMetricIdsState(next);
    saveMetricIds(surface, next);
  }, [surface]);

  const available = useMemo(() => metricsForSurface(surface), [surface]);
  const metrics = useMemo(() => resolveMetricDefs(surface, metricIds), [surface, metricIds]);

  return {
    surface,
    metricIds,
    metrics,
    available,
    setMetricIds,
    resetToDefaults,
  };
}
