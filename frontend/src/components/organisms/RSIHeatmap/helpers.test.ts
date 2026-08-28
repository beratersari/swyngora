import { describe, expect, it } from 'vitest';
import { RSI_HEAT_NEUTRAL } from './constants';
import { formatRSI, nearestDot, plotX, plotY, rsiFill, rowLabel } from './helpers';

describe('rsiFill', () => {
  it('is green when oversold and red when overbought', () => {
    expect(rsiFill(18)).toBe('#0EA872');
    expect(rsiFill(28)).toBe('#16C784');
    expect(rsiFill(50)).toBe(RSI_HEAT_NEUTRAL);
    expect(rsiFill(75)).toBe('#EA3943');
    expect(rsiFill(null)).toBe(RSI_HEAT_NEUTRAL);
  });
});

describe('formatRSI', () => {
  it('formats one decimal or an em dash', () => {
    expect(formatRSI(42.16)).toBe('42.2');
    expect(formatRSI(undefined)).toBe('—');
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
  });
});
