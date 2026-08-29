import { describe, expect, it } from 'vitest';
import { HEATMAP_MAX_TILES, HEATMAP_NEUTRAL, HEATMAP_TILE_INK } from './PriceChangeHeatmap.constants';
import {
  baseSymbol,
  changeFill,
  hoverCardOrigin,
  itemWeight,
  parseNum,
  tileDensity,
  tileInk,
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
    expect(baseSymbol('BTCUSDT', 'BTC')).toBe('BTC');
    expect(baseSymbol('ETH-USD', 'ETH')).toBe('ETH');
    expect(baseSymbol('BTCUSDT')).toBe('BTCUSDT');
  });

  it('uses a discrete CoinMarketCap-style scale', () => {
    expect(changeFill(0)).toBe(HEATMAP_NEUTRAL);
    expect(changeFill(0.02)).toBe(HEATMAP_NEUTRAL);
    expect(changeFill(0.3)).not.toBe(HEATMAP_NEUTRAL);
    expect(changeFill(8)).toMatch(/^#/);
    expect(changeFill(-8)).toMatch(/^#/);
    expect(changeFill(5)).not.toBe(changeFill(-5));
    expect(changeFill(12)).toBe(changeFill(20));
    expect(tileInk(HEATMAP_NEUTRAL)).toBe('#0D1421');
    expect(tileInk(changeFill(8))).toBe(HEATMAP_TILE_INK);
  });

  it('classifies tile density for labels', () => {
    expect(tileDensity(120, 80)).toBe('full');
    expect(tileDensity(56, 36)).toBe('compact');
    expect(tileDensity(36, 24)).toBe('ticker');
    expect(tileDensity(24, 16)).toBe('micro');
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

  it('keeps the hover card inside the map frame', () => {
    const nearEdge = hoverCardOrigin(780, 360, 800, 400);
    expect(nearEdge.x + 210).toBeLessThanOrEqual(800);
    expect(nearEdge.y).toBeGreaterThanOrEqual(10);
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
