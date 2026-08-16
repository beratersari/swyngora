import { describe, expect, it } from 'vitest';
import type { SpotOrderBook } from '@/libs/api';
import {
  buildDepthLayout,
  buildDepthSeries,
  formatDepthAmount,
  hitTestDepth,
  parseDepthNumber,
  priceToX,
} from './OrderDepthChart.helpers';

const book: SpotOrderBook = {
  lastPrice: '100',
  bestBid: '99',
  bestAsk: '101',
  bids: [
    { price: '99', quantity: '1', cumulative: '1', notional: '99', cumulativeNotional: '99' },
    { price: '98', quantity: '2', cumulative: '3', notional: '196', cumulativeNotional: '295' },
  ],
  asks: [
    { price: '101', quantity: '1', cumulative: '1', notional: '101', cumulativeNotional: '101' },
    { price: '102', quantity: '4', cumulative: '5', notional: '408', cumulativeNotional: '509' },
  ],
};

describe('OrderDepthChart helpers', () => {
  it('parses finite numbers only', () => {
    expect(parseDepthNumber('12.5')).toBe(12.5);
    expect(parseDepthNumber('')).toBeNaN();
    expect(parseDepthNumber('nope')).toBeNaN();
  });

  it('builds bid/ask series with cumulative growing away from mid', () => {
    const series = buildDepthSeries(book, 'base');
    expect(series).not.toBeNull();
    expect(series?.mid).toBe(100);
    expect(series?.bids.map((p) => p.price)).toEqual([98, 99]);
    expect(series?.bids.map((p) => p.depth)).toEqual([3, 1]);
    expect(series?.asks.map((p) => p.price)).toEqual([101, 102]);
    expect(series?.asks.map((p) => p.depth)).toEqual([1, 5]);
    expect(series?.maxDepth).toBe(5);
  });

  it('uses cumulative notional when metric is notional', () => {
    const series = buildDepthSeries(book, 'notional');
    expect(series?.asks[1]?.depth).toBe(509);
    expect(series?.maxDepth).toBe(509);
  });

  it('returns null when the book has no usable depth', () => {
    expect(buildDepthSeries({ lastPrice: '1', bids: [], asks: [] }, 'base')).toBeNull();
    expect(buildDepthSeries(undefined, 'base')).toBeNull();
  });

  it('hit-tests the bid side left of mid', () => {
    const series = buildDepthSeries(book, 'base')!;
    const layout = buildDepthLayout(series, 400, 200);
    const midX = priceToX(100, layout);
    const hit = hitTestDepth(midX - 20, layout.plotY + 10, series, layout);
    expect(hit?.side).toBe('bid');
    expect(hit?.price).toBeGreaterThan(0);
  });

  it('formats compact amounts', () => {
    expect(formatDepthAmount(1_500_000)).toBe('1.50M');
    expect(formatDepthAmount(0)).toBe('—');
  });
});
