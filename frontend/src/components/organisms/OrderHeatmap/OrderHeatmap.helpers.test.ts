import { describe, expect, it } from 'vitest';
import {
  buildLayout,
  columnRects,
  formatCollectedSpan,
  formatHeatNotional,
  formatHeatPrice,
  hitTest,
  intensityFromNotional,
  parseBookNumber,
  priceToY,
  depthColor,
} from './OrderHeatmap.helpers';
import type { OrderHeatmapData } from './OrderHeatmap.types';

const sample: OrderHeatmapData = {
  groupSize: '1',
  windowSeconds: 600,
  to: '2026-08-16T12:10:00.000Z',
  columns: [
    {
      t: '2026-08-16T12:09:00.000Z',
      mid: '100',
      bids: [{ price: '99', notional: '5000', isWall: true }],
      asks: [{ price: '101', notional: '2000' }],
    },
    {
      t: '2026-08-16T12:10:00.000Z',
      mid: '100.5',
      bids: [{ price: '99', notional: '8000' }],
      asks: [{ price: '102', notional: '1000' }],
    },
  ],
};

describe('OrderHeatmap helpers', () => {
  it('parses numbers and log intensity', () => {
    expect(parseBookNumber('12.5')).toBe(12.5);
    expect(parseBookNumber('x')).toBeNaN();
    expect(intensityFromNotional(0, 100)).toBe(0);
    expect(intensityFromNotional(100, 100)).toBe(1);
    expect(intensityFromNotional(9, 99)).toBeGreaterThan(0);
    expect(intensityFromNotional(9, 99)).toBeLessThan(1);
  });

  it('uses desk green/red depth colors on a light bed', () => {
    expect(depthColor(null, 0)).toBe('rgb(248, 250, 253)');
    expect(depthColor('bid', 1)).toBe('rgb(22, 199, 132)');
    expect(depthColor('ask', 1)).toBe('rgb(234, 57, 67)');
    expect(depthColor('bid', 0.4)).not.toBe(depthColor('bid', 0.9));
  });

  it('lets one snapshot fill the plot instead of a sliver', () => {
    const one: OrderHeatmapData = { ...sample, columns: [sample.columns![1]!] };
    const layout = buildLayout(one, 640, 360);
    expect(layout).not.toBeNull();
    expect(layout?.rects).toHaveLength(1);
    expect(layout?.rects[0]?.isCob).toBe(true);
    expect(layout?.rects[0]?.w ?? 0).toBeGreaterThan(400);
  });

  it('keeps history on the left and a wide current book on the right', () => {
    const layout = buildLayout(sample, 640, 360);
    expect(layout).not.toBeNull();
    if (!layout) return;
    expect(layout.rects).toHaveLength(2);
    expect(layout.rects[0]?.isCob).toBe(false);
    expect(layout.rects[1]?.isCob).toBe(true);
    expect(layout.rects[1]!.x).toBeGreaterThan(layout.rects[0]!.x);
    expect(layout.rects[1]!.w).toBeGreaterThanOrEqual(88);
    const y = priceToY(99, layout);
    const hit = hitTest(layout.rects[1]!.x + 8, y, sample, layout);
    expect(hit?.bid).toBe(8000);
  });

  it('formats the collected tape span', () => {
    expect(
      formatCollectedSpan('2026-08-16T12:00:00.000Z', '2026-08-16T12:10:00.000Z'),
    ).toBe('10m');
    expect(
      formatCollectedSpan('2026-08-16T12:00:00.000Z', '2026-08-16T12:15:00.000Z'),
    ).toBe('15m');
  });

  it('formats compact notionals and grouped prices', () => {
    expect(formatHeatNotional(1_250_000)).toBe('1.25M');
    expect(formatHeatNotional(12_000)).toBe('12.0K');
    expect(formatHeatPrice(100.25, 0.1)).toBe('100.3');
    expect(formatHeatPrice(100, 1)).toBe('100');
  });

  it('splits many columns without collapsing to 2px', () => {
    const rects = columnRects(20, 80, 520, Array.from({ length: 20 }, (_, i) => i));
    expect(rects).toHaveLength(20);
    expect(Math.min(...rects.map((r) => r.w))).toBeGreaterThan(8);
  });
});
