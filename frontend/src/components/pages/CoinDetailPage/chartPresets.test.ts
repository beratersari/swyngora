import { describe, expect, it } from 'vitest';
import {
  absorptionDeskNote,
  chartPresetLayers,
  chartWallNotes,
  liquidationLevelReview,
  matchingChartPreset,
  perpBidShareNote,
  printSizeDeskNote,
} from './chartPresets';

describe('chart presets', () => {
  it('gives each desk its own layers and matches them back', () => {
    expect(chartPresetLayers('positioning')).toMatchObject({
      showOi: true,
      showFunding: true,
      showCrowd: true,
      showOpenLevels: true,
      showLiqLevels: true,
      showBook: false,
      showFootprint: false,
      showLiqMap: false,
      showDelta: false,
      showCvd: false,
    });
    expect(chartPresetLayers('orderFlow')).toMatchObject({
      showDelta: true,
      showCvd: true,
      showFootprint: true,
      showProfile: true,
      showSweeps: true,
      showBots: true,
      showOi: false,
      showBook: false,
    });
    expect(chartPresetLayers('liquidity')).toEqual({
      ...chartPresetLayers('orderFlow'),
      showDelta: false,
      showCvd: false,
      showFootprint: false,
      showProfile: false,
      showSweeps: false,
      showBots: false,
      showBook: true,
    });
    expect(matchingChartPreset(chartPresetLayers('all'))).toBe('all');
    expect(matchingChartPreset(chartPresetLayers('positioning'))).toBe('positioning');
    expect(matchingChartPreset({ ...chartPresetLayers('orderFlow'), showBook: true })).toBeNull();
  });

  it('puts later-print review on the page venue and the strongest leverage', () => {
    const report = {
      review: {
        binance: {
          byLeverage: [
            { leverage: 25, side: 'long', horizon: '1h', hitRate: 0.4, sizeScale: 0.5 },
            { leverage: 100, side: 'long', horizon: '4h', hitRate: 0.2, sizeScale: 0.3 },
            { leverage: 100, side: 'long', horizon: '1h', hitRate: 0.6, sizeScale: 0.4 },
          ],
        },
        bybit: {
          byLeverage: [{ leverage: 100, side: 'long', horizon: '1h', hitRate: 0.9, sizeScale: 1 }],
        },
      },
    };
    expect(liquidationLevelReview(report, 'binance', 'long', [100, 25])).toEqual({
      leverage: 100,
      hitRate: 0.6,
      sizeScale: 0.4,
    });
    expect(liquidationLevelReview(report, 'bybit', 'long', [100])).toMatchObject({ hitRate: 0.9 });
    expect(liquidationLevelReview(report, 'okx', 'long', [100])).toBeNull();
  });

  it('uses combined print size and absorption only when that read is complete', () => {
    const printSize = {
      combined: {
        windows: [{ window: '15m', complete: false, summary: 'combined should stay off' }],
      },
      venues: [
        {
          exchange: 'binance',
          windows: [{ window: '15m', complete: true, summary: 'Last 15m: buy prints averaged 2K USDT.' }],
        },
        { exchange: 'okx', error: 'quiet', windows: [{ window: '15m', complete: false }] },
      ],
    };
    expect(printSizeDeskNote(printSize, 'binance')).toEqual({
      text: 'Last 15m: buy prints averaged 2K USDT.',
      exchange: 'binance',
      withheld: ['okx'],
    });
    expect(
      printSizeDeskNote(
        {
          combined: { windows: [{ window: '15m', complete: true, summary: 'Last 15m: stable.' }] },
          venues: [],
        },
        'binance',
      )?.exchange,
    ).toBe('combined');

    expect(
      absorptionDeskNote(
        {
          combined: { complete: false, summary: 'partial overlap' },
          venues: [
            { exchange: 'binance', complete: true, summary: 'bids absorbing sells' },
            { exchange: 'bybit', complete: false, summary: 'short' },
          ],
        },
        'binance',
      ),
    ).toEqual({
      text: 'bids absorbing sells',
      exchange: 'binance',
      withheld: ['bybit'],
    });
    expect(
      absorptionDeskNote(
        { combined: { complete: true, summary: 'Futures bids absorbing sells' }, venues: [] },
        'binance',
      )?.exchange,
    ).toBe('combined');
  });

  it('withholds combined perp bid share when a venue does not cover the band', () => {
    const full = perpBidShareNote({
      rangePct: '1',
      venues: [
        head('binance', '60', '40'),
        head('bybit', '50', '50'),
        head('okx', '70', '30'),
      ],
      combined: head('all', '180', '120'),
    });
    expect(full?.definite).toBe(true);
    expect(full?.bidPct).toBeCloseTo(60);
    expect(full?.missing).toEqual([]);

    const missing = perpBidShareNote({
      rangePct: '1',
      venues: [head('binance', '60', '40'), { exchange: 'bybit', error: 'stale' }],
      combined: head('all', '60', '40'),
    });
    expect(missing?.definite).toBe(false);
    expect(missing?.bidPct).toBeNull();
    expect(missing?.missing).toEqual(['bybit', 'okx']);
    expect(missing?.recorded).toEqual([{ exchange: 'binance', bidPct: 60 }]);
  });

  it('keeps the three largest live walls', () => {
    expect(
      chartWallNotes([
        { side: 'bid', price: '100', notional: '10', behavior: 'short' },
        { side: 'ask', price: '110', notional: '90', behavior: 'suspicious' },
        { side: 'bid', price: '99', notional: '40', behavior: 'persistent' },
        { side: 'ask', price: '120', notional: '30', behavior: 'short' },
      ]),
    ).toEqual([
      { side: 'ask', price: '110', behavior: 'suspicious' },
      { side: 'bid', price: '99', behavior: 'persistent' },
      { side: 'ask', price: '120', behavior: 'short' },
    ]);
    expect(chartWallNotes([])).toEqual([]);
  });
});

function head(exchange: string, below: string, above: string) {
  return {
    exchange,
    headline: {
      rangePct: '1',
      complete: true,
      below: { notional: below },
      above: { notional: above },
    },
  };
}
