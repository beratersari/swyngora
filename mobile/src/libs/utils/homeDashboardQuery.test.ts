import { describe, expect, it } from 'vitest';
import {
  buildHomePumpScanQuery,
  buildMoversSpotQuery,
  buildVolumeSpotQuery,
  indexDashboardRows,
  mapFavoritesToDashboardRows,
  mapPumpHitsToTeasers,
  mapSpotListToDashboardRows,
} from './homeDashboardQuery';

describe('homeDashboardQuery', () => {
  it('builds movers query sorted by price change', () => {
    const q = buildMoversSpotQuery('binance', 'USDT', 5);
    expect(q.sort).toBe('priceChangePercent');
    expect(q.order).toBe('desc');
    expect(q.limit).toBe(5);
    expect(q.status).toBe('TRADING');
  });

  it('builds volume query', () => {
    const q = buildVolumeSpotQuery();
    expect(q.sort).toBe('quoteVolume');
    expect(q.limit).toBe(5);
  });

  it('builds pump scan with teaser limit', () => {
    const q = buildHomePumpScanQuery('binance', 3);
    expect(q.symbolLimit).toBe(3);
    expect(q.exchange).toBe('binance');
  });

  it('maps spot items to rows', () => {
    const rows = mapSpotListToDashboardRows(
      [
        {
          symbol: 'BTCUSDT',
          lastPrice: '100',
          priceChangePercent: '1.5',
          quoteVolume: '1000000',
        },
      ],
      'binance',
    );
    expect(rows).toHaveLength(1);
    expect(rows[0]?.symbol).toBe('BTCUSDT');
    expect(rows[0]?.changeTone).toBe('success');
  });

  it('maps favorites with spot merge', () => {
    const spot = mapSpotListToDashboardRows(
      [{ symbol: 'ETHUSDT', lastPrice: '50', priceChangePercent: '-2' }],
      'binance',
    );
    const index = indexDashboardRows(spot);
    const fav = mapFavoritesToDashboardRows(
      [
        { exchange: 'binance', symbol: 'ETHUSDT' },
        { exchange: 'binance', symbol: 'SOLUSDT' },
      ],
      5,
      index,
    );
    expect(fav[0]?.lastPriceLabel).not.toBe('—');
    expect(fav[1]?.lastPriceLabel).toBe('—');
  });

  it('maps pump hits', () => {
    const teasers = mapPumpHitsToTeasers(
      [
        {
          symbol: 'PEPEUSDT',
          exchange: 'binance',
          bestReturnPct: 12,
          events: [{}, {}],
          interval: '15m',
        },
      ],
      'binance',
    );
    expect(teasers[0]?.symbol).toBe('PEPEUSDT');
    expect(teasers[0]?.returnTone).toBe('success');
  });
});
