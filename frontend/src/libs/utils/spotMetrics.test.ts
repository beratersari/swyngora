import { afterEach, describe, expect, it } from 'vitest';
import {
  columnSortMap,
  defaultMetricIds,
  loadMetricIds,
  metricsForSurface,
  normalizeMetricIds,
  resolveMetricDefs,
  saveMetricIds,
  SPOT_METRICS,
} from './spotMetrics';

describe('spotMetrics', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('defaults include core markets columns', () => {
    expect(defaultMetricIds('markets')).toEqual(
      expect.arrayContaining([
        'lastPrice',
        'priceChangePercent',
        'quoteVolume',
        'marketCapCirculating',
        'tradeCount',
        'tags',
      ]),
    );
  });

  it('defaults for watchlist exclude tags', () => {
    const ids = defaultMetricIds('watchlist');
    expect(ids).not.toContain('tags');
    expect(ids).toContain('lastPrice');
    expect(ids).toContain('marketCapCirculating');
  });

  it('normalize drops unknown and surface-invalid ids', () => {
    expect(normalizeMetricIds('watchlist', ['lastPrice', 'tags', 'nope', 'lastPrice'])).toEqual([
      'lastPrice',
    ]);
  });

  it('normalize falls back to defaults when empty after filter', () => {
    expect(normalizeMetricIds('markets', ['nope'])).toEqual(defaultMetricIds('markets'));
  });

  it('persist and load metric selection', () => {
    saveMetricIds('markets', ['lastPrice', 'highPrice', 'quoteVolume']);
    expect(loadMetricIds('markets')).toEqual(['lastPrice', 'highPrice', 'quoteVolume']);
  });

  it('columnSortMap includes symbol and sortable metrics', () => {
    const map = columnSortMap();
    expect(map.symbol).toBe('symbol');
    expect(map.lastPrice).toBe('lastPrice');
    expect(map.quoteVolume).toBe('quoteVolume');
  });

  it('resolveMetricDefs preserves selection order', () => {
    const defs = resolveMetricDefs('markets', ['quoteVolume', 'lastPrice']);
    expect(defs.map((d) => d.id)).toEqual(['quoteVolume', 'lastPrice']);
  });

  it('every metric has a surface and labelKey', () => {
    for (const m of SPOT_METRICS) {
      expect(m.surfaces.length).toBeGreaterThan(0);
      expect(m.labelKey.length).toBeGreaterThan(0);
      expect(m.field).toBeTruthy();
    }
    expect(metricsForSurface('markets').length).toBeGreaterThanOrEqual(6);
    expect(metricsForSurface('watchlist').length).toBeGreaterThanOrEqual(4);
  });
});
