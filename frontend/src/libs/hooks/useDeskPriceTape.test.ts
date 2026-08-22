import { describe, expect, it } from 'vitest';
import { deskTapeItemsFromList } from './useDeskPriceTape';
import type { SpotMarket } from '@/libs/api';

const binanceBtc: SpotMarket = {
  symbol: 'BTCUSDT',
  lastPrice: '67000',
  priceChangePercent: '1.2',
};

describe('deskTapeItemsFromList', () => {
  const display = { currency: 'native' as const };

  it('stamps rows with the current venue', () => {
    const items = deskTapeItemsFromList('binance', [binanceBtc], display, 'binance');
    expect(items).toHaveLength(1);
    expect(items[0]?.exchange).toBe('binance');
    expect(items[0]?.href).toBe('/markets/binance/BTCUSDT');
  });

  it('drops the previous venue list while the new venue is in flight', () => {
    expect(deskTapeItemsFromList('coinbase', [binanceBtc], display, 'binance')).toEqual([]);
  });

  it('returns empty when venue is unset (watchlist mode)', () => {
    expect(deskTapeItemsFromList(undefined, [binanceBtc], display)).toEqual([]);
  });
});
