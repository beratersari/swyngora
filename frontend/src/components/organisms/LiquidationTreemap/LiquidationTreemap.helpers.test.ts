import { describe, expect, it } from 'vitest';
import {
  coinBase,
  coinLongShare,
  dominanceFill,
  toTreemapTiles,
} from './LiquidationTreemap.helpers';
import { LIQ_TREEMAP_NEUTRAL } from './LiquidationTreemap.constants';

describe('LiquidationTreemap helpers', () => {
  it('sizes tiles by notional and colors by long share', () => {
    const tiles = toTreemapTiles(
      [
        { symbol: 'BTCUSDT', base: 'BTC', totalNotional: '900', longNotional: '800', shortNotional: '100' },
        { symbol: 'ETHUSDT', base: 'ETH', totalNotional: '100', longNotional: '10', shortNotional: '90' },
      ],
      400,
      300,
    );
    expect(tiles).toHaveLength(2);
    expect(tiles[0]?.symbol).toBe('BTCUSDT');
    expect(tiles[0]!.w * tiles[0]!.h).toBeGreaterThan(tiles[1]!.w * tiles[1]!.h);
    expect(dominanceFill(0.9)).not.toBe(LIQ_TREEMAP_NEUTRAL);
    expect(dominanceFill(0.5)).toBe(LIQ_TREEMAP_NEUTRAL);
    expect(coinLongShare({ longNotional: '3', shortNotional: '1' })).toBeCloseTo(0.75);
    expect(coinBase('BTCUSDT', '')).toBe('BTC');
  });
});
