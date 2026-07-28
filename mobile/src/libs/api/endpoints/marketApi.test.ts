import { describe, expect, it } from 'vitest';
import {
  compactParams,
  transformExchangesResponse,
  transformIntervalsResponse,
  transformProductTagsResponse,
} from './marketApi';

describe('marketApi pure helpers', () => {
  it('compactParams drops empties', () => {
    expect(compactParams({ a: '1', b: '', c: undefined, d: 0 })).toEqual({ a: '1', d: 0 });
  });

  it('transformExchangesResponse fills defaults', () => {
    expect(transformExchangesResponse({})).toEqual({
      exchanges: [],
      default: 'binance',
    });
    expect(transformExchangesResponse({ exchanges: ['bybit'], default: 'bybit' })).toEqual({
      exchanges: ['bybit'],
      default: 'bybit',
    });
  });

  it('transformProductTagsResponse', () => {
    expect(transformProductTagsResponse({ tags: ['Meme'] }, { exchange: 'binance' })).toEqual({
      exchange: 'binance',
      tags: ['Meme'],
    });
    expect(transformProductTagsResponse({})).toEqual({
      exchange: 'binance',
      tags: [],
    });
  });

  it('transformIntervalsResponse', () => {
    expect(
      transformIntervalsResponse({ intervals: ['1h', '4h'] }, { exchange: 'coinbase' }),
    ).toEqual({ exchange: 'coinbase', intervals: ['1h', '4h'] });
  });
});

describe('postIndicatorsBatch serialize shape', () => {
  it('exports batch types via marketApi module load', async () => {
    const mod = await import('./marketApi');
    expect(typeof mod.usePostIndicatorsBatchQuery).toBe('function');
    expect(mod.marketApi.endpoints.postIndicatorsBatch).toBeTruthy();
  });
});
