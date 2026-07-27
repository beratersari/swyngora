import { describe, expect, it } from 'vitest';
import {
  buildFavoritesOnlyRows,
  favoritesEmptyMessage,
  favoritesForExchange,
  favoritesSummaryLabel,
  mapSpotMarketToRow,
  pageRangeLabel,
} from './MarketsPage.helpers';
import type { MarketRowViewModel } from '@/components/organisms/MarketRow';

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

describe('favoritesForExchange', () => {
  it('filters case-insensitively', () => {
    const items = [
      { exchange: 'binance', symbol: 'BTCUSDT' },
      { exchange: 'Coinbase', symbol: 'ETHUSD' },
      { exchange: 'BINANCE', symbol: 'SOLUSDT' },
    ];
    expect(favoritesForExchange(items, 'binance')).toHaveLength(2);
    expect(favoritesForExchange(items, 'coinbase')).toHaveLength(1);
  });
});

describe('buildFavoritesOnlyRows', () => {
  const loaded: MarketRowViewModel[] = [
    {
      id: 'BTCUSDT',
      symbol: 'BTCUSDT',
      lastPriceLabel: '1',
      changePercentLabel: '+1%',
      changeTone: 'success',
      quoteVolumeLabel: '1B',
      marketCapLabel: '1T',
      tagsLabel: '',
    },
  ];

  it('reuses loaded rows and placeholders for missing', () => {
    const rows = buildFavoritesOnlyRows(
      [
        { exchange: 'binance', symbol: 'BTCUSDT' },
        { exchange: 'binance', symbol: 'ETHUSDT' },
      ],
      loaded,
    );
    expect(rows).toHaveLength(2);
    expect(rows[0].lastPriceLabel).toBe('1');
    expect(rows[1].symbol).toBe('ETHUSDT');
    expect(rows[1].tagsLabel).toBe('Favorite');
    expect(rows[1].lastPriceLabel).toBe('—');
  });
});

describe('favoritesEmptyMessage / summary', () => {
  it('empty message only when favorites-only and no rows', () => {
    expect(favoritesEmptyMessage(true, null, false, 0)).toMatch(/No favorites/);
    expect(favoritesEmptyMessage(false, null, false, 0)).toBeNull();
    expect(favoritesEmptyMessage(true, 'err', false, 0)).toBeNull();
    expect(favoritesEmptyMessage(true, null, true, 0)).toBeNull();
    expect(favoritesEmptyMessage(true, null, false, 2)).toBeNull();
  });

  it('summary label for favorites mode', () => {
    expect(favoritesSummaryLabel(true, 3, 'x')).toBe('Favorites: 3 shown');
    expect(favoritesSummaryLabel(true, 0, 'x')).toBeNull();
    expect(favoritesSummaryLabel(false, 0, 'Showing 1')).toBe('Showing 1');
  });
});
