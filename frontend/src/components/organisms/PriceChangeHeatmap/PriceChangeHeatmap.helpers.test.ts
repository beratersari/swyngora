import { describe, expect, it } from 'vitest';
import { HEATMAP_MAX_TILES, HEATMAP_NEUTRAL, HEATMAP_SCALE } from './PriceChangeHeatmap.constants';
import {
  baseSymbol,
  changeFill,
  itemWeight,
  parseNum,
  tileDensity,
  toHeatmapTiles,
} from './PriceChangeHeatmap.helpers';

describe('PriceChangeHeatmap helpers', () => {
  it('parses percents and weights', () => {
    expect(parseNum('+3.25')).toBeCloseTo(3.25);
    expect(
      itemWeight({ symbol: 'BTCUSDT', exchange: 'binance', quoteVolume: '1000' }, 'quoteVolume'),
    ).toBe(1000);
    expect(
      itemWeight({ symbol: 'ETHUSDT', exchange: 'binance', marketCapCirculating: 50 }, 'marketCap'),
    ).toBe(50);
  });

  it('strips quote suffix for labels', () => {
    expect(baseSymbol('BTCUSDT')).toBe('BTC');
    expect(baseSymbol('ETH-USD')).toBe('ETH');
  });

  it('uses a translucent discrete scale', () => {
    expect(changeFill(0)).toBe(HEATMAP_NEUTRAL);
    expect(changeFill(0.2)).toBe(HEATMAP_NEUTRAL);
    expect(changeFill(8)).toMatch(/^#/);
    expect(changeFill(-8)).toMatch(/^#/);
    expect(changeFill(5)).not.toBe(changeFill(-5));
  });

  it('classifies tile density for labels', () => {
    expect(tileDensity(120, 80)).toBe('full');
    expect(tileDensity(60, 40)).toBe('compact');
    expect(tileDensity(40, 24)).toBe('ticker');
    expect(tileDensity(28, 16)).toBe('micro');
  });

  it('lays out square-ish tiles that cover the canvas', () => {
    const items = Array.from({ length: 8 }, (_, i) => ({
      symbol: `C${i}USDT`,
      exchange: 'binance',
      priceChangePercent: String(i - 3),
      quoteVolume: String((i + 1) * 100),
    }));
    const tiles = toHeatmapTiles(items, 'quoteVolume', 800, 400);
    expect(tiles).toHaveLength(8);
    tiles.forEach((t) => {
      expect(t.w).toBeGreaterThan(0);
      expect(t.h).toBeGreaterThan(0);
    });
    const maxAspect = Math.max(...tiles.map((t) => Math.max(t.w / t.h, t.h / t.w)));
    // Squarify should avoid the extreme strips of slice-and-dice.
    expect(maxAspect).toBeLessThan(12);
  });

  it('caps tile count', () => {
    const items = Array.from({ length: HEATMAP_MAX_TILES + 40 }, (_, i) => ({
      symbol: `S${i}USDT`,
      exchange: 'binance',
      quoteVolume: String(1000 - i),
    }));
    expect(toHeatmapTiles(items, 'quoteVolume', 400, 400)).toHaveLength(HEATMAP_MAX_TILES);
  });
});
