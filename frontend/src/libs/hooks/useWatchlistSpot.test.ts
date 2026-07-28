import { describe, expect, it } from 'vitest';
import { pickSpotForSymbol } from './useWatchlistSpot';
import type { SpotMarket } from '@/libs/api';

const item = (symbol: string): SpotMarket => ({ symbol });

describe('pickSpotForSymbol', () => {
  it('prefers exact symbol match', () => {
    const items = [item('BTCUSDT'), item('ETHUSDT')];
    expect(pickSpotForSymbol(items, 'ETHUSDT')?.symbol).toBe('ETHUSDT');
  });

  it('matches normalized compact forms', () => {
    const items = [item('BTC-USD')];
    expect(pickSpotForSymbol(items, 'btcusd')?.symbol).toBe('BTC-USD');
  });

  it('returns undefined when empty', () => {
    expect(pickSpotForSymbol([], 'BTCUSDT')).toBeUndefined();
    expect(pickSpotForSymbol(undefined, 'BTCUSDT')).toBeUndefined();
  });
});
