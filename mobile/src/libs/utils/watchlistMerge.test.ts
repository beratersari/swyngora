import { describe, expect, it } from 'vitest';
import { watchKey } from './watchlistKey';
import {
  isAtMaxItems,
  mergeWatchlists,
  readLocalWatchlist,
  serializeLocalWatchlist,
} from './watchlistMerge';

describe('watchKey', () => {
  it('normalizes exchange and symbol', () => {
    expect(watchKey('Binance', 'btcusdt')).toBe('binance|BTCUSDT');
  });
});

describe('mergeWatchlists', () => {
  it('unions and prefers local on conflict', () => {
    const server = [
      { exchange: 'binance', symbol: 'BTCUSDT', note: 'server' },
      { exchange: 'coinbase', symbol: 'ETHUSD' },
    ];
    const local = [
      { exchange: 'binance', symbol: 'BTCUSDT', note: 'local' },
      { exchange: 'bybit', symbol: 'SOLUSDT' },
    ];
    const merged = mergeWatchlists(local, server);
    const btc = merged.find((m) => m.symbol === 'BTCUSDT');
    expect(btc?.note).toBe('local');
    expect(merged).toHaveLength(3);
    expect(merged.map((m) => watchKey(m.exchange, m.symbol)).sort()).toEqual(
      ['binance|BTCUSDT', 'bybit|SOLUSDT', 'coinbase|ETHUSD'].sort(),
    );
  });

  it('handles empty server', () => {
    expect(mergeWatchlists([{ exchange: 'binance', symbol: 'X' }], [])).toEqual([
      { exchange: 'binance', symbol: 'X' },
    ]);
  });
});

describe('isAtMaxItems', () => {
  it('detects cap', () => {
    expect(isAtMaxItems(Array.from({ length: 200 }, () => ({ exchange: 'b', symbol: 'S' })))).toBe(
      true,
    );
    expect(isAtMaxItems([{ exchange: 'b', symbol: 'S' }])).toBe(false);
  });
});

describe('local serialization', () => {
  it('round-trips', () => {
    const items = [{ exchange: 'binance', symbol: 'BTCUSDT' }];
    expect(readLocalWatchlist(serializeLocalWatchlist(items))).toEqual(items);
  });

  it('returns empty on bad json', () => {
    expect(readLocalWatchlist('not-json')).toEqual([]);
  });
});
