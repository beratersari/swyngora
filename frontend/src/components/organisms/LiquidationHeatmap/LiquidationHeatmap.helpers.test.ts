import { describe, expect, it } from 'vitest';
import {
  buildLayout,
  cellValue,
  formatLiqNotional,
  formatLiqTime,
  hasHeatTape,
  heatColor,
  hitTest,
  intensityFromNotional,
  formatHitRate,
  formatLookahead,
  formatSignalArea,
  formatSignalHorizon,
  newestSignals,
  pickGrid,
  pickSignalHorizon,
  pickMatrix,
  pickReview,
} from './LiquidationHeatmap.helpers';
import type { LiquidationHeatmapData } from './LiquidationHeatmap.types';

const sample: LiquidationHeatmapData = {
  symbol: 'BTCUSDT',
  range: '24h',
  stepSec: 1800,
  priceMin: 90_000,
  priceMax: 110_000,
  priceStep: 1000,
  prices: [109_500, 108_500, 107_500],
  times: ['2026-08-29T12:00:00Z', '2026-08-29T12:30:00Z'],
  binance: {
    exchange: 'binance',
    longs: [
      [0, 10, 0],
      [0, 0, 80],
    ],
    shorts: [
      [20, 0, 0],
      [40, 0, 0],
    ],
    totals: [
      [20, 10, 0],
      [40, 0, 80],
    ],
    maxIntensity: 80,
    coverage: 1,
    columnsWithOi: 2,
  },
  bybit: {
    exchange: 'bybit',
    longs: [
      [0, 0, 0],
      [0, 0, 0],
    ],
    shorts: [
      [0, 0, 0],
      [0, 0, 0],
    ],
    totals: [
      [0, 0, 0],
      [0, 0, 0],
    ],
    maxIntensity: 0,
    coverage: 0,
    columnsWithOi: 0,
  },
  combined: {
    exchange: 'combined',
    longs: [
      [0, 10, 0],
      [0, 0, 80],
    ],
    shorts: [
      [20, 0, 0],
      [40, 0, 0],
    ],
    totals: [
      [20, 10, 0],
      [40, 0, 80],
    ],
    maxIntensity: 80,
    coverage: 1,
    columnsWithOi: 2,
  },
  review: {
    hotFrac: 0.6,
    binance: {
      exchange: 'binance',
      horizons: [{ horizon: '1h', signals: 6, validated: 4, missing: 2, hits: 2, falseSignals: 2, hitRate: 0.5 }],
    },
    bybit: { exchange: 'bybit', horizons: [] },
    combined: {
      exchange: 'combined',
      horizons: [{ horizon: '1h', signals: 4, hits: 3, falseSignals: 1, hitRate: 0.75 }],
    },
  },
};

describe('LiquidationHeatmap helpers', () => {
  it('picks venue grids and side matrices', () => {
    expect(pickGrid(sample, 'binance')?.exchange).toBe('binance');
    expect(pickGrid(sample, 'combined')?.maxIntensity).toBe(80);
    expect(pickMatrix(sample.binance, 'longs')[1]?.[2]).toBe(80);
    expect(pickMatrix(sample.binance, 'shorts')[0]?.[0]).toBe(20);
    expect(cellValue(sample.binance?.totals ?? [], 1, 2)).toBe(80);
  });

  it('uses a log intensity scale and CoinGlass-like colors', () => {
    expect(intensityFromNotional(0, 80)).toBe(0);
    expect(intensityFromNotional(80, 80)).toBe(1);
    expect(intensityFromNotional(8, 80)).toBeGreaterThan(0);
    expect(intensityFromNotional(8, 80)).toBeLessThan(1);
    expect(heatColor(0, 'totals')).toMatch(/rgb\(/);
    expect(heatColor(1, 'longs')).not.toBe(heatColor(1, 'shorts'));
  });

  it('builds a plot layout and hit-tests cells', () => {
    expect(hasHeatTape(sample)).toBe(true);
    const layout = buildLayout(sample, 400, 300);
    expect(layout?.nT).toBe(2);
    expect(layout?.nP).toBe(3);
    const hover = hitTest(
      (layout?.plotX ?? 0) + (layout?.cellW ?? 0) * 1.5,
      (layout?.plotY ?? 0) + (layout?.cellH ?? 0) * 2.5,
      sample,
      layout!,
      'binance',
    );
    expect(hover?.longs).toBe(80);
    expect(hover?.totals).toBe(80);
  });

  it('picks venue review and formats rates', () => {
    expect(pickReview(sample, 'binance')?.horizons?.[0]?.hits).toBe(2);
    expect(pickReview(sample, 'combined')?.horizons?.[0]?.hitRate).toBe(0.75);
    expect(formatHitRate(0.5)).toBe('50%');
    expect(formatLookahead(2700)).toBe('45m');
    expect(formatLookahead(0)).toBe('—');
    expect(formatSignalArea(99_000, 101_000, 100)).toMatch(/99/);
    expect(formatSignalHorizon({ status: 'hit', timeToHitSec: 1800 })).toMatch(/Hit/);
    expect(formatSignalHorizon({ status: 'price_gap', priceCoveredSec: 1800, horizonSec: 3600 })).toMatch(
      /Price gap/,
    );
    const newest = newestSignals([
      { time: '2026-08-29T12:00:00Z', priceLo: 1, priceHi: 2 },
      { time: '2026-08-29T13:00:00Z', priceLo: 3, priceHi: 4 },
    ]);
    expect(newest[0]?.priceLo).toBe(3);
    expect(pickSignalHorizon({ horizons: [{ horizon: '4h', status: 'miss' }] }, '4h')?.status).toBe(
      'miss',
    );
  });

  it('formats notional and time', () => {
    expect(formatLiqNotional(2_500_000)).toBe('2.50M');
    expect(formatLiqNotional(0)).toBe('—');
    expect(formatLiqTime(Date.parse('2026-08-29T12:30:00Z'), '24h')).toMatch(/\d{2}:\d{2}/);
    expect(formatLiqTime(Date.parse('2026-08-29T12:30:00Z'), '7d')).toMatch(/\d{2}-\d{2}/);
  });
});
