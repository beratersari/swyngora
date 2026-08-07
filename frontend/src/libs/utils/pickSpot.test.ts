import { describe, expect, it } from 'vitest';
import { pickSpotForSymbol } from './pickSpot';
import type { SpotMarket } from '@/libs/api';

const item = (symbol: string, extra?: Partial<SpotMarket>): SpotMarket => ({
  symbol,
  ...extra,
});

describe('pickSpotForSymbol', () => {
  it('prefers exact symbol match', () => {
    const items = [item('BTCUSDT'), item('ETHUSDT')];
    expect(pickSpotForSymbol(items, 'ETHUSDT')?.symbol).toBe('ETHUSDT');
  });

  it('matches normalized compact forms (hyphen vs compact)', () => {
    const items = [item('BTC-USD')];
    expect(pickSpotForSymbol(items, 'btcusd')?.symbol).toBe('BTC-USD');
    expect(pickSpotForSymbol(items, 'BTC_USD')?.symbol).toBe('BTC-USD');
  });

  it('is case-insensitive', () => {
    expect(pickSpotForSymbol([item('btc-usd')], 'BTC-USD')?.symbol).toBe('btc-usd');
  });

  it('returns undefined when empty or no match', () => {
    expect(pickSpotForSymbol([], 'BTCUSDT')).toBeUndefined();
    expect(pickSpotForSymbol(undefined, 'BTCUSDT')).toBeUndefined();
    expect(pickSpotForSymbol([item('ETHUSDT')], 'BTCUSDT')).toBeUndefined();
    expect(pickSpotForSymbol([item('ETHUSDT')], '')).toBeUndefined();
  });
});
