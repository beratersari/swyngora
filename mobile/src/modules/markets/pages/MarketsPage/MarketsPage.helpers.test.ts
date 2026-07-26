import { describe, expect, it } from 'vitest';
import { mapSpotMarketToRow, pageRangeLabel } from './MarketsPage.helpers';

describe('mapSpotMarketToRow', () => {
  it('maps API row to view model', () => {
    const row = mapSpotMarketToRow({
      symbol: 'BTCUSDT',
      lastPrice: '67000',
      priceChangePercent: '1.5',
      quoteVolume: '1500000000',
      marketCapCirculating: 1_300_000_000_000,
      tags: ['Layer1_Layer2', 'pos'],
    });
    expect(row.id).toBe('BTCUSDT');
    expect(row.changeTone).toBe('success');
    expect(row.quoteVolumeLabel).toBe('1.50B');
    expect(row.tagsLabel).toContain('Layer1_Layer2');
  });
});

describe('pageRangeLabel', () => {
  it('formats range', () => {
    expect(pageRangeLabel(0, 30, 100)).toBe('1–30 of 100');
    expect(pageRangeLabel(0, 30, 0)).toBe('0 results');
  });
});
