import { describe, expect, it } from 'vitest';
import { barSide, isLevelsKind, maxSide, toLevelRows, toTimeRows } from './LiquidationBarChart.helpers';

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
    expect(levels[0]?.cumTotal).toBe(0);
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
  });
});
