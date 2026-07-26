import { describe, expect, it } from 'vitest';
import {
  DEFAULT_MARKETS_STATE,
  marketsStateToSearchParams,
  parseMarketsSearchParams,
  toSpotListQuery,
} from './spotQuery';

describe('spotQuery', () => {
  it('parses defaults from empty params', () => {
    expect(parseMarketsSearchParams(new URLSearchParams())).toEqual(DEFAULT_MARKETS_STATE);
  });

  it('round-trips non-default params', () => {
    const state = {
      ...DEFAULT_MARKETS_STATE,
      exchange: 'bybit' as const,
      q: 'btc',
      tag: 'Meme',
      sort: 'lastPrice' as const,
      order: 'asc' as const,
      limit: 25,
      offset: 50,
    };
    const params = marketsStateToSearchParams(state);
    expect(parseMarketsSearchParams(params)).toEqual(state);
  });

  it('builds spot list query with TRADING status', () => {
    const q = toSpotListQuery(DEFAULT_MARKETS_STATE, ' eth ');
    expect(q.q).toBe('eth');
    expect(q.status).toBe('TRADING');
    expect(q.exchange).toBe('binance');
  });
});
