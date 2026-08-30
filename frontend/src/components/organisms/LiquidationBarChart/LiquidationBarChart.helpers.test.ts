import { describe, expect, it } from 'vitest';
import { isLevelsKind, maxSide, toLevelRows, toTimeRows } from './LiquidationBarChart.helpers';

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
    expect(maxSide(levels)).toBe(10);
    expect(isLevelsKind({ kind: 'levels' })).toBe(true);
    const bars = toTimeRows({
      kind: 'totals',
      bars: [{ t: '2026-08-30T12:00:00.000Z', longNotional: '3', shortNotional: '2', totalNotional: '5' }],
    });
    expect(bars[0]?.label).toBe('12:00');
    expect(bars[0]?.totalN).toBe(5);
  });
});
