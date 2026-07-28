import { describe, expect, it, vi, beforeEach } from 'vitest';

vi.mock('@/libs/utils/clientId', () => ({
  getOrCreateClientId: () => 'mobile-test-id',
  peekClientId: () => 'mobile-test-id',
}));

import {
  buildAddWatchlistBody,
  buildReplaceWatchlistBody,
  withClientIdQuery,
} from './watchlistApi';

describe('watchlistApi pure helpers', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('withClientIdQuery merges clientId', () => {
    expect(withClientIdQuery({ exchange: 'binance', symbol: 'BTCUSDT' })).toEqual({
      clientId: 'mobile-test-id',
      exchange: 'binance',
      symbol: 'BTCUSDT',
    });
    expect(withClientIdQuery()).toEqual({ clientId: 'mobile-test-id' });
  });

  it('buildAddWatchlistBody defaults exchange', () => {
    expect(buildAddWatchlistBody({ symbol: 'ETHUSDT' })).toEqual({
      clientId: 'mobile-test-id',
      exchange: 'binance',
      symbol: 'ETHUSDT',
    });
    expect(buildAddWatchlistBody({ exchange: 'bybit', symbol: 'SOLUSDT', note: 'n' })).toEqual({
      clientId: 'mobile-test-id',
      exchange: 'bybit',
      symbol: 'SOLUSDT',
      note: 'n',
    });
  });

  it('buildReplaceWatchlistBody', () => {
    expect(
      buildReplaceWatchlistBody({
        items: [{ exchange: 'binance', symbol: 'BTCUSDT' }],
      }),
    ).toEqual({
      clientId: 'mobile-test-id',
      items: [{ exchange: 'binance', symbol: 'BTCUSDT' }],
    });
  });
});
