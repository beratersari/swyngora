import { describe, expect, it } from 'vitest';
import {
  barSide,
  formatAxisNotional,
  formatHitRate,
  frozenLevelsView,
  isLevelsKind,
  maxCum,
  maxSide,
  pickLevelsReview,
  toLevelRows,
  toTimeRows,
} from './LiquidationBarChart.helpers';

describe('LiquidationBarChart helpers', () => {
  it('maps price levels and time bars', () => {
    const levels = toLevelRows({
      kind: 'levels',
      levels: [
        { price: '100', longNotional: '10', shortNotional: '4', totalNotional: '14' },
        { price: '90', longNotional: '1', shortNotional: '8', totalNotional: '9' },
      ],
    });
    expect(levels).toHaveLength(2);
    expect(levels[0]?.longN).toBe(10);
    expect(levels[0]?.cumByLeverage).toBeDefined();
    expect(maxSide(levels)).toBe(10);
    expect(isLevelsKind({ kind: 'levels' })).toBe(true);
    const bars = toTimeRows({
      kind: 'totals',
      bars: [{ t: '2026-08-30T12:00:00.000Z', longNotional: '3', shortNotional: '2', totalNotional: '5' }],
    });
    expect(bars[0]?.label).toBe('12:00');
    expect(bars[0]?.totalN).toBe(5);
  });

  it('maps leverage slices and side vs last', () => {
    const [row] = toLevelRows({
      kind: 'levels',
      lastPrice: '100',
      levels: [
        {
          price: '110',
          longNotional: '0',
          shortNotional: '40',
          totalNotional: '40',
          cumShort: '40',
          cumTotal: '40',
          byLeverage: [{ leverage: 100, longNotional: '0', shortNotional: '40' }],
        },
      ],
    });
    expect(row?.byLeverage[0]?.leverage).toBe(100);
    expect(barSide(row!, 100)).toBe('short');
    expect(barSide({ ...row!, price: 90 }, 100)).toBe('long');
    expect(row?.cumShort).toBe(40);
    expect(maxCum([row!])).toBe(40);
  });

  it('walks cumulative leverage from last when the API omits cumByLeverage', () => {
    const rows = toLevelRows({
      kind: 'levels',
      lastPrice: '100',
      levels: [
        {
          price: '110',
          longNotional: '0',
          shortNotional: '80',
          totalNotional: '80',
          byLeverage: [{ leverage: 10, longNotional: '0', shortNotional: '80' }],
        },
        {
          price: '101',
          longNotional: '0',
          shortNotional: '40',
          totalNotional: '40',
          byLeverage: [{ leverage: 100, longNotional: '0', shortNotional: '40' }],
        },
        {
          price: '90',
          longNotional: '30',
          shortNotional: '0',
          totalNotional: '30',
          byLeverage: [{ leverage: 50, longNotional: '30', shortNotional: '0' }],
        },
      ],
    });
    const far = rows.find((r) => r.price === 110);
    const near = rows.find((r) => r.price === 101);
    expect(near?.cumTotal).toBe(40);
    expect(far?.cumTotal).toBe(120);
    expect(far?.cumByLeverage.find((s) => s.leverage === 100)?.shortN).toBe(40);
    expect(formatAxisNotional(1_250_000)).toBe('1.3M');
  });

  it('picks combined review and formats hit rate', () => {
    const venue = pickLevelsReview({
      kind: 'levels',
      exchange: 'all',
      review: {
        summary: 'ok',
        combined: {
          horizons: [{ horizon: '1h', validated: 4, hits: 2, hitRate: 0.5 }],
          byLeverage: [{ leverage: 100, side: 'short', reliability: 'fair' }],
        },
      },
    });
    expect(venue?.horizons?.[0]?.hits).toBe(2);
    expect(formatHitRate(0.5)).toBe('50%');
    expect(pickLevelsReview({ kind: 'levels' })).toBeUndefined();
  });

  it('draws a frozen estimate at that last so shorts stay shorts', () => {
    const live = {
      kind: 'levels' as const,
      lastPrice: '110',
      levels: [{ price: '101', longNotional: '40', shortNotional: '0', totalNotional: '40' }],
    };
    const view = frozenLevelsView(
      live,
      [
        {
          time: 't1',
          lastPrice: '100',
          lean: 'short',
          levels: [{ price: '101', longNotional: '0', shortNotional: '40', totalNotional: '40' }],
        },
      ],
      't1',
    );
    expect(view?.lastPrice).toBe('100');
    const [row] = toLevelRows(view);
    expect(barSide(row!, 100)).toBe('short');
    expect(barSide(toLevelRows(live)[0]!, 110)).toBe('long');
  });
});
