import { describe, expect, it } from 'vitest';
import { RSI_HEAT_NEUTRAL } from './constants';
import { formatRSI, nearestDot, plotInner, plotX, plotY, rsiFill, rowLabel, shouldLabelDot } from './helpers';

describe('rsiFill', () => {
  it('is green when oversold and red when overbought', () => {
    expect(rsiFill('oversold')).toBe('#16C784');
    expect(rsiFill('overbought')).toBe('#EA3943');
    expect(rsiFill('neutral')).toBe(RSI_HEAT_NEUTRAL);
    expect(rsiFill(null)).toBe(RSI_HEAT_NEUTRAL);
    expect(rsiFill('')).toBe(RSI_HEAT_NEUTRAL);
  });
});

describe('formatRSI', () => {
  it('formats one decimal or an em dash', () => {
    expect(formatRSI(42.16)).toBe('42.2');
    expect(formatRSI(undefined)).toBe('—');
    expect(formatRSI(Number.NaN)).toBe('—');
  });
});

describe('plot', () => {
  it('puts rank 1 left and RSI 100 at the top', () => {
    expect(plotX(1, 100, 1044)).toBeLessThan(plotX(100, 100, 1044));
    expect(plotY(100, 554)).toBeLessThan(plotY(0, 554));
  });

  it('finds the nearest plotted dot', () => {
    const rows = [
      { rank: 1, symbol: 'BTCUSDT', base: 'BTC', rsi: 80 },
      { rank: 2, symbol: 'ETHUSDT', base: 'ETH', rsi: 20 },
    ];
    const hit = nearestDot(rows, plotX(2, 2, 400), plotY(20, 400), 400, 400);
    expect(hit?.base).toBe('ETH');
    expect(rowLabel(rows[0]!)).toBe('BTC');
    expect(nearestDot(rows, 0, 0, 400, 400)).toBeNull();
    expect(rowLabel({ symbol: 'solusdt' })).toBe('SOLUSDT');
    expect(rowLabel({})).toBe('');
    expect(nearestDot([{ rsi: 40 }], plotX(0, 1, 400), plotY(40, 400), 400, 400)?.rsi).toBe(40);
    expect(nearestDot([{ rank: 1 }, { rsi: 40, rank: 1 }], plotX(1, 2, 400), plotY(40, 400), 400, 400)?.rsi).toBe(40);
  });
});

describe('shouldLabelDot', () => {
  it('labels top ranks, extremes, and short maps', () => {
    expect(shouldLabelDot({ rank: 1, rsi: 50 }, 100)).toBe(true);
    expect(shouldLabelDot({ rank: 40, zone: 'oversold', rsi: 25 }, 100)).toBe(true);
    expect(shouldLabelDot({ rank: 40, zone: 'overbought', rsi: 80 }, 100)).toBe(true);
    expect(shouldLabelDot({ rank: 20, rsi: 50 }, 30)).toBe(true);
    expect(shouldLabelDot({ rank: 50, rsi: 50 }, 100)).toBe(false);
    expect(shouldLabelDot({ rank: 50 }, 100)).toBe(false);
    expect(shouldLabelDot({ rsi: 50 }, 50)).toBe(false);
  });
});

describe('plotInner', () => {
  it('never returns a non-positive inner size', () => {
    const inner = plotInner(10, 10);
    expect(inner.w).toBeGreaterThanOrEqual(1);
    expect(inner.h).toBeGreaterThanOrEqual(1);
  });
});
