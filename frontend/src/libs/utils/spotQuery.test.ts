import { describe, expect, it } from 'vitest';
import {
  DEFAULT_MARKETS_STATE,
  effectiveMarketsStateForQuery,
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

  it('resets offset when debounced q is ahead of URL q', () => {
    const state = { ...DEFAULT_MARKETS_STATE, q: '', offset: 100 };
    expect(effectiveMarketsStateForQuery(state, 'btc').offset).toBe(0);
    expect(toSpotListQuery(state, 'btc').offset).toBe(0);
    expect(toSpotListQuery(state, 'btc').q).toBe('btc');
  });

  it('keeps offset when debounced q matches URL q', () => {
    const state = { ...DEFAULT_MARKETS_STATE, q: 'btc', offset: 100 };
    expect(effectiveMarketsStateForQuery(state, 'btc').offset).toBe(100);
    expect(toSpotListQuery(state, 'btc').offset).toBe(100);
  });
});
