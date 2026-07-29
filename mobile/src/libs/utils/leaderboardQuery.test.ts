import { describe, it, expect } from 'vitest';
import {
  buildLeaderboardSpotQuery,
  mapSpotListToLeaderboardRows,
  rankLabel,
} from './leaderboardQuery';

describe('buildLeaderboardSpotQuery', () => {
  it('builds gainers', () => {
    expect(buildLeaderboardSpotQuery({ board: 'gainers' })).toMatchObject({
      sort: 'priceChangePercent',
      order: 'desc',
      status: 'TRADING',
      quote: 'USDT',
    });
  });

  it('builds losers', () => {
    expect(buildLeaderboardSpotQuery({ board: 'losers', limit: 10 })).toMatchObject({
      sort: 'priceChangePercent',
      order: 'asc',
      limit: 10,
    });
  });

  it('builds volume', () => {
    expect(
      buildLeaderboardSpotQuery({
        board: 'volume',
        exchange: 'coinbase',
        quote: 'USD',
        offset: 30,
      }),
    ).toMatchObject({
      sort: 'quoteVolume',
      order: 'desc',
      exchange: 'coinbase',
      quote: 'USD',
      offset: 30,
    });
  });
});

describe('rankLabel', () => {
  it('is 1-based across pages', () => {
    expect(rankLabel(0, 0)).toBe('#1');
    expect(rankLabel(30, 0)).toBe('#31');
  });
});

describe('mapSpotListToLeaderboardRows', () => {
  it('maps ranks and symbols', () => {
    const rows = mapSpotListToLeaderboardRows(
      [{ symbol: 'BTCUSDT', lastPrice: '1', priceChangePercent: '2', quoteVolume: '3' }],
      0,
    );
    expect(rows[0]?.rankLabel).toBe('#1');
    expect(rows[0]?.symbol).toBe('BTCUSDT');
  });
});
