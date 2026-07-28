import { describe, expect, it } from 'vitest';
import { normalizePair, watchKey } from './watchlistKey';

describe('watchKey', () => {
  it('lowercases exchange and uppercases symbol', () => {
    expect(watchKey('Binance', 'btcusdt')).toBe('binance|BTCUSDT');
  });

  it('defaults empty exchange to binance', () => {
    expect(watchKey('', 'ETHUSDT')).toBe('binance|ETHUSDT');
  });
});

describe('normalizePair', () => {
  it('normalizes fields', () => {
    expect(normalizePair('COINBASE', 'ethusd')).toEqual({
      exchange: 'coinbase',
      symbol: 'ETHUSD',
    });
  });
});
