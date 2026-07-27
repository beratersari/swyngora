import { describe, expect, it } from 'vitest';
import { pickSpotForSymbol } from './WatchlistTable.helpers';

describe('pickSpotForSymbol', () => {
  it('prefers exact match', () => {
    const items = [
      { symbol: 'BTCUSDC' },
      { symbol: 'BTCUSDT' },
    ];
    expect(pickSpotForSymbol(items, 'BTCUSDT')?.symbol).toBe('BTCUSDT');
  });

  it('matches hyphenated coinbase style to compact form', () => {
    const items = [{ symbol: 'BTC-USD' }, { symbol: 'ETH-USD' }];
    expect(pickSpotForSymbol(items, 'BTCUSD')?.symbol).toBe('BTC-USD');
    expect(pickSpotForSymbol(items, 'btc-usd')?.symbol).toBe('BTC-USD');
  });

  it('does not return WBTC for BTC search when no exact BTC pair', () => {
    const items = [{ symbol: 'WBTCUSDT' }, { symbol: 'BTCBUSD' }];
    expect(pickSpotForSymbol(items, 'BTCUSDT')).toBeUndefined();
  });

  it('returns undefined for empty', () => {
    expect(pickSpotForSymbol([], 'BTC')).toBeUndefined();
    expect(pickSpotForSymbol(undefined, 'BTCUSDT')).toBeUndefined();
  });
});
