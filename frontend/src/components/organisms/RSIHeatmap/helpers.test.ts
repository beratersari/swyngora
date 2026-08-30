import { describe, expect, it } from 'vitest';
import { RSI_HEAT_NEUTRAL } from './constants';
import {
  clientToPlot,
  formatRSI,
  nearestDot,
  plotInner,
  plottedRows,
  plotX,
  plotY,
  pointerToPlot,
  rsiFill,
  rowKey,
  rowLabel,
  shouldLabelDot,
  tipOrigin,
} from './helpers';

describe('rsiFill', () => {
  it('interpolates green through ash to red by RSI', () => {
    expect(rsiFill(0)).toBe('#0EA872');
    expect(rsiFill(50)).toBe(RSI_HEAT_NEUTRAL);
    expect(rsiFill(100)).toBe('#F6465D');
    const low = rsiFill(35);
    const mid = rsiFill(50);
    const high = rsiFill(65);
    expect(low).not.toBe(mid);
    expect(high).not.toBe(mid);
    expect(low.toLowerCase()).not.toBe(high.toLowerCase());
    expect(rsiFill(undefined, 'oversold')).toBe('#3FD39A');
    expect(rsiFill(undefined, 'overbought')).toBe('#F07B84');
    expect(rsiFill(null)).toBe(RSI_HEAT_NEUTRAL);
    expect(rsiFill(Number.NaN)).toBe(RSI_HEAT_NEUTRAL);
    expect(rsiFill(-10)).toBe('#0EA872');
    expect(rsiFill(120)).toBe('#F6465D');
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
    const midPlot = plotInner(400, 400);
    expect(
      nearestDot(rows, midPlot.x + midPlot.w / 2, midPlot.y + midPlot.h / 2, 400, 400),
    ).toBeNull();
    expect(rowLabel({ symbol: 'solusdt' })).toBe('SOLUSDT');
    expect(rowLabel({})).toBe('');
    expect(nearestDot([{ rsi: 40 }], plotX(1, 1, 400), plotY(40, 400), 400, 400)?.rsi).toBe(40);
    expect(nearestDot([{ rank: 1 }, { rsi: 40, rank: 1 }], plotX(2, 2, 400), plotY(40, 400), 400, 400)?.rsi).toBe(40);
    const onEth = { x: plotX(2, 2, 400), y: plotY(20, 400) };
    expect(nearestDot(rows, onEth.x + 19, onEth.y, 400, 400)).toBeNull();
  });
});

describe('plottedRows', () => {
  it('keeps only finite RSI and sorts by rank', () => {
    const rows = plottedRows([
      { rank: 40, symbol: 'SHIBUSDT', rsi: 51 },
      { rank: 1, symbol: 'BTCUSDT', rsi: 28 },
      { rank: 2, symbol: 'ETHUSDT', rsi: null },
      { symbol: 'BAD', rsi: Number.NaN },
    ]);
    expect(rows.map((row) => row.symbol)).toEqual(['BTCUSDT', 'SHIBUSDT']);
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

describe('clientToPlot', () => {
  it('maps a stretched SVG box back into viewBox units', () => {
    const p = clientToPlot({ left: 10, top: 20, width: 200, height: 100 }, 110, 70, 400, 200);
    expect(p.x).toBe(200);
    expect(p.y).toBe(100);
  });
});

describe('pointerToPlot', () => {
  it('falls back to the CSS box when the SVG has no CTM', () => {
    const svg = {
      getScreenCTM: () => null,
      getBoundingClientRect: () => ({ left: 10, top: 20, width: 200, height: 100 }),
    } as unknown as SVGSVGElement;
    const p = pointerToPlot(svg, 110, 70, 400, 200);
    expect(p.x).toBe(200);
    expect(p.y).toBe(100);
  });
});

describe('tipOrigin', () => {
  it('flips the card so it stays inside the frame', () => {
    const near = tipOrigin(10, 10, 400, 300);
    expect(near.x).toBeGreaterThan(10);
    expect(near.y).toBeGreaterThan(10);
    const edge = tipOrigin(390, 280, 400, 300);
    expect(edge.x + 200).toBeLessThanOrEqual(400);
    expect(edge.y + 78).toBeLessThanOrEqual(300);
  });
});

describe('rowKey', () => {
  it('prefers symbol then base', () => {
    expect(rowKey({ symbol: 'btcusdt', base: 'BTC' })).toBe('BTCUSDT');
    expect(rowKey({ base: 'eth' })).toBe('ETH');
  });
});

describe('plotInner', () => {
  it('never returns a non-positive inner size', () => {
    const inner = plotInner(10, 10);
    expect(inner.w).toBeGreaterThanOrEqual(1);
    expect(inner.h).toBeGreaterThanOrEqual(1);
  });
});
