import { describe, expect, it } from 'vitest';
import {
  candleTagId,
  compactParams,
  indicatorTagId,
  intervalTagId,
  liqHeatmapTagId,
  productTagId,
  resolveExchangeArg,
  spotListTagId,
  supplyTagId,
  tickerTagId,
  transformExchangesResponse,
  transformIntervalsResponse,
  transformProductTagsResponse,
} from './marketApi.helpers';

describe('compactParams', () => {
  it('drops undefined, null, empty string, and non-primitives', () => {
    expect(
      compactParams({
        a: 'x',
        b: undefined,
        c: null,
        d: '',
        e: 0,
        f: true as unknown as string,
      }),
    ).toEqual({ a: 'x', e: 0 });
  });
});

describe('transforms', () => {
  it('transformExchangesResponse defaults', () => {
    expect(transformExchangesResponse({})).toEqual({ exchanges: [], default: 'binance' });
    expect(transformExchangesResponse({ exchanges: ['bybit'], default: 'bybit' })).toEqual({
      exchanges: ['bybit'],
      default: 'bybit',
    });
  });

  it('transformProductTagsResponse uses raw or arg exchange', () => {
    expect(transformProductTagsResponse({ tags: ['Meme'] }, { exchange: 'bybit' })).toEqual({
      exchange: 'bybit',
      tags: ['Meme'],
    });
    expect(transformProductTagsResponse({ exchange: 'coinbase' }, undefined)).toEqual({
      exchange: 'coinbase',
      tags: [],
    });
    expect(transformProductTagsResponse({}, undefined)).toEqual({
      exchange: 'binance',
      tags: [],
    });
  });

  it('transformIntervalsResponse', () => {
    expect(transformIntervalsResponse({ intervals: ['1h'] }, { exchange: 'coinbase' })).toEqual({
      exchange: 'coinbase',
      intervals: ['1h'],
    });
  });

  it('resolveExchangeArg', () => {
    expect(resolveExchangeArg(undefined, 'x')).toBe('x');
    expect(resolveExchangeArg({ exchange: 'bybit' })).toBe('bybit');
    expect(resolveExchangeArg(undefined)).toBe('binance');
  });
});

describe('tag ids', () => {
  it('spotListTagId is arg-scoped', () => {
    expect(spotListTagId(undefined)).toBe('default');
    const a = spotListTagId({
      exchange: 'binance',
      q: 'btc',
      quote: 'USDT',
      sort: 'quoteVolume',
      order: 'desc',
      limit: 50,
      offset: 0,
    });
    const b = spotListTagId({
      exchange: 'coinbase',
      q: 'btc',
      quote: 'USDT',
      sort: 'quoteVolume',
      order: 'desc',
      limit: 50,
      offset: 0,
    });
    expect(a).not.toBe(b);
    expect(a).toContain('binance');
  });

  it('detail tag helpers', () => {
    expect(productTagId({ exchange: 'bybit' })).toBe('bybit');
    expect(productTagId(undefined)).toBe('binance');
    expect(intervalTagId(undefined)).toBe('binance');
    expect(candleTagId({ symbol: 'BTCUSDT' })).toBe('binance:BTCUSDT:1h:100');
    expect(tickerTagId({ symbol: 'ETHUSDT', exchange: 'bybit' })).toBe('bybit:ETHUSDT');
    expect(supplyTagId({ asset: 'BTC' })).toBe('BTC');
    expect(supplyTagId(undefined)).toBe('unknown');
    expect(indicatorTagId({ symbol: 'BTCUSDT' })).toContain('BTCUSDT');
    expect(liqHeatmapTagId({ symbol: 'BTCUSDT', range: '7d' })).toBe('all:BTCUSDT:7d');
  });
});
