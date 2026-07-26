import { describe, expect, it } from 'vitest';
import { DEFAULT_MARKETS_FILTER, toSpotListQuery } from './spotQuery';

describe('toSpotListQuery', () => {
  it('builds query with defaults and debounced search', () => {
    const q = toSpotListQuery(
      { ...DEFAULT_MARKETS_FILTER, tags: ['Layer1_Layer2', 'pos'] },
      ' btc ',
    );
    expect(q.exchange).toBe('binance');
    expect(q.q).toBe('btc');
    expect(q.quote).toBe('USDT');
    expect(q.tags).toBe('Layer1_Layer2,pos');
    expect(q.sort).toBe('quoteVolume');
    expect(q.order).toBe('desc');
    expect(q.limit).toBe(30);
    expect(q.status).toBe('TRADING');
  });

  it('omits empty q and tags', () => {
    const q = toSpotListQuery(DEFAULT_MARKETS_FILTER, '   ');
    expect(q.q).toBeUndefined();
    expect(q.tags).toBeUndefined();
  });
});
