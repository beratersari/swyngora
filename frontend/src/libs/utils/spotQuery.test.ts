import { describe, expect, it } from 'vitest';
import {
  DEFAULT_MARKETS_STATE,
  defaultQuoteForExchange,
  effectiveMarketsStateForQuery,
  marketsStateToSearchParams,
  parseMarketsSearchParams,
  toSpotListQuery,
} from './spotQuery';

describe('defaultQuoteForExchange', () => {
  it('uses USD on Coinbase and USDT elsewhere', () => {
    const quotes = { coinbase: 'USD', binance: 'USDT', bybit: 'USDT', nasdaq: 'USD', bist: 'TRY' };
    expect(defaultQuoteForExchange('coinbase')).toBe('USD');
    expect(defaultQuoteForExchange('coinbase', quotes)).toBe('USD');
    expect(defaultQuoteForExchange('binance', quotes)).toBe('USDT');
    expect(defaultQuoteForExchange('bist', quotes)).toBe('TRY');
    expect(defaultQuoteForExchange('binance', { binance: 'EUR' })).toBe('EUR');
  });
});

describe('spotQuery', () => {
  it('parses defaults from empty params', () => {
    expect(parseMarketsSearchParams(new URLSearchParams())).toEqual(DEFAULT_MARKETS_STATE);
  });

  it('defaults quote from the API venue map when quote is omitted', () => {
    const quotes = { coinbase: 'USD', bybit: 'USDT', binance: 'USDT' };
    expect(parseMarketsSearchParams(new URLSearchParams('exchange=coinbase'), quotes).quote).toBe(
      'USD',
    );
    expect(parseMarketsSearchParams(new URLSearchParams('exchange=bybit'), quotes).quote).toBe(
      'USDT',
    );
    expect(parseMarketsSearchParams(new URLSearchParams('exchange=coinbase')).quote).toBe('USD');
  });

  it('honors explicit quote override on any exchange', () => {
    expect(
      parseMarketsSearchParams(new URLSearchParams('exchange=coinbase&quote=USDT')).quote,
    ).toBe('USDT');
    expect(
      parseMarketsSearchParams(new URLSearchParams('exchange=binance&quote=EUR')).quote,
    ).toBe('EUR');
  });

  it('lowercases exchange and rejects unknown / invalid sort-order', () => {
    expect(parseMarketsSearchParams(new URLSearchParams('exchange=Coinbase')).exchange).toBe(
      'coinbase',
    );
    expect(parseMarketsSearchParams(new URLSearchParams('exchange=kraken')).exchange).toBe(
      'binance',
    );
    expect(parseMarketsSearchParams(new URLSearchParams('sort=nope')).sort).toBe(
      DEFAULT_MARKETS_STATE.sort,
    );
    expect(parseMarketsSearchParams(new URLSearchParams('order=sideways')).order).toBe(
      DEFAULT_MARKETS_STATE.order,
    );
  });

  it('clamps limit and offset', () => {
    expect(parseMarketsSearchParams(new URLSearchParams('limit=0')).limit).toBe(1);
    expect(parseMarketsSearchParams(new URLSearchParams('limit=9999')).limit).toBe(500);
    expect(parseMarketsSearchParams(new URLSearchParams('limit=50.9')).limit).toBe(50);
    expect(parseMarketsSearchParams(new URLSearchParams('offset=-5')).offset).toBe(0);
    expect(parseMarketsSearchParams(new URLSearchParams('limit=abc')).limit).toBe(50);
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

  it('omits default quote/sort when serializing (per exchange)', () => {
    const quotes = { coinbase: 'USD', binance: 'USDT' };
    const p = marketsStateToSearchParams(DEFAULT_MARKETS_STATE);
    expect(p.toString()).toBe('');

    // Coinbase + USD is venue default → no quote in URL
    const cb = marketsStateToSearchParams(
      {
        ...DEFAULT_MARKETS_STATE,
        exchange: 'coinbase',
        quote: 'USD',
      },
      quotes,
    );
    expect(cb.get('exchange')).toBe('coinbase');
    expect(cb.get('quote')).toBeNull();

    // Coinbase + USDT is an override → keep in URL
    const cbUsdt = marketsStateToSearchParams(
      {
        ...DEFAULT_MARKETS_STATE,
        exchange: 'coinbase',
        quote: 'USDT',
      },
      quotes,
    );
    expect(cbUsdt.get('quote')).toBe('USDT');
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

  it('omits empty q/tag from query args', () => {
    const q = toSpotListQuery({ ...DEFAULT_MARKETS_STATE, tag: '  ' }, '  ');
    expect(q.q).toBeUndefined();
    expect(q.tag).toBeUndefined();
  });
});
